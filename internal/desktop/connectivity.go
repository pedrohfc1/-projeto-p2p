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
	ip, _ := a.ValueForProtocol(ma.P_IP4)
	if mode(s) == "auto" {
		return !adapters.VirtualIPv4(ip)
	}
	if _, e = a.ValueForProtocol(ma.P_CIRCUIT); e == nil {
		return false
	}
	if ip == "" {
		return false
	}
	if _, e = a.ValueForProtocol(ma.P_TCP); e != nil {
		return false
	}
	if v, known := adapters.VPN(mode(s)); known {
		return strings.HasPrefix(ip, v.IPv4Prefix)
	}
	return true
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

type ConnectionTest struct {
	ID        string          `json:"id"`
	Connected bool            `json:"connected"`
	Path      string          `json:"path"`
	RTTMillis int64           `json:"rttMillis"`
	Message   string          `json:"message"`
	Firewall  firewall.Report `json:"firewall"`
}

func (c *Core) testLocked(id string) ConnectionTest {
	r := ConnectionTest{ID: id, Path: "offline"}
	if c.node == nil {
		r.Message = c.networkErr
		return r
	}
	contacts, e := c.db.Contacts()
	if e != nil {
		r.Message = e.Error()
		return r
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
		r.Message = "Adicione um contato por convite primeiro"
		return r
	}
	pid, e := peer.Decode(id)
	if e != nil {
		r.Message = e.Error()
		return r
	}
	ctx, cancel := context.WithTimeout(c.ctx, 8*time.Second)
	defer cancel()
	s, _ := c.db.Settings()
	info := peer.AddrInfo{ID: pid}
	for _, a := range target.Addresses {
		if !c.allowedAddress(a, s) {
			continue
		}
		p, e := network.ParseAddress(a)
		if e == nil {
			info.Addrs = append(info.Addrs, p.Addrs...)
		}
	}
	if len(info.Addrs) == 0 {
		r.Message = "O convite não tem endereço compatível com o modo selecionado"
		return r
	}
	if e = c.node.Connect(ctx, info); e != nil {
		r.Message = "Peer não respondeu. Confirme a mesma rede da VPN selecionada, IP atual, aplicativo aberto e permissão TCP no firewall. Timeout não identifica sozinho a causa. Detalhe: " + e.Error()
		return r
	}
	var result ping.Result
	select {
	case value, ok := <-ping.Ping(libnet.WithAllowLimitedConn(ctx, "teste autenticado"), c.node.Host, pid):
		if !ok {
			r.Message = "Teste interrompido ou sem resposta"
			return r
		}
		result = value
	case <-ctx.Done():
		r.Message = "Tempo do teste esgotado"
		return r
	}
	if result.Error != nil {
		r.Message = "Conexão autenticada, mas o teste de resposta falhou: " + result.Error.Error()
		return r
	}
	r.Connected = true
	r.Path = "direta"
	r.RTTMillis = result.RTT.Milliseconds()
	for _, p := range c.node.Status().Peers {
		if p.ID == id {
			r.Path = p.Path
		}
	}
	r.Message = "Peer ID autenticado e resposta libp2p recebida"
	c.refreshAddresses()
	return r
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
	if c.node == nil {
		return ConnectionTest{}, errors.New(c.networkErr)
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
	ctx, cancel := context.WithTimeout(c.ctx, 8*time.Second)
	defer cancel()
	if e = c.node.Connect(ctx, p); e != nil {
		return ConnectionTest{}, fmt.Errorf("peer não respondeu; confira a VPN selecionada, endereço e firewall: %w", e)
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
				if v.Scope != "" {
					contact.Scope = v.Scope
				}
				break
			}
		}
	}
	if e = c.db.Contact(contact); e != nil {
		return ConnectionTest{}, e
	}
	c.node.Remember(p)
	return c.testLocked(p.ID.String()), nil
}
