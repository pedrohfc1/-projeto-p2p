package overlay

import (
	"context"
	"errors"
	"net"
	"strconv"
)

// MediaForward creates a dedicated UDP-only forwarding path without restarting chat.
// The shared credential only reaches the remote explicitly allowed UDP service.
func (m *Manager) MediaForward(ctx context.Context, id string, remotePort int) (int, error) {
	if remotePort < 1024 || remotePort > 65535 {
		return 0, errors.New("invalid media port")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	token := m.store.Routes[id]
	if token == "" {
		return 0, errors.New("missing authenticated peer route")
	}
	if old := m.media[id]; old != nil {
		if old.remote == remotePort && old.child.alive() {
			return old.local, nil
		}
		old.child.stop()
		delete(m.media, id)
	}
	if len(m.media) >= 1 {
		return 0, errors.New("v0.6.0 allows one remote media peer")
	}
	socket, e := net.ListenPacket("udp4", "127.0.0.1:0")
	if e != nil {
		return 0, e
	}
	port := socket.LocalAddr().(*net.UDPAddr).Port
	socket.Close()
	child, ready, e := m.spawn([]string{"--key", "new", "--relay-file", m.registry, "socks", "--listen", "127.0.0.1:0"}, token, false, "P2P_ETCAT_UDP_FORWARD="+strconv.Itoa(port)+":"+strconv.Itoa(remotePort))
	if e != nil {
		return 0, e
	}
	if _, e = waitReady(ctx, child, ready); e != nil {
		child.stop()
		return 0, e
	}
	m.media[id] = &mediaSession{child: child, local: port, remote: remotePort}
	return port, nil
}
func (m *Manager) CloseMedia(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s := m.media[id]; s != nil {
		s.child.stop()
		delete(m.media, id)
	}
}
