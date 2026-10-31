package network

import (
	"context"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

const RelayNamespace = "projeto-p2p/relays/v0.2"

type relaySource struct {
	mu    sync.Mutex
	peers map[peer.ID]peer.AddrInfo
}

func (s *relaySource) add(p peer.AddrInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.peers == nil {
		s.peers = map[peer.ID]peer.AddrInfo{}
	}
	if len(p.Addrs) > 8 {
		p.Addrs = p.Addrs[:8]
	}
	if _, ok := s.peers[p.ID]; ok || len(s.peers) < 32 {
		s.peers[p.ID] = p
	}
}
func (s *relaySource) source(ctx context.Context, num int) <-chan peer.AddrInfo {
	ch := make(chan peer.AddrInfo, num)
	defer close(ch)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.peers {
		if num <= 0 || ctx.Err() != nil {
			break
		}
		ch <- p
		num--
	}
	return ch
}

type PeerStatus struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}
type Status struct {
	ID             string              `json:"id"`
	Addresses      []string            `json:"addresses"`
	Peers          []PeerStatus        `json:"peers"`
	Reachability   string              `json:"reachability"`
	RelayAddresses int                 `json:"relayAddresses"`
	RelayService   *RelayServiceStatus `json:"relayService,omitempty"`
	DHTPeers       int                 `json:"dhtPeers"`
	LocalOnly      bool                `json:"localOnly"`
}

func (n *Node) Status() Status {
	s := Status{ID: n.Host.ID().String(), Addresses: n.Addresses(), Peers: []PeerStatus{}}
	if n.DHT != nil {
		s.DHTPeers = n.DHT.RoutingTable().Size()
	}
	s.LocalOnly = n.localOnly
	if n.relayMetrics != nil {
		s.RelayService = n.relayMetrics.snapshot()
	}
	n.mu.Lock()
	s.Reachability = n.reachability
	if n.localOnly {
		s.Reachability = "interface selecionada (sem descoberta pública)"
	}
	n.mu.Unlock()
	for _, a := range s.Addresses {
		if strings.Contains(a, "/p2p-circuit") {
			s.RelayAddresses++
		}
	}
	for _, p := range n.Host.Network().Peers() {
		path := "relay"
		for _, c := range n.Host.Network().ConnsToPeer(p) {
			if !strings.Contains(c.RemoteMultiaddr().String(), "/p2p-circuit") {
				path = "direta"
				break
			}
		}
		s.Peers = append(s.Peers, PeerStatus{ID: p.String(), Path: path})
	}
	return s
}
