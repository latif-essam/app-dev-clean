package detectors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/latif-essam/app-dev-clean/internal/detect"
)

func TestExpoDetect(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"),
		[]byte(`{"dependencies":{"expo":"51.0.0"}}`), 0o644)
	if !(expo{}).Detect(dir) {
		t.Fatal("expo dep must detect as expo")
	}
}

func TestExpoTargetsIncludeExpoDir(t *testing.T) {
	found := false
	for _, tg := range (expo{}).Targets() {
		if tg.Name == "expo" {
			found = true
		}
	}
	if !found {
		t.Fatal("expo must offer an 'expo' target cleaning .expo/")
	}
}

// Bare Expo has the same layout as RN after prebuild: native output lives under
// android/ and ios/. See TestRNNativeTargetsUseNativeSubdirs.
func TestExpoNativeTargetsUseNativeSubdirs(t *testing.T) {
	root := filepath.Join("proj", "myapp")
	ctx := detect.Context{ProjectRoot: root}

	assertPaths(t, targetByName(t, (expo{}).Targets(), "android"), ctx,
		[]string{
			filepath.Join(root, "android", "build"),
			filepath.Join(root, "android", "app", "build"),
			filepath.Join(root, "android", ".gradle"),
		},
		[]string{
			filepath.Join(root, "build"),
			filepath.Join(root, "app", "build"),
		})

	assertPaths(t, targetByName(t, (expo{}).Targets(), "ios"), ctx,
		[]string{
			filepath.Join(root, "ios", "build"),
			filepath.Join(root, "ios", "Pods"),
			filepath.Join(root, "ios", "Podfile.lock"),
		},
		[]string{
			filepath.Join(root, "build"),
			filepath.Join(root, "Pods"),
		})
}

// The .expo dirs are project-root relative and must not shift.
func TestExpoDirTargetStaysAtRoot(t *testing.T) {
	root := filepath.Join("proj", "myapp")
	ctx := detect.Context{ProjectRoot: root}
	assertPaths(t, targetByName(t, (expo{}).Targets(), "expo"), ctx,
		[]string{
			filepath.Join(root, ".expo"),
			filepath.Join(root, ".expo-shared"),
		}, nil)
}
