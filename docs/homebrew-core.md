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
The `sha256` below is the real checksum of the `v0.1.0` source tarball, verified
2026-07-29. Re-derive it for any later version with the command in step 1.

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
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}")
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
- **`std_go_args`** supplies `-o #{bin}/#{name}` and the trim/mod flags core
  expects; don't hand-roll `-o`.
- **`-X main.version=#{version}`** matches how `.goreleaser.yaml` stamps it
  (`var version = "dev"` in `main.go`), so `--version` reports the real number
  instead of `dev`.
- **`test do`** must exercise real behaviour, not just `--version`. The refusal
  path is ideal: deterministic, needs no fixture, and asserts the safety
  guarantee. Never write a test that deletes anything.

## Checklist

Do these in order, after `v0.1.0` is tagged and the release exists.

1. Confirm the tarball checksum (already filled in for v0.1.0 — redo per version):
   ```bash
   curl -fsSL https://github.com/latif-essam/app-dev-clean/archive/refs/tags/v0.1.0.tar.gz | shasum -a 256
   # expect d50a6103b61aff56d05f21cf65ca189afff0c521715e62058d94f324fecab984
   ```

2. Fork and branch:
   ```bash
   brew tap --force homebrew/core
   cd "$(brew --repository homebrew/core)"
   git checkout -b app-dev-clean origin/master
   ```

3. Drop the formula at `Formula/a/app-dev-clean.rb` (core shards by first letter).

4. Validate locally — all three must pass before opening the PR:
   ```bash
   brew audit --strict --new-formula --online app-dev-clean
   brew install --build-from-source app-dev-clean
   brew test app-dev-clean
   ```
   `--new-formula` runs extra checks that only apply to first submissions.

5. Open the PR against `Homebrew/homebrew-core`. Title it exactly
   `app-dev-clean 0.1.0 (new formula)`. In the body, state what the tool does,
   confirm the licence, and link the release.

6. Expect review comments. Common ones: `desc` wording, the alias question
   (see above), and whether the test is meaningful. Respond by amending and
   force-pushing the same branch.

Once merged, `brew bump-formula-pr` / BrewTestBot handles future versions
automatically when it detects a new tag — no further action per release. Our own
tap keeps updating in parallel via goreleaser; the two coexist fine.

## Scoop main bucket — the equivalent

`ScoopInstaller/Main` is the analogue. Same idea: a PR adding a manifest, which
then makes `scoop install app-dev-clean` work with no bucket added. Our bucket
stays as-is. Worth doing after the core formula lands, since the Windows side is
also the least-tested path (see the `adc` shim caveat in `PUBLISHING.md`).
