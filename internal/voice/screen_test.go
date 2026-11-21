package voice

import (
	"context"
	"github.com/pion/rtp"
	"net"
	"sync"
	"testing"
	"time"
)

func videoPacket(seq uint16, ts uint32, first, last, key bool) *rtp.Packet {
	descriptor := byte(0)
	if first {
		descriptor = 0x10
	}
	payload := byte(1)
	if key {
		payload = 0
	}
	return &rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 96, SequenceNumber: seq, Timestamp: ts, Marker: last}, Payload: append([]byte{descriptor}, testVP8Payload(first, key, payload)...)}
}
func TestVideoFrameBoundaryLossAndLimits(t *testing.T) {
	a := videoAssembler{}
	if f, _ := a.push(videoPacket(1, 100, true, false, true)); f != nil {
		t.Fatal("incomplete frame emitted")
	}
	f, d := a.push(videoPacket(2, 100, false, true, true))
	if d || f == nil || !f.key || len(f.packets) != 2 {
		t.Fatal("complete encoded frame missing")
	}
	raw, e := encodeVideoFrame(f)
	if e != nil {
		t.Fatal(e)
	}
	decoded, _, e := decodeVideoFrame(raw)
	if e != nil || len(decoded.packets) != 2 {
		t.Fatal(e)
	}
	raw[1] = 0
	if _, _, e = decodeVideoFrame(raw); e == nil {
		t.Fatal("forged keyframe flag accepted")
	}
	a.push(videoPacket(4, 200, true, false, true))
	if f, _ = a.push(videoPacket(6, 200, false, true, true)); f != nil {
		t.Fatal("gapped reference frame emitted")
	}
	if _, _, e = decodeVideoFrame(make([]byte, maxVideoFrame+4096)); e == nil {
		t.Fatal("unbounded frame")
	}
}
func TestVideoPipeEncodedFrameAndBoundedBacklog(t *testing.T) {
	a, b := net.Pipe()
	heard := make(chan *videoFrame, 2)
	key := func() {}
	left := newVideoPipe(a, func(*videoFrame) {}, func(byte) {}, key, func() bool { return true }, func(int) bool { return true })
	right := newVideoPipe(b, func(f *videoFrame) { heard <- f }, func(byte) {}, key, func() bool { return true }, func(int) bool { return true })
	defer left.close()
	defer right.close()
	left.push(&videoFrame{packets: []*rtp.Packet{videoPacket(1, 100, true, true, true)}, key: true, at: time.Now()})
	select {
	case f := <-heard:
		if len(f.packets) != 1 {
			t.Fatal("packet boundary lost")
		}
	case <-time.After(time.Second):
		t.Fatal("encoded frame stalled")
	}
	left.push(&videoFrame{packets: []*rtp.Packet{videoPacket(2, 200, true, true, true)}, key: true, at: time.Now().Add(-time.Second)})
	time.Sleep(50 * time.Millisecond)
	if left.out.Load() != 1 || right.out.Load() != 0 {
		t.Fatal("ACK/control counted as video frames", left.out.Load(), right.out.Load())
	}
	if left.drops.Load() == 0 {
		t.Fatal("late video not dropped")
	}
	c, d := net.Pipe()
	defer d.Close()
	blocked := newVideoPipe(c, func(*videoFrame) {}, func(byte) {}, key, func() bool { return true }, func(int) bool { return true })
	defer blocked.close()
	for n := 0; n < 1000; n++ {
		blocked.push(&videoFrame{packets: []*rtp.Packet{videoPacket(uint16(n), uint32(n), true, true, true)}, key: true, at: time.Now()})
	}
	if len(blocked.frames) > 2 {
		t.Fatal("unbounded video queue")
	}
	select {
	case <-blocked.done:
	case <-time.After(time.Second):
		t.Fatal("blocked video did not obey write deadline")
	}
}
func TestScreenFanoutIsStableAndBounded(t *testing.T) {
	s := &screenCall{maxViewers: 1, peers: map[string]*screenPeer{"a": {wants: true}, "b": {wants: true}, "c": {wants: false}}}
	if !screenAccepts(s, "a") || screenAccepts(s, "b") || screenAccepts(s, "c") {
		t.Fatal("viewer budget violated")
	}
}

