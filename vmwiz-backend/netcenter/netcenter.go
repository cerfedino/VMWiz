package netcenter

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func netcenterRequest(method string, path string, body []byte) (*http.Request, error) {
	url, _ := url.Parse(fmt.Sprintf("%v%v", os.Getenv("NETCENTER_HOST"), path))
	fmt.Println("Requesting URL: '" + url.String() + "'")
	req, err := http.NewRequest(method, url.String(), bytes.NewReader(body))
	if err != nil {
		fmt.Printf("ERROR: %v", err.Error())
		return nil, err
	}

	addAuthHeaders(req, nil)
	req.Host = url.Host
	return req, nil
}

func addAuthHeaders(req *http.Request, via []*http.Request) error {
	req.Header.Set("authorization", fmt.Sprintf("Basic %v", base64.StdEncoding.EncodeToString([]byte(os.Getenv("NETCENTER_USER")+":"+os.Getenv("NETCENTER_PWD")))))
	return nil
}

func GetFreeIPsInSubnet(ipv4 string) {
	req, err := netcenterRequest("GET", fmt.Sprintf("/netcenter/rest/nameToIP/freeIps/v4/%v", ipv4), nil)
	if err != nil {
		fmt.Printf("ERROR: %v", err.Error())
		return
	}
	client := &http.Client{
		CheckRedirect: addAuthHeaders,
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: %v", err.Error())
		return
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response: %v", string(body))
}
