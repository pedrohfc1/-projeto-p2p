package discovery

import (
	"context"
	"fmt"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	routing "github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

const Namespace = "projeto-p2p/v0.1"

// Find advertises a rendezvous provider record, then streams bounded lookup results.
func Find(ctx context.Context, d *dht.IpfsDHT) (<-chan peer.AddrInfo, error) {
	r := routing.NewRoutingDiscovery(d)
	if _, err := r.Advertise(ctx, Namespace); err != nil {
		return nil, fmt.Errorf("announce: %w", err)
	}
	return r.FindPeers(ctx, Namespace)
}
