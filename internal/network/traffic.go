package network

import (
	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"sync"
	"sync/atomic"
	"time"
)

type trafficCounter struct {
	in, out atomic.Int64
	mu      sync.Mutex
	peers   map[peer.ID]metrics.Stats
}

func (t *trafficCounter) record(id peer.ID, n int64, sent bool) {
	if n < 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.peers == nil {
		t.peers = make(map[peer.ID]metrics.Stats)
	}
	v, ok := t.peers[id]
	if !ok && len(t.peers) >= 256 {
		return
	}
	if sent {
		v.TotalOut += n
	} else {
		v.TotalIn += n
	}
	t.peers[id] = v
}

func (t *trafficCounter) LogSentMessage(n int64) { t.out.Add(n) }
func (t *trafficCounter) LogRecvMessage(n int64) { t.in.Add(n) }
func (t *trafficCounter) LogSentMessageStream(n int64, _ protocol.ID, id peer.ID) {
	t.record(id, n, true)
}
func (t *trafficCounter) LogRecvMessageStream(n int64, _ protocol.ID, id peer.ID) {
	t.record(id, n, false)
}
func (t *trafficCounter) GetBandwidthForPeer(id peer.ID) metrics.Stats {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.peers[id]
}
func (t *trafficCounter) GetBandwidthForProtocol(protocol.ID) metrics.Stats { return metrics.Stats{} }
func (t *trafficCounter) GetBandwidthByPeer() map[peer.ID]metrics.Stats {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make(map[peer.ID]metrics.Stats, len(t.peers))
	for id, v := range t.peers {
		result[id] = v
	}
	return result
}
func (t *trafficCounter) GetBandwidthByProtocol() map[protocol.ID]metrics.Stats {
	return map[protocol.ID]metrics.Stats{}
}
func (t *trafficCounter) GetBandwidthTotals() metrics.Stats {
	return metrics.Stats{TotalIn: t.in.Load(), TotalOut: t.out.Load()}
}
func (t *trafficCounter) Reset() {
	t.in.Store(0)
	t.out.Store(0)
	t.mu.Lock()
	t.peers = nil
	t.mu.Unlock()
}
func (t *trafficCounter) TrimIdle(time.Time) {}

type Traffic struct {
	Peers         map[string]PeerTraffic `json:"peers"`
	UploadBytes   int64                  `json:"uploadBytes"`
	DownloadBytes int64                  `json:"downloadBytes"`
	Connections   int                    `json:"connections"`
	Streams       int                    `json:"streams"`
	LastConnectMs *float64               `json:"lastConnectMs"`
}

type PeerTraffic struct {
	UploadBytes   *int64 `json:"uploadBytes"`
	DownloadBytes *int64 `json:"downloadBytes"`
	Streams       int    `json:"streams"`
	Opened        int64  `json:"opened"`
}

func (n *Node) Traffic() Traffic {
	r := Traffic{Peers: map[string]PeerTraffic{}}
	if n.traffic != nil {
		stats := n.traffic.GetBandwidthTotals()
		r.UploadBytes = stats.TotalOut
		r.DownloadBytes = stats.TotalIn
	}
	connections := n.Host.Network().Conns()
	r.Connections = len(connections)
	for _, c := range connections {
		r.Streams += len(c.GetStreams())
		id := c.RemotePeer()
		v := r.Peers[id.String()]
		v.Streams += len(c.GetStreams())
		opened := c.Stat().Opened.UnixMilli()
		if v.Opened == 0 || opened < v.Opened {
			v.Opened = opened
		}
		if n.traffic != nil {
			n.traffic.mu.Lock()
			stats, ok := n.traffic.peers[id]
			n.traffic.mu.Unlock()
			if ok {
				v.UploadBytes = &stats.TotalOut
				v.DownloadBytes = &stats.TotalIn
			}
		}
		r.Peers[id.String()] = v
	}
	if us := n.lastConnectMicros.Load(); us > 0 {
		ms := float64(us) / 1000
		r.LastConnectMs = &ms
	}
	return r
}
