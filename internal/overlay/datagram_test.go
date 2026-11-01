package overlay

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRealUserspaceUDP(t *testing.T) {
	dir := os.Getenv("P2P_ETCAT_TEST_DIR")
	if dir == "" {
		t.Skip("real community relay UDP test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	root, _ := filepath.Abs("../../data")
	os.MkdirAll(root, 0700)
	folder, e := os.MkdirTemp(root, "udp-media-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(folder)
	echo, e := net.ListenPacket("udp4", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer echo.Close()
	go func() {
		b := make([]byte, 1600)
		for {
			n, a, e := echo.ReadFrom(b)
			if e != nil {
				return
			}
			if n == 160 {
				b[159] ^= 0x5a
			}
			echo.WriteTo(b[:n], a)
		}
	}()
	tcp, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer tcp.Close()
	os.MkdirAll(filepath.Join(folder, "server"), 0700)
	os.MkdirAll(filepath.Join(folder, "client"), 0700)
	server, e := Open(ctx, filepath.Join(folder, "server"), filepath.Join(dir, "etcat-p2p.exe"), "")
	if e != nil {
		t.Fatal(e)
	}
	defer server.Close()
	if e = server.Start(ctx, tcp.Addr().(*net.TCPAddr).Port, echo.LocalAddr().(*net.UDPAddr).Port); e != nil {
		t.Fatal(e)
	}
	client, e := Open(ctx, filepath.Join(folder, "client"), filepath.Join(dir, "etcat-p2p.exe"), "")
	if e != nil {
		t.Fatal(e)
	}
	defer client.Close()
	k, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	id, _ := peer.IDFromPrivateKey(k)
	if e = client.Register(id.String(), server.Token()); e != nil {
		t.Fatal(e)
	}
	port, e := client.MediaForward(ctx, id.String(), echo.LocalAddr().(*net.UDPAddr).Port)
	if e != nil {
		t.Fatal(e)
	}
	socket, e := net.Dial("udp4", fmt.Sprintf("127.0.0.1:%d", port))
	if e != nil {
		t.Fatal(e)
	}
	defer socket.Close()
	started := time.Now()
	warmup := make([]byte, 160)
	binary.BigEndian.PutUint32(warmup, 0xffffffff)
	warmReply := make([]byte, 160)
	for {
		socket.SetDeadline(time.Now().Add(time.Second))
		socket.Write(warmup)
		n, e := socket.Read(warmReply)
		if e == nil && n == 160 && binary.BigEndian.Uint32(warmReply) == 0xffffffff && warmReply[159] == 0x5a {
			break
		}
		if time.Since(started) > 30*time.Second {
			t.Fatal("UDP route never became ready", e)
		}
	}
	t.Log("UDP establishment", time.Since(started))
	var total, jitter, last time.Duration
	lost := 0
	b := make([]byte, 160)
	reply := make([]byte, 160)
	for n := 0; n < 50; n++ {
		binary.BigEndian.PutUint32(b, uint32(n))
		start := time.Now()
		socket.SetDeadline(start.Add(time.Second))
		socket.Write(b)
		size, e := socket.Read(reply)
		for e == nil && (size != len(b) || (binary.BigEndian.Uint32(reply) != uint32(n) || reply[159] != 0x5a)) {
			size, e = socket.Read(reply)
		}
		if e != nil || size != len(b) || (binary.BigEndian.Uint32(reply) != uint32(n) || reply[159] != 0x5a) {
			lost++
			continue
		}
		rtt := time.Since(start)
		total += rtt
		if n > 0 {
			delta := rtt - last
			if delta < 0 {
				delta = -delta
			}
			jitter += (delta - jitter) / 16
		}
		last = rtt
		time.Sleep(20 * time.Millisecond)
	}
	if lost > 5 {
		t.Fatalf("UDP packet loss %d/50", lost)
	}
	t.Logf("real userspace UDP: frames=50 bytes/frame=160 lost=%d meanRTT=%s RTT-jitter=%s; one PC, not CGNAT, outer transport not asserted", lost, total/time.Duration(50-lost), jitter)
}
