// Package overlay adapts EasyTier/etcat to libp2p TCP without exposing OS interfaces.
package overlay

import (
	"bufio"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//go:embed manifest.json
var manifestFS embed.FS

type Manifest struct {
	Version      string `json:"version"`
	SHA          string `json:"sha256"`
	UPnPDisabled bool   `json:"upnpDisabled"`
}

func BinaryManifest() Manifest {
	b, _ := manifestFS.ReadFile("manifest.json")
	var m Manifest
	_ = json.Unmarshal(b, &m)
	return m
}
func VerifyBinary(path string) error {
	m := BinaryManifest()
	if len(m.SHA) != 64 || !m.UPnPDisabled {
		return errors.New("verified userspace helper build unavailable")
	}
	st, e := os.Lstat(path)
	if e != nil {
		return e
	}
	if !st.Mode().IsRegular() || st.Size() > 100<<20 {
		return errors.New("invalid helper file")
	}
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, io.LimitReader(f, 100<<20+1)); e != nil {
		return e
	}
	if hex.EncodeToString(h.Sum(nil)) != m.SHA {
		return errors.New("integrated helper SHA-256 mismatch; execution blocked")
	}
	return nil
}

type child struct {
	routePath atomic.Value
	routeAt   atomic.Int64
	cmd       *exec.Cmd
	cancel    context.CancelFunc
	closeJob  func()
	done      chan struct{}
	once      sync.Once
}

func (c *child) stop() {
	c.once.Do(func() { c.cancel(); c.closeJob(); _ = c.cmd.Process.Kill(); <-c.done })
}
func (c *child) alive() bool {
	select {
	case <-c.done:
		return false
	default:
		return true
	}
}

type mediaSession struct {
	child  *child
	local  int
	remote int
}
type session struct {
	child *child
	socks string
}
type State struct {
	Running         bool   `json:"running"`
	Restarts        int    `json:"restarts"`
	ClientProcesses int    `json:"clientProcesses"`
	Warning         string `json:"warning"`
	Error           string `json:"error"`
	Version         string `json:"version"`
}
type pathSample struct {
	path string
	at   time.Time
}
type Manager struct {
	mu                        sync.Mutex
	ctx                       context.Context
	cancel                    context.CancelFunc
	profile, binary, registry string
	store                     saved
	token                     string
	sessions                  map[string]*session
	starting                  map[string]chan struct{}
	recoveryAt                map[string]time.Time
	media                     map[string]*mediaSession
	paths                     map[string]pathSample
	server                    *child
	restarts                  int
	lastError                 string
	warning                   atomic.Bool
	wg                        sync.WaitGroup
	dialers                   sync.WaitGroup
	once                      sync.Once
	probe                     chan struct{}
}

