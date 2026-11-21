package main

import "testing"

func TestNativeScreenVisibilityOverridesRendererTimerState(t *testing.T) {
	if !screenHeartbeatAllowed(true, false) {
		t.Fatal("visible window rejected")
	}
	for _, state := range [][2]bool{{false, false}, {true, true}, {false, true}} {
		if screenHeartbeatAllowed(state[0], state[1]) {
			t.Fatal("hidden/minimized window allowed to renew screen capture")
		}
	}
}
