package diagnostics

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBoundedCursorClearAndConcurrentWriters(t *testing.T) {
	var r Ring
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				r.Add("INFO", "voice", "test", fmt.Sprint(i))
				r.Read(0)
			}
		}()
	}
	wg.Wait()
	count := 0
	var cursor uint64
	for {
		b := r.Read(cursor)
		if len(b.Events) == 0 {
			break
		}
		for _, e := range b.Events {
			if e.ID <= cursor {
				t.Fatal("non-monotonic")
			}
			cursor = e.ID
		}
		count += len(b.Events)
	}
	if count != Capacity {
		t.Fatal(count)
	}
	if r.Read(0).Dropped != 6000 {
		t.Fatal("overrun not reported")
	}
	at := r.Clear()
	if len(r.Read(0).Events) != 0 {
		t.Fatal("clear")
	}
	r.Add("ERROR", "voice", "test", "failure")
	if b := r.Read(at); len(b.Events) != 1 || b.Events[0].ID != at+1 {
		t.Fatal(b)
	}
}
func TestRedactionBoundAndNoPayloadCapture(t *testing.T) {
	s := Redact("token=supersecret Bearer xyz peetwopee://join/abc 192.168.1.2 12D3KooWabcdefghijklmnop " + strings.Repeat("q", 3000))
	for _, secret := range []string{"supersecret", "Bearer xyz", "join/abc", "192.168.1.2", "abcdefghijklmnop"} {
		if strings.Contains(s, secret) {
			t.Fatal(secret)
		}
	}
	if len([]rune(s)) > 769 {
		t.Fatal("unbounded")
	}
	Local.Clear()
	h := Handler{slog.NewTextHandler(io.Discard, nil)}
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "private chat contents", 0)
	r.Add("text", "private message")
	_ = h.Handle(context.Background(), r)
	if len(Local.Read(0).Events) != 0 {
		t.Fatal("arbitrary contents captured")
	}
	r = slog.NewRecord(time.Now(), slog.LevelWarn, "voice_transport", 0)
	r.Add("state", "reconnecting", "token", "never-visible", "reason", "write_timeout_or_closed")
	_ = h.Handle(context.Background(), r)
	b := Local.Read(0)
	if len(b.Events) != 1 || b.Events[0].Level != "WARN" || strings.Contains(b.Events[0].Message, "never-visible") {
		t.Fatal(b)
	}
}

func TestMediaWriterDoesNotWaitForReader(t *testing.T) {
	var r Ring
	r.mu.Lock()
	done := make(chan bool, 1)
	go func() { done <- r.TryAdd("INFO", "voice", "stream", "connected") }()
	select {
	case ok := <-done:
		if ok {
			t.Fatal("wrote through held lock")
		}
	case <-time.After(time.Second):
		t.Fatal("blocked media")
	}
	r.mu.Unlock()
	if !r.TryAdd("INFO", "voice", "stream", "connected") {
		t.Fatal("lost uncontended event")
	}
}

func TestInstallDoesNotRecurseThroughStandardLog(t *testing.T) {
	restore := Install()
	defer restore()
	Local.Clear()
	done := make(chan struct{})
	go func() {
		slog.Info("voice_transport", "state", "connected")
		log.Print("diagnostic bridge regression test")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("recursive standard logger bridge blocked")
	}
	b := Local.Read(0)
	if len(b.Events) != 1 || !strings.Contains(b.Events[0].Message, "connected") {
		t.Fatal(b)
	}
}

func TestScreenAudioDiagnosticsRemainSeparateAndRedacted(t *testing.T) {
	Local.Clear()
	h := Handler{Next: slog.NewTextHandler(io.Discard, nil)}
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "screen_audio_network", 0)
	r.Add("peer", "abc123abc123", "sent", 10, "received", 8, "encoded", 11, "pcm", "never-capture-content", "token", "never-secret")
	_ = h.Handle(context.Background(), r)
	b := Local.Read(0)
	if len(b.Events) != 1 || b.Events[0].Category != "video" || !strings.Contains(b.Events[0].Message, "peer=abc123abc123") || strings.Contains(b.Events[0].Message, "never-") {
		t.Fatal(b)
	}
}
