package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ma "github.com/multiformats/go-multiaddr"
	"projetop2p/internal/roomprobe"
	"tailscale.com/envknob"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/net/netns"
	"tailscale.com/tailcfg"
	"tailscale.com/tsnet"
	"tailscale.com/tstest/integration"
	"tailscale.com/tstest/integration/testcontrol"
	"tailscale.com/types/logger"
)

// This test exercises the actual tsnet TCP stack and WireGuard with SDK test
// control and DERP on loopback. It does not join a hosted Tailscale tailnet.
func TestTailnetUserspaceNoise(t *testing.T) {
	envknob.SetenvForTest(t, "TS_DISABLE_PORTMAPPER", "true")
	envknob.SetenvForTest(t, "TS_NO_LOGS_NO_SUPPORT", "true")
	for _, name := range []string{"TS_AUTHKEY", "TS_AUTH_KEY", "TS_CLIENT_SECRET", "TS_CLIENT_ID", "TS_ID_TOKEN", "TS_AUDIENCE", "TSNET_FORCE_LOGIN"} {
		t.Setenv(name, "")
	}
	netns.SetEnabled(false)
	t.Cleanup(func() { netns.SetEnabled(true) })

	for _, tc := range []struct {
		name        string
		forceDERP   string
		wantPath    string
		testRestart bool
	}{
		{"direct", "false", "direto confirmado por ping disco", true},
		{"derp", "true", "DERP", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			envknob.SetenvForTest(t, "TS_DEBUG_ALWAYS_USE_DERP", tc.forceDERP)
			ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
			defer cancel()
			control := &testcontrol.Server{
				RequireAuth: true,
				DERPMap:     integration.RunDERPAndSTUN(t, logger.Discard, "127.0.0.1"),
			}
			control.HTTPTestServer = httptest.NewUnstartedServer(control)
			control.HTTPTestServer.Start()
			t.Cleanup(control.HTTPTestServer.Close)

			a := newLabTailnetNode(t, control.HTTPTestServer.URL, t.TempDir(), "peetwopee-test-a")
			bDir := t.TempDir()
			b := newLabTailnetNode(t, control.HTTPTestServer.URL, bDir, "peetwopee-test-b")
			aIP := authorizeFixtureNode(t, ctx, control, a)
			bIP := authorizeFixtureNode(t, ctx, control, b)
			if aIP == bIP {
				t.Fatal("the local control plane assigned the same address twice")
			}

			assertLabTailscalePath(t, ctx, a, bIP, tc.wantPath)
			assertUnknownTailnetPeerRejected(t, ctx, a)
			aKey, aID := labApplicationKey(t)
			bKey, bID := labApplicationKey(t)
			listenerHost := newLabHost(t, bKey, aID, b, true)
			acceptLabEcho(t, listenerHost, aID)
			remote := bridgeTailnetHost(t, ctx, b, bIP, listenerHost)
			dialHost := newLabHost(t, aKey, bID, a, false)
			assertLabStream(t, ctx, dialHost, remote)

			// B must reject an unauthorized app identity even though A's
			// authenticated Tailscale node initiated both connections.
			wrongKey, wrongID := labApplicationKey(t)
			wrongHost := newLabHost(t, wrongKey, bID, a, false)
			assertRejectedLabStream(t, ctx, wrongHost, listenerHost, remote, wrongID)
			if tc.forceDERP == "true" {
				assertLabActiveProbe(t, ctx, dialHost, listenerHost)
			}

			if tc.testRestart {
				before := fixtureStatus(t, ctx, b)
				if err := b.Close(); err != nil {
					t.Fatal(err)
				}
				b = newLabTailnetNode(t, control.HTTPTestServer.URL, bDir, "peetwopee-test-b")
				restartCtx, finishRestart := context.WithTimeout(ctx, 10*time.Second)
				defer finishRestart()
				after, err := b.Up(restartCtx)
				if err != nil {
					t.Fatalf("restart from persistent SDK state without new approval: %v", err)
				}
				if after.TUN || after.BackendState != "Running" || before.Self == nil || after.Self == nil || after.Self.PublicKey != before.Self.PublicKey || !slices.Contains(after.TailscaleIPs, bIP) {
					t.Fatalf("restart did not recover the enrolled userspace node")
				}
				t.Log("persistent SDK node key and IP restored without repeating approval")
			}
			if driverLoaded() {
				t.Fatal("Wintun was loaded during userspace traffic")
			}
			t.Logf("validated %s with two tsnet nodes and an authenticated Noise stream", tc.wantPath)
		})
	}
}

