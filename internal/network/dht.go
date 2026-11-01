package network

import (
	"context"
	"github.com/libp2p/go-libp2p/core/routing"
)

type dhtService interface {
	routing.ContentRouting
	Bootstrap(context.Context) error
	Close() error
	PeerCount() int
}
