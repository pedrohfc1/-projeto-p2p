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
	v, e := voice.NewService(c.ctx, voice.Options{Capture: c.captureFactory, CaptureDir: c.dir, RegisterStreamRoute: func(p voice.Participant) error {
		i, e := invite.Parse(p.Invite, time.Now())
		if e != nil || i.ID != p.ID {
			return errors.New("invalid authenticated media route")
		}
		if manager != nil {
			if i.Scope != "integrated" {
				return errors.New("voice route network mismatch")
			}
			return manager.Register(p.ID, i.Route)
		}
		return nil
	}, BindIP: bindIP, Key: c.key, State: func(id string) (community.State, error) {
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
	// UI/network enumeration may block Core.mu. Media liveness must not wait for it.
	if c.ctx != nil && c.ctx.Err() != nil {
		return nil, errors.New("voice connectivity closed")
	}
	if media := c.media.Load(); media != nil {
		return media, nil
	}

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

	view := v.View()
	if view.Joined {
		if record, e := c.db.Community(view.Community); e == nil {
			view.CommunityName = record.State.Name
			view.ChannelName = record.State.Channels[view.Channel].Name
		}
	}
	c.activity.mu.Lock()
	for i := range view.Participants {
		p := &view.Participants[i]
		if p.Self {
			muted, deaf := view.Muted || view.Deaf, view.Deaf
			p.Muted = &muted
			p.Deaf = &deaf
		} else if state, ok := c.activity.presence[p.ID]; ok && time.Now().Before(state.Until) && state.Group == view.Community && state.Channel == view.Channel && v.HasSession(view.Community, view.Channel, p.ID, state.Session) {
			muted, deaf := state.Muted, state.Deaf
			p.Muted = &muted
			p.Deaf = &deaf
		}
	}
	c.activity.mu.Unlock()
	return view
}
func (c *Core) VoiceLeave() {
	c.CancelRoomProbe()
	if v, e := c.voiceService(); e == nil {
		v.Leave()
	}
}
func (c *Core) VoiceControls(muted, deaf bool) {
	if v, e := c.voiceService(); e == nil {
		v.Controls(muted, deaf)
	}
}

func (c *Core) VoiceRebind(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	v, e := c.voiceService()
	if e != nil {
		return webrtc.SessionDescription{}, e
	}
	return v.Rebind(c.ctx, offer)
}

func (c *Core) AudioPreferences() (localdb.Preferences, error)   { return c.db.Preferences() }
func (c *Core) SaveAudioPreferences(p localdb.Preferences) error { return c.db.SavePreferences(p) }

func (c *Core) ScreenConnect(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	v, e := c.voiceService()
	if e != nil {
		return webrtc.SessionDescription{}, e
	}
	ctx, cancel := context.WithTimeout(c.ctx, 8*time.Second)
	defer cancel()
	return v.ScreenConnect(ctx, offer)
}
func (c *Core) ScreenPublish(publish bool, bitrate, maxViewers int) error {
	v, e := c.voiceService()
	if e != nil {
		return e
	}
	return v.ScreenPublish(publish, bitrate, maxViewers)
}
func (c *Core) ScreenWatch(id string) error {
	v, e := c.voiceService()
	if e != nil {
		return e
	}
	return v.ScreenWatchPeer(id)
}
func (c *Core) ScreenStop() {
	if v, e := c.voiceService(); e == nil {
		v.ScreenStop()
	}
}
func (c *Core) ScreenHeartbeat() {
	if v, e := c.voiceService(); e == nil {
		v.ScreenHeartbeat()
	}
}
func (c *Core) ScreenState() voice.ScreenView {
	if v, e := c.voiceService(); e == nil {
		return v.ScreenState()
	}
	return voice.ScreenView{}
}

func (c *Core) VoiceControlsForSession(session string, muted, deaf bool) error {
	v, e := c.voiceService()
	if e != nil {
		return e
	}
	return v.ControlsForSession(session, muted, deaf)
}
