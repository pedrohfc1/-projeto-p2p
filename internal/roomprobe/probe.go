package roomprobe

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sort"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	libnet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const roomProbeProtocol = protocol.ID("/peetwopee/voice-diagnostic/1.0.0")
const roomProbeBlockSize = 4000
const roomProbeInterval = 250 * time.Millisecond
const maxRoomProbeBlocks = 60 * time.Second / roomProbeInterval

type Result struct {
	DurationSeconds float64  `json:"durationSeconds"`
	TotalBlocks     int      `json:"totalBlocks"`
	PayloadBytes    int      `json:"payloadBytes"`
	RTTMeanMs       float64  `json:"rttMeanMs"`
	RTTP95Ms        float64  `json:"rttP95Ms"`
	VariationMs     *float64 `json:"variationMs"`
	PayloadKbps     float64  `json:"payloadKbps"`
}

func SetHandler(h host.Host, permit func(peer.ID) (func() bool, bool)) {
	h.SetStreamHandler(roomProbeProtocol, func(stream libnet.Stream) {
		defer stream.Close()
		valid, ok := permit(stream.Conn().RemotePeer())
		if !ok || !valid() {
			_ = stream.Reset()
			return
		}
		serveRoomProbe(stream, valid, roomProbeInterval, int(maxRoomProbeBlocks))
	})
}

type probeStream interface {
	io.ReadWriter
	SetDeadline(time.Time) error
}

// An echoed block is counted only after a valid receiver approval. Every block
// is rechecked against the local room session and paced by the receiver.
func serveRoomProbe(stream probeStream, valid func() bool, interval time.Duration, limit int) {
	if stream.SetDeadline(time.Now().Add(time.Second)) != nil || writeProbeBlock(stream, []byte{1}) != nil {
		return
	}
	next := time.Now()
	block := make([]byte, roomProbeBlockSize)
	for i := 0; i < limit; i++ {
		if !valid() || stream.SetDeadline(time.Now().Add(time.Second)) != nil {
			return
		}
		if _, err := io.ReadFull(stream, block); err != nil {
			return
		}
		if wait := time.Until(next); wait > 0 {
			time.Sleep(wait)
		}
		if !valid() || writeProbeBlock(stream, block) != nil {
			return
		}
		next = time.Now().Add(interval)
	}
}

func Run(ctx context.Context, h host.Host, id peer.ID, seconds int, valid func() bool) (Result, error) {
	if seconds != 10 && seconds != 30 && seconds != 60 {
		return Result{}, errors.New("use duração de 10, 30 ou 60 segundos")
	}
	if !valid() {
		return Result{}, errors.New("a autorização da sala mudou")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(seconds+5)*time.Second)
	defer cancel()
	ctx = libnet.WithNoDial(libnet.WithAllowLimitedConn(ctx, "room diagnostic"), "existing peer connection only")
	stream, err := h.NewStream(ctx, id, roomProbeProtocol)
	if err != nil {
		return Result{}, errors.New("teste indisponível nesta conexão; verifique a versão do outro participante")
	}
	defer stream.Close()
	stop := context.AfterFunc(ctx, func() { _ = stream.Reset() })
	defer stop()
	return measureRoomProbe(ctx, stream, time.Duration(seconds)*time.Second, roomProbeInterval, valid)
}

func measureRoomProbe(ctx context.Context, stream probeStream, duration, interval time.Duration, valid func() bool) (Result, error) {
	var acceptance [1]byte
	if stream.SetDeadline(time.Now().Add(time.Second)) != nil {
		return Result{}, errors.New("stream de diagnóstico indisponível")
	}
	if _, err := io.ReadFull(stream, acceptance[:]); err != nil || acceptance[0] != 1 {
		return Result{}, errors.New("o participante precisa permitir o teste nesta sala")
	}
	start := time.Now()
	ticker := time.NewTimer(interval)
	defer ticker.Stop()
	timer := time.NewTimer(duration)
	defer timer.Stop()
	sent := make([]byte, roomProbeBlockSize)
	received := make([]byte, roomProbeBlockSize)
	samples := make([]float64, 0, int(duration/interval))
	for {
		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()
		case <-timer.C:
			if len(samples) == 0 {
				return Result{}, errors.New("nenhuma resposta de diagnóstico")
			}
			return summarizeRoomProbe(samples, time.Since(start)), nil
		case <-ticker.C:
			if time.Since(start) >= duration {
				continue
			}
			if !valid() {
				return Result{}, errors.New("a autorização da sala mudou")
			}
			at := time.Now()
			sent[0]++
			deadline := at.Add(time.Second)
			// Keep the measurement window fixed, even if the remote stalls.
			if end := start.Add(duration); end.Before(deadline) {
				deadline = end
			}
			if stream.SetDeadline(deadline) != nil || writeProbeBlock(stream, sent) != nil {
				return Result{}, errors.New("envio do diagnóstico interrompido")
			}
			if _, err := io.ReadFull(stream, received); err != nil || !bytes.Equal(received, sent) {
				if !time.Now().Before(start.Add(duration)) && len(samples) > 0 && ctx.Err() == nil && valid() {
					return summarizeRoomProbe(samples, time.Since(start)), nil
				}
				return Result{}, errors.New("resposta de diagnóstico interrompida")
			}
			samples = append(samples, float64(time.Since(at))/float64(time.Millisecond))
			ticker.Reset(max(time.Until(at.Add(interval)), 0))
		}
	}
}

func writeProbeBlock(stream io.Writer, block []byte) error {
	n, err := stream.Write(block)
	if err == nil && n != len(block) {
		return io.ErrShortWrite
	}
	return err
}

func summarizeRoomProbe(samples []float64, elapsed time.Duration) Result {
	blocks := len(samples)
	var sum, variation float64
	for i, sample := range samples {
		sum += sample
		if i > 0 {
			difference := sample - samples[i-1]
			if difference < 0 {
				difference = -difference
			}
			variation += difference
		}
	}
	result := Result{
		DurationSeconds: elapsed.Seconds(), TotalBlocks: blocks, PayloadBytes: blocks * roomProbeBlockSize,
		RTTMeanMs:   sum / float64(blocks),
		PayloadKbps: float64(blocks*roomProbeBlockSize*8) / elapsed.Seconds() / 1000,
	}
	if blocks > 1 {
		variation /= float64(blocks - 1)
		result.VariationMs = &variation
	}
	sort.Float64s(samples)
	result.RTTP95Ms = samples[(blocks*95+99)/100-1]
	return result
}
