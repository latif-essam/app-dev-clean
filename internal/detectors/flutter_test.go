package detectors

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFlutterDetect(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pubspec.yaml"),
		[]byte("name: app\ndependencies:\n  flutter:\n    sdk: flutter\n"), 0o644)
	if !(flutter{}).Detect(dir) {
		t.Fatal("pubspec with flutter sdk must detect")
	}
	plain := t.TempDir()
	os.WriteFile(filepath.Join(plain, "pubspec.yaml"), []byte("name: pkg\n"), 0o644)
	if (flutter{}).Detect(plain) {
		t.Fatal("dart-only pubspec (no flutter) must NOT detect as flutter")
	}
}
