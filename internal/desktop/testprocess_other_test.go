//go:build !windows

package desktop

import "os/exec"

func hideTestProcess(c *exec.Cmd) {}
