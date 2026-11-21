package desktop

import (
	"context"
	"github.com/pion/webrtc/v4"
	"projetop2p/internal/loopback"
	"projetop2p/internal/voice"
	"time"
)

func (c *Core) ScreenAudioConnect(session, actor string, offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	v, e := c.voiceService()
	if e != nil {
		return webrtc.SessionDescription{}, e
	}
	ctx, cancel := context.WithTimeout(c.ctx, 8*time.Second)
	defer cancel()
	return v.ScreenAudioConnect(ctx, session, actor, offer)
}
func (c *Core) ScreenAudioCapture(session, actor string, revision uint64, mode string, pid int, consent bool) error {
	v, e := c.voiceService()
	if e != nil {
		return e
	}
	return v.ScreenAudioCapture(session, actor, revision, mode, pid, consent)
}
func (c *Core) ScreenAudioStopCapture(session, actor string, revision uint64) {
	if v, e := c.voiceService(); e == nil {
		v.ScreenAudioStopCaptureForActor(session, actor, revision)
	}
}
func (c *Core) ScreenAudioRead(session, actor string) (loopback.Frame, error) {
	v, e := c.voiceService()
	if e != nil {
		return loopback.Frame{}, e
	}
	return v.ScreenAudioRead(session, actor)
}
func (c *Core) ScreenAudioSources() ([]loopback.Source, error) {
	v, e := c.voiceService()
	if e != nil {
		return nil, e
	}
	return v.ScreenAudioSources()
}
func (c *Core) ScreenAudioState() voice.ScreenSoundView {
	if v, e := c.voiceService(); e == nil {
		return v.ScreenAudioState()
	}
	return voice.ScreenSoundView{}
}

func (c *Core) ScreenAudioDisconnect(session, actor string) {
	if v, e := c.voiceService(); e == nil {
		v.ScreenAudioDisconnect(session, actor)
	}
}
