package voice

import (
	"encoding/binary"
	"errors"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"sync"
	"time"
)

const maxVideoFrame = 384 << 10
const maxVideoPackets = 512

type videoFrame struct {
	packets     []*rtp.Packet
	key         bool
	at          time.Time
	encodedSize int
	encodeOnce  sync.Once
	encoded     []byte
	encodeError error
}

func (f *videoFrame) wireSize() int {
	if f.encodedSize > 0 {
		return f.encodedSize
	}
	size := 12
	for _, packet := range f.packets {
		size += 2 + packet.MarshalSize()
	}
	return size
}

func (f *videoFrame) wirePayload() ([]byte, error) {
	f.encodeOnce.Do(func() {
		f.encoded, f.encodeError = encodeVideoFrame(f)
	})
	return f.encoded, f.encodeError
}

type videoAssembler struct {
	packets   []*rtp.Packet
	timestamp uint32
	bytes     int
	key       bool
	started   time.Time
	expected  uint16
}

func (a *videoAssembler) push(p *rtp.Packet) (*videoFrame, bool) {
	var vp codecs.VP8Packet
	if len(p.Payload) > 1500 {
		a.packets = nil
		return nil, true
	}
	if _, err := vp.Unmarshal(p.Payload); err != nil || len(vp.Payload) == 0 {
		a.packets = nil
		return nil, true
	}
	start := vp.S == 1 && vp.PID == 0
	if start && vp.Payload[0]&1 == 0 && !safeVP8Keyframe(vp.Payload) {
		a.packets = nil
		return nil, true
	}
	if len(a.packets) > 0 && (p.Timestamp != a.timestamp || p.SequenceNumber != a.expected || time.Since(a.started) > 250*time.Millisecond) {
		a.packets = nil
		a.bytes = 0
	}
	if len(a.packets) == 0 {
		if !start {
			return nil, true
		}
		a.timestamp = p.Timestamp
		a.started = time.Now()
		a.key = vp.Payload[0]&1 == 0
		a.bytes = 0
	}
	a.bytes += p.MarshalSize()
	if a.bytes > maxVideoFrame || len(a.packets) >= maxVideoPackets {
		a.packets = nil
		return nil, true
	}
	a.expected = p.SequenceNumber + 1
	a.packets = append(a.packets, p.Clone())
	if !p.Marker {
		return nil, false
	}
	frame := &videoFrame{packets: a.packets, key: a.key, at: a.started, encodedSize: 12 + a.bytes + 2*len(a.packets)}
	a.packets = nil
	a.bytes = 0
	return frame, false
}
func encodeVideoFrame(f *videoFrame) ([]byte, error) {
	if len(f.packets) == 0 || len(f.packets) > maxVideoPackets {
		return nil, errors.New("invalid video frame")
	}
	data := make([]byte, 12, min(f.wireSize(), maxVideoFrame+2048))
	data[0] = 1
	if f.key {
		data[1] = 1
	}
	binary.BigEndian.PutUint64(data[2:], uint64(f.at.UnixMilli()))
	binary.BigEndian.PutUint16(data[10:], uint16(len(f.packets)))
	for _, p := range f.packets {
		raw, e := p.Marshal()
		if e != nil || len(raw) > 1600 {
			return nil, errors.New("invalid video RTP")
		}
		data = binary.BigEndian.AppendUint16(data, uint16(len(raw)))
		data = append(data, raw...)
		if len(data) > maxVideoFrame+2048 {
			return nil, errors.New("video frame limit")
		}
	}
	return data, nil
}
func decodeVideoFrame(data []byte) (*videoFrame, int64, error) {
	if len(data) < 12 || data[0] != 1 || len(data) > maxVideoFrame+2048 {
		return nil, 0, errors.New("video frame size")
	}
	count := int(binary.BigEndian.Uint16(data[10:]))
	if count < 1 || count > maxVideoPackets {
		return nil, 0, errors.New("video packet count")
	}
	stamp := int64(binary.BigEndian.Uint64(data[2:]))
	f := &videoFrame{key: data[1]&1 != 0, at: time.Now()}
	data = data[12:]
	var ts uint32
	var seq uint16
	for n := 0; n < count; n++ {
		if len(data) < 2 {
			return nil, 0, errors.New("truncated video packet")
		}
		size := int(binary.BigEndian.Uint16(data))
		data = data[2:]
		if size < 12 || size > 1600 || len(data) < size {
			return nil, 0, errors.New("video packet limit")
		}
		p := &rtp.Packet{}
		if p.Unmarshal(data[:size]) != nil {
			return nil, 0, errors.New("invalid RTP")
		}
		data = data[size:]
		var vp codecs.VP8Packet
		if _, e := vp.Unmarshal(p.Payload); e != nil || len(vp.Payload) == 0 {
			return nil, 0, errors.New("invalid VP8")
		}
		if n == 0 {
			if vp.S != 1 || vp.PID != 0 || f.key != (vp.Payload[0]&1 == 0) {
				return nil, 0, errors.New("invalid VP8 frame boundary")
			}
			if f.key && !safeVP8Keyframe(vp.Payload) {
				return nil, 0, errors.New("video resolution exceeds 1080p limit")
			}
			ts = p.Timestamp
			seq = p.SequenceNumber
		}
		if p.Timestamp != ts || p.SequenceNumber != seq || p.Marker != (n == count-1) {
			return nil, 0, errors.New("inconsistent video frame")
		}
		seq++
		f.packets = append(f.packets, p)
	}
	if len(data) != 0 {
		return nil, 0, errors.New("trailing video data")
	}
	return f, stamp, nil
}

// VP8 dimensions change only in a keyframe. Enforce the decoded surface budget
// as well as compressed frame/bitrate limits, including against malicious peers.
func safeVP8Keyframe(p []byte) bool {
	if len(p) < 10 || p[0]&1 != 0 || p[3] != 0x9d || p[4] != 1 || p[5] != 0x2a {
		return false
	}
	w := binary.LittleEndian.Uint16(p[6:8]) & 0x3fff
	h := binary.LittleEndian.Uint16(p[8:10]) & 0x3fff
	return w > 0 && h > 0 && w <= 1920 && h <= 1080
}
