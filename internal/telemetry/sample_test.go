package telemetry

import (
	"testing"
	"time"
)

func TestMeasuredCountersAndUnavailablePacketLoss(t *testing.T) {
	var s Sampler
	a := s.Read(0, 0)
	if a.HeapBytes == 0 || a.Goroutines < 1 || a.CPUPercent != nil || a.PacketLoss != nil {
		t.Fatal(a)
	}
	time.Sleep(10 * time.Millisecond)
	b := s.Read(100, 200)
	if b.UploadBps <= 0 || b.DownloadBps <= 0 || b.PacketLoss != nil || b.PacketLossReason == "" {
		t.Fatal(b)
	}
	c := s.Read(0, 0)
	if c.UploadBps < 0 || c.DownloadBps < 0 {
		t.Fatal("negative rates after node restart")
	}
}
