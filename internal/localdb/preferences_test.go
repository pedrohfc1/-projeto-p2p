package localdb

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"testing"
)

func TestAudioPreferencesPersistWithoutChangingNetworkProfile(t *testing.T) {
	dir := t.TempDir()
	d, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	initial, _ := d.Preferences()
	if !initial.NoiseSuppression || !initial.CloseToTray || initial.MonitorVolume != .3 || initial.InputVolume != 1 || initial.OutputVolume != 1 {
		t.Fatal(initial)
	}
	if e = d.SaveSettings(Settings{Name: "Legacy", Mode: "radmin", InterfaceID: "old-guid", Port: 4001}); e != nil {
		t.Fatal(e)
	}
	k, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	id, _ := peer.IDFromPrivateKey(k)
	p := Preferences{InputVolume: .7, OutputVolume: .6, Input: "mic", Output: "headphones", MonitorVolume: .4, CloseToTray: false, Listeners: map[string]ListenerPreference{id.String(): {Volume: 1.7, Muted: true}}}
	if e = d.SavePreferences(p); e != nil {
		t.Fatal(e)
	}
	d.Close()
	d, e = Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	got, _ := d.Preferences()
	settings, _ := d.Settings()
	if got.NoiseSuppression || !got.Listeners[id.String()].Muted || got.InputVolume != .7 || got.OutputVolume != .6 || got.Listeners[id.String()].Volume != 1.7 || got.CloseToTray || got.Input != "mic" || settings.Mode != "radmin" || settings.InterfaceID != "old-guid" {
		t.Fatal(got, settings)
	}
	p.Listeners[id.String()] = ListenerPreference{Volume: 3}
	if d.SavePreferences(p) == nil {
		t.Fatal("unbounded volume accepted")
	}
}

func TestLegacyAudioPreferencesDefaultMasterVolumes(t *testing.T) {
	d, e := Open(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	if e = d.put("settings", "audioPreferences", map[string]any{"input": "old-mic", "output": "old-output", "monitorVolume": 0.4}); e != nil {
		t.Fatal(e)
	}
	p, e := d.Preferences()
	if e != nil || !p.NoiseSuppression || p.InputVolume != 1 || p.OutputVolume != 1 || p.Input != "old-mic" || p.MonitorVolume != .4 {
		t.Fatal(p, e)
	}
	p.InputVolume = 1.01
	if d.SavePreferences(p) == nil {
		t.Fatal("unbounded microphone gain accepted")
	}
}

func TestLegacyProfileBusyProbeDoesNotChangeHistory(t *testing.T) {
	root := t.TempDir()
	db, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	if CheckProfileIdle(root) == nil {
		t.Fatal("legacy live profile was accepted")
	}
	db.Close()
	if e = CheckProfileIdle(root); e != nil {
		t.Fatal(e)
	}
}
func TestVoiceInputPolicyPersistenceAndValidation(t *testing.T) {
	d, e := Open(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	p, e := d.Preferences()
	if e != nil {
		t.Fatal(e)
	}
	if p.EntryPolicy != nil {
		t.Fatal("legacy should use frontend defaults")
	}
	p.EntryPolicy = &VoiceInputPreferences{Mode: "ptt", Automatic: false, ThresholdDB: -38, Key: "KeyV", ReleaseMS: 240}
	if e = d.SavePreferences(p); e != nil {
		t.Fatal(e)
	}
	loaded, e := d.Preferences()
	if e != nil || loaded.EntryPolicy == nil || *loaded.EntryPolicy != *p.EntryPolicy {
		t.Fatal(loaded, e)
	}
	for _, key := range []string{"ControlLeft", "MetaLeft", "arbitrary"} {
		p.EntryPolicy.Key = key
		if d.SavePreferences(p) == nil {
			t.Fatal("unsafe key accepted", key)
		}
	}
	p.EntryPolicy.Key = "F8"
	p.EntryPolicy.ReleaseMS = 501
	if d.SavePreferences(p) == nil {
		t.Fatal("unbounded delay accepted")
	}
	p.EntryPolicy.ReleaseMS = 100
	p.EntryPolicy.ThresholdDB = -100
	if d.SavePreferences(p) == nil {
		t.Fatal("unbounded threshold accepted")
	}
}
