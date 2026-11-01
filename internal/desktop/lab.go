package desktop

import (
	"errors"
	"os"
	"path/filepath"
	"projetop2p/internal/easytierlab"
)

func (c *Core) StartIntegratedLab(cfg easytierlab.Config) error {
	c.labMu.Lock()
	defer c.labMu.Unlock()
	if c.ctx.Err() != nil {
		return errors.New("application shutting down")
	}
	if c.lab != nil {
		return errors.New("stop the current lab before reconfiguring")
	}
	if cfg.BinaryDir == "" {
		exe, e := os.Executable()
		if e != nil {
			return e
		}
		cfg.BinaryDir = filepath.Join(filepath.Dir(exe), "easytier")
	}
	l, e := easytierlab.Start(c.ctx, c.key, cfg, c.dir)
	if e != nil {
		return e
	}
	c.lab = l
	return nil
}
func (c *Core) StopIntegratedLab() {
	c.labMu.Lock()
	defer c.labMu.Unlock()
	if c.lab != nil {
		c.lab.Close()
		c.lab = nil
	}
}
func (c *Core) IntegratedLabState() easytierlab.State {
	c.labMu.Lock()
	defer c.labMu.Unlock()
	if c.lab == nil {
		return easytierlab.State{Error: "Experimental lab stopped; ordinary contacts and communities keep their selected network"}
	}
	return c.lab.State(c.ctx)
}
func (c *Core) ProbeIntegratedLab(raw, text string) (easytierlab.Result, error) {
	c.labMu.Lock()
	defer c.labMu.Unlock()
	if c.lab == nil {
		return easytierlab.Result{}, errors.New("start the lab first")
	}
	return c.lab.Ping(c.ctx, raw, text)
}
