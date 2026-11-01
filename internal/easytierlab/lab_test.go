package easytierlab

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/net/proxy"
	"net"
	"os"
	"path/filepath"
	"projetop2p/internal/invite"
	"strings"
	"testing"
	"time"
)

func freePort(t *testing.T) int {
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
func configuration(t *testing.T, ip string) Config {
	return Config{BinaryDir: os.Getenv("EASYTIER_LAB_BIN"), Network: "projetop2p-private-lab-test", Secret: "temporary-lab-secret-32-characters-minimum", VirtualIP: ip, AppPort: freePort(t), TransportPort: freePort(t), SocksPort: freePort(t), RPCPort: freePort(t), BindIP: "127.0.0.1", Peers: []string{}, STUN: []string{}}
}
func TestConfigIsolationAndHash(t *testing.T) {
	c := configuration(t, "10.203.0.1")
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	cfg := c.TOML()
	for _, want := range []string{"no_tun = true", "disable_upnp = true", "enable_ipv6 = false", "accept_dns = false", "private_mode = true", "udp_whitelist = [\"0\"]", "127.0.0.1"} {
		if !strings.Contains(cfg, want) {
			t.Fatal(want)
		}
	}
	if strings.Contains(cfg, c.Secret) || strings.Contains(cfg, "proxy_network") || strings.Contains(cfg, "port_forward") {
		t.Fatal("unsafe config")
	}
	c.Peers = []string{"tcp://attacker:12/path"}
	if c.Validate() == nil {
		t.Fatal("unvalidated endpoint")
	}
	f := filepath.Join(t.TempDir(), "helper.exe")
	os.WriteFile(f, []byte("wrong binary"), 0600)
	if VerifyBinary(f, CoreSHA256) == nil {
		t.Fatal("unverified helper")
	}
}
func TestRealEasyTierNoTUN(t *testing.T) {
	if os.Getenv("EASYTIER_LAB_BIN") == "" {
		t.Skip("optional verified Windows helper integration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()
	k1, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	k2, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	aCfg := configuration(t, "10.203.0.1")
	bCfg := configuration(t, "10.203.0.2")
	bCfg.Peers = []string{"tcp://127.0.0.1:" + itoa(freePort(t)), "tcp://127.0.0.1:" + itoa(aCfg.TransportPort)}
	a, e := Start(ctx, k1, aCfg, t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	b, e := Start(ctx, k2, bCfg, t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	var result Result
	deadline := time.Now().Add(25 * time.Second)
	for {
		result, e = b.Ping(ctx, a.State(ctx).Invite, "Noise over EasyTier no-TUN")
		if e == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("probe failed: %v; A=%+v; B=%+v", e, a.State(ctx), b.State(ctx))
		}
		time.Sleep(time.Second)
	}
	if !result.Authenticated {
		t.Fatal("not authenticated")
	}
	id, _ := peer.IDFromPrivateKey(k1)
	if result.PeerID != id.String() {
		t.Fatal("identity changed")
	}
	// A listening unrelated local TCP service must NOT be reachable via the overlay.
	canary, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer canary.Close()
	accepted := make(chan bool, 1)
	go func() {
		c, e := canary.Accept()
		if e == nil {
			c.Close()
			accepted <- true
		}
	}()
	d, e := proxy.SOCKS5("tcp", "127.0.0.1:"+itoa(bCfg.SocksPort), nil, &net.Dialer{Timeout: 2 * time.Second})
	if e != nil {
		t.Fatal(e)
	}
	blockedCtx, stop := context.WithTimeout(ctx, 2*time.Second)
	conn, e := d.(proxy.ContextDialer).DialContext(blockedCtx, "tcp", net.JoinHostPort(aCfg.VirtualIP, itoa(canary.Addr().(*net.TCPAddr).Port)))
	stop()
	if e == nil {
		conn.Close()
		t.Fatal("unrelated local service exposed")
	}
	select {
	case <-accepted:
		t.Fatal("canary reached")
	default:
	}
	attacker, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	pid, _ := peer.IDFromPrivateKey(attacker)
	forged, _ := invite.NewScoped(attacker, "forged", []string{"/ip4/" + aCfg.VirtualIP + "/tcp/" + itoa(aCfg.AppPort) + "/p2p/" + pid.String()}, "interface")
	if _, e = b.Ping(ctx, InvitePrefix+forged, "no"); e == nil {
		t.Fatal("wrong Noise identity accepted")
	}
	t.Logf("authenticated RTT %.3f ms; unrelated port blocked; wrong peer rejected", result.RTTMillis)
	if path := os.Getenv("EASYTIER_LAB_METRICS"); path != "" {
		report := map[string]any{"scenario": "two real EasyTier helpers on one Windows PC, no TUN; not CGNAT/two houses", "rttMillis": result.RTTMillis, "a": a.State(ctx).Metrics, "b": b.State(ctx).Metrics, "otherLocalServiceBlocked": true, "wrongPeerRejected": true}
		as, bs := a.State(ctx), b.State(ctx)
		report["helperARssBytes"] = as.HelperRSS
		report["helperBRssBytes"] = bs.HelperRSS
		report["helperACpuPercent"] = as.HelperCPU
		report["helperBCpuPercent"] = bs.HelperCPU
		data, _ := json.MarshalIndent(report, "", "  ")
		os.WriteFile(path, data, 0600)
	}
	// Crash the managed helper; bounded supervisor must restart it under a new job.
	b.mu.Lock()
	pidBefore := b.cmd.Process.Pid
	b.cmd.Process.Kill()
	b.mu.Unlock()
	deadline = time.Now().Add(12 * time.Second)
	for {
		st := b.State(ctx)
		if st.Running && st.PID != pidBefore && st.Restarts > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("helper did not restart")
		}
		time.Sleep(200 * time.Millisecond)
	}
	for {
		_, e = b.Ping(ctx, a.State(ctx).Invite, "Recovered after helper crash")
		if e == nil {
			break
		}
		if time.Now().After(deadline.Add(15 * time.Second)) {
			t.Fatal("no stream recovery", e)
		}
		time.Sleep(time.Second)
	}
	b.Close()
	a.Close()
	if a.State(ctx).Running || b.State(ctx).Running {
		t.Fatal("helper still running")
	}
}
func itoa(v int) string { return fmt.Sprint(v) }

func TestRealEasyTierRelay(t *testing.T) {
	if os.Getenv("EASYTIER_LAB_BIN") == "" {
		t.Skip("optional verified helper relay integration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Second)
	defer cancel()
	newLab := func(c Config) *Lab {
		k, _, _ := crypto.GenerateEd25519Key(rand.Reader)
		l, e := Start(ctx, k, c, t.TempDir())
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(l.Close)
		return l
	}
	cCfg := configuration(t, "10.203.0.254")
	relay := newLab(cCfg)
	aCfg := configuration(t, "10.203.0.1")
	bCfg := configuration(t, "10.203.0.2")
	aCfg.ForceRelay = true
	bCfg.ForceRelay = true
	aCfg.Peers = []string{"tcp://127.0.0.1:" + itoa(cCfg.TransportPort)}
	bCfg.Peers = aCfg.Peers
	a := newLab(aCfg)
	b := newLab(bCfg)
	deadline := time.Now().Add(25 * time.Second)
	var r Result
	var e error
	for {
		r, e = b.Ping(ctx, a.State(ctx).Invite, "Through authorized private relay")
		if e == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("relay echo failed", e)
		}
		time.Sleep(time.Second)
	}
	st := b.State(ctx)
	data := strings.Split(st.Diagnostics, "\nConnectors:")[0]
	var rows []map[string]any
	if e = json.Unmarshal([]byte(data), &rows); e != nil {
		t.Fatal(e)
	}
	found := false
	for _, row := range rows {
		if row["ipv4"] == aCfg.VirtualIP {
			t.Logf("relay destination cost=%v, tunnel=%v; RTT %.3f ms", row["cost"], row["tunnel_proto"], r.RTTMillis)
			found = true
			if row["cost"] == "p2p" {
				t.Fatal("unexpected direct route")
			}
		}
	}
	if !found {
		t.Fatal("relay route missing")
	}
	_ = relay
}
