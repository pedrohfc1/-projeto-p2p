//go:build !legacyeasytierlab

package desktop

type legacyLabState struct{}

func (c *Core) closeLegacyLab() {}
