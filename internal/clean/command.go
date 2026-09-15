package clean

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Command resolves tools before executing and handles Windows package-manager
// shims and Gradle batch wrappers explicitly.
func Command(dir, name string, args ...string) (*exec.Cmd, error) {
	if strings.ContainsAny(name, `/\`) && !filepath.IsAbs(name) {
		return nil, fmt.Errorf("executable path must be absolute: %s", name)
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return nil, fmt.Errorf("required command %s is unavailable: %w", name, err)
	}
	cmd, err := platformCommand(path, args...)
	if err != nil {
		return nil, err
	}
	cmd.Dir = dir
	return cmd, nil
}