func newLabTailnetNode(t *testing.T, controlURL, dir, hostname string) *tsnet.Server {
	t.Helper()
	s := &tsnet.Server{
		ControlURL: controlURL,
		Dir:        dir,
		Hostname:   hostname,
		Logf:       logger.Discard,
		UserLogf:   logger.Discard,
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func authorizeFixtureNode(t *testing.T, ctx context.Context, control *testcontrol.Server, node *tsnet.Server) netip.Addr {
	t.Helper()
	client, err := node.LocalClient()
	if err != nil {
		t.Fatal(err)
	}
	if err := client.StartLoginInteractive(ctx); err != nil {
		t.Fatal(err)
	}
	err = waitForFixture(ctx, func() error {
		status, err := client.Status(ctx)
		if err != nil {
			return err
		}
		if status.BackendState == "Running" {
			t.Fatal("unexpected login before local approval")
		}
		if status.AuthURL == "" {
			return errors.New("waiting for local login URL")
		}
		localURL := control.HTTPTestServer.URL + "/auth/"
		if !strings.HasPrefix(status.AuthURL, localURL) {
			t.Fatal("control plane returned an auth URL outside the local fixture")
		}
		if !control.CompleteAuth(status.AuthURL) {
			t.Fatal("fixture could not approve login")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	status, err := node.Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.TUN || status.BackendState != "Running" {
		t.Fatal("tsnet did not become ready in userspace mode")
	}
	for _, ip := range status.TailscaleIPs {
		if ip.Is4() && tailIP(ip) {
			return ip
		}
	}
	t.Fatal("local coordination did not assign a Tailscale IPv4 address")
	return netip.Addr{}
}

func fixtureStatus(t *testing.T, ctx context.Context, node *tsnet.Server) *ipnstate.Status {
	t.Helper()
	client, err := node.LocalClient()
	if err != nil {
		t.Fatal(err)
	}
	status, err := client.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return status
}

func assertLabTailscalePath(t *testing.T, ctx context.Context, node *tsnet.Server, target netip.Addr, want string) {
	t.Helper()
	client, err := node.LocalClient()
	if err != nil {
		t.Fatal(err)
	}
	pathCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := waitForFixture(pathCtx, func() error {
		pingCtx, stop := context.WithTimeout(pathCtx, 2*time.Second)
		defer stop()
		result, err := client.Ping(pingCtx, target, tailcfg.PingDisco)
		if err != nil {
			return err
		}
		path := pathOf(result)
		if path != want || result.LatencySeconds <= 0 {
			return fmt.Errorf("got %s, want %s with measured RTT", path, want)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func waitForFixture(ctx context.Context, ready func() error) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		err := ready()
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: last observation: %v", ctx.Err(), err)
		case <-ticker.C:
		}
	}
}

func labApplicationKey(t *testing.T) (crypto.PrivKey, peer.ID) {
	t.Helper()
	key, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	id, err := peer.IDFromPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return key, id
}

func newLabHost(t *testing.T, key crypto.PrivKey, allowed peer.ID, node *tsnet.Server, listener bool) host.Host {
	t.Helper()
	client, err := node.LocalClient()
	if err != nil {
		t.Fatal(err)
	}
	transport := tcp.WithDialerForAddr(func(ma.Multiaddr) (tcp.ContextDialer, error) {
		return tailDialer{newTailnetDialer(node, client)}, nil
	})
	opts := []libp2p.Option{
		libp2p.Identity(key),
		libp2p.NoTransports,
		libp2p.Transport(tcp.NewTCPTransport, transport),
		libp2p.Security(noise.ID, noise.New),
		libp2p.DisableRelay(),
		libp2p.ConnectionGater(gate{allowed}),
	}
	if listener {
		opts = append(opts, libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	} else {
		opts = append(opts, libp2p.NoListenAddrs)
	}
	h, err := libp2p.New(opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Close() })
	return h
}

func bridgeTailnetHost(t *testing.T, ctx context.Context, node *tsnet.Server, ip netip.Addr, h host.Host) peer.AddrInfo {
	t.Helper()
	port, err := h.Network().ListenAddresses()[0].ValueForProtocol(ma.P_TCP)
	if err != nil {
		t.Fatal(err)
	}
	tailListener, err := node.Listen("tcp", net.JoinHostPort(ip.String(), "0"))
	if err != nil {
		t.Fatal(err)
	}
	stop := bridge(ctx, tailListener, net.JoinHostPort("127.0.0.1", port))
	t.Cleanup(stop)
	_, rawPort, err := net.SplitHostPort(tailListener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	tailPort, err := strconv.ParseUint(rawPort, 10, 16)
	if err != nil {
		t.Fatal(err)
	}
	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, tailPort))
	if err != nil {
		t.Fatal(err)
	}
	return peer.AddrInfo{ID: h.ID(), Addrs: []ma.Multiaddr{addr}}
}

const labTestProtocol protocol.ID = "/peetwopee/tsnet/lab-test/1"
const labEcho = "userspace Noise echo"

func assertLabStream(t *testing.T, ctx context.Context, dialer host.Host, info peer.AddrInfo) {
	t.Helper()
	streamCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := dialer.Connect(streamCtx, info); err != nil {
		t.Fatal(err)
	}
	stream, err := dialer.NewStream(streamCtx, info.ID, labTestProtocol)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if stream.Conn().ConnState().Security != noise.ID || stream.Conn().RemotePeer() != info.ID {
		t.Fatal("stream did not authenticate the approved peer with Noise")
	}
	_ = stream.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.WriteString(stream, labEcho); err != nil {
		t.Fatal(err)
	}
	reply := make([]byte, len(labEcho))
	if _, err := io.ReadFull(stream, reply); err != nil || string(reply) != labEcho {
		t.Fatalf("round trip failed: %q, %v", reply, err)
	}
}

func acceptLabEcho(t *testing.T, h host.Host, allowed peer.ID) {
	h.SetStreamHandler(labTestProtocol, func(stream network.Stream) {
		defer stream.Close()
		if stream.Conn().ConnState().Security != noise.ID || stream.Conn().RemotePeer() != allowed {
			t.Error("a stream reached the server without the required Noise identity")
			_ = stream.Reset()
			return
		}
		_ = stream.SetDeadline(time.Now().Add(3 * time.Second))
		payload := make([]byte, len(labEcho))
		if _, err := io.ReadFull(stream, payload); err != nil {
			return
		}
		_, _ = stream.Write(payload)
	})
}

func assertRejectedLabStream(t *testing.T, ctx context.Context, wrong, listener host.Host, remote peer.AddrInfo, wrongID peer.ID) {
	t.Helper()
	admitted := make(chan struct{}, 1)
	watch := &network.NotifyBundle{ConnectedF: func(_ network.Network, c network.Conn) {
		if c.RemotePeer() == wrongID {
			select {
			case admitted <- struct{}{}:
			default:
			}
		}
	}}
	listener.Network().Notify(watch)
	defer listener.Network().StopNotify(watch)

	rejectedCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	// Connect and NewStream are observed locally. The remote may reject
	// after either returns on the client, so assert no remote admission
	// and no successful application payload rather than requiring Connect to fail.
	if err := wrong.Connect(rejectedCtx, remote); err == nil {
		if stream, err := wrong.NewStream(rejectedCtx, remote.ID, labTestProtocol); err == nil {
			defer stream.Close()
			_ = stream.SetDeadline(time.Now().Add(time.Second))
			if _, err := io.WriteString(stream, labEcho); err == nil {
				reply := make([]byte, len(labEcho))
				if _, err := io.ReadFull(stream, reply); err == nil {
					t.Fatal("unauthorized application identity exchanged payload")
				}
			}
		}
	}
	select {
	case <-admitted:
		t.Fatal("listener admitted the unauthorized application identity")
	default:
	}
}

func assertUnknownTailnetPeerRejected(t *testing.T, ctx context.Context, node *tsnet.Server) {
	t.Helper()
	client, err := node.LocalClient()
	if err != nil {
		t.Fatal(err)
	}
	dialCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	conn, err := newTailnetDialer(node, client).Dial(dialCtx, "tcp", "100.127.254.253:7447")
	if conn != nil {
		_ = conn.Close()
	}
	if !errors.Is(err, errTailnetPeerNotVisible) {
		t.Fatalf("unexpected result when dialing outside the known tailnet: %v", err)
	}
}

func assertLabActiveProbe(t *testing.T, ctx context.Context, dialer, listener host.Host) {
	t.Helper()
	sameSession := func() bool { return true }
	if _, err := roomprobe.Run(ctx, dialer, listener.ID(), 10, sameSession); err == nil {
		t.Fatal("active test was accepted before receiver approval")
	}
	allowSingleLoadProbe(ctx, listener, dialer.ID(), time.Now().Add(90*time.Second))
	result, err := roomprobe.Run(ctx, dialer, listener.ID(), 10, sameSession)
	if err != nil {
		t.Fatal(err)
	}
	if result.DurationSeconds < 10 || result.PayloadBytes < 4000 || result.PayloadBytes > 160000 || result.PayloadKbps <= 0 || result.PayloadKbps > 128 || result.RTTP95Ms <= 0 {
		t.Fatalf("unexpected authenticated load result: %+v", result)
	}
	if _, err := roomprobe.Run(ctx, dialer, listener.ID(), 10, sameSession); err == nil {
		t.Fatal("one-use receiver consent allowed a second active test")
	}
	t.Logf("10 s consented DERP diagnostic: %d payload bytes, %.1f kbps, RTT P95 %.1f ms", result.PayloadBytes, result.PayloadKbps, result.RTTP95Ms)
}
