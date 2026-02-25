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

	"git.sos.ethz.ch/ressorts/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/ressorts/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/storage"
)

var APPRISE_THREAD_TITLE = "VM Request Notifications"

type SMTP struct {
	rate_limit *rate.Limiter
	mutex      sync.Mutex
	smtp_auth  smtp.Auth
}

var SMTP_CLIENT SMTP

func useNotifier(tags string, body string) error {

	log.Println("[-] Sending notification to vmwiz-notifier")
	v := url.Values{}
	v.Add("title", APPRISE_THREAD_TITLE)
	v.Add("tags", tags)
	v.Add("body", body)

	_, err := http.PostForm("http://vmwiz-notifier:8080/notify/default", v)
	if err != nil {
		return err
	}
	return nil
}

func NotifyTest(body string) error {
	return useNotifier("test", body)
}

func NotifyVMRequest(req storage.SQLVMRequest) error {
	return useNotifier("new_vmrequest", fmt.Sprintf("New VM Request %v:\n```\n%v\n```", req.ID, req.ToString()))
}

func NotifyVMRequestStatusChanged(req storage.SQLVMRequest, additional_text string) error {
	switch req.RequestStatus {
	case storage.REQUEST_STATUS_ACCEPTED:
		return useNotifier("vmrequest_accepted", fmt.Sprintf("Request %v approved ! %v", req.ID, additional_text))
	case storage.REQUEST_STATUS_REJECTED:
		return useNotifier("vmrequest_rejected", fmt.Sprintf("Request %v denied ! %v", req.ID, additional_text))
	}

	return nil
}

func NotifyVMCreationUpdate(msg string) error {
	return useNotifier("vmcreation_update", msg)
}

func NotifyVMUsageSurvey(surveyId int64, msg string) error {
	return useNotifier("vmusagesurvey", fmt.Sprintf("VM Usage survey %v: %v", surveyId, msg))
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
