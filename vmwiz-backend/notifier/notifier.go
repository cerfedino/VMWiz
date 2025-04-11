package notifier

import (
	"fmt"
	"net/http"
	"net/url"
)

var THREAD_TITLE = "VM Request Notifications"

func UseNotifier(tags string, body string) error {

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
	return UseNotifier("test", body)
}
