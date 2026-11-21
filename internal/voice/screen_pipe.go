package voice

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

type queuedVideoFrame struct {
	frame    *videoFrame
	queuedAt time.Time
	epoch    uint64
}

type videoPipe struct {
	queueDrops, keyWaitDrops, lateDrops, windowDrops, budgetDrops atomic.Uint64
	ack                                                           chan uint32
	pendingMu                                                     sync.Mutex
	pending                                                       map[uint32]time.Time
	pendingBytes                                                  map[uint32]int
	rtt                                                           atomic.Int64
	lastQueueWait                                                 atomic.Int64 // Nanoseconds plus one. Zero means no frame dequeued.
	conn                                                          MediaConn
	frames                                                        chan queuedVideoFrame
	statusSignal                                                  chan byte
	keySignal                                                     chan struct{}
	done                                                          chan struct{}
	once                                                          sync.Once
	state                                                         atomic.Bool
	closedReason                                                  atomic.Value
	sendMu                                                        sync.Mutex
	sendEpoch                                                     uint64
	awaitKey                                                      bool
	lastKeyRequest                                                atomic.Int64
	in, out, drops, bytesIn, bytesOut                             atomic.Uint64
	keyRequest                                                    func()
	permit                                                        func() bool
	budget                                                        func(int) bool
}

func (p *videoPipe) close() {
	p.closeFor("stopped")
}

func (p *videoPipe) closeFor(reason string) {
	p.once.Do(func() {
		p.closedReason.Store(reason)
		p.state.Store(false)
		close(p.done)
		p.conn.Close()
	})
}
func (p *videoPipe) announce(flags byte) {
	select {
	case p.statusSignal <- flags:
	default:
		select {
		case <-p.statusSignal:
		default:
		}
		select {
		case p.statusSignal <- flags:
		default:
		}
	}
}
func (p *videoPipe) keyframe() {
	select {
	case p.keySignal <- struct{}{}:
	default:
	}
}

// A missed reference invalidates all later frames queued with it. The epoch
// also fences a dequeued frame from a newly queued recovery keyframe.
func (p *videoPipe) invalidateFramesLocked() {
	p.sendEpoch++
	p.awaitKey = true
	for {
		select {
		case <-p.frames:
			p.queueDrops.Add(1)
			p.drops.Add(1)
		default:
			return
		}
	}
}

func (p *videoPipe) requestPublisherKey() {
	now := time.Now().UnixMilli()
	previous := p.lastKeyRequest.Load()
	if now-previous >= 500 && p.lastKeyRequest.CompareAndSwap(previous, now) {
		p.keyRequest()
	}
}

func (p *videoPipe) resetFrames() {
	p.sendMu.Lock()
	p.invalidateFramesLocked()
	p.sendMu.Unlock()
}

func (p *videoPipe) abandonFrame(epoch uint64) {
	p.sendMu.Lock()
	if p.sendEpoch == epoch {
		p.invalidateFramesLocked()
	}
	p.sendMu.Unlock()
	p.requestPublisherKey()
}

func (p *videoPipe) currentFrame(epoch uint64) bool {
	p.sendMu.Lock()
	defer p.sendMu.Unlock()
	return p.sendEpoch == epoch
}

