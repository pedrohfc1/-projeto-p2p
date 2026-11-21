package voice

import (
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"sync/atomic"
	"testing"
	"time"
)

// Mirrors WebView's five transceivers without acquiring/capturing any microphone.
func TestReceiveOnlyThenAttachMicrophone(t *testing.T) {
	engine, e := NewEngine("127.0.0.1")
	if e != nil {
		t.Fatal(e)
	}
	defer engine.Close()
	var received atomic.Uint64
	backend, e := engine.Peer(func(*rtp.Packet) { received.Add(1) }, 5)
	if e != nil {
		t.Fatal(e)
	}
	defer backend.Close()
	browserEngine, e := NewEngine("127.0.0.1")
	if e != nil {
		t.Fatal(e)
	}
	defer browserEngine.Close()
	browser, e := browserEngine.api.NewPeerConnection(webrtc.Configuration{})
	if e != nil {
		t.Fatal(e)
	}
	defer browser.Close()
	first, e := browser.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendrecv})
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 4; i++ {
		if _, e = browser.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); e != nil {
			t.Fatal(e)
		}
	}
	var heard atomic.Uint64
	browser.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		for {
			if _, _, e := track.ReadRTP(); e != nil {
				return
			}
			heard.Add(1)
		}
	})
	offer, e := browser.CreateOffer(nil)
	if e != nil {
		t.Fatal(e)
	}
	gather := webrtc.GatheringCompletePromise(browser)
	if e = browser.SetLocalDescription(offer); e != nil {
		t.Fatal(e)
	}
	<-gather
	if e = backend.PC.SetRemoteDescription(*browser.LocalDescription()); e != nil {
		t.Fatal(e)
	}
	answer, e := backend.PC.CreateAnswer(nil)
	if e != nil {
		t.Fatal(e)
	}
	gather = webrtc.GatheringCompletePromise(backend.PC)
	if e = backend.PC.SetLocalDescription(answer); e != nil {
		t.Fatal(e)
	}
	<-gather
	if e = browser.SetRemoteDescription(*backend.PC.LocalDescription()); e != nil {
		t.Fatal(e)
	}
	packet := func(n uint16) *rtp.Packet {
		return &rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: n, Timestamp: uint32(n) * 960, SSRC: 42}, Payload: []byte{0xf8, 0xff, 0xfe}}
	}
	until := time.Now().Add(12 * time.Second)
	for n := uint16(0); heard.Load() == 0 && time.Now().Before(until); n++ {
		backend.Track.WriteRTP(packet(n))
		time.Sleep(20 * time.Millisecond)
	}
	if heard.Load() == 0 || received.Load() != 0 {
		t.Fatalf("receive-only heard=%d sent=%d browser=%s backend=%s", heard.Load(), received.Load(), browser.ConnectionState(), backend.PC.ConnectionState())
	}
	track, e := webrtc.NewTrackLocalStaticRTP(backend.Track.Codec(), "reconnected-microphone", "local")
	if e != nil {
		t.Fatal(e)
	}
	if e = first.Sender().ReplaceTrack(track); e != nil {
		t.Fatal(e)
	}
	until = time.Now().Add(5 * time.Second)
	for n := uint16(0); received.Load() == 0 && time.Now().Before(until); n++ {
		track.WriteRTP(packet(n))
		time.Sleep(20 * time.Millisecond)
	}
	if received.Load() == 0 {
		t.Fatal("real microphone track could not attach without renegotiation")
	}
}
