package desktop

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	libnet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	ma "github.com/multiformats/go-multiaddr"
	"projetop2p/internal/adapters"
	"projetop2p/internal/endpoint"
	"projetop2p/internal/firewall"
	"projetop2p/internal/invite"
	"projetop2p/internal/localdb"
	"projetop2p/internal/network"
)

func mode(s localdb.Settings) string {
	if s.Mode == "" {
		return "auto"
	}
	return s.Mode
}
func port(s localdb.Settings) int {
	if s.Port == 0 {
		return 4001
	}
	return s.Port
}
func (c *Core) synchronize(s localdb.Settings) {
	if c.ctx.Err() != nil {
		return
	}
	if mode(s) == "integrated" {
		c.synchronizeIntegrated(s)
		return
	}
	items, e := c.enumerate()
	if e != nil {
		c.selection = adapters.Selection{Mode: mode(s), Adapters: []adapters.Info{}, Message: "Não foi possível ler adaptadores: " + e.Error()}
		if mode(s) != "auto" {
			c.stopNode()
			c.networkErr = c.selection.Message
			return
		}
	}
	selection, selectErr := adapters.Select(items, mode(s), s.InterfaceID)
	if e == nil {
		c.selection = selection
	}
	if mode(s) != "auto" && selectErr != nil {
		c.stopNode()
		c.networkErr = selectErr.Error()
		return
	}
	// Pin the resolved GUID so a reboot or a second adapter cannot change the choice.
	if mode(s) != "auto" && s.InterfaceID == "" && c.selection.Selected != nil {
		s.InterfaceID = c.selection.Selected.ID
		s.Mode = mode(s)
		s.Port = port(s)
		if err := c.db.SaveSettings(s); err != nil {
			c.stopNode()
			c.networkErr = "Não foi possível persistir a interface: " + err.Error()
			return
		}
	}
	fingerprint := fmt.Sprintf("%s|%s|%s|%d|%v|%v|%t", mode(s), s.InterfaceID, c.selection.IP, port(s), s.Bootstrap, s.Relays, s.Assisted)
	if c.node != nil && c.binding == fingerprint {
		return
	}
	c.stopNode()
	if e = c.start(s); e != nil {
		c.networkErr = "Não foi possível iniciar o nó: " + e.Error()
		return
	}
	c.binding = fingerprint
	c.networkErr = ""
}
func (c *Core) stopNode() {
	if c.node != nil {
		c.node.SetVoiceHandler(nil)
	}
	if c.voice != nil {
		c.voice.Close()
		c.voice = nil
	}
	if c.integrated != nil {
		c.integrated.Close()
		c.integrated = nil
	}
	if c.groups != nil {
		c.groups.Attach(nil, c.selection.Mode)
	}
	if c.node != nil {
		_ = c.node.Close()
		c.node = nil
	}
	c.binding = ""
}
func (c *Core) watch() {
	defer c.wg.Done()
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-tick.C:
			c.mu.Lock()
			s, e := c.db.Settings()
			if e == nil {
				c.synchronize(s)
				c.refreshAddresses()
			}
			c.mu.Unlock()
		}
	}
}
func (c *Core) allowedAddress(raw string, s localdb.Settings) bool {
	a, e := ma.NewMultiaddr(raw)
	if e != nil {
		return false
	}
	return endpoint.Allowed(a, mode(s))
}
func (c *Core) refreshAddresses() {
	if c.node == nil {
		return
	}
	s, e := c.db.Settings()
	if e != nil {
		return
	}
	contacts, e := c.db.Contacts()
	if e != nil {
		return
	}
	for _, p := range contacts {
		if requiredMode(p.Scope, s) != "" {
			continue
		}
		id, e := peer.Decode(p.ID)
		if e != nil || len(c.node.Host.Network().ConnsToPeer(id)) == 0 {
			continue
		}
		seen := map[string]bool{}
		out := []string{}
		for _, a := range c.node.Host.Peerstore().Addrs(id) {
			raw := a.String() + "/p2p/" + p.ID
			if c.allowedAddress(raw, s) && !seen[raw] && len(out) < 16 {
				out = append(out, raw)
				seen[raw] = true
			}
		}
		if len(out) > 0 && strings.Join(out, "\n") != strings.Join(p.Addresses, "\n") {
			p.Addresses = out
			_ = c.db.Contact(p)
			c.remember(p)
		}
	}
}

type ConnectionTarget struct {
	Address string `json:"address"`
	IP      string `json:"ip"`
	Port    int    `json:"port"`
}
type ConnectionTest struct {
	ID              string             `json:"id"`
	Connected       bool               `json:"connected"`
	Authenticated   bool               `json:"authenticated"`
	ContactSaved    bool               `json:"contactSaved"`
	RequiredMode    string             `json:"requiredMode,omitempty"`
	Path            string             `json:"path"`
	RTTMillis       int64              `json:"rttMillis"`
	RTTMicros       int64              `json:"rttMicros"`
	Message         string             `json:"message"`
	Code            string             `json:"code"`
	Mode            string             `json:"mode"`
	Interface       string             `json:"interface"`
	InterfaceID     string             `json:"interfaceId"`
	InterfaceIndex  int                `json:"interfaceIndex"`
	LocalIP         string             `json:"localIp"`
	LocalPort       int                `json:"localPort"`
	ListenAddresses []string           `json:"listenAddresses"`
	Targets         []ConnectionTarget `json:"targets"`
	LocalAddress    string             `json:"localAddress"`
	RemoteAddress   string             `json:"remoteAddress"`
	Firewall        firewall.Report    `json:"firewall"`
}

