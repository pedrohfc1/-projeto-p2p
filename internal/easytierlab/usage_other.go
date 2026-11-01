//go:build !windows

package easytierlab

func helperUsage(pid int) (uint64, float64, bool) { return 0, 0, false }
