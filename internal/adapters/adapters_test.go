package adapters

import "testing"

func TestRadminWithoutHamachi(t *testing.T) {
	r := Classify(Info{ID: "radmin-guid", Index: 88, Name: "Renamed virtual adapter", DriverID: "Famatech Radmin VPN Ethernet Adapter", Description: "Ethernet", Up: true, IPv4: []string{"26.10.20.30"}})
	if r.Provider != "radmin" {
		t.Fatal("Radmin driver metadata not recognized")
	}
	s, e := Select([]Info{r}, "radmin", "")
	if e != nil || !s.Ready || !s.Detected || s.IP != "26.10.20.30" || s.Selected.Index != 88 || s.DetectedProviders["hamachi"] {
		t.Fatal(s, e)
	}
	if _, e = Select([]Info{r}, "hamachi", ""); e == nil {
		t.Fatal("Radmin mistaken for Hamachi")
	}
	fake := Classify(Info{ID: "other", Name: "Radmin VPN", Up: true, IPv4: []string{"26.1.1.1"}})
	if fake.Provider != "unknown" {
		t.Fatal("friendly name/range alone accepted")
	}
	fallback := Classify(Info{ID: "fallback", Description: "Famatech RadminVPN Ethernet Adapter", Up: true, IPv4: []string{"26.1.2.3"}})
	if fallback.Provider != "radmin" {
		t.Fatal("driver description fallback failed")
	}
	r.Up = false
	if _, e = Select([]Info{r}, "radmin", ""); e == nil {
		t.Fatal("inactive Radmin selected")
	}
	r.Up = true
	r.IPv4 = nil
	if _, e = Select([]Info{r}, "radmin", ""); e == nil {
		t.Fatal("missing virtual IP accepted")
	}
	if _, e = Select(nil, "radmin", ""); e == nil {
		t.Fatal("missing Radmin accepted")
	}
}

func TestIndependentVPNSelection(t *testing.T) {
	h := Classify(Info{ID: "h", DriverID: "hamachi", Up: true, IPv4: []string{"25.1.2.3"}})
	r := Classify(Info{ID: "r", DriverID: "Famatech RadminVPN", Up: true, IPv4: []string{"26.1.2.3"}})
	for _, m := range []string{"hamachi", "radmin"} {
		s, e := Select([]Info{h, r}, m, "")
		if e != nil || s.Selected.Provider != m || !s.DetectedProviders["hamachi"] || !s.DetectedProviders["radmin"] {
			t.Fatal(s, e)
		}
	}
	if _, e := Select([]Info{h, r}, "radmin", "h"); e == nil {
		t.Fatal("wrong provider selected by GUID")
	}
	s, e := Select([]Info{h, r}, "interface", "r")
	if e != nil || s.Selected.ID != "r" {
		t.Fatal(s, e)
	}
}

func TestDetectionAndSelection(t *testing.T) {
	h := Classify(Info{ID: "guid", Name: "Renamed connection", Description: "Virtual Ethernet", DriverID: "ROOT\\HAMACHI", Up: true, IPv4: []string{"25.10.20.30"}})
	if h.Provider != "hamachi" {
		t.Fatal("driver metadata ignored")
	}
	s, e := Select([]Info{h}, "hamachi", "")
	if e != nil || s.IP != "25.10.20.30" || !s.Detected || !s.Ready {
		t.Fatal(s, e)
	}
	misleading := Classify(Info{ID: "other", Name: "Hamachi", Up: true, IPv4: []string{"25.1.1.1"}})
	if misleading.Provider == "hamachi" {
		t.Fatal("friendly name treated as proof")
	}
	s, e = Select([]Info{misleading}, "interface", "other")
	if e != nil || s.Selected.ID != "other" {
		t.Fatal(s, e)
	}
	h.Up = false
	if _, e = Select([]Info{h}, "hamachi", ""); e == nil {
		t.Fatal("down adapter accepted")
	}
	h.Up = true
	h.IPv4 = nil
	if _, e = Select([]Info{h}, "hamachi", ""); e == nil {
		t.Fatal("missing IP accepted")
	}
	if _, e = Select(nil, "hamachi", ""); e == nil {
		t.Fatal("absence accepted")
	}
	if _, e = Select([]Info{misleading}, "interface", "missing"); e == nil {
		t.Fatal("missing manual interface accepted")
	}
}
