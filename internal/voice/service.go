package voice

import (
	"context"
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"projetop2p/internal/community"
	"projetop2p/internal/invite"
	"sync"
	"sync/atomic"
	"time"
)

type Transport interface {
	VoiceRequest(context.Context, string, []string, Request) (Response, error)
}
type Options struct {
	BindIP     string
	Key        crypto.PrivKey
	State      func(string) (community.State, error)
	Sync       func(context.Context, string) error
	Addresses  func(string) []string
	SelfInvite func() (string, error)
	Route      func(context.Context, Participant) (string, int, error)
	Release    func(string)
}
type call struct {
	ctx                            context.Context
	cancel                         context.CancelFunc
	group, channel, session, owner string
	lease                          Lease
	local, remote                  *MediaPeer
	remoteID, remoteSession, nonce string
	answer                         *webrtc.SessionDescription
	setting                        bool
	muted, deaf                    atomic.Bool
	lastError                      string
	started                        time.Time
}
type Service struct {
	pending   *call
	closing   bool
	handlers  sync.WaitGroup
	mu        sync.Mutex
	engine    *Engine
	authority *Authority
	options   Options
	id        string
	transport Transport
	ctx       context.Context
	cancel    context.CancelFunc
	current   *call
	lastError string
	wg        sync.WaitGroup
}
type ParticipantView struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Self  bool   `json:"self"`
	State string `json:"state"`
}
type View struct {
	Joined       bool              `json:"joined"`
	Community    string            `json:"community"`
	Channel      string            `json:"channel"`
	Session      string            `json:"session"`
	Muted        bool              `json:"muted"`
	Deaf         bool              `json:"deaf"`
	Participants []ParticipantView `json:"participants"`
	Media        MediaStats        `json:"media"`
	Error        string            `json:"error"`
	Expires      int64             `json:"expires"`
	Scope        string            `json:"scope"`
}

