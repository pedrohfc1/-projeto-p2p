// Package endpoint shares invitation and transport address policy.
package endpoint

import (
	ma "github.com/multiformats/go-multiaddr"
	"net"
	"projetop2p/internal/adapters"
	"strconv"
	"strings"
)

func LinkLocal(a ma.Multiaddr) bool {
	for _, code := range []int{ma.P_IP4, ma.P_IP6} {
		if value, e := a.ValueForProtocol(code); e == nil {
			ip := net.ParseIP(value)
			if ip != nil && (ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()) {
				return true
			}
		}
	}
	return false
}

// TCP4 accepts only a direct TCP endpoint, optionally followed by Peer ID.
func TCP4(a ma.Multiaddr) (string, int, bool) {
	ps := a.Protocols()
	if len(ps) < 2 || len(ps) > 3 || ps[0].Code != ma.P_IP4 || ps[1].Code != ma.P_TCP || (len(ps) == 3 && ps[2].Code != ma.P_P2P) {
		return "", 0, false
	}
	value, _ := a.ValueForProtocol(ma.P_IP4)
	ip := net.ParseIP(value)
	raw, _ := a.ValueForProtocol(ma.P_TCP)
	port, e := strconv.Atoi(raw)
	if e != nil || port < 1 || port > 65535 || ip == nil || ip.To4() == nil || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.Equal(net.IPv4bcast) {
		return "", 0, false
	}
	return value, port, true
}
func Allowed(a ma.Multiaddr, scope string) bool {
	if LinkLocal(a) {
		return false
	}
	if scope == "" || scope == "auto" {
		ip, _ := a.ValueForProtocol(ma.P_IP4)
		return !adapters.VirtualIPv4(ip)
	}
	ip, _, ok := TCP4(a)
	if !ok {
		return false
	}
	if v, known := adapters.VPN(scope); known {
		return strings.HasPrefix(ip, v.IPv4Prefix)
	}
	return scope == "interface"
}
func VPNHint(a ma.Multiaddr) string {
	ip, _, ok := TCP4(a)
	if !ok {
		return ""
	}
	for _, id := range []string{"radmin", "hamachi"} {
		v, _ := adapters.VPN(id)
		if strings.HasPrefix(ip, v.IPv4Prefix) {
			return id
		}
	}
	return ""
}
