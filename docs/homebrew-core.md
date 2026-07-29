# Submitting app-dev-clean to homebrew-core

This is **Tier 2** — the thing that makes `brew install app-dev-clean` work with
no tap, and gets a `formulae.brew.sh/formula/app-dev-clean` page (which Google
indexes). It is separate from, and does not replace, our own tap.

## Tier 1 vs Tier 2 — what actually differs

| | Own tap (`latif-essam/homebrew-tap`) | homebrew-core |
|---|---|---|
| Install | `brew install latif-essam/tap/app-dev-clean` | `brew install app-dev-clean` |
| Found by `brew search`? | Only after the user taps | Yes, out of the box |
| Listed on formulae.brew.sh | No | Yes |
| Formula author | goreleaser, automatically per tag | hand-written, maintained by Homebrew |
| How it installs | downloads our prebuilt release archive | **builds from source** on Homebrew CI, then bottles it |
| Version bumps | automatic on every tag | Homebrew's `BrewTestBot` opens a PR when it sees a new tag |

Because core builds from source, its formula is **not** the one goreleaser
generates. It's the file below.

## Eligibility — checked against docs.brew.sh/Acceptable-Formulae (2026-07-27)

The current document contains **no popularity threshold** — the words "notable",
"popular", "stars", "forks" and "watchers" do not appear in it. The old
30-forks / 30-watchers / 75-stars rule is gone from that page. The hard rules
that do apply:

| Rule | Status |
|---|---|
| Upstream provides an immutable stable tag or release | ✅ once `v0.1.0` is tagged |
| Builds from source, or installs portable output | ✅ Go source build |
| No fetching from a moving branch or unchecksummed archive | ✅ pinned tag tarball + sha256 |
| Open source, licence compatible with the DFSG | ✅ MIT |
| Not self-updating | ✅ |
| Not a native macOS `.app` (that would be a cask) | ✅ |

Maintainer discretion still applies — nothing here guarantees acceptance.

## Known review risk: the `adc` alias

Our tap formula installs a short `adc` symlink. **Drop it for the core
submission.** `adc` is a generic three-letter name in a shared `bin` directory,
and core reviewers push back on formulae claiming names that could collide with
another package. The formula below omits it deliberately.

Users who want the short name can still `alias adc=app-dev-clean` in their
shell, and our own tap keeps installing the symlink. Don't re-add it to the core
formula to "match" the tap — that's the likeliest reason for a rejection.

## The formula

Goes to `Formula/a/app-dev-clean.rb` in a fork of `Homebrew/homebrew-core`.

**This exact file has been validated locally** against Homebrew 6.0.13 (macOS
arm64, 2026-07-29) — see the verification block below the formula. The `sha256`
is the real checksum of the `v0.1.0` source tarball.

```ruby
class AppDevClean < Formula
  desc "Dev-cache cleaner for React Native, Expo, Flutter, Android, and iOS projects"
  homepage "https://github.com/latif-essam/app-dev-clean"
  url "https://github.com/latif-essam/app-dev-clean/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "d50a6103b61aff56d05f21cf65ca189afff0c521715e62058d94f324fecab984"
  license "MIT"
  head "https://github.com/latif-essam/app-dev-clean.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-X main.version=#{version}")
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/app-dev-clean --version")

    # Outside a recognized project the tool must refuse local cleanup and exit 1.
    # testpath is a fresh temp dir, so the walk-up finds no project markers.
    output = shell_output("#{bin}/app-dev-clean js 2>&1", 1)
    assert_match "refusing local cleanup", output
  end
end
```

Notes on the choices:

- **`desc`** must not begin with the formula name or an article, and is kept
  under 80 chars — Homebrew's `brew audit` enforces both.
- **`std_go_args`** supplies `-o #{bin}/#{name}` **and `-s -w`**; don't hand-roll
  `-o`, and don't repeat `-s -w` in `ldflags:`. Passing
  `ldflags: "-s -w -X …"` emits `-ldflags=-s -w -s -w -X …` — harmless but
  duplicated, and the kind of thing a reviewer flags. Pass only the `-X`.
- **`-X main.version=#{version}`** matches how `.goreleaser.yaml` stamps it
  (`var version = "dev"` in `main.go`), so `--version` reports the real number
  instead of `dev`. Confirmed: the source build reports `0.1.0`.
- **`test do`** must exercise real behaviour, not just `--version`. The refusal
  path is ideal: deterministic, needs no fixture, and asserts the safety
  guarantee. Never write a test that deletes anything.

