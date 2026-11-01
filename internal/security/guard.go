package security

import (
	"sync"
	"time"
)

// Guard bounds replay state and application message frequency independently of transport limits.
type Guard struct {
	mu   sync.Mutex
	seen map[string]time.Time
	rate map[string]time.Time
}

func (g *Guard) Accept(sender, id string, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.seen == nil {
		g.seen = map[string]time.Time{}
		g.rate = map[string]time.Time{}
	}
	for k, t := range g.seen {
		if now.Sub(t) > 5*time.Minute {
			delete(g.seen, k)
		}
	}
	for k, t := range g.rate {
		if now.Sub(t) > time.Minute {
			delete(g.rate, k)
		}
	}
	key := sender + ":" + id
	if _, ok := g.seen[key]; ok {
		return false
	}
	_, known := g.rate[sender]
	if now.Sub(g.rate[sender]) < 100*time.Millisecond || len(g.seen) >= 4096 || (!known && len(g.rate) >= 256) {
		return false
	}
	g.seen[key] = now
	g.rate[sender] = now
	return true
}

// Retry permits a bounded acknowledgement of an already committed message.
// The caller must independently compare the complete immutable message digest.
func (g *Guard) Retry(sender, id string, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	at, ok := g.seen[sender+":"+id]
	if !ok || now.Sub(at) > 5*time.Minute || now.Sub(g.rate[sender]) < 100*time.Millisecond {
		return false
	}
	g.rate[sender] = now
	return true
}

// Forget permits retry after persistence failed; it preserves the rate limit.
func (g *Guard) Forget(sender, id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.seen, sender+":"+id)
}
