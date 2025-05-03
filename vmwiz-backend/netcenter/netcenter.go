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
	"strings"
	"time"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/actionlog"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/config"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

// https://netcenter.ethz.ch/netcenter/rest/nameToIP/freeIps/v4/{}
type NetcenterFreeIPv4 struct {
	IP            *ipaddr.IPv4Address
	IpSubnet      *ipaddr.IPv4Address
	SubnetAndMask *ipaddr.IPv4Address

	IpMask     int
	SubnetName string
}
type netcenterFreeIPv4List struct {
	XMLName xml.Name            `xml:"freeIps"`
	FreeIps []NetcenterFreeIPv4 `xml:"freeIp"`
}

func (out *NetcenterFreeIPv4) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// We first unmarshal the XML into a struct with string fields, after which we parse the complex fields separately
	type RawNetcenterFreeIPv4 struct {
		IPString            string `xml:"ip"`
		IpSubnetString      string `xml:"ipSubnet"`
		SubnetAndMaskString string `xml:"subnetAndMask"`

		IpMask     int    `xml:"ipMask"`
		SubnetName string `xml:"subnetName"`
	}
	aux := &RawNetcenterFreeIPv4{}

	// Let the default XML decoding fill `aux`
	if err := d.DecodeElement(aux, &start); err != nil {
		return fmt.Errorf("Unmarshalling NetcenterFreeIPv4: %v", err.Error())
	}

	// copy over fields
	out.IpMask = aux.IpMask
	out.SubnetName = aux.SubnetName

	// Parse complex types (e.g IP addresses)

	ip, err := ipaddr.NewIPAddressString(aux.IPString).ToAddress()
	if err != nil {
		return fmt.Errorf("Unmarshalling NetcenterFreeIPv4: %v", err.Error())
	}
	out.IP = ip.ToIPv4()

	ipsubnet, err := ipaddr.NewIPAddressString(aux.IpSubnetString).ToAddress()
	if err != nil {
		return fmt.Errorf("Unmarshalling NetcenterFreeIPv4: %v", err.Error())
	}
	out.IpSubnet = ipsubnet.ToIPv4()

	subnetandmask, err := ipaddr.NewIPAddressString(aux.SubnetAndMaskString).ToAddress()
	if err != nil {
		return fmt.Errorf("Unmarshalling NetcenterFreeIPv4: %v", err.Error())
	}
	out.SubnetAndMask = subnetandmask.ToIPv4()

	return nil
}

type NetcenterFreeIPv6 struct {
	IP              *ipaddr.IPv6Address
	IpSubnet        *ipaddr.IPv6Address
	SubnetAndPrefix *ipaddr.IPv6Address
	Prefix          int
	SubnetName      string
	SubnetType      string
}
type netcenterFreeIPv6List struct {
	XMLName xml.Name            `xml:"freeIpV6s"`
	FreeIps []NetcenterFreeIPv6 `xml:"freeIpV6"`
}

func (out *NetcenterFreeIPv6) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type RawNetcenterFreeIPv6 struct {
		IPString              string `xml:"ipv6"`
		IpSubnetString        string `xml:"ipv6Subnet"`
		SubnetAndPrefixString string `xml:"subnetAndPrefix"`

		Prefix     int    `xml:"prefix"`
		SubnetName string `xml:"subnetName"`
		SubnetType string `xml:"subnetType"`
	}
	aux := &RawNetcenterFreeIPv6{}

	// Let the default XML decoding fill `aux`
	if err := d.DecodeElement(aux, &start); err != nil {
		return fmt.Errorf("Unmarshalling NetcenterFreeIPv6: %v", err.Error())
	}

	// copy over fields
	out.Prefix = aux.Prefix
	out.SubnetName = aux.SubnetName
	out.SubnetType = aux.SubnetType

	// Parse complex types (e.g IP addresses)
	ip, err := ipaddr.NewIPAddressString(aux.IPString).ToAddress()
	if err != nil {
		return fmt.Errorf("Unmarshalling NetcenterFreeIPv6: %v", err.Error())
	}
	out.IP = ip.ToIPv6()

	ipsubnet, err := ipaddr.NewIPAddressString(aux.IpSubnetString).ToAddress()
	if err != nil {
		return fmt.Errorf("Unmarshalling NetcenterFreeIPv6: %v", err.Error())
	}
	out.IpSubnet = ipsubnet.ToIPv6()

	subnetandprefix, err := ipaddr.NewIPAddressString(aux.SubnetAndPrefixString).ToAddress()
	if err != nil {
		return fmt.Errorf("Unmarshalling NetcenterFreeIPv6: %v", err.Error())
	}
	out.SubnetAndPrefix = subnetandprefix.ToIPv6()

	return nil
}

