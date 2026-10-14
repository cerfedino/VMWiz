package netcenter

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"time"

	api "git.sos.ethz.ch/_public/isg/go-netcenter"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/logger"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

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

const DNS_TTL int64 = 3600

var DNS_VIEWS = []api.DNSView{api.Intern, api.Extern}

func client() *api.ClientWithResponses {
	return &api.ClientWithResponses{ClientInterface: &api.Client{
		Server: config.AppConfig.NETCENTER_HOST + "/api/v1/",
		Client: &http.Client{Timeout: 10 * time.Second},
		RequestEditors: []api.RequestEditorFn{func(ctx context.Context, req *http.Request) error {
			req.SetBasicAuth("access-key", config.AppConfig.NETCENTER_TOKEN)
			return nil
		}},
	}}
}

// The generated client fills JSON200 exactly for a 200 with a JSON content type, so after a passing check it is never nil.
// Error bodies come either as {"message": ...} (401) or as {"code": ..., "messages": [{"field": ..., "message": ...}]}.
func check(res *http.Response, body []byte) error {
	if res.StatusCode == http.StatusOK && strings.Contains(res.Header.Get("Content-Type"), "json") {
		return nil
	}
	var parsed struct {
		Message  string                              `json:"message"`
		Messages []api.TypesResponseResponsesMessage `json:"messages"`
	}
	var msgs []string
	if json.Unmarshal(body, &parsed) == nil {
		if parsed.Message != "" {
			msgs = append(msgs, parsed.Message)
		}
		for _, m := range parsed.Messages {
			if m.Field != nil {
				msgs = append(msgs, *m.Field+": "+m.Message)
			} else {
				msgs = append(msgs, m.Message)
			}
		}
	}
	if len(msgs) == 0 {
		msgs = []string{string(body)}
	}
	return fmt.Errorf("%s %s: %s: %s", res.Request.Method, res.Request.URL, res.Status, strings.Join(msgs, "; "))
}

func parseIPs(strs *[]string, version ipaddr.IPVersion) ([]*ipaddr.IPAddress, error) {
	ips := []*ipaddr.IPAddress{}
	if strs == nil {
		return ips, nil
	}
	for _, s := range *strs {
		ip, err := ipaddr.NewIPAddressString(s).ToAddress()
		if err != nil || ip.GetIPVersion() != version {
			return nil, fmt.Errorf("'%v' is not an IPv%d address", s, version)
		}
		ips = append(ips, ip)
	}
	return ips, nil
}

func GetFreeIPv4sInSubnet(ctx context.Context, subnet *ipaddr.IPv4Address) ([]*ipaddr.IPAddress, error) {
	var ips []*ipaddr.IPAddress
	res, err := client().ListFreeIpv4ByIpSubnetWithResponse(ctx, &api.ListFreeIpv4ByIpSubnetParams{IpSubnet: subnet.WithoutPrefixLen().String()})
	if err == nil {
		err = check(res.HTTPResponse, res.Body)
	}
	if err == nil {
		ips, err = parseIPs(res.JSON200.Ips, ipaddr.IPv4)
	}
	if err != nil {
		return nil, fmt.Errorf("Get free IPv4 addresses in subnet '%v': %w", subnet, err)
	}
	return ips, nil
}

func GetFreeIPv6sInSubnet(ctx context.Context, subnet *ipaddr.IPv6Address) ([]*ipaddr.IPAddress, error) {
	var ips []*ipaddr.IPAddress
	res, err := client().ListFreeIpv6ByIpSubnetWithResponse(ctx, &api.ListFreeIpv6ByIpSubnetParams{IpSubnet: subnet.WithoutPrefixLen().String()})
	if err == nil {
		err = check(res.HTTPResponse, res.Body)
	}
	if err == nil {
		ips, err = parseIPs(res.JSON200.Ips, ipaddr.IPv6)
	}
	if err != nil {
		return nil, fmt.Errorf("Get free IPv6 addresses in subnet '%v': %w", subnet, err)
	}
	return ips, nil
}

// Returns the Netcenter records of fqdn, IPv4 ones first, IPv6 ones second.
func GetHostIPs(ctx context.Context, fqdn string) ([]api.NameToIP, []api.NameToIP, error) {
	res, err := client().SearchNameToIPsWithResponse(ctx, &api.SearchNameToIPsParams{FqName: &fqdn})
	if err == nil {
		err = check(res.HTTPResponse, res.Body)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("Get host IPs of '%v': %w", fqdn, err)
	}
	var ipv4s, ipv6s []api.NameToIP
	for _, r := range *res.JSON200 {
		// The fqName filter accepts wildcards; the deletion path relies on an exact match
		if !strings.EqualFold(r.FqName, fqdn) {
			continue
		}
		if ipaddr.NewIPAddressString(r.Ip).IsIPv4() {
			ipv4s = append(ipv4s, r)
		} else {
			ipv6s = append(ipv6s, r)
		}
	}
	return ipv4s, ipv6s, nil
}

