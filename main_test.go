package main

import (
	"os/exec"
	"strings"
	"testing"
)

// build the binary once, run --version, assert it prints the injected version.
func TestVersionFlag(t *testing.T) {
	bin := t.TempDir() + "/adc"
	build := exec.Command("go", "build", "-ldflags", "-X main.version=9.9.9", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	out, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "9.9.9") {
		t.Fatalf("want version 9.9.9 in output, got %q", out)
	}
}
