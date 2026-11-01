package easytierlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ma "github.com/multiformats/go-multiaddr"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"projetop2p/internal/invite"
	"projetop2p/internal/telemetry"
	"runtime"
	"strings"
	"sync"
	"time"
)

const labProtocol protocol.ID = "/projeto-p2p/easytier-lab/0.5.2"

type State struct {
	HelperRSS   *uint64          `json:"helperRssBytes"`
	HelperCPU   *float64         `json:"helperCpuPercent"`
	Running     bool             `json:"running"`
	PID         int              `json:"pid"`
	Restarts    int              `json:"restarts"`
	Error       string           `json:"error"`
	ID          string           `json:"id"`
	Invite      string           `json:"invite"`
	VirtualIP   string           `json:"virtualIp"`
	Port        int              `json:"port"`
	Diagnostics string           `json:"diagnostics"`
	Metrics     telemetry.Sample `json:"metrics"`
	Path        string           `json:"path"`
}
type Result struct {
	Authenticated bool    `json:"authenticated"`
	PeerID        string  `json:"peerId"`
	RTTMillis     float64 `json:"rttMillis"`
	Echo          string  `json:"echo"`
	Path          string  `json:"path"`
}
type Lab struct {
	helperAt       time.Time
	helperCPU      float64
	helperPID      int
	mu             sync.Mutex
	ctx            context.Context
	cancel         context.CancelFunc
	done           chan struct{}
	h              host.Host
	c              Config
	core, cli, dir string
	cmd            *exec.Cmd
	jobClose       func()
	running        bool
	restarts       int
	lastError      string
	key            crypto.PrivKey
	bw             *trafficCounter
	sample         telemetry.Sampler
	incoming       chan struct{}
	once           sync.Once
	pingSlot       chan struct{}
}