type netcenterRequestErrors struct {
	XMLName xml.Name `xml:"errors"`
	Errors  []string `xml:"error>msg"`
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
	url, err := url.Parse(fmt.Sprintf("%v%v", config.AppConfig.NETCENTER_HOST, path))
	if err != nil {
		return nil, nil, fmt.Errorf("Creating request: Parsing URL: %v", err.Error())
	}

	req, err := http.NewRequest(method, url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("Creating request %v %v: %v", method, url.String(), err.Error())
	}

	req.Header.Set("Content-Type", "text/xml")

	addAuthHeaders(req, nil)
	req.Host = url.Host
	return req, &http.Client{CheckRedirect: addAuthHeaders, Timeout: time.Second * 10}, nil
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

	// Netcenter adds error xml tags in the response body. We try to unmarshal the errors and return them if they exist
	var errors netcenterRequestErrors
	err = xml.Unmarshal(body, &errors)
	if err == nil && len(errors.Errors) > 0 {
		return nil, fmt.Errorf("Making request %v %v: Errors: \n- %v", req.Method, req.URL, strings.Join(errors.Errors, "\n- "))
	}

	return body, nil
}

func addAuthHeaders(req *http.Request, via []*http.Request) error {
	req.Header.Set("authorization", fmt.Sprintf("Basic %v", base64.StdEncoding.EncodeToString([]byte(config.AppConfig.NETCENTER_USER+":"+config.AppConfig.NETCENTER_PWD))))
	return nil
}

func GetFreeIPv4sInSubnet(ip *ipaddr.IPv4Address) (*[]NetcenterFreeIPv4, error) {
	/*
		<freeIps>
			<freeIp>
				<ip>192.33.91.173</ip>
				<ipSubnet>192.33.91.0</ipSubnet>
				<ipMask>24</ipMask>
				<subnetAndMask>192.33.91.0/24</subnetAndMask>
				<subnetName>sos-dcz2-server-1-a</subnetName>
			</freeIp>
			...
		</freeIps>
	*/
	req, client, err := netcenterMakeRequest("GET", fmt.Sprintf("/netcenter/rest/nameToIP/freeIps/v4/%v", ip.WithoutPrefixLen().String()), nil)
	if err != nil {
		return nil, fmt.Errorf("Get free IPv4 addresses in subnet '%v': %v", ip.String(), err.Error())
	}

	body, err := netcenterDoRequest(req, client)
	if err != nil {
		return nil, fmt.Errorf("Get free IPv4 addresses in subnet '%v': %v", ip.String(), err.Error())
	}

	var freeIps netcenterFreeIPv4List
	err = xml.Unmarshal(body, &freeIps)
	if err != nil {
		return nil, fmt.Errorf("Get free IPv4 addresses in subnet '%v': %v", ip.String(), err.Error())
	}

	// log.Printf("[+] Found %d free IPv4 addresses in subnet '%v'", len(freeIps.FreeIps), ip)
	return &(freeIps.FreeIps), nil
}

