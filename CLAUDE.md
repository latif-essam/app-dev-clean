# app-dev-clean

Cross-platform dev-cache cleaner CLI (Go). Walks up from cwd to the real project
root, detects project type(s), cleans the right caches. Windows/macOS/Linux.

## Current status (2026-07-24)

- **All code complete + validated on branch `go-rewrite`** (25 commits). Tests,
  vet, gofmt clean; cross-compiles to all 6 targets; goreleaser snapshot builds
  the full release (archives + brew formula + scoop manifest).
- **Not pushed to any remote. No GitHub repo exists yet. Not merged to `main`.**
- **Next step = publish.** Follow `PUBLISHING.md` (repos to create, the PAT the
  user must mint, ordered steps). User deferred this pending their code review.
- Design spec + task plan: `docs/superpowers/specs/` and `docs/superpowers/plans/`.

## Repo is mid-migration — do not touch the legacy bash tree

`bin/ lib/ apps/ Formula/ tests/` and `docs/plans/2026-07-14-devclean.md` are the
**old bash implementation**, still present so the installed `dev-app-clean`
symlink keeps working until the Go version ships. They get deleted when
`go-rewrite` merges to `main`. Never edit or "fix" them.

Local dir is `devclean` but the module/binary is `app-dev-clean` — intentional.

## Commands

- `go test ./...` — full suite (includes `e2e_test.go`, which builds + runs the real binary)
- `gofmt -w main.go e2e_test.go internal/` — **run before every commit**; `gofmt -l` must be empty
- `go vet ./...`
- `for c in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do GOOS=${c%/*} GOARCH=${c#*/} go build -o /dev/null . || echo "FAIL $c"; done` — cross-compile check
- `HOMEBREW_TAP_GITHUB_TOKEN=dummy goreleaser release --snapshot --clean` — full release dry-run into `dist/` (gitignored)
- `goreleaser check` — config lint; **the `brews` deprecation warning is expected, see below**

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

- Conventional Commits, ending with:
  `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`
- TDD: failing test → minimal implementation → green → commit.
- Deletion always goes through `clean.Remove` so dry-run + size accounting stay
  uniform; absent paths and per-path errors are logged and skipped, never fatal.

## Working style note

Per-task subagent dispatch was tried for this rewrite and abandoned after 4 tasks
— wall-clock ran 30 min to 2.7 hrs per subagent. Inline execution (write, test,
commit, batch several tasks per turn) finished the remaining 13 tasks in minutes.
Prefer inline work here unless a task genuinely needs isolated exploration.
