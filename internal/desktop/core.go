package desktop

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"projetop2p/internal/adapters"
	"projetop2p/internal/communities"
	"projetop2p/internal/community"
	"projetop2p/internal/endpoint"
	"projetop2p/internal/identity"
	"projetop2p/internal/invite"
	"projetop2p/internal/localdb"
	"projetop2p/internal/messaging"
	"projetop2p/internal/network"
	"projetop2p/internal/overlay"
	"projetop2p/internal/profile"
	"projetop2p/internal/roomprobe"
	"projetop2p/internal/telemetry"
	"projetop2p/internal/transportaddr"
	"projetop2p/internal/voice"
)

type Core struct {
	roomProbe      roomProbeState
	voiceDiag      voiceDiagnosticCache
	captureFactory voice.CaptureFactory
	media          atomic.Pointer[voice.Service]
	activity       activityState
	adapterChecked time.Time
	voice          *voice.Service
	legacyLabState
	integrated *overlay.Manager
	mu         sync.Mutex
	db         *localdb.DB
	node       *network.Node
	ctx        context.Context
	cancel     context.CancelFunc
	dir        string
	lastEvent  string
	lastTest   *ConnectionTest
	groups     *communities.Service
	profiles   *profile.Store
	enumerate  adapters.Provider
	selection  adapters.Selection
	networkErr string
	binding    string
	key        crypto.PrivKey
	wg         sync.WaitGroup
	closed     bool
	metrics    telemetry.Sampler
}
type ContactView struct {
	localdb.Contact
	Path string `json:"path"`
}
type State struct {
	PeerTraffic  map[string]network.PeerTraffic `json:"peerTraffic"`
	Unread       map[string]int                 `json:"unread"`
	Settings     localdb.Settings               `json:"settings"`
	Network      network.Status                 `json:"network"`
	Contacts     []ContactView                  `json:"contacts"`
	Messages     []localdb.Message              `json:"messages"`
	Notice       string                         `json:"notice"`
	Event        string                         `json:"event"`
	Connectivity adapters.Selection             `json:"connectivity"`
	NetworkError string                         `json:"networkError"`
	Diagnostic   *ConnectionTest                `json:"diagnostic"`
	Communities  []communities.Summary          `json:"communities"`
	Integrated   overlay.State                  `json:"integrated"`
	Metrics      telemetry.Sample               `json:"metrics"`
}

