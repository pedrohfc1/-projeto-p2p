// Package voice provides authenticated-room media primitives. It never captures audio.
package voice

import (
	"errors"
	"fmt"
	"github.com/pion/ice/v4"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Engine struct {
	socket net.PacketConn
	mux    ice.UDPMux
	api    *webrtc.API
}
type MediaPeer struct {
	metricsMu      sync.Mutex
	started        time.Time
	samples        uint64
	first, highest int64
	lastArrival    float64
	lastTimestamp  uint32
	jitter         float64
	rtt            *float64
	PC             *webrtc.PeerConnection
	Track          *webrtc.TrackLocalStaticRTP
	received       atomic.Uint64
	sent           atomic.Uint64
	bytesIn        atomic.Uint64
	bytesOut       atomic.Uint64
	state          atomic.Value
}
type MediaStats struct {
	JitterMs    *float64 `json:"jitterMs"`
	PacketsLost *int64   `json:"packetsLost"`
	RoundTripMs *float64 `json:"roundTripMs"`
	State       string   `json:"state"`
	PacketsIn   uint64   `json:"packetsIn"`
	PacketsOut  uint64   `json:"packetsOut"`
	BytesIn     uint64   `json:"bytesIn"`
	BytesOut    uint64   `json:"bytesOut"`
}

func NewEngine(ip string) (*Engine, error) {
	if net.ParseIP(ip) == nil {
		return nil, errors.New("invalid media bind IP")
	}
	socket, e := net.ListenPacket("udp4", net.JoinHostPort(ip, "0"))
	if e != nil {
		return nil, e
	}
	codecs := &webrtc.MediaEngine{}
	e = codecs.RegisterCodec(webrtc.RTPCodecParameters{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2, SDPFmtpLine: "minptime=10;useinbandfec=1;stereo=0;sprop-stereo=0"}, PayloadType: 111}, webrtc.RTPCodecTypeAudio)
	if e != nil {
		socket.Close()
		return nil, e
	}
	registry := &interceptor.Registry{}
	if e = webrtc.RegisterDefaultInterceptors(codecs, registry); e != nil {
		socket.Close()
		return nil, e
	}
	mux := webrtc.NewICEUDPMux(nil, socket)
	settings := webrtc.SettingEngine{}
	settings.SetICEUDPMux(mux)
	settings.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4})
	settings.SetIncludeLoopbackCandidate(true)
	settings.SetIPFilter(func(addr net.IP) bool { return ip == "0.0.0.0" || addr.Equal(net.ParseIP(ip)) })
	settings.SetICEMulticastDNSMode(ice.MulticastDNSModeDisabled)
	settings.SetICETimeouts(5*time.Second, 12*time.Second, time.Second)
	return &Engine{socket: socket, mux: mux, api: webrtc.NewAPI(webrtc.WithMediaEngine(codecs), webrtc.WithInterceptorRegistry(registry), webrtc.WithSettingEngine(settings))}, nil
}
func (e *Engine) Port() int { return e.socket.LocalAddr().(*net.UDPAddr).Port }
func (e *Engine) Close()    { e.mux.Close(); e.socket.Close() }
func (e *Engine) Peer(onPacket func(*rtp.Packet)) (*MediaPeer, error) {
	pc, err := e.api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, err
	}
	track, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2, SDPFmtpLine: "minptime=10;useinbandfec=1;stereo=0;sprop-stereo=0"}, "audio", "p2p-voice")
	if err != nil {
		pc.Close()
		return nil, err
	}
	sender, err := pc.AddTrack(track)
	if err != nil {
		pc.Close()
		return nil, err
	}
	p := &MediaPeer{PC: pc, Track: track, started: time.Now()}
	p.state.Store("new")
	go func() {
		for {
			packets, _, err := sender.ReadRTCP()
			if err != nil {
				return
			}
			for _, packet := range packets {
				rr, ok := packet.(*rtcp.ReceiverReport)
				if !ok {
					continue
				}
				for _, report := range rr.Reports {
					if report.LastSenderReport == 0 {
						continue
					}
					now := time.Now()
					ntp := (uint64(now.Unix()+2208988800) << 16) | (uint64(now.Nanosecond()) * 65536 / 1000000000)
					delta := uint32(ntp) - report.LastSenderReport - report.Delay
					ms := float64(delta) * 1000 / 65536
					if ms >= 0 && ms < 30000 {
						p.metricsMu.Lock()
						p.rtt = &ms
						p.metricsMu.Unlock()
					}
				}
			}
		}
	}()
	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) { p.state.Store(s.String()) })
	pc.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		if !strings.EqualFold(t.Codec().MimeType, webrtc.MimeTypeOpus) {
			return
		}
		for {
			packet, _, e := t.ReadRTP()
			if e != nil {
				return
			}
			if len(packet.Payload) > 1200 {
				continue
			}
			p.metricsMu.Lock()
			arrival := float64(time.Since(p.started).Nanoseconds()) * 48000 / 1e9
			sequence := int64(packet.SequenceNumber)
			if p.samples == 0 {
				p.first = sequence
				p.highest = sequence
			} else {
				extended := p.highest + int64(int16(packet.SequenceNumber-uint16(p.highest)))
				if extended > p.highest {
					p.highest = extended
				}
				d := math.Abs(arrival - p.lastArrival - float64(int32(packet.Timestamp-p.lastTimestamp)))
				p.jitter += (d - p.jitter) / 16
			}
			p.samples++
			p.lastArrival = arrival
			p.lastTimestamp = packet.Timestamp
			p.metricsMu.Unlock()
			p.received.Add(1)
			p.bytesIn.Add(uint64(packet.MarshalSize()))
			if onPacket != nil {
				onPacket(packet)
			}
		}
	})
	return p, nil
}
func (p *MediaPeer) Write(packet *rtp.Packet) error {
	if p.Stats().State != "connected" {
		return errors.New("media peer is not connected")
	}
	err := p.Track.WriteRTP(packet)
	if err == nil {
		p.sent.Add(1)
		p.bytesOut.Add(uint64(packet.MarshalSize()))
	}
	return err
}
func (p *MediaPeer) Stats() MediaStats {
	state, _ := p.state.Load().(string)
	result := MediaStats{State: state, PacketsIn: p.received.Load(), PacketsOut: p.sent.Load(), BytesIn: p.bytesIn.Load(), BytesOut: p.bytesOut.Load()}
	p.metricsMu.Lock()
	defer p.metricsMu.Unlock()
	if p.samples > 1 {
		jitter := p.jitter * 1000 / 48000
		lost := p.highest - p.first + 1 - int64(p.samples)
		if lost < 0 {
			lost = 0
		}
		result.JitterMs = &jitter
		result.PacketsLost = &lost
	}
	if p.rtt != nil {
		rtt := *p.rtt
		result.RoundTripMs = &rtt
	}
	return result
}