func GetFreeIPv6sInSubnet(ip *ipaddr.IPv6Address) (*[]NetcenterFreeIPv6, error) {
	/*
		<freeIpV6>
			<freeIpV6>
				<ipv6>2001:67c:10ec:49c3::3cc</ipv6>
				<ipv6Subnet>2001:67c:10ec:49c3::</ipv6Subnet>
				<prefix>118</prefix>
				<subnetAndPrefix>2001:67c:10ec:49c3::/118</subnetAndPrefix>
				<subnetName>sos-dcz2-server-1-static</subnetName>
				<subnetType>Subnet_Static</subnetType>
			</freeIpV6>
			...
		</freeIpV6>
	*/
	req, client, err := netcenterMakeRequest("GET", fmt.Sprintf("/netcenter/rest/nameToIP/freeIps/v6/%v", ip.WithoutPrefixLen().String()), nil)
	if err != nil {
		return nil, fmt.Errorf("Get free IPv6 addresses in subnet '%v': %v", ip.String(), err.Error())
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

	log.Printf("[+] Deleted DNS entry for IP '%v'", ip)
	return nil
}

func DeleteDNSEntryByHostname(uuid string, fqdn string) error {
	hostIPv4s, hostIPv6s, err := GetHostIPs(fqdn)
	if err != nil {
		return fmt.Errorf("Deleting DNS entries by hostname: %v", err.Error())
	}

	// ? Deleting DNS entries for all IPs of the host
	var errors []string
	for _, ip := range hostIPv4s {
		err = DeleteDNSEntryByIP(ip.IP.ToIP())
		if err != nil {
			errors = append(errors, err.Error())
		}
	}
	for _, ip := range hostIPv6s {
		err = DeleteDNSEntryByIP(ip.IP.ToIP())
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("Deleting DNS entries by hostname: Couldn't delete all DNS entries: Errors: \n- %v", strings.Join(errors, "\n- "))
	}
	actionlog.Printf(uuid, "[+] Deleted %d DNS entries for host '%v'", len(hostIPv4s)+len(hostIPv6s), fqdn)
	return nil
}

func GetHostIPs(fqdn string) ([]NetcenterUsedIPv4, []NetcenterUsedIPv6, error) {

	var res_ipv4 []NetcenterUsedIPv4
	var res_ipv6 []NetcenterUsedIPv6

	ipv4s, err := GetUsedIPv4sInSubnet(VM_SUBNET.V4net)
	if err != nil {
		return nil, nil, fmt.Errorf("Get host IPs: %v", err.Error())
	}

	for _, ip := range *ipv4s {
		if ip.Fqname == fqdn {
			res_ipv4 = append(res_ipv4, ip)
		}
	}

	ipv6s, err := GetUsedIPv6sInSubnet(VM_SUBNET.V6net)
	if err != nil {
		return nil, nil, fmt.Errorf("Get host IPs: %v", err.Error())
	}

	for _, ip := range *ipv6s {
		if ip.Fqname == fqdn {
			res_ipv6 = append(res_ipv6, ip)
		}
	}

	return res_ipv4, res_ipv6, nil
}

type NetcenterUsedIPv4 struct {
	IP       *ipaddr.IPv4Address
	IPSubnet *ipaddr.IPv4Address

	Fqname   string
	Forward  string
	Reverse  string
	TTL      int
	Dhcp     string
	Ddns     string
	IsgGroup string
	Views    []string
}
type netcenterUsedIPv4List struct {
	XMLName xml.Name            `xml:"usedIps"`
	UsedIps []NetcenterUsedIPv4 `xml:"usedIp"`
}

func (out *NetcenterUsedIPv4) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type RawNetcenterUsedIPv4 struct {
		IPString       string `xml:"ip"`
		IPSubnetString string `xml:"ipSubnet"`

		Fqname   string   `xml:"fqname"`
		Forward  string   `xml:"forward"`
		Reverse  string   `xml:"reverse"`
		TTL      int      `xml:"ttl"`
		Dhcp     string   `xml:"dhcp"`
		Ddns     string   `xml:"ddns"`
		IsgGroup string   `xml:"isgGroup"`
		Views    []string `xml:"views>view"`
	}

	aux := &RawNetcenterUsedIPv4{}

	// Let the default XML decoding fill `aux`
	if err := d.DecodeElement(aux, &start); err != nil {
		return fmt.Errorf("Unmarshalling NetcenterUsedIPv4: %v", err.Error())
	}

	// copy over fields
	out.Fqname = aux.Fqname
	out.Forward = aux.Forward
	out.Reverse = aux.Reverse
	out.TTL = aux.TTL
	out.Dhcp = aux.Dhcp
	out.Ddns = aux.Ddns
	out.IsgGroup = aux.IsgGroup
	out.Views = aux.Views

	// Parse complex types (e.g IP addresses)
	ip, err := ipaddr.NewIPAddressString(aux.IPString).ToAddress()
	if err != nil {
		return fmt.Errorf("Unmarshalling NetcenterUsedIPv4: %v", err.Error())
	}
	out.IP = ip.ToIPv4()

	ipsubnet, err := ipaddr.NewIPAddressString(aux.IPSubnetString).ToAddress()
	if err != nil {
		return fmt.Errorf("Unmarshalling NetcenterUsedIPv4: %v", err.Error())
	}
	out.IPSubnet = ipsubnet.ToIPv4()

	return nil
}