func Open(parent context.Context, profile, binary, registry string) (*Manager, error) {
	if runtime.GOOS != "windows" {
		return nil, errors.New("integrated beta requires Windows protected credential storage")
	}
	if e := VerifyBinary(binary); e != nil {
		return nil, e
	}
	s, e := loadStore(profile)
	if e != nil {
		return nil, e
	}
	ctx, cancel := context.WithCancel(parent)
	m := &Manager{ctx: ctx, cancel: cancel, profile: profile, binary: binary, store: s, sessions: map[string]*session{}, media: map[string]*mediaSession{}, paths: map[string]pathSample{}, probe: make(chan struct{}, 1)}
	if registry == "" {
		registry = DefaultRegistry
	}
	if !strings.Contains(registry, "public_key") {
		m.warning.Store(true)
	}
	if len(registry) > 16384 {
		cancel()
		return nil, errors.New("relay registry too large")
	}
	m.registry = filepath.Join(profile, "integrated-relays.toml")
	if e = os.WriteFile(m.registry, []byte(registry), 0600); e != nil {
		cancel()
		return nil, e
	}
	return m, nil
}
func (m *Manager) spawn(args []string, target string, server bool, extraEnv ...string) (*child, <-chan string, error) {
	if e := VerifyBinary(m.binary); e != nil {
		return nil, nil, e
	}
	ctx, cancel := context.WithCancel(m.ctx)
	cmd := exec.CommandContext(ctx, m.binary, args...)
	cmd.Dir = m.profile
	cmd.Env = append(cmd.Env, extraEnv...)
	for _, v := range os.Environ() {
		k := strings.ToUpper(strings.SplitN(v, "=", 2)[0])
		if strings.HasPrefix(k, "ET_") || strings.HasPrefix(k, "ETCAT_") || strings.HasPrefix(k, "TAILCAT_") || strings.HasPrefix(k, "P2P_ETCAT_") {
			continue
		}
		cmd.Env = append(cmd.Env, v)
	}
	if target != "" {
		cmd.Env = append(cmd.Env, "P2P_ETCAT_TARGET="+target)
	}
	out, e := cmd.StdoutPipe()
	if e != nil {
		cancel()
		return nil, nil, e
	}
	errOut, e := cmd.StderrPipe()
	if e != nil {
		cancel()
		return nil, nil, e
	}
	prepareProcess(cmd)
	if e = cmd.Start(); e != nil {
		cancel()
		return nil, nil, e
	}
	closeJob, e := isolateProcess(cmd)
	if e != nil {
		cancel()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, nil, e
	}
	c := &child{cmd: cmd, cancel: cancel, closeJob: closeJob, done: make(chan struct{})}
	ready := make(chan string, 1)
	var readers sync.WaitGroup
	readers.Add(2)
	go func() {
		defer readers.Done()
		s := bufio.NewScanner(out)
		s.Buffer(make([]byte, 1024), 16384)
		for s.Scan() {
			if !server {
				var v struct {
					Path string `json:"p2pPath"`
				}
				if json.Unmarshal(s.Bytes(), &v) == nil {
					path := "integrada (caminho não verificado)"
					if v.Path == "direct" {
						path = "direta"
					} else if v.Path == "relay" {
						path = "relay"
					}
					c.routePath.Store(path)
					c.routeAt.Store(time.Now().UnixNano())
				}
			}
			if server {
				var v struct {
					Token string `json:"listenAddr"`
				}
				if json.Unmarshal(s.Bytes(), &v) == nil && validToken(v.Token) {
					select {
					case ready <- v.Token:
					default:
					}
				}
			}
		}
	}()
	go func() {
		defer readers.Done()
		s := bufio.NewScanner(errOut)
		s.Buffer(make([]byte, 1024), 16384)
		for s.Scan() {
			line := s.Text()
			if strings.Contains(line, "no pinned identity") {
				m.warning.Store(true)
			}
			if !server && strings.HasPrefix(line, "SOCKS running at socks5h://127.0.0.1:") {
				select {
				case ready <- strings.TrimPrefix(line, "SOCKS running at socks5h://"):
				default:
				}
			}
		}
	}()
	go func() { readers.Wait(); _ = cmd.Wait(); closeJob(); close(c.done) }()
	return c, ready, nil
}
func waitReady(ctx context.Context, c *child, ready <-chan string) (string, error) {
	timer := time.NewTimer(25 * time.Second)
	defer timer.Stop()
	select {
	case s := <-ready:
		return s, nil
	case <-c.done:
		return "", errors.New("verified helper exited before readiness; check relay connectivity")
	case <-ctx.Done():
		return "", ctx.Err()
	case <-timer.C:
		return "", errors.New("integrated relay startup timed out")
	}
}
func (m *Manager) Start(ctx context.Context, appPort int, mediaPorts ...int) error {
	extraEnv := []string{}
	if len(mediaPorts) > 0 {
		if mediaPorts[0] < 1024 || mediaPorts[0] > 65535 {
			return errors.New("invalid media UDP port")
		}
		extraEnv = append(extraEnv, "P2P_ETCAT_UDP_SERVE="+strconv.Itoa(mediaPorts[0]))
	}
	if appPort < 1024 || appPort > 65535 {
		return errors.New("integrated beta requires unprivileged application port")
	}
	first := make(chan error, 1)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		for attempt := 0; attempt < 4; attempt++ {
			if m.ctx.Err() != nil {
				return
			}
			m.mu.Lock()
			material := append([]byte(nil), m.store.Server...)
			m.mu.Unlock()
			path, erase, e := privateMaterial(m.profile, material)
			var c *child
			var ready <-chan string
			if e == nil {
				c, ready, e = m.spawn([]string{"--key", path, "--relay-file", m.registry, "--serve", strconv.Itoa(appPort), "--json", "--full-address"}, "", true, extraEnv...)
			}
			var token string
			if e == nil {
				token, e = waitReady(m.ctx, c, ready)
			}
			if erase != nil {
				erase()
			}
			m.mu.Lock()
			m.server = c
			m.restarts = attempt
			if e == nil {
				if m.token != "" && m.token != token {
					e = errors.New("persistent transport token changed unexpectedly")
				} else {
					m.token = token
					m.lastError = ""
				}
			}
			if e != nil {
				m.lastError = "Integrated helper/relay unavailable; no automatic switch to another connectivity mode"
			}
			m.mu.Unlock()
			if attempt == 0 {
				first <- e
			}
			if e == nil {
				<-c.done
			}
			if c != nil {
				c.stop()
			}
			if m.ctx.Err() != nil {
				return
			}
			m.mu.Lock()
			m.lastError = "Auxiliar interrompido; tentando reconectar sem alterar identidade"
			m.mu.Unlock()
			select {
			case <-m.ctx.Done():
				return
			case <-time.After(time.Duration(1<<attempt) * time.Second):
			}
		}
	}()
	select {
	case e := <-first:
		return e
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (m *Manager) Token() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.server == nil || !m.server.alive() {
		return ""
	}
	return m.token
}
func (m *Manager) Register(id, token string) error {
	if _, e := peer.Decode(id); e != nil {
		return e
	}
	if !validToken(token) {
		return errors.New("invalid integrated route credential")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.store.Routes[id]; !ok && len(m.store.Routes) >= 256 {
		return errors.New("integrated route limit")
	}
	old := m.store.Routes[id]
	m.store.Routes[id] = token
	if e := saveStore(m.profile, m.store); e != nil {
		if old == "" {
			delete(m.store.Routes, id)
		} else {
			m.store.Routes[id] = old
		}
		return e
	}
	if old != "" && old != token {
		if media := m.media[id]; media != nil {
			media.child.stop()
			delete(m.media, id)
		}
		if s := m.sessions[id]; s != nil {
			s.child.stop()
			delete(m.sessions, id)
		}
	}
	return nil
}
func (m *Manager) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, e := net.SplitHostPort(address)
	if e != nil {
		return nil, e
	}
	number, e := strconv.Atoi(port)
	if e != nil || number < 1024 || number > 65535 {
		return nil, errors.New("invalid integrated destination port")
	}
	m.mu.Lock()
	id := ""
	token := ""
	for key, value := range m.store.Routes {
		if LogicalIP(key) == host {
			id, token = key, value
			break
		}
	}
	if id == "" {
		m.mu.Unlock()
		return nil, errors.New("no authenticated invitation route for synthetic address")
	}
	m.mu.Unlock()
	s, e := m.dialSession(ctx, id, token)
	if e != nil {
		return nil, e
	}
	d, e := proxy.SOCKS5("tcp", s.socks, nil, &net.Dialer{Timeout: 5 * time.Second})
	if e != nil {
		return nil, e
	}
	conn, e := d.(proxy.ContextDialer).DialContext(ctx, "tcp", net.JoinHostPort("server.etcat", port))
	if e != nil && !errors.Is(ctx.Err(), context.Canceled) {
		m.mu.Lock()
		if m.recoveryAt == nil {
			m.recoveryAt = map[string]time.Time{}
		}
		reset := m.sessions[id] == s && time.Since(m.recoveryAt[id]) >= 5*time.Second
		if reset {
			delete(m.sessions, id)
			m.recoveryAt[id] = time.Now()
		}
		m.mu.Unlock()
		if reset {
			s.child.stop()
		}
	}
	return conn, e
}

