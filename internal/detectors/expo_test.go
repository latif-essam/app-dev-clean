package detectors

import (
	"os"
	"path/filepath"
	"testing"
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
