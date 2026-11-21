package main

import (
	"github.com/pion/webrtc/v4"
	"projetop2p/internal/loopback"
	"projetop2p/internal/voice"
)

func (a *App) ScreenAudioConnect(session, actor string, offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	if e := a.initialized(); e != nil {
		return webrtc.SessionDescription{}, e
	}
	return a.core.ScreenAudioConnect(session, actor, offer)
}
func (a *App) ScreenAudioCapture(session, actor string, revision uint64, mode string, pid int, consent bool) error {
	if e := a.initialized(); e != nil {
		return e
	}
	return a.core.ScreenAudioCapture(session, actor, revision, mode, pid, consent)
}
func (a *App) ScreenAudioStopCapture(session, actor string, revision uint64) error {
	if e := a.initialized(); e != nil {
		return e
	}
	a.core.ScreenAudioStopCapture(session, actor, revision)
	return nil
}
func (a *App) ScreenAudioRead(session, actor string) (loopback.Frame, error) {
	if e := a.initialized(); e != nil {
		return loopback.Frame{}, e
	}
	return a.core.ScreenAudioRead(session, actor)
}
func (a *App) ScreenAudioSources() ([]loopback.Source, error) {
	if e := a.initialized(); e != nil {
		return nil, e
	}
	return a.core.ScreenAudioSources()
}
func (a *App) ScreenAudioState() (voice.ScreenSoundView, error) {
	if e := a.initialized(); e != nil {
		return voice.ScreenSoundView{}, e
	}
	return a.core.ScreenAudioState(), nil
}

func (a *App) ScreenAudioDisconnect(session, actor string) error {
	if e := a.initialized(); e != nil {
		return e
	}
	a.core.ScreenAudioDisconnect(session, actor)
	return nil
}
