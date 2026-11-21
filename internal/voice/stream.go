package voice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/pion/rtp"
	"io"
	"log/slog"
	"projetop2p/internal/community"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type MediaConn interface {
	io.ReadWriteCloser
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}
type StreamTransport interface {
	OpenVoiceMedia(context.Context, string, []string, Request) (MediaConn, error)
}
type queuedAudio struct {
	packet *rtp.Packet
	at     time.Time
}
type StreamMedia struct {
	flow                                     string
	conn                                     MediaConn
	out                                      chan queuedAudio
	done                                     chan struct{}
	once                                     sync.Once
	state                                    atomic.Value
	in, outCount, bytesIn, bytesOut, dropped atomic.Uint64
	lastReceive                              atomic.Int64
	delay                                    atomic.Int64
}

func correlation(value string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))[:12] }
func (m *StreamMedia) Close()         { m.closeReason("local", "consumer_close") }
func (m *StreamMedia) closeReason(origin, reason string) {
	m.once.Do(func() {
		m.state.Store("reconnecting")
		if origin == "local" {
			slog.Info("voice_transport", "flow", m.flow, "state", "closed", "origin", origin, "reason", reason)
		} else {
			slog.Warn("voice_transport", "flow", m.flow, "state", "closed", "origin", origin, "reason", reason)
		}
		close(m.done)
		m.conn.Close()
	})
}
func (m *StreamMedia) Stats() MediaStats {
	state, _ := m.state.Load().(string)
	return MediaStats{LastReceiveMs: m.lastReceive.Load(), QueueDepth: len(m.out), State: state, PacketsIn: m.in.Load(), PacketsOut: m.outCount.Load(), BytesIn: m.bytesIn.Load(), BytesOut: m.bytesOut.Load(), Dropped: m.dropped.Load(), QueueDelayMs: float64(m.delay.Load()) / 1e6, Transport: "libp2p-stream"}
}
func (m *StreamMedia) Push(p *rtp.Packet) {
	if len(p.Payload) > 1200 || !opus20ms(p.Payload) {
		return
	}
	q := queuedAudio{p.Clone(), time.Now()}
	select {
	case <-m.done:
		return
	default:
	}
	select {
	case m.out <- q:
	default:
		select {
		case <-m.out:
			m.dropped.Add(1)
		default:
		}
		select {
		case m.out <- q:
		default:
			m.dropped.Add(1)
		}
	}
}
func newStreamMedia(conn MediaConn, receive func(*rtp.Packet)) *StreamMedia {
	return newCorrelatedStreamMedia(conn, receive, "")
}
func newCorrelatedStreamMedia(conn MediaConn, receive func(*rtp.Packet), flow string) *StreamMedia {
	m := &StreamMedia{flow: flow, conn: conn, out: make(chan queuedAudio, 3), done: make(chan struct{})}
	m.state.Store("connected")
	slog.Info("voice_transport", "flow", flow, "state", "connected", "transport", "libp2p-stream")
	go func() {
		defer m.Close()
		keep := time.NewTicker(time.Second)
		defer keep.Stop()
		lastReport := time.Now()
		for {
			if time.Since(lastReport) >= 5*time.Second {
				slog.Info("voice_media_counters", "flow", m.flow, "sent", m.outCount.Load(), "received", m.in.Load(), "discarded", m.dropped.Load(), "queue_ms", float64(m.delay.Load())/1e6)
				lastReport = time.Now()
			}
			var data []byte
			select {
			case <-m.done:
				return
			case q := <-m.out:
				if time.Since(q.at) > 100*time.Millisecond {
					m.dropped.Add(1)
					continue
				}
				m.delay.Store(int64(time.Since(q.at)))
				raw, e := q.packet.Marshal()
				if e != nil {
					return
				}
				data = make([]byte, 8+len(raw))
				binary.BigEndian.PutUint64(data, uint64(q.at.UnixMilli()))
				copy(data[8:], raw)
			case <-keep.C:
			}
			conn.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
			var header [2]byte
			binary.BigEndian.PutUint16(header[:], uint16(len(data)))
			frame := append(header[:], data...)
			if _, e := io.Copy(conn, bytes.NewReader(frame)); e != nil {
				m.closeReason("transport", "write: "+e.Error())
				return
			}
			if len(data) > 0 {
				m.outCount.Add(1)
				m.bytesOut.Add(uint64(len(data) - 8))
			}
		}
	}()
	go func() {
		defer m.Close()
		play := make(chan queuedAudio, 3)
		playDone := make(chan struct{})
		defer close(playDone)
		go func() {
			tick := time.NewTicker(20 * time.Millisecond)
			defer tick.Stop()
			for {
				select {
				case <-playDone:
					return
				case <-m.done:
					return
				case <-tick.C:
					select {
					case q := <-play:
						if time.Since(q.at) <= 100*time.Millisecond {
							receive(q.packet)
						} else {
							m.dropped.Add(1)
						}
					default:
					}
				}
			}
		}()
		var offset int64
		windowStart := time.Now()
		framesInWindow := 0
		first := true
		for {
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			var header [2]byte
			if _, e := io.ReadFull(conn, header[:]); e != nil {
				origin := "transport"
				if strings.Contains(e.Error(), "remote") {
					origin = "remote"
				}
				m.closeReason(origin, "read header: "+e.Error())
				return
			}
			if time.Since(windowStart) >= time.Second {
				windowStart = time.Now()
				framesInWindow = 0
			}
			framesInWindow++
			if framesInWindow > 120 {
				slog.Warn("voice_transport", "reason", "media_rate_limit")
				return
			}
			n := int(binary.BigEndian.Uint16(header[:]))
			if n == 0 {
				continue
			}
			if n < 20 || n > 1300 {
				slog.Warn("voice_transport", "reason", "invalid_frame_length")
				return
			}
			data := make([]byte, n)
			if _, e := io.ReadFull(conn, data); e != nil {
				slog.Warn("voice_transport", "state", "disconnected", "reason", "read frame: "+e.Error())
				return
			}
			now := time.Now()
			transit := now.UnixMilli() - int64(binary.BigEndian.Uint64(data))
			if first || transit < offset {
				offset = transit
				first = false
			}
			if transit-offset > 150 {
				m.dropped.Add(1)
				continue
			}
			var p rtp.Packet
			if p.Unmarshal(data[8:]) != nil || len(p.Payload) > 1200 {
				return
			}
			if !opus20ms(p.Payload) {
				m.dropped.Add(1)
				continue
			}
			m.in.Add(1)
			m.lastReceive.Store(now.UnixMilli())
			m.bytesIn.Add(uint64(n - 8))
			q := queuedAudio{&p, now}
			select {
			case play <- q:
			default:
				select {
				case <-play:
					m.dropped.Add(1)
				default:
				}
				select {
				case play <- q:
				default:
				}
			}
		}
	}()
	return m
}
func (s *Service) receivePeer(c *call, expected *roomPeer, packet *rtp.Packet) {
	s.mu.Lock()
	local := c.local
	ok := s.current == c && c.peers[expected.participant.ID] == expected && !c.deaf.Load() && !c.authorizationInvalid && time.Now().Unix() < c.lease.Expires
	slot := expected.slot
	s.mu.Unlock()
	if ok && local != nil {
		_ = local.WriteSlot(slot, packet)
	}
}
func (s *Service) attachStream(c *call, other Participant, conn MediaConn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := c.peers[other.ID]
	if s.current != c || p == nil || p.participant.Session != other.Session || c.ctx.Err() != nil || c.authorizationInvalid || c.lease.Expires <= time.Now().Unix() {
		conn.Close()
		return
	}
	old := p.stream
	p.stream = newStreamMedia(conn, func(packet *rtp.Packet) { s.receivePeer(c, p, packet) })
	c.stream = p.stream
	c.remoteID = other.ID
	c.remoteSession = other.Session
	if old != nil {
		old.Close()
	}
}
func (s *Service) reconcileMesh(c *call, lease Lease, t StreamTransport) {
	s.mu.Lock()
	if s.current != c || c.lease.Generation != lease.Generation || c.lease.Sequence != lease.Sequence {
		s.mu.Unlock()
		return
	}
	wanted := map[string]Participant{}
	for _, p := range lease.Participants {
		if p.ID != s.id {
			wanted[p.ID] = p
		}
	}
	for id, p := range c.peers {
		next, ok := wanted[id]
		if !ok || next.Session != p.participant.Session {
			if p.attemptCancel != nil {
				p.attemptCancel()
			}
			if p.stream != nil {
				p.stream.Close()
			}
			delete(c.peers, id)
		}
	}
	for id, participant := range wanted {
		p := c.peers[id]
		if p == nil {
			used := map[int]bool{}
			for _, x := range c.peers {
				used[x.slot] = true
			}
			slot := 0
			for used[slot] {
				slot++
			}
			if slot >= 5 {
				continue
			}
			p = &roomPeer{participant: participant, slot: slot, firstSeen: time.Now()}
			c.peers[id] = p
		}
		// A renewed signed route for the same identity/session replaces the old endpoint.
		p.participant = participant
		if p.firstSeen.IsZero() {
			p.firstSeen = time.Now()
		}
		if (s.id < id || time.Since(p.firstSeen) >= 3*time.Second) && !p.connecting && !time.Now().Before(p.nextAttempt) && (p.stream == nil || p.stream.Stats().State != "connected") {
			p.connecting = true
			attemptCtx, cancel := context.WithTimeout(c.ctx, 15*time.Second)
			p.attemptCancel = cancel
			snapshot := p.participant
			s.wg.Add(1)
			go func(p *roomPeer, other Participant) {
				defer s.wg.Done()
				defer cancel()
				s.connectStreamAttempt(attemptCtx, c, p, other, t)
				s.mu.Lock()
				p.connecting = false
				p.attemptCancel = nil
				if p.stream != nil && p.stream.Stats().State == "connected" {
					p.failures = 0
					p.nextAttempt = time.Time{}
				} else {
					p.failures++
					delay := time.Second * time.Duration(1<<min(p.failures-1, 3))
					p.nextAttempt = time.Now().Add(delay)
				}
				s.mu.Unlock()
			}(p, snapshot)
		}
	}
	s.mu.Unlock()
}
func (s *Service) connectStream(c *call, other Participant, t StreamTransport) {
	ctx, cancel := context.WithTimeout(c.ctx, 15*time.Second)
	defer cancel()
	s.mu.Lock()
	expected := c.peers[other.ID]
	s.mu.Unlock()
	s.connectStreamAttempt(ctx, c, expected, other, t)
}
func (s *Service) connectStreamAttempt(ctx context.Context, c *call, expected *roomPeer, other Participant, t StreamTransport) {
	if s.options.RegisterStreamRoute != nil {
		if e := s.options.RegisterStreamRoute(other); e != nil {
			s.setError(c, e)
			return
		}
	}
	s.mu.Lock()
	lease := c.lease
	s.mu.Unlock()
	req := Request{Method: "media", Community: c.group, Channel: c.channel, Participant: Participant{ID: s.id, Session: c.session}, Lease: &lease, Session: other.Session, Nonce: community.RandomID()}
	conn, e := t.OpenVoiceMedia(ctx, other.ID, addresses(other.Invite), req)
	if e != nil {
		s.mu.Lock()
		if expected != nil && s.current == c && c.peers[other.ID] == expected {
			expected.lastError = e.Error()
		}
		s.mu.Unlock()
		s.setError(c, e)
		return
	}
	s.mu.Lock()
	valid := s.current == c && c.peers[other.ID] == expected && expected != nil && expected.participant.Session == other.Session && c.lease.Generation == lease.Generation && !c.authorizationInvalid && c.lease.Expires > time.Now().Unix() && ctx.Err() == nil
	if valid && expected.stream != nil && expected.stream.Stats().State == "connected" && expected.initiator != "" && expected.initiator <= s.id {
		valid = false
	}
	if valid {
		expected.initiator = s.id
		expected.lastError = ""
		old := expected.stream
		expected.stream = newCorrelatedStreamMedia(conn, func(packet *rtp.Packet) { s.receivePeer(c, expected, packet) }, correlation(req.Nonce))
		c.stream = expected.stream
		c.mediaError = ""
		c.lastError = c.authorizationError
		if old != nil {
			old.Close()
		}
	}
	s.mu.Unlock()
	if !valid {
		conn.Close()
	}
}
func (s *Service) AcceptMedia(remote string, r Request, conn MediaConn) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.current
	if s.closing || c == nil || r.Method != "media" || c.group != r.Community || c.channel != r.Channel || c.session != r.Session || r.Participant.ID != remote || remote == s.id || len(r.Nonce) != 32 || r.Lease == nil {
		return errors.New("unauthorized media session")
	}
	state, e := s.options.State(c.group)
	if e != nil {
		return e
	}
	if e = VerifyLease(*r.Lease, state, c.channel, time.Now()); e != nil {
		return e
	}

	var other Participant
	self := false
	for _, p := range r.Lease.Participants {
		if p.ID == remote && p.Session == r.Participant.Session {
			other = p
		}
		if p.ID == s.id && p.Session == c.session {
			self = true
		}
	}
	if other.ID == "" || !self {
		return errors.New("participants not authorized")
	}
	incoming := *r.Lease
	newer := leaseCanAdvance(incoming, c.lease)
	if newer {
		c.lease = incoming
		c.authorizationInvalid = false
		c.authorizationError = ""
	}
	if c.authorizationInvalid || c.lease.Expires <= time.Now().Unix() || c.lease.Generation != incoming.Generation {
		return errors.New("stale local authorization")
	}
	known := false
	for _, x := range c.lease.Participants {
		if x.ID == other.ID && x.Session == other.Session {
			known = true
		}
	}
	if !known {
		return errors.New("participants not authorized")
	}
	p := c.peers[other.ID]
	if p == nil {
		used := map[int]bool{}
		for _, x := range c.peers {
			used[x.slot] = true
		}
		slot := 0
		for used[slot] {
			slot++
		}
		if slot >= 5 {
			return errors.New("media slot limit")
		}
		p = &roomPeer{participant: other, slot: slot}
		c.peers[other.ID] = p
	}
	if p.participant.Session != other.Session {
		return errors.New("peer session not synchronized")
	}

	if p.stream != nil && p.stream.Stats().State == "connected" && (p.initiator == "" || p.initiator <= remote) {
		return errors.New("active preferred media stream; duplicate rejected")
	}
	if !reserveMediaNonce(c, remote, r.Nonce, r.Lease.Expires, time.Now().Unix()) {
		return errors.New("media replay/rate limit")
	}
	old := p.stream
	p.initiator = remote
	p.failures = 0
	p.lastError = ""
	p.stream = newCorrelatedStreamMedia(conn, func(packet *rtp.Packet) { s.receivePeer(c, p, packet) }, correlation(r.Nonce))
	c.stream = p.stream
	c.mediaError = ""
	c.lastError = c.authorizationError
	c.remoteID = other.ID
	c.remoteSession = other.Session
	if old != nil {
		old.Close()
	}

	return nil
}

