package voice

import (
	"errors"
	"github.com/pion/ice/v4"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type videoBridge struct {
	pc        *webrtc.PeerConnection
	track     *webrtc.TrackLocalStaticRTP
	socket    net.PacketConn
	mux       ice.UDPMux
	inputSSRC atomic.Uint32
	mu        sync.Mutex
	seq       uint16
	started   time.Time
	pliAt     atomic.Int64
}

func newVideoBridge(frame func(*videoFrame), requestKey func()) (*videoBridge, error) {
	socket, e := net.ListenPacket("udp4", "0.0.0.0:0")
	if e != nil {
		return nil, e
	}
	m := &webrtc.MediaEngine{}
	feedback := []webrtc.RTCPFeedback{{Type: "nack", Parameter: "pli"}, {Type: "ccm", Parameter: "fir"}, {Type: "goog-remb"}}
	if e = m.RegisterCodec(webrtc.RTPCodecParameters{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000, RTCPFeedback: feedback}, PayloadType: 96}, webrtc.RTPCodecTypeVideo); e != nil {
		socket.Close()
		return nil, e
	}
	r := &interceptor.Registry{}
	if e = webrtc.RegisterDefaultInterceptors(m, r); e != nil {
		socket.Close()
		return nil, e
	}
	mux := webrtc.NewICEUDPMux(nil, socket)
	settings := webrtc.SettingEngine{}
	settings.SetICEUDPMux(mux)
	settings.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4})
	settings.SetIncludeLoopbackCandidate(true)
	settings.SetICEMulticastDNSMode(ice.MulticastDNSModeDisabled)
	pc, e := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(r), webrtc.WithSettingEngine(settings)).NewPeerConnection(webrtc.Configuration{})
	if e != nil {
		mux.Close()
		socket.Close()
		return nil, e
	}
	track, e := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}, "screen", "peetwopee-screen")
	if e != nil {
		pc.Close()
		mux.Close()
		socket.Close()
		return nil, e
	}
	sender, e := pc.AddTrack(track)
	if e != nil {
		pc.Close()
		mux.Close()
		socket.Close()
		return nil, e
	}
	b := &videoBridge{pc: pc, track: track, socket: socket, mux: mux, started: time.Now()}
	go func() {
		for {
			packets, _, e := sender.ReadRTCP()
			if e != nil {
				return
			}
			for _, p := range packets {
				switch p.(type) {
				case *rtcp.PictureLossIndication, *rtcp.FullIntraRequest:
					if requestKey != nil {
						requestKey()
					}
				}
			}
		}
	}()
	pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		if !strings.EqualFold(track.Codec().MimeType, webrtc.MimeTypeVP8) {
			return
		}
		b.inputSSRC.Store(uint32(track.SSRC()))
		a := videoAssembler{}
		for {
			p, _, e := track.ReadRTP()
			if e != nil {
				return
			}
			f, dropped := a.push(p)
			if dropped {
				b.requestKeyframe()
			}
			if f != nil {
				frame(f)
			}
		}
	})
	return b, nil
}
func (b *videoBridge) requestKeyframe() {
	now := time.Now().UnixMilli()
	old := b.pliAt.Load()
	if now-old < 500 || !b.pliAt.CompareAndSwap(old, now) {
		return
	}
	if s := b.inputSSRC.Load(); s != 0 {
		_ = b.pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: s}})
	}
}
func (b *videoBridge) write(f *videoFrame) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	timestamp := uint32(time.Since(b.started).Seconds() * 90000)
	for _, p := range f.packets {
		copy := p.Clone()
		var vp codecs.VP8Packet
		if _, err := vp.Unmarshal(copy.Payload); err != nil {
			return err
		}
		// Optional picture IDs belong to the source. Remove them on the local SFU
		// output so a new source/session cannot be mistaken for an old decoder frame.
		copy.Payload = append([]byte{copy.Payload[0] & 0x3f}, vp.Payload...)
		b.seq++
		copy.SequenceNumber = b.seq
		copy.Timestamp = timestamp
		if e := b.track.WriteRTP(copy); e != nil {
			return e
		}
	}
	return nil
}
func (b *videoBridge) close() { b.pc.Close(); b.mux.Close(); b.socket.Close() }
func checkVideoSDP(d webrtc.SessionDescription) error {
	if len(d.SDP) > 32768 || d.Type != webrtc.SDPTypeOffer || !strings.Contains(d.SDP, "a=fingerprint:sha-256 ") {
		return errors.New("invalid screen SDP")
	}
	count := 0
	for _, line := range strings.Split(d.SDP, "\n") {
		if strings.HasPrefix(line, "m=") {
			if !strings.HasPrefix(line, "m=video ") {
				return errors.New("screen bridge permits video only")
			}
			count++
		}
	}
	if count != 1 {
		return errors.New("one video section required")
	}
	return nil
}
