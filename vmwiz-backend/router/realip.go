package router

import (
	"net"
	"net/http"
	"strings"

	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/config"
)

// Walks the header from the right: entries left of the last trusted proxy are client-supplied.
func GetRealIP(r *http.Request) string {
	peer := r.RemoteAddr
	if host, _, err := net.SplitHostPort(peer); err == nil {
		peer = host
	}

	if !isTrustedProxy(peer) {
		return peer
	}
	hops := strings.Split(r.Header.Get(config.AppConfig.REAL_IP_HEADER), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		hop := strings.TrimSpace(hops[i])
		if isTrustedProxy(hop) {
			continue
		}
		if net.ParseIP(hop) != nil {
			return hop
		}
		return peer
	}
	return peer
}

func isTrustedProxy(ip string) bool {
	addr := net.ParseIP(ip)
	if addr == nil {
		return false
	}
	for _, n := range config.AppConfig.TRUSTED_PROXIES {
		if n.Contains(addr) {
			return true
		}
	}
	return false
}
