# app-dev-clean Cross-Platform Rewrite — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite `app-dev-clean` from macOS-only bash into a single cross-platform Go binary that detects the project type(s) at the resolved root and cleans the right dev caches on Windows, macOS, and Linux.

**Architecture:** A tiny `main` wires an `internal/cli` dispatcher over four leaf/near-leaf packages: `platform` (per-OS cache paths), `detect` (Detector interface + registry + walk-up resolver + Target/Context types), `clean` (size/remove/exec engine with dry-run), and `detectors` (rn/android/ios/flutter/expo, registered at init). An `internal/ui` bubbletea checklist feeds the same dispatch path as CLI args. Distribution is goreleaser + GitHub Actions publishing to Homebrew tap, Scoop bucket, GitHub Releases, and curl/irm install scripts.

**Tech Stack:** Go 1.23+, `github.com/charmbracelet/bubbletea` + `lipgloss` (TUI), goreleaser, GitHub Actions. Test harness: stdlib `testing` (`go test ./...`).

## Global Constraints

- Binary + module name: `app-dev-clean`; module path `github.com/latif-essam/app-dev-clean`. Optional `adc` alias symlink (convenience only).
- License: MIT © Latif Essam.
- Go 1.23+. No non-stdlib deps except `charmbracelet/bubbletea` + `charmbracelet/lipgloss`.
- Local targets operate on the resolved `ProjectRoot`, never raw cwd. If no registered project root is found up-tree, refuse local cleanup and exit non-zero.
- Global cache targets (`gradle-global`, `xcode-dd`, `pods-cache`, `pub-cache`) are allowed with no project but always confirm (unless `--yes`). `nuclear` always confirms.
- macOS-only caches (Xcode DerivedData, CocoaPods) must resolve to `""` on Windows/Linux and their targets must not be offered there.
- `--dry-run` must delete nothing AND must not run destructive external commands (`gradlew clean`, `flutter clean`); it only reports paths/commands.
- Deletion always flows through `clean.Remove` so dry-run + size accounting stay uniform. Absent paths and per-path errors are tolerated (log, continue) — never abort a run.
- Work on branch `go-rewrite`. Keep bash `main` installable until parity + green CI, then merge and remove the bash tree.
- Commit after every task. Conventional Commits. End commit messages with the `Co-Authored-By` trailer used by this repo.

---

### Task 1: Go module + main skeleton + version

**Files:**
- Create: `go.mod` (via `go mod init`)
- Create: `main.go`
- Create: `internal/cli/cli.go` (stub)
- Test: `main_test.go`

**Interfaces:**
- Produces: `cli.Run(args []string, version string) int` — entrypoint returning an exit code. `main.version` string (overridden at build via ldflags).

- [ ] **Step 1: Init the module**

Run from repo root (`~/dev-tools/devclean`, already on branch `go-rewrite`):
```bash
go mod init github.com/latif-essam/app-dev-clean
go mod edit -go=1.23
```

- [ ] **Step 2: Write the failing test**

`main_test.go`:
```go
package main

import (
	"os/exec"
	"strings"
	"testing"
)

// build the binary once, run --version, assert it prints the injected version.
func TestVersionFlag(t *testing.T) {
	bin := t.TempDir() + "/adc"
	build := exec.Command("go", "build", "-ldflags", "-X main.version=9.9.9", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	out, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "9.9.9") {
		t.Fatalf("want version 9.9.9 in output, got %q", out)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test . -run TestVersionFlag -v`
Expected: FAIL (build error: `cli` package / `Run` undefined).

- [ ] **Step 4: Write minimal implementation**

`internal/cli/cli.go`:
```go
package cli

import "fmt"

// Run is the CLI entrypoint. Returns a process exit code.
func Run(args []string, version string) int {
	if len(args) == 1 && (args[0] == "--version" || args[0] == "-v") {
		fmt.Println("app-dev-clean", version)
		return 0
	}
	fmt.Println("app-dev-clean", version, "(not yet implemented)")
	return 0
}
```

`main.go`:
```go
package main

import (
	"os"

	"github.com/latif-essam/app-dev-clean/internal/cli"
)

var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], version))
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test . -run TestVersionFlag -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add go.mod main.go main_test.go internal/cli/cli.go
git commit -m "feat: go module skeleton + --version

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: `platform` — per-OS cache paths

**Files:**
- Create: `internal/platform/paths.go`
- Test: `internal/platform/paths_test.go`

**Interfaces:**
- Produces:
  - `type Paths struct { Home, GradleCache, XcodeDD, CocoaPods, PubCache, TmpDir string }` — `XcodeDD`/`CocoaPods` are `""` when unavailable on the OS.
  - `func Detect() Paths` — resolves from `runtime.GOOS` + real env.
  - `func detectFor(goos string, env func(string) string) Paths` — testable core.

- [ ] **Step 1: Write the failing test**

`internal/platform/paths_test.go`:
```go
package platform

import "testing"

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestDetectForDarwin(t *testing.T) {
	p := detectFor("darwin", envFrom(map[string]string{"HOME": "/Users/x"}))
	if p.XcodeDD == "" || p.CocoaPods == "" {
		t.Fatalf("darwin must have xcode+cocoapods paths, got %+v", p)
	}
	if p.GradleCache != "/Users/x/.gradle/caches" {
		t.Fatalf("gradle path wrong: %q", p.GradleCache)
	}
}

func TestDetectForLinux(t *testing.T) {
	p := detectFor("linux", envFrom(map[string]string{"HOME": "/home/x"}))
	if p.XcodeDD != "" || p.CocoaPods != "" {
		t.Fatalf("linux must NOT have xcode/cocoapods, got %+v", p)
	}
	if p.PubCache != "/home/x/.pub-cache" {
		t.Fatalf("pub path wrong: %q", p.PubCache)
	}
}

