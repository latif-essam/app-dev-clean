package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func safetyProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, p := range []string{"android/build", "node_modules"} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(p)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for p, body := range map[string]string{
		"package.json":         `{"dependencies":{"react-native":"0.74.0"}}`,
		"package-lock.json":    `{"lockfileVersion":3,"packages":{}}`,
		"node_modules/fixture": "dependency", "android/build/fixture": "build output",
	} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(p)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestSafetyBeforeCleanup(t *testing.T) {
	bin := buildBin(t)
	for _, args := range [][]string{{"js", "--dryrun", "-y"}, {"js", "unknown-target", "-y"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			dir := safetyProject(t)
			cmd := exec.Command(bin, args...)
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("invalid input must fail before cleanup: %s", out)
			}
			if _, err := os.Stat(filepath.Join(dir, "node_modules", "fixture")); err != nil {
				t.Fatalf("invalid input deleted dependencies: %v", err)
			}
		})
	}
	t.Run("lockfile preserved", func(t *testing.T) {
		dir := safetyProject(t)
		lock := filepath.Join(dir, "package-lock.json")
		before, _ := os.ReadFile(lock)
		cmd := exec.Command(bin, "js", "-y")
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("cleanup: %v %s", err, out)
		}
		after, err := os.ReadFile(lock)
		if err != nil || string(after) != string(before) {
			t.Fatalf("lockfile must remain unchanged: %v", err)
		}
	})
	t.Run("symlink ancestor rejected", func(t *testing.T) {
		dir := safetyProject(t)
		outside := t.TempDir()
		if err := os.Mkdir(filepath.Join(outside, "Pods"), 0o755); err != nil {
			t.Fatal(err)
		}
		sentinel := filepath.Join(outside, "Pods", "fixture")
		if err := os.WriteFile(sentinel, []byte("preserve"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(dir, "ios")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		cmd := exec.Command(bin, "js", "ios", "-y")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("escaping path must fail: %s", out)
		}
		for _, p := range []string{sentinel, filepath.Join(dir, "node_modules", "fixture")} {
			if _, err := os.Stat(p); err != nil {
				t.Fatalf("preflight must protect every target: %s: %v", p, err)
			}
		}
	})
	t.Run("failed command preserves remaining artifacts", func(t *testing.T) {
		dir := safetyProject(t)
		wrapper, body := "gradlew", "#!/bin/sh\necho fixture failure >&2\nexit 7\n"
		if runtime.GOOS == "windows" {
			wrapper, body = "gradlew.bat", "@echo off\r\necho fixture failure 1>&2\r\nexit /b 7\r\n"
		}
		if err := os.WriteFile(filepath.Join(dir, "android", wrapper), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(bin, "android", "-y")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err == nil || strings.Contains(string(out), "Done. Reclaimed") || !strings.Contains(string(out), "fixture failure") {
			t.Fatalf("failed external command must fail ADC: %v %s", err, out)
		}
		if _, err := os.Stat(filepath.Join(dir, "android", "build", "fixture")); err != nil {
			t.Fatalf("command failure must stop target cleanup: %v", err)
		}
	})
	t.Run("nuclear and nested root resolution", func(t *testing.T) {
		dir := safetyProject(t)
		nested := filepath.Join(dir, "android", "app", "src")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(dir, "android", "settings.gradle"), []byte("// fixture"), 0o644)
		os.WriteFile(filepath.Join(dir, "android", "app", "build.gradle"), []byte("// fixture"), 0o644)
		for _, args := range [][]string{{"--root"}, {"nuclear", "--root"}} {
			cmd := exec.Command(bin, args...)
			cmd.Dir = nested
			out, err := cmd.CombinedOutput()
			if err != nil || !strings.Contains(string(out), dir+" (rn)") {
				t.Fatalf("must resolve owning app: %v %s", err, out)
			}
		}
	})
}

