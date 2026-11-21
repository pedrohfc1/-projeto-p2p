package main

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"slices"

	"tailscale.com/client/local"
	"tailscale.com/tsnet"
	"tailscale.com/wgengine/netstack"
)

type tailnetDialer struct {
	server *tsnet.Server
	client *local.Client
}

var errTailnetPeerNotVisible = errors.New("destino ausente dos peers visíveis nesta tailnet")

func newTailnetDialer(server *tsnet.Server, client *local.Client) tailStack {
	return tailnetDialer{server: server, client: client}
}

func (d tailnetDialer) Dial(ctx context.Context, network, address string) (net.Conn, error) {
	target, err := netip.ParseAddrPort(address)
	if err != nil || target.Port() == 0 || !tailIP(target.Addr()) {
		return nil, errors.New("destino TCP Tailscale inválido")
	}
	switch network {
	case "tcp":
	case "tcp4":
		if !target.Addr().Is4() {
			return nil, errors.New("destino Tailscale não é IPv4")
		}
	case "tcp6":
		if !target.Addr().Is6() {
			return nil, errors.New("destino Tailscale não é IPv6")
		}
	default:
		return nil, errors.New("transporte do laboratório aceita somente TCP")
	}

	status, err := d.client.Status(ctx)
	if err != nil {
		return nil, errors.New("estado da tailnet indisponível")
	}
	if status.BackendState != "Running" || status.TUN {
		return nil, errors.New("tailnet userspace indisponível")
	}
	known := false
	for _, peer := range status.Peer {
		if peer != nil && slices.Contains(peer.TailscaleIPs, target.Addr()) {
			known = true
			break
		}
	}
	if !known {
		return nil, errTailnetPeerNotVisible
	}

	// Server.Dial may select an OS socket if its netmap changes between
	// routing and dial. Use the same userspace netstack call as tsnet's
	// own TCP path. Sys and its subsystems are not stable API; keep this
	// seam isolated and validate it when upgrading the pinned SDK.
	system := d.server.Sys()
	if system == nil || !system.IsNetstack() {
		return nil, errors.New("netstack Tailscale indisponível")
	}
	stack, _ := system.Netstack.GetOK()
	impl, ok := stack.(*netstack.Impl)
	if !ok || impl == nil {
		return nil, errors.New("netstack Tailscale indisponível")
	}
	ip4, ip6 := d.server.TailscaleIPs()
	source := ip4
	if target.Addr().Is6() {
		source = ip6
	}
	if !source.IsValid() {
		return nil, errors.New("endereço local Tailscale indisponível")
	}
	conn, err := impl.DialContextTCPWithBind(ctx, source, target)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
