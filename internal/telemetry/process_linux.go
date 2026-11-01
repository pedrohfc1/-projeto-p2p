package telemetry

import (
	"golang.org/x/sys/unix"
	"os"
	"strconv"
	"strings"
)

func process() (uint64, float64, bool) {
	var usage unix.Rusage
	e := unix.Getrusage(unix.RUSAGE_SELF, &usage)
	cpu := float64(usage.Utime.Sec+usage.Stime.Sec) + float64(usage.Utime.Usec+usage.Stime.Usec)/1e6
	var rss uint64
	if b, err := os.ReadFile("/proc/self/statm"); err == nil {
		f := strings.Fields(string(b))
		if len(f) > 1 {
			n, _ := strconv.ParseUint(f[1], 10, 64)
			rss = n * uint64(os.Getpagesize())
		}
	}
	return rss, cpu, e == nil
}