func DeleteDNSEntryByHostname(ctx context.Context, fqdn string) error {
	ipv4s, ipv6s, err := GetHostIPs(ctx, fqdn)
	if err != nil {
		return fmt.Errorf("Deleting DNS entries of '%v': %w", fqdn, err)
	}

	var errors []string
	for _, r := range slices.Concat(ipv4s, ipv6s) {
		res, err := client().DeleteNameToIPWithResponse(ctx, r.Id)
		if err == nil {
			err = check(res.HTTPResponse, res.Body)
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("IP '%v': %v", r.Ip, err))
			continue
		}
		logger.From(ctx).Infof("[+] Deleted DNS entry for IP '%v'", r.Ip)
	}

	if len(errors) > 0 {
		return fmt.Errorf("Deleting DNS entries of '%v': Couldn't delete all DNS entries: Errors: \n- %v", fqdn, strings.Join(errors, "\n- "))
	}
	logger.From(ctx).Infof("[+] Deleted %d DNS entries for host '%v'", len(ipv4s)+len(ipv6s), fqdn)
	return nil
}

func CreateDNSEntry(ctx context.Context, ip *ipaddr.IPAddress, fqdn string) error {
	name, subdomain, found := strings.Cut(fqdn, ".")
	if !found {
		return fmt.Errorf("Creating DNS entry for IP '%v' with FQDN '%v': FQDN has no subdomain", ip, fqdn)
	}

	body := api.CreateNameToIPJSONRequestBody{
		Name:    name,
		Ip:      ip.String(),
		Ttl:     DNS_TTL,
		Views:   DNS_VIEWS,
		Forward: true,
		Reverse: true,
		Ddns:    false,
		Dhcp:    false,
		Locked:  false,
	}
	err := body.Subdomain.FromSubdomainByName(api.SubdomainByName{Name: subdomain})
	if err == nil {
		err = body.NetsupporterGroup.FromNetsupporterGroupByName(api.NetsupporterGroupByName{Name: ISG_GROUP})
	}
	var res *api.CreateNameToIPResponse
	if err == nil {
		res, err = client().CreateNameToIPWithResponse(ctx, body)
	}
	if err == nil {
		err = check(res.HTTPResponse, res.Body)
	}
	if err != nil {
		return fmt.Errorf("Creating DNS entry for IP '%v' with FQDN '%v': %w", ip, fqdn, err)
	}

	logger.From(ctx).Infof("[+] Created DNS entry for IP '%v' with FQDN '%v'", ip, fqdn)
	return nil
}

func Registerhost(ctx context.Context, net string, fqdn string) (*ipaddr.IPv4Address, *ipaddr.IPv6Address, error) {
	op := fmt.Sprintf("Registering host with FQDN '%v'", fqdn)
	freeIPv4s, err := GetFreeIPv4sInSubnet(ctx, VM_SUBNET.V4net)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}
	if len(freeIPv4s) == 0 {
		return nil, nil, fmt.Errorf("%s: No free IPv4 in subnet %v", op, VM_SUBNET.V4net)
	}
	freeIPv6s, err := GetFreeIPv6sInSubnet(ctx, VM_SUBNET.V6net)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	chosenIPv4 := freeIPv4s[0]
	var chosenIPv6 *ipaddr.IPAddress
	// Discard IPv6 addresses that are not in the usable range (0 address + ipv6_offset)
	for _, ip := range freeIPv6s {
		if VM_SUBNET.V6net.Enumerate(ip.ToIPv6()).Cmp(big.NewInt(VM_SUBNET.ipv6_offset)) > 0 {
			chosenIPv6 = ip
			break
		}
	}
	if chosenIPv6 == nil {
		return nil, nil, fmt.Errorf("%s: No usable IPv6 in subnet %v", op, VM_SUBNET.V6net)
	}

	for _, ip := range []*ipaddr.IPAddress{chosenIPv4, chosenIPv6} {
		if err := CreateDNSEntry(ctx, ip, fqdn); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", op, err)
		}
	}

	logger.From(ctx).Infof("[+] Registered host '%v' with FQDN '%v'\n\tIPv4: %v\n\tIPv6: %v", net, fqdn, chosenIPv4, chosenIPv6)
	return chosenIPv4.ToIPv4(), chosenIPv6.ToIPv6(), nil
}