func requiredMode(scope string, s localdb.Settings) string {
	if scope == "integrated" && mode(s) != "integrated" {
		return "integrated"
	}
	if (scope == "" || scope == "auto") && mode(s) == "integrated" {
		return "auto"
	}
	if adapters.IsVPN(scope) && mode(s) != scope && mode(s) != "interface" {
		return scope
	}
	if scope == "interface" && mode(s) == "auto" {
		return "interface"
	}
	return ""
}
func (c *Core) newTest(id string) ConnectionTest {
	s, _ := c.db.Settings()
	r := ConnectionTest{ID: id, Path: "offline", Mode: mode(s), LocalIP: c.selection.IP, LocalPort: port(s), Targets: []ConnectionTarget{}, ListenAddresses: []string{}}
	if i := c.selection.Selected; i != nil {
		r.Interface = i.Name
		r.InterfaceID = i.ID
		r.InterfaceIndex = i.Index
	}
	if c.node != nil {
		for _, a := range c.node.Host.Network().ListenAddresses() {
			r.ListenAddresses = append(r.ListenAddresses, a.String())
			if value, e := a.ValueForProtocol(ma.P_TCP); e == nil {
				if n, e := strconv.Atoi(value); e == nil && n > 0 {
					r.LocalPort = n
				}
			}
		}
	}
	return r
}
func (c *Core) probeLocked(ctx context.Context, info peer.AddrInfo, r *ConnectionTest) {
	for _, a := range info.Addrs {
		ip, n, _ := endpoint.TCP4(a)
		r.Targets = append(r.Targets, ConnectionTarget{Address: a.String() + "/p2p/" + info.ID.String(), IP: ip, Port: n})
	}
	if len(info.Addrs) == 0 {
		r.Code = "no_compatible_address"
		r.Message = "Nenhum endereço compatível. Peça um novo convite gerado com a VPN correta selecionada"
		return
	}
	if e := c.node.Connect(ctx, info); e != nil {
		r.Code = "dial_failed"
		r.Message = fmt.Sprintf("Peer não respondeu. Modo: %s; interface: %s (índice %d); origem: %s; porta de escuta: %d; destinos: %v. Confirme o convite atual, a mesma rede VPN e o aplicativo aberto. A falha não comprova bloqueio no firewall. Detalhe: %v", adapters.Name(r.Mode), r.Interface, r.InterfaceIndex, r.LocalIP, r.LocalPort, r.Targets, e)
		if r.Mode == "auto" && c.selection.Detected {
			r.Message += ". VPN detectada, mas não selecionada: ative Radmin/Hamachi nos dois PCs e gere um novo convite"
		}
		return
	}
	r.Authenticated = true
	for _, conn := range c.node.Host.Network().ConnsToPeer(info.ID) {
		r.LocalAddress = conn.LocalMultiaddr().String()
		r.RemoteAddress = conn.RemoteMultiaddr().String()
		r.Path = "direta"
		if conn.Stat().Limited {
			r.Path = "relay"
		}
		break
	}
	select {
	case value, ok := <-ping.Ping(libnet.WithAllowLimitedConn(ctx, "teste autenticado"), c.node.Host, info.ID):
		if !ok {
			r.Code = "ping_failed"
			r.Message = "Handshake autenticado, mas o teste de resposta foi interrompido"
			return
		}
		if value.Error != nil {
			r.Code = "ping_failed"
			r.Message = "Handshake autenticado, mas o ping libp2p falhou: " + value.Error.Error()
			return
		}
		if mode(sForProbe(c)) == "integrated" && c.integrated != nil {
			r.Path = c.integrated.ProbePath(ctx, info.ID.String())
		}
		r.Connected = true
		r.RTTMillis = value.RTT.Milliseconds()
		r.RTTMicros = value.RTT.Microseconds()
		r.Code = "connected"
		r.Message = "Peer ID autenticado e ping libp2p recebido de " + r.RemoteAddress
	case <-ctx.Done():
		r.Code = "ping_timeout"
		r.Message = "Handshake autenticado, mas o tempo do ping esgotou"
	}
}
func (c *Core) testLocked(id string) (r ConnectionTest) {
	s, _ := c.db.Settings()
	c.synchronize(s)
	r = c.newTest(id)
	defer func() { c.lastTest = &r }()
	if c.node == nil {
		r.Code = "interface_unavailable"
		r.Message = c.networkErr
		return
	}
	contacts, e := c.db.Contacts()
	if e != nil {
		r.Message = e.Error()
		return
	}
	var target *localdb.Contact
	for _, p := range contacts {
		if p.ID == id {
			v := p
			target = &v
			break
		}
	}
	if target == nil {
		r.Code = "unknown_contact"
		r.Message = "Adicione um contato por convite primeiro"
		return
	}
	r.ContactSaved = true
	if required := requiredMode(target.Scope, s); required != "" {
		r.Code = "mode_mismatch"
		r.RequiredMode = required
		r.Message = "Este contato exige o modo " + adapters.Name(required)
		return
	}
	pid, e := peer.Decode(id)
	if e != nil {
		r.Message = e.Error()
		return
	}
	info := peer.AddrInfo{ID: pid}
	for _, raw := range target.Addresses {
		if c.allowedAddress(raw, s) {
			p, e := network.ParseAddress(raw)
			if e == nil {
				info.Addrs = append(info.Addrs, p.Addrs...)
			}
		}
	}
	timeout := 8 * time.Second
	if c.integrated != nil {
		timeout = 45 * time.Second
	}
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	defer cancel()
	c.probeLocked(ctx, info, &r)
	if r.Connected {
		c.refreshAddresses()
	}
	return
}
func (c *Core) TestConnectivity(id string) (ConnectionTest, error) {
	c.mu.Lock()
	r := c.testLocked(id)
	s, _ := c.db.Settings()
	localPort := port(s)
	if c.node != nil {
		for _, address := range c.node.Host.Network().ListenAddresses() {
			value, e := address.ValueForProtocol(ma.P_TCP)
			if e == nil {
				if n, e := strconv.Atoi(value); e == nil && n > 0 {
					localPort = n
					break
				}
			}
		}
	}
	c.mu.Unlock()
	exe, _ := os.Executable()
	r.Firewall = firewall.Inspect(localPort, exe)
	return r, nil
}
func (c *Core) ConnectInvite(raw string) (ConnectionTest, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) > 24000 {
		return ConnectionTest{}, errors.New("convite ou endereço muito grande")
	}
	if !strings.HasPrefix(raw, "/") {
		i, e := invite.Parse(raw, time.Now())
		if e != nil {
			return ConnectionTest{}, e
		}
		c.mu.Lock()
		settings, _ := c.db.Settings()
		required := requiredMode(i.Scope, settings)
		if required != "" {
			r := c.newTest(i.ID)
			r.RequiredMode = required
			r.Code = "mode_mismatch"
			r.Message = "O convite usa " + adapters.Name(required) + ". Confirme a troca de modo antes de conectar"
			c.lastTest = &r
			c.mu.Unlock()
			return r, nil
		}
		c.mu.Unlock()
		if e = c.AddContact(raw); e != nil {
			return ConnectionTest{}, e
		}
		return c.TestConnectivity(i.ID)
	}
	p, e := network.ParseAddress(raw)
	if e != nil {
		return ConnectionTest{}, errors.New("multiaddr inválido: " + e.Error())
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	s0, _ := c.db.Settings()
	c.synchronize(s0)
	r := c.newTest(p.ID.String())
	defer func() { c.lastTest = &r }()
	a, _ := ma.NewMultiaddr(raw)
	if hint := endpoint.VPNHint(a); hint != "" && requiredMode(hint, s0) != "" {
		r.RequiredMode = hint
		r.Code = "mode_mismatch"
		r.Message = "Este endereço exige " + adapters.Name(hint) + " ou a interface manual correspondente"
		return r, nil
	}
	if c.node == nil {
		r.Code = "interface_unavailable"
		r.Message = c.networkErr
		return r, nil
	}
	s, e := c.db.Settings()
	if e != nil {
		return ConnectionTest{}, e
	}
	if !c.allowedAddress(raw, s) {
		return ConnectionTest{}, errors.New("endereço incompatível com o modo de conectividade")
	}
	if p.ID == c.node.Host.ID() {
		return ConnectionTest{}, errors.New("este endereço é do próprio perfil")
	}
	timeout := 8 * time.Second
	if c.integrated != nil {
		timeout = 45 * time.Second
	}
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	defer cancel()
	c.probeLocked(ctx, p, &r)
	if !r.Authenticated {
		return r, nil
	}
	name := "Contato " + p.ID.String()[:8]
	scope := ""
	if mode(s) != "auto" {
		scope = mode(s)
	}
	contact := localdb.Contact{ID: p.ID.String(), Name: name, Addresses: []string{raw}, Scope: scope}
	if existing, err := c.db.Contacts(); err == nil {
		for _, v := range existing {
			if v.ID == contact.ID {
				contact.Name = v.Name
				contact.PublicKey = v.PublicKey

				break
			}
		}
	}
	if e = c.db.Contact(contact); e != nil {
		return ConnectionTest{}, e
	}
	c.node.Remember(p)
	r.ContactSaved = true
	c.refreshAddresses()
	return r, nil
}

func sForProbe(c *Core) localdb.Settings { s, _ := c.db.Settings(); return s }
