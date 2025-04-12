package notifier

import (
	"fmt"
	"net/http"
	"net/url"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
)

var THREAD_TITLE = "VM Request Notifications"

func useNotifier(tags string, body string) error {

	fmt.Println("[-] Sending notification to vmwiz-notifier")
	v := url.Values{}
	v.Add("title", THREAD_TITLE)
	v.Add("tags", tags)
	v.Add("body", body)

	_, err := http.PostForm("http://vmwiz-notifier:8080/notify/default", v)
	if err != nil {
		return err
	}
	return nil
}

func SendTestNotification(body string) error {
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