func TestDetectForWindows(t *testing.T) {
	p := detectFor("windows", envFrom(map[string]string{
		"USERPROFILE": `C:\Users\x`, "LOCALAPPDATA": `C:\Users\x\AppData\Local`, "TEMP": `C:\Temp`,
	}))
	if p.XcodeDD != "" || p.CocoaPods != "" {
		t.Fatalf("windows must NOT have xcode/cocoapods")
	}
	if p.TmpDir != `C:\Temp` {
		t.Fatalf("windows tmp wrong: %q", p.TmpDir)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/ -v`
Expected: FAIL (`detectFor` undefined).

- [ ] **Step 3: Write minimal implementation**

`internal/platform/paths.go`:
```go
package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

type Paths struct {
	Home        string
	GradleCache string
	XcodeDD     string // "" when unavailable on this OS
	CocoaPods   string // "" when unavailable on this OS
	PubCache    string
	TmpDir      string
}

func Detect() Paths { return detectFor(runtime.GOOS, os.Getenv) }

func detectFor(goos string, env func(string) string) Paths {
	home := env("HOME")
	if goos == "windows" {
		home = env("USERPROFILE")
	}
	p := Paths{Home: home}
	p.GradleCache = filepath.Join(home, ".gradle", "caches")
	switch goos {
	case "windows":
		p.PubCache = filepath.Join(env("LOCALAPPDATA"), "Pub", "Cache")
		p.TmpDir = env("TEMP")
	case "darwin":
		p.XcodeDD = filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData")
		p.CocoaPods = filepath.Join(home, "Library", "Caches", "CocoaPods")
		p.PubCache = filepath.Join(home, ".pub-cache")
		p.TmpDir = tmpOr(env, "/tmp")
	default: // linux and others
		p.PubCache = filepath.Join(home, ".pub-cache")
		p.TmpDir = tmpOr(env, "/tmp")
	}
	return p
}

func tmpOr(env func(string) string, def string) string {
	if t := env("TMPDIR"); t != "" {
		return t
	}
	return def
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/ -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/platform/
git commit -m "feat: per-OS cache path resolution

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: `detect` core — types + registry

**Files:**
- Create: `internal/detect/detector.go`
- Test: `internal/detect/detector_test.go`

**Interfaces:**
- Consumes: `platform.Paths` (Task 2).
- Produces:
  - `type Scope int` with `const ( Local Scope = iota; Global )`.
  - `type Context struct { ProjectRoot string; Paths platform.Paths; DryRun, Yes, Force bool }` (`Force` = nuclear reinstall without prompting).
  - `type Target struct { Name, Label, Desc string; Scope Scope; Paths func(Context) []string; Run func(Context) (freed int64, err error) }`.
  - `type Detector interface { Name() string; Detect(dir string) bool; Targets() []Target }`.
  - `type PostRunner interface { PostRun(ctx Context, ran []string) error }` (optional, type-asserted).
  - `func Register(Detector)` / `func Detectors() []Detector`.
  - `func RegisterGlobal(Target)` / `func Globals() []Target`.

- [ ] **Step 1: Write the failing test**

`internal/detect/detector_test.go`:
```go
package detect

import "testing"

type fake struct{ name string }

func (f fake) Name() string          { return f.name }
func (f fake) Detect(string) bool    { return true }
func (f fake) Targets() []Target     { return nil }

func TestRegistry(t *testing.T) {
	registry = nil
	Register(fake{"a"})
	Register(fake{"b"})
	if len(Detectors()) != 2 {
		t.Fatalf("want 2 detectors, got %d", len(Detectors()))
	}
}

func TestGlobalRegistry(t *testing.T) {
	globals = nil
	RegisterGlobal(Target{Name: "gradle-global", Scope: Global})
	if len(Globals()) != 1 || Globals()[0].Scope != Global {
		t.Fatalf("global target not registered: %+v", Globals())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/detect/ -v`
Expected: FAIL (types/functions undefined).

- [ ] **Step 3: Write minimal implementation**

`internal/detect/detector.go`:
```go
package detect

import "github.com/latif-essam/app-dev-clean/internal/platform"

type Scope int

const (
	Local Scope = iota
	Global
)

type Context struct {
	ProjectRoot string
	Paths       platform.Paths
	DryRun      bool
	Yes         bool
	Force       bool // nuclear: run reinstall without prompting
}

type Target struct {
	Name  string
	Label string
	Desc  string
	Scope Scope
	Paths func(ctx Context) []string
	Run   func(ctx Context) (freed int64, err error)
}

type Detector interface {
	Name() string
	Detect(dir string) bool
	Targets() []Target
}

// PostRunner is optionally implemented by detectors that offer post-clean
// actions (e.g. RN reinstall prompts). cli type-asserts for it.
type PostRunner interface {
	PostRun(ctx Context, ran []string) error
}

var registry []Detector

func Register(d Detector)      { registry = append(registry, d) }
func Detectors() []Detector    { return registry }

var globals []Target

func RegisterGlobal(t Target)  { globals = append(globals, t) }
func Globals() []Target        { return globals }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/detect/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/detect/detector.go internal/detect/detector_test.go
git commit -m "feat: detector interface, target/context types, registries

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 4: `clean` — size / remove / exec engine

**Files:**
- Create: `internal/clean/clean.go`
- Test: `internal/clean/clean_test.go`

**Interfaces:**
- Produces:
  - `func Size(path string) int64` — total bytes under path; 0 if absent.
  - `func Remove(dryRun bool, paths ...string) (freed int64)` — deletes each existing path unless dryRun; tolerates absent/errored paths; logs each; returns bytes freed (estimated by pre-size).
  - `func Exec(dryRun bool, dir, name string, args ...string)` — runs a command in dir unless dryRun; no-ops with a warning if `name` not on PATH; never aborts on failure.
  - `func Human(b int64) string` — human-readable size.

- [ ] **Step 1: Write the failing test**

`internal/clean/clean_test.go`:
```go
package clean

import (
	"os"
	"path/filepath"
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
	freed := Remove(false, nm)
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
	freed := Remove(true, nm)
	if _, err := os.Stat(nm); err != nil {
		t.Fatalf("dry-run must NOT delete: %v", err)
	}
	if freed < 2048 {
		t.Fatalf("dry-run should still estimate freed bytes, got %d", freed)
	}
}

func TestRemoveAbsentTolerated(t *testing.T) {
	if freed := Remove(false, filepath.Join(t.TempDir(), "nope")); freed != 0 {
		t.Fatalf("absent path must free 0 and not panic, got %d", freed)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/clean/ -v`
Expected: FAIL (undefined `Size`/`Remove`).

- [ ] **Step 3: Write minimal implementation**

`internal/clean/clean.go`:
```go
package clean

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Size(path string) int64 {
	var total int64
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, e := d.Info(); e == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func Remove(dryRun bool, paths ...string) (freed int64) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			continue // absent -> skip silently
		}
		sz := Size(p)
		if dryRun {
			fmt.Printf("  [dry-run] would remove %s (%s)\n", p, Human(sz))
			freed += sz
			continue
		}
		if err := os.RemoveAll(p); err != nil {
			fmt.Fprintf(os.Stderr, "  ! could not remove %s: %v\n", p, err)
			continue
		}
		fmt.Printf("  removed %s (%s)\n", p, Human(sz))
		freed += sz
	}
	return freed
}

func Exec(dryRun bool, dir, name string, args ...string) {
	if dryRun {
		fmt.Printf("  [dry-run] would run: %s %s\n", name, strings.Join(args, " "))
		return
	}
	// A name containing a path separator is an explicit executable path
	// (e.g. an absolute gradlew path) — stat it directly. Otherwise resolve
	// via PATH. Never pass a bare relative path like "./gradlew" to
	// exec.Command: its resolution is NOT relative to cmd.Dir and varies by
	// OS/Go version — callers must pass an absolute path instead.
	if strings.ContainsAny(name, `/\`) {
		if _, err := os.Stat(name); err != nil {
			fmt.Printf("  ! %s not found; skipping\n", name)
			return
		}
	} else if _, err := exec.LookPath(name); err != nil {
		fmt.Printf("  ! %s not found; skipping\n", name)
		return
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		fmt.Printf("  ! %s failed (continuing): %v\n", name, err)
	}
}

func Human(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/clean/ -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/clean/
git commit -m "feat: clean engine (size, remove, exec) with dry-run

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 5: `detect` — walk-up root resolution

**Files:**
- Create: `internal/detect/resolve.go`
- Test: `internal/detect/resolve_test.go`

**Interfaces:**
- Consumes: `Detectors()` (Task 3).
- Produces:
  - `type Result struct { Root string; Matched []Detector }`.
  - `func Resolve(start string) (*Result, error)` — walks up from `start`; at the first dir where ≥1 registered detector matches, returns that dir + the full matched set. Returns wrapped `os.ErrNotExist` if none found up-tree.

- [ ] **Step 1: Write the failing test**

`internal/detect/resolve_test.go`:
```go
package detect

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type markerDet struct {
	name   string
	marker string // filename that identifies a root of this type
}

func (m markerDet) Name() string { return m.name }
func (m markerDet) Detect(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, m.marker))
	return err == nil
}
func (m markerDet) Targets() []Target { return nil }

func TestResolveFindsRootFromNested(t *testing.T) {
	registry = []Detector{markerDet{"rn", "package.json"}}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "src", "deep", "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Resolve(nested)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if res.Root != root {
		t.Fatalf("want root %q, got %q", root, res.Root)
	}
	if len(res.Matched) != 1 || res.Matched[0].Name() != "rn" {
		t.Fatalf("want matched [rn], got %+v", res.Matched)
	}
}

func TestResolveReturnsUnion(t *testing.T) {
	registry = []Detector{markerDet{"rn", "package.json"}, markerDet{"android", "settings.gradle"}}
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "package.json"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(root, "settings.gradle"), []byte(""), 0o644)
	res, err := Resolve(root)
	if err != nil || len(res.Matched) != 2 {
		t.Fatalf("want 2 matched, got %+v err=%v", res, err)
	}
}

func TestResolveRefusesOutsideProject(t *testing.T) {
	registry = []Detector{markerDet{"rn", "package.json"}}
	_, err := Resolve(t.TempDir())
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want ErrNotExist, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/detect/ -run TestResolve -v`
Expected: FAIL (`Resolve` undefined).

- [ ] **Step 3: Write minimal implementation**

`internal/detect/resolve.go`:
```go
package detect

import (
	"fmt"
	"os"
	"path/filepath"
)

type Result struct {
	Root    string
	Matched []Detector
}

func Resolve(start string) (*Result, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	for {
		var matched []Detector
		for _, d := range Detectors() {
			if d.Detect(dir) {
				matched = append(matched, d)
			}
		}
		if len(matched) > 0 {
			return &Result{Root: dir, Matched: matched}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir { // reached filesystem root
			return nil, fmt.Errorf("no project root up-tree from %s: %w", start, os.ErrNotExist)
		}
		dir = parent
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/detect/ -v`
Expected: PASS (all detect tests).

- [ ] **Step 5: Commit**

```bash
git add internal/detect/resolve.go internal/detect/resolve_test.go
git commit -m "feat: walk-up root resolution returning matched detector set

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 6: `detectors` shared helpers + global cache targets

**Files:**
- Create: `internal/detectors/shared.go`
- Test: `internal/detectors/shared_test.go`

**Interfaces:**
- Consumes: `detect` (Task 3), `clean` (Task 4), `platform.Paths` via `detect.Context`.
- Produces (all in package `detectors`):
  - `func exists(path string) bool`.
  - `func isWindows() bool`.
  - `func globalTarget(name, label, desc string, pick func(detect.Context) string) detect.Target` — builds a `Global`-scope target whose single path comes from `pick` (empty string ⇒ no path, target self-skips).
  - `init()` registers four globals via `detect.RegisterGlobal`: `gradle-global`, `xcode-dd`, `pods-cache`, `pub-cache`.

- [ ] **Step 1: Write the failing test**

`internal/detectors/shared_test.go`:
```go
package detectors

import (
	"testing"

	"github.com/latif-essam/app-dev-clean/internal/detect"
	"github.com/latif-essam/app-dev-clean/internal/platform"
)

func globalByName(t *testing.T, name string) detect.Target {
	t.Helper()
	for _, g := range detect.Globals() {
		if g.Name == name {
			return g
		}
	}
	t.Fatalf("global target %q not registered", name)
	return detect.Target{}
}

func TestGlobalsRegistered(t *testing.T) {
	for _, n := range []string{"gradle-global", "xcode-dd", "pods-cache", "pub-cache"} {
		g := globalByName(t, n)
		if g.Scope != detect.Global {
			t.Fatalf("%s must be Global scope", n)
		}
	}
}

func TestGradleGlobalUsesPlatformPath(t *testing.T) {
	g := globalByName(t, "gradle-global")
	ctx := detect.Context{Paths: platform.Paths{GradleCache: "/home/x/.gradle/caches"}}
	got := g.Paths(ctx)
	if len(got) != 1 || got[0] != "/home/x/.gradle/caches" {
		t.Fatalf("gradle-global path wrong: %+v", got)
	}
}

func TestXcodeGlobalEmptyOnNonMac(t *testing.T) {
	g := globalByName(t, "xcode-dd")
	// XcodeDD "" (as on win/linux) -> no paths offered
	if got := g.Paths(detect.Context{Paths: platform.Paths{XcodeDD: ""}}); len(got) != 0 {
		t.Fatalf("xcode-dd must offer no path when XcodeDD empty, got %+v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/detectors/ -v`
Expected: FAIL (undefined helpers / no globals registered).

- [ ] **Step 3: Write minimal implementation**

`internal/detectors/shared.go`:
```go
package detectors

import (
	"os"
	"runtime"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isWindows() bool { return runtime.GOOS == "windows" }

// globalTarget builds a Global-scope target cleaning a single machine-wide
// cache path chosen by pick. If pick returns "", the target offers no path
// (e.g. macOS-only caches on Windows/Linux) and cleaning is a no-op.
func globalTarget(name, label, desc string, pick func(detect.Context) string) detect.Target {
	pathsFn := func(ctx detect.Context) []string {
		if p := pick(ctx); p != "" {
			return []string{p}
		}
		return nil
	}
	return detect.Target{
		Name:  name,
		Label: label,
		Desc:  desc,
		Scope: detect.Global,
		Paths: pathsFn,
		Run: func(ctx detect.Context) (int64, error) {
			return clean.Remove(ctx.DryRun, pathsFn(ctx)...), nil
		},
	}
}

func init() {
	detect.RegisterGlobal(globalTarget("gradle-global", "gradle cache", "~/.gradle/caches (all projects)",
		func(c detect.Context) string { return c.Paths.GradleCache }))
	detect.RegisterGlobal(globalTarget("xcode-dd", "xcode dd", "Xcode DerivedData (macOS)",
		func(c detect.Context) string { return c.Paths.XcodeDD }))
	detect.RegisterGlobal(globalTarget("pods-cache", "pods cache", "CocoaPods cache (macOS)",
		func(c detect.Context) string { return c.Paths.CocoaPods }))
	detect.RegisterGlobal(globalTarget("pub-cache", "pub cache", "Flutter pub cache",
		func(c detect.Context) string { return c.Paths.PubCache }))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/detectors/ -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/detectors/shared.go internal/detectors/shared_test.go
git commit -m "feat: shared detector helpers + global cache targets

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 7: `android` detector

**Files:**
- Create: `internal/detectors/android.go`
- Test: `internal/detectors/android_test.go`

**Interfaces:**
- Produces:
  - `func androidLocalPaths(root string) []string` (package-level, reused by rn/expo/flutter).
  - `func androidLocal(ctx detect.Context) (int64, error)` (runs `gradlew clean` unless dry-run, then removes local paths).
  - `type android struct{}` implementing `detect.Detector`; registered via `init()`.

- [ ] **Step 1: Write the failing test**

`internal/detectors/android_test.go`:
```go
package detectors

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAndroidDetect(t *testing.T) {
	dir := t.TempDir()
	if (android{}).Detect(dir) {
		t.Fatal("empty dir must not detect as android")
	}
	os.WriteFile(filepath.Join(dir, "settings.gradle"), []byte(""), 0o644)
	if !(android{}).Detect(dir) {
		t.Fatal("settings.gradle must detect as android")
	}
}

func TestAndroidLocalPaths(t *testing.T) {
	got := androidLocalPaths("/proj")
	want := filepath.Join("/proj", ".gradle")
	found := false
	for _, p := range got {
		if p == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("android paths must include %q, got %+v", want, got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/detectors/ -run TestAndroid -v`
Expected: FAIL (`android` undefined).

- [ ] **Step 3: Write minimal implementation**

`internal/detectors/android.go`:
```go
package detectors

import (
	"path/filepath"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

type android struct{}

func (android) Name() string { return "android" }

func (android) Detect(dir string) bool {
	for _, m := range []string{"settings.gradle", "settings.gradle.kts", "build.gradle", "build.gradle.kts"} {
		if exists(filepath.Join(dir, m)) {
			return true
		}
	}
	return false
}

func androidLocalPaths(root string) []string {
	return []string{
		filepath.Join(root, "build"),
		filepath.Join(root, "app", "build"),
		filepath.Join(root, ".gradle"),
		filepath.Join(root, ".cxx"),
		filepath.Join(root, "app", ".cxx"),
	}
}

func androidLocal(ctx detect.Context) (int64, error) {
	wrapperName := "gradlew"
	if isWindows() {
		wrapperName = "gradlew.bat"
	}
	wrapperPath := filepath.Join(ctx.ProjectRoot, wrapperName)
	if exists(wrapperPath) {
		// Pass the ABSOLUTE wrapper path: bare "./gradlew" does not resolve
		// relative to cmd.Dir in os/exec (see clean.Exec).
		clean.Exec(ctx.DryRun, ctx.ProjectRoot, wrapperPath, "clean")
	}
	return clean.Remove(ctx.DryRun, androidLocalPaths(ctx.ProjectRoot)...), nil
}

func (android) Targets() []detect.Target {
	return []detect.Target{{
		Name:  "android",
		Label: "android",
		Desc:  "build/, app/build, .gradle, .cxx + gradlew clean",
		Scope: detect.Local,
		Paths: func(ctx detect.Context) []string { return androidLocalPaths(ctx.ProjectRoot) },
		Run:   androidLocal,
	}}
}

func init() { detect.Register(android{}) }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/detectors/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/detectors/android.go internal/detectors/android_test.go
git commit -m "feat: android-native detector

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 8: `ios` detector (macOS-gated)

**Files:**
- Create: `internal/detectors/ios.go`
- Test: `internal/detectors/ios_test.go`

**Interfaces:**
- Produces:
  - `func iosLocalPaths(root string) []string`.
  - `func iosLocal(ctx detect.Context) (int64, error)`.
  - `type ios struct{}` implementing `detect.Detector`; matches `.xcodeproj`/`.xcworkspace`/`Package.swift` via glob; registered via `init()`.

- [ ] **Step 1: Write the failing test**

`internal/detectors/ios_test.go`:
```go
package detectors

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIOSDetect(t *testing.T) {
	dir := t.TempDir()
	if (ios{}).Detect(dir) {
		t.Fatal("empty dir must not detect as ios")
	}
	os.MkdirAll(filepath.Join(dir, "App.xcodeproj"), 0o755)
	if !(ios{}).Detect(dir) {
		t.Fatal(".xcodeproj must detect as ios")
	}
}

func TestIOSLocalPaths(t *testing.T) {
	got := iosLocalPaths("/proj")
	want := filepath.Join("/proj", "Pods")
	found := false
	for _, p := range got {
		if p == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("ios paths must include %q, got %+v", want, got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/detectors/ -run TestIOS -v`
Expected: FAIL (`ios` undefined).

- [ ] **Step 3: Write minimal implementation**

`internal/detectors/ios.go`:
```go
package detectors

import (
	"path/filepath"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

type ios struct{}

func (ios) Name() string { return "ios" }

func (ios) Detect(dir string) bool {
	if exists(filepath.Join(dir, "Package.swift")) {
		return true
	}
	for _, pat := range []string{"*.xcodeproj", "*.xcworkspace"} {
		if m, _ := filepath.Glob(filepath.Join(dir, pat)); len(m) > 0 {
			return true
		}
	}
	return false
}

func iosLocalPaths(root string) []string {
	return []string{
		filepath.Join(root, "build"),
		filepath.Join(root, "Pods"),
		filepath.Join(root, "Podfile.lock"),
		filepath.Join(root, ".build"), // SwiftPM
	}
}

func iosLocal(ctx detect.Context) (int64, error) {
	return clean.Remove(ctx.DryRun, iosLocalPaths(ctx.ProjectRoot)...), nil
}

func (ios) Targets() []detect.Target {
	return []detect.Target{{
		Name:  "ios",
		Label: "ios",
		Desc:  "build/, Pods, Podfile.lock, SwiftPM .build",
		Scope: detect.Local,
		Paths: func(ctx detect.Context) []string { return iosLocalPaths(ctx.ProjectRoot) },
		Run:   iosLocal,
	}}
}

func init() { detect.Register(ios{}) }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/detectors/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/detectors/ios.go internal/detectors/ios_test.go
git commit -m "feat: ios-native detector (macOS-gated globals)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 9: `rn` detector (+ js/metro/watchman + reinstall PostRunner)

**Files:**
- Create: `internal/detectors/rn.go`
- Test: `internal/detectors/rn_test.go`

**Interfaces:**
- Consumes: `androidLocal`/`iosLocal` (Tasks 7–8), `clean`, `platform.Paths.TmpDir`.
- Produces:
  - `func pkgJSONHas(dir, needle string) bool` (package-level, reused by expo).
  - `func containsStr(sl []string, s string) bool` (package-level, reused by expo/flutter tests if needed).
  - `func rnJS(ctx) (int64, error)`, `func rnMetro(ctx) (int64, error)`, `func rnWatchman(ctx) (int64, error)`.
  - `type rn struct{}` implementing `detect.Detector` + `detect.PostRunner`; PostRun honors `ctx.DryRun`/`ctx.Force`/`ctx.Yes`; registered via `init()`.

- [ ] **Step 1: Write the failing test**

`internal/detectors/rn_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/detectors/ -run TestRN -v`
Expected: FAIL (`rn` undefined).

- [ ] **Step 3: Write minimal implementation**

`internal/detectors/rn.go`:
```go
package detectors

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

// pkgJSONHas reports whether dir/package.json contains needle (substring match,
// same coarse check the bash version used via grep).
func pkgJSONHas(dir, needle string) bool {
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return false
	}
	return strings.Contains(string(b), needle)
}

type rn struct{}

func (rn) Name() string { return "rn" }

func (rn) Detect(dir string) bool {
	if !pkgJSONHas(dir, `"react-native"`) {
		return false
	}
	return exists(filepath.Join(dir, "android")) || exists(filepath.Join(dir, "ios"))
}

func rnJS(ctx detect.Context) (int64, error) {
	return clean.Remove(ctx.DryRun,
		filepath.Join(ctx.ProjectRoot, "node_modules"),
		filepath.Join(ctx.ProjectRoot, "package-lock.json"),
	), nil
}

func rnMetro(ctx detect.Context) (int64, error) {
	// Metro/Haste temp caches live in the OS temp dir as metro-* / haste-map-*.
	var freed int64
	patterns := []string{"metro-*", "haste-map-*", "metro-cache"}
	for _, pat := range patterns {
		matches, _ := filepath.Glob(filepath.Join(ctx.Paths.TmpDir, pat))
		freed += clean.Remove(ctx.DryRun, matches...)
	}
	return freed, nil
}

func rnWatchman(ctx detect.Context) (int64, error) {
	// Reset stale watch; non-destructive, skipped in dry-run.
	if !ctx.DryRun {
		clean.Exec(false, ctx.ProjectRoot, "watchman", "watch-del", ctx.ProjectRoot)
		clean.Exec(false, ctx.ProjectRoot, "watchman", "watch-project", ctx.ProjectRoot)
	}
	return 0, nil
}

func (rn) Targets() []detect.Target {
	return []detect.Target{
		{Name: "android", Label: "android", Desc: "android build + gradle caches", Scope: detect.Local,
			Paths: func(c detect.Context) []string { return androidLocalPaths(c.ProjectRoot) }, Run: androidLocal},
		{Name: "ios", Label: "ios", Desc: "ios build, Pods, Podfile.lock", Scope: detect.Local,
			Paths: func(c detect.Context) []string { return iosLocalPaths(c.ProjectRoot) }, Run: iosLocal},
		{Name: "js", Label: "js", Desc: "node_modules + package-lock.json", Scope: detect.Local,
			Paths: func(c detect.Context) []string {
				return []string{filepath.Join(c.ProjectRoot, "node_modules"), filepath.Join(c.ProjectRoot, "package-lock.json")}
			}, Run: rnJS},
		{Name: "metro", Label: "metro", Desc: "Metro/Haste temp caches", Scope: detect.Local,
			Paths: func(c detect.Context) []string { return nil }, Run: rnMetro},
		{Name: "watchman", Label: "watchman", Desc: "reset stale watch", Scope: detect.Local,
			Paths: func(c detect.Context) []string { return nil }, Run: rnWatchman},
	}
}

// PostRun handles reinstall after a js/ios clean:
//   - dry-run: never reinstall
//   - Force (nuclear): reinstall unconditionally, no prompt
//   - Yes (non-interactive, non-nuclear): skip reinstall
//   - otherwise (interactive): prompt per action
func (rn) PostRun(ctx detect.Context, ran []string) error {
	if ctx.DryRun {
		return nil
	}
	doJS := containsStr(ran, "js")
	doIOS := containsStr(ran, "ios")
	if ctx.Force {
		if doJS {
			clean.Exec(false, ctx.ProjectRoot, "npm", "install")
		}
		if doIOS {
			clean.Exec(false, filepath.Join(ctx.ProjectRoot, "ios"), "pod", "install", "--repo-update")
		}
		return nil
	}
	if ctx.Yes {
		return nil
	}
	if doJS && promptYes("  Run 'npm install' now? [y/N] ") {
		clean.Exec(false, ctx.ProjectRoot, "npm", "install")
	}
	if doIOS && promptYes("  Run 'pod install' now? [y/N] ") {
		clean.Exec(false, filepath.Join(ctx.ProjectRoot, "ios"), "pod", "install", "--repo-update")
	}
	return nil
}

func promptYes(msg string) bool {
	fmt.Print(msg)
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

func containsStr(sl []string, s string) bool {
	for _, x := range sl {
		if x == s {
			return true
		}
	}
	return false
}

func init() { detect.Register(rn{}) }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/detectors/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/detectors/rn.go internal/detectors/rn_test.go
git commit -m "feat: react-native detector (js/metro/watchman + reinstall)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 10: `expo` detector

**Files:**
- Create: `internal/detectors/expo.go`
- Test: `internal/detectors/expo_test.go`

**Interfaces:**
- Consumes: `pkgJSONHas`, `rnJS`/`rnMetro`/`androidLocal`/`iosLocal`.
- Produces: `type expo struct{ rn }` (embeds `rn` to inherit its `PostRun` reinstall) implementing `detect.Detector` + `detect.PostRunner`; adds `.expo/` cleanup on top of RN targets; registered via `init()`.

- [ ] **Step 1: Write the failing test**

`internal/detectors/expo_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/detectors/ -run TestExpo -v`
Expected: FAIL (`expo` undefined).

- [ ] **Step 3: Write minimal implementation**

`internal/detectors/expo.go`:
```go
package detectors

import (
	"path/filepath"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

// expo embeds rn so it inherits rn's PostRun (npm/pod reinstall) behavior.
// Name/Detect/Targets below shadow the embedded rn's methods.
type expo struct{ rn }

func (expo) Name() string { return "expo" }

func (expo) Detect(dir string) bool { return pkgJSONHas(dir, `"expo"`) }

func expoDirPaths(root string) []string {
	return []string{filepath.Join(root, ".expo"), filepath.Join(root, ".expo-shared")}
}

func (expo) Targets() []detect.Target {
	t := []detect.Target{
		{Name: "expo", Label: "expo", Desc: ".expo/ + prebuild caches", Scope: detect.Local,
			Paths: func(c detect.Context) []string { return expoDirPaths(c.ProjectRoot) },
			Run: func(c detect.Context) (int64, error) {
				return clean.Remove(c.DryRun, expoDirPaths(c.ProjectRoot)...), nil
			}},
		{Name: "js", Label: "js", Desc: "node_modules + package-lock.json", Scope: detect.Local,
			Paths: func(c detect.Context) []string {
				return []string{filepath.Join(c.ProjectRoot, "node_modules"), filepath.Join(c.ProjectRoot, "package-lock.json")}
			}, Run: rnJS},
		{Name: "metro", Label: "metro", Desc: "Metro/Haste temp caches", Scope: detect.Local,
			Paths: func(c detect.Context) []string { return nil }, Run: rnMetro},
	}
	// Bare Expo also has native dirs -> include android/ios local cleanups.
	t = append(t,
		detect.Target{Name: "android", Label: "android", Desc: "android build + gradle caches", Scope: detect.Local,
			Paths: func(c detect.Context) []string { return androidLocalPaths(c.ProjectRoot) }, Run: androidLocal},
		detect.Target{Name: "ios", Label: "ios", Desc: "ios build, Pods, Podfile.lock", Scope: detect.Local,
			Paths: func(c detect.Context) []string { return iosLocalPaths(c.ProjectRoot) }, Run: iosLocal},
	)
	return t
}

func init() { detect.Register(expo{}) }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/detectors/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/detectors/expo.go internal/detectors/expo_test.go
git commit -m "feat: expo detector (managed + bare)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 11: `flutter` detector

**Files:**
- Create: `internal/detectors/flutter.go`
- Test: `internal/detectors/flutter_test.go`

**Interfaces:**
- Produces: `type flutter struct{}` implementing `detect.Detector`; targets `build/`, `.dart_tool/`, `flutter clean`; registered via `init()`.

- [ ] **Step 1: Write the failing test**

`internal/detectors/flutter_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/detectors/ -run TestFlutter -v`
Expected: FAIL (`flutter` undefined).

- [ ] **Step 3: Write minimal implementation**

`internal/detectors/flutter.go`:
```go
package detectors

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

type flutter struct{}

func (flutter) Name() string { return "flutter" }

func (flutter) Detect(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, "pubspec.yaml"))
	if err != nil {
		return false
	}
	return strings.Contains(string(b), "flutter")
}

func flutterLocalPaths(root string) []string {
	return []string{filepath.Join(root, "build"), filepath.Join(root, ".dart_tool")}
}

func (flutter) Targets() []detect.Target {
	return []detect.Target{{
		Name:  "flutter",
		Label: "flutter",
		Desc:  "build/, .dart_tool/ + flutter clean",
		Scope: detect.Local,
		Paths: func(c detect.Context) []string { return flutterLocalPaths(c.ProjectRoot) },
		Run: func(c detect.Context) (int64, error) {
			clean.Exec(c.DryRun, c.ProjectRoot, "flutter", "clean")
			return clean.Remove(c.DryRun, flutterLocalPaths(c.ProjectRoot)...), nil
		},
	}}
}

func init() { detect.Register(flutter{}) }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/detectors/ -v`
Expected: PASS (all detector tests).

- [ ] **Step 5: Commit**

```bash
git add internal/detectors/flutter.go internal/detectors/flutter_test.go
git commit -m "feat: flutter detector

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 12: `cli` — flag parsing, dispatch, gates, combos

**Files:**
- Modify: `internal/cli/cli.go` (replace stub)
- Create: `internal/cli/parse.go`
- Modify: `main.go` (blank-import detectors so they register)
- Test: `internal/cli/parse_test.go`

**Interfaces:**
- Consumes: `detect.Resolve`, `detect.Detectors`, `detect.Globals`, `detect.Context`, `platform.Detect`, `detect.PostRunner`.
- Produces:
  - `type Options struct { Targets []string; TypeFilter string; DryRun, Yes, ShowRoot, Help, Version bool }`.
  - `func parse(args []string) (Options, error)`.
  - `func Run(args []string, version string) int` (full behavior; sets `ctx.Force` when `nuclear` is requested, then always invokes matched `PostRunner`s).
  - `func isGlobalName(name string) bool`, `func containsStr(sl []string, s string) bool`.

- [ ] **Step 1: Write the failing test**

`internal/cli/parse_test.go`:
```go
package cli

import "testing"

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -v`
Expected: FAIL (`parse`/`Options` undefined).

- [ ] **Step 3: Write minimal implementation**

`internal/cli/parse.go`:
```go
package cli

import (
	"fmt"

	"github.com/latif-essam/app-dev-clean/internal/detect"
)

type Options struct {
	Targets    []string
	TypeFilter string
	DryRun     bool
	Yes        bool
	ShowRoot   bool
	Help       bool
	Version    bool
}

func parse(args []string) (Options, error) {
	var o Options
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--help", "-h":
			o.Help = true
		case "--version", "-v":
			o.Version = true
		case "--root":
			o.ShowRoot = true
		case "--dry-run":
			o.DryRun = true
		case "--yes", "-y":
			o.Yes = true
		case "--type":
			if i+1 >= len(args) {
				return o, fmt.Errorf("--type needs a value")
			}
			i++
			o.TypeFilter = args[i]
		default:
			o.Targets = append(o.Targets, a)
		}
	}
	return o, nil
}

func isGlobalName(name string) bool {
	for _, g := range detect.Globals() {
		if g.Name == name {
			return true
		}
	}
	return name == "nuclear"
}
```

`internal/cli/cli.go` (full replacement):
```go
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
	"github.com/latif-essam/app-dev-clean/internal/platform"
	"github.com/latif-essam/app-dev-clean/internal/ui"
)

const usage = `app-dev-clean - cross-platform dev-cache cleaner

  app-dev-clean                interactive menu (inside a known project)
  app-dev-clean <target>...    run named targets (e.g. ios js metro)
  app-dev-clean local-all      all local targets for detected type(s)
  app-dev-clean nuclear        local-all + global caches + reinstall (confirmed)
  app-dev-clean --type <t>     scope to one detector (rn|android|ios|flutter|expo)
  app-dev-clean --dry-run      show what would be freed; delete nothing
  app-dev-clean -y, --yes      skip confirmation prompts
  app-dev-clean --root         print resolved root + detected type(s)
  app-dev-clean --version      print version
  app-dev-clean --help         this help
`

func Run(args []string, version string) int {
	o, err := parse(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}
	switch {
	case o.Help:
		fmt.Print(usage)
		return 0
	case o.Version:
		fmt.Println("app-dev-clean", version)
		return 0
	}

	paths := platform.Detect()
	ctx := detect.Context{Paths: paths, DryRun: o.DryRun, Yes: o.Yes}

	// global-only invocation is allowed without a project.
	onlyGlobals := len(o.Targets) > 0
	for _, t := range o.Targets {
		if !isGlobalName(t) {
			onlyGlobals = false
		}
	}

	var res *detect.Result
	if !onlyGlobals || len(o.Targets) == 0 {
		res, err = detect.Resolve(mustCwd())
		if err != nil {
			fmt.Fprintln(os.Stderr, "✗ not inside a recognized project.")
			fmt.Fprintln(os.Stderr, "  refusing local cleanup so nothing is deleted in the wrong place.")
			fmt.Fprintln(os.Stderr, "  cd into a project, or run a global target (e.g. gradle-global).")
			return 1
		}
		ctx.ProjectRoot = res.Root
		fmt.Printf("==> project: %s (%s)\n", res.Root, typeNames(res, o.TypeFilter))
	}

	if o.ShowRoot {
		if res == nil {
			fmt.Fprintln(os.Stderr, "✗ not a project")
			return 1
		}
		fmt.Println(res.Root)
		fmt.Println("types:", typeNames(res, ""))
		return 0
	}

	targets := collectTargets(res, o.TypeFilter)

	// Gather raw selections (menu rows OR CLI args, either may include combos).
	var raw []string
	if len(o.Targets) == 0 {
		rows := ui.Rows(targets, detect.Globals(), ctx)
		raw = ui.Run(rows)
		if len(raw) == 0 {
			fmt.Println("nothing selected")
			return 0
		}
	} else {
		raw = o.Targets
	}

	// Detect nuclear BEFORE expansion (expandCombos replaces the token); nuclear
	// forces unconditional reinstall. Then expand once for both paths.
	nuclear := containsStr(raw, "nuclear")
	ctx.Force = nuclear
	selected := expandCombos(raw, targets)

	if !o.Yes && needsConfirm(selected) {
		fmt.Printf("About to clean shared/global caches: %s\n", strings.Join(selected, " "))
		if !promptYes("  These affect ALL projects. Proceed? [y/N] ") {
			fmt.Println("aborted")
			return 0
		}
	}

	freed := runTargets(ctx, res, selected, targets)
	fmt.Printf("\nDone. Reclaimed ~%s\n", clean.Human(freed))
	return 0
}
```

(Helper functions `mustCwd`, `typeNames`, `collectTargets`, `expandCombos`, `needsConfirm`, `runTargets`, `promptYes` follow in Step 3b.)

- [ ] **Step 3b: Add cli helpers**

Append to `internal/cli/cli.go`:
```go
func mustCwd() string {
	d, err := os.Getwd()
	if err != nil {
		return "."
	}
	return d
}

func typeNames(res *detect.Result, filter string) string {
	if res == nil {
		return ""
	}
	var names []string
	for _, d := range res.Matched {
		if filter == "" || d.Name() == filter {
			names = append(names, d.Name())
		}
	}
	return strings.Join(names, "+")
}

// collectTargets returns the union of local targets from matched detectors
// (deduped by name), honoring an optional --type filter.
func collectTargets(res *detect.Result, filter string) []detect.Target {
	var out []detect.Target
	seen := map[string]bool{}
	if res != nil {
		for _, d := range res.Matched {
			if filter != "" && d.Name() != filter {
				continue
			}
			for _, tg := range d.Targets() {
				if !seen[tg.Name] {
					seen[tg.Name] = true
					out = append(out, tg)
				}
			}
		}
	}
	return out
}

func targetByName(name string, local []detect.Target) (detect.Target, bool) {
	for _, tg := range local {
		if tg.Name == name {
			return tg, true
		}
	}
	for _, g := range detect.Globals() {
		if g.Name == name {
			return g, true
		}
	}
	return detect.Target{}, false
}

func expandCombos(requested []string, local []detect.Target) []string {
	var out []string
	for _, r := range requested {
		switch r {
		case "local-all":
			for _, tg := range local {
				out = append(out, tg.Name)
			}
		case "nuclear":
			for _, tg := range local {
				out = append(out, tg.Name)
			}
			for _, g := range detect.Globals() {
				out = append(out, g.Name)
			}
		default:
			out = append(out, r)
		}
	}
	return out
}

func needsConfirm(selected []string) bool {
	for _, s := range selected {
		if isGlobalName(s) {
			return true
		}
	}
	return false
}

// runTargets executes each selected target (already combo-expanded), summing
// reclaimed bytes, then runs each matched detector's PostRun hook. PostRun
// itself decides reinstall behavior from ctx (DryRun/Force/Yes). Unknown names
// warn and are skipped.
func runTargets(ctx detect.Context, res *detect.Result, selected []string, local []detect.Target) int64 {
	var freed int64
	for _, name := range selected {
		tg, ok := targetByName(name, local)
		if !ok {
			fmt.Printf("  ! unknown target: %s\n", name)
			continue
		}
		f, err := tg.Run(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ! %s: %v\n", name, err)
		}
		freed += f
	}
	if res != nil {
		for _, d := range res.Matched {
			if pr, ok := d.(detect.PostRunner); ok {
				_ = pr.PostRun(ctx, selected)
			}
		}
	}
	return freed
}

func promptYes(msg string) bool {
	fmt.Print(msg)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

func containsStr(sl []string, s string) bool {
	for _, x := range sl {
		if x == s {
			return true
		}
	}
	return false
}
```

`main.go` (add blank import):
```go
package main

import (
	"os"

	"github.com/latif-essam/app-dev-clean/internal/cli"
	_ "github.com/latif-essam/app-dev-clean/internal/detectors" // register detectors
)

var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], version))
}
```

> NOTE: `internal/ui` (Task 13) must exist for this to compile. If executing tasks strictly in order, temporarily stub `ui.Rows`/`ui.Run` (return `nil`) in Task 12 and replace in Task 13, OR do Task 13 before wiring the menu path. Recommended: create the `ui` stub file in this task's Step 3c.

- [ ] **Step 3c: Stub `internal/ui` so cli compiles**

`internal/ui/menu.go` (temporary stub, fully implemented in Task 13):
```go
package ui

import "github.com/latif-essam/app-dev-clean/internal/detect"

type Row struct {
	Target string
	Label  string
	Desc   string
	Header string // non-empty => section header row
}

func Rows(local []detect.Target, globals []detect.Target, ctx detect.Context) []Row { return nil }
func Run(rows []Row) []string                                                        { return nil }
```

- [ ] **Step 4: Run tests + build**

Run:
```bash
go test ./internal/cli/ -v
go build ./...
```
Expected: cli parse tests PASS; whole module builds.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/ internal/ui/menu.go main.go
git commit -m "feat: cli dispatch, flags, confirm gates, combos, dry-run

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 13: `ui` — bubbletea interactive checklist

**Files:**
- Modify: `internal/ui/menu.go` (replace stub)
- Test: `internal/ui/menu_test.go`
- Modify: `go.mod`/`go.sum` (add bubbletea + lipgloss)

**Interfaces:**
- Consumes: `detect.Target`, `detect.Context`.
- Produces:
  - `func Rows(local, globals []detect.Target, ctx detect.Context) []Row` — builds grouped rows (LOCAL / GLOBAL / COMBOS headers + selectable target rows). Globals whose `Paths(ctx)` is empty (unavailable on this OS) are omitted.
  - `func Run(rows []Row) []string` — runs the bubbletea checklist; returns selected target names.
  - `type model` with `toggle`/`selected` logic unit-tested without the event loop.

- [ ] **Step 1: Add deps**

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
```

- [ ] **Step 2: Write the failing test**

`internal/ui/menu_test.go`:
```go
package ui

import (
	"testing"

	"github.com/latif-essam/app-dev-clean/internal/detect"
	"github.com/latif-essam/app-dev-clean/internal/platform"
)

func TestRowsOmitsUnavailableGlobals(t *testing.T) {
	local := []detect.Target{{Name: "js", Label: "js", Scope: detect.Local}}
	globals := []detect.Target{
		{Name: "gradle-global", Label: "gradle", Scope: detect.Global,
			Paths: func(c detect.Context) []string { return []string{"/g"} }},
		{Name: "xcode-dd", Label: "xcode", Scope: detect.Global,
			Paths: func(c detect.Context) []string { return nil }}, // unavailable
	}
	rows := Rows(local, globals, detect.Context{Paths: platform.Paths{}})
	for _, r := range rows {
		if r.Target == "xcode-dd" {
			t.Fatal("unavailable global must be omitted")
		}
	}
}

func TestModelToggleSelects(t *testing.T) {
	m := newModel([]Row{
		{Header: "LOCAL"},
		{Target: "js", Label: "js"},
		{Target: "ios", Label: "ios"},
	})
	m.cursor = 1 // on "js"
	m = m.toggle()
	got := m.selectedTargets()
	if len(got) != 1 || got[0] != "js" {
		t.Fatalf("want [js], got %v", got)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/ui/ -v`
Expected: FAIL (`newModel`/`toggle` undefined).

- [ ] **Step 4: Write implementation**

`internal/ui/menu.go` (full replacement):
```go
package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/latif-essam/app-dev-clean/internal/detect"
)

type Row struct {
	Target string
	Label  string
	Desc   string
	Header string // non-empty => section header (not selectable)
}

func Rows(local, globals []detect.Target, ctx detect.Context) []Row {
	var rows []Row
	if len(local) > 0 {
		rows = append(rows, Row{Header: "LOCAL (project — fast to rebuild)"})
		for _, t := range local {
			rows = append(rows, Row{Target: t.Name, Label: t.Label, Desc: t.Desc})
		}
	}
	var avail []detect.Target
	for _, g := range globals {
		if g.Paths != nil && len(g.Paths(ctx)) > 0 {
			avail = append(avail, g)
		}
	}
	if len(avail) > 0 {
		rows = append(rows, Row{Header: "GLOBAL (shared across ALL projects)"})
		for _, g := range avail {
			rows = append(rows, Row{Target: g.Name, Label: g.Label, Desc: g.Desc})
		}
	}
	rows = append(rows, Row{Header: "COMBOS"})
	rows = append(rows, Row{Target: "local-all", Label: "local-all", Desc: "all local targets"})
	rows = append(rows, Row{Target: "nuclear", Label: "nuclear", Desc: "everything + reinstall"})
	return rows
}

type model struct {
	rows    []Row
	cursor  int
	checked map[int]bool
	done    bool
	quit    bool
}

func newModel(rows []Row) model {
	m := model{rows: rows, checked: map[int]bool{}}
	m.cursor = m.firstSelectable(0, 1)
	return m
}

func (m model) firstSelectable(from, dir int) int {
	i := from
	for i >= 0 && i < len(m.rows) {
		if m.rows[i].Header == "" {
			return i
		}
		i += dir
	}
	return from
}

func (m model) toggle() model {
	if m.cursor >= 0 && m.cursor < len(m.rows) && m.rows[m.cursor].Header == "" {
		m.checked[m.cursor] = !m.checked[m.cursor]
	}
	return m
}

func (m model) selectedTargets() []string {
	var out []string
	for i, r := range m.rows {
		if r.Header == "" && m.checked[i] {
			out = append(out, r.Target)
		}
	}
	return out
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "q", "ctrl+c":
		m.quit = true
		return m, tea.Quit
	case "up", "k":
		m.cursor = m.move(-1)
	case "down", "j":
		m.cursor = m.move(1)
	case " ":
		m = m.toggle()
	case "a":
		for i, r := range m.rows {
			if r.Header == "" {
				m.checked[i] = true
			}
		}
	case "n":
		m.checked = map[int]bool{}
	case "enter":
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m model) move(dir int) int {
	i := m.cursor + dir
	for i >= 0 && i < len(m.rows) {
		if m.rows[i].Header == "" {
			return i
		}
		i += dir
	}
	return m.cursor
}

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

func (m model) View() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("  app-dev-clean") + "\n")
	b.WriteString("  up/down move · SPACE toggle · a all · n none · ENTER run · q quit\n\n")
	for i, r := range m.rows {
		if r.Header != "" {
			b.WriteString("\n  " + headerStyle.Render(r.Header) + "\n")
			continue
		}
		mark := " "
		if m.checked[i] {
			mark = "x"
		}
		pointer := "  "
		label := fmt.Sprintf("%-14s %s", r.Label, r.Desc)
		if i == m.cursor {
			pointer = cursorStyle.Render("> ")
			label = cursorStyle.Render(label)
		}
		b.WriteString(fmt.Sprintf("  %s[%s] %s\n", pointer, mark, label))
	}
	return b.String()
}

func Run(rows []Row) []string {
	m := newModel(rows)
	p := tea.NewProgram(m)
	res, err := p.Run()
	if err != nil {
		return nil
	}
	fm := res.(model)
	if fm.quit || !fm.done {
		return nil
	}
	return fm.selectedTargets()
}
```

- [ ] **Step 5: Run test to verify it passes**

Run:
```bash
go test ./internal/ui/ -v
go build ./...
```
Expected: PASS; module builds with bubbletea.

- [ ] **Step 6: Manual smoke (interactive, no auto-test)**

In a real RN project dir:
```bash
go run . 
```
Confirm arrows/space/enter/a/n/q work and selecting `js` + ENTER triggers cleanup. (Skip if no project handy — covered by unit tests + later end-to-end.)

- [ ] **Step 7: Commit**

```bash
git add internal/ui/ go.mod go.sum
git commit -m "feat: bubbletea cross-platform interactive checklist

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 14: End-to-end binary tests (parity + refusal + dry-run)

**Files:**
- Create: `e2e_test.go` (package `main`, build-and-run)

**Interfaces:**
- Consumes: the built binary. Verifies refusal outside a project, `--root`, and `--dry-run` non-deletion end-to-end.

- [ ] **Step 1: Write the failing test**

`e2e_test.go`:
```go
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildBin(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "adc")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

func TestRefusesOutsideProject(t *testing.T) {
	bin := buildBin(t)
	cmd := exec.Command(bin, "js")
	cmd.Dir = t.TempDir() // empty, not a project
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit outside project, got success:\n%s", out)
	}
	if !strings.Contains(string(out), "refusing local cleanup") {
		t.Fatalf("want refusal message, got:\n%s", out)
	}
}

func TestDryRunDoesNotDelete(t *testing.T) {
	bin := buildBin(t)
	proj := t.TempDir()
	os.WriteFile(filepath.Join(proj, "package.json"),
		[]byte(`{"dependencies":{"react-native":"0.74.0"}}`), 0o644)
	os.MkdirAll(filepath.Join(proj, "android"), 0o755)
	nm := filepath.Join(proj, "node_modules")
	os.MkdirAll(nm, 0o755)
	os.WriteFile(filepath.Join(nm, "x"), make([]byte, 1024), 0o644)

	cmd := exec.Command(bin, "js", "--dry-run", "-y")
	cmd.Dir = proj
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dry-run failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(nm); err != nil {
		t.Fatalf("dry-run must NOT delete node_modules: %v", err)
	}
	if !strings.Contains(string(out), "dry-run") {
		t.Fatalf("want dry-run output, got:\n%s", out)
	}
}
```

- [ ] **Step 2: Run to verify it fails/passes**

Run: `go test . -run 'TestRefuses|TestDryRun' -v`
Expected: PASS (implementation from Tasks 1–13 satisfies these). If FAIL, fix the cli refusal message / dry-run wiring until green.

- [ ] **Step 3: Full suite**

Run: `go test ./...`
Expected: ALL PASS.

- [ ] **Step 4: Commit**

```bash
git add e2e_test.go
git commit -m "test: end-to-end refusal + dry-run guarantees

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 15: goreleaser + install scripts + LICENSE

**Files:**
- Create: `.goreleaser.yaml`
- Create: `install.sh`
- Create: `install.ps1`
- Create: `LICENSE`

**Interfaces:**
- Produces: a validated goreleaser config + working install scripts. Tap/bucket blocks reference repos created in Task 17.

- [ ] **Step 1: Write `LICENSE` (MIT)**

`LICENSE`:
```
MIT License

Copyright (c) 2026 Latif Essam

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 2: Write `.goreleaser.yaml`**

`.goreleaser.yaml`:
```yaml
version: 2
project_name: app-dev-clean

before:
  hooks:
    - go mod tidy

builds:
  - id: app-dev-clean
    main: .
    binary: app-dev-clean
    env: [CGO_ENABLED=0]
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - id: default
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: "checksums.txt"

brews:
  - repository:
      owner: latif-essam
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    homepage: "https://github.com/latif-essam/app-dev-clean"
    description: "Cross-platform dev-cache cleaner (RN/Expo/Flutter/native)"
    license: "MIT"
    install: |
      bin.install "app-dev-clean"
      bin.install_symlink "app-dev-clean" => "adc"

scoops:
  - repository:
      owner: latif-essam
      name: scoop-bucket
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    homepage: "https://github.com/latif-essam/app-dev-clean"
    description: "Cross-platform dev-cache cleaner (RN/Expo/Flutter/native)"
    license: "MIT"

release:
  github:
    owner: latif-essam
    name: app-dev-clean
```

- [ ] **Step 3: Write `install.sh`**

`install.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail
REPO="latif-essam/app-dev-clean"
BIN="app-dev-clean"
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in x86_64|amd64) arch=amd64;; arm64|aarch64) arch=arm64;; esac
tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep -o '"tag_name": *"[^"]*"' | head -1 | cut -d'"' -f4)"
url="https://github.com/${REPO}/releases/download/${tag}/${BIN}_${os}_${arch}.tar.gz"
tmp="$(mktemp -d)"
echo "Downloading ${url}"
curl -fsSL "$url" | tar -xz -C "$tmp"
dest="${HOME}/.local/bin"
mkdir -p "$dest"
install -m 0755 "$tmp/${BIN}" "$dest/${BIN}"
ln -sf "$dest/${BIN}" "$dest/adc" || true
echo "Installed to ${dest}/${BIN} (alias: adc). Ensure ${dest} is on your PATH."
```

- [ ] **Step 4: Write `install.ps1`**

`install.ps1`:
```powershell
$ErrorActionPreference = "Stop"
$Repo = "latif-essam/app-dev-clean"
$Bin  = "app-dev-clean"
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$tag  = (Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest").tag_name
$url  = "https://github.com/$Repo/releases/download/$tag/${Bin}_windows_${arch}.zip"
$dest = "$env:LOCALAPPDATA\Programs\$Bin"
New-Item -ItemType Directory -Force -Path $dest | Out-Null
$zip = "$env:TEMP\$Bin.zip"
Write-Host "Downloading $url"
Invoke-WebRequest $url -OutFile $zip
Expand-Archive -Path $zip -DestinationPath $dest -Force
Write-Host "Installed to $dest. Add it to PATH: setx PATH \"$env:PATH;$dest\""
```

- [ ] **Step 5: Validate config**

Run:
```bash
chmod +x install.sh
go install github.com/goreleaser/goreleaser/v2@latest   # if not present
goreleaser check
goreleaser release --snapshot --clean --skip=publish
```
Expected: `goreleaser check` passes; snapshot builds all OS/arch archives under `dist/` without publishing.

- [ ] **Step 6: Commit**

```bash
git add .goreleaser.yaml install.sh install.ps1 LICENSE
git commit -m "build: goreleaser config, install scripts, MIT license

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 16: GitHub Actions — CI + release

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/release.yml`

**Interfaces:**
- Produces: CI running `go test ./...` on linux/macos/windows; release job running goreleaser on tag push using `HOMEBREW_TAP_GITHUB_TOKEN`.

- [ ] **Step 1: Write `ci.yml`**

`.github/workflows/ci.yml`:
```yaml
name: ci
on:
  push:
    branches: [main, go-rewrite]
  pull_request:
jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.23" }
      - run: go test ./...
      - run: go vet ./...
```

- [ ] **Step 2: Write `release.yml`**

`.github/workflows/release.yml`:
```yaml
name: release
on:
  push:
    tags: ["v*"]
permissions:
  contents: write
jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version: "1.23" }
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

- [ ] **Step 3: Validate YAML locally**

Run: `python3 -c "import yaml,sys; [yaml.safe_load(open(f)) for f in ['.github/workflows/ci.yml','.github/workflows/release.yml']]; print('yaml ok')"`
Expected: `yaml ok`.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/
git commit -m "ci: test matrix (linux/mac/windows) + goreleaser release

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 17: README + release rollout (repos, secret, tag, verify)

**Files:**
- Create: `README.md`
- Modify: remove bash tree (`bin/`, `lib/`, `apps/`, `Formula/`, `tests/run.sh`, `docs/plans/2026-07-14-devclean.md`) on merge to main.

**Interfaces:**
- Produces: published v0.1.0 across channels. **Manual maintainer step:** create the `HOMEBREW_TAP_GITHUB_TOKEN` secret (only the user can mint the PAT).

- [ ] **Step 1: Write `README.md`**

`README.md` (complete):
````markdown
# app-dev-clean

Cross-platform dev-cache cleaner for mobile & native projects. Run it from
**anywhere inside a project** — it walks up to the real project root, detects the
project type(s), and cleans the right caches. Refuses to touch anything if you're
not inside a recognized project.

Supports **React Native, Expo, Flutter, native Android (Gradle), and native
iOS/macOS (Xcode/SwiftPM)** on **Windows, macOS, and Linux**.

## Install

**Homebrew (macOS/Linux):**
```bash
brew install latif-essam/tap/app-dev-clean
```

**Scoop (Windows):**
```powershell
scoop bucket add latif-essam https://github.com/latif-essam/scoop-bucket
scoop install app-dev-clean
```

**Install script (macOS/Linux):**
```bash
curl -fsSL https://raw.githubusercontent.com/latif-essam/app-dev-clean/main/install.sh | bash
```

**Install script (Windows PowerShell):**
```powershell
irm https://raw.githubusercontent.com/latif-essam/app-dev-clean/main/install.ps1 | iex
```

**Go:**
```bash
go install github.com/latif-essam/app-dev-clean@latest
```

Or grab a prebuilt binary from [Releases](https://github.com/latif-essam/app-dev-clean/releases).
An `adc` alias is installed alongside the full name.

## Usage

```bash
app-dev-clean                interactive menu (inside a known project)
app-dev-clean ios js         run named targets, no prompt
app-dev-clean local-all      all local targets for the detected type(s)
app-dev-clean nuclear        local-all + global caches + reinstall (confirmed)
app-dev-clean --type flutter scope to one detector
app-dev-clean --dry-run      show what would be freed; delete nothing
app-dev-clean -y             skip confirmation prompts (CI)
app-dev-clean --root         print resolved root + detected type(s)
adc ios js                   same, via the short alias
```

## Safety

- Local targets only ever act on the resolved project root — never your cwd blindly.
- Outside a recognized project, local cleanup is refused (non-zero exit).
- Global caches (`~/.gradle`, Xcode DerivedData, CocoaPods, pub cache) always
  prompt before deletion because they affect every project on the machine.

## Adding a detector

Create `internal/detectors/<type>.go` implementing `detect.Detector`
(`Name`, `Detect(dir)`, `Targets()`) and call `detect.Register(...)` in `init()`.
Detection, the menu, and CLI dispatch pick it up automatically. See existing
detectors for the pattern.

## License

MIT © Latif Essam
````

- [ ] **Step 2: Commit README**

```bash
git add README.md
git commit -m "docs: README with cross-platform install matrix + usage

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

- [ ] **Step 3: Create GitHub repos (maintainer, via gh)**

```bash
gh repo create latif-essam/app-dev-clean --public --source=. --remote=origin --push
gh repo create latif-essam/homebrew-tap --public --add-readme
gh repo create latif-essam/scoop-bucket --public --add-readme
```

- [ ] **Step 4: MANUAL — create the PAT secret (only the user can do this)**

Instruct the user:
1. GitHub → Settings → Developer settings → Personal access tokens → Fine-grained → Generate.
2. Repository access: `homebrew-tap` + `scoop-bucket` (or all repos).
3. Permissions: Contents → Read and write.
4. Copy the token.
5. `gh secret set HOMEBREW_TAP_GITHUB_TOKEN --repo latif-essam/app-dev-clean` (paste when prompted), OR add via the repo's Settings → Secrets → Actions.

Do NOT proceed to Step 6 until the secret exists.

- [ ] **Step 5: Merge to main, remove bash tree**

```bash
git checkout main
git merge --no-ff go-rewrite -m "feat: cross-platform Go rewrite"
git rm -r bin lib apps Formula tests/run.sh docs/plans/2026-07-14-devclean.md
git commit -m "chore: remove legacy bash implementation

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
git push origin main
```

- [ ] **Step 6: Tag + release**

```bash
git tag v0.1.0
git push origin v0.1.0
```
Watch the release workflow: `gh run watch`. Expected: GitHub Release with all archives; formula committed to `homebrew-tap`; manifest committed to `scoop-bucket`.

- [ ] **Step 7: Verify install on each OS**

- macOS/Linux: `brew install latif-essam/tap/app-dev-clean && app-dev-clean --version`
- Windows: `scoop bucket add latif-essam https://github.com/latif-essam/scoop-bucket && scoop install app-dev-clean && app-dev-clean --version`
- Expected: prints `app-dev-clean v0.1.0`.

---

## Self-Review

**1. Spec coverage:**
- Language/repo/module/binary → Task 1, Global Constraints. ✓
- Per-OS cache paths (win/mac/linux, mac-only gating) → Task 2 + Task 6. ✓
- Detector interface + registry + set-based resolution → Tasks 3, 5. ✓
- clean engine + dry-run + size + tolerate errors → Task 4. ✓
- Detectors rn/android/ios/flutter/expo → Tasks 7–11. ✓
- Safety (refuse outside project, confirm globals, nuclear) → Task 12 + e2e Task 14. ✓
- New features: --dry-run, reclaimed report, --type, --yes, --version → Tasks 1, 12. ✓
- Interactive menu (bubbletea, cross-platform, grouped) → Task 13. ✓
- Reinstall PostRunner (RN/Expo) → Task 9 (+ cli wiring Task 12). ✓
- Distribution (goreleaser: releases/brew/scoop, install scripts, go install) → Tasks 15–16. ✓
- Testing (`go test ./...`, per-OS CI) → Tasks 2–14, 16. ✓
- README + rollout + bash removal + LICENSE → Tasks 15, 17. ✓
- Maintainer token step → Task 17 Step 4. ✓

**2. Placeholder scan:** No TBD/TODO. Task 12 flags a real ordering dependency (ui stub) and resolves it inline in Step 3c — not a placeholder.

**3. Type consistency:** `detect.Context`, `detect.Target{Name,Label,Desc,Scope,Paths,Run}`, `Target.Run(ctx) (int64, error)`, `detect.Result{Root,Matched}`, `detect.Resolve`, `detect.Register/Detectors/RegisterGlobal/Globals`, `platform.Paths{...}`, `platform.Detect`, `clean.Size/Remove/Exec/Human`, `ui.Row/Rows/Run/newModel`, `cli.Options/parse/Run/isGlobalName`, detector helper funcs `androidLocalPaths/androidLocal/iosLocalPaths/iosLocal/rnJS/rnMetro/rnWatchman/pkgJSONHas` — all used with consistent names/signatures across tasks.

**Known intentional deviation from spec:** expo/flutter compose android/ios *local* cleanups by reusing `androidLocal`/`iosLocal`; global-cache composition for those types happens by selecting the shared global targets in the menu / via `nuclear`, matching the spec's "shared helpers" intent. `expo` embeds `rn` to inherit reinstall; `flutter` intentionally has no reinstall hook (no node_modules/Pods — `flutter clean`/`pub get` are the user's rebuild step).

**Pre-flight plan fixes (2026-07-23, before execution):** (1) removed a no-op `os` import in the Task 1 test; (2) added `Context.Force` + made `rn.PostRun` honor DryRun/Force/Yes so `nuclear` reinstalls unconditionally (spec requires reinstall) while dry-run never does; (3) unified menu/CLI combo expansion in `cli.Run` and tracked `nuclear` before expansion; (4) made `clean.Exec` separator-aware and `androidLocal` pass an ABSOLUTE gradlew path (bare `./gradlew` does not resolve against `cmd.Dir` in `os/exec`); (5) `expo` embeds `rn` to inherit reinstall.
