package detectors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/latif-essam/app-dev-clean/internal/detect"
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

// In a native Xcode/SwiftPM repo the project root IS the build dir.
func TestIOSTargetStaysAtRoot(t *testing.T) {
	root := filepath.Join("proj", "App")
	ctx := detect.Context{ProjectRoot: root}
	assertPaths(t, targetByName(t, (ios{}).Targets(), "ios"), ctx,
		[]string{
			filepath.Join(root, "build"),
			filepath.Join(root, "Pods"),
			filepath.Join(root, "Podfile.lock"),
			filepath.Join(root, ".build"),
		},
		[]string{
			filepath.Join(root, "ios", "build"),
			filepath.Join(root, "ios", "Pods"),
		})
}
