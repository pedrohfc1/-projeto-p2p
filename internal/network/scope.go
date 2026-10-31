package network

import (
	"github.com/libp2p/go-libp2p/core/control"
	n "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"net"
	"projetop2p/internal/adapters"
)

// Interface-scoped hosts accept only TCP IPv4 on the selected local address.
// Known invite destinations are still authenticated by their exact Peer ID.
type interfaceGate struct{ ip string }

func withoutVPNAddresses(as []ma.Multiaddr) []ma.Multiaddr {
	out := []ma.Multiaddr{}
	for _, a := range as {
		ip, _ := a.ValueForProtocol(ma.P_IP4)
		if !adapters.VirtualIPv4(ip) {
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
	_, e := a.ValueForProtocol(ma.P_CIRCUIT)
	if e == nil {
		return false
	}
	ip, e := a.ValueForProtocol(ma.P_IP4)
	if e != nil {
		return false
	}
	v := net.ParseIP(ip)
	return v != nil && !v.IsUnspecified() && !v.IsMulticast() && !v.IsLinkLocalUnicast()
}
func (g interfaceGate) InterceptAccept(a n.ConnMultiaddrs) bool {
	ip, e := a.LocalMultiaddr().ValueForProtocol(ma.P_IP4)
	return e == nil && ip == g.ip
}
func (g interfaceGate) InterceptSecured(_ n.Direction, _ peer.ID, a n.ConnMultiaddrs) bool {
	return g.InterceptAccept(a)
}
func (g interfaceGate) InterceptUpgraded(n.Conn) (bool, control.DisconnectReason) { return true, 0 }
