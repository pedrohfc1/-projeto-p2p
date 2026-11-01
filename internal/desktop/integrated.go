package desktop

import (
	"fmt"
	"projetop2p/internal/adapters"
	"projetop2p/internal/localdb"
)

func (c *Core) synchronizeIntegrated(s localdb.Settings) {
	fingerprint := fmt.Sprintf("integrated|%d|%s|%v|%s", port(s), s.IntegratedBinaryDir, s.IntegratedRelays, s.IntegratedRelayKey)
	items, _ := c.enumerate()
	selection, _ := adapters.Select(items, "auto", "")
	selection.Mode = "integrated"
	selection.Selected = nil
	selection.IP = ""
	selection.Ready = false
	selection.Message = "Rede P2P integrada: ponte userspace, sem adaptador virtual. Estado do processo não comprova peer conectado."
	c.selection = selection
	if c.node != nil && c.binding == fingerprint {
		c.selection.Ready = c.integrated != nil && c.integrated.State().Running
		return
	}
	c.stopNode()
	if e := c.start(s); e != nil {
		c.networkErr = "Não foi possível iniciar a conectividade integrada: " + e.Error()
		return
	}
	c.binding = fingerprint
	c.networkErr = ""
	c.selection.Ready = true
}
