package detectors

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

func TestAndroidDetect(t *testing.T) {
	dir := t.TempDir()
	if (android{}).Detect(dir) {
		t.Fatal("empty dir must not detect as android")
	}
	os.WriteFile(filepath.Join(dir, "settings.gradle"), []byte(""), 0o644)
	if !(android{}).Detect(dir) {
		t.Fatal("settings.gradle must detect as android")
	}
}

func TestAndroidLocalPaths(t *testing.T) {
	got := androidLocalPaths("/proj")
	want := filepath.Join("/proj", ".gradle")
	found := false
	for _, p := range got {
		if p == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("android paths must include %q, got %+v", want, got)
	}
}

// In a native Android repo the project root IS the gradle dir, so paths must
// stay at the root — no "android/" prefix. Guards the RN fix from leaking here.
func TestAndroidTargetStaysAtRoot(t *testing.T) {
	root := filepath.Join("proj", "app")
	ctx := detect.Context{ProjectRoot: root}
	assertPaths(t, targetByName(t, (android{}).Targets(), "android"), ctx,
		[]string{
			filepath.Join(root, "build"),
			filepath.Join(root, "app", "build"),
			filepath.Join(root, ".gradle"),
		},
		[]string{
			filepath.Join(root, "android", "build"),
			filepath.Join(root, "android", ".gradle"),
		})
}

// gradlew clean is best effort: a failing or non-executable wrapper must not
// stop the build directories from being removed.
func TestAndroidCleanContinuesWhenGradlewFails(t *testing.T) {
	if isWindows() {
		t.Skip("uses a POSIX shell wrapper")
	}
	for name, mode := range map[string]os.FileMode{"failing": 0o755, "non-executable": 0o644} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			android := filepath.Join(root, "android")
			build := filepath.Join(android, "app", "build")
			if err := os.MkdirAll(build, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(build, "out.bin"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(android, "gradlew"), []byte("#!/bin/sh\nexit 1\n"), mode); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			previous := clean.Out
			clean.Out = &out
			t.Cleanup(func() { clean.Out = previous })

			target := targetByName(t, (rn{}).Targets(), "android")
			ctx := detect.Context{ProjectRoot: root}
			if err := target.Check(ctx); err != nil {
				t.Fatalf("preflight must not block on gradlew: %v", err)
			}
			if _, err := target.Run(ctx); err != nil {
				t.Fatalf("cleanup must continue after gradlew clean fails: %v", err)
			}
			if exists(build) {
				t.Fatalf("%s must be removed even though gradlew clean failed", build)
			}
			if !strings.Contains(out.String(), "gradlew clean failed") {
				t.Fatalf("expected a skip warning, got:\n%s", out.String())
			}
		})
	}
}
