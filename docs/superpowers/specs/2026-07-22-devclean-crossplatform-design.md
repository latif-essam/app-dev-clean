# devclean — Cross-Platform Go Rewrite Design

**Date:** 2026-07-22
**Status:** Approved (brainstorming) — pending implementation plan
**Supersedes runtime of:** `docs/plans/2026-07-14-devclean.md` (bash version)

## Goal

Turn `devclean` from a macOS-only bash tool into a **single cross-platform Go
binary** that runs first-class on Windows, macOS, and Linux (intel + arm),
installs through mainstream package managers, and cleans dev caches for
whatever project type(s) you're standing in — resolving the real project root by
walking up from cwd, exactly like the bash version does today.

One tool, many pluggable detectors. Not sub-tools per platform.

## Non-Goals (YAGNI cuts, agreed)

- **No winget in v1.** Requires manual PRs into `microsoft/winget-pkgs` + MS
  review. Scoop covers Windows for v1; winget is a later follow-up.
- **No generic Node/web detector in v1.** Not requested. The detector interface
  makes it a one-file add later.
- **No live pre-delete size scan in the interactive menu.** `--dry-run` (shows
  estimated space to free) plus a post-run "reclaimed" total is enough. Live
  per-row sizing can be added later if wanted.
- **No sub-tool split** (`devclean-rn`, `devclean-flutter`, …). One binary,
  scoping via `--type`.

## Architecture

### Language & repo

- **Go 1.23+**, module path `github.com/latif-essam/devclean`, binary `devclean`.
- **Reuse the existing repo** at `~/dev-tools/devclean` — preserve git history
  (3 commits + the bash implementation). Create GitHub remote
  `github.com/latif-essam/devclean` (public) and push.
- **Transition strategy:** develop the Go rewrite on branch `go-rewrite`. Keep
  `main` (bash) working and installable until the Go version reaches feature
  parity and green tests. Then fast-forward/merge to `main`, remove the bash
  `bin/lib/apps` tree, and cut the first tagged release. No period where the
  installed tool is broken.

### Repo layout (target)

```
devclean/
  main.go                     # thin: build cli, run, os.Exit(code)
  internal/
    detect/
      detector.go             # Detector interface, Target type, registry
      resolve.go              # walk-up root resolution -> matched detector set
    detectors/
      rn.go                   # React Native (bare)
      android.go              # native Android (Gradle)
      ios.go                  # native iOS/macOS (Xcode/SwiftPM)
      flutter.go              # Flutter
      expo.go                 # Expo (managed + bare)
    platform/
      paths.go                # per-OS cache path resolution (GOOS + env)
    clean/
      clean.go                # remove paths, size-estimate, dry-run engine
      shared.go               # shared global-cache helpers (gradle/xcode/pods/pub)
    cli/
      cli.go                  # flag parsing, dispatch, confirm gates
    ui/
      menu.go                 # bubbletea interactive checklist
  .goreleaser.yaml
  .github/workflows/
    ci.yml                    # go test ./... on push/PR (linux+mac+windows)
    release.yml               # goreleaser on tag push
  install.sh                  # curl | bash  (mac/linux)
  install.ps1                 # irm | iex     (windows)
  docs/superpowers/specs/2026-07-22-devclean-crossplatform-design.md
  README.md
```

Files stay small and single-purpose so each unit is independently testable.

### Core model

**`Detector` interface** (`internal/detect`):

```go
type Detector interface {
    Name() string             // "rn", "android", "ios", "flutter", "expo"
    Detect(dir string) bool   // is dir a root of this type?
    Targets() []Target        // cleanup targets this type offers
}
```

**`Target`:**

```go
type Target struct {
    Name  string              // "ios", "js", "gradle-global"
    Label string              // menu label
    Desc  string              // menu description
    Scope Scope               // Local | Global
    Paths func(ctx Context) []string   // paths this target deletes (for dry-run/size)
    Run   func(ctx Context) error      // perform cleanup (may also shell out)
}
```

- `ctx Context` carries `ProjectRoot`, resolved `platform` paths, dry-run flag,
  and a logger. Deletion always goes through the `clean` engine so dry-run and
  size accounting are uniform.

**Registry:** detectors register into a slice at package init (mirrors the
bash `register_app`). Adding a type = one file in `detectors/` + one
`register()` call. No edits to core.

