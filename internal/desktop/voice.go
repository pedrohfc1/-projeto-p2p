package desktop

import (
	"context"
	"errors"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pion/webrtc/v4"
	"net"
	"projetop2p/internal/community"
	"projetop2p/internal/endpoint"
	"projetop2p/internal/invite"
	"projetop2p/internal/localdb"
	"projetop2p/internal/network"
	"projetop2p/internal/voice"
	"time"
)

func (c *Core) startVoice(n *network.Node, settings localdb.Settings) (*voice.Service, error) {
	manager := c.integrated
	bindIP := "0.0.0.0"
	if mode(settings) == "integrated" {
		bindIP = "127.0.0.1"
	} else if mode(settings) != "auto" {
		bindIP = c.selection.IP
	}
	v, e := voice.NewService(c.ctx, voice.Options{BindIP: bindIP, Key: c.key, State: func(id string) (community.State, error) {
		record, e := c.db.Community(id)
		if e != nil {
			return community.State{}, e
		}
		if record.Denied {
			return community.State{}, errors.New("membership was revoked")
		}
		return record.State, nil
	}, Sync: c.groups.Sync, Addresses: func(id string) []string {
		record, e := c.db.Community(id)
		if e != nil {
			return nil
		}
		return record.Addresses
	}, SelfInvite: func() (string, error) {
		if manager != nil {
			return invite.NewIntegrated(c.key, settings.Name, n.Addresses(), manager.Token())
		}
		if mode(settings) != "auto" {
			return invite.NewScoped(c.key, settings.Name, n.Addresses(), mode(settings))
		}
		return invite.New(c.key, settings.Name, n.Addresses(), settings.Relays)
	}, Route: func(ctx context.Context, p voice.Participant) (string, int, error) {
		i, e := invite.Parse(p.Invite, time.Now())
		if e != nil || i.ID != p.ID {
			return "", 0, errors.New("authenticated voice peer route invalid")
		}
		if manager != nil {
			if i.Scope != "integrated" {
				return "", 0, errors.New("voice peer uses another network")
			}
			if e = manager.Register(p.ID, i.Route); e != nil {
				return "", 0, e
			}
			port, e := manager.MediaForward(ctx, p.ID, p.UDPPort)
			return "127.0.0.1", port, e
		}
		scope := i.Scope
		if scope == "" {
			scope = "auto"
		}
		if scope != mode(settings) {
			return "", 0, errors.New("voice network mismatch; select same connectivity mode")
		}
		for _, raw := range i.Addresses {
			a, e := ma.NewMultiaddr(raw)
			if e != nil || !endpoint.Allowed(a, scope) {
				continue
			}
			ip, e := a.ValueForProtocol(ma.P_IP4)
			if e == nil && net.ParseIP(ip) != nil {
				return ip, p.UDPPort, nil
			}
		}
		return "", 0, errors.New("no authorized IPv4 UDP media endpoint")
	}, Release: func(id string) {
		if manager != nil {
			manager.CloseMedia(id)
		}
	}})
	if e != nil {
		return nil, e
	}
	v.Attach(n)
	return v, nil
}
func (c *Core) voiceService() (*voice.Service, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.voice == nil {
		return nil, errors.New("voice connectivity unavailable")
	}
	return c.voice, nil
}
func (c *Core) VoiceJoin(group, channel string, offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	v, e := c.voiceService()
	if e != nil {
		return webrtc.SessionDescription{}, e
	}
	ctx, cancel := context.WithTimeout(c.ctx, 45*time.Second)
	defer cancel()
	return v.Join(ctx, group, channel, offer)
}
func (c *Core) VoiceState() voice.View {
	v, e := c.voiceService()
	if e != nil {
		return voice.View{Participants: []voice.ParticipantView{}, Error: e.Error()}
	}
	return v.View()
}
func (c *Core) VoiceLeave() {
	if v, e := c.voiceService(); e == nil {
		v.Leave()
	}
}
func (c *Core) VoiceControls(muted, deaf bool) {
	if v, e := c.voiceService(); e == nil {
		v.Controls(muted, deaf)
	}
}
