package notifier

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/logger"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/storage"
)

var APPRISE_THREAD_TITLE = "VM Request Notifications"

type SMTP struct {
	rate_limit *rate.Limiter
	mutex      sync.Mutex
	smtp_auth  smtp.Auth
}

var SMTP_CLIENT SMTP

func useNotifier(ctx context.Context, tags string, body string) error {

	logger.From(ctx).Info("[-] Sending notification to vmwiz-notifier")
	if email := auth.UserEmail(ctx); email != "" {
		body += fmt.Sprintf("\n(by %s)", email)
	}
	v := url.Values{}
	v.Add("title", APPRISE_THREAD_TITLE)
	v.Add("tags", tags)
	v.Add("body", body)

	_, err := http.PostForm("http://vmwiz-notifier:8000/notify/default", v)
	if err != nil {
		return err
	}
	return nil
}

func NotifyTest(ctx context.Context, body string) error {
	return useNotifier(ctx, "test", body)
}

func NotifyVMRequest(ctx context.Context, req storage.Request) error {
	return useNotifier(ctx, "new_vmrequest", fmt.Sprintf("New VM Request %v:\n```\n%v\n```", req.Requestid, req.ToString()))
}

func NotifyVMRequestStatusChanged(ctx context.Context, req storage.Request, additional_text string) error {
	switch req.Requeststatus {
	case storage.REQUEST_STATUS_ACCEPTED:
		return useNotifier(ctx, "vmrequest_accepted", fmt.Sprintf("Request %v approved ! %v", req.Requestid, additional_text))
	case storage.REQUEST_STATUS_REJECTED:
		return useNotifier(ctx, "vmrequest_rejected", fmt.Sprintf("Request %v denied ! %v", req.Requestid, additional_text))
	}

	return nil
}

func NotifyVMRequestEdited(ctx context.Context, before, after storage.Request) error {
	changes := describeChanges([]fieldChange{
		{"hostname", before.Hostname, after.Hostname},
		{"cores", fmt.Sprint(before.Cores), fmt.Sprint(after.Cores)},
		{"RAM", gb(before.Ramgb), gb(after.Ramgb)},
		{"SSD", gb(before.Diskgb), gb(after.Diskgb)},
		{"HDD", gb(before.Secondarydiskgb), gb(after.Secondarydiskgb)},
	})
	return useNotifier(ctx, "vmrequest_edited", fmt.Sprintf("Request %v edited: %s", after.Requestid, changes))
}

func NotifyVMCreationUpdate(ctx context.Context, msg string) error {
	return useNotifier(ctx, "vmcreation_update", msg)
}

func NotifyVMUsageSurvey(ctx context.Context, surveyId int64, msg string) error {
	return useNotifier(ctx, "vmusagesurvey", fmt.Sprintf("VM Usage survey %v: %v", surveyId, msg))
}

func InitSMTP() error {
	SMTP_CLIENT.rate_limit = rate.NewLimiter(rate.Every(time.Second*2), 1)

	SMTP_CLIENT.smtp_auth = smtp.PlainAuth("", config.AppConfig.SMTP_USER, config.AppConfig.SMTP_PASSWORD, config.AppConfig.SMTP_HOST)

	// Sends an email to no one to test the SMTP connection
	err := SendEmail("Test", []byte{}, []string{})
	if err != nil {
		return fmt.Errorf("Failed to send test email: %v", err)
	}
	return nil
}

func SendEmail(subject string, body []byte, to []string) error {
	if config.AppConfig.SMTP_ENABLE == false {
		return nil
	}

	// Rate limit
	SMTP_CLIENT.mutex.Lock()
	defer SMTP_CLIENT.mutex.Unlock()
	err := SMTP_CLIENT.rate_limit.Wait(context.Background())
	if err != nil {
		return fmt.Errorf("Failed to wait for rate limit: %v", err)
	}

	if SMTP_CLIENT.smtp_auth == nil {
		return fmt.Errorf("SMTP not initialized")
	}

	// Body is formatted according to RFC 822
	date := time.Now().Format(time.RFC1123Z)
	mailbody := fmt.Sprintf("Subject: %s\r\nFrom: %s\r\nTo: %s\r\nReply-To: %s\r\nDate: %s\r\n%s\r\n", subject, config.AppConfig.SMTP_SENDER, strings.Join(to, ","), config.AppConfig.SMTP_REPLYTO, date, body)
	err = smtp.SendMail(config.AppConfig.SMTP_HOST+":"+config.AppConfig.SMTP_PORT, SMTP_CLIENT.smtp_auth, config.AppConfig.SMTP_SENDER, to, []byte(mailbody))
	if err != nil {
		return fmt.Errorf("Failed to send email: %v", err)
	}

	return nil
}

func NotifyResourceChangeRequest(ctx context.Context, req storage.ResourceChangeRequest) error {
	review := "eligible for automatic approval"
	if len(req.NeedsReviewReasons) > 0 {
		review = "needs admin review"
	}
	return notifyResourceChange(ctx, "resourcechange_new", fmt.Sprintf("New resource change request %d (%s). Replaces any earlier pending request for this VM.\n```\n%v\n```", req.ID, review, req.ToString()))
}

