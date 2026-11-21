package voice

import (
	"encoding/binary"
	"fmt"
	"github.com/pion/rtp"
	"net"
	"testing"
	"time"
)

func TestStreamMediaBidirectionalAndBoundedQueue(t *testing.T) {
	a, b := net.Pipe()
	heard := make(chan *rtp.Packet, 20)
	left := newStreamMedia(a, func(p *rtp.Packet) { heard <- p })
	right := newStreamMedia(b, func(p *rtp.Packet) { heard <- p })
	defer left.Close()
	defer right.Close()
	for n := 0; n < 5; n++ {
		p := &rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: uint16(n), Timestamp: uint32(n) * 960}, Payload: []byte{0xf8, 0xff, 0xfe}}
		left.Push(p)
		right.Push(p)
		time.Sleep(20 * time.Millisecond)
	}
	for n := 0; n < 8; n++ {
		select {
		case <-heard:
		case <-time.After(time.Second):
			t.Fatal("duplex playback stalled")
		}
	}
	left.Close()
	select {
	case <-right.done:
	case <-time.After(time.Second):
		t.Fatal("peer did not close")
	}
	c, d := net.Pipe()
	defer d.Close()
	stalled := newStreamMedia(c, nil)
	defer stalled.Close()
	for n := 0; n < 1000; n++ {
		stalled.Push(&rtp.Packet{Header: rtp.Header{Version: 2}, Payload: []byte{0xf8, 0xff, 0xfe}})
	}
	if len(stalled.out) > 3 || stalled.Stats().Dropped == 0 {
		t.Fatal("unbounded/stale audio queue")
	}
	select {
	case <-stalled.done:
	case <-time.After(time.Second):
		t.Fatal("write deadline did not terminate blocked TCP")
	}
}
func TestStreamDropsTCPBacklog(t *testing.T) {
	a, b := net.Pipe()
	defer b.Close()
	heard := make(chan *rtp.Packet, 3)
	m := newStreamMedia(a, func(p *rtp.Packet) { heard <- p })
	defer m.Close()
	send := func(at int64) {
		p := &rtp.Packet{Header: rtp.Header{Version: 2}, Payload: []byte{0xf8, 0xff, 0xfe}}
		raw, _ := p.Marshal()
		frame := make([]byte, len(raw)+10)
		binary.BigEndian.PutUint16(frame, uint16(len(raw)+8))
		binary.BigEndian.PutUint64(frame[2:], uint64(at))
		copy(frame[10:], raw)
		b.SetWriteDeadline(time.Now().Add(time.Second))
		if _, e := b.Write(frame); e != nil {
			t.Fatal(e)
		}
	}
	now := time.Now().UnixMilli()
	send(now)
	select {
	case <-heard:
	case <-time.After(time.Second):
		t.Fatal("initial frame missing")
	}
	send(now - 500)
	time.Sleep(60 * time.Millisecond)
	if m.Stats().Dropped != 1 {
		t.Fatal("late TCP audio not discarded", m.Stats())
	}
}

func TestOpusDurationValidation(t *testing.T) {
	if !opus20ms([]byte{0xf8, 0xff, 0xfe}) {
		t.Fatal("20ms DTX refused")
	}
	for _, p := range [][]byte{nil, {0x80}, {0x90}, {0xfb, 0}, {0xfb, 63}} {
		if opus20ms(p) {
			t.Fatal("invalid/non20ms accepted", p)
		}
	}
}

func TestOfflineMediaReplayBudgetIsIsolatedPerPeer(t *testing.T) {
	c := &call{}
	now := time.Now().Unix()
	for n := 0; n < 256; n++ {
		if !reserveMediaNonce(c, "a", fmt.Sprint(n), now+LeaseSeconds, now) {
			t.Fatal("premature limit", n)
		}
	}
	if reserveMediaNonce(c, "a", "extra", now+LeaseSeconds, now) {
		t.Fatal("missing peer rate limit")
	}
	if !reserveMediaNonce(c, "b", "fresh", now+LeaseSeconds, now) {
		t.Fatal("another member exhausted this member's budget")
	}
	if reserveMediaNonce(c, "b", "fresh", now+LeaseSeconds, now+60) {
		t.Fatal("replay accepted during valid offline capability")
	}
	if !reserveMediaNonce(c, "a", "fresh", now+LeaseSeconds*2, now+LeaseSeconds+1) {
		t.Fatal("expired replay entries leaked budget")
	}
}
