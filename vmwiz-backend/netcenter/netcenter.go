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
		return nil, nil, fmt.Errorf("Creating request: %v", err.Error())
	}

	req.Header.Set("Content-Type", "text/xml")

	addAuthHeaders(req, nil)
	req.Host = url.Host
	return req, &http.Client{CheckRedirect: addAuthHeaders}, nil
}

func netcenterDoRequest(req *http.Request, client *http.Client) ([]byte, error) {
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Making request: %v", err.Error())
	}
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("Making request: Status %v\nBody: %v", res.Status, string(body))
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

var SUBNET_NAME_TO_ADDR = map[string]string{
	"vm": "192.33.91.255/24",
}

const ISG_GROUP string = "adm-soseth"

func DeleteDNSEntryByIP(ip string) error {
	req, client, err := netcenterMakeRequest("DELETE", fmt.Sprintf("/netcenter/rest/nameToIP/%s", ip), nil)
	if err != nil {
		return fmt.Errorf("Deleting DNS entry: %v", err.Error())
	}

	_, err = netcenterDoRequest(req, client)
	if err != nil {
		return fmt.Errorf("Deleting DNS entry: %v", err.Error())
	}

	return nil
}

func CreateDNSEntry(ip string, fqdn string) error {
	bodyStruct := netcenterCreateDNSRequest{
		Ip:       ip,
		Reverse:  "Y",
		ISGGroup: ISG_GROUP,
		FqName:   fqdn,
	}
	body, err := xml.Marshal(bodyStruct)
	if err != nil {
		return fmt.Errorf("Creating DNS entry: Marshalling: %v", err.Error())
	}
	fmt.Println(string(body))

	req, client, err := netcenterMakeRequest("POST", "/netcenter/rest/nameToIP", body)
	if err != nil {
		return fmt.Errorf("Creating DNS entry: Creating Request: %v", err.Error())
	}
	body, err = netcenterDoRequest(req, client)
	if err != nil {
		return fmt.Errorf("Creating DNS entry: %v", err.Error())
	}
	fmt.Println("DNS created ! Body: ", string(body))
	return nil
}

type netcenterCreateDNSRequest struct {
	XMLName  xml.Name `xml:"insert"`
	Ip       string   `xml:"nameToIP>ip,omitempty"`
	Reverse  string   `xml:"nameToIP>reverse,omitempty"`
	ISGGroup string   `xml:"nameToIP>isgGroup,omitempty"`
	FqName   string   `xml:"nameToIP>fqName,omitempty"`
}

func Registerhost(net string, fqdn string) (string, error) {
	vm_subnet, _ := ipaddr.NewIPAddressString(SUBNET_NAME_TO_ADDR[net]).GetAddress().ToZeroHost()

	freeIps, err := GetFreeIPsInSubnet(vm_subnet.WithoutPrefixLen())
	if err != nil {
		return "", fmt.Errorf("Registering host with FQDN '%v': %v", fqdn, err.Error())
	}

	if len(*freeIps) == 0 {
		return "", fmt.Errorf("Registering host with FQDN '%v': No free IPs in subnet %v", fqdn, vm_subnet)
	}

	fmt.Printf("There are %d free available IPs in the subnet %v\n", len(*freeIps), vm_subnet)

	chosenIP := (*freeIps)[0].IP
	fmt.Printf("Choosing first available IP: %v\n", chosenIP)

	// ? Adding DNS entry for chosen IP and FQDN through Netcenter
	err = CreateDNSEntry(chosenIP, fqdn)
	if err != nil {
		return "", fmt.Errorf("Registering host %v with FQDN '%v': %v", chosenIP, fqdn, err.Error())
	}

	return chosenIP, nil
}
