// Package telemetry reports actual local counters, never infers UDP packet loss from TCP errors.
package telemetry

import (
	"runtime"
	"sync"
	"time"
)

type Sample struct {
	HeapBytes        uint64   `json:"heapBytes"`
	RSSBytes         *uint64  `json:"rssBytes"`
	CPUPercent       *float64 `json:"cpuPercent"`
	CPUSecs          float64  `json:"cpuSecs"`
	Goroutines       int      `json:"goroutines"`
	UploadBytes      int64    `json:"uploadBytes"`
	DownloadBytes    int64    `json:"downloadBytes"`
	UploadBps        float64  `json:"uploadBps"`
	DownloadBps      float64  `json:"downloadBps"`
	Connections      int      `json:"connections"`
	Streams          int      `json:"streams"`
	LastConnectMs    *float64 `json:"lastConnectMs"`
	PingMs           *float64 `json:"pingMs"`
	PacketLoss       *float64 `json:"packetLoss"`
	PacketLossReason string   `json:"packetLossReason"`
	Scope            string   `json:"scope"`
}
type Sampler struct {
	mu               sync.Mutex
	at               time.Time
	cpu              float64
	upload, download int64
}

func (s *Sampler) Read(upload, download int64) Sample {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	rss, cpu, ok := process()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	r := Sample{HeapBytes: mem.HeapAlloc, CPUPercent: nil, CPUSecs: cpu, Goroutines: runtime.NumGoroutine(), UploadBytes: upload, DownloadBytes: download, PacketLossReason: "não mensurada: transporte TCP não fornece perda de pacotes UDP", Scope: "processo Go atual; não inclui processos WebView2 separados; CPU 100% equivale a um núcleo lógico"}
	if rss > 0 {
		r.RSSBytes = &rss
	}
	if !s.at.IsZero() {
		delta := now.Sub(s.at).Seconds()
		if delta > 0 {
			if ok && cpu >= s.cpu {
				usage := (cpu - s.cpu) / delta * 100
				r.CPUPercent = &usage
			}
			if upload >= s.upload {
				r.UploadBps = float64(upload-s.upload) / delta
			}
			if download >= s.download {
				r.DownloadBps = float64(download-s.download) / delta
			}
		}
	}
	s.at = now
	s.cpu = cpu
	s.upload = upload
	s.download = download
	return r
}
