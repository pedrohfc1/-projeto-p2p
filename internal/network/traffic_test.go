package network

import (
	"context"
	"fmt"
	libnet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"io"
	"sync"
	"testing"
	"time"
)

func TestPerPeerTrafficBoundedIndependentAndConcurrent(t *testing.T) {
	c := &trafficCounter{}
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for n := 0; n < 100; n++ {
				c.LogSentMessageStream(10, "probe", "a")
				c.LogRecvMessageStream(5, "media", "b")
			}
		}()
	}
	group.Wait()
	if c.GetBandwidthForPeer("a").TotalOut != 8000 || c.GetBandwidthForPeer("b").TotalIn != 4000 {
		t.Fatal("per-peer counters mixed or lost")
	}
	if c.GetBandwidthTotals().TotalOut != 0 {
		t.Fatal("stream hooks double counted global bytes")
	}
	copy := c.GetBandwidthByPeer()
	delete(copy, "a")
	if c.GetBandwidthForPeer("a").TotalOut != 8000 {
		t.Fatal("mutable snapshot")
	}
	for i := 0; i < 300; i++ {
		c.LogSentMessageStream(1, "probe", peer.ID(fmt.Sprint(i)))
	}
	if len(c.GetBandwidthByPeer()) > 256 {
		t.Fatal("unbounded identities")
	}
	c.Reset()
	if len(c.GetBandwidthByPeer()) != 0 {
		t.Fatal("reset retained identities")
	}
}
func TestRealPeerTrafficCountsAuthenticatedStream(t *testing.T) {
	a := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer a.Close()
	b := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer b.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	info, e := ParseAddress(b.Addresses()[0])
	if e != nil {
		t.Fatal(e)
	}
	if e = a.Connect(ctx, info); e != nil {
		t.Fatal(e)
	}
	b.Host.SetStreamHandler("/qa/diagnostic-count/1", func(s libnet.Stream) { defer s.Close(); io.Copy(io.Discard, s) })
	stream, e := a.Host.NewStream(ctx, b.Host.ID(), "/qa/diagnostic-count/1")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = stream.Write([]byte("synthetic")); e != nil {
		t.Fatal(e)
	}
	stream.Close()
	wait(t, func() bool {
		return a.traffic.GetBandwidthForPeer(b.Host.ID()).TotalOut >= 9 && b.traffic.GetBandwidthForPeer(a.Host.ID()).TotalIn >= 9
	})
	snapshot := a.Traffic().Peers[b.Host.ID().String()]
	if snapshot.UploadBytes == nil || *snapshot.UploadBytes < 9 || snapshot.Opened == 0 {
		t.Fatal(snapshot)
	}
}
