package notifier

import (
	"fmt"
	"net/http"
	"net/url"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
)

func useNotifier(tags string, title string, body string) error {

	fmt.Println("[-] Sending notification to vmwiz-notifier")
	v := url.Values{}
	v.Add("title", title)
	v.Add("tags", tags)
	v.Add("body", body)

	_, err := http.PostForm("http://vmwiz-notifier:8080/notify/default", v)
	if err != nil {
		return err
	}
	return nil
}

var THREAD_TITLE = "VM Request Notifications"

func NotifyVMRequest(req storage.SQLVMRequest) error {
	return useNotifier("new_vmrequest", THREAD_TITLE, req.ToString())
}

func NotifyVMRequestStatusChanged(req storage.SQLVMRequest) error {
	switch req.RequestStatus {
	case storage.STATUS_ACCEPTED:
		return useNotifier("vmrequest_accepted", THREAD_TITLE, fmt.Sprintf("Request %v approved !", req.ID))
	case storage.STATUS_REJECTED:
		return useNotifier("vmrequest_denied", THREAD_TITLE, fmt.Sprintf("Request %v denied !", req.ID))
	}

	return nil
}

func SendTestNotification(body string) error {
	return useNotifier("test", "VMWIZ notification test", body)
}