// One helper startup per peer; unrelated peers never wait on the global mutex.
func (m *Manager) dialSession(ctx context.Context, id, token string) (*session, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	m.mu.Lock()
	if m.ctx.Err() != nil {
		m.mu.Unlock()
		return nil, m.ctx.Err()
	}
	m.dialers.Add(1)
	m.mu.Unlock()
	defer m.dialers.Done()
	for {
		m.mu.Lock()
		if m.ctx.Err() != nil {
			m.mu.Unlock()
			return nil, m.ctx.Err()
		}
		if m.store.Routes[id] != token {
			m.mu.Unlock()
			return nil, errors.New("integrated route changed during dial")
		}
		if s := m.sessions[id]; s != nil && s.child.alive() {
			m.mu.Unlock()
			return s, nil
		}
		if m.starting == nil {
			m.starting = map[string]chan struct{}{}
		}
		if wait := m.starting[id]; wait != nil {
			m.mu.Unlock()
			select {
			case <-wait:
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-m.ctx.Done():
				return nil, m.ctx.Err()
			}
		}
		old := m.sessions[id]
		delete(m.sessions, id)
		if len(m.sessions)+len(m.starting) >= MaxSessions {
			m.mu.Unlock()
			if old != nil {
				old.child.stop()
			}
			return nil, errors.New("integrated active peer process limit reached")
		}
		readySlot := make(chan struct{})
		m.starting[id] = readySlot
		m.mu.Unlock()
		if old != nil {
			old.child.stop()
		}
		child, ready, e := m.spawn([]string{"--key", "new", "--relay-file", m.registry, "socks", "--listen", "127.0.0.1:0"}, token, false)
		var socks string
		if e == nil {
			startCtx, cancel := context.WithCancel(ctx)
			stop := context.AfterFunc(m.ctx, cancel)
			socks, e = waitReady(startCtx, child, ready)
			stop()
			cancel()
		}
		var created *session
		m.mu.Lock()
		if e == nil && (m.ctx.Err() != nil || m.store.Routes[id] != token) {
			e = errors.New("integrated startup superseded")
		}
		if e == nil {
			created = &session{child: child, socks: socks}
			m.sessions[id] = created
		}
		delete(m.starting, id)
		close(readySlot)
		m.mu.Unlock()
		if e != nil {
			if child != nil {
				child.stop()
			}
			return nil, e
		}
		return created, nil
	}
}

