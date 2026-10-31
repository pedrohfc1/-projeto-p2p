package network

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/control"
	net "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// A real relay is exercised on loopback. The gater prevents direct B/C paths;
// this is a deterministic transport test, not a claim of physical NAT testing.
type relayOnly struct {
	mu      sync.RWMutex
	blocked peer.ID
}

func (g *relayOnly) set(id peer.ID)                 { g.mu.Lock(); g.blocked = id; g.mu.Unlock() }
func (g *relayOnly) InterceptPeerDial(peer.ID) bool { return true }
func (g *relayOnly) InterceptAddrDial(id peer.ID, a ma.Multiaddr) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return id != g.blocked || strings.Contains(a.String(), "/p2p-circuit")
}
func (g *relayOnly) InterceptAccept(net.ConnMultiaddrs) bool { return true }
func (g *relayOnly) InterceptSecured(_ net.Direction, id peer.ID, a net.ConnMultiaddrs) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return id != g.blocked || strings.Contains(a.RemoteMultiaddr().String(), "/p2p-circuit")
}
func (g *relayOnly) InterceptUpgraded(net.Conn) (bool, control.DisconnectReason) { return true, 0 }
func TestRelayDeliveryAndFailover(t *testing.T) {
	relayConfig := func() Config {
		return Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}, RelayServer: true, ForcePublic: true, Interval: time.Second}
	}
	r1 := create(t, relayConfig())
	defer func() {
		if r1 != nil {
			r1.Close()
		}
	}()
	r2 := create(t, relayConfig())
	defer r2.Close()
	seeds := append(r1.Addresses(), r2.Addresses()...)
	ga, gb := &relayOnly{}, &relayOnly{}
	ca := Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}, Relays: seeds, ForcePrivate: true, ConnectionGater: ga, Interval: time.Second}
	cb := ca
	cb.DataDir = t.TempDir()
	cb.ConnectionGater = gb
	// B has only bootstrap addresses: relay candidates must be found via DHT.
	cb.Relays = nil
	cb.Bootstrap = seeds
	a := create(t, ca)
	defer a.Close()
	b := create(t, cb)
	defer b.Close()
	ga.set(b.Host.ID())
	gb.set(a.Host.ID())
	a.Host.Network().ClosePeer(b.Host.ID())
	b.Host.Network().ClosePeer(a.Host.ID())
	connectVia := func(r *Node) {
		t.Helper()
		address := r.Addresses()[0] + "/p2p-circuit/p2p/" + b.Host.ID().String()
		p, e := ParseAddress(address)
		if e != nil {
			t.Fatal(e)
		}
		wait(t, func() bool {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			return a.Connect(ctx, p) == nil
		})
	}
	connectVia(r1)
	for _, r := range []*Node{r1, r2} {
		p, e := ParseAddress(r.Addresses()[0] + "/p2p-circuit/p2p/" + b.Host.ID().String())
		if e != nil {
			t.Fatal(e)
		}
		a.Remember(p)
	}
	if e := a.Send(context.Background(), b.Host.ID(), "relay-one"); e != nil {
		t.Fatal(e)
	}
	select {
	case m := <-b.Messages:
		if m.Text != "relay-one" {
			t.Fatal(m)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("missing relay delivery")
	}
	wait(t, func() bool {
		s := r1.Status().RelayService
		return s != nil && s.Enabled && s.ReservationsAccepted > 0 && s.CircuitsOpened > 0 && s.BytesForwarded > 0
	})
	for _, p := range a.Status().Peers {
		if p.ID == b.Host.ID().String() && p.Path != "relay" {
			t.Fatal("test used a direct path")
		}
	}
	_ = r1.Close()
	r1 = nil
	_ = a.Host.Network().ClosePeer(b.Host.ID())
	_ = b.Host.Network().ClosePeer(a.Host.ID())
	// No manual redial after failure: reconnect uses the stored alternative routes.
	wait(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		return a.Send(ctx, b.Host.ID(), "relay-two") == nil
	})
	select {
	case m := <-b.Messages:
		if m.Text != "relay-two" {
			t.Fatal(m)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("missing failover delivery")
	}
}

func TestRelayProbeAndPublicAnnouncement(t *testing.T) {
	// Example DNS is only advertised; the test dials the real loopback listener.
	c := Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}, RelayServer: true, ForcePublic: true, Announce: []string{"/dns4/relay.example.com/tcp/4001"}}
	r := create(t, c)
	defer r.Close()
	wait(t, func() bool { return r.Status().RelayService.Enabled })
	var local string
	announced := false
	for _, a := range r.Addresses() {
		if strings.HasPrefix(a, "/ip4/127.") {
			local = a
		}
		if strings.HasPrefix(a, "/dns4/relay.example.com/") {
			announced = true
		}
	}
	if local == "" || !announced {
		t.Fatal("missing local or configured announced address")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, e := CheckRelay(ctx, local)
	if e != nil {
		t.Fatal(e)
	}
	if !result.Reserved || !result.VoucherVerified || result.RelayID != r.Host.ID().String() || result.LimitBytes != 1<<20 || result.LimitSeconds != 120 || !result.Expires.After(time.Now()) {
		t.Fatal(result)
	}
	if len(result.Addresses) == 0 {
		t.Fatal("relay did not return its configured public DNS address")
	}
	wait(t, func() bool { return r.Status().RelayService.ReservationsAccepted > 0 })
	// A normal participant must not be reported as a functional relay.
	p := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer p.Close()
	if _, e = CheckRelay(ctx, p.Addresses()[0]); e == nil {
		t.Fatal("participant accepted as relay")
	}
}
