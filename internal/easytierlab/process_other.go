//go:build !windows

package easytierlab

import (
	"errors"
	"os/exec"
)

func prepareProcess(cmd *exec.Cmd) {}
func isolateProcess(cmd *exec.Cmd) (func(), error) {
	return nil, errors.New("experimental helper pinned and isolated only on Windows in this prototype")
}
