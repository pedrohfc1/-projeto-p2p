package network

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	libnet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	ma "github.com/multiformats/go-multiaddr"
	"io"
	"strings"
	"sync"
	"time"
)

type mediaObservation struct {
	EOFReports      uint64   `json:"eofReports"`
	LocalResets     uint64   `json:"localResets"`
	ResetReports    uint64   `json:"resetReports"`
	TransportErrors uint64   `json:"transportErrors"`
	OpenMs          *float64 `json:"openMs"`
	OpenAttempts    uint64   `json:"openAttempts"`
	OpenFailures    uint64   `json:"openFailures"`
	Timeouts        uint64   `json:"timeouts"`
}
type mediaObservations struct {
	sync.Mutex
	values map[peer.ID]mediaObservation
}

func (n *Node) observeMediaOpen(id peer.ID, elapsed time.Duration, err error) {
	n.mediaObservations.Lock()
	defer n.mediaObservations.Unlock()
	if n.mediaObservations.values == nil {
		n.mediaObservations.values = map[peer.ID]mediaObservation{}
	}
	if _, ok := n.mediaObservations.values[id]; !ok && len(n.mediaObservations.values) >= 128 {
		return
	}
	v := n.mediaObservations.values[id]
	v.OpenAttempts++
	ms := float64(elapsed) / float64(time.Millisecond)
	v.OpenMs = &ms
	if err != nil {
		v.OpenFailures++
		if timeout, ok := err.(interface{ Timeout() bool }); ok && timeout.Timeout() {
			v.Timeouts++
		} else if errors.Is(err, context.DeadlineExceeded) {
			v.Timeouts++
		}
	}
	n.mediaObservations.values[id] = v
}

type VoicePeerDiagnostic struct {
	Alias       string   `json:"alias"`
	Connections int      `json:"connections"`
	Streams     int      `json:"streams"`
	Path        string   `json:"path"`
	Transport   string   `json:"transport"`
	RTTMs       *float64 `json:"rttMs"`
	RTTSource   string   `json:"rttSource"`
	mediaObservation
}

// Samples only an already connected, room-authorized peer. Never dials stale invitations.
// The small libp2p ping is an application RTT probe, not a media loss measurement.
func (n *Node) VoicePeerDiagnostic(ctx context.Context, id string) VoicePeerDiagnostic {
	sum := sha256.Sum256([]byte(id))
	r := VoicePeerDiagnostic{Alias: fmt.Sprintf("%x", sum)[:12], Path: "desconhecido", Transport: "libp2p", RTTSource: "sonda libp2p; não é RTT RTP"}
	pid, e := peer.Decode(id)
	if e != nil {
		return r
	}
	conns := n.Host.Network().ConnsToPeer(pid)
	r.Connections = len(conns)
	paths := map[string]bool{}
	for _, c := range conns {
		r.Streams += len(c.GetStreams())
		path := "direto libp2p (underlay desconhecido)"
		if _, e := c.RemoteMultiaddr().ValueForProtocol(ma.P_CIRCUIT); e == nil {
			path = "relay libp2p"
		} else if c.Stat().Limited {
			path = "conexão limitada; caminho desconhecido"
		}
		paths[path] = true
	}
	if len(paths) == 1 {
		for path := range paths {
			r.Path = path
		}
	}
	n.mediaObservations.Lock()
	r.mediaObservation = n.mediaObservations.values[pid]
	n.mediaObservations.Unlock()
	if len(conns) == 0 || ctx.Err() != nil {
		return r
	}
	probe, cancel := context.WithTimeout(ctx, 900*time.Millisecond)
	defer cancel()
	probe = libnet.WithNoDial(libnet.WithAllowLimitedConn(probe, "authorized room diagnostic"), "diagnostic only on existing connections")
	select {
	case result, ok := <-ping.Ping(probe, n.Host, pid):
		if ok && result.Error == nil {
			ms := float64(result.RTT) / float64(time.Millisecond)
			r.RTTMs = &ms
		}
	case <-probe.Done():
	}
	return r
}

// Counters are cumulative for the node. A reset report is not proof of which side initiated it.
func (n *Node) observeMediaEnd(id peer.ID, err error, local bool) {
	n.mediaObservations.Lock()
	defer n.mediaObservations.Unlock()
	if n.mediaObservations.values == nil {
		n.mediaObservations.values = map[peer.ID]mediaObservation{}
	}
	if _, ok := n.mediaObservations.values[id]; !ok && len(n.mediaObservations.values) >= 128 {
		return
	}
	v := n.mediaObservations.values[id]
	if local {
		v.LocalResets++
	} else if errors.Is(err, io.EOF) {
		v.EOFReports++
	} else {
		v.TransportErrors++
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "reset") {
			v.ResetReports++
		}
		if e, ok := err.(interface{ Timeout() bool }); ok && e.Timeout() {
			v.Timeouts++
		}
	}
	n.mediaObservations.values[id] = v
}
