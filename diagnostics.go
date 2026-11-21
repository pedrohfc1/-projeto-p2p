package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"projetop2p/internal/desktop"
	"projetop2p/internal/diagnostics"
	"projetop2p/internal/roomprobe"
	"strings"
	"time"
)

func (a *App) DiagnosticEvents(after uint64) diagnostics.Batch { return diagnostics.Local.Read(after) }
func (a *App) ClearDiagnostics() uint64                        { return diagnostics.Local.Clear() }
func (a *App) startDiagnostics(ctx context.Context) {
	a.backgroundWG.Add(1)
	go func() {
		defer a.backgroundWG.Done()
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		var previousVoice, previousVideo, previousNetwork, previousError, previousSession string
		peers := map[string]string{}
		var lastReceived uint64
		lastAudio := time.Now()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				v := a.core.VoiceState()
				screen := a.core.ScreenState()
				network := a.core.DiagnosticNetwork()
				if network != previousNetwork {
					level := "INFO"
					if strings.Contains(network, "erro=true") {
						level = "ERROR"
					} else if strings.Contains(network, "aviso=true") {
						level = "WARN"
					}
					diagnostics.Local.Add(level, "network", "backend/EasyTier", network)
					previousNetwork = network
				}
				key := fmt.Sprintf("joined=%t mute=%t deaf=%t transporte=%s estado=%s participantes=%d", v.Joined, v.Muted, v.Deaf, v.Media.Transport, v.Media.State, len(v.Participants))
				if key != previousVoice {
					diagnostics.Local.Add("INFO", "voice", "backend/sessão", key)
					previousVoice = key
				}
				if v.Error != "" && v.Error != previousError {
					diagnostics.Local.Add("ERROR", "voice", "backend/sessão", v.Error)
				}
				if v.Session != previousSession {
					lastAudio = time.Now()
					lastReceived = v.Media.PacketsIn
					previousSession = v.Session
				}
				previousError = v.Error
				nextPeers := map[string]string{}
				for _, p := range v.Participants {
					if !p.Self {
						diagnostics.Local.Add("INFO", "voice", "backend/peer/rede", fmt.Sprintf("peer=%s slot=%d estado=%s TX=%d RX=%d erro=%s", fmt.Sprintf("%x", sha256.Sum256([]byte(p.ID)))[:12], p.Slot, p.State, p.Media.PacketsOut, p.Media.PacketsIn, p.ConnectionError))
					}
					key := fmt.Sprintf("slot=%d self=%t estado=%s", p.Slot, p.Self, p.State)
					nextPeers[p.ID] = key
					if peers[p.ID] != key {
						diagnostics.Local.Add("INFO", "voice", "backend/peer", key)
					}
				}
				for id, key := range peers {
					if _, ok := nextPeers[id]; !ok {
						diagnostics.Local.Add("INFO", "voice", "backend/peer", "Participante saiu: "+key)
					}
				}
				peers = nextPeers
				if v.Joined {
					if v.Media.PacketsIn != lastReceived {
						lastAudio = time.Now()
						lastReceived = v.Media.PacketsIn
					}
					diagnostics.Local.Add("INFO", "voice", "backend/mídia", fmt.Sprintf("session=%s autorização=%s ponte_local=%d pacotes_rede rx=%d tx=%d bytes rx=%d tx=%d descartados=%d fila_ms=%.1f sem_rx_s=%.1f (silêncio/DTX também pode não enviar)", fmt.Sprintf("%x", sha256.Sum256([]byte(v.Session)))[:12], v.Authorization, v.BridgePackets, v.Media.PacketsIn, v.Media.PacketsOut, v.Media.BytesIn, v.Media.BytesOut, v.Media.Dropped, v.Media.QueueDelayMs, time.Since(lastAudio).Seconds()))
				}
				skey := fmt.Sprintf("active=%t publicação=%t assistindo=%t espectadores=%d", screen.Active, screen.Publishing, screen.Watching != "", screen.Viewers)
				if skey != previousVideo {
					diagnostics.Local.Add("INFO", "video", "backend/tela", skey)
					previousVideo = skey
				}
				if screen.Active && (screen.Publishing || screen.Watching != "") {
					diagnostics.Local.Add("INFO", "video", "backend/mídia", fmt.Sprintf("frames rx=%d tx=%d bytes rx=%d tx=%d descartados=%d bitrate_limite=%d rtt_ms=%.1f", screen.FramesIn, screen.FramesOut, screen.BytesIn, screen.BytesOut, screen.Dropped, screen.Bitrate, screen.RoundTripMs))
					if screen.Error != "" {
						diagnostics.Local.Add("ERROR", "video", "backend/tela", screen.Error)
					}
				}
			}
		}
	}()
}

// Diagnostic snapshots can exceed the short text limit used by invitations.
// Only local UI invokes this operation, after its explicit copy confirmation.
func (a *App) CopyDiagnostics(text string) error {
	if len(text) > 12*1024*1024 {
		return fmt.Errorf("diagnostic snapshot exceeds clipboard limit")
	}
	return wruntime.ClipboardSetText(a.ctx, text)
}

func (a *App) VoiceDiagnosticSnapshot() desktop.VoiceDiagnosticSnapshot {
	return a.core.VoiceDiagnosticSnapshot()
}

func (a *App) VoiceDiagnosticSample(probe bool) desktop.VoiceDiagnosticSnapshot {
	return a.core.VoiceDiagnosticSample(probe)
}

// Explicit user export, bounded and local; no automatic uploads.
func (a *App) ExportVoiceDiagnostic(report string, text bool) error {
	if len(report) > 2<<20 {
		return errors.New("relatório excede 2 MiB")
	}
	var value any
	if json.Unmarshal([]byte(report), &value) != nil {
		return errors.New("relatório JSON inválido")
	}
	ext := "json"
	if text {
		ext = "txt"
	}
	path, e := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "Exportar diagnóstico local", DefaultFilename: "peetwopee-diagnostico." + ext})
	if e != nil || path == "" {
		return e
	}
	return os.WriteFile(path, []byte(report), 0600)
}

func (a *App) ApproveRoomProbe(alias string) error { return a.core.ApproveRoomProbe(alias) }
func (a *App) CancelRoomProbe()                    { a.core.CancelRoomProbe() }
func (a *App) RunRoomProbe(alias string, seconds int) (roomprobe.Result, error) {
	return a.core.RunRoomProbe(alias, seconds)
}
