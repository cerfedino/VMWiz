package osscan

import (
	"fmt"
	"net"
)

// DefaultCIDR is the example range from the script. Used as the prefilled
// value in the admin UI; admins can override per scan.
const DefaultCIDR = "192.33.91.0/24"

// MaxExpandHosts caps how many hosts a single CIDR can expand into. We
// assume we will never have a block bigger than 512.
const MaxExpandHosts = 512

// ExpandCIDR returns every usable host address inside the CIDR. For IPv4
// /31 and /32 the address(es) are returned as-is; otherwise the network
// and broadcast addresses are excluded.
func ExpandCIDR(cidr string) ([]string, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}

	ones, bits := ipnet.Mask.Size()
	if bits == 0 {
		return nil, fmt.Errorf("invalid CIDR mask in %q", cidr)
	}
	hostBits := bits - ones
	if hostBits >= 10 {
		return nil, fmt.Errorf("CIDR %q is too large to expand (%d host bits, cap is /%d)", cidr, hostBits, bits-9)
	}

	var hosts []string
	skipNetAndBcast := bits == 32 && hostBits >= 2

	for ip := dupIP(ipnet.IP.Mask(ipnet.Mask)); ipnet.Contains(ip); incIP(ip) {
		if len(hosts) >= MaxExpandHosts {
			return nil, fmt.Errorf("CIDR %q expanded to more than %d hosts", cidr, MaxExpandHosts)
		}
		if skipNetAndBcast && (isNetworkAddr(ip, ipnet) || isBroadcastAddr(ip, ipnet)) {
			continue
		}
		hosts = append(hosts, ip.String())
	}
	return hosts, nil
}

func dupIP(ip net.IP) net.IP {
	out := make(net.IP, len(ip))
	copy(out, ip)
	return out
}

func incIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			return
		}
	}
}

func isNetworkAddr(ip net.IP, ipnet *net.IPNet) bool {
	return ip.Equal(ipnet.IP.Mask(ipnet.Mask))
}

func isBroadcastAddr(ip net.IP, ipnet *net.IPNet) bool {
	bcast := dupIP(ipnet.IP.Mask(ipnet.Mask))
	for i := range bcast {
		bcast[i] |= ^ipnet.Mask[i]
	}
	return ip.Equal(bcast)
}
