package easytierlab

import (
	"golang.org/x/sys/windows"
	"unsafe"
)

type memoryCounters struct {
	Size, PageFaults                                                                             uint32
	PeakWorkingSet, WorkingSet, PeakPaged, Paged, PeakNonPaged, NonPaged, PageFile, PeakPageFile uintptr
}

func helperUsage(pid int) (uint64, float64, bool) {
	handle, e := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, uint32(pid))
	if e != nil {
		return 0, 0, false
	}
	defer windows.CloseHandle(handle)
	var created, exited, kernel, user windows.Filetime
	e = windows.GetProcessTimes(handle, &created, &exited, &kernel, &user)
	ticks := func(f windows.Filetime) uint64 { return uint64(f.HighDateTime)<<32 | uint64(f.LowDateTime) }
	cpu := float64(ticks(kernel)+ticks(user)) / 1e7
	m := memoryCounters{}
	m.Size = uint32(unsafe.Sizeof(m))
	proc := windows.NewLazySystemDLL("psapi.dll").NewProc("GetProcessMemoryInfo")
	ok, _, _ := proc.Call(uintptr(handle), uintptr(unsafe.Pointer(&m)), uintptr(m.Size))
	var rss uint64
	if ok != 0 {
		rss = uint64(m.WorkingSet)
	}
	return rss, cpu, e == nil
}