func (p *videoPipe) push(f *videoFrame) {
	if !p.state.Load() {
		return
	}
	p.sendMu.Lock()
	if p.awaitKey && !f.key {
		p.sendMu.Unlock()
		p.keyWaitDrops.Add(1)
		p.drops.Add(1)
		p.requestPublisherKey()
		return
	}
	overflow := len(p.frames) == cap(p.frames)
	if overflow {
		p.invalidateFramesLocked()
		if !f.key {
			p.sendMu.Unlock()
			p.queueDrops.Add(1)
			p.drops.Add(1)
			p.requestPublisherKey()
			return
		}
	}
	// The writer validates this epoch again before committing the frame.
	// Queueing a keyframe safely admits its following references in FIFO order.
	p.frames <- queuedVideoFrame{frame: f, queuedAt: time.Now(), epoch: p.sendEpoch}
	p.awaitKey = false
	p.sendMu.Unlock()
}
func newVideoPipe(conn MediaConn, receive func(*videoFrame), status func(byte), key func(), permit func() bool, budget func(int) bool) *videoPipe {
	p := &videoPipe{ack: make(chan uint32, 1), pending: map[uint32]time.Time{}, pendingBytes: map[uint32]int{}, conn: conn, frames: make(chan queuedVideoFrame, 2), statusSignal: make(chan byte, 1), keySignal: make(chan struct{}, 1), done: make(chan struct{}), keyRequest: key, permit: permit, budget: budget, awaitKey: true}
	p.state.Store(true)
	go func() {
		reason := "write_interrupted"
		defer func() { p.closeFor(reason) }()
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			var data []byte
			select {
			case <-p.done:
				return
			case stamp := <-p.ack:
				data = make([]byte, 5)
				data[0] = 4
				binary.BigEndian.PutUint32(data[1:], stamp)
			case flags := <-p.statusSignal:
				data = []byte{2, flags}
			case <-p.keySignal:
				data = []byte{3}
			case <-tick.C:
				if p.ackExpired(time.Now()) {
					reason = "ack_timeout"
					return
				}
				data = []byte{0}
			case queued := <-p.frames:
				p.lastQueueWait.Store(time.Since(queued.queuedAt).Nanoseconds() + 1)
				f := queued.frame
				if !p.currentFrame(queued.epoch) {
					p.queueDrops.Add(1)
					p.drops.Add(1)
					continue
				}
				if !p.permit() || time.Since(f.at) > 150*time.Millisecond {
					p.lateDrops.Add(1)
					p.drops.Add(1)
					p.abandonFrame(queued.epoch)
					continue
				}
				frameBytes := f.wireSize()
				if !p.reserveFrame(f.packets[0].Timestamp, frameBytes) {
					p.windowDrops.Add(1)
					p.drops.Add(1)
					p.abandonFrame(queued.epoch)
					continue
				}
				if !p.budget(frameBytes) {
					p.budgetDrops.Add(1)
					p.pendingMu.Lock()
					delete(p.pending, f.packets[0].Timestamp)
					delete(p.pendingBytes, f.packets[0].Timestamp)
					p.pendingMu.Unlock()
					p.drops.Add(1)
					p.abandonFrame(queued.epoch)
					continue
				}
				var e error
				data, e = f.wirePayload()
				if e != nil {
					reason = "invalid_local_frame"
					return
				}
			}
			// The payload is shared by all viewers. A separate size prefix
			// avoids cloning that payload per stream. Only this goroutine writes.
			var header [4]byte
			binary.BigEndian.PutUint32(header[:], uint32(len(data)))
			if conn.SetWriteDeadline(time.Now().Add(200*time.Millisecond)) != nil || !writeVideoBytes(conn, header[:]) || !writeVideoBytes(conn, data) {
				return
			}
			if len(data) > 2 && data[0] == 1 {
				p.out.Add(1)
				p.bytesOut.Add(uint64(len(data)))
			}
		}
	}()
	go func() {
		reason := "read_interrupted"
		defer func() {
			p.closeFor(reason) // Publish disconnection before attempting the service lock.
			status(0x80)
		}()
		var offset int64
		first := true
		needsKey := true
		window := time.Now()
		frames := 0
		bytesWindow := 0
		messages := 0
		for {
			reason = "read_interrupted"
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			var header [4]byte
			if _, e := io.ReadFull(conn, header[:]); e != nil {
				return
			}
			n := int(binary.BigEndian.Uint32(header[:]))
			reason = "protocol_error"
			if n < 1 || n > maxVideoFrame+2048 {
				return
			}
			data := make([]byte, n)
			if _, e := io.ReadFull(conn, data); e != nil {
				reason = "read_interrupted"
				return
			}
			if time.Since(window) > time.Second {
				window = time.Now()
				frames = 0
				bytesWindow = 0
				messages = 0
			}
			messages++
			if messages > 200 {
				reason = "receive_rate_limit"
				return
			}
			switch data[0] {
			case 0:
				if n != 1 {
					return
				}
				continue
			case 2:
				if n != 2 {
					return
				}
				status(data[1])
				continue
			case 4:
				if n != 5 {
					return
				}
				p.acknowledge(binary.BigEndian.Uint32(data[1:]))
				continue
			case 3:
				if n != 1 {
					return
				}
				key()
				continue
			case 1:
			default:
				return
			}
			if time.Since(window) > time.Second {
				window = time.Now()
				frames = 0
				bytesWindow = 0
			}
			frames++
			bytesWindow += n
			if frames > 90 || bytesWindow > 2000000 {
				reason = "receive_rate_limit"
				return
			}
			f, stamp, e := decodeVideoFrame(data)
			if e != nil {
				return
			}
			p.queueAck(f.packets[0].Timestamp)
			transit := time.Now().UnixMilli() - stamp
			if first || transit < offset {
				offset = transit
				first = false
			}
			if transit-offset > 200 {
				p.drops.Add(1)
				needsKey = true
				p.keyframe()
				continue
			}
			if needsKey && !f.key {
				p.drops.Add(1)
				p.keyframe()
				continue
			}
			if f.key {
				needsKey = false
			}
			p.in.Add(1)
			p.bytesIn.Add(uint64(n))
			receive(f)
		}
	}()
	return p
}

