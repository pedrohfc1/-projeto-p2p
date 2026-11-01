//go:build legacyeasytierlab

package main

import "projetop2p/internal/easytierlab"

func (a *App) StartIntegratedLab(c easytierlab.Config) error {
	if e := a.initialized(); e != nil {
		return e
	}
	return a.core.StartIntegratedLab(c)
}
func (a *App) StopIntegratedLab() error {
	if e := a.initialized(); e != nil {
		return e
	}
	a.core.StopIntegratedLab()
	return nil
}
func (a *App) IntegratedLabState() (easytierlab.State, error) {
	if e := a.initialized(); e != nil {
		return easytierlab.State{}, e
	}
	return a.core.IntegratedLabState(), nil
}
func (a *App) ProbeIntegratedLab(raw, text string) (easytierlab.Result, error) {
	if e := a.initialized(); e != nil {
		return easytierlab.Result{}, e
	}
	return a.core.ProbeIntegratedLab(raw, text)
}
