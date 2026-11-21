package voice

import (
	"context"
	"errors"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"log/slog"
	"projetop2p/internal/community"
	"projetop2p/internal/loopback"
	"time"
)

// PCM capture is an injected platform boundary; production always defaults to WASAPI.
type CaptureSource interface {
	Read() (loopback.Frame, error)
	Close()
	Alive() bool
}
type CaptureFactory func(context.Context, string, string, int) (CaptureSource, error)

// Independent protocol/bridge: failure never closes video or microphone media.
type ScreenSoundTransport interface {
	OpenScreenAudioMedia(context.Context, string, []string, Request) (MediaConn, error)
}
type soundPeer struct {
	participant Participant
	stream      *StreamMedia
	connecting  bool
	initiator   string
	next        time.Time
	failures    int
	first       time.Time
}
type screenSound struct {
	actor           string
	captureRevision uint64
	captureCancel   context.CancelFunc
	parent          *screenCall
	ctx             context.Context
	cancel          context.CancelFunc
	local           *MediaPeer
	capture         CaptureSource
	peers           map[string]*soundPeer
	nonces          call
	encoded         uint64
	source          string
}
type ScreenSoundView struct {
	Connected bool                  `json:"connected"`
	Capturing bool                  `json:"capturing"`
	Source    string                `json:"source"`
	Encoded   uint64                `json:"encoded"`
	Peers     map[string]MediaStats `json:"peers"`
	Error     string                `json:"error"`
}

