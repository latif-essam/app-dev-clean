package detectors

import (
	"os"
	"path/filepath"
	"testing"
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