func DefaultDir() (string, error) {
	if d := os.Getenv("PROJETOP2P_DATA"); d != "" {
		return filepath.Abs(d)
	}
	d, e := os.UserConfigDir()
	if e != nil {
		return "", e
	}
	return filepath.Join(d, "ProjetoP2P"), nil
}
func Open(parent context.Context, dir string) (*Core, error) {
	return OpenWithProvider(parent, dir, adapters.List)
}
func OpenWithProvider(parent context.Context, dir string, provider adapters.Provider) (*Core, error) {
	key, e := identity.Load(dir)
	if e != nil {
		return nil, e
	}
	db, e := localdb.Open(dir)
	if e != nil {
		return nil, e
	}
	ctx, cancel := context.WithCancel(parent)
	c := &Core{activity: newActivityState(), db: db, ctx: ctx, cancel: cancel, dir: dir, enumerate: provider, key: key}
	settings, e := db.Settings()
	if e != nil {
		cancel()
		db.Close()
		return nil, e
	}
	c.profiles, e = profile.Open(dir, key, settings.Name)
	if e != nil {
		cancel()
		db.Close()
		return nil, e
	}
	c.groups = communities.NewWithAssets(ctx, db, key, c.profiles)
	_ = c.db.PollInbox(c.selfID(), nil)
	c.synchronize(settings)
	c.wg.Add(2)
	go c.watch()
	go c.activityWatch()
	return c, nil
}
func (c *Core) start(s localdb.Settings) error {
	cfg := c.config(s)
	if mode(s) == "integrated" {
		binaryDir := s.IntegratedBinaryDir
		if binaryDir == "" {
			exe, e := os.Executable()
			if e != nil {
				return e
			}
			binaryDir = filepath.Join(filepath.Dir(exe), "easytier")
		}
		registry, e := overlay.Registry(s.IntegratedRelays, s.IntegratedRelayKey)
		if e != nil {
			return e
		}
		m, e := overlay.Open(c.ctx, c.dir, filepath.Join(binaryDir, "etcat-p2p.exe"), registry)
		if e != nil {
			return e
		}
		c.integrated = m
		id, _ := peer.IDFromPrivateKey(c.key)
		servicePort := transportaddr.ServicePort(id.String())
		cfg.Listen = []string{fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", servicePort)}
		cfg.IntegratedDialer = m
		cfg.IntegratedAddresses = []string{fmt.Sprintf("/ip6/%s/tcp/%d", transportaddr.LogicalIP(id.String()), servicePort)}
		cfg.InvitationRoute = m.Token
		cfg.RegisterRoute = m.Register
		cfg.HasRoute = m.HasRoute
		cfg.OverlayPath = m.Path
	}
	scope := ""
	if mode(s) != "auto" {
		scope = mode(s)
	}
	cfg.OnMessage = func(m messaging.Message) error { return c.receiveScoped(m, scope) }
	n, e := network.New(c.ctx, cfg)
	if e != nil {
		if c.integrated != nil {
			c.integrated.Close()
			c.integrated = nil
		}
		return e
	}
	v, e := c.startVoice(n, s)
	if e != nil {
		n.Close()
		if c.integrated != nil {
			c.integrated.Close()
			c.integrated = nil
		}
		return e
	}
	if c.integrated != nil {
		if e = c.integrated.Start(c.ctx, transportaddr.ServicePort(n.SelfID()), v.Port()); e != nil {
			v.Close()
			c.integrated.Close()
			c.integrated = nil
			n.Close()
			return e
		}
	}
	c.node = n
	c.voice = v
	c.media.Store(v)
	n.SetVoiceHandler(v.Handle)
	n.SetVoiceMediaHandler(v.AcceptMedia)
	roomprobe.SetHandler(n.Host, func(id peer.ID) (func() bool, bool) {
		return c.acceptRoomProbe(v, id)
	})
	n.SetScreenMediaHandler(v.AcceptScreen)
	n.SetScreenAudioHandler(v.AcceptScreenAudio)
	n.SetCommunityHandler(func(ctx context.Context, remote string, req community.Request) community.Response {
		if strings.HasPrefix(req.Method, "activity.") {
			return c.activityHandle(remote, req)
		}
		return c.groups.Handle(ctx, remote, req)
	})
	c.groups.Attach(n, mode(s))
	if c.profiles != nil {
		n.SetProfileStore(c.profiles, c.groups.AuthorizeArtwork)
	}
	contacts, e := c.db.Contacts()
	if e != nil {
		v.Close()
		c.voice = nil
		n.Close()
		c.node = nil
		return e
	}
	for _, p := range contacts {
		c.remember(p)
	}
	return nil
}
func (c *Core) remember(p localdb.Contact) {
	s, e := c.db.Settings()
	if e != nil {
		return
	}
	if mode(s) == "auto" && p.Scope != "" {
		return
	}
	info := peer.AddrInfo{}
	for _, a := range p.Addresses {
		if c.allowedAddress(a, s) {
			v, e := network.ParseAddress(a)
			if e == nil {
				info.ID = v.ID
				info.Addrs = append(info.Addrs, v.Addrs...)
			}
		}
	}
	if len(info.Addrs) > 0 {
		c.node.Remember(info)
	}
}
func (c *Core) receiveScoped(m messaging.Message, scope string) error {
	contacts, e := c.db.Contacts()
	if e != nil {
		return e
	}
	found := false
	for _, p := range contacts {
		if p.ID == m.From {
			found = true
			break
		}
	}
	if !found {
		if e = c.db.Contact(localdb.Contact{ID: m.From, Name: m.From[:12], Addresses: []string{}, Scope: scope, Unsolicited: true}); e != nil {
			return e
		}
	}
	return c.db.AddMessage(localdb.Message{ID: m.ID, Peer: m.From, From: m.From, Text: m.Text, Time: m.Time, State: "recebida"})
}
func (c *Core) Close() error {
	c.cancel()
	c.closeLegacyLab()
	if c.groups != nil {
		c.groups.Close()
	}
	c.wg.Wait()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	c.stopNode()
	return c.db.Close()
}
func (c *Core) State(selected string) (State, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := State{Contacts: []ContactView{}, Messages: []localdb.Message{}}
	var e error
	s.Settings, e = c.db.Settings()
	if e != nil {
		return s, e
	}
	c.synchronize(s.Settings)
	s.Settings, e = c.db.Settings()
	if e != nil {
		return s, e
	}
	s.Unread, _ = c.db.Unread()
	s.Diagnostic = c.lastTest
	s.Communities, e = c.groups.List()
	if e != nil {
		return s, e
	}
	s.Connectivity = c.selection
	s.NetworkError = c.networkErr
	id, _ := peer.IDFromPrivateKey(c.key)
	s.Network = network.Status{ID: id.String(), Addresses: []string{}, Peers: []network.PeerStatus{}, LocalOnly: mode(s.Settings) != "auto"}
	if c.node != nil {
		s.Network = c.node.Status()
	}
	if c.node != nil {
		for {
			select {
			case ev := <-c.node.Events:
				c.lastEvent = ev
			default:
				goto drained
			}
		}
	drained:
	}
	traffic := network.Traffic{}
	if c.node != nil {
		traffic = c.node.Traffic()
	}
	s.Metrics = c.metrics.Read(traffic.UploadBytes, traffic.DownloadBytes)
	s.PeerTraffic = traffic.Peers
	s.Metrics.Connections = traffic.Connections
	s.Metrics.Streams = traffic.Streams
	s.Metrics.LastConnectMs = traffic.LastConnectMs
	if c.lastTest != nil && c.lastTest.Connected {
		rtt := float64(c.lastTest.RTTMillis)
		if c.lastTest.RTTMicros > 0 {
			rtt = float64(c.lastTest.RTTMicros) / 1000
		}
		s.Metrics.PingMs = &rtt
	}
	if !network.DHTEnabled && mode(s.Settings) == "auto" {
		s.Notice = "DHT provider em quarentena no beta por GO-2024-3218; use convites, bootstraps e relays explícitos."
	}
	if c.integrated != nil {
		s.Integrated = c.integrated.State()
	}
	s.Event = c.lastEvent
	contacts, e := c.db.Contacts()
	if e != nil {
		return s, e
	}
	for _, p := range contacts {
		path := "offline"
		for _, v := range s.Network.Peers {
			if v.ID == p.ID {
				path = v.Path
			}
		}
		if c.profiles != nil {
			if item, ok := c.profiles.Get(p.ID); ok {
				p.Name = item.DisplayName
			}
		}
		s.Contacts = append(s.Contacts, ContactView{Contact: p, Path: path})
	}
	if selected != "" {
		s.Messages, e = c.db.Messages(selected)
	}
	if len(s.Network.Peers) == 0 {
		s.Notice = "Sem conexão com outros participantes. Adicione um convite ou configure bootstraps e relays comunitários."
	} else if s.Settings.Assisted && s.Network.RelayAddresses == 0 {
		s.Notice = "Conectado a participantes; ainda sem endereço de relay reservado para receber conexões pela internet."
	} else {
		s.Notice = "Rede ativa. O caminho de cada contato aparece na conversa."
	}
	if mode(s.Settings) != "auto" {
		s.Notice = c.selection.Message
		if c.networkErr != "" {
			s.Notice = c.networkErr
		}
		if len(s.Network.Peers) > 0 {
			s.Notice = "Peer P2P conectado pela interface selecionada. A VPN pode usar retransmissão própria."
		}
	}
	if !network.DHTEnabled && mode(s.Settings) == "auto" {
		s.Notice += " DHT provider desativada neste beta por segurança."
	}
	return s, e
}
func (c *Core) Invite() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, e := c.db.Settings()
	if e != nil {
		return "", e
	}
	c.synchronize(s)
	if c.node == nil {
		return "", errors.New("rede indisponível: " + c.networkErr)
	}
	addrs := []string{}
	for _, raw := range c.node.Addresses() {
		a, err := ma.NewMultiaddr(raw)
		if err == nil && endpoint.Allowed(a, mode(s)) {
			addrs = append(addrs, raw)
		}
	}
	if len(addrs) == 0 {
		return "", errors.New("nenhum endereço válido anunciado; escolha e ative a interface antes de gerar convite")
	}
	if len(addrs) > 16 {
		addrs = addrs[:16]
	}
	if mode(s) == "integrated" {
		if c.integrated == nil {
			return "", errors.New("integrated connectivity unavailable")
		}
		return invite.NewIntegrated(c.key, s.Name, addrs, c.integrated.Token())
	}
	if mode(s) != "auto" {
		return invite.NewScoped(c.key, s.Name, addrs, mode(s))
	}
	return invite.New(c.key, s.Name, addrs, s.Relays)
}
func (c *Core) AddContact(raw string) error {
	i, e := invite.Parse(strings.TrimSpace(raw), time.Now())
	if e != nil {
		return e
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.node == nil {
		return errors.New("rede indisponível")
	}
	if i.ID == c.node.Host.ID().String() {
		return errors.New("este convite é do seu próprio perfil")
	}
	settings, e := c.db.Settings()
	if e != nil {
		return e
	}
	if requiredMode(i.Scope, settings) != "" {
		return fmt.Errorf("este convite usa %s; ative esse modo antes de conectar", adapters.Name(i.Scope))
	}
	addresses := []string{}
	for _, a := range i.Addresses {
		if c.allowedAddress(a, settings) {
			addresses = append(addresses, a)
		}
	}
	if len(addresses) == 0 {
		return errors.New("o convite não contém endereço compatível; peça um novo convite gerado no modo Radmin/Hamachi correto")
	}
	scope := i.Scope
	if scope == "" && mode(settings) != "auto" {
		scope = mode(settings)
	}
	if scope == "integrated" {
		if c.integrated == nil {
			return errors.New("integrated transport not active")
		}
		if e = c.integrated.Register(i.ID, i.Route); e != nil {
			return e
		}
	}
	p := localdb.Contact{ID: i.ID, Name: i.Name, Addresses: addresses, PublicKey: i.PublicKey, Scope: scope}
	if e = c.db.Contact(p); e != nil {
		return e
	}
	c.remember(p)
	// Learning a relay is explicit through a signed invite. Infrastructure changes
	// require SaveSettings; a contact cannot silently replace the user's network.
	return nil
}
func validateSettings(s localdb.Settings) error {
	if mode(s) != "auto" && !adapters.IsVPN(mode(s)) && mode(s) != "interface" && mode(s) != "integrated" {
		return errors.New("modo inválido")
	}
	if mode(s) == "integrated" {
		if port(s) < 1024 {
			return errors.New("integrated beta requires an unprivileged TCP port")
		}
		if len(s.IntegratedBinaryDir) > 1024 {
			return errors.New("invalid helper path")
		}
		if _, e := overlay.Registry(s.IntegratedRelays, s.IntegratedRelayKey); e != nil {
			return e
		}
	}
	if s.Port < 0 || s.Port > 65535 {
		return errors.New("porta deve estar entre 1 e 65535")
	}
	if len(s.InterfaceID) > 256 {
		return errors.New("identificador de interface inválido")
	}
	if len(s.Name) == 0 || len(s.Name) > 80 || !utf8.ValidString(s.Name) {
		return errors.New("nome deve conter de 1 a 80 bytes UTF-8")
	}
	if len(s.Bootstrap) > 16 || len(s.Relays) > 16 {
		return errors.New("máximo de 16 bootstraps e 16 relays")
	}
	for _, a := range append(append([]string{}, s.Bootstrap...), s.Relays...) {
		if len(a) > 1024 {
			return errors.New("endereço muito grande")
		}
		if _, e := network.ParseAddress(a); e != nil {
			return e
		}
	}
	return nil
}
func (c *Core) SaveSettings(s localdb.Settings) error {
	if e := validateSettings(s); e != nil {
		return e
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.saveSettingsLocked(s)
}
func (c *Core) saveSettingsLocked(s localdb.Settings) error {
	if c.profiles != nil {
		previous, err := c.db.Settings()
		if err != nil {
			return err
		}
		if previous.Name != s.Name {
			local := c.profiles.Local()
			updated, err := c.profiles.Update(s.Name, local.DisplayFontID, local.AvatarHash)
			if err != nil {
				return err
			}
			s.Name = updated.DisplayName
			if c.node != nil && updated.Revision != local.Revision {
				c.node.ProfileChanged()
			}
		}
	}
	s.Mode = mode(s)
	if s.Mode != "auto" {
		s.Port = port(s)
	}
	if e := c.db.SaveSettings(s); e != nil {
		return e
	}
	c.synchronize(s)
	if c.networkErr != "" {
		return errors.New("Configuração salva, mas o nó não iniciou: " + c.networkErr)
	}
	return nil
}
func (c *Core) Send(id, text string) error {
	c.mu.Lock()
	node := c.node
	integrated := c.integrated != nil
	c.mu.Unlock()
	if node == nil {
		return errors.New("rede indisponível")
	}
	p, e := peer.Decode(id)
	if e != nil {
		return e
	}
	contacts, e := c.db.Contacts()
	if e != nil {
		return e
	}
	known := false
	for _, v := range contacts {
		if v.ID == id {
			known = true
		}
	}
	if !known {
		return errors.New("adicione o contato por convite antes de enviar")
	}
	m, e := messaging.New(c.key, p, text)
	if e != nil {
		return e
	}
	record := localdb.Message{ID: m.ID, Peer: id, From: m.From, Text: m.Text, Time: m.Time, State: "enviando"}
	if e = c.db.AddMessage(record); e != nil {
		return e
	}
	timeout := 12 * time.Second
	if integrated {
		timeout = 45 * time.Second
	}
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	defer cancel()
	e = node.Deliver(ctx, m)
	record.State = "confirmada"
	if e != nil {
		record.State = "sem confirmação"
	}
	if dbErr := c.db.AddMessage(record); dbErr != nil {
		return dbErr
	}
	if e != nil {
		return errors.New("Sem confirmacao; pode ter havido recebimento. Verifique modo, interface, IP, firewall ou relays: " + e.Error())
	}
	return nil
}

func (c *Core) config(s localdb.Settings) network.Config {
	cfg := network.Config{DataDir: c.dir, Listen: []string{"/ip4/0.0.0.0/tcp/0"}, Bootstrap: s.Bootstrap, Relays: s.Relays, ForcePrivate: s.Assisted, PublicDiscovery: true}
	if mode(s) == "integrated" {
		return network.Config{DataDir: c.dir, Listen: []string{fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port(s))}, LocalOnly: true, ScopedIP: "127.0.0.1", ScopedMode: "integrated"}
	}
	if mode(s) != "auto" {
		cfg = network.Config{DataDir: c.dir, Listen: []string{fmt.Sprintf("/ip4/%s/tcp/%d", c.selection.IP, port(s))}, LocalOnly: true, ScopedIP: c.selection.IP, ScopedMode: mode(s)}
	}
	return cfg
}

// SelectNetwork persists only the user's transport choice, preserving unrelated profile data.
// Persistence precedes startup: a failed selected transport never silently falls back.
func (c *Core) SelectNetwork(selected, interfaceID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, e := c.db.Settings()
	if e != nil {
		return e
	}
	s.Mode = selected
	s.InterfaceID = interfaceID
	if e = validateSettings(s); e != nil {
		return e
	}
	return c.saveSettingsLocked(s)
}
