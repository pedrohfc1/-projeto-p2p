package roomprobe

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

type delayedProbeWriter struct{ net.Conn }

func (s delayedProbeWriter) Write(block []byte) (int, error) {
	time.Sleep(2 * time.Millisecond)
	return s.Conn.Write(block)
}

func TestRoomProbePacedEcho(t *testing.T) {
	client, server := net.Pipe()
	done := make(chan struct{})
	var valid atomic.Bool
	valid.Store(true)
	go func() {
		defer close(done)
		defer server.Close()
		serveRoomProbe(delayedProbeWriter{server}, valid.Load, 5*time.Millisecond, 15)
	}()
	defer func() {
		_ = client.Close()
		<-done
	}()
	const period = 10 * time.Millisecond
	result, err := measureRoomProbe(context.Background(), client, 90*time.Millisecond, period, valid.Load)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalBlocks < 2 || result.TotalBlocks > 9 || result.PayloadBytes != result.TotalBlocks*roomProbeBlockSize {
		t.Fatalf("wrong accounting: %+v", result)
	}
	if result.RTTMeanMs <= 0 || result.RTTP95Ms <= 0 || result.VariationMs == nil {
		t.Fatalf("missing measurements: %+v", result)
	}
}

func TestRoomProbeStopsOnRevokedReceiverApproval(t *testing.T) {
	client, server := net.Pipe()
	var valid atomic.Bool
	valid.Store(true)
	go func() {
		defer server.Close()
		serveRoomProbe(server, valid.Load, time.Millisecond, 100)
	}()
	_ = client.SetDeadline(time.Now().Add(time.Second))
	defer client.Close()
	var ack [1]byte
	if _, err := client.Read(ack[:]); err != nil {
		t.Fatal(err)
	}
	valid.Store(false)
	if err := writeProbeBlock(client, make([]byte, roomProbeBlockSize)); err == nil {
		if _, err := client.Read(ack[:]); err == nil {
			t.Fatal("receiver continued echoing after permission was revoked")
		}
	}
}