## Local verification (done — Homebrew 6.0.13, macOS arm64, 2026-07-29)

The formula above was installed into a scratch tap (`brew tap-new local/coretest`)
and put through the full gate. All three passed:

| Command | Result |
|---|---|
| `brew audit --new local/coretest/app-dev-clean` | exit 0, **zero output** |
| `brew style local/coretest/app-dev-clean` | exit 0, "1 file inspected, no offenses detected" |
| `brew install --build-from-source local/coretest/app-dev-clean` | built in 1s, `--version` → `0.1.0` |
| `brew test local/coretest/app-dev-clean` | exit 0, both assertions ran |

Also confirmed: the source build installs **only** `bin/app-dev-clean`, no `adc`
symlink — the deliberate omission described above is actually in effect.

**Flag name changed.** Older docs (and the previous version of this file) say
`brew audit --strict --new-formula`. In current Homebrew that flag does not exist:

```
Error: invalid option: --new-formula
```

It is now `--new`, which *implies* `--strict` and `--online`. Use `--new` alone.

**Name is free.** `https://formulae.brew.sh/api/formula/app-dev-clean.json`
returns 404, so nothing in core claims the name. (`app-cleaner` exists and is
unrelated; expect a reviewer to glance at the overlap.)

## Submitting it

Validation is already done, so this is just the PR mechanics. Steps 1-2 clone
homebrew-core, which is a large repo — expect it to take a few minutes.

```bash
# 1. get a local homebrew-core checkout and branch off master
brew tap --force homebrew/core
cd "$(brew --repository homebrew/core)"
git fetch origin
git checkout -b app-dev-clean origin/master

# 2. write the formula (core shards formulae by first letter)
#    copy the ruby block from this document into:
#    Formula/a/app-dev-clean.rb

# 3. re-run the gate in the real core context
brew audit --new app-dev-clean
brew style app-dev-clean
brew install --build-from-source app-dev-clean
brew test app-dev-clean

# 4. commit — the message format matters to Homebrew
git add Formula/a/app-dev-clean.rb
git commit -m "app-dev-clean 0.1.0 (new formula)"

# 5. push to your fork and open the PR
gh repo fork Homebrew/homebrew-core --remote --remote-name fork
git push fork app-dev-clean
gh pr create --repo Homebrew/homebrew-core --base master \
  --title "app-dev-clean 0.1.0 (new formula)" --body "…"
```

The PR title must be exactly `app-dev-clean 0.1.0 (new formula)` — Homebrew's
automation parses it.

Suggested PR body:

> Cross-platform dev-cache cleaner for React Native, Expo, Flutter, native
> Android (Gradle) and native iOS/macOS (Xcode/SwiftPM) projects. Walks up from
> the working directory to the real project root, detects the project type(s),
> and cleans only the caches that apply. Refuses to act outside a recognised
> project, and `--dry-run` reports reclaimable space without deleting anything.
>
> - Upstream: https://github.com/latif-essam/app-dev-clean
> - Release: https://github.com/latif-essam/app-dev-clean/releases/tag/v0.1.0
> - Licence: MIT
> - `brew audit --new`, `brew style`, `brew install --build-from-source` and
>   `brew test` all pass locally on macOS arm64.

Then restore your machine afterwards, since building from source replaces the
tap install:

```bash
brew uninstall app-dev-clean
brew install latif-essam/tap/app-dev-clean   # brings back the adc alias
```

### Expect review comments

Likely ones, and the prepared answer for each:

| Comment | Response |
|---|---|
| "why no `adc` alias / add one" | deliberate — generic 3-letter name in a shared `bin`; the alias lives in our own tap |
| `desc` wording | it's already audit-clean; reword as asked, it's cosmetic |
| "is the test meaningful" | it asserts the safety guarantee (refusal outside a project) plus the version, and deletes nothing |
| overlap with `app-cleaner` | unrelated tool; this one targets mobile/native build caches |

Amend and force-push the same branch to respond.

Once merged, `brew bump-formula-pr` / BrewTestBot handles future versions
automatically when it detects a new tag — no further action per release. Our own
tap keeps updating in parallel via goreleaser; the two coexist fine.

## Scoop main bucket — the equivalent

`ScoopInstaller/Main` is the analogue. Same idea: a PR adding a manifest, which
then makes `scoop install app-dev-clean` work with no bucket added. Our bucket
stays as-is. Worth doing after the core formula lands, since the Windows side is
also the least-tested path (see the `adc` shim caveat in `PUBLISHING.md`).
