//go:build !windows && !linux

package tray

import "errors"

type Tray struct{}

func Start(string, string, Callbacks) (*Tray, error) {
	return nil, errors.New("system tray not supported on this platform")
}
func (*Tray) Update(State)               {}
func (*Tray) Close()                     {}
func (*Tray) Notify(string, string) bool { return false }

func (*Tray) Available() bool { return false }
