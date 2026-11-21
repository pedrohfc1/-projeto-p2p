package network

import (
	"context"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/record"
	ma "github.com/multiformats/go-multiaddr"
	"projetop2p/internal/community"
	"projetop2p/internal/transportaddr"
	"testing"
	"time"
)

func TestIntegratedResolutionIgnoresStaleLANAndPort(t *testing.T) {
	owner := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer owner.Close()
	id := owner.Host.ID()
	n := &Node{integrated: true, hasRoute: func(s string) bool { return s == id.String() }}
	p := n.resolveEndpoints(peer.AddrInfo{ID: id, Addrs: []ma.Multiaddr{ma.StringCast("/ip4/127.0.0.1/tcp/60129"), ma.StringCast("/ip4/192.168.100.2/tcp/60129"), ma.StringCast("/ip6/fd00::1/tcp/10000")}})
	if len(p.Addrs) != 1 {
		t.Fatal(p)
	}
	ip, _ := p.Addrs[0].ValueForProtocol(ma.P_IP6)
	if ip != transportaddr.LogicalIP(id.String()) {
		t.Fatal("identity endpoint mismatch", p)
	}
	n.hasRoute = func(string) bool { return false }
	if len(n.resolveEndpoints(p).Addrs) != 0 {
		t.Fatal("missing signed route fell back to LAN")
	}
}
func TestCommunityLoopbackRequiresLocalListener(t *testing.T) {
	local := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer local.Close()
	remote := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/0.0.0.0/tcp/0"}})
	defer remote.Close()
	a := ma.StringCast("/ip4/127.0.0.1/tcp/60129")
	if !local.communityEndpointAllowed(a) {
		t.Fatal("explicit local test listener rejected")
	}
	if remote.communityEndpointAllowed(a) {
		t.Fatal("loopback advertised as internet owner path")
	}
	if local.communityEndpointAllowed(ma.StringCast("/ip4/169.254.12.3/tcp/4001")) {
		t.Fatal("link-local accepted")
	}
	_, e := remote.CommunityRequest(context.Background(), local.Host.ID().String(), []string{a.String() + "/p2p/" + local.Host.ID().String()}, community.Request{Method: "sync"})
	if e == nil {
		t.Fatal("unreachable owner endpoint accepted")
	}
}

func TestCommunityUsesCertifiedCurrentEndpointAndClearsOldCache(t *testing.T) {
	owner := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer owner.Close()
	client := create(t, Config{DataDir: t.TempDir(), Listen: []string{"/ip4/127.0.0.1/tcp/0"}})
	defer client.Close()
	owner.SetCommunityHandler(func(context.Context, string, community.Request) community.Response {
		return community.Response{Code: "endpoint-authenticated"}
	})
	signed, err := record.Seal(&peer.PeerRecord{PeerID: owner.Host.ID(), Seq: uint64(time.Now().UnixNano()), Addrs: owner.Host.Addrs()}, owner.Host.Peerstore().PrivKey(owner.Host.ID()))
	if err != nil {
		t.Fatal(err)
	}
	book, ok := peerstore.GetCertifiedAddrBook(client.Host.Peerstore())
	if !ok {
		t.Fatal("no certified address book")
	}
	if _, err = book.ConsumePeerRecord(signed, time.Hour); err != nil {
		t.Fatal(err)
	}
	stale := ma.StringCast("/ip4/127.0.0.1/tcp/1")
	client.Host.Peerstore().AddAddrs(owner.Host.ID(), []ma.Multiaddr{stale}, peerstore.PermanentAddrTTL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := client.CommunityRequest(ctx, owner.SelfID(), []string{stale.String() + "/p2p/" + owner.SelfID()}, community.Request{Method: "sync"})
	if err != nil || res.Code != "endpoint-authenticated" {
		t.Fatal("certified fallback failed", res, err)
	}
	if _, err = client.CommunityRequest(ctx, owner.SelfID(), owner.Addresses(), community.Request{Method: "sync"}); err != nil {
		t.Fatal(err)
	}
	for _, a := range client.Host.Peerstore().Addrs(owner.Host.ID()) {
		if a.Equal(stale) {
			t.Fatal("permanent obsolete endpoint survived refreshed owner candidates")
		}
	}
}
