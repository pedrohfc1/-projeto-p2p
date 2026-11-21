package desktop

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4"
	"os"
	"os/exec"
	"path/filepath"
	"projetop2p/internal/community"
	"projetop2p/internal/localdb"
	"projetop2p/internal/loopback"
	"projetop2p/internal/voice"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type screenBrowser struct {
	receivedAt []time.Time
	pc         *webrtc.PeerConnection
	track      *webrtc.TrackLocalStaticRTP
	mu         sync.Mutex
	frames     [][]byte
	seq        uint16
	timestamp  uint32
	fps        int
}

func newScreenBrowser(t *testing.T) *screenBrowser {
	t.Helper()
	settings := webrtc.SettingEngine{}
	settings.SetIncludeLoopbackCandidate(true)
	settings.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4})
	pc, e := webrtc.NewAPI(webrtc.WithSettingEngine(settings)).NewPeerConnection(webrtc.Configuration{})
	if e != nil {
		t.Fatal(e)
	}
	track, e := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}, "screen", "generated-screen")
	if e != nil {
		t.Fatal(e)
	}
	sender, e := pc.AddTrack(track)
	if e != nil {
		t.Fatal(e)
	}
	go func() {
		for {
			if _, _, e := sender.ReadRTCP(); e != nil {
				return
			}
		}
	}()
	b := &screenBrowser{pc: pc, track: track}
	pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		var frame []byte
		var ts uint32
		var expected uint16
		waiting := true
		for {
			p, _, e := track.ReadRTP()
			if e != nil {
				return
			}
			var vp codecs.VP8Packet
			if _, e = vp.Unmarshal(p.Payload); e != nil {
				continue
			}
			if vp.S == 1 && vp.PID == 0 {
				if len(vp.Payload) == 0 {
					continue
				}
				if waiting && vp.Payload[0]&1 != 0 {
					continue
				}
				frame = nil
				ts = p.Timestamp
				expected = p.SequenceNumber
				waiting = false
			}
			if waiting || p.Timestamp != ts || p.SequenceNumber != expected {
				waiting = true
				continue
			}
			expected++
			frame = append(frame, vp.Payload...)
			if p.Marker {
				b.mu.Lock()
				b.frames = append(b.frames, append([]byte(nil), frame...))
				b.receivedAt = append(b.receivedAt, time.Now())
				b.mu.Unlock()
				frame = nil
			}
		}
	})
	return b
}
func (b *screenBrowser) offer(t *testing.T) webrtc.SessionDescription {
	offer, e := b.pc.CreateOffer(nil)
	if e != nil {
		t.Fatal(e)
	}
	ready := webrtc.GatheringCompletePromise(b.pc)
	if e = b.pc.SetLocalDescription(offer); e != nil {
		t.Fatal(e)
	}
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("video ICE")
	}
	return *b.pc.LocalDescription()
}
func (b *screenBrowser) send(t *testing.T, frame []byte) {
	t.Helper()
	chunks := (&codecs.VP8Payloader{EnablePictureID: true}).Payload(1200, frame)
	fps := b.fps
	if fps == 0 {
		fps = 30
	}
	b.timestamp += uint32(90000 / fps)
	for n, payload := range chunks {
		b.seq++
		if e := b.track.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 96, SequenceNumber: b.seq, Timestamp: b.timestamp, Marker: n == len(chunks)-1}, Payload: payload}); e != nil {
			t.Fatal(e)
		}
	}
}
func ivfFrames(t *testing.T, path string) ([]byte, [][]byte) {
	raw, e := os.ReadFile(path)
	if e != nil || len(raw) < 32 || string(raw[:4]) != "DKIF" || string(raw[8:12]) != "VP80" {
		t.Fatal("generated VP8 fixture", e)
	}
	header := append([]byte(nil), raw[:32]...)
	raw = raw[32:]
	frames := [][]byte{}
	for len(raw) >= 12 {
		n := int(binary.LittleEndian.Uint32(raw))
		raw = raw[12:]
		if n < 1 || n > len(raw) {
			t.Fatal("invalid IVF fixture")
		}
		frames = append(frames, append([]byte(nil), raw[:n]...))
		raw = raw[n:]
	}
	return header, frames
}
func TestRealScreenAndVoiceOverIntegratedTransport(t *testing.T) {
	testRealScreenAndVoice(t, "easytier")
}

