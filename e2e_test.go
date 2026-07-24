package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildBin(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "adc")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

func TestRefusesOutsideProject(t *testing.T) {
	bin := buildBin(t)
	cmd := exec.Command(bin, "js")
	cmd.Dir = t.TempDir() // empty, not a project
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit outside project, got success:\n%s", out)
	}
	if !strings.Contains(string(out), "refusing local cleanup") {
		t.Fatalf("want refusal message, got:\n%s", out)
	}
}

func TestDryRunDoesNotDelete(t *testing.T) {
	bin := buildBin(t)
	proj := t.TempDir()
	os.WriteFile(filepath.Join(proj, "package.json"),
		[]byte(`{"dependencies":{"react-native":"0.74.0"}}`), 0o644)
	os.MkdirAll(filepath.Join(proj, "android"), 0o755)
	nm := filepath.Join(proj, "node_modules")
	os.MkdirAll(nm, 0o755)
	os.WriteFile(filepath.Join(nm, "x"), make([]byte, 1024), 0o644)

	cmd := exec.Command(bin, "js", "--dry-run", "-y")
	cmd.Dir = proj
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dry-run failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(nm); err != nil {
		t.Fatalf("dry-run must NOT delete node_modules: %v", err)
	}
	if !strings.Contains(string(out), "dry-run") {
		t.Fatalf("want dry-run output, got:\n%s", out)
	}
}
