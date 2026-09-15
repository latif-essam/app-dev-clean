package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/latif-essam/app-dev-clean/internal/detect"

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

func TestRejectInvalidOptions(t *testing.T) {
	for _, args := range [][]string{{"js", "--dryrun", "-y"}, {"--type", "bogus"}, {"--type", "--yes"}} {
		if _, err := parse(args); err == nil {
			t.Errorf("invalid arguments accepted: %v", args)
		}
	}
}

func TestNuclearRequiresProject(t *testing.T) {
	if isGlobalName("nuclear") {
		t.Fatal("nuclear includes local targets and must resolve a project")
	}
}

func TestExpandCombosDeduplicates(t *testing.T) {
	local := []detect.Target{{Name: "js"}, {Name: "ios"}}
	got := expandCombos([]string{"js", "local-all", "ios"}, local)
	if strings.Join(got, ",") != "js,ios" {
		t.Fatalf("each target must run once, got %v", got)
	}
}

func TestReinstallPlansJSBeforePods(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "ios"), 0o755); err != nil {
		t.Fatal(err)
	}
	for file, body := range map[string]string{"package.json": `{}`, "package-lock.json": `{}`, "ios/Podfile": "# fixture", "ios/Podfile.lock": "# fixture"} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(file)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx := detect.Context{ProjectRoot: dir}
	targets := []detect.Target{{Name: "js"}, {Name: "ios", Paths: func(detect.Context) []string { return []string{filepath.Join(dir, "ios", "Pods")} }}}
	actions, err := planReinstalls(ctx, []string{"ios", "js"}, targets)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 2 || actions[0].Target != "js" || actions[1].Target != "ios" {
		t.Fatalf("Pods can depend on node_modules; JS must install first: %+v", actions)
	}
}