func TestScreenAndVoiceOverLoopback(t *testing.T) {
	if os.Getenv("P2P_SCREEN_LOOPBACK") != "1" {
		t.Skip("optional production libp2p on loopback, with generated fixtures")
	}
	testRealScreenAndVoice(t, "libp2p-loopback")
}

func testRealScreenAndVoice(t *testing.T, transport string) {
	t.Helper()
	integrated := transport == "easytier"
	helper, fixture := os.Getenv("P2P_ETCAT_TEST_DIR"), os.Getenv("P2P_SCREEN_IVF")
	if fixture == "" || (integrated && helper == "") {
		t.Skip("requires audited helper and generated VP8 fixture; no physical capture")
	}
	captureMediaDiagnostics(t)
	header, frames := ivfFrames(t, fixture)
	fps := 30
	if value, _ := strconv.Atoi(os.Getenv("P2P_SCREEN_TEST_FPS")); value == 60 {
		fps = 60
	}
	if len(frames) < 15 {
		t.Fatal("VP8 fixture needs at least 15 complete frames")
	}
	if binary.LittleEndian.Uint32(header[20:24]) == 0 || binary.LittleEndian.Uint32(header[16:20])/binary.LittleEndian.Uint32(header[20:24]) != uint32(fps) {
		t.Fatal("fixture frame rate differs from requested transport load; no fabricated FPS")
	}
	frameInterval := time.Second / time.Duration(fps)
	bitrate := 3000000
	if value, _ := strconv.Atoi(os.Getenv("P2P_SCREEN_TEST_BITRATE")); value >= 256000 && value <= 12000000 {
		bitrate = value
	}
	root, _ := filepath.Abs("../../data")
	os.MkdirAll(root, 0700)
	dir, e := os.MkdirTemp(root, "screen-voice-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(dir)
	systemAudio := os.Getenv("P2P_SCREEN_AUDIO_INTEGRATION") == "1"
	var systemHeard atomic.Uint64
	open := func(name string) *Core {
		var c *Core
		var e error
		dataDir := filepath.Join(dir, name)
		if integrated {
			c, e = Open(context.Background(), dataDir)
		} else {
			c, e = OpenWithProvider(context.Background(), dataDir, screenLoopbackInterface)
		}
		if e != nil {
			t.Fatal(e)
		}
		if systemAudio {
			c.mu.Lock()
			c.captureFactory = func(context.Context, string, string, int) (voice.CaptureSource, error) {
				return &fixtureSystemCapture{}, nil
			}
			c.mu.Unlock()
		}
		settings := localdb.Settings{Name: name, Mode: "integrated", Port: integratedPort(t), IntegratedBinaryDir: helper}
		if !integrated {
			settings.Mode, settings.InterfaceID = "interface", "media-qa-loopback"
		}
		if e = c.SaveSettings(settings); e != nil {
			c.Close()
			t.Fatal(e)
		}
		return c
	}
	a, b := open("a"), open("b")
	defer a.Close()
	defer b.Close()
	group, e := a.CreateCommunity("Screen QA", "generated video, real transport", "")
	if e != nil {
		t.Fatal(e)
	}
	ticket, e := a.CommunityInvite(group)
	if e != nil {
		t.Fatal(e)
	}
	r, e := b.JoinCommunity(ticket)
	if e != nil || r.Status != "participante" {
		t.Fatal(r, e)
	}
	if e = a.SyncCommunity(group); e != nil {
		t.Fatal(e)
	}
	var heard [2]atomic.Uint64
	engines := make([]*voice.Engine, 2)
	audio := make([]*voice.MediaPeer, 2)
	for i, c := range []*Core{a, b} {
		engines[i], e = voice.NewEngine("127.0.0.1")
		if e != nil {
			t.Fatal(e)
		}
		defer engines[i].Close()
		index := i
		audio[i], e = engines[i].Peer(func(*rtp.Packet) { heard[index].Add(1) })
		if e != nil {
			t.Fatal(e)
		}
		defer audio[i].Close()
		answer, e := c.VoiceJoin(group, "voz", browserOffer(t, audio[i]))
		if e != nil {
			t.Fatal(e)
		}
		if e = audio[i].PC.SetRemoteDescription(answer); e != nil {
			t.Fatal(e)
		}
	}
	deadline := time.Now().Add(60 * time.Second)
	for a.VoiceState().Media.State != "connected" || b.VoiceState().Media.State != "connected" {
		if time.Now().After(deadline) {
			t.Fatal("audio connect timeout")
		}
		time.Sleep(100 * time.Millisecond)
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		tick := time.NewTicker(20 * time.Millisecond)
		defer tick.Stop()
		var n uint16
		for {
			select {
			case <-done:
				return
			case <-tick.C:
				n++
				for _, p := range audio {
					_ = p.Write(&rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 111, SequenceNumber: n, Timestamp: uint32(n) * 960}, Payload: []byte{0xf8, 0xff, 0xfe}})
				}
			}
		}
	}()
	_, _ = a.State("")
	_, _ = b.State("")
	time.Sleep(time.Second)
	voiceOnlyA, _ := a.State("")
	voiceOnlyB, _ := b.State("")
	videoA, videoB := newScreenBrowser(t), newScreenBrowser(t)
	videoA.fps = fps
	videoB.fps = fps
	defer videoA.pc.Close()
	defer videoB.pc.Close()
	for i, c := range []*Core{a, b} {
		browser := []*screenBrowser{videoA, videoB}[i]
		answer, e := c.ScreenConnect(browser.offer(t))
		if e != nil {
			t.Fatal(e)
		}
		if e = browser.pc.SetRemoteDescription(answer); e != nil {
			t.Fatal(e)
		}
	}
	heart := make(chan struct{})
	defer close(heart)
	var heartbeat atomic.Bool
	heartbeat.Store(true)
	go func() {
		tick := time.NewTicker(400 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-heart:
				return
			case <-tick.C:
				if heartbeat.Load() {
					a.ScreenHeartbeat()
				}
				b.ScreenHeartbeat()
			}
		}
	}()
	if systemAudio {
		for i, c := range []*Core{a, b} {
			eng, e := voice.NewEngine("127.0.0.1")
			if e != nil {
				t.Fatal(e)
			}
			defer eng.Close()
			index := i
			client, e := eng.Peer(func(*rtp.Packet) {
				if index == 1 {
					systemHeard.Add(1)
				}
			})
			if e != nil {
				t.Fatal(e)
			}
			defer client.Close()
			answer, e := c.ScreenAudioConnect(c.VoiceState().Session, c.VoiceState().Session, browserOffer(t, client))
			if e != nil {
				t.Fatal(e)
			}
			if e = client.PC.SetRemoteDescription(answer); e != nil {
				t.Fatal(e)
			}
			if i == 0 {
				go func() {
					tick := time.NewTicker(20 * time.Millisecond)
					defer tick.Stop()
					var n uint16
					for {
						select {
						case <-done:
							return
						case <-tick.C:
							n++
							_ = client.Write(&rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 111, SequenceNumber: n, Timestamp: uint32(n) * 960}, Payload: []byte{0xf8, 0xff, 0xfe}})
						}
					}
				}()
			}
		}
	}
	deadline = time.Now().Add(20 * time.Second)
	for {
		sa, sb := a.ScreenState(), b.ScreenState()
		if len(sa.Peers) == 1 && len(sb.Peers) == 1 && sa.Peers[0].Connected && sb.Peers[0].Connected && videoA.pc.ConnectionState() == webrtc.PeerConnectionStateConnected && videoB.pc.ConnectionState() == webrtc.PeerConnectionStateConnected {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("video session not connected", sa, sb)
		}
		time.Sleep(100 * time.Millisecond)
	}
	// Preview packets are generated locally but cannot leave while publishing is false.
	for n := 0; n < 15; n++ {
		videoA.send(t, frames[n])
		time.Sleep(33 * time.Millisecond)
	}
	if a.ScreenState().FramesOut != 0 || b.ScreenState().FramesIn != 0 {
		t.Fatal("preview leaked before explicit publish")
	}
	if e = b.ScreenWatch(a.node.SelfID()); e != nil {
		t.Fatal(e)
	}
	if e = a.ScreenPublish(true, bitrate, 5); e != nil {
		t.Fatal(e)
	}
	if systemAudio {
		if e = a.ScreenAudioCapture(a.VoiceState().Session, a.VoiceState().Session, 1, "process", 123, true); e != nil {
			t.Fatal(e)
		}
	}
	time.Sleep(time.Second)
	_, _ = a.State("")
	_, _ = b.State("")
	beforeA, beforeB := heard[0].Load(), heard[1].Load()
	started := time.Now()
	sentAt := map[[32]byte]time.Time{}
	pacing := time.NewTicker(frameInterval)
	for n := 0; n < len(frames); n++ {
		<-pacing.C
		sentAt[sha256.Sum256(frames[n])] = time.Now()
		videoA.send(t, frames[n])
	}
	pacing.Stop()
	time.Sleep(300 * time.Millisecond)
	videoB.mu.Lock()
	received := append([][]byte(nil), videoB.frames...)
	receivedAt := append([]time.Time(nil), videoB.receivedAt...)
	videoB.mu.Unlock()
	if len(received) < len(frames)*6/10 {
		t.Fatal("too few complete decoded video frames", len(received), len(frames), a.ScreenState(), b.ScreenState())
	}
	if heard[0].Load()-beforeA < 150 || heard[1].Load()-beforeB < 150 {
		t.Fatal("video disrupted bidirectional audio", heard[0].Load()-beforeA, heard[1].Load()-beforeB)
	}
	if _, e = b.CommunityAction(group, community.Text, community.Body{ChannelID: "geral", Text: "chat while sharing"}); e != nil {
		t.Fatal(e)
	}
	encoded := append([]byte(nil), header...)
	binary.LittleEndian.PutUint32(encoded[24:], uint32(len(received)))
	for i, frame := range received {
		encoded = binary.LittleEndian.AppendUint32(encoded, uint32(len(frame)))
		encoded = binary.LittleEndian.AppendUint64(encoded, uint64(i))
		encoded = append(encoded, frame...)
	}
	file := filepath.Join(dir, "received-generated.ivf")
	if e = os.WriteFile(file, encoded, 0600); e != nil {
		t.Fatal(e)
	}
	output, e := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-i", file, "-f", "null", "-").CombinedOutput()
	if e != nil || len(output) != 0 {
		t.Fatal("received VP8 failed real decoding", e, string(output))
	}
	var delayTotal, delayMax time.Duration
	var delayCount int
	for i, f := range received {
		if at, ok := sentAt[sha256.Sum256(f)]; ok {
			d := receivedAt[i].Sub(at)
			delayTotal += d
			delayCount++
			if d > delayMax {
				delayMax = d
			}
		}
	}
	steadyFPS := float64(0)
	if len(receivedAt) > 1 {
		steadyFPS = float64(len(receivedAt)-1) / receivedAt[len(receivedAt)-1].Sub(receivedAt[0]).Seconds()
	}
	var delayMean time.Duration
	if delayCount > 0 {
		delayMean = delayTotal / time.Duration(delayCount)
	}
	va, vb := a.ScreenState(), b.ScreenState()
	stateA, _ := a.State("")
	stateB, _ := b.State("")
	metrics, _ := json.Marshal(map[string]any{"scope": "same Go test process, two production cores, generated VP8 and Opus; excludes WebView/helper and physical capture", "transport": transport, "seconds": time.Since(started).Seconds(), "deliveryMeanMs": float64(delayMean) / 1e6, "deliveryMaxMs": float64(delayMax) / 1e6, "latencyScope": "encoded RTP send to complete received VP8; excludes physical capture, browser encoder and display", "videoFrames": len(received), "targetSendFPS": fps, "steadyCompleteFPS": steadyFPS, "receivedCompleteFPS": float64(len(received)) / time.Since(started).Seconds(), "fixtureWidth": binary.LittleEndian.Uint16(header[12:14]), "fixtureHeight": binary.LittleEndian.Uint16(header[14:16]), "audioFrames": []uint64{heard[0].Load() - beforeA, heard[1].Load() - beforeB}, "screenA": va, "screenB": vb, "voiceOnlyA": voiceOnlyA.Metrics, "voiceOnlyB": voiceOnlyB.Metrics, "backendA": stateA.Metrics, "backendB": stateB.Metrics})
	t.Log("SCREEN_METRICS", string(metrics))
	if systemAudio && systemHeard.Load() < 100 {
		t.Fatal("screen audio did not accompany video/voice", systemHeard.Load(), a.ScreenAudioState(), b.ScreenAudioState())
	}
	if seconds, err := strconv.Atoi(os.Getenv("P2P_SCREEN_SOAK_SECONDS")); err == nil && seconds != 0 {
		runScreenSoak(t, a, b, videoA, videoB, &heard, &systemHeard, systemAudio, header, frames, fps, seconds, transport)
	}
	runScreenResolutionSwitch(t, videoA, videoB, header, &heard)
	// Independent publishers can coexist; each receiver explicitly selects its source.
	// Start the reverse browser track locally before sending its first published
	// keyframe. Otherwise the synthetic burst competes with Pion's OnTrack setup.
	for _, frame := range frames[:15] {
		videoB.send(t, frame)
		time.Sleep(frameInterval)
	}
	if b.ScreenState().FramesOut != 0 || a.ScreenState().FramesIn != 0 {
		t.Fatal("reverse preview leaked before explicit publish")
	}
	if e = b.ScreenPublish(true, bitrate, 5); e != nil {
		t.Fatal(e)
	}
	if e = a.ScreenWatch(b.node.SelfID()); e != nil {
		t.Fatal(e)
	}
	deadline = time.Now().Add(5 * time.Second)
	for {
		sa, sb := a.ScreenState(), b.ScreenState()
		if len(sa.Peers) == 1 && sa.Peers[0].Sharing && sa.Peers[0].Accepted && sb.Viewers > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("reverse subscription did not become authorized", sa, sb)
		}
		time.Sleep(50 * time.Millisecond)
	}
	videoA.mu.Lock()
	previousReverse := len(videoA.frames)
	videoA.mu.Unlock()
	reversePacing := time.NewTicker(frameInterval)
	for n := 0; n < fps*3; n++ {
		<-reversePacing.C
		videoA.send(t, frames[n%len(frames)])
		videoB.send(t, frames[n%len(frames)])
	}
	reversePacing.Stop()
	time.Sleep(300 * time.Millisecond)
	videoA.mu.Lock()
	reverseDelivered := len(videoA.frames) - previousReverse
	videoA.mu.Unlock()
	if reverseDelivered < 30 || !a.ScreenState().Publishing || !b.ScreenState().Publishing {
		t.Fatal("simultaneous publishers failed", reverseDelivered, a.ScreenState(), b.ScreenState())
	}
	if e = a.ScreenWatch(""); e != nil {
		t.Fatal(e)
	}
	if e = b.ScreenPublish(false, bitrate, 5); e != nil {
		t.Fatal(e)
	}
	t.Log("simultaneous independent publishers and explicit subscription passed; generated media, same PC")
	// A valid community membership does not authorize a screen subscription without a current voice session.
	outsider := open("not-in-call")
	defer outsider.Close()
	inviteText, _ := a.CommunityInvite(group)
	receipt, err := outsider.JoinCommunity(inviteText)
	if err != nil || receipt.Status != "participante" {
		t.Fatal(receipt, err)
	}
	ownerInvite, _ := a.Invite()
	if err = outsider.AddContact(ownerInvite); err != nil {
		t.Fatal(err)
	}
	proof := a.voice.Handle(context.Background(), a.node.SelfID(), voice.Request{Method: "renew", Community: group, Channel: "voz", Participant: voice.Participant{ID: a.node.SelfID(), Session: a.VoiceState().Session, Invite: ownerInvite, UDPPort: a.voice.Port()}})
	if proof.Lease == nil {
		t.Fatal("owner omitted signed lease")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	unauthorized, e := outsider.node.OpenScreenMedia(ctx, a.node.SelfID(), a.node.Addresses(), voice.Request{Method: "screen", Community: group, Channel: "voz", Participant: voice.Participant{ID: outsider.node.SelfID(), Session: community.RandomID()}, Session: a.VoiceState().Session, Lease: proof.Lease, Nonce: community.RandomID()})
	cancel()
	if e == nil {
		unauthorized.Close()
		t.Fatal("member without current call session received screen transport")
	}
	// Closing the encrypted peer connection must reconnect both protocols without a room leave.
	systemBeforeReset := systemHeard.Load()
	screenBeforeResetA := a.ScreenState()
	screenBeforeResetB := b.ScreenState()
	if len(screenBeforeResetA.Peers) != 1 || len(screenBeforeResetB.Peers) != 1 {
		t.Fatal("expected one stream to recover", screenBeforeResetA, screenBeforeResetB)
	}
	generationA := screenBeforeResetA.Peers[0].PipeGeneration
	generationB := screenBeforeResetB.Peers[0].PipeGeneration
	_ = a.node.Host.Network().ClosePeer(b.node.Host.ID())
	deadline = time.Now().Add(35 * time.Second)
	for {
		sa, sb := a.ScreenState(), b.ScreenState()
		fresh := len(sa.Peers) == 1 && len(sb.Peers) == 1 &&
			sa.Peers[0].PipeGeneration > generationA && sb.Peers[0].PipeGeneration > generationB
		if fresh && sa.Peers[0].Connected && sa.Peers[0].StatusKnown && sa.Viewers > 0 && sb.Peers[0].Connected && sb.Peers[0].Accepted && a.VoiceState().Media.State == "connected" && b.VoiceState().Media.State == "connected" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("screen/audio reconnect stalled", sa, sb, a.VoiceState())
		}
		time.Sleep(100 * time.Millisecond)
	}
	videoB.mu.Lock()
	oldCount := len(videoB.frames)
	videoB.mu.Unlock()
	for n := 0; n < fps*2; n++ {
		videoA.send(t, frames[n%len(frames)])
		time.Sleep(frameInterval)
	}
	time.Sleep(300 * time.Millisecond)
	videoB.mu.Lock()
	newCount := len(videoB.frames)
	videoB.mu.Unlock()
	if newCount-oldCount < 30 {
		t.Fatal("video did not recover after encrypted stream reset", newCount-oldCount, a.ScreenState(), b.ScreenState())
	}
	if systemAudio && systemHeard.Load() <= systemBeforeReset {
		t.Fatal("screen audio did not recover with video/voice")
	}
	heartbeat.Store(false)
	time.Sleep(3500 * time.Millisecond)
	if a.ScreenState().Publishing {
		t.Fatal("visibility lease did not stop hidden publisher")
	}
	out := a.ScreenState().FramesOut
	for n := 0; n < 15; n++ {
		videoA.send(t, frames[n])
		time.Sleep(frameInterval)
	}
	if a.ScreenState().FramesOut != out {
		t.Fatal("video transmitted after hidden stop")
	}
	a.ScreenStop()
	b.ScreenStop()
	if a.ScreenState().Active || b.ScreenState().Active {
		t.Fatal("video resources not released")
	}
	if !a.VoiceState().Joined || !b.VoiceState().Joined {
		t.Fatal("stopping screen expelled audio participants")
	}
	a.VoiceLeave()
	b.VoiceLeave()
	runScreenRejoins(t, [2]*Core{a, b}, group)
	t.Log("PASS generated VP8 decoded after encrypted libp2p transport, preview isolation, simultaneous Opus/chat, screen+voice reconnect, visibility expiry and cleanup; no physical screen/microphone test")
}

// Fake device only; real authorization, Opus bridge, encrypted network and video stay in use.
type fixtureSystemCapture struct{ closed atomic.Bool }

func (s *fixtureSystemCapture) Read() (loopback.Frame, error) { return loopback.Frame{}, nil }
func (s *fixtureSystemCapture) Alive() bool                   { return !s.closed.Load() }
func (s *fixtureSystemCapture) Close()                        { s.closed.Store(true) }
