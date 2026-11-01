package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"projetop2p/internal/voice"
	"regexp"
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
	pa, e := engineA.Peer(func(*rtp.Packet) { heardA.Add(1) })
	if e != nil {
		t.Fatal(e)
	}
	defer pa.Close()
	pb, e := engineB.Peer(func(*rtp.Packet) { heardB.Add(1) })
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
	for n := 1; n <= 50; n++ {
		for _, p := range []*voice.MediaPeer{pa, pb} {
			packet := &rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 111, SequenceNumber: uint16(n), Timestamp: uint32(n) * 960, SSRC: 42}, Payload: []byte{0xf8, 0xff, 0xfe}}
			if e = p.Write(packet); e != nil {
				t.Fatal(e)
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)
	if heardA.Load() < 45 || heardB.Load() < 45 {
		t.Fatal("separate-process audio bridge", heardA.Load(), heardB.Load())
	}
	a.rpc(t, "VoiceState", nil, &va)
	b.rpc(t, "VoiceState", nil, &vb)
	summary, _ := json.Marshal(map[string]any{"a": va.Media, "b": vb.Media})
	t.Log("measured RTP statistics", string(summary))
	for _, bridge := range []voiceBridge{a, b} {
		var state map[string]any
		bridge.rpc(t, "State", map[string]any{"ID": ""}, &state)
		raw, _ := json.Marshal(state["metrics"])
		t.Log("backend metrics during synthetic audio", string(raw))
	}
	t.Logf("two production processes, connected in %s, synthetic Opus delivered %d/50 + %d/50, RTP stats: %+v / %+v", connectionDuration, heardA.Load(), heardB.Load(), va.Media, vb.Media)
	a.rpc(t, "VoiceLeave", nil, nil)
	b.rpc(t, "VoiceLeave", nil, nil)
	a.rpc(t, "VoiceState", nil, &va)
	b.rpc(t, "VoiceState", nil, &vb)
	if va.Joined || vb.Joined {
		t.Fatal("call cleanup failed")
	}
}
