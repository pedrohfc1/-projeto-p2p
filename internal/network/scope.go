package network

import (
	"github.com/libp2p/go-libp2p/core/control"
	n "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"projetop2p/internal/endpoint"
	"strconv"
)

// Interface-scoped hosts accept only TCP IPv4 on the selected local address.
// Known invite destinations are still authenticated by their exact Peer ID.
type interfaceGate struct{ ip, scope string }

func withoutVPNAddresses(as []ma.Multiaddr) []ma.Multiaddr {
	out := []ma.Multiaddr{}
	for _, a := range as {
		if endpoint.Allowed(a, "auto") {
			out = append(out, a)
		}
	}
	return out
}
func publicAddresses(as []ma.Multiaddr) []ma.Multiaddr {
	out := []ma.Multiaddr{}
	for _, a := range withoutVPNAddresses(as) {
		if manet.IsPublicAddr(a) {
			out = append(out, a)
		}
	}
	return out
}
func (g interfaceGate) InterceptPeerDial(peer.ID) bool { return true }
func (g interfaceGate) InterceptAddrDial(_ peer.ID, a ma.Multiaddr) bool {
	scope := g.scope
	if scope == "" {
		scope = "interface"
	}
	return endpoint.Allowed(a, scope)
}
func (g interfaceGate) InterceptAccept(a n.ConnMultiaddrs) bool {
	ip, e := a.LocalMultiaddr().ValueForProtocol(ma.P_IP4)
	return e == nil && ip == g.ip && g.InterceptAddrDial("", a.RemoteMultiaddr())
}
func (g interfaceGate) InterceptSecured(_ n.Direction, _ peer.ID, a n.ConnMultiaddrs) bool {
	return g.InterceptAccept(a)
}
func (g interfaceGate) InterceptUpgraded(n.Conn) (bool, control.DisconnectReason) { return true, 0 }

func scopedAddresses(as []ma.Multiaddr, ip string, listens []string) []ma.Multiaddr {
	out := []ma.Multiaddr{}
	for _, a := range as {
		host, port, valid := endpoint.TCP4(a)
		if !valid || host != ip {
			continue
		}
		for _, raw := range listens {
			listen, e := ma.NewMultiaddr(raw)
			if e != nil {
				continue
			}
			value, _ := listen.ValueForProtocol(ma.P_TCP)
			if value == "0" || value == strconv.Itoa(port) {
				out = append(out, a)
				break
			}
		}
	}
	return out
}
