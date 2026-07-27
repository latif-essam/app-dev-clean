package detectors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/latif-essam/app-dev-clean/internal/detect"
)

func writeRNApp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"),
		[]byte(`{"dependencies":{"react-native":"0.74.0"}}`), 0o644)
	os.MkdirAll(filepath.Join(dir, "android"), 0o755)
	os.MkdirAll(filepath.Join(dir, "ios"), 0o755)
	return dir
}

func TestRNDetect(t *testing.T) {
	if !(rn{}).Detect(writeRNApp(t)) {
		t.Fatal("RN app must detect")
	}
	plain := t.TempDir()
	os.WriteFile(filepath.Join(plain, "package.json"), []byte(`{"dependencies":{"express":"4"}}`), 0o644)
	if (rn{}).Detect(plain) {
		t.Fatal("plain node app must NOT detect as rn")
	}
}

func TestRNTargetsIncludeJS(t *testing.T) {
	names := map[string]bool{}
	for _, tg := range (rn{}).Targets() {
		names[tg.Name] = true
	}
	for _, want := range []string{"js", "metro", "watchman", "android", "ios"} {
		if !names[want] {
			t.Fatalf("rn targets missing %q; got %v", want, names)
		}
	}
}

// An RN project keeps its native build output under android/ and ios/, not at
// the project root — unlike a native Android/Xcode repo, where the root IS the
// build dir. Reusing the native path builders unchanged made these targets
// resolve to <root>/app/build and silently clean nothing.
func TestRNNativeTargetsUseNativeSubdirs(t *testing.T) {
	root := filepath.Join("proj", "myapp")
	ctx := detect.Context{ProjectRoot: root}

	assertPaths(t, targetByName(t, (rn{}).Targets(), "android"), ctx,
		[]string{
			filepath.Join(root, "android", "build"),
			filepath.Join(root, "android", "app", "build"),
			filepath.Join(root, "android", ".gradle"),
			filepath.Join(root, "android", ".cxx"),
			filepath.Join(root, "android", "app", ".cxx"),
		},
		[]string{
			filepath.Join(root, "build"),
			filepath.Join(root, "app", "build"),
			filepath.Join(root, ".gradle"),
		})

	assertPaths(t, targetByName(t, (rn{}).Targets(), "ios"), ctx,
		[]string{
			filepath.Join(root, "ios", "build"),
			filepath.Join(root, "ios", "Pods"),
			filepath.Join(root, "ios", "Podfile.lock"),
		},
		[]string{
			filepath.Join(root, "build"),
			filepath.Join(root, "Pods"),
			filepath.Join(root, "Podfile.lock"),
		})
}

// js/metro stay at the project root — only the native targets shift.
func TestRNJSTargetStaysAtRoot(t *testing.T) {
	root := filepath.Join("proj", "myapp")
	ctx := detect.Context{ProjectRoot: root}
	assertPaths(t, targetByName(t, (rn{}).Targets(), "js"), ctx,
		[]string{
			filepath.Join(root, "node_modules"),
			filepath.Join(root, "package-lock.json"),
		}, nil)
}
