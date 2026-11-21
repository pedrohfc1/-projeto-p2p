package desktop

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"os"
	"path/filepath"
	"projetop2p/internal/community"
	"projetop2p/internal/localdb"
	"projetop2p/internal/voice"
	"sync/atomic"
	"testing"
	"time"
)

func TestRealThreePeerOwnerOutageAndStreamRecovery(t *testing.T) {
	helper := os.Getenv("P2P_ETCAT_TEST_DIR")
	if helper == "" {
		t.Skip("requires audited EasyTier helper; synthetic media, one PC")
	}
	root, _ := filepath.Abs("../../data")
	os.MkdirAll(root, 0700)
	dir, e := os.MkdirTemp(root, "voice-recovery-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(dir)
	cores := make([]*Core, 3)
	engines := make([]*voice.Engine, 3)
	browsers := make([]*voice.MediaPeer, 3)
	var heard [3]atomic.Uint64
	defer func() {
		for i, c := range cores {
			if c != nil {
				c.Close()
			}
			if browsers[i] != nil {
				browsers[i].Close()
			}
			if engines[i] != nil {
				engines[i].Close()
			}
		}
	}()
	open := func(i int) *Core {
		c, e := Open(context.Background(), filepath.Join(dir, fmt.Sprint(i)))
		if e != nil {
			t.Fatal(e)
		}
		if e = c.SaveSettings(localdb.Settings{Name: fmt.Sprint(i), Mode: "integrated", Port: integratedPort(t), IntegratedBinaryDir: helper}); e != nil {
			c.Close()
			t.Fatal(e)
		}
		return c
	}
	for i := range cores {
		cores[i] = open(i)
	}
	group, e := cores[0].CreateCommunity("recovery", "isolated synthetic media", "")
	if e != nil {
		t.Fatal(e)
	}
	ticket, e := cores[0].CommunityInvitePolicy(group, community.InviteOptions{Kind: "multi", MaxUses: 10, LifetimeHours: 24})
	if e != nil {
		t.Fatal(e)
	}
	for i := 1; i < 3; i++ {
		r, e := cores[i].JoinCommunity(ticket)
		if e != nil || r.Status != "participante" {
			t.Fatal(r, e)
		}
	}
	for i, c := range cores {
		if e = c.SyncCommunity(group); e != nil {
			t.Fatal(e)
		}
		engines[i], e = voice.NewEngine("127.0.0.1")
		if e != nil {
			t.Fatal(e)
		}
		index := i
		browsers[i], e = engines[i].Peer(func(*rtp.Packet) { heard[index].Add(1) })
		if e != nil {
			t.Fatal(e)
		}
		for n := 0; n < 4; n++ {
			if _, e = browsers[i].PC.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); e != nil {
				t.Fatal(e)
			}
		}
		answer, e := c.VoiceJoin(group, "voz", browserOffer(t, browsers[i]))
		if e != nil {
			t.Fatal(e)
		}
		if e = browsers[i].PC.SetRemoteDescription(answer); e != nil {
			t.Fatal(e)
		}
	}
	ready := func(deadline time.Time, all bool) {
		for {
			ok := true
			for i := 1; i < 3; i++ {
				v := cores[i].VoiceState()
				connected := 0
				for _, p := range v.Participants {
					if !p.Self && p.State == "connected" {
						connected++
					}
				}
				need := 1
				if all {
					need = 2
				}
				if !v.Joined || v.Authorization != "valid" || connected < need {
					ok = false
				}
			}
			if ok {
				return
			}
			if time.Now().After(deadline) {
				t.Fatal("recovery timeout", cores[1].VoiceState(), cores[2].VoiceState())
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	ready(time.Now().Add(60*time.Second), true)
	sessions := [2]string{cores[1].VoiceState().Session, cores[2].VoiceState().Session}
	if e = cores[0].Close(); e != nil {
		t.Fatal(e)
	}
	before := [2]uint64{heard[1].Load(), heard[2].Load()}
	started := time.Now()
	sequence := uint16(1)
	pump := func(frames int) {
		for n := 0; n < frames; n++ {
			for i := 1; i < 3; i++ {
				if e := browsers[i].Write(&rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 111, SequenceNumber: sequence, Timestamp: uint32(sequence) * 960, SSRC: uint32(i + 1)}, Payload: []byte{0xf8, 0xff, 0xfe}}); e != nil {
					t.Fatal(e)
				}
			}
			sequence++
			time.Sleep(20 * time.Millisecond)
		}
	}
	pump(100)
	if heard[1].Load() <= before[0]+50 || heard[2].Load() <= before[1]+50 {
		t.Fatal("owner outage stopped authorized member media")
	}
	// Reset exactly the authenticated connection between surviving members.
	pid, _ := peer.Decode(cores[2].node.SelfID())
	if e = cores[1].node.Host.Network().ClosePeer(pid); e != nil {
		t.Fatal(e)
	}
	ready(time.Now().Add(35*time.Second), false)
	before = [2]uint64{heard[1].Load(), heard[2].Load()}
	pump(100)
	if heard[1].Load() <= before[0]+50 || heard[2].Load() <= before[1]+50 {
		t.Fatal("stream reset did not recover bidirectionally")
	}
	// Cross the authority's 45-second presence window while cached signed grants remain valid.
	for time.Since(started) < 50*time.Second {
		pump(50)
	}
	for i := 1; i < 3; i++ {
		v := cores[i].VoiceState()
		if !v.Joined || v.Session != sessions[i-1] || v.Authorization != "valid" {
			t.Fatal("owner outage invalidated a still-valid grant", v)
		}
	}
	cores[0] = open(0)
	ready(time.Now().Add(40*time.Second), false)
	deadline := time.Now().Add(40 * time.Second)
	for {
		a, b := cores[1].VoiceState(), cores[2].VoiceState()
		if a.Expires > started.Unix()+voice.LeaseSeconds && b.Expires > started.Unix()+voice.LeaseSeconds {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("owner return failed to renew grants", a, b)
		}
		time.Sleep(200 * time.Millisecond)
	}
	ready(time.Now().Add(40*time.Second), false)
	before = [2]uint64{heard[1].Load(), heard[2].Load()}
	pump(100)
	if heard[1].Load() <= before[0]+50 || heard[2].Load() <= before[1]+50 {
		t.Fatal("media not restored after fresh authority generation")
	}
	t.Log("PASS: three actual cores, owner/EasyTier endpoint offline >45s, signed offline continuity, peer stream reset, owner restart/new generation and bidirectional recovery without new sessions; synthetic Opus on one PC")
}
