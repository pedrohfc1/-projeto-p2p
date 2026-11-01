package easytierlab

import (
	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"sync/atomic"
	"time"
)

type trafficCounter struct{ in, out atomic.Int64 }

func (t *trafficCounter) LogSentMessage(n int64)                            { t.out.Add(n) }
func (t *trafficCounter) LogRecvMessage(n int64)                            { t.in.Add(n) }
func (t *trafficCounter) LogSentMessageStream(int64, protocol.ID, peer.ID)  {}
func (t *trafficCounter) LogRecvMessageStream(int64, protocol.ID, peer.ID)  {}
func (t *trafficCounter) GetBandwidthForPeer(peer.ID) metrics.Stats         { return metrics.Stats{} }
func (t *trafficCounter) GetBandwidthForProtocol(protocol.ID) metrics.Stats { return metrics.Stats{} }
func (t *trafficCounter) GetBandwidthByPeer() map[peer.ID]metrics.Stats {
	return map[peer.ID]metrics.Stats{}
}
func (t *trafficCounter) GetBandwidthByProtocol() map[protocol.ID]metrics.Stats {
	return map[protocol.ID]metrics.Stats{}
}
func (t *trafficCounter) GetBandwidthTotals() metrics.Stats {
	return metrics.Stats{TotalIn: t.in.Load(), TotalOut: t.out.Load()}
}
func (t *trafficCounter) Reset()             { t.in.Store(0); t.out.Store(0) }
func (t *trafficCounter) TrimIdle(time.Time) {}
