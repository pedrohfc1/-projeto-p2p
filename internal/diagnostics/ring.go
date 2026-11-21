// Package diagnostics keeps only bounded, local, redacted operational events.
package diagnostics

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const Capacity = 2000

type Event struct {
	ID        uint64 `json:"id"`
	Time      int64  `json:"time"`
	Level     string `json:"level"`
	Category  string `json:"category"`
	Component string `json:"component"`
	Message   string `json:"message"`
}
type Batch struct {
	Events  []Event `json:"events"`
	Last    uint64  `json:"last"`
	Dropped uint64  `json:"dropped"`
}
type Ring struct {
	mu     sync.Mutex
	events [Capacity]Event
	next   uint64
	size   int
}

var Local Ring
var sensitive = regexp.MustCompile(`(?i)(?:bearer\s+\S+|(?:token|secret|password|private.?key|invite|sdp|credential)["']?\s*[:=]\s*["']?[^\s,;]+|(?:peetwopee|projetop2p)://\S+|(?:12D3KooW|Qm)[A-Za-z0-9]+|[A-Za-z0-9_+/=-]{40,})`)
var address = regexp.MustCompile(`(?:\d{1,3}\.){3}\d{1,3}|\[[0-9a-fA-F:]+\]`)

func Redact(s string) string {
	if len(s) > 4096 {
		s = s[:4096]
	}
	s = sensitive.ReplaceAllString(s, "[oculto]")
	s = address.ReplaceAllString(s, "[endereço]")
	s = strings.Map(func(r rune) rune {
		if r < ' ' && r != '\t' {
			return ' '
		}
		return r
	}, s)
	r := []rune(s)
	if len(r) > 768 {
		s = string(r[:768]) + "…"
	}
	return s
}
func (r *Ring) Add(level, category, component, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.addLocked(level, category, component, message)
}

// TryAdd never waits for the ring lock on the media logging path.
func (r *Ring) TryAdd(level, category, component, message string) bool {
	if !r.mu.TryLock() {
		return false
	}
	defer r.mu.Unlock()
	r.addLocked(level, category, component, message)
	return true
}
func (r *Ring) addLocked(level, category, component, message string) {
	r.next++
	r.events[(r.next-1)%Capacity] = Event{r.next, time.Now().UnixMilli(), level, category, Redact(component), Redact(message)}
	if r.size < Capacity {
		r.size++
	}
}
func (r *Ring) Read(after uint64) Batch {
	r.mu.Lock()
	defer r.mu.Unlock()
	if after > r.next {
		after = r.next
	}
	b := Batch{Events: []Event{}, Last: after}
	first := r.next - uint64(r.size) + 1
	if after+1 < first {
		b.Dropped = first - after - 1
	}
	for n := first; n <= r.next && len(b.Events) < 250; n++ {
		if n > after {
			b.Events = append(b.Events, r.events[(n-1)%Capacity])
			b.Last = n
		}
	}
	return b
}
func (r *Ring) Clear() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = [Capacity]Event{}
	r.size = 0
	return r.next
}

// Only fixed operational slog names/attributes are captured, never arbitrary messages.
type Handler struct{ Next slog.Handler }

func (h Handler) Enabled(ctx context.Context, l slog.Level) bool {
	return l >= slog.LevelInfo || h.Next.Enabled(ctx, l)
}
func (h Handler) Handle(ctx context.Context, r slog.Record) error {
	var profileEvent bool
	switch r.Message {
	case "profile_updated", "avatar_changed", "avatar_asset_requested", "avatar_asset_received", "avatar_hash_mismatch", "community_icon_updated":
		profileEvent = true
	}
	if profileEvent || r.Message == "voice_authorization" || r.Message == "voice_transport" || r.Message == "voice_media_counters" || r.Message == "community_connectivity" || r.Message == "screen_audio_capture" || r.Message == "screen_audio_network" {
		parts := []string{r.Message}
		screenAudio := strings.HasPrefix(r.Message, "screen_audio_")
		r.Attrs(func(a slog.Attr) bool {
			if profileEvent {
				if a.Key == "revision" || a.Key == "size" {
					parts = append(parts, a.Key+"="+fmt.Sprint(a.Value.Any()))
				}
				return true
			}
			if a.Key == "flow" && strings.HasPrefix(fmt.Sprint(a.Value.Any()), "screen-audio/") {
				screenAudio = true
			}
			switch a.Key {
			case "session", "flow", "source_session", "target_session", "stream", "origin", "component", "state", "transport", "reason", "sent", "received", "discarded", "queue_ms", "path", "endpoints", "result", "peer", "source", "encoded", "codec", "samples", "level", "dropped":
				parts = append(parts, a.Key+"="+fmt.Sprint(a.Value.Any()))
			}
			return true
		})
		level := "INFO"
		if r.Level >= slog.LevelError {
			level = "ERROR"
		} else if r.Level >= slog.LevelWarn {
			level = "WARN"
		}
		category := "voice"
		if screenAudio {
			category = "video"
		}
		if profileEvent || r.Message == "community_connectivity" {
			category = "network"
		}
		component := "backend/libp2p"
		if profileEvent {
			component = "backend/perfil"
		}
		Local.TryAdd(level, category, component, strings.Join(parts, " "))
	}
	if h.Next.Enabled(ctx, r.Level) {
		return h.Next.Handle(ctx, r)
	}
	return nil
}
func (h Handler) WithAttrs(a []slog.Attr) slog.Handler { return Handler{h.Next.WithAttrs(a)} }
func (h Handler) WithGroup(g string) slog.Handler      { return Handler{h.Next.WithGroup(g)} }

// Install uses a direct sink: wrapping slog's defaultHandler while SetDefault
// bridges the standard logger would recurse through log.Output and deadlock.
func Install() func() {
	previous := slog.Default()
	writer := log.Writer()
	flags := log.Flags()
	prefix := log.Prefix()
	slog.SetDefault(slog.New(Handler{Next: slog.NewTextHandler(os.Stderr, nil)}))
	return func() { slog.SetDefault(previous); log.SetOutput(writer); log.SetFlags(flags); log.SetPrefix(prefix) }
}
