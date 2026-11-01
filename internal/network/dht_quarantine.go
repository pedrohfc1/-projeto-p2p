//go:build !legacydht

package network

import (
	"context"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// The beta does not link the provider DHT affected by GO-2024-3218.
// Explicit bootstrap connections and libp2p Circuit Relay remain available.
const DHTEnabled = false

func newDHT(context.Context, host.Host, Config, []peer.AddrInfo) (dhtService, error) { return nil, nil }
