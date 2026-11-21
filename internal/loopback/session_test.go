package loopback

import (
	"encoding/base64"
	"testing"
)

func TestPCMQueueBoundedAndNoPersistence(t *testing.T) {
	s := &Session{}
	for i := 0; i < 100; i++ {
		s.push(make([]byte, 1920))
	}
	if len(s.frames) != 3 || s.dropped != 97 {
		t.Fatal("unbounded queue")
	}
	f, e := s.Read()
	b, _ := base64.StdEncoding.DecodeString(f.PCM)
	if e != nil || len(b) != 1920 || f.Samples != 96000 || f.Level != 0 {
		t.Fatal(f, e)
	}
}
func TestSourceRequiresExplicitChoice(t *testing.T) {
	for _, m := range []string{"", "microphone", "auto"} {
		if valid(m, 0) == nil {
			t.Fatal("implicit source accepted")
		}
	}
	if valid("process", 0) == nil {
		t.Fatal("missing process accepted")
	}
}
