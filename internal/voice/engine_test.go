package voice

import (
	"context"
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"net"
	"os"
	"path/filepath"
	"projetop2p/internal/overlay"
	"strings"
	"testing"
	"time"
)

func description(t *testing.T, p *MediaPeer, offer bool) webrtc.SessionDescription {
	t.Helper()
	var d webrtc.SessionDescription
	var e error
	if offer {
		d, e = p.PC.CreateOffer(nil)
	} else {
		d, e = p.PC.CreateAnswer(nil)
	}
	if e != nil {
		t.Fatal(e)
	}
	ready := webrtc.GatheringCompletePromise(p.PC)
	if e = p.PC.SetLocalDescription(d); e != nil {
		t.Fatal(e)
	}
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("ICE gathering")
	}
	return *p.PC.LocalDescription()
}
func TestRealOpusSRTPOverEasyTierUDP(t *testing.T) {
	dir := os.Getenv("P2P_ETCAT_TEST_DIR")
	if dir == "" {
		t.Skip("optional real userspace UDP relay")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	root, _ := filepath.Abs("../../data")
	os.MkdirAll(root, 0700)
	folder, e := os.MkdirTemp(root, "voice-udp-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(folder)
	a, e := NewEngine("127.0.0.1")
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	b, e := NewEngine("127.0.0.1")
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	managers := []*overlay.Manager{}
	ids := []string{}
	for n, engine := range []*Engine{a, b} {
		profile := filepath.Join(folder, string(rune('a'+n)))
		os.MkdirAll(profile, 0700)
		m, e := overlay.Open(ctx, profile, filepath.Join(dir, "etcat-p2p.exe"), "")
		if e != nil {
			t.Fatal(e)
		}
		defer m.Close()
		tcp, e := net.Listen("tcp", "127.0.0.1:0")
		if e != nil {
			t.Fatal(e)
		}
		defer tcp.Close()
		if e = m.Start(ctx, tcp.Addr().(*net.TCPAddr).Port, engine.Port()); e != nil {
			t.Fatal(e)
		}
		managers = append(managers, m)
		k, _, _ := crypto.GenerateEd25519Key(rand.Reader)
		id, _ := peer.IDFromPrivateKey(k)
		ids = append(ids, id.String())
	}
	if e = managers[0].Register(ids[1], managers[1].Token()); e != nil {
		t.Fatal(e)
	}
	if e = managers[1].Register(ids[0], managers[0].Token()); e != nil {
		t.Fatal(e)
	}
	forwardA, e := managers[0].MediaForward(ctx, ids[1], b.Port())
	if e != nil {
		t.Fatal(e)
	}
	forwardB, e := managers[1].MediaForward(ctx, ids[0], a.Port())
	if e != nil {
		t.Fatal(e)
	}
	received := make(chan uint16, 128)
	pa, e := a.Peer(nil)
	if e != nil {
		t.Fatal(e)
	}
	defer pa.Close()
	pb, e := b.Peer(func(packet *rtp.Packet) {
		select {
		case received <- packet.SequenceNumber:
		default:
		}
	})
	if e != nil {
		t.Fatal(e)
	}
	defer pb.Close()
	offer, e := ForwardSDP(description(t, pa, true), "127.0.0.1", forwardB)
	if e != nil {
		t.Fatal(e)
	}
	if e = pb.PC.SetRemoteDescription(offer); e != nil {
		t.Fatal(e)
	}
	answer, e := ForwardSDP(description(t, pb, false), "127.0.0.1", forwardA)
	if e != nil {
		t.Fatal(e)
	}
	start := time.Now()
	if e = pa.PC.SetRemoteDescription(answer); e != nil {
		t.Fatal(e)
	}
	for pa.Stats().State != "connected" || pb.Stats().State != "connected" {
		if time.Since(start) > 35*time.Second {
			t.Fatal("UDP ICE/DTLS failed", pa.Stats(), pb.Stats())
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Log("ICE+DTLS establishment", time.Since(start))
	for n := 0; n < 50; n++ {
		if e = pa.Write(&rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 111, SequenceNumber: uint16(n), Timestamp: uint32(n * 960), SSRC: 1234}, Payload: []byte{0xf8, 0xff, 0xfe}}); e != nil {
			t.Fatal(e)
		}
		time.Sleep(20 * time.Millisecond)
	}
	deadline := time.Now().Add(3 * time.Second)
	for pb.Stats().PacketsIn < 45 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if pb.Stats().PacketsIn < 45 {
		t.Fatal("Opus SRTP loss", pa.Stats(), pb.Stats())
	}
	t.Log("50 synthetic Opus DTX frames over SRTP", pa.Stats(), pb.Stats(), "not microphone/device or two-house validation")
}

func TestMediaSDPBoundsAndRoutingPreserveFingerprint(t *testing.T) {
	d := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: "v=0\r\nm=audio 9 UDP/TLS/RTP/SAVPF 111\r\na=fingerprint:sha-256 TEST\r\na=candidate:1 1 udp 2130706431 192.168.1.2 5000 typ host\r\n"}
	routed, e := ForwardSDP(d, "127.0.0.1", 6000)
	if e != nil || !strings.Contains(routed.SDP, "a=fingerprint:sha-256 TEST") || !strings.Contains(routed.SDP, "127.0.0.1 6000") {
		t.Fatal(routed, e)
	}
	for _, invalid := range []string{d.SDP + "m=video 9 UDP/TLS/RTP/SAVPF 96\r\n", strings.Repeat("x", 32769), "m=audio 9 UDP/TLS/RTP/SAVPF 111\r\n"} {
		if CheckAudioSDP(webrtc.SessionDescription{SDP: invalid}) == nil {
			t.Fatal("unsafe SDP accepted")
		}
	}
}
