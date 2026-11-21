package voice

import (
	"context"
	"errors"
	"github.com/pion/webrtc/v4"
	"log/slog"
	"projetop2p/internal/community"
	"sort"
	"time"
)

type ScreenTransport interface {
	OpenScreenMedia(context.Context, string, []string, Request) (MediaConn, error)
}
type screenPeer struct {
	nextAttempt        time.Time
	failures           int
	lastError          string
	participant        Participant
	pipe               *videoPipe
	connecting         bool
	publishing         bool
	statusKnown        bool
	wants              bool
	accepted           bool
	pipeGeneration     uint64
	previous           screenPipeCounters
	lastRecoveryAt     time.Time
	lastRecoveryReason string
}
type screenCall struct {
	audioPaused        bool
	audioPriorityDrops uint64
	validatedUntil     time.Time
	parent             *call
	local              *videoBridge
	ctx                context.Context
	cancel             context.CancelFunc
	peers              map[string]*screenPeer
	publishing         bool
	watching           string
	playing            string
	visibleAt          time.Time
	bitrate            int
	maxViewers         int
	budgetAt           time.Time
	tokens             float64
	nonceCall          call
}
type ScreenPeerView struct {
	StatusKnown        bool   `json:"statusKnown"`
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Sharing            bool   `json:"sharing"`
	Connected          bool   `json:"connected"`
	Accepted           bool   `json:"accepted"`
	PipeGeneration     uint64 `json:"pipeGeneration"`
	LastRecoveryAt     int64  `json:"lastRecoveryAt,omitempty"`
	LastRecoveryReason string `json:"lastRecoveryReason,omitempty"`
}
type ScreenView struct {
	DropReasons         map[string]uint64 `json:"dropReasons"`
	Participants        []string          `json:"participants"`
	RoundTripMs         float64           `json:"roundTripMs"`
	Active              bool              `json:"active"`
	Publishing          bool              `json:"publishing"`
	Playing             string            `json:"playing"`
	Watching            string            `json:"watching"`
	Viewers             int               `json:"viewers"`
	Bitrate             int               `json:"bitrate"`
	MaxViewers          int               `json:"maxViewers"`
	FramesIn            uint64            `json:"framesIn"`
	FramesOut           uint64            `json:"framesOut"`
	Dropped             uint64            `json:"dropped"`
	BytesIn             uint64            `json:"bytesIn"`
	BytesOut            uint64            `json:"bytesOut"`
	SendQueue           *ScreenSendQueue  `json:"sendQueue,omitempty"`
	Peers               []ScreenPeerView  `json:"peers"`
	Error               string            `json:"error"`
	AudioPriorityPaused bool              `json:"audioPriorityPaused"`
}

type screenPipeCounters struct {
	in, out, bytesIn, bytesOut, drops    uint64
	queue, keyWait, late, window, budget uint64
}

func (c *screenPipeCounters) include(p *videoPipe) {
	if p == nil {
		return
	}
	c.in += p.in.Load()
	c.out += p.out.Load()
	c.bytesIn += p.bytesIn.Load()
	c.bytesOut += p.bytesOut.Load()
	c.drops += p.drops.Load()
	c.queue += p.queueDrops.Load()
	c.keyWait += p.keyWaitDrops.Load()
	c.late += p.lateDrops.Load()
	c.window += p.windowDrops.Load()
	c.budget += p.budgetDrops.Load()
}

func (c screenPipeCounters) addTo(v *ScreenView) {
	v.FramesIn += c.in
	v.FramesOut += c.out
	v.BytesIn += c.bytesIn
	v.BytesOut += c.bytesOut
	v.Dropped += c.drops
	v.DropReasons["queue"] += c.queue
	v.DropReasons["keyframeWait"] += c.keyWait
	v.DropReasons["lateOrUnauthorized"] += c.late
	v.DropReasons["window"] += c.window
	v.DropReasons["bandwidthOrAudioPriority"] += c.budget
}

