package voice

import (
	"context"
	"errors"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"net"
	"os"
	"projetop2p/internal/community"
	"projetop2p/internal/loopback"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

type soundTestTransport struct{ other *Service }

func (t soundTestTransport) VoiceRequest(context.Context, string, []string, Request) (Response, error) {
	return Response{}, errors.New("unit transport")
}
func (t soundTestTransport) OpenScreenAudioMedia(ctx context.Context, id string, addr []string, r Request) (MediaConn, error) {
	a, b := net.Pipe()
	if e := t.other.AcceptScreenAudio(r.Participant.ID, r, b); e != nil {
		a.Close()
		b.Close()
		return nil, e
	}
	return a, nil
}
func soundOffer(t *testing.T, p *MediaPeer) webrtc.SessionDescription {
	t.Helper()
	offer, e := p.PC.CreateOffer(nil)
	if e != nil {
		t.Fatal(e)
	}
	ready := webrtc.GatheringCompletePromise(p.PC)
	if e = p.PC.SetLocalDescription(offer); e != nil {
		t.Fatal(e)
	}
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("ICE")
	}
	return *p.PC.LocalDescription()
}
func TestScreenAudioIndependentAuthorizedFlowAndCleanup(t *testing.T) {
	f := hotfixRoom(t)
	a, b := f.people[1], f.people[2]
	f.join(t, a)
	lease := f.join(t, b)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var heard atomic.Uint64
	svc := make([]*Service, 2)
	browsers := make([]*MediaPeer, 2)
	people := []Participant{a, b}
	for i, p := range people {
		engine, e := NewEngine("127.0.0.1")
		if e != nil {
			t.Fatal(e)
		}
		s := &Service{id: p.ID, ctx: ctx, localEngine: engine, options: Options{State: func(string) (community.State, error) { return f.state, nil }}}
		callCtx, cc := context.WithCancel(ctx)
		c := &call{ctx: callCtx, cancel: cc, group: f.state.ID, channel: "voz", session: p.Session, lease: lease, peers: map[string]*roomPeer{}}
		s.current = c
		sc := &screenCall{parent: c, ctx: callCtx, cancel: cc, validatedUntil: time.Now().Add(14 * time.Minute), visibleAt: time.Now(), publishing: i == 0, peers: map[string]*screenPeer{}, maxViewers: 5}
		s.screen = sc
		other := people[1-i]
		pipe := &videoPipe{}
		pipe.state.Store(true)
		sc.peers[other.ID] = &screenPeer{participant: other, pipe: pipe, wants: i == 0, publishing: i == 1, accepted: true, statusKnown: true}
		if i == 1 {
			sc.watching = a.ID
		}
		svc[i] = s
		browserEngine, e := NewEngine("127.0.0.1")
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(browserEngine.Close)
		browser, e := browserEngine.Peer(func(*rtp.Packet) {
			if i == 1 {
				heard.Add(1)
			}
		})
		if e != nil {
			t.Fatal(e)
		}
		browsers[i] = browser
		answer, e := s.ScreenAudioConnect(ctx, p.Session, p.Session, soundOffer(t, browser))
		if e != nil {
			t.Fatal(e)
		}
		if e = browser.PC.SetRemoteDescription(answer); e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { cc(); s.wg.Wait(); browser.Close(); engine.Close() })
	}
	svc[0].ScreenAudioStopCapture("obsolete-session")
	svc[0].ScreenAudioDisconnect("obsolete-session", "obsolete-actor")
	if svc[0].sound.ctx.Err() != nil {
		t.Fatal("obsolete session canceled current screen audio")
	}
	if svc[0].ScreenAudioCapture("obsolete-session", "obsolete-actor", 1, "device", 0, true) == nil {
		t.Fatal("obsolete session started capture")
	}
	svc[0].mu.Lock()
	svc[0].sound.capture = &loopback.Session{}
	svc[0].mu.Unlock()
	for i, s := range svc {
		s.mu.Lock()
		s.transport = soundTestTransport{svc[1-i]}
		s.mu.Unlock()
	}
	deadline := time.Now().Add(5 * time.Second)
	for len(svc[0].ScreenAudioState().Peers) == 0 || len(svc[1].ScreenAudioState().Peers) == 0 || browsers[0].Stats().State != "connected" || browsers[1].Stats().State != "connected" {
		if time.Now().After(deadline) {
			t.Fatal("independent stream did not connect")
		}
		time.Sleep(20 * time.Millisecond)
	}
	var sequence uint16
	send := func(n int) {
		for k := 0; k < n; k++ {
			sequence++
			_ = browsers[0].Write(&rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 111, SequenceNumber: sequence, Timestamp: uint32(sequence) * 960}, Payload: []byte{0xf8, 0xff, 0xfe}})
			time.Sleep(20 * time.Millisecond)
		}
	}
	send(50)
	if heard.Load() < 35 {
		t.Fatal("no authorized system audio playback bridge", heard.Load())
	}
	sv := svc[0].ScreenAudioState()
	if sv.Encoded == 0 || sv.Peers[b.ID].PacketsOut == 0 || svc[1].ScreenAudioState().Peers[a.ID].PacketsIn == 0 {
		t.Fatal("local encoding confused with network", sv)
	}
	svc[0].mu.Lock()
	svc[0].screen.peers[b.ID].wants = false
	svc[0].mu.Unlock()
	time.Sleep(150 * time.Millisecond)
	before := heard.Load()
	send(15)
	time.Sleep(100 * time.Millisecond)
	if heard.Load() != before {
		t.Fatal("non-viewer received audio")
	}
	svc[0].mu.Lock()
	svc[0].screen.peers[b.ID].wants = true
	svc[0].mu.Unlock()
	if seconds, _ := strconv.Atoi(os.Getenv("P2P_SCREEN_AUDIO_SOAK_SECONDS")); seconds > 0 {
		start := time.Now()
		before := heard.Load()
		send(seconds * 50)
		delta := heard.Load() - before
		if delta < uint64(seconds*40) {
			t.Fatal("system audio soak stalled", delta)
		}
		t.Logf("SYSTEM_AUDIO_SOAK generated Opus, mocked WASAPI/video admission, real local Pion and bounded stream: seconds=%.3f received=%d", time.Since(start).Seconds(), delta)
	}
	svc[0].mu.Lock()
	svc[0].current.authorizationInvalid = true
	svc[0].mu.Unlock()
	time.Sleep(150 * time.Millisecond)
	before = heard.Load()
	send(15)
	time.Sleep(100 * time.Millisecond)
	if heard.Load() != before {
		t.Fatal("audio continued after authorization invalidated")
	}
	if svc[0].screen == nil || svc[1].screen == nil {
		t.Fatal("audio failure stopped video")
	}
}
func TestScreenAudioConsentRequiredWithoutCapture(t *testing.T) {
	s := &Service{}
	if s.ScreenAudioCapture("", "", 1, "device", 0, false) == nil {
		t.Fatal("capture without consent")
	}
	if e := s.AcceptScreenAudio("x", Request{}, nil); e == nil {
		t.Fatal("unsigned audio stream")
	}
}
func TestVoiceFallbackAcceptsAuthenticatedHigherInitiator(t *testing.T) {
	f := hotfixRoom(t)
	a, b := f.people[1], f.people[2]
	if a.ID > b.ID {
		a, b = b, a
	}
	f.join(t, a)
	lease := f.join(t, b)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &call{ctx: ctx, cancel: cancel, group: f.state.ID, channel: "voz", session: a.Session, lease: lease, peers: map[string]*roomPeer{b.ID: {participant: b}}}
	s := &Service{id: a.ID, current: c, options: Options{State: func(string) (community.State, error) { return f.state, nil }}}
	left, right := net.Pipe()
	defer right.Close()
	r := Request{Method: "media", Community: f.state.ID, Channel: "voz", Participant: b, Session: a.Session, Lease: &lease, Nonce: community.RandomID()}
	if e := s.AcceptMedia(b.ID, r, left); e != nil {
		left.Close()
		t.Fatal("valid reverse path blocked", e)
	}
	defer c.peers[b.ID].stream.Close()
	if c.peers[b.ID].initiator != b.ID {
		t.Fatal("initiator association lost")
	}
	if s.View().Participants == nil {
		t.Fatal("roster lost")
	}
}

