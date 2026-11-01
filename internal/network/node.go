package network

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	stdnet "net"
	"sync"
	"sync/atomic"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	corecm "github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	net "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	routing "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	relayv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"projetop2p/internal/discovery"
	"projetop2p/internal/endpoint"
	"projetop2p/internal/identity"
	"projetop2p/internal/messaging"
	"projetop2p/internal/security"
)

const Protocol = protocol.ID("/projeto-p2p/text/0.1.0")

type Config struct {
	IntegratedDialer    tcp.ContextDialer
	IntegratedAddresses []string
	InvitationRoute     func() string
	RegisterRoute       func(string, string) error
	OverlayPath         func(string) string
	DataDir             string
	Listen              []string
	Bootstrap           []string
	Relays              []string
	RelayServer         bool
	Interval            time.Duration
	ForcePrivate        bool
	ForcePublic         bool
	OnMessage           func(messaging.Message) error
	ConnectionGater     corecm.ConnectionGater
	Announce            []string
	LocalOnly           bool
	ScopedIP            string
	ScopedMode          string
	PublicDiscovery     bool
}
type committedReceipt struct {
	digest [32]byte
	at     time.Time
}
type Node struct {
	receiveMu         sync.Mutex
	committed         map[string]committedReceipt
	invitationRoute   func() string
	registerRoute     func(string, string) error
	overlayPath       func(string) string
	Host              host.Host
	DHT               dhtService
	Messages          chan messaging.Message
	Events            chan string
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	mu                sync.Mutex
	known             map[peer.ID]peer.AddrInfo
	guard             security.Guard
	relays            *relaySource
	relayServer       bool
	reachability      string
	onMessage         func(messaging.Message) error
	relayMetrics      *relayMetrics
	localOnly         bool
	scopedMode        string
	traffic           *trafficCounter
	lastConnectMicros atomic.Int64
}

