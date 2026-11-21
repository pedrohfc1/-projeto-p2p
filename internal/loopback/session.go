package loopback

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"math"
	"sync"
	"time"
)

type Source struct {
	PID   int    `json:"pid"`
	Name  string `json:"name"`
	Title string `json:"title,omitempty"`
}
type Frame struct {
	PCM     string  `json:"pcm"`
	Level   float64 `json:"level"`
	Samples uint64  `json:"samples"`
	Dropped uint64  `json:"dropped"`
}
type Session struct {
	last             time.Time
	mu               sync.Mutex
	frames           [][]byte
	samples, dropped uint64
	level            float64
	err              error
	cancel           context.CancelFunc
	done             chan struct{}
	once             sync.Once
}

func (s *Session) push(p []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q := append([]byte(nil), p...)
	var energy float64
	for i := 0; i+1 < len(q); i += 2 {
		v := float64(int16(binary.LittleEndian.Uint16(q[i:]))) / 32768
		energy += v * v
	}
	s.last = time.Now()
	s.level = math.Sqrt(energy / float64(len(q)/2))
	s.samples += uint64(len(q) / 2)
	if len(s.frames) >= 3 {
		s.frames = s.frames[1:]
		s.dropped++
	}
	s.frames = append(s.frames, q)
}
func (s *Session) Read() (Frame, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return Frame{}, s.err
	}
	if time.Since(s.last) > 200*time.Millisecond {
		s.level = 0
	}
	f := Frame{Level: s.level, Samples: s.samples, Dropped: s.dropped}
	if len(s.frames) > 0 {
		f.PCM = base64.StdEncoding.EncodeToString(s.frames[0])
		s.frames = s.frames[1:]
	}
	return f, nil
}
func (s *Session) Close() {
	s.once.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
		if s.done != nil {
			<-s.done
		}
		s.mu.Lock()
		s.frames = nil
		s.mu.Unlock()
	})
}
func (s *Session) fail(e error) { s.mu.Lock(); s.err = e; s.mu.Unlock() }
func valid(mode string, pid int) error {
	if mode != "exclude" && mode != "process" && mode != "device" {
		return errors.New("fonte WASAPI inválida")
	}
	if mode == "process" && pid <= 0 {
		return errors.New("selecione um aplicativo")
	}
	return nil
}

func (s *Session) Alive() bool {
	if s.done == nil {
		return true
	}
	select {
	case <-s.done:
		return false
	default:
		return true
	}
}
