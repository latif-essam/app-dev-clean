package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type report struct {
	Version    string  `json:"version"`
	DryRun     bool    `json:"dryRun"`
	Executed   bool    `json:"executed"`
	FreedTotal int64   `json:"freedTotal"`
	Error      *string `json:"error"`
	Project    *struct {
		Root  string   `json:"root"`
		Types []string `json:"types"`
	} `json:"project"`
	Targets []struct {
		Name  string   `json:"name"`
		Scope string   `json:"scope"`
		Paths []string `json:"paths"`
		Size  int64    `json:"size"`
		Freed int64    `json:"freed"`
	} `json:"targets"`
}

// jsonRun isolates HOME so global caches resolve inside the test, never the
// machine's real ones.
func jsonRun(t *testing.T, bin, dir string, args ...string) (report, string, error) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	var errBuf strings.Builder
	cmd.Stderr = &errBuf
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir(), "TMPDIR="+t.TempDir())
	out, err := cmd.Output()
	var r report
	if jsonErr := json.Unmarshal(out, &r); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON, got %q (stderr %q)", out, errBuf.String())
	}
	return r, errBuf.String(), err
}

func TestJSONPlanDeletesNothing(t *testing.T) {
	bin := buildBin(t)
	dir := safetyProject(t)
	r, _, err := jsonRun(t, bin, dir, "--json")
	if err != nil {
		t.Fatalf("plan must succeed: %v", err)
	}
	if r.Executed || r.FreedTotal != 0 || r.Error != nil {
		t.Fatalf("plan must not execute: %+v", r)
	}
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if r.Project == nil || r.Project.Root != want {
		t.Fatalf("plan must name the project %s, got %+v", want, r.Project)
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules", "fixture")); err != nil {
		t.Fatalf("plan deleted dependencies: %v", err)
	}
	var js bool
	for _, tg := range r.Targets {
		if tg.Name == "js" {
			js = true
			if tg.Size == 0 || tg.Freed != 0 {
				t.Fatalf("js must report a size and no freed bytes: %+v", tg)
			}
		}
	}
	if !js {
		t.Fatalf("plan must list js, got %+v", r.Targets)
	}
}

func TestJSONDryRunKeepsStdoutParseable(t *testing.T) {
	bin := buildBin(t)
	dir := safetyProject(t)
	r, stderr, err := jsonRun(t, bin, dir, "js", "--json", "--dry-run")
	if err != nil {
		t.Fatalf("dry run must succeed: %v", err)
	}
	if !r.DryRun || !r.Executed || r.FreedTotal == 0 {
		t.Fatalf("dry run must report a plan: %+v", r)
	}
	if len(r.Targets) != 1 || r.Targets[0].Freed != 0 {
		t.Fatalf("a dry run frees nothing: %+v", r.Targets)
	}
	if !strings.Contains(stderr, "dry-run") {
		t.Fatalf("progress belongs on stderr, got %q", stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules", "fixture")); err != nil {
		t.Fatalf("dry run deleted dependencies: %v", err)
	}
}

func TestJSONRunReportsFreedBytes(t *testing.T) {
	bin := buildBin(t)
	dir := safetyProject(t)
	r, _, err := jsonRun(t, bin, dir, "js", "--json")
	if err != nil {
		t.Fatalf("cleanup must succeed: %v", err)
	}
	if r.FreedTotal == 0 || len(r.Targets) != 1 || r.Targets[0].Freed != r.FreedTotal {
		t.Fatalf("freed bytes must be reported: %+v", r)
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules")); !os.IsNotExist(err) {
		t.Fatalf("node_modules must be gone, got %v", err)
	}
}

func TestJSONReportsFailures(t *testing.T) {
	bin := buildBin(t)
	cases := []struct {
		name string
		dir  func(*testing.T) string
		args []string
	}{
		{"outside a project", func(t *testing.T) string { return t.TempDir() }, []string{"js", "--json"}},
		{"shared without consent", safetyProject, []string{"metro", "--json"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, _, err := jsonRun(t, bin, c.dir(t), c.args...)
			if err == nil {
				t.Fatal("a refused run must exit non-zero")
			}
			if r.Error == nil || *r.Error == "" {
				t.Fatalf("a refused run must explain itself: %+v", r)
			}
			if r.Executed {
				t.Fatalf("a refused run must not report execution: %+v", r)
			}
		})
	}
}