// RFC 6716 TOC duration, expressed in half-milliseconds; no codec implementation.
func opus20ms(p []byte) bool {
	if len(p) == 0 {
		return false
	}
	config := p[0] >> 3
	duration := 0
	if config >= 16 {
		duration = 5 << uint(config&3)
	} else if config >= 12 {
		duration = 20 << uint(config&1)
	} else {
		duration = []int{20, 40, 80, 120}[config&3]
	}
	frames := 1
	switch p[0] & 3 {
	case 1, 2:
		frames = 2
	case 3:
		if len(p) < 2 {
			return false
		}
		frames = int(p[1] & 63)
	}
	return frames > 0 && duration*frames == 40
}

func (m *StreamMedia) Flush() {
	for {
		select {
		case <-m.out:
			m.dropped.Add(1)
		default:
			return
		}
	}
}

// Caller holds Service.mu. Replay protection covers the whole signed offline
// window; one authorized member cannot consume every other member's budget.
func reserveMediaNonce(c *call, remote, nonce string, expires, now int64) bool {
	if c.mediaNonces == nil {
		c.mediaNonces = map[string]int64{}
	}
	prefix := remote + ":"
	count := 0
	for key, expiry := range c.mediaNonces {
		if expiry <= now {
			delete(c.mediaNonces, key)
		} else if strings.HasPrefix(key, prefix) {
			count++
		}
	}
	key := prefix + nonce
	if _, seen := c.mediaNonces[key]; seen || count >= 256 || len(c.mediaNonces) >= MaxParticipants*256 {
		return false
	}
	c.mediaNonces[key] = expires
	return true
}