func ParseAddress(s string) (peer.AddrInfo, error) {
	a, e := ma.NewMultiaddr(s)
	if e != nil {
		return peer.AddrInfo{}, e
	}
	p, e := peer.AddrInfoFromP2pAddr(a)
	if e != nil {
		return peer.AddrInfo{}, e
	}
	return *p, nil
}
func New(parent context.Context, c Config) (*Node, error) {
	if c.LocalOnly {
		ip := stdnet.ParseIP(c.ScopedIP)
		if ip == nil || ip.To4() == nil || ip.IsUnspecified() {
			return nil, errors.New("IPv4 da interface inválido")
		}
		if len(c.Bootstrap) > 0 || len(c.Relays) > 0 || c.RelayServer || len(c.Announce) > 0 {
			return nil, errors.New("modo de interface não aceita infraestrutura pública")
		}
		if c.ForcePublic {
			return nil, errors.New("modo de interface não é um operador público")
		}
		for _, value := range c.Listen {
			a, e := ma.NewMultiaddr(value)
			if e != nil {
				return nil, e
			}
			ip, _ := a.ValueForProtocol(ma.P_IP4)
			if ip != c.ScopedIP {
				return nil, errors.New("listener deve usar somente o IPv4 selecionado")
			}
		}
	}
	if c.ForcePrivate && c.ForcePublic {
		return nil, errors.New("conflicting reachability modes")
	}
	announced := []ma.Multiaddr{}
	for _, s := range c.Announce {
		a, e := ma.NewMultiaddr(s)
		if e != nil {
			return nil, e
		}
		if _, e = a.ValueForProtocol(ma.P_P2P); e == nil {
			return nil, errors.New("--announce deve conter transporte e porta, sem /p2p/ID")
		}
		if _, e = a.ValueForProtocol(ma.P_CIRCUIT); e == nil {
			return nil, errors.New("--announce não aceita /p2p-circuit")
		}
		port, e := a.ValueForProtocol(ma.P_TCP)
		if e != nil || port == "0" {
			return nil, errors.New("--announce exige porta TCP fixa")
		}
		if !manet.IsPublicAddr(a) {
			return nil, errors.New("--announce exige IP público ou DNS não local; não comprova acessibilidade")
		}
		announced = append(announced, a)
	}
	key, err := identity.Load(c.DataDir)
	if err != nil {
		return nil, err
	}
	cm, err := connmgr.NewConnManager(32, 64, connmgr.WithGracePeriod(time.Minute))
	if err != nil {
		return nil, err
	}
	limits := rcmgr.DefaultLimits.AutoScale()
	limits = rcmgr.PartialLimitConfig{System: rcmgr.ResourceLimits{Conns: 128, Streams: 256, Memory: 64 << 20, FD: 128}}.Build(limits)
	rm, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(limits))
	if err != nil {
		cm.Close()
		return nil, err
	}
	traffic := &trafficCounter{}
	opts := []libp2p.Option{libp2p.BandwidthReporter(traffic), libp2p.Identity(key), libp2p.ListenAddrStrings(c.Listen...), libp2p.Security(noise.ID, noise.New), libp2p.ConnectionManager(cm), libp2p.ResourceManager(rm)}
	if !c.LocalOnly {
		opts = append(opts, libp2p.AddrsFactory(func(as []ma.Multiaddr) []ma.Multiaddr { return withoutVPNAddresses(append(as, announced...)) }))
	}
	if c.IntegratedDialer != nil {
		announced := []ma.Multiaddr{}
		for _, raw := range c.IntegratedAddresses {
			a, e := ma.NewMultiaddr(raw)
			if e != nil || !endpoint.Allowed(a, "integrated") {
				rm.Close()
				cm.Close()
				return nil, errors.New("invalid integrated announcement")
			}
			announced = append(announced, a)
		}
		opts = append(opts, libp2p.WithDialTimeout(45*time.Second), libp2p.SwarmOpts(swarm.WithDialTimeout(40*time.Second), swarm.WithDialTimeoutLocal(40*time.Second)), libp2p.DisableRelay(), libp2p.NoTransports, libp2p.Transport(tcp.NewTCPTransport, tcp.WithConnectionTimeout(35*time.Second), tcp.WithDialerForAddr(func(ma.Multiaddr) (tcp.ContextDialer, error) { return c.IntegratedDialer, nil })), libp2p.ForceReachabilityPrivate(), libp2p.AddrsFactory(func([]ma.Multiaddr) []ma.Multiaddr { return announced }))
	} else if c.LocalOnly {
		opts = append(opts, libp2p.DisableRelay(), libp2p.NoTransports, libp2p.Transport(tcp.NewTCPTransport, tcp.WithDialerForAddr(func(ma.Multiaddr) (tcp.ContextDialer, error) {
			return &stdnet.Dialer{LocalAddr: &stdnet.TCPAddr{IP: stdnet.ParseIP(c.ScopedIP)}, Timeout: 8 * time.Second}, nil
		})), libp2p.ConnectionGater(interfaceGate{ip: c.ScopedIP, scope: c.ScopedMode}), libp2p.ForceReachabilityPrivate(), libp2p.AddrsFactory(func(as []ma.Multiaddr) []ma.Multiaddr {
			return scopedAddresses(as, c.ScopedIP, c.Listen)
		}))
	} else {
		opts = append(opts, libp2p.EnableHolePunching(), libp2p.EnableRelay())
	}
	if c.ConnectionGater != nil && !c.LocalOnly {
		opts = append(opts, libp2p.ConnectionGater(c.ConnectionGater))
	}
	source := &relaySource{}
	var relays []peer.AddrInfo
	for _, s := range c.Relays {
		p, e := ParseAddress(s)
		if e != nil {
			rm.Close()
			cm.Close()
			return nil, e
		}
		relays = append(relays, p)
		source.add(p)
	}
	if c.ForcePrivate && !c.LocalOnly {
		opts = append(opts, libp2p.ForceReachabilityPrivate())
	}
	if c.ForcePublic {
		opts = append(opts, libp2p.ForceReachabilityPublic())
	}
	if !c.RelayServer && !c.LocalOnly {
		opts = append(opts, libp2p.EnableAutoRelayWithPeerSource(source.source, autorelay.WithMinCandidates(1), autorelay.WithMaxCandidates(16), autorelay.WithNumRelays(2), autorelay.WithBootDelay(2*time.Second), autorelay.WithBackoff(10*time.Second), autorelay.WithMinInterval(5*time.Second)))
	}
	var metrics *relayMetrics
	if c.RelayServer {
		metrics = newRelayMetrics()
		res := relayv2.DefaultResources()
		res.MaxReservations = 64
		res.MaxCircuits = 8
		res.MaxReservationsPerIP = 8
		res.MaxReservationsPerASN = 32
		res.ReservationTTL = 20 * time.Minute
		res.Limit = &relayv2.RelayLimit{Duration: 2 * time.Minute, Data: 1 << 20}
		opts = append(opts, libp2p.EnableRelayService(relayv2.WithResources(res), relayv2.WithMetricsTracer(metrics)))
	}
	h, err := libp2p.New(opts...)
	if err != nil {
		rm.Close()
		cm.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	var boot []peer.AddrInfo
	for _, s := range c.Bootstrap {
		p, e := ParseAddress(s)
		if e != nil {
			cancel()
			h.Close()
			return nil, e
		}
		boot = append(boot, p)
	}
	boot = append(boot, relays...)
	var kd dhtService
	if !c.LocalOnly {
		kd, err = newDHT(ctx, h, c, boot)
	}
	if err != nil {
		cancel()
		h.Close()
		return nil, err
	}
	n := &Node{Host: h, DHT: kd, Messages: make(chan messaging.Message, 64), Events: make(chan string, 32), ctx: ctx, cancel: cancel, known: map[peer.ID]peer.AddrInfo{}, relays: source, relayServer: c.RelayServer, reachability: "desconhecida", onMessage: c.OnMessage}
	n.invitationRoute = c.InvitationRoute
	n.registerRoute = c.RegisterRoute
	n.overlayPath = c.OverlayPath
	n.relayMetrics = metrics
	n.traffic = traffic
	n.localOnly = c.LocalOnly
	n.scopedMode = c.ScopedMode
	if c.ForcePrivate {
		n.reachability = "privada (modo assistido)"
	}
	if c.ForcePublic {
		n.reachability = "pública (configurada pelo operador)"
	}
	for _, p := range boot {
		n.remember(p)
	}
	sub, e := h.EventBus().Subscribe(new(event.EvtLocalReachabilityChanged))
	if e != nil {
		n.Close()
		return nil, e
	}
	n.wg.Add(1)
	go func() {
		defer n.wg.Done()
		defer sub.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-sub.Out():
				if !ok {
					return
				}
				if ev, ok := v.(event.EvtLocalReachabilityChanged); ok {
					n.mu.Lock()
					n.reachability = ev.Reachability.String()
					n.mu.Unlock()
				}
			}
		}
	}()
	for _, s := range c.Bootstrap {
		p, e := ParseAddress(s)
		if e != nil {
			n.Close()
			return nil, e
		}
		n.remember(p)
	}
	h.SetStreamHandler(Protocol, n.receive)
	if kd != nil {
		err = kd.Bootstrap(ctx)
	}
	if err != nil {
		n.Close()
		return nil, err
	}
	if c.Interval <= 0 {
		c.Interval = 10 * time.Second
	}
	n.wg.Add(1)
	go n.run(c.Interval)
	return n, nil
}
func (n *Node) remember(p peer.AddrInfo) {
	if n.localOnly {
		filtered := []ma.Multiaddr{}
		scope := n.scopedMode
		if scope == "" {
			scope = "interface"
		}
		for _, a := range p.Addrs {
			if endpoint.Allowed(a, scope) {
				filtered = append(filtered, a)
			}
		}
		p.Addrs = filtered
	}
	if len(p.Addrs) == 0 {
		return
	}
	if p.ID == n.Host.ID() {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	_, exists := n.known[p.ID]
	if exists || len(n.known) < 256 {
		// Preserve alternative direct/relay routes when invitations contain several
		// addresses for one identity. Prefer freshly observed addresses and bound memory.
		merged := []ma.Multiaddr{}
		seen := map[string]bool{}
		previous := n.known[p.ID].Addrs
		if n.localOnly {
			previous = nil
		}
		for _, a := range append(append([]ma.Multiaddr{}, p.Addrs...), previous...) {
			if !seen[a.String()] && len(merged) < 16 {
				merged = append(merged, a)
				seen[a.String()] = true
			}
		}
		p.Addrs = merged
		n.known[p.ID] = p
		n.Host.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.PermanentAddrTTL)
	}
}
func (n *Node) Connect(ctx context.Context, p peer.AddrInfo) error {
	n.remember(p)
	if n.localOnly && n.Host.Network().Connectedness(p.ID) != net.Connected {
		n.Host.Peerstore().ClearAddrs(p.ID)
	}
	existed := len(n.Host.Network().ConnsToPeer(p.ID)) > 0
	started := time.Now()
	e := n.Host.Connect(ctx, p)
	if e == nil && !existed {
		n.lastConnectMicros.Store(time.Since(started).Microseconds())
	}
	return e
}