func (s *Service) soundValid(a *screenSound) bool {
	return a != nil && s.sound == a && s.screenLive(a.parent) && a.ctx.Err() == nil
}
func (s *Service) ScreenAudioSources() ([]loopback.Source, error) {
	return loopback.Sources(s.ctx, s.options.CaptureDir)
}
func (s *Service) ScreenAudioConnect(ctx context.Context, expected, actor string, offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	if len(actor) != 32 {
		return webrtc.SessionDescription{}, errors.New("invalid local audio actor")
	}
	if e := checkAudioSDP(offer, 1); e != nil {
		return webrtc.SessionDescription{}, e
	}
	s.soundMu.Lock()
	defer s.soundMu.Unlock()
	s.mu.Lock()
	sc := s.screen
	if sc == nil || sc.parent.session != expected || !s.screenAuthorized(sc.parent) {
		s.mu.Unlock()
		return webrtc.SessionDescription{}, errScreenUnavailable
	}
	if s.sound != nil {
		s.mu.Unlock()
		return webrtc.SessionDescription{}, errors.New("ponte de áudio da tela já existe")
	}
	s.handlers.Add(1)
	defer s.handlers.Done()
	child, cancel := context.WithCancel(sc.ctx)
	a := &screenSound{actor: actor, parent: sc, ctx: child, cancel: cancel, peers: map[string]*soundPeer{}}
	s.sound = a
	s.mu.Unlock()
	peer, e := s.localEngine.Peer(func(p *rtp.Packet) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if !s.soundValid(a) || a.capture == nil || !a.capture.Alive() || !sc.publishing {
			return
		}
		a.encoded++
		for id, pv := range a.peers {
			if pv.stream != nil && sc.peers[id] != nil && sc.peers[id].pipe != nil && sc.peers[id].pipe.state.Load() && screenAccepts(sc, id) {
				pv.stream.Push(p)
			}
		}
	})
	fail := func(e error) (webrtc.SessionDescription, error) {
		cancel()
		if peer != nil {
			peer.Close()
		}
		s.mu.Lock()
		if s.sound == a {
			s.sound = nil
		}
		s.mu.Unlock()
		return webrtc.SessionDescription{}, e
	}
	if e != nil {
		return fail(e)
	}
	if e = peer.PC.SetRemoteDescription(offer); e != nil {
		return fail(e)
	}
	answer, e := peer.PC.CreateAnswer(nil)
	if e != nil {
		return fail(e)
	}
	ready := webrtc.GatheringCompletePromise(peer.PC)
	if e = peer.PC.SetLocalDescription(answer); e != nil {
		return fail(e)
	}
	select {
	case <-ready:
	case <-ctx.Done():
		return fail(ctx.Err())
	case <-child.Done():
		return fail(child.Err())
	}
	s.mu.Lock()
	if !s.soundValid(a) {
		s.mu.Unlock()
		return fail(errScreenUnavailable)
	}
	a.local = peer
	s.wg.Add(1)
	s.mu.Unlock()
	go s.soundWatch(a)
	return *peer.PC.LocalDescription(), nil
}
func (s *Service) ScreenAudioCapture(expected, actor string, revision uint64, mode string, pid int, consent bool) error {
	if !consent {
		return errors.New("consentimento explícito necessário para capturar áudio do computador")
	}
	s.mu.Lock()
	intent := s.sound
	if !s.soundValid(intent) || intent.parent.parent.session != expected || intent.actor != actor || revision <= intent.captureRevision {
		s.mu.Unlock()
		return errScreenUnavailable
	}
	intent.captureRevision = revision
	s.mu.Unlock()
	s.soundMu.Lock()
	defer s.soundMu.Unlock()
	s.mu.Lock()
	a := s.sound
	if a != intent || !s.soundValid(a) || a.actor != actor || a.captureRevision != revision || a.parent.parent.session != expected || !a.parent.publishing {
		s.mu.Unlock()
		return errScreenUnavailable
	}
	old := a.capture
	oldCancel := a.captureCancel
	captureCtx, captureCancel := context.WithCancel(a.ctx)
	a.captureCancel = captureCancel
	a.capture = nil
	s.mu.Unlock()
	if oldCancel != nil {
		oldCancel()
	}
	if old != nil {
		old.Close()
	}
	factory := s.options.Capture
	if factory == nil {
		factory = func(ctx context.Context, dir, mode string, pid int) (CaptureSource, error) {
			return loopback.Start(ctx, dir, mode, pid)
		}
	}
	capture, e := factory(captureCtx, s.options.CaptureDir, mode, pid)
	if e != nil {
		captureCancel()
		return e
	}
	s.mu.Lock()
	if s.sound != a || !s.soundValid(a) || a.captureRevision != revision || a.actor != actor || a.parent.parent.session != expected || !a.parent.publishing {
		s.mu.Unlock()
		captureCancel()
		capture.Close()
		return errScreenUnavailable
	}
	a.capture = capture
	a.source = mode
	s.mu.Unlock()
	slog.Info("screen_audio_capture", "state", "started", "source", mode, "codec", "opus/20ms")
	return nil
}
func (s *Service) ScreenAudioStopCapture(expected string) {
	s.mu.Lock()
	a := s.sound
	var capture CaptureSource
	var cancel context.CancelFunc
	if a != nil && a.parent.parent.session == expected && (a.capture != nil || a.captureCancel != nil) {
		a.captureRevision++
		capture = a.capture
		a.capture = nil
		cancel = a.captureCancel
		a.captureCancel = nil
	}
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if capture != nil {
		capture.Close()
		slog.Info("screen_audio_capture", "state", "stopped")
	}
}
func (s *Service) ScreenAudioRead(expected, actor string) (loopback.Frame, error) {
	s.mu.Lock()
	a := s.sound
	var capture CaptureSource
	if s.soundValid(a) && a.actor == actor && a.parent.parent.session == expected && a.parent.publishing {
		capture = a.capture
	}
	s.mu.Unlock()
	if capture == nil {
		return loopback.Frame{}, errors.New("captura de áudio da tela inativa")
	}
	return capture.Read()
}
func (s *Service) ScreenAudioState() ScreenSoundView {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := ScreenSoundView{Peers: map[string]MediaStats{}}
	a := s.sound
	if a == nil {
		return v
	}
	v.Connected = a.local != nil && a.local.Stats().State == "connected"
	v.Capturing = a.capture != nil && a.capture.Alive() && s.soundValid(a) && a.parent.publishing
	v.Source = a.source
	v.Encoded = a.encoded
	for id, p := range a.peers {
		if p.stream != nil {
			v.Peers[id] = p.stream.Stats()
		}
	}
	return v
}
func (s *Service) soundWatch(a *screenSound) {
	defer s.wg.Done()
	defer func() {
		s.soundMu.Lock()
		defer s.soundMu.Unlock()
		s.mu.Lock()
		if s.sound == a {
			s.sound = nil
		}
		capture := a.capture
		a.capture = nil
		local := a.local
		var streams []*StreamMedia
		for _, p := range a.peers {
			if p.stream != nil {
				streams = append(streams, p.stream)
			}
		}
		s.mu.Unlock()
		a.cancel()
		if capture != nil {
			capture.Close()
		}
		for _, p := range streams {
			p.Close()
		}
		if local != nil {
			local.Close()
		}
	}()
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	lastReport := time.Now()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-tick.C:
		}
		s.mu.Lock()
		if s.sound != a || s.screen != a.parent {
			s.mu.Unlock()
			return
		}
		if !s.soundValid(a) {
			for _, p := range a.peers {
				if p.stream != nil {
					p.stream.Close()
					p.stream = nil
				}
			}
			s.mu.Unlock()
			s.ScreenAudioStopCapture(a.parent.parent.session)
			continue
		}
		if !a.parent.publishing && a.capture != nil {
			s.mu.Unlock()
			s.ScreenAudioStopCapture(a.parent.parent.session)
			s.mu.Lock()
		}
		wanted := map[string]Participant{}
		for _, p := range a.parent.parent.lease.Participants {
			if p.ID != s.id {
				wanted[p.ID] = p
			}
		}
		for id, p := range a.peers {
			next, ok := wanted[id]
			if !ok || next.Session != p.participant.Session {
				if p.stream != nil {
					p.stream.Close()
				}
				delete(a.peers, id)
			}
		}
		t, ok := s.transport.(ScreenSoundTransport)
		for id, part := range wanted {
			p := a.peers[id]
			if p == nil {
				p = &soundPeer{participant: part, first: time.Now()}
				a.peers[id] = p
			}
			p.participant = part
			if ok && !p.connecting && !time.Now().Before(p.next) && (s.id < id || time.Since(p.first) > 3*time.Second) && (p.stream == nil || p.stream.Stats().State != "connected") {
				p.connecting = true
				s.wg.Add(1)
				go s.soundDial(a, p, part, t)
			}
		}
		if time.Since(lastReport) >= 3*time.Second {
			for _, p := range a.peers {
				if p.stream != nil {
					v := p.stream.Stats()
					slog.Info("screen_audio_network", "peer", correlation(p.participant.ID), "sent", v.PacketsOut, "received", v.PacketsIn, "state", v.State, "encoded", a.encoded)
				}
			}
			lastReport = time.Now()
		}
		s.mu.Unlock()
	}
}
func (s *Service) soundDial(a *screenSound, p *soundPeer, part Participant, t ScreenSoundTransport) {
	defer s.wg.Done()
	defer func() {
		s.mu.Lock()
		p.connecting = false
		if p.stream != nil && p.stream.Stats().State == "connected" {
			p.failures = 0
		} else {
			p.failures++
		}
		p.next = time.Now().Add(time.Second * time.Duration(1<<min(p.failures, 3)))
		s.mu.Unlock()
	}()
	if s.options.RegisterStreamRoute != nil {
		if s.options.RegisterStreamRoute(part) != nil {
			return
		}
	}
	s.mu.Lock()
	lease := a.parent.parent.lease
	session := a.parent.parent.session
	s.mu.Unlock()
	r := Request{Method: "screen-audio", Community: lease.Community, Channel: lease.Channel, Participant: Participant{ID: s.id, Session: session}, Session: part.Session, Lease: &lease, Nonce: community.RandomID()}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	conn, e := t.OpenScreenAudioMedia(ctx, part.ID, addresses(part.Invite), r)
	if e != nil {
		return
	}
	s.mu.Lock()
	valid := s.soundValid(a) && a.peers[part.ID] == p && p.participant.Session == part.Session && a.parent.parent.lease.Generation == lease.Generation && ctx.Err() == nil
	if valid && p.stream != nil && p.stream.Stats().State == "connected" && p.initiator <= s.id {
		valid = false
	}
	if valid {
		s.attachSound(a, p, conn, s.id, r.Nonce)
	}
	s.mu.Unlock()
	if !valid {
		conn.Close()
	}
}

