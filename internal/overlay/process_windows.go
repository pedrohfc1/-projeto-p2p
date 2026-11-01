package overlay

import (
	"golang.org/x/sys/windows"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"
)

func prepareProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
}
func isolateProcess(cmd *exec.Cmd) (func(), error) {
	j, e := windows.CreateJobObject(nil, nil)
	if e != nil {
		return nil, e
	}
	fail := func(e error) (func(), error) { windows.CloseHandle(j); return nil, e }
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_PROCESS_MEMORY
	info.ProcessMemoryLimit = 256 << 20
	if _, e = windows.SetInformationJobObject(j, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); e != nil {
		return fail(e)
	}
	cpu := struct {
		Flags uint32
		Rate  uint32
	}{5, 2500}
	if _, e = windows.SetInformationJobObject(j, windows.JobObjectCpuRateControlInformation, uintptr(unsafe.Pointer(&cpu)), 8); e != nil {
		return fail(e)
	}
	h, e := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if e != nil {
		return fail(e)
	}
	defer windows.CloseHandle(h)
	if e = windows.AssignProcessToJobObject(j, h); e != nil {
		return fail(e)
	}
	var once sync.Once
	return func() { once.Do(func() { windows.CloseHandle(j) }) }, nil
}
