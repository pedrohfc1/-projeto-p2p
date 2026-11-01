//go:build legacydht

package network

import (
	"context"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Legacy research build ONLY: GO-2024-3218 remains unresolved and publication is blocked.
const DHTEnabled = true

type legacyDHT struct{ *dht.IpfsDHT }

func (d legacyDHT) PeerCount() int { return d.RoutingTable().Size() }
func newDHT(ctx context.Context, h host.Host, c Config, boot []peer.AddrInfo) (dhtService, error) {
	opts := []dht.Option{dht.Mode(dht.ModeServer), dht.ProtocolPrefix("/projeto-p2p"), dht.BootstrapPeers(boot...), dht.DisableValues()}
	if c.PublicDiscovery {
		opts = append(opts, dht.AddressFilter(publicAddresses))
	}
	d, e := dht.New(h, opts...)
	if e != nil {
		return nil, e
	}
	return legacyDHT{d}, nil
}
