//go:build !windows && !linux

package telemetry

func process() (uint64, float64, bool) { return 0, 0, false }