// Each value is the maximum observed across the admitted, connected viewers.
// Wait measures local enqueue to dequeue, before framing or socket writes.
type ScreenSendQueue struct {
	Depth      int      `json:"depth"`
	Capacity   int      `json:"capacity"`
	LastWaitMs *float64 `json:"lastWaitMs"`
}

// Callers hold Service.mu. A room definition alone never grants a media session.
func (s *Service) screenAuthorized(c *call) bool {
	if s.closing || s.current != c || c == nil || c.authorizationInvalid || c.lease.Expires <= time.Now().Unix() {
		return false
	}
	state, e := s.options.State(c.group)
	return e == nil && VerifyLease(c.lease, state, c.channel, time.Now()) == nil && permitted(state, c.channel, s.id)
}
func (s *Service) ScreenConnect(ctx context.Context, offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	if e := checkVideoSDP(offer); e != nil {
		return webrtc.SessionDescription{}, e
	}
	s.ScreenStop()
	s.mu.Lock()
	c := s.current
	if !s.screenAuthorized(c) {
		s.mu.Unlock()
		return webrtc.SessionDescription{}, errScreenUnavailable
	}
	s.handlers.Add(1)
	defer s.handlers.Done()
	child, cancel := context.WithCancel(c.ctx)
	sc := &screenCall{parent: c, ctx: child, cancel: cancel, peers: map[string]*screenPeer{}, visibleAt: time.Now(), validatedUntil: time.Now().Add(1500 * time.Millisecond), bitrate: 1500000, maxViewers: 5, budgetAt: time.Now(), tokens: 128000}
	s.screen = sc
	s.mu.Unlock()
	local, e := newVideoBridge(func(f *videoFrame) { s.screenSend(sc, f) }, func() {
		s.mu.Lock()
		if s.screen == sc {
			if p := sc.peers[sc.watching]; p != nil && p.pipe != nil {
				p.pipe.keyframe()
			}
		}
		s.mu.Unlock()
	})
	if e != nil {
		s.screenStop(sc, nil)
		return webrtc.SessionDescription{}, e
	}
	s.mu.Lock()
	if s.screen != sc {
		s.mu.Unlock()
		local.close()
		return webrtc.SessionDescription{}, errScreenUnavailable
	}
	sc.local = local
	s.mu.Unlock()
	if e = local.pc.SetRemoteDescription(offer); e != nil {
		s.screenStop(sc, nil)
		return webrtc.SessionDescription{}, e
	}
	answer, e := local.pc.CreateAnswer(nil)
	if e != nil {
		s.screenStop(sc, nil)
		return answer, e
	}
	ready := webrtc.GatheringCompletePromise(local.pc)
	if e = local.pc.SetLocalDescription(answer); e != nil {
		s.screenStop(sc, nil)
		return answer, e
	}
	select {
	case <-ctx.Done():
		s.screenStop(sc, nil)
		return answer, ctx.Err()
	case <-sc.ctx.Done():
		return answer, sc.ctx.Err()
	case <-ready:
	}
	s.mu.Lock()
	if s.screen != sc {
		s.mu.Unlock()
		return answer, errScreenUnavailable
	}
	s.wg.Add(1)
	s.mu.Unlock()
	go s.screenWatch(sc)
	return *local.pc.LocalDescription(), nil
}
func (s *Service) ScreenStop() { s.screenStop(nil, nil) }
func (s *Service) screenLive(sc *screenCall) bool {
	return s.screen == sc && s.current == sc.parent && !s.closing && !sc.parent.authorizationInvalid && sc.parent.lease.Expires > time.Now().Unix() && time.Now().Before(sc.validatedUntil)
}
func (s *Service) screenStop(expected *screenCall, parent *call) {
	s.mu.Lock()
	sc := s.screen
	if sc == nil || expected != nil && sc != expected || parent != nil && sc.parent != parent {
		s.mu.Unlock()
		return
	}
	s.screen = nil
	var pipes []*videoPipe
	var local *videoBridge
	if sc != nil {
		sc.cancel()
		local = sc.local
		for _, p := range sc.peers {
			if p.pipe != nil {
				pipes = append(pipes, p.pipe)
			}
		}
	}
	s.mu.Unlock()
	for _, p := range pipes {
		p.close()
	}
	if local != nil {
		local.close()
	}
}
func (s *Service) ScreenHeartbeat() {
	s.mu.Lock()
	if s.screen != nil {
		s.screen.visibleAt = time.Now()
	}
	s.mu.Unlock()
}
func (s *Service) ScreenPublish(publish bool, bitrate, maxViewers int) error {
	if bitrate < 256000 || bitrate > 12000000 || maxViewers < 1 || maxViewers > 5 {
		return errors.New("screen resource limit: 256–12000 kbps, 1–5 viewers")
	}
	s.mu.Lock()
	sc := s.screen
	if sc == nil || !s.screenAuthorized(sc.parent) {
		s.mu.Unlock()
		return errScreenUnavailable
	}
	sc.publishing = publish
	sc.audioPaused = false
	sc.bitrate = bitrate
	sc.maxViewers = maxViewers
	sc.visibleAt = time.Now()
	var pipes []*videoPipe
	for _, p := range sc.peers {
		if p.pipe != nil {
			pipes = append(pipes, p.pipe)
		}
	}
	local := sc.local
	s.mu.Unlock()
	// Abort queued/in-flight video on stop; audio uses its independent protocol.
	if !publish {
		s.ScreenAudioStopCapture(sc.parent.session)
		for _, p := range pipes {
			p.close()
		}
	} else if local != nil {
		local.requestKeyframe()
	}
	return nil
}
func (s *Service) ScreenWatchPeer(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc := s.screen
	if sc == nil || !s.screenAuthorized(sc.parent) {
		return errScreenUnavailable
	}
	if id != "" {
		p := sc.peers[id]
		if p == nil {
			return errors.New("screen viewer is not in the authorized room")
		}
		if p.pipe != nil {
			p.pipe.keyframe()
		}
	}
	if sc.watching != id {
		sc.playing = ""
	}
	sc.watching = id
	return nil
}
func (s *Service) ScreenState() ScreenView {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := ScreenView{Peers: []ScreenPeerView{}, DropReasons: map[string]uint64{}}
	sc := s.screen
	if sc == nil {
		return v
	}
	for _, participant := range sc.parent.lease.Participants {
		v.Participants = append(v.Participants, participant.ID)
	}
	v.Active = true
	v.Publishing = sc.publishing
	v.Watching = sc.watching
	v.Playing = sc.playing
	v.Bitrate = sc.bitrate
	v.MaxViewers = sc.maxViewers
	v.AudioPriorityPaused = sc.audioPaused && sc.publishing
	v.DropReasons["audioPriority"] = sc.audioPriorityDrops
	if v.AudioPriorityPaused {
		v.Error = "Video pausado temporariamente para priorizar a conexao de voz"
	}
	for id, p := range sc.peers {
		if p.lastError != "" && (p.pipe == nil || !p.pipe.state.Load()) {
			v.Error = p.lastError
		}
		connected := p.pipe != nil && p.pipe.state.Load()
		peer := ScreenPeerView{ID: id, Name: p.participant.Name, Sharing: p.publishing, Connected: connected, Accepted: connected && p.accepted, StatusKnown: connected && p.statusKnown, PipeGeneration: p.pipeGeneration, LastRecoveryReason: p.lastRecoveryReason}
		if !p.lastRecoveryAt.IsZero() {
			peer.LastRecoveryAt = p.lastRecoveryAt.UnixMilli()
		}
		v.Peers = append(v.Peers, peer)
		if !connected && v.Playing == id {
			v.Playing = ""
		}
		if connected && p.wants {
			v.Viewers++
		}
		counters := p.previous
		counters.include(p.pipe)
		counters.addTo(&v)
		if p.pipe != nil {
			if sc.publishing && p.pipe.state.Load() && screenAccepts(sc, id) && cap(p.pipe.frames) > 0 {
				if v.SendQueue == nil {
					v.SendQueue = &ScreenSendQueue{}
				}
				v.SendQueue.Depth = max(v.SendQueue.Depth, len(p.pipe.frames))
				v.SendQueue.Capacity = max(v.SendQueue.Capacity, cap(p.pipe.frames))
				if observed := p.pipe.lastQueueWait.Load(); observed != 0 {
					ms := float64(observed-1) / float64(time.Millisecond)
					if v.SendQueue.LastWaitMs == nil || ms > *v.SendQueue.LastWaitMs {
						v.SendQueue.LastWaitMs = &ms
					}
				}
			}
			if rtt := float64(p.pipe.rtt.Load()) / 1e6; rtt > v.RoundTripMs {
				v.RoundTripMs = rtt
			}
		}
	}
	if v.Viewers > sc.maxViewers {
		v.Viewers = sc.maxViewers
	}
	return v
}
func (s *Service) screenSend(sc *screenCall, f *videoFrame) {
	s.mu.Lock()
	if !s.screenLive(sc) || !sc.publishing || time.Since(sc.visibleAt) > 3*time.Second {
		s.mu.Unlock()
		return
	}
	ids := []string{}
	for id, p := range sc.peers {
		if p.wants && p.pipe != nil && p.pipe.state.Load() {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) > sc.maxViewers {
		ids = ids[:sc.maxViewers]
	}
	pipes := []*videoPipe{}
	for _, id := range ids {
		pipes = append(pipes, sc.peers[id].pipe)
	}
	s.mu.Unlock()
	for _, p := range pipes {
		p.push(f)
	}
}
func (s *Service) screenPipe(sc *screenCall, p *screenPeer, conn MediaConn) *videoPipe {
	var pipe *videoPipe
	pipe = newVideoPipe(conn, func(f *videoFrame) {
		s.mu.Lock()
		local := sc.local
		valid := s.screenLive(sc) && sc.watching == p.participant.ID && sc.peers[p.participant.ID] == p && p.pipe == pipe
		s.mu.Unlock()
		if valid && local != nil {
			if local.write(f) == nil {
				s.mu.Lock()
				if s.screen == sc && sc.watching == p.participant.ID && p.pipe == pipe && pipe.state.Load() {
					sc.playing = p.participant.ID
				}
				s.mu.Unlock()
			}
		}
	}, func(flags byte) {
		s.mu.Lock()
		if s.screen != sc || sc.peers[p.participant.ID] != p || p.pipe != pipe {
			s.mu.Unlock()
			return
		}
		if flags == 0x80 { // Internal stream loss is not a publisher Stop announcement.
			p.statusKnown = false
			if sc.watching == p.participant.ID {
				sc.playing = ""
			}
			p.accepted = false
			p.wants = false
			s.mu.Unlock()
			return
		}
		p.statusKnown = true
		p.publishing = flags&1 != 0
		if !p.publishing && sc.playing == p.participant.ID {
			sc.playing = ""
		}
		want := flags&2 != 0
		becameViewer := want && !p.wants
		if becameViewer {
			pipe.resetFrames()
		}
		p.wants = want
		p.accepted = flags&4 != 0
		s.mu.Unlock()
		if becameViewer {
			pipe.requestPublisherKey()
		}
	}, func() {
		s.mu.Lock()
		local := sc.local
		active := s.screen == sc && sc.publishing
		s.mu.Unlock()
		if active && local != nil {
			local.requestKeyframe()
		}
	}, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.screenLive(sc) && p.pipe == pipe && sc.peers[p.participant.ID] == p && sc.publishing && screenAccepts(sc, p.participant.ID) && time.Since(sc.visibleAt) < 3*time.Second
	}, func(bytes int) bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.screen != sc {
			return false
		}
		sc.audioPaused = false
		for _, audio := range sc.parent.peers {
			// An old delay sample can stay high throughout silence or mute.
			// Use observed pending Opus frames or a disconnected voice stream.
			if audio.stream == nil || voiceHasBackpressure(audio.stream) {
				sc.audioPaused = true
				sc.audioPriorityDrops++
				return false
			}
		}
		now := time.Now()
		sc.tokens += now.Sub(sc.budgetAt).Seconds() * float64(sc.bitrate) / 8
		sc.budgetAt = now
		cap := max(float64(sc.bitrate)/32+65536, float64(maxVideoFrame+2048))
		if sc.tokens > cap {
			sc.tokens = cap
		}
		if sc.tokens < float64(bytes) {
			return false
		}
		sc.tokens -= float64(bytes)
		return true
	})
	return pipe
}