// Remember schedules bounded reconnection without blocking the caller.
func (n *Node) Remember(p peer.AddrInfo) { n.remember(p) }
func (n *Node) run(interval time.Duration) {
	defer n.wg.Done()
	tick := time.NewTicker(interval)
	defer tick.Stop()
	type retry struct {
		next  time.Time
		delay time.Duration
	}
	retries := map[peer.ID]retry{}
	for {
		n.mu.Lock()
		peers := make([]peer.AddrInfo, 0, len(n.known))
		for _, p := range n.known {
			peers = append(peers, p)
		}
		n.mu.Unlock()
		attempts := 0
		for _, p := range peers {
			if n.Host.Network().Connectedness(p.ID) == net.Connected {
				delete(retries, p.ID)
				continue
			}
			r := retries[p.ID]
			if time.Now().Before(r.next) || attempts >= 8 {
				continue
			}
			attempts++
			timeout := 3 * time.Second
			if n.overlayPath != nil {
				timeout = 40 * time.Second
			}
			ctx, cancel := context.WithTimeout(n.ctx, timeout)
			err := n.Host.Connect(ctx, p)
			cancel()
			if err != nil {
				if r.delay == 0 {
					r.delay = interval
				}
				r.delay *= 2
				if r.delay > time.Minute {
					r.delay = time.Minute
				}
				r.next = time.Now().Add(r.delay + time.Duration(rand.Int64N(int64(time.Second))))
				retries[p.ID] = r
				n.event(fmt.Sprintf("Conexão com %s falhou; nova tentativa em até %s: %v", p.ID, r.delay+interval, err))
			} else {
				delete(retries, p.ID)
			}
		}
		if n.DHT != nil {
			ctx, cancel := context.WithTimeout(n.ctx, 5*time.Second)
			rd := routing.NewRoutingDiscovery(n.DHT)
			if n.relayServer {
				_, _ = rd.Advertise(ctx, RelayNamespace)
			} else {
				found, e := rd.FindPeers(ctx, RelayNamespace)
				if e == nil {
					for p := range found {
						if p.ID != n.Host.ID() {
							n.relays.add(p)
						}
					}
				}
			}
			found, err := discovery.Find(ctx, n.DHT)
			if err == nil {
				for p := range found {
					if p.ID == n.Host.ID() {
						continue
					}
					n.remember(p)
					if n.Host.Network().Connectedness(p.ID) != net.Connected {
						_ = n.Host.Connect(ctx, p)
					}
				}
			}
			if err != nil && n.ctx.Err() == nil {
				n.event("Descoberta DHT: " + err.Error())
			}
			cancel()
		}
		select {
		case <-n.ctx.Done():
			return
		case <-tick.C:
		}
	}
}
func (n *Node) event(s string) {
	select {
	case n.Events <- s:
	default:
	}
}
func (n *Node) receive(s net.Stream) {
	defer s.Close()
	_ = s.SetDeadline(time.Now().Add(5 * time.Second))
	// A strict bounded frame is consumed before JSON decoding; no unbounded allocation.
	b, err := bufio.NewReader(io.LimitReader(s, messaging.MaxFrame+1)).ReadBytes('\n')
	if err != nil || len(b) > messaging.MaxFrame {
		s.Reset()
		return
	}
	var m messaging.Message
	if json.Unmarshal(b, &m) != nil || m.Validate(s.Conn().RemotePeer(), n.Host.ID(), s.Conn().RemotePublicKey(), time.Now()) != nil {
		s.Reset()
		return
	}
	// Serialize commit/receipt decisions so an ACK never precedes persistence.
	n.receiveMu.Lock()
	defer n.receiveMu.Unlock()
	now := time.Now()
	key := m.From + ":" + m.ID
	canonical, _ := json.Marshal(m)
	digest := sha256.Sum256(canonical)
	if n.committed == nil {
		n.committed = map[string]committedReceipt{}
	}
	for k, v := range n.committed {
		if now.Sub(v.at) > 5*time.Minute {
			delete(n.committed, k)
		}
	}
	if receipt, ok := n.committed[key]; ok {
		if receipt.digest == digest && n.guard.Retry(m.From, m.ID, now) {
			_, _ = s.Write([]byte("OK\n"))
		} else {
			s.Reset()
		}
		return
	}
	if !n.guard.Accept(m.From, m.ID, now) {
		s.Reset()
		return
	}
	committed := false
	if n.onMessage != nil {
		if n.onMessage(m) == nil {
			committed = true
			select {
			case n.Messages <- m:
			default:
			}
		}
	} else {
		select {
		case n.Messages <- m:
			committed = true
		default:
		}
	}
	if !committed {
		n.guard.Forget(m.From, m.ID)
		s.Reset()
		return
	}
	n.committed[key] = committedReceipt{digest: digest, at: now}
	_, _ = s.Write([]byte("OK\n"))
}
func (n *Node) Send(ctx context.Context, id peer.ID, text string) error {
	m, err := n.Compose(id, text)
	if err != nil {
		return err
	}
	return n.Deliver(ctx, m)
}
func (n *Node) Compose(id peer.ID, text string) (messaging.Message, error) {
	return messaging.New(n.Host.Peerstore().PrivKey(n.Host.ID()), id, text)
}
func (n *Node) Deliver(ctx context.Context, m messaging.Message) error {
	id, err := peer.Decode(m.To)
	if err != nil {
		return err
	}
	// Relay circuits carry limited connections. Explicit opt-in is required.
	s, err := n.Host.NewStream(net.WithAllowLimitedConn(ctx, "text via Circuit Relay v2"), id, Protocol)
	if err != nil {
		return err
	}
	defer s.Close()
	deadline := time.Now().Add(5 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = s.SetDeadline(deadline)
	if err = json.NewEncoder(s).Encode(m); err != nil {
		s.Reset()
		return err
	}
	b, err := bufio.NewReader(io.LimitReader(s, 4)).ReadString('\n')
	if err != nil {
		return err
	}
	if b != "OK\n" {
		return errors.New("message not acknowledged")
	}
	return nil
}
func (n *Node) Addresses() []string {
	out := []string{}
	for _, a := range n.Host.Addrs() {
		out = append(out, fmt.Sprintf("%s/p2p/%s", a, n.Host.ID()))
	}
	return out
}
func (n *Node) Close() error {
	n.cancel()
	n.wg.Wait()
	if n.DHT != nil {
		_ = n.DHT.Close()
	}
	return n.Host.Close()
}

func (n *Node) InvitationRoute() string {
	if n.invitationRoute != nil {
		return n.invitationRoute()
	}
	return ""
}
func (n *Node) RegisterInvitationRoute(id, route string) error {
	if route == "" {
		return nil
	}
	if n.registerRoute == nil {
		return errors.New("integrated transport not active")
	}
	return n.registerRoute(id, route)
}
