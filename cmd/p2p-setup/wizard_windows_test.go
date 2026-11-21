package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWizardOptionsValidation(t *testing.T) {
	t.Setenv("PROJETOP2P_DATA", filepath.Join(t.TempDir(), "profile"))
	for _, tc := range []struct {
		name, raw string
		ok        bool
	}{
		{"valid", `{"directory":"C:\\Beta PeeTwoPee","desktop":true,"startMenu":false,"startup":false}`, true},
		{"bom", "\xef\xbb\xbf" + `{"directory":"C:\\Beta PeeTwoPee"}`, true},
		{"relative", `{"directory":"relative"}`, false},
		{"root", `{"directory":"C:\\"}`, false},
		{"unknown", `{"directory":"C:\\Beta PeeTwoPee","command":"x"}`, false},
		{"trailing", `{"directory":"C:\\Beta PeeTwoPee"}{}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "options.json")
			os.WriteFile(p, []byte(tc.raw), 0600)
			_, e := readOptions(p)
			if (e == nil) != tc.ok {
				t.Fatalf("error=%v", e)
			}
		})
	}
}
