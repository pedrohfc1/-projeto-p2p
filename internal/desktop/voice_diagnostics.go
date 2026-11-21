package desktop

import (
	"context"
	"crypto/sha256"
	"fmt"
	"projetop2p/internal/network"
	"projetop2p/internal/telemetry"
	"projetop2p/internal/voice"
	"sync"
	"time"
)

type voiceDiagnosticFrame struct {
	sync.Mutex
	at    time.Time
	busy  bool
	value VoiceDiagnosticSnapshot
}

type voiceDiagnosticCache struct {
	active, passive voiceDiagnosticFrame
	resources       telemetry.Sampler
}

type VoiceDiagnosticPeer struct {
	network.VoicePeerDiagnostic
	Media voice.MediaStats `json:"media"`
}
type VoiceDiagnosticSnapshot struct {
	Time          int64                 `json:"time"`
	Session       string                `json:"session"`
	Mode          string                `json:"mode"`
	Authorization string                `json:"authorization"`
	Expires       int64                 `json:"expires"`
	Peers         []VoiceDiagnosticPeer `json:"peers"`
	BridgePackets uint64                `json:"bridgePackets"`
	HeapBytes     uint64                `json:"heapBytes"`
	CPUPercent    *float64              `json:"cpuPercent"`
	RSSBytes      *uint64               `json:"rssBytes"`
	Joined        bool                  `json:"joined"`
}

// VoiceDiagnosticSnapshot retains the initial binding for diagnostic clients.
func (c *Core) VoiceDiagnosticSnapshot() VoiceDiagnosticSnapshot {
	return c.VoiceDiagnosticSample(true)
}

// VoiceDiagnosticSample reads room counters. Only the visible diagnostic panel
// asks for active 32-byte RTT probes to connected, authorized participants.
func (c *Core) VoiceDiagnosticSample(probe bool) VoiceDiagnosticSnapshot {
	v := c.VoiceState()
	session := ""
	if v.Joined {
		sum := sha256.Sum256([]byte(v.Session))
		session = fmt.Sprintf("%x", sum[:6])
	}
	// A passive read must never satisfy an active RTT request (or the reverse).
	sample := &c.voiceDiag.passive
	if probe {
		sample = &c.voiceDiag.active
	}
	sample.Lock()
	if sample.busy || (sample.value.Session == session && time.Since(sample.at) < time.Second) {
		r := sample.value
		if r.Session != session {
			r = VoiceDiagnosticSnapshot{}
		}
		if r.Peers == nil {
			r.Peers = []VoiceDiagnosticPeer{}
		}
		sample.Unlock()
		return r
	}
	sample.busy = true
	sample.at = time.Now()
	sample.Unlock()
	var result VoiceDiagnosticSnapshot
	defer func() {
		sample.Lock()
		sample.value = result
		sample.busy = false
		sample.Unlock()
	}()
	r := VoiceDiagnosticSnapshot{Session: session, Time: time.Now().UnixMilli(), Authorization: v.Authorization, Expires: v.Expires, BridgePackets: v.BridgePackets, Joined: v.Joined, Peers: []VoiceDiagnosticPeer{}}
	if !v.Joined {
		result = r
		return r
	}
	usage := c.voiceDiag.resources.Read(0, 0)
	r.CPUPercent = usage.CPUPercent
	r.RSSBytes = usage.RSSBytes
	r.HeapBytes = usage.HeapBytes
	c.mu.Lock()
	n := c.node
	s, _ := c.db.Settings()
	r.Mode = mode(s)
	c.mu.Unlock()
	if n == nil {
		result = r
		return r
	}
	ctx, cancel := context.WithTimeout(c.ctx, time.Second)
	defer cancel()
	if !probe || v.Authorization != "valid" {
		cancel()
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	count := 0
	for _, p := range v.Participants {
		if p.Self || count >= 5 {
			continue
		}
		count++
		p := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			observed := n.VoicePeerDiagnostic(ctx, p.ID)
			mu.Lock()
			r.Peers = append(r.Peers, VoiceDiagnosticPeer{observed, p.Media})
			mu.Unlock()
		}()
	}
	wg.Wait()
	result = r
	return r
}
