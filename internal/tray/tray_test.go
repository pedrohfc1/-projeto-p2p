package tray

import (
	"strings"
	"testing"
)

func TestClosePolicy(t *testing.T) {
	if !ShouldHide(true, true, false) {
		t.Fatal("default close must hide")
	}
	for _, args := range [][3]bool{{false, true, false}, {true, false, false}, {true, true, true}} {
		if ShouldHide(args[0], args[1], args[2]) {
			t.Fatal("unrestorable/explicit exit must not hide", args)
		}
	}
}
func TestMutedTooltip(t *testing.T) {
	if !strings.Contains((State{Joined: true, Muted: true}).Tooltip(), "silenciado") {
		t.Fatal("mute indicator absent")
	}
	if !strings.Contains((State{}).Tooltip(), "sem chamada") {
		t.Fatal("idle indistinguishable from active microphone")
	}
}

func TestSharingIndicatorOverridesMutedAndIdle(t *testing.T) {
	for _, s := range []State{{Sharing: true}, {Joined: true, Muted: true, Sharing: true}} {
		if !strings.Contains(s.Tooltip(), "AO VIVO") || !strings.Contains(s.Tooltip(), "parar") {
			t.Fatal("missing persistent sharing/stop indicator")
		}
	}
}