func writeVideoBytes(writer io.Writer, data []byte) bool {
	n, err := writer.Write(data)
	return err == nil && n == len(data)
}

var errScreenUnavailable = errors.New("screen session unavailable or unauthorized")

func (p *videoPipe) ackExpired(now time.Time) bool {
	p.pendingMu.Lock()
	defer p.pendingMu.Unlock()
	return p.ackExpiredLocked(now)
}

func (p *videoPipe) ackExpiredLocked(now time.Time) bool {
	for _, at := range p.pending {
		if now.Sub(at) > time.Second {
			return true
		}
	}
	return false
}

// An explicit delivery window bounds data accepted by TCP before receiver ACKs.
// Dropped complete frames trigger a new keyframe rather than replaying references.
func (p *videoPipe) reserveFrame(stamp uint32, sizes ...int) bool {
	p.pendingMu.Lock()
	defer p.pendingMu.Unlock()
	now := time.Now()
	if p.ackExpiredLocked(now) {
		return false
	}
	// Pipeline enough frames for measured RTT without unbounded latency or buffering.
	window := 3
	if rtt := time.Duration(p.rtt.Load()); rtt > 0 {
		window = min(16, max(3, int(rtt/(time.Second/60))+2))
	}
	if len(p.pending) >= window {
		return false
	}
	size := 0
	if len(sizes) > 0 {
		size = sizes[0]
	}
	inFlight := size
	for _, bytes := range p.pendingBytes {
		inFlight += bytes
	}
	if inFlight > 512<<10 {
		return false
	}
	if p.pendingBytes == nil {
		p.pendingBytes = map[uint32]int{}
	}
	p.pendingBytes[stamp] = size
	p.pending[stamp] = now
	return true
}
func (p *videoPipe) acknowledge(stamp uint32) {
	p.pendingMu.Lock()
	defer p.pendingMu.Unlock()
	now := time.Now()
	for seq, at := range p.pending {
		if int32(stamp-seq) >= 0 {
			if seq == stamp {
				p.rtt.Store(int64(now.Sub(at)))
			}
			delete(p.pending, seq)
			delete(p.pendingBytes, seq)
		}
	}
}
func (p *videoPipe) queueAck(stamp uint32) {
	select {
	case p.ack <- stamp:
	default:
		select {
		case <-p.ack:
		default:
		}
		select {
		case p.ack <- stamp:
		default:
		}
	}
}
