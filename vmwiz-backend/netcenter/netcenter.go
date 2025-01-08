package netcenter

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

// https://netcenter.ethz.ch/netcenter/rest/nameToIP/freeIps/v4/{}
type NetcenterFreeIPv4 struct {
	IP            string `xml:"ip"`
	IpSubnet      string `xml:"ipSubnet"`
	IpMask        int    `xml:"ipMask"`
	SubnetAndMask string `xml:"subnetAndMask"`
	SubnetName    string `xml:"subnetName"`
}
type netcenterFreeIPv4List struct {
	XMLName xml.Name            `xml:"freeIps"`
	FreeIps []NetcenterFreeIPv4 `xml:"freeIp"`
}

type NetcenterFreeIPv6 struct {
	IP              string `xml:"ipv6"`
	IpSubnet        string `xml:"ipv6Subnet"`
	Prefix          int    `xml:"prefix"`
	SubnetAndPrefix string `xml:"subnetAndPrefix"`
	SubnetName      string `xml:"subnetName"`
	SubnetType      string `xml:"subnetType"`
}
type netcenterFreeIPv6List struct {
	XMLName xml.Name            `xml:"freeIpV6s"`
	FreeIps []NetcenterFreeIPv6 `xml:"freeIpV6"`
}

type NetcenterSubnet struct {
	Name        string
	V4net       *ipaddr.IPv4Address
	V6net       *ipaddr.IPv6Address
	ipv6_offset int64
	Comment     string
}

func NewNetcenterSubnet(name string, v4net string, v6net string, offset int64, comment string) NetcenterSubnet {
	n := NetcenterSubnet{}
	n.Name = name
	v4, _ := ipaddr.NewIPAddressString(v4net).GetAddress().ToZeroHost()
	n.V4net = v4.ToIPv4()
	v6, _ := ipaddr.NewIPAddressString(v6net).GetAddress().ToZeroHost()
	n.V6net = v6.ToIPv6()
	n.ipv6_offset = offset
	n.Comment = comment

	return n
}

var VM_SUBNET NetcenterSubnet = NewNetcenterSubnet("vm",
	"192.33.91.255/24",
	"2001:67c:10ec:49c3::/118",
	0x020,
	"VM (sos-dc-server-1)")

const ISG_GROUP string = "adm-soseth"

func netcenterMakeRequest(method string, path string, body []byte) (*http.Request, *http.Client, error) {
	url, _ := url.Parse(fmt.Sprintf("%v%v", os.Getenv("NETCENTER_HOST"), path))
	// fmt.Println("Requesting URL: '" + url.String() + "'")
	req, err := http.NewRequest(method, url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("Creating request %v %v: %v", method, url.String(), err.Error())
	}

	req.Header.Set("Content-Type", "text/xml")

	addAuthHeaders(req, nil)
	req.Host = url.Host
	return req, &http.Client{CheckRedirect: addAuthHeaders}, nil
}

func netcenterDoRequest(req *http.Request, client *http.Client) ([]byte, error) {
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Making request %v %v: %v", req.Method, req.URL, err.Error())
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Making request: Cannot read body: %v", err.Error())
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("Making request %v %v: Status %v\nBody: %v", req.Method, req.URL, res.Status, string(body))
	}

	// TODO: Netcenter adds error xml tags in body sometimes, parse those aswell if they are present
	// Check function get_tree_or_raise_error in netcenter.py in sans api

	return body, nil
}

func addAuthHeaders(req *http.Request, via []*http.Request) error {
	req.Header.Set("authorization", fmt.Sprintf("Basic %v", base64.StdEncoding.EncodeToString([]byte(os.Getenv("NETCENTER_USER")+":"+os.Getenv("NETCENTER_PWD")))))
	return nil
}

func GetFreeIPv4sInSubnet(ip *ipaddr.IPv4Address) (*[]NetcenterFreeIPv4, error) {
	req, client, err := netcenterMakeRequest("GET", fmt.Sprintf("/netcenter/rest/nameToIP/freeIps/v4/%v", ip.String()), nil)
	if err != nil {
		return nil, err
	}

	body, err := netcenterDoRequest(req, client)
	if err != nil {
		return nil, err
	}

	var freeIps netcenterFreeIPv4List
	err = xml.Unmarshal(body, &freeIps)
	if err != nil {
		return nil, fmt.Errorf("Get free IPv4 addresses in subnet '%v': %v", ip.String(), err.Error())
	}

	return &(freeIps.FreeIps), nil
}

func GetFreeIPv6sInSubnet(ip *ipaddr.IPv6Address) (*[]NetcenterFreeIPv6, error) {
	req, client, err := netcenterMakeRequest("GET", fmt.Sprintf("/netcenter/rest/nameToIP/freeIps/v6/%v", ip.String()), nil)
	if err != nil {
		return nil, err
	}

	body, err := netcenterDoRequest(req, client)
	if err != nil {
		return nil, fmt.Errorf("Get free IPv6 addresses in subnet '%v': %v", ip.String(), err.Error())
	}

	var freeIps netcenterFreeIPv6List
	err = xml.Unmarshal(body, &freeIps)
	if err != nil {
		return nil, fmt.Errorf("Get free IPv6 addresses in subnet '%v': Unmarshal: %v", ip.String(), err.Error())
	}

	return &(freeIps.FreeIps), nil
}

