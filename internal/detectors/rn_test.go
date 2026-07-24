package detectors

import (
	"os"
	"path/filepath"
	"testing"
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