func NotifyResourceChangeRequestApplied(ctx context.Context, req storage.ResourceChangeRequest, auto bool) error {
	prefix := "approved"
	if auto {
		prefix = "auto-approved"
	}
	return notifyResourceChange(ctx, "resourcechange_accepted", fmt.Sprintf("Resource change request %d %s and applied: %s -> cores=%d ram=%dGB SSD=%dGB HDD=%dGB", req.ID, prefix, req.VmHostname, req.NewCores, req.NewRamGb, req.NewDiskGb, req.NewSecondaryDiskGb))
}

func NotifyResourceChangeRequestFailed(ctx context.Context, req storage.ResourceChangeRequest, errMsg string) error {
	return notifyResourceChange(ctx, "resourcechange_failed", fmt.Sprintf("Resource change request %d failed to apply for %s: %s", req.ID, req.VmHostname, errMsg))
}

type fieldChange struct{ name, old, new string }

// describeChanges lists "name old -> new" for every field that differs.
func describeChanges(fields []fieldChange) string {
	var out []string
	for _, f := range fields {
		if f.old != f.new {
			out = append(out, fmt.Sprintf("%s %s -> %s", f.name, f.old, f.new))
		}
	}
	if len(out) == 0 {
		return "no change"
	}
	return strings.Join(out, ", ")
}

func gb(v int32) string { return fmt.Sprintf("%d GB", v) }

func NotifyResourceChangeRequestEdited(ctx context.Context, before, after storage.ResourceChangeRequest) error {
	changes := describeChanges([]fieldChange{
		{"cores", fmt.Sprint(before.NewCores), fmt.Sprint(after.NewCores)},
		{"RAM", gb(before.NewRamGb), gb(after.NewRamGb)},
		{"SSD", gb(before.NewDiskGb), gb(after.NewDiskGb)},
		{"HDD", gb(before.NewSecondaryDiskGb), gb(after.NewSecondaryDiskGb)},
	})
	return notifyResourceChange(ctx, "resourcechange_edited", fmt.Sprintf("Resource change request %d for %s edited: %s", after.ID, after.VmHostname, changes))
}

func NotifyResourceChangeRequestStatusChanged(ctx context.Context, req storage.ResourceChangeRequest) error {
	tag := "resourcechange_rejected"
	if req.Status == storage.RcrStatusCancelled {
		tag = "resourcechange_cancelled"
	}
	return notifyResourceChange(ctx, tag, fmt.Sprintf("Resource change request %d for %s: %s", req.ID, req.VmHostname, req.Status))
}

func notifyResourceChange(ctx context.Context, tag, body string) error {
	err := useNotifier(ctx, tag, body)
	if err != nil {
		log.Printf("Resource change notification (%s) failed: %v", tag, err)
	}
	return err
}

// SendResourceChangeAppliedEmail emails the VM owner that a resource change has
// been applied to their VM. The change is staged in Proxmox; for memory and
// core changes the VM must reboot before the new specs take effect.
func SendResourceChangeAppliedEmail(req storage.ResourceChangeRequest, contacts []string, autoApproved bool, note string) error {
	if len(contacts) == 0 {
		return nil
	}
	verb := "approved by an admin"
	if autoApproved {
		verb = "automatically approved"
	}
	subject := fmt.Sprintf("Resource change applied to %s", req.VmHostname)
	body := fmt.Sprintf(`Your resource change request for %s has been %s.

  cores:    %d -> %d
  RAM:      %d GB -> %d GB
  SSD disk: %d GB -> %d GB
  HDD disk: %d GB -> %d GB

The new configuration is staged in Proxmox. Memory and CPU changes only
take effect after the VM reboots.

Reason given: %s
`, req.VmHostname, verb,
		req.CurrentCores, req.NewCores,
		req.CurrentRamGb, req.NewRamGb,
		req.CurrentDiskGb, req.NewDiskGb,
		req.CurrentSecondaryDiskGb, req.NewSecondaryDiskGb,
		req.RequestReason)
	if note != "" {
		body += fmt.Sprintf("Note from the admin: %s\n", note)
	}
	return SendEmail(subject, []byte(body), contacts)
}

func SendResourceChangeRejectedEmail(req storage.ResourceChangeRequest, contacts []string, reason string) error {
	if len(contacts) == 0 {
		return nil
	}
	subject := fmt.Sprintf("Resource change request for %s rejected", req.VmHostname)
	body := fmt.Sprintf(`Your resource change request for %s has been rejected by an admin.
Your VM has not been changed.

Requested:
  cores:    %d    -> %d
  RAM:      %d GB -> %d GB
  SSD disk: %d GB -> %d GB
  HDD disk: %d GB -> %d GB

Reason for the rejection:
%s
`, req.VmHostname,
		req.CurrentCores, req.NewCores,
		req.CurrentRamGb, req.NewRamGb,
		req.CurrentDiskGb, req.NewDiskGb,
		req.CurrentSecondaryDiskGb, req.NewSecondaryDiskGb,
		reason)
	return SendEmail(subject, []byte(body), contacts)
}
