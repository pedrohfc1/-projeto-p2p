package desktop

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"projetop2p/internal/adapters"
	"projetop2p/internal/telemetry"
	"projetop2p/internal/voice"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

func screenLoopbackInterface() ([]adapters.Info, error) {
	return []adapters.Info{{ID: "media-qa-loopback", Name: "Loopback QA", Up: true, IPv4: []string{"127.0.0.1"}}}, nil
}

type screenSoakInterval struct {
	SinceSeconds float64          `json:"sinceSeconds"`
	RTTMs        *float64         `json:"rttMs"`
	Path         string           `json:"libp2pPath"`
	MediaOpens   uint64           `json:"mediaOpens"`
	StreamResets uint64           `json:"streamResets"`
	GoResources  telemetry.Sample `json:"goResources"`
}

type screenSoakReport struct {
	Scope                string               `json:"scope"`
	Transport            string               `json:"transport"`
	FixtureSHA256        string               `json:"fixtureSHA256"`
	FixtureWidth         uint16               `json:"fixtureWidth"`
	FixtureHeight        uint16               `json:"fixtureHeight"`
	TargetFPS            int                  `json:"targetFps"`
	Seconds              float64              `json:"seconds"`
	FramesSent           int                  `json:"framesSent"`
	FramesComplete       int                  `json:"framesComplete"`
	CompleteFrameFPS     float64              `json:"completeFrameFps"`
	AudioPackets         [2]uint64            `json:"audioPackets"`
	AudioStagnantSeconds [2]int               `json:"audioStagnantSeconds"`
	VideoDropFrames      uint64               `json:"reportedVideoDrops"`
	VideoFreezes         int                  `json:"gapsOver500msBetweenCompleteFrames"`
	MaxVideoGapMs        float64              `json:"maxCompleteFrameGapMs"`
	PayloadMbps          float64              `json:"videoPayloadMbps"`
	RTTMeanMs            *float64             `json:"rttMeanMs"`
	RTTP95Ms             *float64             `json:"rttP95Ms"`
	RSSBeforeBytes       *uint64              `json:"rssBeforeBytes"`
	RSSAfterBytes        *uint64              `json:"rssAfterBytes"`
	RSSPeakBytes         *uint64              `json:"rssPeakBytes"`
	GoCPUCoreMeanPercent float64              `json:"goCpuCoreMeanPercent"`
	HeapBeforeBytes      uint64               `json:"heapBeforeBytes"`
	HeapAfterBytes       uint64               `json:"heapAfterBytes"`
	PeakLiveGoroutines   int                  `json:"peakLiveGoroutines"`
	Intervals            []screenSoakInterval `json:"intervals"`
}

func screenSoakProbe(node *Core, began time.Time, res chan<- screenSoakInterval, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	var sampler telemetry.Sampler
	for {
		select {
		case <-stop:
			return
		case at := <-tick.C:
			diagnostic := node.VoiceDiagnosticSnapshot()
			point := screenSoakInterval{SinceSeconds: at.Sub(began).Seconds(), GoResources: sampler.Read(0, 0)}
			if len(diagnostic.Peers) == 1 {
				peer := diagnostic.Peers[0]
				point.RTTMs = peer.RTTMs
				point.Path = peer.Path
				point.MediaOpens = peer.OpenAttempts
				point.StreamResets = peer.ResetReports
			}
			select {
			case res <- point:
			case <-stop:
				return
			}
		}
	}
}