// Called under the service mutex. Admission never trusts IP address or an unsigned room.
func (s *Service) AcceptScreenAudio(remote string, r Request, conn MediaConn) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := s.sound
	if !s.soundValid(a) || a.local == nil || r.Method != "screen-audio" || remote == s.id || r.Participant.ID != remote || r.Session != a.parent.parent.session || r.Community != a.parent.parent.group || r.Channel != a.parent.parent.channel || r.Lease == nil || len(r.Nonce) != 32 {
		return errScreenUnavailable
	}
	state, e := s.options.State(r.Community)
	if e != nil || VerifyLease(*r.Lease, state, r.Channel, time.Now()) != nil || r.Lease.Generation != a.parent.parent.lease.Generation {
		return errScreenUnavailable
	}
	self, other := false, false
	for _, p := range r.Lease.Participants {
		if p.ID == s.id && p.Session == r.Session {
			self = true
		}
		if p.ID == remote && p.Session == r.Participant.Session {
			other = true
		}
	}
	var part Participant
	for _, p := range a.parent.parent.lease.Participants {
		if p.ID == remote && p.Session == r.Participant.Session {
			part = p
		}
	}
	if !self || !other || part.ID == "" {
		return errScreenUnavailable
	}
	p := a.peers[remote]
	if p == nil {
		p = &soundPeer{participant: part, first: time.Now()}
		a.peers[remote] = p
	}
	if p.participant.Session != part.Session || p.stream != nil && p.stream.Stats().State == "connected" && p.initiator <= remote {
		return errors.New("duplicate screen audio stream")
	}
	if !reserveMediaNonce(&a.nonces, remote, r.Nonce, r.Lease.Expires, time.Now().Unix()) {
		return errors.New("screen audio replay rejected")
	}
	s.attachSound(a, p, conn, remote, r.Nonce)
	return nil
}
func (s *Service) attachSound(a *screenSound, p *soundPeer, conn MediaConn, initiator, nonce string) {
	old := p.stream
	p.initiator = initiator
	p.stream = newCorrelatedStreamMedia(conn, func(packet *rtp.Packet) {
		s.mu.Lock()
		pv := a.parent.peers[p.participant.ID]
		valid := s.soundValid(a) && a.peers[p.participant.ID] == p && a.parent.watching == p.participant.ID && pv != nil && pv.publishing && pv.accepted && pv.pipe != nil && pv.pipe.state.Load()
		local := a.local
		s.mu.Unlock()
		if valid && local != nil {
			_ = local.Write(packet)
		}
	}, "screen-audio/"+correlation(nonce))
	if old != nil {
		old.Close()
	}
}

func (s *Service) ScreenAudioDisconnect(expected, actor string) {
	s.mu.Lock()
	a := s.sound
	if a != nil && a.actor == actor && a.parent.parent.session == expected {
		a.cancel()
	}
	s.mu.Unlock()
}

func (s *Service) ScreenAudioStopCaptureForActor(expected, actor string, revision uint64) {
	s.mu.Lock()
	a := s.sound
	var source CaptureSource
	var cancel context.CancelFunc
	if a != nil && a.actor == actor && a.parent.parent.session == expected && revision >= a.captureRevision {
		a.captureRevision = revision
		source = a.capture
		a.capture = nil
		cancel = a.captureCancel
		a.captureCancel = nil
	}
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if source != nil {
		source.Close()
	}
}
