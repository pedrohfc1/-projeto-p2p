package discovery

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	discoveryrouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

const Namespace = "projeto-p2p/v0.1"

// Find advertises a rendezvous provider record, then streams bounded lookup results.
func Find(ctx context.Context, d routing.ContentRouting) (<-chan peer.AddrInfo, error) {
	r := discoveryrouting.NewRoutingDiscovery(d)
	if _, err := r.Advertise(ctx, Namespace); err != nil {
		return nil, fmt.Errorf("announce: %w", err)
	}
	return r.FindPeers(ctx, Namespace)
}
