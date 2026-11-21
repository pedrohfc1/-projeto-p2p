package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"os"
	"path/filepath"
	"projetop2p/internal/voice"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestRealGroupVoiceIndependentProcesses(t *testing.T) {
	binary := os.Getenv("P2P_WEBUI_TEST_BIN")
	helper := os.Getenv("P2P_ETCAT_TEST_DIR")
	if binary == "" || helper == "" {
		t.Skip("optional actual mesh integration")
	}
	for _, count := range []int{2, 3, 4, 6} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			root, _ := filepath.Abs("../../data")
			_ = os.MkdirAll(root, 0700)
			dir, e := os.MkdirTemp(root, "group-process-")
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			bridges := make([]voiceBridge, count)
			engines := make([]*voice.Engine, count)
			browsers := make([]*voice.MediaPeer, count)
			heard := make([]map[uint32]int, count)
			var mu sync.Mutex
			sentAt := map[uint32]time.Time{}
			var delayTotal, delayMax time.Duration
			var delaySamples int
			for i := range bridges {
				bridges[i] = startVoiceBridge(t, binary, helper, filepath.Join(dir, fmt.Sprint(i)))
				engines[i], e = voice.NewEngine("127.0.0.1")
				if e != nil {
					t.Fatal(e)
				}
				defer engines[i].Close()
				heard[i] = map[uint32]int{}
				index := i
				browsers[i], e = engines[i].Peer(func(p *rtp.Packet) {
					mu.Lock()
					heard[index][p.SSRC]++
					if len(p.CSRC) > 0 {
						if at, ok := sentAt[p.CSRC[0]]; ok {
							d := time.Since(at)
							delayTotal += d
							delaySamples++
							if d > delayMax {
								delayMax = d
							}
						}
					}
					mu.Unlock()
				})
				if e != nil {
					t.Fatal(e)
				}
				defer browsers[i].Close()
				for n := 0; n < 4; n++ {
					if _, e = browsers[i].PC.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); e != nil {
						t.Fatal(e)
					}
				}
			}
			var group, ticket string
			bridges[0].rpc(t, "CreateCommunity", map[string]any{"Name": "Mesh QA"}, &group)
			bridges[0].rpc(t, "CommunityInvitePolicy", map[string]any{"ID": group, "Options": map[string]any{"kind": "multi", "maxUses": 10, "lifetimeHours": 24}}, &ticket)
			for i := 1; i < count; i++ {
				var receipt map[string]any
				bridges[i].rpc(t, "JoinCommunity", map[string]any{"Text": ticket}, &receipt)
				if receipt["status"] != "participante" {
					t.Fatal(receipt)
				}
			}
			for i := range bridges {
				bridges[i].rpc(t, "SyncCommunity", map[string]any{"ID": group}, nil)
				var answer webrtc.SessionDescription
				bridges[i].rpc(t, "VoiceJoin", map[string]any{"ID": group, "Channel": "voz", "Offer": browserOffer(t, browsers[i])}, &answer)
				if e = browsers[i].PC.SetRemoteDescription(answer); e != nil {
					t.Fatal(e)
				}
			}
			started := time.Now()
			views := make([]voice.View, count)
			deadline := time.Now().Add(90 * time.Second)
			for {
				ready := true
				for i, b := range bridges {
					b.rpc(t, "VoiceState", nil, &views[i])
					connected := 0
					for _, p := range views[i].Participants {
						if !p.Self && p.State == "connected" {
							connected++
						}
					}
					if connected != count-1 || views[i].Expires <= time.Now().Unix() {
						ready = false
					}
				}
				if ready {
					break
				}
				if time.Now().After(deadline) {
					raw, _ := json.Marshal(views)
					t.Fatal("mesh connection timeout", string(raw))
				}
				time.Sleep(300 * time.Millisecond)
			}
			connection := time.Since(started)
			for _, b := range bridges {
				var baseline map[string]any
				b.rpc(t, "State", map[string]any{"ID": ""}, &baseline)
			}
			frames := 100
			if seconds, _ := strconv.Atoi(os.Getenv("P2P_GROUP_SOAK_SECONDS")); seconds > 0 && count == 6 {
				frames = seconds * 50
			}
			time.Sleep(3 * time.Second) // Allow a newly signed roster lease to settle before measuring delivery.
			for n := 1; n <= frames; n++ {
				mu.Lock()
				sentAt[uint32(n)] = time.Now()
				mu.Unlock()
				for i, p := range browsers {
					packet := &rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 111, SequenceNumber: uint16(n), Timestamp: uint32(n) * 960, SSRC: uint32(i + 1), CSRC: []uint32{uint32(n)}}, Payload: []byte{0xf8, 0xff, 0xfe}}
					if e = p.Write(packet); e != nil {
						t.Fatal(e)
					}
				}
				time.Sleep(20 * time.Millisecond)
				if n%500 == 0 {
					bridges[1].rpc(t, "CommunityAction", map[string]any{"ID": group, "Kind": "message.text", "Body": map[string]any{"channelId": "geral", "text": "chat during group call"}}, nil)
				}
			}
			time.Sleep(200 * time.Millisecond)
			mu.Lock()
			for i, streams := range heard {
				if len(streams) != count-1 {
					t.Errorf("peer %d: got %d independent sources; expected %d", i, len(streams), count-1)
				}
				for ssrc, n := range streams {
					if n < frames*8/10 {
						t.Errorf("peer %d source %d: %d/%d frames", i, ssrc, n, frames)
					}
				}
			}
			if delaySamples > 0 {
				t.Logf("MESH_RTP_LATENCY mean=%s max=%s samples=%d scope=same-machine-Pion-bridges-not-mouth-to-ear", delayTotal/time.Duration(delaySamples), delayMax, delaySamples)
			}
			mu.Unlock()
			for i, b := range bridges {
				var state map[string]any
				b.rpc(t, "State", map[string]any{"ID": ""}, &state)
				b.rpc(t, "VoiceState", nil, &views[i])
				raw, _ := json.Marshal(map[string]any{"peers": count, "index": i, "connectionSeconds": connection.Seconds(), "frames": frames, "backend": state["metrics"], "media": views[i].Media})
				t.Log("MESH_METRICS", string(raw))
			}
			// One participant leaves while all remaining members and their streams stay joined.
			bridges[count-1].rpc(t, "VoiceLeave", nil, nil)
			time.Sleep(4 * time.Second)
			for i := 0; i < count-1; i++ {
				bridges[i].rpc(t, "VoiceState", nil, &views[i])
				if !views[i].Joined {
					t.Fatal("one departure interrupted another participant")
				}
			}
			// Rejoin the same identity with a new local WebRTC session. Slot RTP sequence continuity must survive.
			last := count - 1
			browsers[last].Close()
			browsers[last], e = engines[last].Peer(func(p *rtp.Packet) { mu.Lock(); heard[last][p.SSRC]++; mu.Unlock() })
			if e != nil {
				t.Fatal(e)
			}
			defer browsers[last].Close()
			for n := 0; n < 4; n++ {
				_, e = browsers[last].PC.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
				if e != nil {
					t.Fatal(e)
				}
			}
			var newAnswer webrtc.SessionDescription
			bridges[last].rpc(t, "VoiceJoin", map[string]any{"ID": group, "Channel": "voz", "Offer": browserOffer(t, browsers[last])}, &newAnswer)
			if e = browsers[last].PC.SetRemoteDescription(newAnswer); e != nil {
				t.Fatal(e)
			}
			time.Sleep(8 * time.Second)
			baseline := make([]int, count)
			mu.Lock()
			for i, sources := range heard {
				for _, n := range sources {
					baseline[i] += n
				}
			}
			mu.Unlock()
			for n := 1; n <= 50; n++ {
				for _, browser := range browsers {
					if e = browser.Write(&rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 111, SequenceNumber: uint16(frames + n), Timestamp: uint32(frames+n) * 960, SSRC: 42}, Payload: []byte{0xf8, 0xff, 0xfe}}); e != nil {
						t.Fatal(e)
					}
				}
				time.Sleep(20 * time.Millisecond)
			}
			time.Sleep(300 * time.Millisecond)
			mu.Lock()
			for i, sources := range heard {
				total := 0
				for _, n := range sources {
					total += n
				}
				if total-baseline[i] < 30*(count-1) {
					t.Errorf("rejoin/slot reuse silent at peer %d: %d new frames", i, total-baseline[i])
				}
			}
			mu.Unlock()
			for _, b := range bridges {
				b.rpc(t, "VoiceLeave", nil, nil)
			}
			_ = context.Background()
		})
	}
}
