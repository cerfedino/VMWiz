package main

import (
	"fmt"
	"net/http"
	"net/url"
)

func useNotifier(tags string, title string, body string) error {

	fmt.Println("[-] Sending notification to vmwhiz-notifier")
	v := url.Values{}
	v.Add("title", title)
	v.Add("tags", tags)
	v.Add("body", body)

	_, err := http.PostForm("http://vmwhiz-notifier:8080/notify/default", v)
	if err != nil {
		return err
	}
	return nil
}

func NotifyVMRequest(f Form) error {
	return useNotifier("new_vmrequest", fmt.Sprintf("[VMWHIZ] %v", f.Email), f.toString())
}

func SendTestNotification(body string) error {
	return useNotifier("test", "VMWHIZ notification test", body)
}
