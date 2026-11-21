package tray

import (
	"path/filepath"
	"testing"
	"time"
)

func TestWindowsNotificationIconLifecycle(t *testing.T) {
	normal, _ := filepath.Abs("../../assets/brand/tray.ico")
	muted, _ := filepath.Abs("../../assets/brand/tray-muted.ico")
	opened := make(chan struct{}, 1)
	stopped := make(chan struct{}, 1)
	tr, e := Start(normal, muted, Callbacks{StopScreen: func() { stopped <- struct{}{} }, Open: func() { opened <- struct{}{} }})
	if e != nil {
		t.Skipf("interactive notification area unavailable: %v", e)
	}
	if !tr.Available() {
		t.Fatal("registered icon not available")
	}
	tr.Update(State{Joined: true, Muted: true, Sharing: true})
	tr.action(5)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("stop sharing callback missing")
	}
	user.NewProc("PostMessageW").Call(uintptr(tr.window), 0x8001, 0, 0x203)
	select {
	case <-opened:
	case <-time.After(2 * time.Second):
		t.Fatal("double-click callback not dispatched")
	}
	closed := make(chan struct{})
	go func() { tr.Close(); close(closed) }()
	select {
	case <-closed:
		if tr.Available() {
			t.Fatal("closed tray authorizes background capture")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("tray message loop/resources did not stop")
	}
}
