package notifier

import (
	"fmt"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
)

var APPRISE_THREAD_TITLE = "VM Request Notifications"
var SMTP_AUTH smtp.Auth

func useNotifier(tags string, body string) error {

	fmt.Println("[-] Sending notification to vmwiz-notifier")
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
	return useNotifier("new_vmrequest", "```\n"+req.ToString()+"\n```")
}

func NotifyVMRequestStatusChanged(req storage.SQLVMRequest, additional_text string) error {
	switch req.RequestStatus {
	case storage.STATUS_ACCEPTED:
		return useNotifier("vmrequest_accepted", fmt.Sprintf("Request %v approved ! %v", req.ID, additional_text))
	case storage.STATUS_REJECTED:
		return useNotifier("vmrequest_rejected", fmt.Sprintf("Request %v denied ! %v", req.ID, additional_text))
	}

	return nil
}

func NotifyVMCreationUpdate(msg string) error {
	return useNotifier("vmcreation_update", msg)
}

func NotifyVMUsageSurvey(surveyID int64, msg string) error {
	return useNotifier("vmusagesurvey", fmt.Sprintf("VM Usage survey %v: %v", surveyID, msg))
}

func InitSMTP() error {
	// SMTP is not used in this notifier
	SMTP_AUTH = smtp.PlainAuth("", config.AppConfig.SMTP_USER, config.AppConfig.SMTP_PASSWORD, config.AppConfig.SMTP_HOST)
	// Sends an email to no one to test the SMTP connection
	err := smtp.SendMail("Test", SMTP_AUTH, config.AppConfig.SMTP_SENDER, []string{}, []byte("Test"))
	if err != nil {
		return fmt.Errorf("Failed to send test email: %v", err)
	}
	return nil
}

func SendEmail(subject string, body []byte, to []string) error {
	if config.AppConfig.SMTP_ENABLE == false {
		return nil
	}
	if SMTP_AUTH == nil {
		return fmt.Errorf("SMTP not initialized")
	}

	// Rate limit
	time.Sleep(time.Second * 2)

	// Body is formatted according to RFC 822
	mailbody := fmt.Sprintf("Subject: %s\r\nTo: %s\r\n%s\r\n", subject, strings.Join(to, ","), body)
	err := smtp.SendMail(config.AppConfig.SMTP_HOST+":"+config.AppConfig.SMTP_PORT, SMTP_AUTH, config.AppConfig.SMTP_SENDER, to, []byte(mailbody))
	if err != nil {
		return fmt.Errorf("Failed to send email: %v", err)
	}

	return nil
}
