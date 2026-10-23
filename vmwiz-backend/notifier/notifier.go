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

func NotifyResourceChangeRequest(req storage.ResourceChangeRequest) error {
	review := "eligible for automatic approval"
	if len(req.Needsreviewreasons) > 0 {
		review = "needs admin review"
	}
	return notifyResourceChange("resourcechange_new", fmt.Sprintf("New resource change request %d (%s). Replaces any earlier pending request for this VM.\n```\n%v\n```", req.ID, review, req.ToString()))
}

func NotifyResourceChangeRequestApplied(req storage.ResourceChangeRequest, auto bool) error {
	prefix := "approved"
	if auto {
		prefix = "auto-approved"
	}
	return notifyResourceChange("resourcechange_accepted", fmt.Sprintf("Resource change request %d %s and applied: %s -> cores=%d ram=%dGB SSD=%dGB HDD=%dGB", req.ID, prefix, req.Vmhostname, req.Newcores, req.Newramgb, req.Newdiskgb, req.Newsecondarydiskgb))
}

func NotifyResourceChangeRequestFailed(req storage.ResourceChangeRequest, errMsg string) error {
	return notifyResourceChange("resourcechange_failed", fmt.Sprintf("Resource change request %d failed to apply for %s: %s", req.ID, req.Vmhostname, errMsg))
}

func NotifyResourceChangeRequestStatusChanged(req storage.ResourceChangeRequest) error {
	tag := "resourcechange_rejected"
	if req.Status == storage.RcrStatusCancelled {
		tag = "resourcechange_cancelled"
	}
	return notifyResourceChange(tag, fmt.Sprintf("Resource change request %d for %s: %s", req.ID, req.Vmhostname, req.Status))
}

func notifyResourceChange(tag, body string) error {
	err := useNotifier(context.Background(), tag, body)
	if err != nil {
		log.Printf("Resource change notification (%s) failed: %v", tag, err)
	}
	return err
}

// SendResourceChangeAppliedEmail emails the VM owner that a resource change has
// been applied to their VM. The change is staged in Proxmox; for memory and
// core changes the VM must reboot before the new specs take effect.
func SendResourceChangeAppliedEmail(req storage.ResourceChangeRequest, contacts []string, autoApproved bool) error {
	if len(contacts) == 0 {
		return nil
	}
	verb := "approved by an admin"
	if autoApproved {
		verb = "automatically approved"
	}
	subject := fmt.Sprintf("Resource change applied to %s", req.Vmhostname)
	body := fmt.Sprintf(`Your resource change request for %s has been %s.

  cores:    %d -> %d
  RAM:      %d GB -> %d GB
  SSD disk: %d GB -> %d GB
  HDD disk: %d GB -> %d GB

The new configuration is staged in Proxmox. Memory and CPU changes only
take effect after the VM reboots. If you do not want the change to take effect,
please undo the request now via the SSH portal.

Reason given: %s
`, req.Vmhostname, verb,
		req.Currentcores, req.Newcores,
		req.Currentramgb, req.Newramgb,
		req.Currentdiskgb, req.Newdiskgb,
		req.Currentsecondarydiskgb, req.Newsecondarydiskgb,
		req.Requestreason)
	return SendEmail(subject, []byte(body), contacts)
}