func TestOldCaptureRevisionAndBridgeActorCannotStartOrStopNewSource(t *testing.T) {
	f := hotfixRoom(t)
	p := f.people[1]
	lease := f.join(t, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &call{session: p.Session, lease: lease}
	sc := &screenCall{parent: c, validatedUntil: time.Now().Add(time.Second), publishing: true}
	a := &screenSound{actor: "01234567890123456789012345678901", parent: sc, ctx: ctx, cancel: cancel, captureRevision: 5}
	s := &Service{current: c, screen: sc, sound: a}
	if s.ScreenAudioCapture(p.Session, a.actor, 4, "device", 0, true) == nil {
		t.Fatal("stale capture intent accepted")
	}
	a.capture = &loopback.Session{}
	s.ScreenAudioStopCaptureForActor(p.Session, "old-actor", 99)
	if a.capture == nil {
		t.Fatal("old bridge stopped current capture")
	}
	s.ScreenAudioStopCaptureForActor(p.Session, a.actor, 6)
	if a.capture != nil {
		t.Fatal("valid stop did not release capture")
	}
	if s.ScreenAudioCapture(p.Session, a.actor, 5, "device", 0, true) == nil {
		t.Fatal("late start after stop accepted")
	}
	s.ScreenAudioDisconnect(p.Session, "old-actor")
	if ctx.Err() != nil {
		t.Fatal("old bridge canceled new session")
	}
}
