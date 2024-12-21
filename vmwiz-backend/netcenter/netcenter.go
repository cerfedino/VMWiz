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

func GetFreeIPsInSubnet(ipv4 *ipaddr.IPAddress) (*[]NetcenterFreeIP, error) {
	req, err := netcenterRequest("GET", fmt.Sprintf("/netcenter/rest/nameToIP/freeIps/v4/%v", ipv4.String()), nil)
	if err != nil {
		fmt.Printf("ERROR Creating Request: ", err.Error())
		return nil, err
	}
	client := &http.Client{
		CheckRedirect: addAuthHeaders,
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Printf("ERROR Making request: ", err.Error())
		return nil, err
	}

	var freeIps netcenterFreeIPList
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("ERROR Reading Body: ", err.Error())
		return nil, err
	}

	err = xml.Unmarshal(body, &freeIps)
	if err != nil {
		fmt.Println("ERROR Unmarshal: ", err.Error())
		return nil, err
	}

	return &(freeIps.FreeIps), nil
}
