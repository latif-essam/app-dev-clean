//go:build !windows

package clean

import "os/exec"

func platformCommand(path string, args ...string) (*exec.Cmd, error) {
	return exec.Command(path, args...), nil
}
