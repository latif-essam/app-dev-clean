package clean

import (
	"os"
	"path/filepath"
	"testing"
)

func mkTree(t *testing.T) string {
	dir := t.TempDir()
	sub := filepath.Join(dir, "node_modules", "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "f.js"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSize(t *testing.T) {
	dir := mkTree(t)
	if got := Size(filepath.Join(dir, "node_modules")); got < 2048 {
		t.Fatalf("want >=2048 bytes, got %d", got)
	}
	if got := Size(filepath.Join(dir, "absent")); got != 0 {
		t.Fatalf("absent path must be 0, got %d", got)
	}
}

func TestRemoveReal(t *testing.T) {
	dir := mkTree(t)
	nm := filepath.Join(dir, "node_modules")
	freed := Remove(false, nm)
	if _, err := os.Stat(nm); !os.IsNotExist(err) {
		t.Fatalf("node_modules should be gone")
	}
	if freed < 2048 {
		t.Fatalf("freed should be >=2048, got %d", freed)
	}
}

func TestRemoveDryRun(t *testing.T) {
	dir := mkTree(t)
	nm := filepath.Join(dir, "node_modules")
	freed := Remove(true, nm)
	if _, err := os.Stat(nm); err != nil {
		t.Fatalf("dry-run must NOT delete: %v", err)
	}
	if freed < 2048 {
		t.Fatalf("dry-run should still estimate freed bytes, got %d", freed)
	}
}

func TestRemoveAbsentTolerated(t *testing.T) {
	if freed := Remove(false, filepath.Join(t.TempDir(), "nope")); freed != 0 {
		t.Fatalf("absent path must free 0 and not panic, got %d", freed)
	}
}
