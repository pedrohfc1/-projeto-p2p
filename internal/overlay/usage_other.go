//go:build !windows

package overlay

func childUsage(pid int) (uint64, float64, bool) { return 0, 0, false }