**Root resolution** (`resolve.go`): walk up from cwd to filesystem root. At the
first dir where *any* registered detector's `Detect` returns true, that dir is
`ProjectRoot` and the **set** of all detectors matching there is the active set.
The offered targets are the union across the matched set (deduped by target
name). If no dir matches anywhere up-tree, resolution fails → local cleanup is
refused (see Safety).

**Per-OS paths** (`platform/paths.go`): resolves global cache locations from
`runtime.GOOS` + environment. macOS-only caches (Xcode DerivedData, CocoaPods)
return "unavailable" on Windows/Linux and their targets are simply not offered.

| Cache | macOS | Linux | Windows |
|---|---|---|---|
| Gradle | `~/.gradle/caches` | `~/.gradle/caches` | `%USERPROFILE%\.gradle\caches` |
| Xcode DerivedData | `~/Library/Developer/Xcode/DerivedData` | — | — |
| CocoaPods | `~/Library/Caches/CocoaPods` | — | — |
| Pub (Flutter) | `~/.pub-cache` | `~/.pub-cache` | `%LOCALAPPDATA%\Pub\Cache` |
| Metro/Haste tmp | `$TMPDIR` | `/tmp` | `%TEMP%` |

## Detectors (v1)

All five ship in v1. Local targets operate on the resolved `ProjectRoot`, never
raw cwd. Global targets touch shared `~`/`%USERPROFILE%` caches and are gated.

### android-native
- **Marker:** `settings.gradle`/`settings.gradle.kts` or root `build.gradle`
  present *at this dir*. In an RN/Flutter project these files live under
  `android/`, not the project root, so at the JS/Dart root this detector does
  not match — no exclusion logic needed. The active set is determined purely by
  which markers exist at the resolved dir (order-independent).
- **Local:** `build/`, `app/build/`, `.gradle/`, `.cxx/`, `app/.cxx/`; run
  `./gradlew clean` (or `gradlew.bat` on Windows) if wrapper present.
- **Global (shared helper):** Gradle caches.

### ios-native (macOS only)
- **Marker:** `*.xcodeproj`, `*.xcworkspace`, or `Package.swift`.
- **Local:** `build/`, `Pods/`, `Podfile.lock`, SwiftPM `.build/`.
- **Global (shared helper):** Xcode DerivedData, CocoaPods cache.
- On Windows/Linux this detector never matches (no Xcode).

### rn (React Native bare)
- **Marker:** `package.json` containing `"react-native"` AND (`android/` OR
  `ios/`). (Ported verbatim behavior from current `rn_markers`.)
- **Local:** `node_modules/`, lockfile, Metro/Haste temp caches, kill Metro on
  `:8081`, watchman reset; plus the android + ios local cleanups.
- **Global:** Gradle, Xcode DD, CocoaPods (composed via shared helpers).
- **Post-run:** offer `npm install` / `pod install` reinstall (interactive only,
  skipped under `-y` unless `nuclear`).

### expo
- **Marker:** `package.json` containing `"expo"`.
- **Local:** everything `rn` cleans **plus** `.expo/`; handles both managed
  (no `android/ios` — clean `.expo/`, prebuild output) and bare (has native
  dirs — full RN cleanup). Detection overlaps: a bare Expo app matches both
  `expo` and `rn`; union handles it.
- **Global:** same as RN.

### flutter
- **Marker:** `pubspec.yaml` containing a `flutter:` key / `sdk: flutter` dep.
- **Local:** `build/`, `.dart_tool/`; run `flutter clean` if `flutter` on PATH.
- **Global:** pub cache; plus android/ios globals when the Flutter project also
  contains `android/`/`ios/`.

## Safety model (preserved from bash)

- Local targets require a resolved `ProjectRoot`. If cwd is not inside a
  recognized project, **refuse** local cleanup and exit non-zero — never delete
  in the wrong place.
- Global cache targets are allowed anywhere (no project needed) but always
  **prompt for confirmation**, because they affect every project on the machine.
- `nuclear` (local-all + globals + reinstall) always confirms.
- `--yes/-y` bypasses confirmation prompts for scripting/CI.

## New features

- **`--dry-run`:** resolve targets, walk their paths, print each path and an
  **estimated space that would be freed** (sum of dir sizes), delete nothing.