func (m *Manager) Path(id string) string {
	m.mu.Lock()
	s := m.sessions[id]
	m.mu.Unlock()
	if s == nil || !s.child.alive() || time.Since(time.Unix(0, s.child.routeAt.Load())) > 6*time.Second {
		return "integrada (caminho não verificado)"
	}
	v := s.child.routePath.Load()
	if v == nil {
		return "integrada (caminho não verificado)"
	}
	return v.(string)
}
func (m *Manager) ProbePath(ctx context.Context, id string) string { return m.Path(id) }
func cleanEnvironment() []string {
	out := []string{}
	for _, v := range os.Environ() {
		k := strings.ToUpper(strings.SplitN(v, "=", 2)[0])
		if !strings.HasPrefix(k, "ET_") && !strings.HasPrefix(k, "ETCAT_") && !strings.HasPrefix(k, "TAILCAT_") && !strings.HasPrefix(k, "P2P_ETCAT_") {
			out = append(out, v)
		}
	}
	return out
}
func (m *Manager) State() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := State{Restarts: m.restarts, ClientProcesses: len(m.sessions), Error: m.lastError, Version: BinaryManifest().Version}
	s.Running = m.server != nil && m.server.alive()
	if m.warning.Load() {
		s.Warning = "Relay comunitário sem chave pública fixada; disponibilidade e identidade do relay não garantidas. Peer ID do aplicativo é verificado por Noise."
	}
	return s
}
func (m *Manager) Close() {
	m.once.Do(func() {
		m.cancel()
		m.mu.Lock()
		server := m.server
		clients := []*child{}
		for _, s := range m.media {
			clients = append(clients, s.child)
		}
		for _, s := range m.sessions {
			clients = append(clients, s.child)
		}
		m.mu.Unlock()
		if server != nil {
			server.stop()
		}
		for _, c := range clients {
			c.stop()
		}
		m.dialers.Wait()
		m.wg.Wait()
	})
}

// HasRoute exposes only availability; credentials never leave the sealed manager.
func (m *Manager) HasRoute(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.store.Routes[id] != ""
}