func TestScreenStateObservesSenderQueueDepthAndWait(t *testing.T) {
	left, right := net.Pipe()
	receiver := newVideoPipe(right, func(*videoFrame) {}, func(byte) {}, func() {}, func() bool { return true }, func(int) bool { return true })
	defer receiver.close()
	entered := make(chan struct{}, 2)
	resume := make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(resume) }) }
	defer release()
	pipe := newVideoPipe(left, func(*videoFrame) {}, func(byte) {}, func() {}, func() bool {
		entered <- struct{}{}
		<-resume
		return true
	}, func(int) bool { return true })
	defer pipe.close()
	service := &Service{screen: &screenCall{
		parent: &call{}, publishing: true, maxViewers: 1,
		peers: map[string]*screenPeer{"viewer": {pipe: pipe, wants: true}},
	}}
	readDequeue := func() {
		t.Helper()
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("frame did not leave the send queue")
		}
	}
	var sequence uint16
	push := func() {
		sequence++
		packet := videoPacket(sequence, uint32(sequence)*3000, true, true, true)
		pipe.push(&videoFrame{packets: []*rtp.Packet{packet}, key: true, at: time.Now()})
	}
	if queue := service.ScreenState().SendQueue; queue == nil || queue.Depth != 0 || queue.Capacity != 2 || queue.LastWaitMs != nil {
		t.Fatalf("queue before first sample = %+v", queue)
	}
	push()
	readDequeue()
	push()
	if queue := service.ScreenState().SendQueue; queue == nil || queue.Depth != 1 || queue.LastWaitMs == nil {
		t.Fatalf("queue with one waiting frame = %+v", queue)
	}
	time.Sleep(20 * time.Millisecond)
	release()
	readDequeue()
	if queue := service.ScreenState().SendQueue; queue == nil || queue.Depth != 0 || queue.LastWaitMs == nil || *queue.LastWaitMs < 20 {
		t.Fatalf("queue did not report observed wait: %+v", queue)
	}
	pipe.close()
	if queue := service.ScreenState().SendQueue; queue != nil {
		t.Fatalf("closed sender leaked queue metrics: %+v", queue)
	}
}

func testVP8Payload(first, key bool, payload byte) []byte {
	if first && key {
		return []byte{0, 0, 0, 0x9d, 1, 0x2a, 0x40, 1, 0xb4, 0, 3}
	}
	return []byte{payload, 2, 3}
}
func TestVideoRejectsOversizedDecodedSurface(t *testing.T) {
	p := videoPacket(1, 1, true, true, true)
	p.Payload[7] = 0xff
	p.Payload[8] = 0x3f
	a := videoAssembler{}
	if f, dropped := a.push(p); f != nil || !dropped {
		t.Fatal("oversized encoded surface accepted")
	}
	if safeVP8Keyframe([]byte{0, 0, 0}) {
		t.Fatal("truncated dimensions accepted")
	}
}
func TestVideoDeliveryWindowWaitsForCumulativeACK(t *testing.T) {
	p := &videoPipe{pending: map[uint32]time.Time{}, done: make(chan struct{})}
	for n := uint32(1); n <= 3; n++ {
		if !p.reserveFrame(n) {
			t.Fatal("window closed early")
		}
	}
	if p.reserveFrame(4) {
		t.Fatal("unbounded unacknowledged video")
	}
	time.Sleep(time.Millisecond)
	p.acknowledge(2)
	if len(p.pending) != 1 || p.rtt.Load() <= 0 {
		t.Fatal("cumulative delivery ACK not applied")
	}
	if !p.reserveFrame(4) {
		t.Fatal("delivery window did not recover")
	}
}
func TestRoomLeaveClosesOnlyItsScreenGeneration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &call{ctx: ctx, cancel: cancel}
	scCtx, scCancel := context.WithCancel(ctx)
	sc := &screenCall{parent: c, ctx: scCtx, cancel: scCancel, peers: map[string]*screenPeer{}}
	s := &Service{current: c, screen: sc}
	s.stop(c, "left")
	if s.screen != nil || s.current != nil {
		t.Fatal("room leave retained screen")
	}
	select {
	case <-sc.ctx.Done():
	default:
		t.Fatal("screen context leaked")
	}
	newCtx, newCancel := context.WithCancel(context.Background())
	defer newCancel()
	next := &call{ctx: newCtx, cancel: newCancel}
	s.current = next
	s.screen = &screenCall{parent: next, ctx: newCtx, cancel: newCancel, peers: map[string]*screenPeer{}}
	s.stop(c, "late old stop")
	if s.screen == nil || s.current != next {
		t.Fatal("old room stopped a new screen session")
	}
	s.ScreenStop()
}
func TestOldVoiceControlsCannotUnmuteNewSession(t *testing.T) {
	s := &Service{current: &call{session: "new"}}
	s.current.muted.Store(true)
	if e := s.ControlsForSession("old", false, false); e == nil || !s.current.muted.Load() {
		t.Fatal("old control changed new capture")
	}
	if e := s.ControlsForSession("new", false, true); e != nil || s.current.muted.Load() || !s.current.deaf.Load() {
		t.Fatal("current controls rejected", e)
	}
}

