package desktop

import (
	"context"
	"projetop2p/internal/voice"
	"testing"
	"time"
)

func TestMediaLivenessDoesNotWaitForCoreUIPoll(t *testing.T) {
	c := &Core{ctx: context.Background()}
	v := &voice.Service{}
	c.media.Store(v)
	c.mu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		got, e := c.voiceService()
		if e != nil || got != v {
			t.Error("media handle lost")
		}
		c.ScreenHeartbeat()
	}()
	select {
	case <-done:
		c.mu.Unlock()
	case <-time.After(200 * time.Millisecond):
		c.mu.Unlock()
		<-done
		t.Fatal("UI/status mutex blocked media heartbeat")
	}
}