func NewService(parent context.Context, o Options) (*Service, error) {
	engine, e := NewEngine(o.BindIP)
	if e != nil {
		return nil, e
	}
	ctx, cancel := context.WithCancel(parent)
	id, _ := peer.IDFromPrivateKey(o.Key)
	s := &Service{engine: engine, authority: NewAuthority(o.Key, o.State), options: o, id: id.String(), ctx: ctx, cancel: cancel}
	s.wg.Add(2)
	go s.guard()
	go s.watch()
	return s, nil
}
func (s *Service) Port() int          { return s.engine.Port() }
func (s *Service) Attach(t Transport) { s.mu.Lock(); s.transport = t; s.mu.Unlock() }
func (s *Service) authorityRequest(ctx context.Context, c *call, method string) (*Lease, error) {
	ticket, e := s.options.SelfInvite()
	if e != nil {
		return nil, e
	}
	r := Request{Method: method, Community: c.group, Channel: c.channel, Participant: Participant{ID: s.id, Session: c.session, Invite: ticket, UDPPort: s.Port()}}
	var result Response
	if c.owner == s.id {
		result = s.authority.Handle(s.id, r)
	} else {
		s.mu.Lock()
		t := s.transport
		s.mu.Unlock()
		if t == nil {
			return nil, errors.New("voice network unavailable")
		}
		result, e = t.VoiceRequest(ctx, c.owner, s.options.Addresses(c.group), r)
	}
	if e != nil {
		return nil, e
	}
	if result.Error != "" {
		return nil, errors.New(result.Error)
	}
	if result.Lease == nil {
		return nil, errors.New("owner omitted voice authorization")
	}
	state, e := s.options.State(c.group)
	if e != nil {
		return nil, e
	}
	if e = VerifyLease(*result.Lease, state, c.channel, time.Now()); e != nil {
		return nil, e
	}
	for _, p := range result.Lease.Participants {
		if p.ID == s.id && p.Session == c.session {
			return result.Lease, nil
		}
	}
	if method == "leave" {
		return result.Lease, nil
	}
	return nil, errors.New("owner did not authorize this call session")
}
func (s *Service) Join(ctx context.Context, group, channel string, offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return webrtc.SessionDescription{}, errors.New("voice service stopped")
	}
	s.handlers.Add(1)
	s.mu.Unlock()
	defer s.handlers.Done()
	ctx, cancelJoin := context.WithCancel(ctx)
	stopJoin := context.AfterFunc(s.ctx, cancelJoin)
	defer stopJoin()
	defer cancelJoin()

	if offer.Type != webrtc.SDPTypeOffer {
		return webrtc.SessionDescription{}, errors.New("local audio offer required")
	}
	if e := CheckAudioSDP(offer); e != nil {
		return webrtc.SessionDescription{}, e
	}
	s.mu.Lock()
	if s.current != nil || s.pending != nil {
		s.mu.Unlock()
		return webrtc.SessionDescription{}, errors.New("leave the active voice room first")
	}
	s.mu.Unlock()
	state, e := s.options.State(group)
	if e != nil || !permitted(state, channel, s.id) {
		return webrtc.SessionDescription{}, errors.New("voice permission unavailable")
	}
	callCtx, cancel := context.WithCancel(s.ctx)
	c := &call{ctx: callCtx, cancel: cancel, group: group, channel: channel, session: community.RandomID(), owner: state.Owner, started: time.Now()}
	s.mu.Lock()
	if s.current != nil || s.pending != nil {
		s.mu.Unlock()
		cancel()
		return webrtc.SessionDescription{}, errors.New("another voice operation is pending")
	}
	s.pending = c
	s.mu.Unlock()
	stopPending := context.AfterFunc(c.ctx, cancelJoin)
	defer stopPending()
	granted := false
	published := false
	defer func() {
		s.mu.Lock()
		if s.pending == c {
			s.pending = nil
		}
		s.mu.Unlock()
		if !published {
			c.cancel()
			if granted {
				cleanupCtx, cleanupCancel := context.WithTimeout(s.ctx, 2*time.Second)
				_, _ = s.authorityRequest(cleanupCtx, c, "leave")
				cleanupCancel()
			}
		}
	}()
	lease, e := s.authorityRequest(ctx, c, "join")
	if e != nil {
		cancel()
		return webrtc.SessionDescription{}, e
	}
	c.lease = *lease
	granted = true
	local, e := s.engine.Peer(func(packet *rtp.Packet) {
		s.mu.Lock()
		active := s.current
		var remote *MediaPeer
		if active == c && !c.muted.Load() && !c.deaf.Load() && time.Now().Unix() < c.lease.Expires {
			remote = c.remote
		}
		s.mu.Unlock()
		if remote != nil {
			_ = remote.Write(packet)
		}
	})
	if e != nil {
		cancel()
		return webrtc.SessionDescription{}, e
	}
	c.local = local
	if e = local.PC.SetRemoteDescription(offer); e != nil {
		local.Close()
		cancel()
		return webrtc.SessionDescription{}, e
	}
	answer, e := local.PC.CreateAnswer(nil)
	if e != nil {
		local.Close()
		cancel()
		return answer, e
	}
	ready := webrtc.GatheringCompletePromise(local.PC)
	if e = local.PC.SetLocalDescription(answer); e != nil {
		local.Close()
		cancel()
		return answer, e
	}
	select {
	case <-ready:
	case <-ctx.Done():
		local.Close()
		cancel()
		return answer, ctx.Err()
	}
	s.mu.Lock()
	if s.current != nil || s.pending != c || c.ctx.Err() != nil || ctx.Err() != nil || s.ctx.Err() != nil {
		s.mu.Unlock()
		local.Close()
		cancel()
		return answer, errors.New("voice join superseded")
	}
	s.current = c
	s.pending = nil
	published = true
	s.lastError = ""
	s.mu.Unlock()
	return *local.PC.LocalDescription(), nil
}
func (s *Service) Controls(muted, deaf bool) {
	s.mu.Lock()
	if c := s.current; c != nil {
		c.muted.Store(muted)
		c.deaf.Store(deaf)
	}
	s.mu.Unlock()
}
func (s *Service) stop(c *call, message string) {
	s.mu.Lock()
	if s.current != c {
		s.mu.Unlock()
		return
	}
	s.current = nil
	s.lastError = message
	c.cancel()
	local, remote, id := c.local, c.remote, c.remoteID
	s.mu.Unlock()
	if local != nil {
		local.Close()
	}
	if remote != nil {
		remote.Close()
	}
	if id != "" && s.options.Release != nil {
		s.options.Release(id)
	}
}
func (s *Service) Leave() {
	s.mu.Lock()
	if s.pending != nil {
		s.pending.cancel()
	}
	c := s.current
	s.mu.Unlock()
	if c == nil {
		return
	}
	s.stop(c, "")
	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Second)
	defer cancel()
	_, _ = s.authorityRequest(ctx, c, "leave")
}
func (s *Service) guard() {
	defer s.wg.Done()
	timer := time.NewTicker(500 * time.Millisecond)
	defer timer.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-timer.C:
		}
		s.mu.Lock()
		c := s.current
		var lease Lease
		if c != nil {
			lease = c.lease
		}
		s.mu.Unlock()
		if c == nil {
			continue
		}
		state, e := s.options.State(c.group)
		if e != nil || !permitted(state, c.channel, s.id) || lease.Expires <= time.Now().Unix() {
			s.stop(c, "Autorizacao de voz expirada ou revogada; microfone desconectado")
			continue
		}
		if VerifyLease(lease, state, c.channel, time.Now()) != nil {
			s.stop(c, "Permissoes da comunidade mudaram; entre novamente para obter autorizacao atual")
		}
	}
}
func (s *Service) watch() {
	defer s.wg.Done()
	timer := time.NewTicker(3 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-timer.C:
		}
		s.mu.Lock()
		c := s.current
		s.mu.Unlock()
		if c == nil {
			continue
		}
		ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
		if s.options.Sync != nil {
			_ = s.options.Sync(ctx, c.group)
		}
		lease, e := s.authorityRequest(ctx, c, "renew")
		cancel()
		if e != nil {
			s.mu.Lock()
			if s.current == c {
				c.lastError = e.Error()
			}
			s.mu.Unlock()
			continue
		}
		s.mu.Lock()
		if s.current != c {
			s.mu.Unlock()
			continue
		}
		c.lease = *lease
		c.lastError = ""
		var other Participant
		for _, p := range lease.Participants {
			if p.ID != s.id {
				other = p
			}
		}
		oldRemote := ""
		var closePeer *MediaPeer
		if c.remote != nil && (other.ID != c.remoteID || other.Session != c.remoteSession) {
			closePeer = c.remote
			oldRemote = c.remoteID
			c.remote = nil
			c.remoteID = ""
			c.answer = nil
		}
		shouldStart := other.ID != "" && s.id < other.ID && !c.setting && (c.remote == nil || c.remote.Stats().State == "failed" || c.remote.Stats().State == "closed")
		if shouldStart {
			if c.remote != nil {
				closePeer = c.remote
				oldRemote = c.remoteID
				c.remote = nil
			}
			c.setting = true
			s.wg.Add(1)
		}
		s.mu.Unlock()
		if closePeer != nil {
			closePeer.Close()
		}
		if oldRemote != "" && s.options.Release != nil {
			s.options.Release(oldRemote)
		}
		if shouldStart {
			go func() { defer s.wg.Done(); s.connect(c, other) }()
		}
	}
}
func (s *Service) newRemote(c *call) (*MediaPeer, error) {
	return s.engine.Peer(func(packet *rtp.Packet) {
		s.mu.Lock()
		active := s.current
		var local *MediaPeer
		if active == c && !c.deaf.Load() && time.Now().Unix() < c.lease.Expires {
			local = c.local
		}
		s.mu.Unlock()
		if local != nil {
			_ = local.Write(packet)
		}
	})
}
func (s *Service) connect(c *call, other Participant) {
	defer func() {
		s.mu.Lock()
		if s.current == c {
			c.setting = false
		}
		s.mu.Unlock()
	}()
	ctx, cancel := context.WithTimeout(c.ctx, 35*time.Second)
	defer cancel()
	ip, port, e := s.options.Route(ctx, other)
	if e != nil {
		s.setError(c, e)
		return
	}
	pc, e := s.newRemote(c)
	if e != nil {
		if s.options.Release != nil {
			s.options.Release(other.ID)
		}
		s.setError(c, e)
		return
	}
	keep := false
	defer func() {
		if !keep {
			pc.Close()
			if s.options.Release != nil {
				s.options.Release(other.ID)
			}
		}
	}()
	offer, e := pc.PC.CreateOffer(nil)
	if e != nil {
		s.setError(c, e)
		return
	}
	ready := webrtc.GatheringCompletePromise(pc.PC)
	if e = pc.PC.SetLocalDescription(offer); e != nil {
		s.setError(c, e)
		return
	}
	select {
	case <-ready:
	case <-ctx.Done():
		return
	}
	s.mu.Lock()
	if s.current != c {
		s.mu.Unlock()
		return
	}
	lease := c.lease
	t := s.transport
	s.mu.Unlock()
	if t == nil {
		return
	}
	ticket, e := s.options.SelfInvite()
	if e != nil {
		s.setError(c, e)
		return
	}
	request := Request{Method: "signal", Community: c.group, Channel: c.channel, Participant: Participant{ID: s.id, Session: c.session, Invite: ticket, UDPPort: s.Port()}, Lease: &lease, Session: other.Session, Nonce: community.RandomID(), SDP: pc.PC.LocalDescription()}
	result, e := t.VoiceRequest(ctx, other.ID, addresses(other.Invite), request)
	if e != nil {
		s.setError(c, e)
		return
	}
	if result.Error != "" {
		s.setError(c, errors.New(result.Error))
		return
	}
	if result.SDP == nil || result.SDP.Type != webrtc.SDPTypeAnswer {
		s.setError(c, errors.New("authenticated audio answer missing"))
		return
	}
	remote, e := ForwardSDP(*result.SDP, ip, port)
	if e != nil {
		s.setError(c, e)
		return
	}
	if e = pc.PC.SetRemoteDescription(remote); e != nil {
		s.setError(c, e)
		return
	}
	s.mu.Lock()
	if s.current == c {
		c.remote = pc
		c.remoteID = other.ID
		c.remoteSession = other.Session
		c.nonce = request.Nonce
		c.lastError = ""
		keep = true
	}
	s.mu.Unlock()
}
func (s *Service) setError(c *call, e error) {
	s.mu.Lock()
	if s.current == c {
		c.lastError = e.Error()
	}
	s.mu.Unlock()
}
func (s *Service) Handle(ctx context.Context, remote string, r Request) Response {
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return Response{Error: "voice stopped"}
	}
	s.handlers.Add(1)
	s.mu.Unlock()
	defer s.handlers.Done()
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(s.ctx, cancel)
	defer stop()
	defer cancel()

	if r.Method != "signal" {
		return s.authority.Handle(remote, r)
	}
	fail := func(e string) Response { return Response{Error: e} }
	s.mu.Lock()
	c := s.current
	if c == nil || c.group != r.Community || c.channel != r.Channel || c.session != r.Session {
		s.mu.Unlock()
		return fail("voice_not_joined: user did not join this room")
	}
	s.mu.Unlock()
	if r.SDP == nil || r.SDP.Type != webrtc.SDPTypeOffer || len(r.Nonce) != 32 || r.Participant.ID != remote || r.Lease == nil {
		return fail("invalid audio signaling")
	}
	state, e := s.options.State(c.group)
	if e != nil {
		return fail("community unavailable")
	}
	if e = VerifyLease(*r.Lease, state, c.channel, time.Now()); e != nil {
		return fail(e.Error())
	}
	if remote >= s.id {
		return fail("designated signal initiator mismatch")
	}
	if e = CheckAudioSDP(*r.SDP); e != nil {
		return fail(e.Error())
	}
	authorized := false
	destination := false
	var other Participant
	for _, p := range r.Lease.Participants {
		if p.ID == remote && p.Session == r.Participant.Session {
			authorized = true
			other = p
		}
		if p.ID == s.id && p.Session == c.session {
			destination = true
		}
	}
	if !authorized || !destination {
		return fail("unauthorized signal participants")
	}
	s.mu.Lock()
	if s.current != c {
		s.mu.Unlock()
		return fail("call ended")
	}
	localAuthorized := false
	for _, p := range c.lease.Participants {
		if p.ID == remote && p.Session == r.Participant.Session {
			localAuthorized = true
		}
	}
	if !localAuthorized || c.lease.Generation != r.Lease.Generation {
		s.mu.Unlock()
		return fail("room membership not known locally; refresh authorization")
	}
	if c.setting {
		s.mu.Unlock()
		return fail("signaling busy")
	}
	if c.remote != nil {
		if c.remoteID == remote && c.nonce == r.Nonce && c.answer != nil {
			answer := *c.answer
			s.mu.Unlock()
			return Response{SDP: &answer}
		}
		if c.remote.Stats().State != "failed" && c.remote.Stats().State != "closed" {
			s.mu.Unlock()
			return fail("active media session; replay or glare rejected")
		}
	}
	c.setting = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		if s.current == c {
			c.setting = false
		}
		s.mu.Unlock()
	}()
	ip, port, e := s.options.Route(ctx, other)
	if e != nil {
		return fail(e.Error())
	}
	attached := false
	defer func() {
		if !attached && s.options.Release != nil {
			s.options.Release(other.ID)
		}
	}()
	offer, e := ForwardSDP(*r.SDP, ip, port)
	if e != nil {
		return fail(e.Error())
	}
	pc, e := s.newRemote(c)
	if e != nil {
		return fail(e.Error())
	}
	if e = pc.PC.SetRemoteDescription(offer); e != nil {
		pc.Close()
		return fail(e.Error())
	}
	answer, e := pc.PC.CreateAnswer(nil)
	if e != nil {
		pc.Close()
		return fail(e.Error())
	}
	ready := webrtc.GatheringCompletePromise(pc.PC)
	if e = pc.PC.SetLocalDescription(answer); e != nil {
		pc.Close()
		return fail(e.Error())
	}
	select {
	case <-ready:
	case <-ctx.Done():
		pc.Close()
		return fail(ctx.Err().Error())
	}
	s.mu.Lock()
	if s.current != c {
		s.mu.Unlock()
		pc.Close()
		return fail("call ended")
	}
	old := c.remote
	c.remote = pc
	attached = true
	c.remoteID = remote
	c.remoteSession = other.Session
	c.nonce = r.Nonce
	answer = *pc.PC.LocalDescription()
	c.answer = &answer
	c.lastError = ""
	s.mu.Unlock()
	if old != nil {
		old.Close()
	}
	return Response{SDP: &answer}
}
func (s *Service) View() View {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := View{Participants: []ParticipantView{}, Error: s.lastError, Scope: "Opus/DTLS-SRTP entre processos; ICE UDP via adaptador. Transporte externo EasyTier pode usar relay TCP; metricas nao presumem caminho direto."}
	c := s.current
	if c == nil {
		return v
	}
	v.Joined = true
	v.Community = c.group
	v.Channel = c.channel
	v.Session = c.session
	v.Muted = c.muted.Load()
	v.Deaf = c.deaf.Load()
	v.Error = c.lastError
	v.Expires = c.lease.Expires
	if c.remote != nil {
		v.Media = c.remote.Stats()
	}
	for _, p := range c.lease.Participants {
		state := "aguardando"
		if p.ID == s.id {
			state = "local"
		} else if c.remote != nil && c.remoteID == p.ID {
			state = c.remote.Stats().State
		}
		v.Participants = append(v.Participants, ParticipantView{ID: p.ID, Name: p.Name, Self: p.ID == s.id, State: state})
	}
	return v
}
func (s *Service) Close() {
	s.mu.Lock()
	s.closing = true
	if s.pending != nil {
		s.pending.cancel()
	}
	s.mu.Unlock()
	s.cancel()
	s.mu.Lock()
	c := s.current
	s.mu.Unlock()
	if c != nil {
		s.stop(c, "Conectividade encerrada")
	}
	s.wg.Wait()
	s.handlers.Wait()
	s.engine.Close()
}
func addresses(raw string) []string {
	ticket, e := parseInvitation(raw)
	if e != nil {
		return nil
	}
	return ticket
}
func parseInvitation(raw string) ([]string, error) {
	i, e := invite.Parse(raw, time.Now())
	return i.Addresses, e
}