func TestVideoWindowTracksRTTAndBoundsBytes(t *testing.T) {
	p := &videoPipe{pending: map[uint32]time.Time{}, done: make(chan struct{})}
	p.rtt.Store(int64(100 * time.Millisecond))
	for n := uint32(1); n <= 8; n++ {
		if !p.reserveFrame(n, 1024) {
			t.Fatal("RTT pipeline closed early", n)
		}
	}
	if p.reserveFrame(9, 1024) {
		t.Fatal("RTT frame window unbounded")
	}
	p.acknowledge(8)
	if len(p.pendingBytes) != 0 {
		t.Fatal("ACK retained byte budget")
	}
	if !p.reserveFrame(10, 300<<10) || p.reserveFrame(11, 300<<10) {
		t.Fatal("in-flight byte budget incorrect")
	}
}

func TestVideoQueueDropsDependentFramesAndResumesAtKeyframe(t *testing.T) {
	pipe := &videoPipe{frames: make(chan queuedVideoFrame, 2), awaitKey: true, keyRequest: func() {}}
	pipe.state.Store(true)
	key := &videoFrame{key: true}
	delta := &videoFrame{}
	pipe.push(key)
	first := <-pipe.frames
	pipe.push(delta)
	pipe.push(delta)
	pipe.push(delta)
	if pipe.currentFrame(first.epoch) || len(pipe.frames) != 0 || pipe.queueDrops.Load() != 3 {
		t.Fatal("queue overflow did not invalidate the old keyframe and dependent frames")
	}
	pipe.push(delta)
	if len(pipe.frames) != 0 || pipe.keyWaitDrops.Load() != 1 {
		t.Fatal("reference frame queued before the replacement keyframe")
	}
	pipe.push(key)
	pipe.push(delta)
	newKey, newDelta := <-pipe.frames, <-pipe.frames
	if !newKey.frame.key || newDelta.frame.key || !pipe.currentFrame(newKey.epoch) || newKey.epoch != newDelta.epoch {
		t.Fatal("new reference chain was not queued in order")
	}
	pipe.abandonFrame(first.epoch)
	if !pipe.currentFrame(newDelta.epoch) {
		t.Fatal("a failed write from a previous epoch invalidated the new chain")
	}
}

func TestVideoYieldsForCurrentVoiceBackpressure(t *testing.T) {
	stream := &StreamMedia{out: make(chan queuedAudio, 3)}
	stream.state.Store("connected")
	stream.delay.Store(int64(70 * time.Millisecond))
	if voiceHasBackpressure(stream) {
		t.Fatal("a stale delay blocked video while voice is silent")
	}
	stream.out <- queuedAudio{}
	stream.out <- queuedAudio{}
	if !voiceHasBackpressure(stream) {
		t.Fatal("video failed to yield while voice has pending packets")
	}
	<-stream.out
	<-stream.out
	stream.state.Store("reconnecting")
	if !voiceHasBackpressure(stream) {
		t.Fatal("video failed to yield during voice reconnect")
	}
}
