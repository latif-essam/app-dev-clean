package cli

import (
	"testing"

	// Register the real detectors/globals so isGlobalName has data to match,
	// mirroring main.go's blank import.
	_ "github.com/latif-essam/app-dev-clean/internal/detectors"
)

func TestParseFlags(t *testing.T) {
	o, err := parse([]string{"ios", "js", "--dry-run", "--type", "rn", "-y"})
	if err != nil {
		t.Fatal(err)
	}
	if !o.DryRun || !o.Yes || o.TypeFilter != "rn" {
		t.Fatalf("flags not parsed: %+v", o)
	}
	if len(o.Targets) != 2 || o.Targets[0] != "ios" {
		t.Fatalf("targets wrong: %+v", o.Targets)
	}
}

func TestParseRootAndVersion(t *testing.T) {
	if o, _ := parse([]string{"--root"}); !o.ShowRoot {
		t.Fatal("--root not parsed")
	}
	if o, _ := parse([]string{"--version"}); !o.Version {
		t.Fatal("--version not parsed")
	}
}

func TestIsGlobalName(t *testing.T) {
	if !isGlobalName("gradle-global") || isGlobalName("ios") {
		t.Fatal("isGlobalName wrong")
	}
}
