package desktop

import (
	"os/exec"
	"syscall"
)

func hideTestProcess(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
