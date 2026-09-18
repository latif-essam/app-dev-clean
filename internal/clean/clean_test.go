package clean

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mkTree(t *testing.T) string {
	dir := t.TempDir()
	sub := filepath.Join(dir, "node_modules", "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "f.js"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSize(t *testing.T) {
	dir := mkTree(t)
	if got := Size(filepath.Join(dir, "node_modules")); got < 2048 {
		t.Fatalf("want >=2048 bytes, got %d", got)
	}
	if got := Size(filepath.Join(dir, "absent")); got != 0 {
		t.Fatalf("absent path must be 0, got %d", got)
	}
}

func TestRemoveReal(t *testing.T) {
	dir := mkTree(t)
	nm := filepath.Join(dir, "node_modules")
	freed, err := Remove(false, dir, nm)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(nm); !os.IsNotExist(err) {
		t.Fatalf("node_modules should be gone")
	}
	if freed < 2048 {
		t.Fatalf("freed should be >=2048, got %d", freed)
	}
}

func TestRemoveDryRun(t *testing.T) {
	dir := mkTree(t)
	nm := filepath.Join(dir, "node_modules")
	freed, err := Remove(true, dir, nm)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(nm); err != nil {
		t.Fatalf("dry-run must NOT delete: %v", err)
	}
	if freed < 2048 {
		t.Fatalf("dry-run should still estimate freed bytes, got %d", freed)
	}
}

func TestRemoveAbsentTolerated(t *testing.T) {
	dir := t.TempDir()
	if freed, err := Remove(false, dir, filepath.Join(dir, "nope")); freed != 0 || err != nil {
		t.Fatalf("absent path must free 0 and not panic, got %d", freed)
	}
}

func TestExecStreamsChildOutput(t *testing.T) {
	if os.Getenv("ADC_EXEC_TEST_HELPER") == "1" {
		fmt.Fprintln(os.Stdout, "install complete")
		fmt.Fprintln(os.Stderr, "npm WARN peer dependency")
		return
	}

	t.Setenv("ADC_EXEC_TEST_HELPER", "1")
	stdout, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	oldStdout, oldStderr, oldOut := os.Stdout, os.Stderr, Out
	os.Stdout, os.Stderr, Out = stdout, stderr, stdout
	t.Cleanup(func() { os.Stdout, os.Stderr, Out = oldStdout, oldStderr, oldOut })

	Exec(false, t.TempDir(), os.Args[0], "-test.run=^TestExecStreamsChildOutput$")
	os.Stdout, os.Stderr, Out = oldStdout, oldStderr, oldOut
	if err := stdout.Close(); err != nil {
		t.Fatal(err)
	}
	if err := stderr.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(stdout.Name())
	if err != nil {
		t.Fatal(err)
	}
	warnings, err := os.ReadFile(stderr.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "install complete") || !strings.Contains(string(warnings), "npm WARN peer dependency") {
		t.Fatalf("child stdout and stderr must be visible; stdout=%q stderr=%q", out, warnings)
	}
}

func TestScopeProtectionAndLeafSymlinks(t *testing.T) {
	dir, outside := mkTree(t), mkTree(t)
	for _, path := range []string{dir, outside, filepath.Join(dir, "..", "escape")} {
		if _, err := Remove(false, dir, path); err == nil {
			t.Fatalf("unsafe path accepted: %s", path)
		}
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := ValidatePaths(dir, filepath.Join(link, "missing")); err == nil {
		t.Fatal("escaping ancestor must be rejected even for missing leaves")
	}
	if _, err := Remove(false, dir, link); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outside, "node_modules", "pkg", "f.js")); err != nil {
		t.Fatal("leaf symlink removal must preserve destination", err)
	}
}

func TestCacheScopeRejectsBroadDirectories(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(home, "dev", "app")
	for _, path := range []string{"relative-cache", home, filepath.Dir(home), project, filepath.Dir(project)} {
		if _, err := CacheScope(path, home, project); err == nil {
			t.Fatalf("broad cache path accepted: %s", path)
		}
	}
}

func TestCacheScopeRejectsPersonalFoldersAndProjectRoots(t *testing.T) {
	home := t.TempDir()
	current := filepath.Join(home, "current")
	for _, name := range []string{"Documents", ".ssh", "Library"} {
		path := filepath.Join(home, name)
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := CacheScope(path, home, current); err == nil {
			t.Errorf("personal folder accepted as cache: %s", name)
		}
	}
	project := filepath.Join(home, "another-project")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CacheScope(project, home, current); err == nil {
		t.Error("another project root accepted as cache")
	}
	file := filepath.Join(home, "credentials")
	if err := os.WriteFile(file, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CacheScope(file, home, current); err == nil {
		t.Error("regular file accepted as cache")
	}
}

func TestProtectTrackedWithoutGit(t *testing.T) {
	dir := mkTree(t)
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "")
	err := ProtectTracked(dir, filepath.Join(dir, "node_modules"))
	if !errors.Is(err, ErrGitUnavailable) {
		t.Fatalf("want ErrGitUnavailable, got %v", err)
	}
	if !strings.Contains(err.Error(), "install Git") {
		t.Fatalf("error must say how to fix it, got %v", err)
	}
}

func TestProtectTrackedOutsideRepoIgnoresMissingGit(t *testing.T) {
	dir := mkTree(t)
	t.Setenv("PATH", "")
	if err := ProtectTracked(dir, filepath.Join(dir, "node_modules")); err != nil {
		t.Fatalf("no repository means no git needed, got %v", err)
	}
}
