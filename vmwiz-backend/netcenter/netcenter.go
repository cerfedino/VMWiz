package netcenter

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

// https://netcenter.ethz.ch/netcenter/rest/nameToIP/freeIps/v4/{}
type NetcenterFreeIP struct {
	IP            string `xml:"ip"`
	IpSubnet      string `xml:"ipSubnet"`
	IpMask        int    `xml:"ipMask"`
	SubnetAndMask string `xml:"subnetAndMask"`
	SubnetName    string `xml:"subnetName"`
}
type netcenterFreeIPList struct {
	XMLName xml.Name          `xml:"freeIps"`
	FreeIps []NetcenterFreeIP `xml:"freeIp"`
}

func netcenterMakeRequest(method string, path string, body []byte) (*http.Request, *http.Client, error) {
	url, _ := url.Parse(fmt.Sprintf("%v%v", os.Getenv("NETCENTER_HOST"), path))
	// fmt.Println("Requesting URL: '" + url.String() + "'")
	req, err := http.NewRequest(method, url.String(), bytes.NewReader(body))
	if err != nil {
		fmt.Printf("ERROR Creating request: %v", err.Error())
		return nil, nil, err
	}

	addAuthHeaders(req, nil)
	req.Host = url.Host
	return req, &http.Client{CheckRedirect: addAuthHeaders}, nil
}

func netcenterDoRequest(req *http.Request, client *http.Client) ([]byte, error) {
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR Making request: ", err.Error())
		return nil, err
	}
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("ERROR Making Request: Status %v\nBody: %v", res.Status, body)
	}

	// TODO: Netcenter adds error xml tags in body sometimes, parse those aswell if they are present
	// Check function get_tree_or_raise_error in netcenter.py

	return body, nil
}

func addAuthHeaders(req *http.Request, via []*http.Request) error {
	req.Header.Set("authorization", fmt.Sprintf("Basic %v", base64.StdEncoding.EncodeToString([]byte(os.Getenv("NETCENTER_USER")+":"+os.Getenv("NETCENTER_PWD")))))
	return nil
}

func GetFreeIPsInSubnet(ipv4 *ipaddr.IPAddress) (*[]NetcenterFreeIP, error) {
	req, client, err := netcenterMakeRequest("GET", fmt.Sprintf("/netcenter/rest/nameToIP/freeIps/v4/%v", ipv4.String()), nil)
	if err != nil {
		return nil, err
	}

	body, err := netcenterDoRequest(req, client)
	if err != nil {
		return nil, err
	}

	var freeIps netcenterFreeIPList
	err = xml.Unmarshal(body, &freeIps)
	if err != nil {
		fmt.Println("ERROR Unmarshal: ", err.Error())
		return nil, err
	}

	return &(freeIps.FreeIps), nil
}
func DeleteHost(ip string) error {
	req, client, err := netcenterMakeRequest("DELETE", fmt.Sprintf("/netcenter/rest/nameToIP/%s", ip), nil)
	if err != nil {
		return err
	}

	body, err := netcenterDoRequest(req, client)
	if err != nil {
		return err
	}

	fmt.Println(string(body))
	return nil
}
