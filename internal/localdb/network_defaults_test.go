package localdb

import (
	"testing"
)

func TestNetworkDefaultAndSavedPreferences(t *testing.T) {
	dir := t.TempDir()
	d, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	p, e := d.Settings()
	if e != nil || p.Mode != "integrated" {
		t.Fatal(p, e)
	}
	for _, mode := range []string{"auto", "radmin", "hamachi", "interface", "integrated", ""} {
		p.Mode = mode
		p.Name = "preserved"
		p.InterfaceID = "saved-adapter"
		if e = d.SaveSettings(p); e != nil {
			t.Fatal(e)
		}
		d.Close()
		d, e = Open(dir)
		if e != nil {
			t.Fatal(e)
		}
		got, e := d.Settings()
		if e != nil || got.Mode != mode || got.Name != "preserved" || got.InterfaceID != "saved-adapter" {
			t.Fatal(got, e)
		}
	}
	if e = d.put("settings", "profile", map[string]any{"name": "No mode preference", "port": 4123}); e != nil {
		t.Fatal(e)
	}
	got, e := d.Settings()
	if e != nil || got.Mode != "integrated" || got.Port != 4123 || got.Name != "No mode preference" {
		t.Fatal(got, e)
	}

	if e = d.put("settings", "profile", map[string]any{"name": "Legacy native", "bootstrap": []string{"configured-native-operator"}}); e != nil {
		t.Fatal(e)
	}
	got, e = d.Settings()
	if e != nil || got.Mode != "auto" || len(got.Bootstrap) != 1 {
		t.Fatal("legacy native configuration replaced", got, e)
	}
	d.Close()
}