func fakeNPM(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	name := "npm"
	script := `#!/bin/sh
if [ "$1" = "--version" ]; then
  echo 10.8.2
  exit 0
fi
echo "$*" >> "$ADC_FIXTURE_LOG"
echo 'fixture npm output'
echo 'npm WARN fixture warning' >&2
exit "${ADC_FIXTURE_EXIT:-0}"
`
	if runtime.GOOS == "windows" {
		name = "npm.cmd"
		script = "@echo off\r\nif \"%~1\"==\"--version\" (\r\n echo 10.8.2\r\n exit /b 0\r\n)\r\necho %*>>\"%ADC_FIXTURE_LOG%\"\r\necho fixture npm output\r\necho npm WARN fixture warning 1>&2\r\nif defined ADC_FIXTURE_EXIT exit /b %ADC_FIXTURE_EXIT%\r\nexit /b 0\r\n"
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestInstallAndSharedSafety(t *testing.T) {
	bin := buildBin(t)
	t.Run("Expo installs once and streams both channels", func(t *testing.T) {
		dir := safetyProject(t)
		os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"expo":"51","react-native":"0.74"},"packageManager":"npm@10.8.2"}`), 0o644)
		log := filepath.Join(t.TempDir(), "installs")
		t.Setenv("PATH", fakeNPM(t)+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("ADC_FIXTURE_LOG", log)
		cmd := exec.Command(bin, "js", "js", "--type", "expo", "--reinstall", "-y")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("reinstall failed: %v %s", err, out)
		}
		calls, err := os.ReadFile(log)
		if err != nil || strings.Count(string(calls), "ci") != 1 || !strings.Contains(string(calls), "--prefer-offline") {
			t.Fatalf("expected one cache-aware install: %q %v", calls, err)
		}
		for _, want := range []string{"fixture npm output", "npm WARN fixture warning"} {
			if !strings.Contains(string(out), want) {
				t.Errorf("missing child output %q: %s", want, out)
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "package-lock.json")); err != nil {
			t.Fatal("lockfile removed", err)
		}
	})
	t.Run("install failure exits nonzero", func(t *testing.T) {
		dir := safetyProject(t)
		t.Setenv("PATH", fakeNPM(t)+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("ADC_FIXTURE_LOG", filepath.Join(t.TempDir(), "installs"))
		t.Setenv("ADC_FIXTURE_EXIT", "7")
		cmd := exec.Command(bin, "js", "--reinstall", "-y")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), "reinstall failed") || strings.Contains(string(out), "Done. Reclaimed") {
			t.Fatalf("install failure hidden: %v %s", err, out)
		}
	})
	t.Run("missing installer blocks cleanup", func(t *testing.T) {
		dir := safetyProject(t)
		t.Setenv("PATH", t.TempDir())
		cmd := exec.Command(bin, "js", "--reinstall", "-y")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), "nothing deleted") {
			t.Fatalf("missing installer accepted: %v %s", err, out)
		}
		if _, err := os.Stat(filepath.Join(dir, "node_modules", "fixture")); err != nil {
			t.Fatal("dependencies removed before installer check", err)
		}
	})
	t.Run("dry run executes no installer", func(t *testing.T) {
		dir := safetyProject(t)
		t.Setenv("PATH", t.TempDir())
		cmd := exec.Command(bin, "js", "--reinstall", "--dry-run", "-y")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(out), "npm ci --prefer-offline") {
			t.Fatalf("dry-run plan failed: %v %s", err, out)
		}
		if _, err := os.Stat(filepath.Join(dir, "node_modules", "fixture")); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("shared caches require explicit consent", func(t *testing.T) {
		dir := safetyProject(t)
		cache := t.TempDir()
		sentinel := filepath.Join(cache, "preserve")
		os.WriteFile(sentinel, []byte("cached download"), 0o644)
		t.Setenv("PUB_CACHE", cache)
		cmd := exec.Command(bin, "js", "pub-cache", "-y")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), "--allow-shared") {
			t.Fatalf("shared consent bypassed: %v %s", err, out)
		}
		for _, p := range []string{sentinel, filepath.Join(dir, "node_modules", "fixture")} {
			if _, err := os.Stat(p); err != nil {
				t.Fatal("data deleted without shared consent", err)
			}
		}
	})
	t.Run("tracked artifacts block every selected target", func(t *testing.T) {
		dir := safetyProject(t)
		for _, args := range [][]string{{"init", "--quiet"}, {"add", "android/build/fixture"}} {
			cmd := exec.Command("git", args...)
			cmd.Dir = dir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("fixture git: %v %s", err, out)
			}
		}
		cmd := exec.Command(bin, "js", "android", "-y")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), "tracked data") {
			t.Fatalf("tracked data unprotected: %v %s", err, out)
		}
		if _, err := os.Stat(filepath.Join(dir, "node_modules", "fixture")); err != nil {
			t.Fatal("earlier target ran before tracked-file check", err)
		}
	})
}

func TestVersionAndWorkspacePreflight(t *testing.T) {
	bin := buildBin(t)
	for _, kind := range []string{"manager version mismatch", "conflicting locks", "workspace", "type mismatch"} {
		t.Run(kind, func(t *testing.T) {
			dir := safetyProject(t)
			args := []string{"js", "--reinstall", "-y"}
			switch kind {
			case "manager version mismatch":
				t.Setenv("PATH", fakeNPM(t)+string(os.PathListSeparator)+os.Getenv("PATH"))
				os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"react-native":"0.74"},"packageManager":"npm@9.0.0"}`), 0o644)
			case "conflicting locks":
				os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lockfileVersion: '9.0'"), 0o644)
			case "workspace":
				os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"react-native":"0.74"},"workspaces":["packages/*"]}`), 0o644)
			case "type mismatch":
				args = []string{"js", "--type", "flutter", "-y"}
			}
			cmd := exec.Command(bin, args...)
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("invalid cleanup accepted: %s", out)
			}
			if _, err := os.Stat(filepath.Join(dir, "node_modules", "fixture")); err != nil {
				t.Fatal("preflight deleted dependencies", err)
			}
		})
	}
}

func withoutGit(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH=")
	return cmd
}

func TestMissingGitBlocksCleanupButNotPreview(t *testing.T) {
	bin := buildBin(t)
	t.Run("dry run previews", func(t *testing.T) {
		dir := safetyProject(t)
		if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		out, err := withoutGit(dir, bin, "js", "--dry-run", "-y").CombinedOutput()
		if err != nil {
			t.Fatalf("dry run must survive a missing git: %v\n%s", err, out)
		}
		if !strings.Contains(string(out), "git is unavailable") {
			t.Fatalf("want a warning about git, got:\n%s", out)
		}
		if _, err := os.Stat(filepath.Join(dir, "node_modules", "fixture")); err != nil {
			t.Fatalf("dry run deleted dependencies: %v", err)
		}
	})
	t.Run("cleanup refuses", func(t *testing.T) {
		dir := safetyProject(t)
		if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		out, err := withoutGit(dir, bin, "js", "-y").CombinedOutput()
		if err == nil {
			t.Fatalf("cleanup must refuse when tracked files cannot be checked:\n%s", out)
		}
		if _, err := os.Stat(filepath.Join(dir, "node_modules", "fixture")); err != nil {
			t.Fatalf("refused cleanup still deleted dependencies: %v", err)
		}
	})
}
