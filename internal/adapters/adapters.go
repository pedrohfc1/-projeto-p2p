// Package adapters enumerates interfaces without changing OS or VPN settings.
package adapters

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

type Info struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Index       int      `json:"index"`
	Up          bool     `json:"up"`
	IPv4        []string `json:"ipv4"`
	Provider    string   `json:"provider"`
	Evidence    string   `json:"evidence"`
	DriverID    string   `json:"-"`
}
type Provider func() ([]Info, error)
type Selection struct {
	Mode              string          `json:"mode"`
	Detected          bool            `json:"detected"`
	DetectedProviders map[string]bool `json:"detectedProviders"`
	Ready             bool            `json:"ready"`
	Selected          *Info           `json:"selected"`
	IP                string          `json:"ip"`
	Message           string          `json:"message"`
	Adapters          []Info          `json:"adapters"`
}

func Classify(i Info) Info {
	i.Provider = "unknown"
	i.Evidence = "seleção manual disponível"
	for _, v := range vpnDefinitions {
		if metadataMatches(i.DriverID, v) {
			i.Provider = v.ID
			i.Evidence = "identificador/descrição do driver no registro"
			return i
		}
	}
	// FriendlyName alone (and 25/8 alone) is insufficient. Description comes from
	// the adapter driver, and the virtual address provides a second signal.
	for _, v := range vpnDefinitions {
		for _, ip := range i.IPv4 {
			if metadataMatches(i.Description, v) && strings.HasPrefix(ip, v.IPv4Prefix) {
				i.Provider = v.ID
				i.Evidence = "descrição do driver e faixa virtual compatível"
				return i
			}
		}
	}
	return i
}
func Select(items []Info, mode, id string) (Selection, error) {
	if mode == "" {
		mode = "auto"
	}
	s := Selection{Mode: mode, Adapters: items, DetectedProviders: map[string]bool{}}
	for _, v := range vpnDefinitions {
		s.DetectedProviders[v.ID] = false
	}
	for _, i := range items {
		if IsVPN(i.Provider) {
			s.DetectedProviders[i.Provider] = true
		}
	}
	if IsVPN(mode) {
		s.Detected = s.DetectedProviders[mode]
	} else {
		for _, found := range s.DetectedProviders {
			if found {
				s.Detected = true
			}
		}
	}
	if mode == "auto" {
		s.Ready = true
		s.Message = "Conectividade libp2p nativa"
		return s, nil
	}
	if !IsVPN(mode) && mode != "interface" {
		return s, errors.New("modo de conectividade inválido")
	}
	candidates := []Info{}
	for _, i := range items {
		if (id == "" || i.ID == id) && (!IsVPN(mode) || i.Provider == mode) {
			candidates = append(candidates, i)
		}
	}
	if mode == "interface" && id == "" {
		s.Message = "Selecione uma interface de rede manualmente"
		return s, errors.New(s.Message)
	}
	if len(candidates) == 0 {
		s.Message = "Adaptador selecionado ausente"
		if IsVPN(mode) {
			s.Message = Name(mode) + " não detectado. Verifique essa VPN ou use seleção manual; a outra VPN não é necessária"
		}
		return s, errors.New(s.Message)
	}
	if id == "" && len(candidates) > 1 {
		active := []Info{}
		for _, i := range candidates {
			if i.Up && len(i.IPv4) > 0 {
				active = append(active, i)
			}
		}
		if len(active) == 1 {
			candidates = active
		} else {
			s.Message = "Mais de um adaptador candidato; selecione explicitamente a interface"
			return s, errors.New(s.Message)
		}
	}
	i := candidates[0]
	s.Selected = &i
	if !i.Up {
		s.Message = "Adaptador desconectado/inativo. Conecte " + Name(mode) + " ou a interface selecionada"
		return s, errors.New(s.Message)
	}
	for _, value := range i.IPv4 {
		ip := net.ParseIP(value)
		if ip != nil && ip.To4() != nil && !ip.IsUnspecified() && !ip.IsMulticast() && !ip.IsLinkLocalUnicast() {
			if v, known := VPN(mode); known && !strings.HasPrefix(value, v.IPv4Prefix) {
				continue
			}
			s.IP = value
			break
		}
	}
	if s.IP == "" {
		s.Message = "Adaptador sem IPv4 virtual válido"
		return s, errors.New(s.Message)
	}
	s.Ready = true
	s.Message = fmt.Sprintf("Interface ativa em %s; isso ainda não comprova conexão com um peer P2P", s.IP)
	return s, nil
}
func basic() ([]Info, error) {
	interfaces, e := net.Interfaces()
	if e != nil {
		return nil, e
	}
	out := []Info{}
	for _, i := range interfaces {
		v := Info{ID: fmt.Sprint(i.Index), Index: i.Index, Name: i.Name, Up: i.Flags&net.FlagUp != 0, IPv4: []string{}}
		addresses, e := i.Addrs()
		if e != nil {
			continue
		}
		for _, a := range addresses {
			ip, _, e := net.ParseCIDR(a.String())
			if e == nil && ip.To4() != nil {
				v.IPv4 = append(v.IPv4, ip.String())
			}
		}
		out = append(out, v)
	}
	return out, nil
}
