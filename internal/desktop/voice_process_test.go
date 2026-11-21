package desktop

import (
	"bytes"
	"context"
	wirebinary "encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media/oggwriter"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"projetop2p/internal/voice"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type voiceBridge struct {
	url, token string
	client     *http.Client
}

func (b voiceBridge) rpc(t *testing.T, method string, args map[string]any, out any) {
	t.Helper()
	if args == nil {
		args = map[string]any{}
	}
	args["Method"] = method
	raw, _ := json.Marshal(args)
	req, _ := http.NewRequest("POST", b.url+"/rpc", bytes.NewReader(raw))
	req.Header.Set("X-QA-Token", b.token)
	req.Header.Set("Origin", b.url)
	res, e := b.client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode != 200 {
		t.Fatalf("%s: %s", method, payload)
	}
	if out != nil {
		if e = json.Unmarshal(payload, out); e != nil {
			t.Fatal(e)
		}
	}
}
func startVoiceBridge(t *testing.T, binary, helper, dir string) voiceBridge {
	t.Helper()
	port := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, binary, "--data", dir, "--port", fmt.Sprint(port))
	hideTestProcess(cmd)
	cmd.Dir = "../.."
	log, e := os.Create(filepath.Join(filepath.Dir(dir), filepath.Base(dir)+".log"))
	if e != nil {
		t.Fatal(e)
	}
	cmd.Stdout = log
	cmd.Stderr = log
	if e = cmd.Start(); e != nil {
		cancel()
		log.Close()
		t.Fatal(e)
	}
	t.Cleanup(func() { cancel(); _ = cmd.Wait(); log.Close() })
	b := voiceBridge{url: fmt.Sprintf("http://127.0.0.1:%d", port), client: &http.Client{Timeout: 50 * time.Second}}
	deadline := time.Now().Add(15 * time.Second)
	for {
		res, e := b.client.Get(b.url)
		if e == nil {
			data, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
			res.Body.Close()
			match := regexp.MustCompile(`X-QA-Token':'([a-f0-9]{64})'`).FindSubmatch(data)
			if len(match) == 2 {
				b.token = string(match[1])
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("QA backend startup; see isolated log")
		}
		time.Sleep(100 * time.Millisecond)
	}
	b.rpc(t, "SaveSettings", map[string]any{"Settings": map[string]any{"name": "Process voice QA", "mode": "integrated", "port": freePort(t), "integratedBinaryDir": helper}}, nil)
	return b
}
func TestRealVoiceTwoBackendProcesses(t *testing.T) {
	binary := os.Getenv("P2P_WEBUI_TEST_BIN")
	helper := os.Getenv("P2P_ETCAT_TEST_DIR")
	if binary == "" || helper == "" {
		t.Skip("optional separate-process production backend media test")
	}
	root, _ := filepath.Abs("../../data")
	_ = os.MkdirAll(root, 0700)
	dir, e := os.MkdirTemp(root, "voice-process-")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	a := startVoiceBridge(t, binary, helper, filepath.Join(dir, "a"))
	b := startVoiceBridge(t, binary, helper, filepath.Join(dir, "b"))
	var group, ticket string
	a.rpc(t, "CreateCommunity", map[string]any{"Name": "Two processes"}, &group)
	a.rpc(t, "CommunityInvite", map[string]any{"ID": group}, &ticket)
	var joined map[string]any
	b.rpc(t, "JoinCommunity", map[string]any{"Text": ticket}, &joined)
	if joined["status"] != "participante" {
		t.Fatal(joined)
	}
	var heardA, heardB atomic.Uint64
	var latencyMu sync.Mutex
	sentAt := map[uint16]time.Time{}
	var latencySum, latencyMax time.Duration
	var latencyCount int
	measureLatency := func(p *rtp.Packet) {
		latencyMu.Lock()
		defer latencyMu.Unlock()
		if at, ok := sentAt[p.SequenceNumber]; ok {
			elapsed := time.Since(at)
			latencySum += elapsed
			if elapsed > latencyMax {
				latencyMax = elapsed
			}
			latencyCount++
		}
	}
	var payloads [][]byte
	if fixture := os.Getenv("P2P_VOICE_OPUS_FIXTURE"); fixture != "" {
		raw, e := os.ReadFile(fixture)
		if e != nil {
			t.Fatal(e)
		}
		for len(raw) > 0 {
			if len(raw) < 2 {
				t.Fatal("invalid fixture")
			}
			n := int(wirebinary.BigEndian.Uint16(raw))
			raw = raw[2:]
			if n > len(raw) {
				t.Fatal("invalid fixture size")
			}
			payloads = append(payloads, append([]byte(nil), raw[:n]...))
			raw = raw[n:]
		}
	}
	var writerA, writerB *oggwriter.OggWriter
	if len(payloads) > 0 {
		writerA, e = oggwriter.New(filepath.Join(dir, "a-received.ogg"), 48000, 1)
		if e != nil {
			t.Fatal(e)
		}
		defer writerA.Close()
		writerB, e = oggwriter.New(filepath.Join(dir, "b-received.ogg"), 48000, 1)
		if e != nil {
			t.Fatal(e)
		}
		defer writerB.Close()
	}
	engineA, e := voice.NewEngine("127.0.0.1")
	if e != nil {
		t.Fatal(e)
	}
	defer engineA.Close()
	engineB, e := voice.NewEngine("127.0.0.1")
	if e != nil {
		t.Fatal(e)
	}
	defer engineB.Close()
	pa, e := engineA.Peer(func(p *rtp.Packet) {
		heardA.Add(1)
		measureLatency(p)
		if writerA != nil {
			_ = writerA.WriteRTP(p)
		}
	})
	if e != nil {
		t.Fatal(e)
	}
	defer pa.Close()
	pb, e := engineB.Peer(func(p *rtp.Packet) {
		heardB.Add(1)
		measureLatency(p)
		if writerB != nil {
			_ = writerB.WriteRTP(p)
		}
	})
	if e != nil {
		t.Fatal(e)
	}
	defer pb.Close()
	for i, bridge := range []voiceBridge{a, b} {
		p := []*voice.MediaPeer{pa, pb}[i]
		var answer webrtc.SessionDescription
		bridge.rpc(t, "VoiceJoin", map[string]any{"ID": group, "Channel": "voz", "Offer": browserOffer(t, p)}, &answer)
		if e = p.PC.SetRemoteDescription(answer); e != nil {
			t.Fatal(e)
		}
	}
	started := time.Now()
	var va, vb voice.View
	for {
		a.rpc(t, "VoiceState", nil, &va)
		b.rpc(t, "VoiceState", nil, &vb)
		if va.Media.State == "connected" && vb.Media.State == "connected" {
			break
		}
		if time.Since(started) > 60*time.Second {
			t.Fatal(va, vb)
		}
		time.Sleep(200 * time.Millisecond)
	}
	connectionDuration := time.Since(started)
	for _, bridge := range []voiceBridge{a, b} {
		var baseline map[string]any
		bridge.rpc(t, "State", map[string]any{"ID": ""}, &baseline)
	}
	initialExpiryA, initialExpiryB := va.Expires, vb.Expires
	frames := 50
	if seconds, _ := strconv.Atoi(os.Getenv("P2P_VOICE_SOAK_SECONDS")); seconds > 0 {
		frames = seconds * 50
	}
	for n := 1; n <= frames; n++ {
		latencyMu.Lock()
		sentAt[uint16(n)] = time.Now()
		latencyMu.Unlock()
		for _, p := range []*voice.MediaPeer{pa, pb} {
			packet := &rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 111, SequenceNumber: uint16(n), Timestamp: uint32(n) * 960, SSRC: 42}, Payload: []byte{0xf8, 0xff, 0xfe}}
			if len(payloads) > 0 {
				packet.Payload = payloads[(n-1)%len(payloads)]
			}
			if e = p.Write(packet); e != nil {
				t.Fatal(e)
			}
		}
		time.Sleep(20 * time.Millisecond)
		if n%500 == 0 {
			a.rpc(t, "VoiceState", nil, &va)
			b.rpc(t, "VoiceState", nil, &vb)
			if !va.Joined || !vb.Joined {
				t.Fatal("unexpected room leave during media")
			}
			b.rpc(t, "CommunityAction", map[string]any{"ID": group, "Kind": "message.text", "Body": map[string]any{"channelId": "geral", "text": "chat during continuous media"}}, nil)
		}
	}
	time.Sleep(500 * time.Millisecond)
	if heardA.Load() < uint64(frames*9/10) || heardB.Load() < uint64(frames*9/10) {
		t.Fatal("separate-process audio bridge", heardA.Load(), heardB.Load())
	}
	a.rpc(t, "VoiceState", nil, &va)
	b.rpc(t, "VoiceState", nil, &vb)
	if frames >= 500 && (va.Expires <= initialExpiryA || vb.Expires <= initialExpiryB) {
		t.Fatal("signed leases did not renew during soak")
	}
	summary, _ := json.Marshal(map[string]any{"a": va.Media, "b": vb.Media})
	latencyMu.Lock()
	if latencyCount > 0 {
		t.Log("measured same-PC RTP bridge delivery mean/max", latencySum/time.Duration(latencyCount), latencyMax, "samples", latencyCount)
	}
	latencyMu.Unlock()
	t.Log("measured RTP statistics", string(summary), "frames each direction", frames)
	for _, bridge := range []voiceBridge{a, b} {
		var state map[string]any
		bridge.rpc(t, "State", map[string]any{"ID": ""}, &state)
		raw, _ := json.Marshal(state["metrics"])
		t.Log("backend metrics during synthetic audio", string(raw))
	}
	t.Logf("two production processes, connected in %s, delivered %d/%d + %d/%d synthetic frames", connectionDuration, heardA.Load(), frames, heardB.Load(), frames)
	a.rpc(t, "VoiceLeave", nil, nil)
	b.rpc(t, "VoiceLeave", nil, nil)
	a.rpc(t, "VoiceState", nil, &va)
	b.rpc(t, "VoiceState", nil, &vb)
	if writerA != nil {
		// Leave closes production senders first; close receiver tracks before flushing the recordings.
		pa.Close()
		pb.Close()
		writerA.Close()
		writerB.Close()
		for _, name := range []string{"a", "b"} {
			pcm := filepath.Join(dir, name+".pcm")
			cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-i", filepath.Join(dir, name+"-received.ogg"), "-f", "f32le", "-ac", "1", "-ar", "48000", "-y", pcm)
			hideTestProcess(cmd)
			if data, e := cmd.CombinedOutput(); e != nil {
				t.Fatal("Opus decode", e, string(data))
			}
			raw, e := os.ReadFile(pcm)
			if e != nil || len(raw) < 48000 {
				t.Fatal("decoded PCM missing", e)
			}
			sum := 0.0
			for n := 0; n+4 <= len(raw); n += 4 {
				v := float64(math.Float32frombits(wirebinary.LittleEndian.Uint32(raw[n:])))
				sum += v * v
			}
			rms := math.Sqrt(sum / float64(len(raw)/4))
			if rms < .02 {
				t.Fatal("decoded audio silent", rms)
			}
			t.Log("REAL Opus decoded PCM RMS", name, rms, "samples", len(raw)/4)
		}
	}
	if va.Joined || vb.Joined {
		t.Fatal("call cleanup failed")
	}
}