func Start(parent context.Context, key crypto.PrivKey, c Config, profile string) (*Lab, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return nil, errors.New("EasyTier lab currently validated only on Windows AMD64")
	}
	if e := c.Validate(); e != nil {
		return nil, e
	}
	core, cli, e := helperPaths(c.BinaryDir)
	if e != nil {
		return nil, e
	}
	dir, e := os.MkdirTemp(profile, "easytier-lab-")
	if e != nil {
		return nil, e
	}
	ctx, cancel := context.WithCancel(parent)
	l := &Lab{ctx: ctx, cancel: cancel, done: make(chan struct{}), c: c, core: core, cli: cli, dir: dir, key: key, bw: &trafficCounter{}, incoming: make(chan struct{}, 4), pingSlot: make(chan struct{}, 1)}
	fail := func(e error) (*Lab, error) {
		cancel()
		if l.h != nil {
			l.h.Close()
		}
		os.RemoveAll(dir)
		return nil, e
	}
	if e = os.WriteFile(filepath.Join(dir, "config.toml"), []byte(c.TOML()), 0600); e != nil {
		return fail(e)
	}
	limits := rcmgr.PartialLimitConfig{System: rcmgr.ResourceLimits{Conns: 32, Streams: 32, Memory: 32 << 20, FD: 64}}.Build(rcmgr.DefaultLimits.AutoScale())
	rm, e := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(limits))
	if e != nil {
		return fail(e)
	}
	dial, e := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", c.SocksPort), nil, &net.Dialer{Timeout: 5 * time.Second})
	if e != nil {
		rm.Close()
		return fail(e)
	}
	d, ok := dial.(proxy.ContextDialer)
	if !ok {
		rm.Close()
		return fail(errors.New("SOCKS context dialer required"))
	}
	h, e := libp2p.New(libp2p.Identity(key), libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", c.AppPort)), libp2p.NoTransports, libp2p.DisableRelay(), libp2p.Security(noise.ID, noise.New), libp2p.ResourceManager(rm), libp2p.BandwidthReporter(l.bw), libp2p.Transport(tcp.NewTCPTransport, tcp.WithDialerForAddr(func(a ma.Multiaddr) (tcp.ContextDialer, error) {
		ip, _ := a.ValueForProtocol(ma.P_IP4)
		if !Virtual(ip) || len(a.Protocols()) != 2 {
			return nil, errors.New("lab dials only virtual IPv4/TCP")
		}
		return d, nil
	})))
	if e != nil {
		rm.Close()
		return fail(e)
	}
	l.h = h
	h.SetStreamHandler(labProtocol, func(s network.Stream) {
		select {
		case l.incoming <- struct{}{}:
			defer func() { <-l.incoming }()
		default:
			s.Reset()
			return
		}
		defer s.Close()
		s.SetDeadline(time.Now().Add(8 * time.Second))
		b, e := io.ReadAll(io.LimitReader(s, 4097))
		if e != nil || len(b) > 4096 {
			s.Reset()
			return
		}
		_, _ = s.Write(b)
	})
	go l.supervise()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-l.done:
			l.Close()
			return nil, errors.New("EasyTier helper failed to start; inspect configuration and process limits")
		case <-deadline.C:
			l.Close()
			return nil, errors.New("EasyTier loopback RPC readiness timeout")
		case <-ctx.Done():
			l.Close()
			return nil, ctx.Err()
		case <-tick.C:
			conn, e := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", c.RPCPort), 100*time.Millisecond)
			if e == nil {
				conn.Close()
				return l, nil
			}
		}
	}
}
func (l *Lab) supervise() {
	defer close(l.done)
	for attempt := 0; attempt < 4; attempt++ {
		if l.ctx.Err() != nil {
			return
		}
		if _, _, e := helperPaths(l.c.BinaryDir); e != nil {
			l.mu.Lock()
			l.lastError = e.Error()
			l.mu.Unlock()
			return
		}
		cmd := exec.Command(l.core, "-c", filepath.Join(l.dir, "config.toml"), "--network-name", l.c.Network, "--rpc-portal", fmt.Sprintf("127.0.0.1:%d", l.c.RPCPort), "--rpc-portal-whitelist", "127.0.0.1", "--console-log-level", "error")
		cmd.Dir = l.dir
		for _, v := range os.Environ() {
			if !strings.HasPrefix(strings.ToUpper(v), "ET_") {
				cmd.Env = append(cmd.Env, v)
			}
		}
		cmd.Env = append(cmd.Env, "ET_NETWORK_SECRET="+l.c.Secret)
		prepareProcess(cmd)
		e := cmd.Start()
		var closeJob func()
		if e == nil {
			closeJob, e = isolateProcess(cmd)
			if e != nil {
				cmd.Process.Kill()
				_ = cmd.Wait()
			}
		}
		l.mu.Lock()
		l.cmd = cmd
		l.jobClose = closeJob
		l.running = e == nil
		if e == nil {
			l.lastError = ""
		}
		l.restarts = attempt
		if e != nil {
			l.lastError = "helper isolation/start failed: " + e.Error()
		}
		l.mu.Unlock()
		if e == nil {
			e = cmd.Wait()
			closeJob()
			l.mu.Lock()
			l.running = false
			l.jobClose = nil
			l.lastError = fmt.Sprintf("helper exited: %v", e)
			l.mu.Unlock()
		}
		if l.ctx.Err() != nil {
			return
		}
		select {
		case <-l.ctx.Done():
			return
		case <-time.After(time.Duration(1<<attempt) * time.Second):
		}
	}
}
func (l *Lab) Close() {
	l.once.Do(func() {
		l.cancel()
		l.mu.Lock()
		if l.jobClose != nil {
			l.jobClose()
		}
		if l.cmd != nil && l.cmd.Process != nil {
			_ = l.cmd.Process.Kill()
		}
		l.mu.Unlock()
		<-l.done
		l.h.Close()
		l.bw.Reset()
		_ = os.RemoveAll(l.dir)
	})
}
func (l *Lab) State(ctx context.Context) State {
	l.mu.Lock()
	s := State{Running: l.running, Restarts: l.restarts, Error: l.lastError, ID: l.h.ID().String(), VirtualIP: l.c.VirtualIP, Port: l.c.AppPort, Path: "unknown; inspect EasyTier peer diagnostics (libp2p sees the SOCKS leg)"}
	if l.cmd != nil && l.cmd.Process != nil {
		s.PID = l.cmd.Process.Pid
	}
	if s.Running {
		rss, cpu, ok := helperUsage(s.PID)
		if ok {
			s.HelperRSS = &rss
			if l.helperPID == s.PID && !l.helperAt.IsZero() {
				seconds := time.Since(l.helperAt).Seconds()
				if seconds > 0 {
					usage := 100 * (cpu - l.helperCPU) / seconds
					s.HelperCPU = &usage
				}
			}
			l.helperPID = s.PID
			l.helperCPU = cpu
			l.helperAt = time.Now()
		}
	}
	l.mu.Unlock()
	addr := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", l.c.VirtualIP, l.c.AppPort, l.h.ID())
	raw, e := invite.NewScoped(l.key, "Laboratorio P2P", []string{addr}, "interface")
	if e == nil {
		s.Invite = InvitePrefix + raw
	}
	st := l.bw.GetBandwidthTotals()
	s.Metrics = l.sample.Read(st.TotalOut, st.TotalIn)
	s.Metrics.Connections = len(l.h.Network().Conns())
	if s.Running {
		if e := VerifyBinary(l.cli, CLISHA256); e == nil {
			c, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			cmd := exec.CommandContext(c, l.cli, "-p", fmt.Sprintf("127.0.0.1:%d", l.c.RPCPort), "-o", "json", "peer")
			prepareProcess(cmd)
			var out boundedBuffer
			cmd.Stdout = &out
			if e = cmd.Run(); e == nil && json.Valid(out.b) {
				s.Diagnostics = string(out.b)
				var rows []struct {
					IPv4 string `json:"ipv4"`
					Cost string `json:"cost"`
				}
				if json.Unmarshal(out.b, &rows) == nil {
					paths := []string{}
					for _, r := range rows {
						if r.Cost != "Local" {
							paths = append(paths, r.IPv4+"="+r.Cost)
						}
					}
					if len(paths) > 0 {
						s.Path = "EasyTier: " + strings.Join(paths, ", ")
					} else {
						s.Path = "Nenhum peer EasyTier descoberto; processo ativo não comprova conexão"
					}
				}
				cmd = exec.CommandContext(c, l.cli, "-p", fmt.Sprintf("127.0.0.1:%d", l.c.RPCPort), "-o", "json", "connector")
				prepareProcess(cmd)
				var connOut boundedBuffer
				cmd.Stdout = &connOut
				if e = cmd.Run(); e == nil {
					s.Diagnostics += "\nConnectors: " + string(connOut.b)
				}
			}
		}
	}
	return s
}
func (l *Lab) Ping(ctx context.Context, raw, text string) (Result, error) {
	select {
	case l.pingSlot <- struct{}{}:
		defer func() { <-l.pingSlot }()
	default:
		return Result{}, errors.New("one lab probe at a time")
	}
	if len(text) == 0 || len(text) > 4096 || !strings.HasPrefix(raw, InvitePrefix) {
		return Result{}, errors.New("bounded text and lab invite required")
	}
	i, e := invite.Parse(strings.TrimPrefix(raw, InvitePrefix), time.Now())
	if e != nil {
		return Result{}, e
	}
	if i.Scope != "interface" || len(i.Addresses) != 1 {
		return Result{}, errors.New("one virtual lab endpoint required")
	}
	a, e := ma.NewMultiaddr(i.Addresses[0])
	if e != nil {
		return Result{}, e
	}
	ip, _ := a.ValueForProtocol(ma.P_IP4)
	if !Virtual(ip) {
		return Result{}, errors.New("not a lab virtual address")
	}
	info, e := peer.AddrInfoFromP2pAddr(a)
	if e != nil {
		return Result{}, e
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	start := time.Now()
	if e = l.h.Connect(ctx, *info); e != nil {
		return Result{}, e
	}
	s, e := l.h.NewStream(ctx, info.ID, labProtocol)
	if e != nil {
		return Result{}, e
	}
	defer s.Close()
	s.SetDeadline(time.Now().Add(8 * time.Second))
	if _, e = s.Write([]byte(text)); e != nil {
		return Result{}, e
	}
	if e = s.CloseWrite(); e != nil {
		return Result{}, e
	}
	b, e := io.ReadAll(io.LimitReader(s, 4097))
	if e != nil || string(b) != text {
		return Result{}, errors.New("echo mismatch or failed")
	}
	return Result{Authenticated: true, PeerID: info.ID.String(), Echo: string(b), RTTMillis: float64(time.Since(start).Microseconds()) / 1000, Path: "unknown overlay path; Noise peer verified"}, nil
}

type boundedBuffer struct{ b []byte }

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if len(b.b)+len(p) > 65536 {
		return 0, errors.New("diagnostic exceeds 64 KiB")
	}
	b.b = append(b.b, p...)
	return len(p), nil
}
