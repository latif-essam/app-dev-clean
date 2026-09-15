package clean

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func platformCommand(path string, args ...string) (*exec.Cmd, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".cmd" && ext != ".bat" {
		return exec.Command(path, args...), nil
	}
	// Batch parsing differs from CommandLineToArgvW. Quote every word and reject
	// expansion/quote characters rather than letting project paths become code.
	words := append([]string{path}, args...)
	for i, word := range words {
		if strings.ContainsAny(word, "%!\"\r\n\x00") {
			return nil, fmt.Errorf("unsupported batch command path/argument containing expansion characters: %q", word)
		}
		words[i] = `"` + word + `"`
	}
	shell := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	if !filepath.IsAbs(shell) {
		return nil, fmt.Errorf("SystemRoot is unavailable; cannot safely run batch commands")
	}
	cmd := exec.Command(shell)
	cmd.Args = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `"` + shell + `" /d /v:off /s /c "` + strings.Join(words, " ") + `"`}
	return cmd, nil
}