- **Reclaimed report:** after a real run, print total space freed.
- **`--type <name>`:** restrict to one detector even when the repo matches
  several (e.g. `--type flutter` in a repo that also has native `android/`).
- **`--version`:** version string embedded at build time by goreleaser
  (`-ldflags -X main.version=...`).

## CLI surface

```
devclean                 interactive menu (must be inside a known project)
devclean <target>...     run named targets (e.g. ios js metro)
devclean local-all       all local targets for detected type(s)
devclean nuclear         local-all + global caches + reinstall (confirmed)
devclean --type <t>      scope to one detector (rn|android|ios|flutter|expo)
devclean --dry-run       show what would be freed; delete nothing
devclean -y, --yes       skip confirmation prompts
devclean --root          print resolved project root + detected type(s)
devclean --version       print version
devclean --help          usage
```

Global targets (project-agnostic): `gradle-global`, `xcode-dd` (mac),
`pods-cache` (mac), `pub-cache`.

## Interactive menu → bubbletea

- Use `charmbracelet/bubbletea` for the checklist TUI. It handles Windows
  terminals correctly — the exact raw-mode problem (`read -rsn1`, `tput`) that
  makes the bash menu non-portable.
- Same UX as today: arrow/`j`/`k` move, SPACE toggle, `a` all, `n` none, ENTER
  run, `q` quit. Rows grouped by detected type with LOCAL / GLOBAL / COMBO
  section headers. Selected targets flow into the same dispatch path as CLI args.

## Distribution → goreleaser + GitHub Actions

A single tag push (`git tag v0.1.0 && git push --tags`) triggers `release.yml`,
which runs goreleaser to produce:

- **GitHub Releases:** archives for `darwin/linux/windows` × `amd64/arm64` plus a
  checksums file. Version embedded via ldflags.
- **Homebrew:** goreleaser auto-commits/updates a formula in tap repo
  `latif-essam/homebrew-tap` → `brew install latif-essam/tap/devclean`.
- **Scoop:** goreleaser auto-updates a manifest in a scoop bucket repo →
  `scoop install devclean` (Windows).
- **Install scripts** (committed in-repo):
  - `install.sh` — `curl -fsSL https://raw.githubusercontent.com/latif-essam/devclean/main/install.sh | bash`
    detects OS/arch, downloads the matching release archive, installs to
    `~/.local/bin` (or `/usr/local/bin`).
  - `install.ps1` — `irm https://raw.githubusercontent.com/latif-essam/devclean/main/install.ps1 | iex`
    for Windows PowerShell.
- **`go install github.com/latif-essam/devclean@latest`** works for Go users
  with no extra setup.

winget: deferred (documented as future).

## Testing

- `go test ./...` — table-driven tests:
  - **detectors:** each detector's `Detect` over fixture dirs (positive +
    negative), created under `t.TempDir()`.
  - **resolve:** walk-up from nested subdir finds correct root + detector set;
    refuse-outside-project returns error; multi-type repo returns union.
  - **platform:** path resolution per-OS by injecting GOOS + env (table-driven);
    macOS-only caches absent on win/linux.
  - **clean:** dry-run computes sizes and deletes nothing; real run removes and
    reports reclaimed bytes; absent paths tolerated.
- **CI (`ci.yml`):** run `go test ./...` on ubuntu + macos + windows runners on
  every push/PR. Release job gated on green CI.

## Migration / rollout

1. Branch `go-rewrite` off `main`.
2. Implement core + detectors + CLI + menu; reach parity with bash behavior.
3. Green `go test ./...` on all three OSes in CI.
4. Wire goreleaser + install scripts; dry-run a release build locally
   (`goreleaser release --snapshot --clean`).
5. Create GitHub repo, push, create `homebrew-tap` + scoop bucket repos.
6. Merge to `main`, remove bash tree, tag `v0.1.0`, verify install on each OS.
7. Update README (install matrix, usage, adding a detector).

## Open items to resolve during planning

- Confirm tap/bucket repo names (`latif-essam/homebrew-tap`, scoop bucket name).
- Decide `flutter clean` / `gradlew clean` invocation policy under `--dry-run`
  (dry-run must NOT run destructive external commands — only report paths).
- Windows path/permission edge cases for locked files (e.g. Gradle daemon
  holding a handle) — clean engine should tolerate per-path failures like the
  bash `nuke` does and continue.
```
