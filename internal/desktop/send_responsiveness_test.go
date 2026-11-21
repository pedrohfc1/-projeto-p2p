package desktop

import (
	"encoding/json"
	net "github.com/libp2p/go-libp2p/core/network"
	"projetop2p/internal/messaging"
	"projetop2p/internal/network"
	"testing"
	"time"
)

func TestPendingChatAcknowledgementDoesNotBlockAudioControlsOrState(t *testing.T) {
	a := communityCore(t, t.TempDir(), "a", freePort(t))
	defer a.Close()
	b := communityCore(t, t.TempDir(), "b", freePort(t))
	defer b.Close()
	ai, _ := a.Invite()
	bi, _ := b.Invite()
	if e := a.AddContact(bi); e != nil {
		t.Fatal(e)
	}
	if e := b.AddContact(ai); e != nil {
		t.Fatal(e)
	}
	received, release := make(chan struct{}), make(chan struct{})
	b.node.Host.SetStreamHandler(network.Protocol, func(stream net.Stream) {
		defer stream.Close()
		var m messaging.Message
		if json.NewDecoder(stream).Decode(&m) != nil {
			return
		}
		close(received)
		<-release
		stream.Write([]byte("OK\n"))
	})
	done := make(chan error, 1)
	go func() { done <- a.Send(b.node.Host.ID().String(), "delayed acknowledgement fixture") }()
	defer func() {
		close(release)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("send did not finish after releasing acknowledgement")
		}
	}()
	select {
	case <-received:
	case e := <-done:
		t.Fatal(e)
	case <-time.After(8 * time.Second):
		t.Fatal("no pending message")
	}
	state := make(chan struct{})
	go func() { _ = a.VoiceState(); _, _ = a.State(""); close(state) }()
	select {
	case <-state:
	case <-time.After(time.Second):
		t.Fatal("pending chat blocked audio/network controls")
	}
}
