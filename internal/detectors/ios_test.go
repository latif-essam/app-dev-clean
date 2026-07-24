package detectors

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIOSDetect(t *testing.T) {
	dir := t.TempDir()
	if (ios{}).Detect(dir) {
		t.Fatal("empty dir must not detect as ios")
	}
	os.MkdirAll(filepath.Join(dir, "App.xcodeproj"), 0o755)
	if !(ios{}).Detect(dir) {
		t.Fatal(".xcodeproj must detect as ios")
	}
}

func TestIOSLocalPaths(t *testing.T) {
	got := iosLocalPaths("/proj")
	want := filepath.Join("/proj", "Pods")
	found := false
	for _, p := range got {
		if p == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("ios paths must include %q, got %+v", want, got)
	}
}