func (p *MediaPeer) Close() { p.PC.Close() }
func CheckAudioSDP(d webrtc.SessionDescription) error {
	if len(d.SDP) > 32768 || !strings.Contains(d.SDP, "a=fingerprint:sha-256 ") {
		return errors.New("invalid authenticated audio SDP")
	}
	audio := 0
	for _, line := range strings.Split(d.SDP, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "m=") {
			if !strings.HasPrefix(line, "m=audio ") {
				return errors.New("only audio is allowed")
			}
			audio++
		}
	}
	if audio != 1 {
		return errors.New("one audio media section required")
	}
	return nil
}

// ForwardSDP changes only ICE routing. DTLS fingerprints remain as authenticated by libp2p.
func ForwardSDP(d webrtc.SessionDescription, ip string, port int) (webrtc.SessionDescription, error) {
	if e := CheckAudioSDP(d); e != nil {
		return d, e
	}
	if net.ParseIP(ip) == nil || port < 1024 || port > 65535 {
		return d, errors.New("invalid authorized media route")
	}
	lines := strings.Split(d.SDP, "\n")
	count := 0
	for n, line := range lines {
		if !strings.HasPrefix(line, "a=candidate:") {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 8 || !strings.EqualFold(fields[2], "udp") || fields[1] != "1" {
			lines[n] = ""
			continue
		}
		fields[4] = ip
		fields[5] = strconv.Itoa(port)
		fields[7] = "host"
		lines[n] = strings.Join(fields[:8], " ") + "\r"
		count++
	}
	if count == 0 {
		return d, fmt.Errorf("no UDP ICE candidate; media cannot use chat SOCKS CONNECT")
	}
	d.SDP = strings.Join(lines, "\n")
	return d, nil
}
