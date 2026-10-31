package network

import (
	"sync"
	"time"

	pb "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/pb"
	relay "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
)

// These counters observe the relay implementation's callbacks, not merely flags
// or transport connections. They are per process and reset on restart.
type RelayServiceStatus struct {
	Enabled              bool   `json:"enabled"`
	ActiveReservations   int    `json:"activeReservations"`
	ReservationsAccepted uint64 `json:"reservationsAccepted"`
	ReservationsRenewed  uint64 `json:"reservationsRenewed"`
	ReservationsRejected uint64 `json:"reservationsRejected"`
	ActiveCircuits       int    `json:"activeCircuits"`
	CircuitsOpened       uint64 `json:"circuitsOpened"`
	CircuitsRejected     uint64 `json:"circuitsRejected"`
	BytesForwarded       uint64 `json:"bytesForwarded"`
}
type relayMetrics struct {
	base  relay.MetricsTracer
	mu    sync.Mutex
	state RelayServiceStatus
}

func newRelayMetrics() *relayMetrics { return &relayMetrics{base: relay.NewMetricsTracer()} }
func (m *relayMetrics) snapshot() *RelayServiceStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	v := m.state
	return &v
}
func (m *relayMetrics) RelayStatus(enabled bool) {
	m.base.RelayStatus(enabled)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.Enabled = enabled
	if !enabled {
		m.state.ActiveReservations = 0
		m.state.ActiveCircuits = 0
	}
}
func (m *relayMetrics) ReservationAllowed(renewal bool) {
	m.base.ReservationAllowed(renewal)
	m.mu.Lock()
	defer m.mu.Unlock()
	if renewal {
		m.state.ReservationsRenewed++
	} else {
		m.state.ReservationsAccepted++
		m.state.ActiveReservations++
	}
}
func (m *relayMetrics) ReservationClosed(count int) {
	m.base.ReservationClosed(count)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.ActiveReservations -= count
	if m.state.ActiveReservations < 0 {
		m.state.ActiveReservations = 0
	}
}
func (m *relayMetrics) ReservationRequestHandled(status pb.Status) {
	m.base.ReservationRequestHandled(status)
	if status != pb.Status_OK {
		m.mu.Lock()
		m.state.ReservationsRejected++
		m.mu.Unlock()
	}
}
func (m *relayMetrics) ConnectionOpened() {
	m.base.ConnectionOpened()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.ActiveCircuits++
	m.state.CircuitsOpened++
}
func (m *relayMetrics) ConnectionClosed(d time.Duration) {
	m.base.ConnectionClosed(d)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state.ActiveCircuits > 0 {
		m.state.ActiveCircuits--
	}
}
func (m *relayMetrics) ConnectionRequestHandled(status pb.Status) {
	m.base.ConnectionRequestHandled(status)
	if status != pb.Status_OK {
		m.mu.Lock()
		m.state.CircuitsRejected++
		m.mu.Unlock()
	}
}
func (m *relayMetrics) BytesTransferred(count int) {
	m.base.BytesTransferred(count)
	if count > 0 {
		m.mu.Lock()
		m.state.BytesForwarded += uint64(count)
		m.mu.Unlock()
	}
}
