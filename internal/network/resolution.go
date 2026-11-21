package network

import (
	"context"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	ma "github.com/multiformats/go-multiaddr"
	"log/slog"
	"net"
	"projetop2p/internal/endpoint"
	"projetop2p/internal/transportaddr"
	"strings"
	"time"
)

func (n *Node) HasInvitationRoute(id string) bool {
	return n.integrated && n.hasRoute != nil && n.hasRoute(id)
}
func (n *Node) resolveEndpoints(p peer.AddrInfo) peer.AddrInfo {
	if n.integrated {
		// Only the userspace dialer understands this identity-derived endpoint. Noise
		// still verifies the expected Peer ID; no OS IPv6 routing is involved.
		if n.HasInvitationRoute(p.ID.String()) {
			p.Addrs = []ma.Multiaddr{ma.StringCast(fmt.Sprintf("/ip6/%s/tcp/%d", transportaddr.LogicalIP(p.ID.String()), transportaddr.ServicePort(p.ID.String())))}
		} else {
			p.Addrs = nil
		}
		return p
	}
	out := []ma.Multiaddr{}
	for _, a := range p.Addrs {
		if endpoint.LinkLocal(a) {
			continue
		}
		if n.localOnly {
			if endpoint.Allowed(a, n.scopedMode) {
				out = append(out, a)
			}
			continue
		}
		ip4, _ := a.ValueForProtocol(ma.P_IP4)
		ip6, _ := a.ValueForProtocol(ma.P_IP6)
		ip := net.ParseIP(ip4)
		if ip == nil {
			ip = net.ParseIP(ip6)
		}
		// Native LAN remains available to same-network callers. Never transfer these
		// endpoints into the integrated dialer or advertise them as public routes.
		if ip != nil && ip.IsUnspecified() {
			continue
		}
		out = append(out, a)
	}
	p.Addrs = out
	return p
}
func (n *Node) clearBackoff(id peer.ID) {
	n.mu.Lock()
	now := time.Now()
	due := now.Sub(n.retryAt[id]) >= 5*time.Second
	if due {
		if n.retryAt == nil {
			n.retryAt = map[peer.ID]time.Time{}
		}
		if len(n.retryAt) > 256 {
			n.retryAt = map[peer.ID]time.Time{}
		}
		n.retryAt[id] = now
	}
	n.mu.Unlock()
	if due {
		if s, ok := n.Host.Network().(*swarm.Swarm); ok {
			s.Backoff().Clear(id)
		}
	}
}

type routeLog struct {
	state string
	at    time.Time
}

func (n *Node) routeDiagnostic(id peer.ID, count int, err error) {
	transport := "libp2p"
	path := "native"
	if n.integrated {
		transport = "libp2p/easytier-socks"
		path = "identity-derived service"
	}
	reason := "connected"
	if err != nil {
		reason = "dial_failed"
		if strings.Contains(err.Error(), "deadline") || strings.Contains(err.Error(), "timeout") {
			reason = "timeout"
		}
		if strings.Contains(err.Error(), "dial backoff") {
			reason = "dial_backoff"
		}
		if n.integrated && !n.HasInvitationRoute(id.String()) {
			reason = "transport_unavailable: signed integrated route missing"
		}
	}
	n.mu.Lock()
	if n.routeLogs == nil {
		n.routeLogs = map[peer.ID]routeLog{}
	}
	old := n.routeLogs[id]
	if old.state == reason && time.Since(old.at) < 30*time.Second {
		n.mu.Unlock()
		return
	}
	if len(n.routeLogs) > 256 {
		n.routeLogs = map[peer.ID]routeLog{}
	}
	n.routeLogs[id] = routeLog{reason, time.Now()}
	n.mu.Unlock()
	// Do not log credentials, complete invites, addresses, or full peer identities.
	slog.Info("community_connectivity", "transport", transport, "path", path, "endpoints", count, "result", reason)
}

// Community endpoints have a stricter boundary than local discovery. Loopback
// is remote-reachable only for an explicitly local listener; RFC1918 is useful
// only when it belongs to an actual subnet on this computer.
func (n *Node) communityEndpointAllowed(a ma.Multiaddr) bool {
	if endpoint.LinkLocal(a) {
		return false
	}
	if n.integrated {
		return endpoint.Allowed(a, "integrated")
	}
	if n.localOnly {
		return endpoint.Allowed(a, n.scopedMode)
	}
	v, _ := a.ValueForProtocol(ma.P_IP4)
	if v == "" {
		v, _ = a.ValueForProtocol(ma.P_IP6)
	}
	ip := net.ParseIP(v)
	if ip == nil {
		return true
	}
	if ip.IsUnspecified() {
		return false
	}
	if ip.IsLoopback() {
		localCount := 0
		for _, listen := range n.Host.Network().ListenAddresses() {
			v, _ := listen.ValueForProtocol(ma.P_IP4)
			if v == "" {
				v, _ = listen.ValueForProtocol(ma.P_IP6)
			}
			local := net.ParseIP(v)
			if local == nil {
				continue
			}
			localCount++
			if !local.IsLoopback() {
				return false
			}
		}
		return localCount > 0
	}
	if ip.IsPrivate() {
		local, _ := net.InterfaceAddrs()
		for _, v := range local {
			if subnet, ok := v.(*net.IPNet); ok && !subnet.IP.IsLoopback() && subnet.Contains(ip) {
				return true
			}
		}
		return false
	}
	return true
}

// ResolvePeerEndpoints returns transport endpoints without revealing route credentials.
func (n *Node) ResolvePeerEndpoints(p peer.AddrInfo) peer.AddrInfo { return n.resolveEndpoints(p) }

// AuthenticatedPeerAddresses uses an existing Noise-authenticated connection to
// the pinned identity. It lets a community follow a manually selected transport
// without treating its old invitation scope as a new authorization boundary.
func (n *Node) AuthenticatedPeerAddresses(id string) ([]string, bool) {
	pid, err := peer.Decode(id)
	if err != nil {
		return nil, false
	}
	connections := n.Host.Network().ConnsToPeer(pid)
	if len(connections) == 0 {
		return nil, false
	}
	out := []string{}
	for _, conn := range connections {
		a := conn.RemoteMultiaddr()
		if n.communityEndpointAllowed(a) && len(out) < 8 {
			out = append(out, a.String()+"/p2p/"+id)
		}
	}
	return out, true
}

// A connectedness flag is not evidence that a muxed connection is still usable.
// Only stream-open deadline failures invalidate it, with a per-peer cooldown.
func (n *Node) failedVoiceStream(id peer.ID, err error) {
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "deadline") {
		return
	}
	n.mu.Lock()
	now := time.Now()
	due := now.Sub(n.retryAt[id]) >= 5*time.Second
	if due {
		if n.retryAt == nil {
			n.retryAt = map[peer.ID]time.Time{}
		}
		n.retryAt[id] = now
	}
	n.mu.Unlock()
	if due {
		_ = n.Host.Network().ClosePeer(id)
		if s, ok := n.Host.Network().(*swarm.Swarm); ok {
			s.Backoff().Clear(id)
		}
		slog.Warn("voice_transport", "origin", "local", "reason", "stream_open_deadline_invalidated_connection")
	}
}
