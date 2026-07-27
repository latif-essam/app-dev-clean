package main

import (
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

// goreleaser injects the version via -X main.version. `go install` does not pass
// ldflags, so on that path the version must come from the module info Go records
// in the binary instead of falling back to the literal "dev".
func TestResolveVersion(t *testing.T) {
	buildInfo := func(v string) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Version: v}}, true
		}
	}
	noBuildInfo := func() (*debug.BuildInfo, bool) { return nil, false }

	for _, tc := range []struct {
		name     string
		injected string
		info     func() (*debug.BuildInfo, bool)
		want     string
	}{
		{"ldflag wins over build info", "0.1.0", buildInfo("v9.9.9"), "0.1.0"},
		{"go install falls back to module version", "dev", buildInfo("v0.1.0"), "0.1.0"},
		{"leading v stripped so both paths agree", "dev", buildInfo("v1.2.3"), "1.2.3"},
		{"local go build reports devel, keep dev", "dev", buildInfo("(devel)"), "dev"},
		{"empty module version keeps dev", "dev", buildInfo(""), "dev"},
		{"no build info at all keeps dev", "dev", noBuildInfo, "dev"},
		{"empty injected treated as dev", "", buildInfo("v0.2.0"), "0.2.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveVersion(tc.injected, tc.info); got != tc.want {
				t.Fatalf("resolveVersion(%q) = %q, want %q", tc.injected, got, tc.want)
			}
		})
	}
}

// build the binary once, run --version, assert it prints the injected version.
func TestVersionFlag(t *testing.T) {
	bin := filepath.Join(t.TempDir(), exeName("adc"))
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
