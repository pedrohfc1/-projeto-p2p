package network

import (
	"bufio"
	"bytes"
	libnet "github.com/libp2p/go-libp2p/core/network"
	"io"
	"projetop2p/internal/voice"
	"testing"
	"time"
)

func TestMediaHandshakeKeepsCoalescedAudio(t *testing.T) {
	raw := append([]byte("{}\n"), []byte{0, 3, 7, 8, 9}...)
	r := bufio.NewReader(bytes.NewReader(raw))
	var response voice.Response
	if e := readVoiceMediaFrame(r, 1024, &response); e != nil {
		t.Fatal(e)
	}
	audio, _ := io.ReadAll(r)
	if !bytes.Equal(audio, []byte{0, 3, 7, 8, 9}) {
		t.Fatal("audio consumed by handshake", audio)
	}
}
func TestMediaHandshakeBounds(t *testing.T) {
	for _, raw := range []string{"{\"unknown\":1}\n", string(bytes.Repeat([]byte{'x'}, 1025)), "{} {}\n"} {
		var res voice.Response
		if readVoiceMediaFrame(bufio.NewReader(bytes.NewBufferString(raw)), 1024, &res) == nil {
			t.Fatal("invalid handshake accepted")
		}
	}
}

type deadlineTestStream struct {
	libnet.Stream
	called chan struct{}
}

func (s *deadlineTestStream) SetWriteDeadline(time.Time) error { close(s.called); return nil }
func TestMediaDeadlineCannotInterruptHandshake(t *testing.T) {
	f := &deadlineTestStream{called: make(chan struct{})}
	c := &mediaConn{Stream: f, ready: make(chan struct{}), done: make(chan struct{})}
	done := make(chan struct{})
	go func() { _ = c.SetWriteDeadline(time.Now().Add(time.Second)); close(done) }()
	select {
	case <-f.called:
		t.Fatal("media deadline applied before acknowledgement")
	case <-time.After(20 * time.Millisecond):
	}
	close(c.ready)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("media deadline stayed blocked")
	}
}
