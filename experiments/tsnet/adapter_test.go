package main

import (
	"context"
	"crypto/rand"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ma "github.com/multiformats/go-multiaddr"
	"net"
	"testing"
	"time"
)

type isolatedStack struct{ address string }

func (s isolatedStack) Dial(ctx context.Context, _, _ string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "tcp", s.address)
}

// Real Noise/libp2p over the listener adapter, but a loopback fake underlay. This is NOT a tailnet/authentication test.
func TestNoiseIdentityThroughUserspaceAdapter(t *testing.T) {
	aKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	bKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	aID, _ := peer.IDFromPrivateKey(aKey)
	bID, _ := peer.IDFromPrivateKey(bKey)
	b, e := libp2p.New(libp2p.Identity(bKey), libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"), libp2p.NoTransports, libp2p.Transport(tcp.NewTCPTransport), libp2p.DisableRelay(), libp2p.Security(noise.ID, noise.New), libp2p.ConnectionGater(gate{aID}))
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	localPort, _ := b.Network().ListenAddresses()[0].ValueForProtocol(ma.P_TCP)
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stop := bridge(ctx, listener, "127.0.0.1:"+localPort)
	defer stop()
	makeClient := func(key crypto.PrivKey) (peer.ID, error) {
		a, e := libp2p.New(libp2p.Identity(key), libp2p.NoListenAddrs, libp2p.NoTransports, libp2p.Transport(tcp.NewTCPTransport, tcp.WithDialerForAddr(func(ma.Multiaddr) (tcp.ContextDialer, error) {
			return tailDialer{isolatedStack{listener.Addr().String()}}, nil
		})), libp2p.DisableRelay(), libp2p.Security(noise.ID, noise.New), libp2p.ConnectionGater(gate{bID}))
		if e != nil {
			return "", e
		}
		defer a.Close()
		remote, _ := ma.NewMultiaddr("/ip4/100.80.0.2/tcp/7447")
		if e = a.Connect(ctx, peer.AddrInfo{ID: bID, Addrs: []ma.Multiaddr{remote}}); e != nil {
			return "", e
		}
		result := <-ping.Ping(ctx, a, bID)
		return a.ID(), result.Error
	}
	got, e := makeClient(aKey)
	if e != nil || got != aID {
		t.Fatal("authorized Noise identity", got, e)
	}
	wrong, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	if _, e = makeClient(wrong); e == nil {
		t.Fatal("unauthorized application identity accepted")
	}
}