func GetUsedIPv4sInSubnet(ip *ipaddr.IPv4Address) (*[]NetcenterUsedIPv4, error) {
	/*
		<usedIps>
			<usedIp>
				<ip>192.33.91.1</ip>
				<ipSubnet>192.33.91.0</ipSubnet>
				<fqname>rou-dcz2-dg-sos-dcz2-server-1-a.ethz.ch</fqname>
				<forward>Y</forward>
				<reverse>Y</reverse>
				<ttl>7200</ttl>
				<dhcp>N</dhcp>
				<ddns>N</ddns>
				<isgGroup>id-kom-net</isgGroup>
				<views>
					<view>intern</view>
				</views>
			</usedIp>
			...
		</usedIps>
	*/
	req, client, err := netcenterMakeRequest("GET", fmt.Sprintf("/netcenter/rest/nameToIP/usedIps/v4/%v", ip.WithoutPrefixLen().String()), nil)
	if err != nil {
		return nil, fmt.Errorf("Get used IPv4 addresses in subnet '%v': %v", ip.String(), err.Error())
	}

	body, err := netcenterDoRequest(req, client)
	if err != nil {
		return nil, fmt.Errorf("Get used IPv4 addresses in subnet '%v': %v", ip.String(), err.Error())
	}

	var usedIps netcenterUsedIPv4List
	err = xml.Unmarshal(body, &usedIps)
	if err != nil {
		return nil, fmt.Errorf("Get used IPv4 addresses in subnet '%v': %v", ip.String(), err.Error())
	}

	return &(usedIps.UsedIps), nil
}

type NetcenterUsedIPv6 struct {
	IP              *ipaddr.IPv6Address
	IPSubnet        *ipaddr.IPv6Address
	SubnetAndPrefix *ipaddr.IPv6Address

	Fqname        string
	Forward       string
	Reverse       string
	TTL           string
	Dhcp          string
	Ddns          string
	IsgGroup      string
	LastDetection string
	Views         []string
}
type netcenterUsedIPv6List struct {
	XMLName xml.Name            `xml:"usedIps"`
	UsedIps []NetcenterUsedIPv6 `xml:"usedIp"`
}

func (out *NetcenterUsedIPv6) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type RawNetcenterUsedIPv6 struct {
		IPString              string `xml:"ip"`
		IPSubnetString        string `xml:"ipSubnet"`
		SubnetAndPrefixString string `xml:"subnetAndPrefix"`

		Fqname        string   `xml:"fqname"`
		Forward       string   `xml:"forward"`
		Reverse       string   `xml:"reverse"`
		TTL           string   `xml:"ttl"`
		Dhcp          string   `xml:"dhcp"`
		Ddns          string   `xml:"ddns"`
		IsgGroup      string   `xml:"isgGroup"`
		LastDetection string   `xml:"lastDetection"`
		Views         []string `xml:"views>view"`
	}

	aux := &RawNetcenterUsedIPv6{}

	// Let the default XML decoding fill `aux`
	if err := d.DecodeElement(aux, &start); err != nil {
		return fmt.Errorf("Unmarshalling NetcenterUsedIPv6: %v", err.Error())
	}

	// copy over fields
	out.Fqname = aux.Fqname
	out.Forward = aux.Forward
	out.Reverse = aux.Reverse
	out.TTL = aux.TTL
	out.Dhcp = aux.Dhcp
	out.Ddns = aux.Ddns
	out.IsgGroup = aux.IsgGroup
	out.LastDetection = aux.LastDetection
	out.Views = aux.Views

	// Parse complex types (e.g IP addresses)
	ip, err := ipaddr.NewIPAddressString(aux.IPString).ToAddress()
	if err != nil {
		return fmt.Errorf("Unmarshalling NetcenterUsedIPv6: %v", err.Error())
	}
	out.IP = ip.ToIPv6()

	ipsubnet, err := ipaddr.NewIPAddressString(aux.IPSubnetString).ToAddress()
	if err != nil {
		return fmt.Errorf("Unmarshalling NetcenterUsedIPv6: %v", err.Error())
	}
	out.IPSubnet = ipsubnet.ToIPv6()

	subnetandprefix, err := ipaddr.NewIPAddressString(aux.SubnetAndPrefixString).ToAddress()
	if err != nil {
		return fmt.Errorf("Unmarshalling NetcenterUsedIPv6: %v", err.Error())
	}
	out.SubnetAndPrefix = subnetandprefix.ToIPv6()

	return nil
}