func runScreenSoak(t *testing.T, a, b *Core, videoA, videoB *screenBrowser, heard *[2]atomic.Uint64, systemHeard *atomic.Uint64, systemAudio bool, header []byte, frames [][]byte, fps, seconds int, transport string) {
	t.Helper()
	if seconds < 1 || seconds > 1800 {
		t.Fatal("screen soak duration must be between 1 and 1800 seconds")
	}
	fixture := sha256.New()
	fixture.Write(header)
	for _, frame := range frames {
		fixture.Write(binary.LittleEndian.AppendUint32(nil, uint32(len(frame))))
		fixture.Write(frame)
	}
	report := screenSoakReport{
		Scope:     "Two local production cores with generated VP8 and Opus. Complete reassembled frames, no camera, physical capture, browser encoding/rendering or remote VPN claim.",
		Transport: transport, FixtureSHA256: hex.EncodeToString(fixture.Sum(nil)),
		FixtureWidth: binary.LittleEndian.Uint16(header[12:14]), FixtureHeight: binary.LittleEndian.Uint16(header[14:16]),
		TargetFPS: fps,
	}
	drain := func(began time.Time, last *time.Time) {
		videoB.mu.Lock()
		defer videoB.mu.Unlock()
		for _, received := range videoB.receivedAt {
			if received.Before(began) {
				continue
			}
			if !last.IsZero() {
				gap := received.Sub(*last)
				if gap > 500*time.Millisecond {
					report.VideoFreezes++
				}
				report.MaxVideoGapMs = max(report.MaxVideoGapMs, float64(gap)/float64(time.Millisecond))
			}
			*last = received
			report.FramesComplete++
		}
		videoB.frames = nil
		videoB.receivedAt = nil
	}

	var sampling telemetry.Sampler
	initialResources := sampling.Read(0, 0)
	initialScreen := a.ScreenState()
	beforeA, beforeB := heard[0].Load(), heard[1].Load()
	beforeSystem := systemHeard.Load()
	previousAudio := [2]uint64{beforeA, beforeB}
	began := time.Now()
	stopAt := began.Add(time.Duration(seconds) * time.Second)
	var lastFrame time.Time
	points := make(chan screenSoakInterval, 1)
	stop, done := make(chan struct{}), make(chan struct{})
	go screenSoakProbe(a, began, points, stop, done)
	defer func() { close(stop); <-done }()
	tick := time.NewTicker(time.Second / time.Duration(fps))
	defer tick.Stop()
	check := time.NewTicker(time.Second)
	defer check.Stop()
	until := time.NewTimer(time.Until(stopAt))
	defer until.Stop()
	running := true
	for running {
		select {
		case <-until.C:
			running = false
		case <-tick.C:
			if time.Now().Before(stopAt) {
				videoA.send(t, frames[report.FramesSent%len(frames)])
				report.FramesSent++
			}
		case <-check.C:
			if !a.VoiceState().Joined || !b.VoiceState().Joined || !a.ScreenState().Publishing {
				t.Fatal("continuous call or publication interrupted")
			}
			for i := range heard {
				value := heard[i].Load()
				if value == previousAudio[i] {
					report.AudioStagnantSeconds[i]++
				}
				previousAudio[i] = value
			}
			drain(began, &lastFrame)
			stallStart := lastFrame
			if stallStart.IsZero() {
				stallStart = began
			}
			if time.Since(stallStart) >= 5*time.Second {
				t.Fatal("video did not deliver complete frames for five seconds with the generated source active")
			}
		case point := <-points:
			report.Intervals = append(report.Intervals, point)
		}
	}
	time.Sleep(300 * time.Millisecond)
	drain(began, &lastFrame)
	finalResources := sampling.Read(0, 0)
	finalScreen := a.ScreenState()
	report.Seconds = time.Since(began).Seconds()
	report.CompleteFrameFPS = float64(report.FramesComplete) / report.Seconds
	if finalScreen.BytesOut < initialScreen.BytesOut || finalScreen.Dropped < initialScreen.Dropped {
		t.Fatal("screen counters reset during continuous publication")
	}
	report.PayloadMbps = float64(finalScreen.BytesOut-initialScreen.BytesOut) * 8 / report.Seconds / 1e6
	report.VideoDropFrames = finalScreen.Dropped - initialScreen.Dropped
	report.AudioPackets = [2]uint64{heard[0].Load() - beforeA, heard[1].Load() - beforeB}
	report.RSSBeforeBytes, report.RSSAfterBytes = initialResources.RSSBytes, finalResources.RSSBytes
	report.HeapBeforeBytes, report.HeapAfterBytes = initialResources.HeapBytes, finalResources.HeapBytes
	report.GoCPUCoreMeanPercent = (finalResources.CPUSecs - initialResources.CPUSecs) / report.Seconds * 100
	report.PeakLiveGoroutines = runtime.NumGoroutine()
	var rtts []float64
	var mean float64
	resources := []telemetry.Sample{initialResources, finalResources}
	for _, point := range report.Intervals {
		resources = append(resources, point.GoResources)
		if point.RTTMs != nil {
			rtts = append(rtts, *point.RTTMs)
			mean += *point.RTTMs
		}
	}
	for _, value := range resources {
		if value.RSSBytes != nil && (report.RSSPeakBytes == nil || *value.RSSBytes > *report.RSSPeakBytes) {
			report.RSSPeakBytes = value.RSSBytes
		}
		report.PeakLiveGoroutines = max(report.PeakLiveGoroutines, value.Goroutines)
	}
	if len(rtts) > 0 {
		sort.Float64s(rtts)
		mean /= float64(len(rtts))
		report.RTTMeanMs = &mean
		report.RTTP95Ms = &rtts[(len(rtts)*95+99)/100-1]
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("SCREEN_SOAK_METRICS", string(raw))
	if report.AudioPackets[0] < uint64(seconds*40) || report.AudioPackets[1] < uint64(seconds*40) || report.FramesComplete < report.FramesSent*7/10 {
		t.Fatal("continuous media delivery below threshold", report.AudioPackets, report.FramesSent, report.FramesComplete)
	}
	if systemAudio && systemHeard.Load()-beforeSystem < uint64(seconds*40) {
		t.Fatal("synthetic system audio did not survive concurrent voice/video")
	}
}

// Reusing the already authenticated room exercises fresh synthetic RTC bridges
// over the same enrolled identity and ensures leave releases old media.
func runScreenRejoins(t *testing.T, cores [2]*Core, room string) {
	t.Helper()
	raw := os.Getenv("P2P_SCREEN_REJOIN_CYCLES")
	if raw == "" {
		return
	}
	cycles, err := strconv.Atoi(raw)
	if err != nil || cycles < 0 || cycles > 10 {
		t.Fatal("set P2P_SCREEN_REJOIN_CYCLES to an integer from 0 to 10")
	}
	if cycles == 0 {
		return
	}
	before := runtime.NumGoroutine()
	var previous [2]string
	for cycle := 0; cycle < cycles; cycle++ {
		func() {
			for i, core := range cores {
				engine, err := voice.NewEngine("127.0.0.1")
				if err != nil {
					t.Fatal(err)
				}
				defer engine.Close()
				playout, err := engine.Peer(func(*rtp.Packet) {})
				if err != nil {
					t.Fatal(err)
				}
				defer playout.Close()
				answer, err := core.VoiceJoin(room, "voz", browserOffer(t, playout))
				if err != nil {
					t.Fatal("rejoin voice", cycle, i, err)
				}
				defer core.VoiceLeave()
				if err = playout.PC.SetRemoteDescription(answer); err != nil {
					t.Fatal(err)
				}
				view := core.VoiceState()
				if !view.Joined || view.Session == "" || view.Session == previous[i] {
					t.Fatal("rejoin did not establish a fresh authenticated session", cycle, i)
				}
				previous[i] = view.Session
				browser := newScreenBrowser(t)
				defer browser.pc.Close()
				answer, err = core.ScreenConnect(browser.offer(t))
				if err != nil {
					t.Fatal("rejoin screen", cycle, i, err)
				}
				if err = browser.pc.SetRemoteDescription(answer); err != nil {
					t.Fatal(err)
				}
				if !core.ScreenState().Active {
					t.Fatal("rejoin did not create an active screen bridge", cycle, i)
				}
				core.ScreenStop()
				if core.ScreenState().Active || !core.VoiceState().Joined {
					t.Fatal("closing player bridge affected the voice session", cycle, i)
				}
				// Wait for browser transports to finish negotiation before
				// introducing the other core and verify they were real offers.
				if playout.PC.SignalingState() != webrtc.SignalingStateStable {
					t.Fatal("voice answer did not complete signaling", cycle, i)
				}
			}
		}()
		for i, core := range cores {
			if core.VoiceState().Joined || core.ScreenState().Active {
				t.Fatal("leaving the room retained live media", cycle, i)
			}
		}
	}
	t.Logf("SCREEN_REJOIN_CYCLES %d, go goroutines before=%d after=%d in shared process", cycles, before, runtime.NumGoroutine())
}

// The same publisher and viewer keep their existing peer connections. Only
// generated VP8 keyframe dimensions change, and voice must advance throughout.
func runScreenResolutionSwitch(t *testing.T, videoA, videoB *screenBrowser, previousHeader []byte, heard *[2]atomic.Uint64) {
	t.Helper()
	fixturePath := os.Getenv("P2P_SCREEN_SWITCH_IVF")
	if fixturePath == "" {
		return
	}
	header, frames := ivfFrames(t, fixturePath)
	width := binary.LittleEndian.Uint16(header[12:14])
	height := binary.LittleEndian.Uint16(header[14:16])
	if width == binary.LittleEndian.Uint16(previousHeader[12:14]) && height == binary.LittleEndian.Uint16(previousHeader[14:16]) {
		t.Fatal("resolution switch fixture must use different dimensions")
	}
	duration := binary.LittleEndian.Uint32(header[20:24])
	if duration == 0 {
		t.Fatal("resolution switch fixture has invalid time base")
	}
	fps := int(binary.LittleEndian.Uint32(header[16:20]) / duration)
	if (fps != 30 && fps != 60) || len(frames) < fps*2 || len(frames[0]) < 10 || frames[0][0]&1 != 0 {
		t.Fatal("resolution switch fixture must begin with VP8 keyframe and contain at least two seconds")
	}
	beforeAudio := [2]uint64{heard[0].Load(), heard[1].Load()}
	videoB.mu.Lock()
	frameOffset := len(videoB.frames)
	videoB.mu.Unlock()
	originalFPS := videoA.fps
	videoA.fps = fps
	defer func() { videoA.fps = originalFPS }()
	tick := time.NewTicker(time.Second / time.Duration(fps))
	defer tick.Stop()
	for i := range frames[:fps*2] {
		<-tick.C
		videoA.send(t, frames[i])
	}
	time.Sleep(300 * time.Millisecond)
	videoB.mu.Lock()
	delivered := len(videoB.frames) - frameOffset
	dimensionsFound := false
	for _, frame := range videoB.frames[frameOffset:] {
		if len(frame) >= 10 && frame[0]&1 == 0 && frame[3] == 0x9d && frame[4] == 0x01 && frame[5] == 0x2a &&
			binary.LittleEndian.Uint16(frame[6:8])&0x3fff == width &&
			binary.LittleEndian.Uint16(frame[8:10])&0x3fff == height {
			dimensionsFound = true
			break
		}
	}
	videoB.mu.Unlock()
	audible := [2]uint64{heard[0].Load() - beforeAudio[0], heard[1].Load() - beforeAudio[1]}
	if !dimensionsFound || delivered < fps || audible[0] < 70 || audible[1] < 70 {
		t.Fatal("resolution change interrupted the media test", dimensionsFound, delivered, audible)
	}
	t.Logf("SCREEN_RESOLUTION_SWITCH %dx%d %d/%d complete frames, synthetic audio packets RX=%v", width, height, delivered, fps*2, audible)
}
