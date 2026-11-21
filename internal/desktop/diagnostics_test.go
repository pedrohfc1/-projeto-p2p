package desktop

import (
	"projetop2p/internal/diagnostics"
	"strings"
	"testing"
)

// Enable the production collector while the optional real media test runs.
func captureMediaDiagnostics(t *testing.T) {
	restore := diagnostics.Install()
	diagnostics.Local.Clear()
	t.Cleanup(func() {
		restore()
		if t.Failed() {
			return
		}
		batch := diagnostics.Local.Read(0)
		found := false
		for _, e := range batch.Events {
			if e.Component == "backend/libp2p" && strings.Contains(e.Message, "voice_media_counters") && !strings.Contains(e.Message, "sent=0 ") {
				found = true
			}
		}
		if !found {
			t.Error("production diagnostic collector did not receive real audio counters during video")
		}
		t.Log("production local diagnostic collector active during actual encrypted voice/video; bounded operational records, no payload capture")
	})
}