func DeleteDNSEntryByIP(ip *ipaddr.IPAddress) error {
	req, client, err := netcenterMakeRequest("DELETE", fmt.Sprintf("/netcenter/rest/nameToIP/%s", ip.WithoutPrefixLen().String()), nil)
	if err != nil {
		return fmt.Errorf("Deleting DNS entry: %v", err.Error())
	}

	_, err = netcenterDoRequest(req, client)
	if err != nil {
		return fmt.Errorf("Deleting DNS entry: %v", err.Error())
	}

	return nil
}

type netcenterCreateIPv4DNSEntryRequest struct {
	XMLName  xml.Name `xml:"insert"`
	IPv4     string   `xml:"nameToIP>ip,omitempty"`
	Reverse  string   `xml:"nameToIP>reverse,omitempty"`
	ISGGroup string   `xml:"nameToIP>isgGroup,omitempty"`
	FqName   string   `xml:"nameToIP>fqName,omitempty"`
}
type netcenterCreateIPv6DNSEntryRequest struct {
	XMLName  xml.Name `xml:"insert"`
	IPv6     string   `xml:"nameToIP>ipv6,omitempty"`
	Reverse  string   `xml:"nameToIP>reverse,omitempty"`
	ISGGroup string   `xml:"nameToIP>isgGroup,omitempty"`
	FqName   string   `xml:"nameToIP>fqName,omitempty"`
}

func CreateDNSEntry(ip *ipaddr.IPAddress, fqdn string) error {
	// bodyStruct := netcenterCreateIPv4DNSEntryRequest{
	// 	Ip:       ip,
	// 	Reverse:  "Y",
	// 	ISGGroup: ISG_GROUP,
	// 	FqName:   fqdn,
	// }
	var body []byte
	var err error
	if ip.IsIPv4() {
		body, err = xml.Marshal(netcenterCreateIPv4DNSEntryRequest{
			IPv4:     ip.String(),
			Reverse:  "Y",
			ISGGroup: ISG_GROUP,
			FqName:   fqdn,
		})
	} else if ip.IsIPv6() {
		body, err = xml.Marshal(netcenterCreateIPv6DNSEntryRequest{
			IPv6:     ip.String(),
		Reverse:  "Y",
		ISGGroup: ISG_GROUP,
		FqName:   fqdn,
		})
	} else {
		return fmt.Errorf("Creating DNS entry: IP is neither IPv4 nor IPv6 ?: %v", ip)
	}

	if err != nil {
		return fmt.Errorf("Creating DNS entry: Marshalling: %v", err.Error())
	}

	req, client, err := netcenterMakeRequest("POST", "/netcenter/rest/nameToIP", body)
	if err != nil {
		return fmt.Errorf("Creating DNS entry: Creating Request: %v", err.Error())
	}
	body, err = netcenterDoRequest(req, client)
	if err != nil {
		return fmt.Errorf("Creating DNS entry: %v", err.Error())
	}

	return nil
}

func Registerhost(net string, fqdn string) (*ipaddr.IPv4Address, *ipaddr.IPv6Address, error) {
	v4_subnet := VM_SUBNET.V4net.ToIP()
	v6_subnet := VM_SUBNET.V6net.ToIP()

	freeIPv4s, err := GetFreeIPv4sInSubnet(v4_subnet.ToIPv4().WithoutPrefixLen())
	if err != nil {
		return nil, nil, fmt.Errorf("Registering host with FQDN '%v': %v", fqdn, err.Error())
	}
	if len(*freeIPv4s) == 0 {
		return nil, nil, fmt.Errorf("Registering host with FQDN '%v': No free IPv4 in subnet %v", fqdn, v4_subnet)
	}

	freeIPv6s, err := GetFreeIPv6sInSubnet(v6_subnet.ToIPv6().WithoutPrefixLen())
	if err != nil {
		return nil, nil, fmt.Errorf("Registering host with FQDN '%v': %v", fqdn, err.Error())
	}

	log.Printf("There are %d free available IPv4s in the subnet %v\n", len(*freeIPv4s), v4_subnet)
	// fmt.Printf("There are %d free available IPv6s in the subnet %v\n", len(*freev6Ips), v6_subnet)

	chosenIPv4 := ipaddr.NewIPAddressString((*freeIPv4s)[0].IP).GetAddress().ToIPv4()
	var chosenIPv6 *ipaddr.IPv6Address = nil
	// Discard IPv6 addresses that are not in the usable range (0 address + ipv6_offset)
	for _, ip := range *freeIPv6s {
		ipv6 := ipaddr.NewIPAddressString(ip.IP).GetAddress().ToIPv6()
		if v6_subnet.Enumerate(ipv6).Cmp(big.NewInt(VM_SUBNET.ipv6_offset)) > 0 {
			chosenIPv6 = ipv6
			break
		}
	}
	if chosenIPv6 == nil {
		return nil, nil, fmt.Errorf("Registering host with FQDN '%v': No usable IPv6 in subnet %v", fqdn, v6_subnet)
	}

	// fmt.Printf("Choosing first available IP: %v\n", chosenIPv4)

	// ? Adding DNS entry for chosen IP and FQDN through Netcenter
	err = CreateDNSEntry(chosenIPv4.ToIP(), fqdn)
	if err != nil {
		return nil, nil, fmt.Errorf("Registering host %v with FQDN '%v': %v", chosenIPv4, fqdn, err.Error())
	}

	err = CreateDNSEntry(chosenIPv6.ToIP(), fqdn)
	if err != nil {
		return nil, nil, fmt.Errorf("Registering host %v with FQDN '%v': %v", chosenIPv6, fqdn, err.Error())
	}

	fmt.Println(chosenIPv4, chosenIPv6)

	return chosenIPv4, chosenIPv6, nil
}
