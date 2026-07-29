# app-dev-clean

Cross-platform dev-cache cleaner CLI (Go). Walks up from cwd to the real project
root, detects project type(s), cleans the right caches. Windows/macOS/Linux.

## Current status (2026-07-27)

- **Go rewrite is merged to `main`** via PR #1 (`c1f7ad7`, 30 commits). CI runs on
  ubuntu + macos + windows and is green. The legacy bash tree is deleted.
- **Repo is public: `github.com/latif-essam/app-dev-clean`.**
- **`v0.1.0` is released.** Tag push ran `release.yml` → goreleaser green: 6
  archives + checksums on the GitHub Release, formula committed to
  `latif-essam/homebrew-tap`, manifest to `latif-essam/scoop-bucket`.
  `HOMEBREW_TAP_GITHUB_TOKEN` is set (expires — re-mint per `PUBLISHING.md`).
- **Verified:** `brew install latif-essam/tap/app-dev-clean` + `adc` alias on
  macOS arm64, and `go install …@v0.1.0`, both reporting `0.1.0`.
- **Still unverified: Scoop on real Windows** — the `adc` shim is a `post_install`
  block that has never run on Windows hardware. Non-fatal by design.
- Cutting the next release = tag a green commit; everything else is automatic.
  See `PUBLISHING.md`. Tier 2 (homebrew-core) is prepped in `docs/homebrew-core.md`.
- Design spec + task plan: `docs/superpowers/specs/` and `docs/superpowers/plans/`.

Local dir is `devclean` but the module/binary is `app-dev-clean` — intentional.

## Own tap ≠ "in Homebrew"

`brew install latif-essam/tap/app-dev-clean` works off our own tap, but is **not
discoverable**: `brew search` only covers homebrew-core plus taps already added
locally. Plain `brew install app-dev-clean` and a formulae.brew.sh listing require
a PR to `Homebrew/homebrew-core`, and a core formula **builds from source** — it
would be hand-written, not the goreleaser-generated binary formula. Same split on
Scoop: our bucket vs. Scoop's main bucket. Don't conflate the two when discussing
release scope.

## Commands

- `go test ./...` — full suite (includes `e2e_test.go`, which builds + runs the real binary)
- `gofmt -w main.go main_test.go e2e_test.go internal/` — **run before every commit**; `gofmt -l` must be empty
- `go vet ./...`
- `for c in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do GOOS=${c%/*} GOARCH=${c#*/} go build -o /dev/null . || echo "FAIL $c"; done` — cross-compile check
- `HOMEBREW_TAP_GITHUB_TOKEN=dummy goreleaser release --snapshot --clean` — full release dry-run into `dist/` (gitignored)
- `goreleaser check` — config lint; **the `brews` deprecation warning is expected, see below**

**Manual behaviour check** (catches what unit tests miss — the dry-run output *is*
the resolved path list): build synthetic projects (rn/bare-expo/managed-expo/
flutter/native-android/native-ios/empty), run `app-dev-clean local-all --dry-run -y`
in each, and diff a `find`-based file listing before/after — it must be identical.
Swap `gradlew` for a script that appends to a sentinel file to prove `--dry-run`
never executes it.

## Architecture

`main.go` (thin) → `internal/cli` (flags, dispatch, confirm gates, combos) over
`internal/detect` (Detector interface + registry + walk-up resolver),
`internal/detectors` (rn/android/ios/flutter/expo, registered in `init()`),
`internal/platform` (per-OS cache paths), `internal/clean` (size/remove/exec,
dry-run engine), `internal/ui` (bubbletea checklist).

**Adding a detector:** one file in `internal/detectors/` implementing
`Name/Detect/Targets` + `detect.Register(x{})` in `init()`. Nothing else changes.

Detectors return a **set** — a repo can match several types (bare Expo is both
`expo` and `rn`); offered targets are the deduped union.

## Gotchas (all bit us once)

- **Registry needs a blank import.** `detect.Globals()`/`Detectors()` are filled
  by `detectors` package `init()`. Any binary *or test* that depends on them must
  `_ "…/internal/detectors"` — see `main.go` and `internal/cli/parse_test.go`.
  Without it `isGlobalName` silently sees zero globals.
- **Never pass a bare relative path to `clean.Exec`** (e.g. `./gradlew`). It does
  not resolve against `cmd.Dir` in `os/exec`. Pass an absolute path; `Exec` is
  separator-aware and stats explicit paths directly.
- **Native path builders take a BASE dir, not the project root.**
  `androidLocalPaths`/`iosLocalPaths` are relative to whatever holds the gradle /
  Xcode project, which is *not* always the detected root: native repo → the root
  itself (`projectRoot`), RN/bare Expo → `<root>/android`, `<root>/ios`
  (`androidSubdir`/`iosSubdir` in `shared.go`). Passing the root for RN made every
  native target probe a nonexistent path and report `0 B` while
  `android/app/build` was full, and `gradlew clean` never ran. Build targets via
  the `androidTarget`/`iosTarget` factories so the two geometries can't drift.
- **Test `Targets()` paths, not just `Detect()`.** The bug above survived a green
  suite because rn/expo tests only asserted detection. Assert the resolved path
  set — with negative cases, so an RN prefix can't leak into a native repo.
- **Windows can't exec an extension-less file.** Tests that `go build -o <tmp>/adc`
  then run it must append `.exe` on `GOOS=windows` — see `exeName` in
  `e2e_test.go`. Passes locally on macOS, fails only on the Windows runner.
- **`filepath.Join` uses the HOST separator**, not the `goos` you're simulating.
  In per-OS tests build expected values with `filepath.Join` too — hardcoded
  `/`-literals fail on the Windows CI runner.
- **`--dry-run` must never delete or run destructive external commands**
  (`gradlew clean`, `flutter clean`) — only report. Guard new targets accordingly.
- **`brews` (Homebrew formula) is deliberate**, though goreleaser deprecates it in
  favor of `homebrew_casks`: the binary is unsigned and casks quarantine unsigned
  binaries (Gatekeeper), while formulae install clean and support the `adc`
  symlink. Don't "upgrade" it unless the binary gets code-signed. Rationale is
  also commented in `.goreleaser.yaml`.
- macOS-only caches (Xcode DerivedData, CocoaPods) must resolve to `""` on
  Win/Linux and not be offered there.

## Conventions

- Conventional Commits, ending with a `Co-Authored-By:` trailer naming the model
  that actually wrote the commit (e.g. `Claude Opus 5 (1M context)
  <noreply@anthropic.com>`) — don't copy a stale model name forward.
- TDD: failing test → minimal implementation → green → commit.
- Deletion always goes through `clean.Remove` so dry-run + size accounting stay
  uniform; absent paths and per-path errors are logged and skipped, never fatal.

## Working style note

Per-task subagent dispatch was tried for this rewrite and abandoned after 4 tasks
— wall-clock ran 30 min to 2.7 hrs per subagent. Inline execution (write, test,
commit, batch several tasks per turn) finished the remaining 13 tasks in minutes.
Prefer inline work here unless a task genuinely needs isolated exploration.
