package security

import (
	"sync"
	"time"
)

type streamPeer struct {
	active int
	tokens float64
	last   time.Time
}
type StreamGate struct {
	mu     sync.Mutex
	active int
	peers  map[string]*streamPeer
}

func (g *StreamGate) Acquire(id string, now time.Time) (func(), bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.peers == nil {
		g.peers = map[string]*streamPeer{}
	}
	for k, p := range g.peers {
		if p.active == 0 && now.Sub(p.last) > time.Minute {
			delete(g.peers, k)
		}
	}
	p := g.peers[id]
	if p == nil {
		if len(g.peers) >= 256 {
			return nil, false
		}
		p = &streamPeer{tokens: 64, last: now}
		g.peers[id] = p
	}
	p.tokens += now.Sub(p.last).Seconds() * 12
	if p.tokens > 64 {
		p.tokens = 64
	}
	p.last = now
	if g.active >= 16 || p.active >= 2 || p.tokens < 1 {
		return nil, false
	}
	g.active++
	p.active++
	p.tokens--
	var once sync.Once
	return func() { once.Do(func() { g.mu.Lock(); g.active--; p.active--; g.mu.Unlock() }) }, true
}
