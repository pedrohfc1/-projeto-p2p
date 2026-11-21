package desktop

import (
	"fmt"
	"projetop2p/internal/diagnostics"
)

// DiagnosticNetwork exposes only operational counters, no invites or profile data.
func (c *Core) DiagnosticNetwork() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return "nó encerrado"
	}
	peers := 0
	paths := map[string]int{}
	if c.node != nil {
		status := c.node.Status()
		peers = len(status.Peers)
		for _, p := range status.Peers {
			paths[diagnostics.Redact(p.Path)]++
		}
	}
	if c.integrated == nil {
		return fmt.Sprintf("modo=%s libp2p peers=%d caminhos=%v EasyTier=indisponível erro=%t motivo=%s", c.selection.Mode, peers, paths, c.networkErr != "", diagnostics.Redact(c.networkErr))
	}
	s := c.integrated.State()
	return fmt.Sprintf("modo=%s caminhos=%v libp2p peers=%d EasyTier running=%t processes=%d restarts=%d erro=%t aviso=%t", c.selection.Mode, paths, peers, s.Running, s.ClientProcesses, s.Restarts, s.Error != "", s.Warning != "") + " motivo=" + diagnostics.Redact(s.Error) + " " + diagnostics.Redact(s.Warning)
}