func voiceHasBackpressure(stream *StreamMedia) bool {
	stats := stream.Stats()
	return stats.State != "connected" || stats.QueueDepth >= 2
}

func screenFlags(sc *screenCall, id string) byte {
	var flags byte
	if sc.publishing {
		flags |= 1
	}
	if sc.watching == id {
		flags |= 2
	}
	if screenAccepts(sc, id) {
		flags |= 4
	}
	return flags
}

// Callers hold Service.mu. Replacement preserves the subscription, publication
// and counters, but requires fresh remote status and a complete keyframe.
func (s *Service) replaceScreenPipe(sc *screenCall, p *screenPeer, conn MediaConn) {
	if old := p.pipe; old != nil {
		p.previous.include(old)
		p.lastRecoveryAt = time.Now()
		p.lastRecoveryReason, _ = old.closedReason.Load().(string)
		slog.Info("screen_transport_recovery", "peer", correlation(p.participant.ID), "reason", p.lastRecoveryReason, "result", "stream_replaced")
	}
	if sc.playing == p.participant.ID {
		sc.playing = ""
	}
	p.statusKnown = false
	p.wants = false
	p.accepted = false
	p.pipe = s.screenPipe(sc, p, conn)
	p.pipeGeneration++
}
func (s *Service) screenWatch(sc *screenCall) {
	defer s.wg.Done()
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-sc.ctx.Done():
			return
		case <-tick.C:
		}
		// DB reads and signature/route validation must not hold the audio mutex.
		// Fence the result against a capability replaced while validation ran.
		lease, valid := s.validateScreenLease(sc)
		s.mu.Lock()
		if s.screen != sc {
			s.mu.Unlock()
			return
		}
		if sc.parent.lease.Generation != lease.Generation || sc.parent.lease.Sequence != lease.Sequence || sc.parent.lease.Expires != lease.Expires {
			s.mu.Unlock()
			continue
		}
		if !valid || s.current != sc.parent || sc.parent.authorizationInvalid || s.closing {
			s.mu.Unlock()
			s.screenStop(sc, nil)
			return
		}
		sc.validatedUntil = time.Now().Add(1500 * time.Millisecond)
		if time.Since(sc.visibleAt) > 3*time.Second {
			sc.publishing = false
			sc.watching = ""
		}
		t, ok := s.transport.(ScreenTransport)
		if !ok {
			s.mu.Unlock()
			s.screenStop(sc, nil)
			return
		}
		wanted := map[string]Participant{}
		for _, p := range sc.parent.lease.Participants {
			if p.ID != s.id {
				wanted[p.ID] = p
			}
		}
		var closePipes []*videoPipe
		for id, p := range sc.peers {
			next, ok := wanted[id]
			if !ok || next.Session != p.participant.Session {
				if p.pipe != nil {
					closePipes = append(closePipes, p.pipe)
				}
				delete(sc.peers, id)
			}
		}
		for id, part := range wanted {
			p := sc.peers[id]
			if p == nil {
				p = &screenPeer{participant: part}
				sc.peers[id] = p
			}
			if p.pipe != nil && p.pipe.state.Load() {
				p.pipe.announce(screenFlags(sc, id))
			}
			if s.id < id && !p.connecting && !time.Now().Before(p.nextAttempt) && (p.pipe == nil || !p.pipe.state.Load()) {
				p.connecting = true
				s.wg.Add(1)
				go func(p *screenPeer) {
					defer s.wg.Done()
					defer s.screenDialCompleted(p)
					s.screenDial(sc, p, t)
				}(p)
			}
		}
		s.mu.Unlock()
		for _, p := range closePipes {
			p.close()
		}
	}
}
func (s *Service) screenDial(sc *screenCall, p *screenPeer, t ScreenTransport) {
	if s.options.RegisterStreamRoute != nil {
		if s.options.RegisterStreamRoute(p.participant) != nil {
			s.mu.Lock()
			p.lastError = "Rota assinada de tela indisponivel"
			s.mu.Unlock()
			return
		}
	}
	s.mu.Lock()
	if s.screen != sc {
		s.mu.Unlock()
		return
	}
	lease := sc.parent.lease
	request := Request{Method: "screen", Community: sc.parent.group, Channel: sc.parent.channel, Participant: Participant{ID: s.id, Session: sc.parent.session}, Session: p.participant.Session, Lease: &lease, Nonce: community.RandomID()}
	s.mu.Unlock()
	ctx, cancel := context.WithTimeout(sc.ctx, 15*time.Second)
	defer cancel()
	conn, e := t.OpenScreenMedia(ctx, p.participant.ID, addresses(p.participant.Invite), request)
	if e != nil {
		s.mu.Lock()
		p.lastError = "Transporte de tela indisponivel ou sessao remota ainda nao preparada"
		s.mu.Unlock()
		return
	}
	s.mu.Lock()
	if s.screen != sc || sc.peers[p.participant.ID] != p {
		s.mu.Unlock()
		conn.Close()
		return
	}
	p.lastError = ""
	s.replaceScreenPipe(sc, p, conn)
	p.pipe.announce(screenFlags(sc, p.participant.ID))
	s.mu.Unlock()
}
func (s *Service) AcceptScreen(remote string, r Request, conn MediaConn) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc := s.screen
	if sc == nil || !s.screenAuthorized(sc.parent) || r.Method != "screen" || r.Community != sc.parent.group || r.Channel != sc.parent.channel || r.Session != sc.parent.session || r.Participant.ID != remote || r.Lease == nil || r.Lease.Generation != sc.parent.lease.Generation || len(r.Nonce) != 32 || remote >= s.id {
		return errScreenUnavailable
	}
	state, e := s.options.State(r.Community)
	if e != nil || VerifyLease(*r.Lease, state, r.Channel, time.Now()) != nil {
		return errScreenUnavailable
	}
	self, other := false, false
	for _, p := range r.Lease.Participants {
		if p.ID == s.id && p.Session == sc.parent.session {
			self = true
		}
		if p.ID == remote && p.Session == r.Participant.Session {
			other = true
		}
	}
	p := sc.peers[remote]
	if !self || !other || p == nil || p.participant.Session != r.Participant.Session || p.pipe != nil && p.pipe.state.Load() {
		return errScreenUnavailable
	}
	if !reserveMediaNonce(&sc.nonceCall, remote, r.Nonce, r.Lease.Expires, time.Now().Unix()) {
		return errors.New("screen replay/rate limit")
	}
	s.replaceScreenPipe(sc, p, conn)
	return nil
}

// Stable subscriber admission avoids rotating viewers or exceeding the chosen fanout.
func screenAccepts(sc *screenCall, id string) bool {
	p := sc.peers[id]
	if p == nil || !p.wants {
		return false
	}
	rank := 0
	for other, v := range sc.peers {
		if v.wants && other < id {
			rank++
		}
	}
	return rank < sc.maxViewers
}

func (s *Service) screenDialCompleted(p *screenPeer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.connecting = false
	if p.lastError != "" {
		p.failures++
		p.nextAttempt = time.Now().Add(500 * time.Millisecond * time.Duration(1<<min(p.failures-1, 4)))
	} else {
		p.failures = 0
		p.nextAttempt = time.Time{}
	}
}

// Snapshot under the media lock; potentially slow state lookup stays outside it.
func (s *Service) validateScreenLease(sc *screenCall) (Lease, bool) {
	s.mu.Lock()
	lease := sc.parent.lease
	s.mu.Unlock()
	state, e := s.options.State(sc.parent.group)
	return lease, e == nil && VerifyLease(lease, state, sc.parent.channel, time.Now()) == nil && permitted(state, sc.parent.channel, s.id)
}