func GetUsedIPv6sInSubnet(ip *ipaddr.IPv6Address) (*[]NetcenterUsedIPv6, error) {
	/*
	  <usedIps>
	    <usedIp>
	      <ip>2001:67c:10ec:49c3::23a</ip>
	      <ipSubnet>2001:67c:10ec:49c3::</ipSubnet>
	      <subnetAndPrefix>2001:67c:10ec:49c3::/118</subnetAndPrefix>
	      <fqname>emilschaetzle.vsos.ethz.ch</fqname>
	      <forward>Y</forward>
	      <reverse>Y</reverse>
	      <ttl>3600</ttl>
	      <dhcp>N</dhcp>
	      <ddns>N</ddns>
	      <isgGroup>adm-soseth</isgGroup>
	      <lastDetection>2025-01-08 23:03</lastDetection>
	      <views>
	        <view>extern</view>
	        <view>intern</view>
	      </views>
	    </usedIp>
	    ...
	  </usedIps>
	*/
	req, client, err := netcenterMakeRequest("GET", fmt.Sprintf("/netcenter/rest/nameToIP/usedIps/v6/%v", ip.WithoutPrefixLen().String()), nil)
	if err != nil {
		return nil, fmt.Errorf("Get used IPv6 addresses in subnet '%v': %v", ip.String(), err.Error())
	}

	body, err := netcenterDoRequest(req, client)
	if err != nil {
		return nil, fmt.Errorf("Get used IPv6 addresses in subnet '%v': %v", ip.String(), err.Error())
	}

	var usedIps netcenterUsedIPv6List
	err = xml.Unmarshal(body, &usedIps)
	if err != nil {
		return nil, fmt.Errorf("Get used IPv6 addresses in subnet '%v': %v", ip.String(), err.Error())
	}

	return &(usedIps.UsedIps), nil
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
	var reqBody any
	if ip.IsIPv4() {
		reqBody = netcenterCreateIPv4DNSEntryRequest{
			IPv4:     ip.String(),
			Reverse:  "Y",
			ISGGroup: ISG_GROUP,
			FqName:   fqdn,
		}
	} else if ip.IsIPv6() {
		reqBody = netcenterCreateIPv6DNSEntryRequest{
			IPv6:     ip.String(),
			Reverse:  "Y",
			ISGGroup: ISG_GROUP,
			FqName:   fqdn,
		}
	} else {
		return fmt.Errorf("Creating DNS entry: IP %v is neither IPv4 nor IPv6 ?", ip)
	}

	body, err := xml.Marshal(reqBody)
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

	log.Printf("[+] Created DNS entry for IP '%v' with FQDN '%v'", ip, fqdn)
	return nil
}

func Registerhost(net string, fqdn string) (*ipaddr.IPv4Address, *ipaddr.IPv6Address, error) {
	var v4_subnet *ipaddr.IPv4Address = VM_SUBNET.V4net
	var v6_subnet *ipaddr.IPv6Address = VM_SUBNET.V6net

	freeIPv4s, err := GetFreeIPv4sInSubnet(v4_subnet.WithoutPrefixLen())
	if err != nil {
		return nil, nil, fmt.Errorf("Registering host with FQDN '%v': %v", fqdn, err.Error())
	}
	if len(*freeIPv4s) == 0 {
		return nil, nil, fmt.Errorf("Registering host with FQDN '%v': No free IPv4 in subnet %v", fqdn, v4_subnet)
	}

	freeIPv6s, err := GetFreeIPv6sInSubnet(v6_subnet.WithoutPrefixLen())
	if err != nil {
		return nil, nil, fmt.Errorf("Registering host with FQDN '%v': %v", fqdn, err.Error())
	}

	var chosenIPv4 *ipaddr.IPv4Address = (*freeIPv4s)[0].IP
	var chosenIPv6 *ipaddr.IPv6Address = nil
	// Discard IPv6 addresses that are not in the usable range (0 address + ipv6_offset)
	for _, ip := range *freeIPv6s {
		ipv6 := ip.IP
		if v6_subnet.Enumerate(ipv6).Cmp(big.NewInt(VM_SUBNET.ipv6_offset)) > 0 {
			chosenIPv6 = ipv6
			break
		}
	}
	if chosenIPv6 == nil {
		return nil, nil, fmt.Errorf("Registering host with FQDN '%v': No usable IPv6 in subnet %v", fqdn, v6_subnet)
	}

	// ? Adding DNS entry for chosen IP and FQDN through Netcenter
	err = CreateDNSEntry(chosenIPv4.ToIP(), fqdn)
	if err != nil {
		return nil, nil, fmt.Errorf("Registering host %v with FQDN '%v': %v", chosenIPv4, fqdn, err.Error())
	}

	err = CreateDNSEntry(chosenIPv6.ToIP(), fqdn)
	if err != nil {
		return nil, nil, fmt.Errorf("Registering host %v with FQDN '%v': %v", chosenIPv6, fqdn, err.Error())
	}

	log.Printf("[+] Registered host '%v' with FQDN '%v'\n\tIPv4: %v\n\tIPv6: %v", net, fqdn, chosenIPv4, chosenIPv6)
	return chosenIPv4, chosenIPv6, nil
}
