package localdb

import (
	"encoding/json"
	"errors"
	"github.com/libp2p/go-libp2p/core/peer"
	bolt "go.etcd.io/bbolt"
	"math"
	"regexp"
)

type ListenerPreference struct {
	Volume float64 `json:"volume"`
	Muted  bool    `json:"muted"`
}
type VoiceInputPreferences struct {
	Mode        string  `json:"mode"`
	Automatic   bool    `json:"automatic"`
	ThresholdDB float64 `json:"thresholdDb"`
	Key         string  `json:"key"`
	ReleaseMS   int     `json:"releaseMs"`
}

var voiceInputKey = regexp.MustCompile(`^(F[1-9]|F1[0-2]|Key[A-Z]|Digit[0-9]|Space|CapsLock|ShiftLeft|ShiftRight)$`)

type Preferences struct {
	EntryPolicy      *VoiceInputPreferences        `json:"entryPolicy,omitempty"`
	NoiseSuppression bool                          `json:"noiseSuppression"`
	InputVolume      float64                       `json:"inputVolume"`
	OutputVolume     float64                       `json:"outputVolume"`
	Input            string                        `json:"input"`
	Output           string                        `json:"output"`
	MonitorVolume    float64                       `json:"monitorVolume"`
	CloseToTray      bool                          `json:"closeToTray"`
	Listeners        map[string]ListenerPreference `json:"listeners"`
}

func (d *DB) Preferences() (Preferences, error) {
	p := Preferences{NoiseSuppression: true, InputVolume: 1, OutputVolume: 1, MonitorVolume: .3, CloseToTray: true, Listeners: map[string]ListenerPreference{}}
	e := d.db.View(func(tx *bolt.Tx) error {
		raw := tx.Bucket([]byte("settings")).Get([]byte("audioPreferences"))
		if raw == nil {
			return nil
		}
		return json.Unmarshal(raw, &p)
	})
	if p.Listeners == nil {
		p.Listeners = map[string]ListenerPreference{}
	}
	return p, e
}
func (d *DB) SavePreferences(p Preferences) error {
	if v := p.EntryPolicy; v != nil {
		if (v.Mode != "voice" && v.Mode != "ptt") || !voiceInputKey.MatchString(v.Key) || math.IsNaN(v.ThresholdDB) || math.IsInf(v.ThresholdDB, 0) || v.ThresholdDB < -60 || v.ThresholdDB > -10 || v.ReleaseMS < 0 || v.ReleaseMS > 500 {
			return errors.New("invalid voice input policy")
		}
	}
	if math.IsNaN(p.InputVolume) || math.IsInf(p.InputVolume, 0) || p.InputVolume < 0 || p.InputVolume > 1 || math.IsNaN(p.OutputVolume) || math.IsInf(p.OutputVolume, 0) || p.OutputVolume < 0 || p.OutputVolume > 1 {
		return errors.New("invalid master volume")
	}
	if len(p.Input) > 1024 || len(p.Output) > 1024 || len(p.Listeners) > 256 || math.IsNaN(p.MonitorVolume) || math.IsInf(p.MonitorVolume, 0) || p.MonitorVolume < 0 || p.MonitorVolume > 1 {
		return errors.New("invalid audio preferences")
	}
	for id, v := range p.Listeners {
		if _, e := peer.Decode(id); e != nil || math.IsNaN(v.Volume) || math.IsInf(v.Volume, 0) || v.Volume < 0 || v.Volume > 2 {
			return errors.New("invalid participant volume")
		}
	}
	return d.put("settings", "audioPreferences", p)
}
