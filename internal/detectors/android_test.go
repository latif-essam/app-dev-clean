package detectors

import (
	"os"
	"path/filepath"
	"testing"

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
