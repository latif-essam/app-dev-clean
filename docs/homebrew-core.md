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

## BLOCKER: the notability gate (measured 2026-07-29)

**`brew audit --new` fails for this repo, and that is a hard stop:**

```
Error: 1 problem in 1 formula detected.
app-dev-clean
  * GitHub repository not notable enough (<30 forks, <30 watchers and <75 stars)
```

Current: **0 stars, 0 forks, 0 watchers.** The gate needs **75 stars OR 30 forks
OR 30 watchers**. Until one of those is met, a core PR cannot pass CI — this is
mechanical, not a matter of reviewer taste.

**Don't trust the docs on this — trust the audit.** The rule is *not* in
`docs.brew.sh/Acceptable-Formulae`; searching that page (and homebrew-core's
`CONTRIBUTING.md`) for "notable", "popular", "stars", "forks" and "watchers"
returns nothing. An earlier version of this document therefore claimed there was
no popularity requirement. That was wrong: the check is implemented in
`brew audit --new`, not written in the prose.

It's also easy to miss because of *where* it fires:

| Where audited | Command | Result |
|---|---|---|
| scratch tap (`local/coretest`) | `brew audit --new` | exit 0 — check skipped for third-party taps |
| homebrew-core checkout | `brew audit --strict` | exit 0 — notability is `--new`-only |
| homebrew-core checkout | `brew audit --new` | **exit 1 — not notable enough** |

So a green audit in your own tap proves nothing about core eligibility. Only
`--new` inside a homebrew-core checkout gives the real answer.

### What this means

Tier 2 is deferred until the repo gains traction — the formula is otherwise
ready, so it becomes a ~10-minute job once the threshold is met. Watch it with:

```
gh api repos/latif-essam/app-dev-clean --jq '"stars=\(.stargazers_count) forks=\(.forks_count) watchers=\(.subscribers_count)"'
```

Meanwhile the own tap works on every machine with no setup, which is the whole
of Tier 1 and needs nothing from Homebrew.

## Other eligibility rules — checked against docs.brew.sh/Acceptable-Formulae

These are all satisfied; notability above is the only failure.

| Rule | Status |
|---|---|
| Upstream provides an immutable stable tag or release | ✅ once `v0.1.0` is tagged |
| Builds from source, or installs portable output | ✅ Go source build |
| No fetching from a moving branch or unchecksummed archive | ✅ pinned tag tarball + sha256 |
| Open source, licence compatible with the DFSG | ✅ MIT |
| Not self-updating | ✅ |
| Not a native macOS `.app` (that would be a cask) | ✅ |
| **Repo notable enough (75 stars / 30 forks / 30 watchers)** | ❌ **0 / 0 / 0 — see blocker above** |

Maintainer discretion still applies on top of all of this.

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
| `brew audit --new local/coretest/app-dev-clean` | exit 0, zero output — **but see the notability blocker; this check is skipped outside core** |
| `brew style local/coretest/app-dev-clean` | exit 0, "1 file inspected, no offenses detected" |
| `brew install --build-from-source local/coretest/app-dev-clean` | built in 1s, `--version` → `0.1.0` |
| `brew test local/coretest/app-dev-clean` | exit 0, both assertions ran |

Re-run inside the homebrew-core checkout, everything passed except notability:

| Command | Result |
|---|---|
| `HOMEBREW_NO_INSTALL_FROM_API=1 brew install --build-from-source app-dev-clean` | exit 0, built in 1s |
| `brew test app-dev-clean` | exit 0 |
| `brew style app-dev-clean` | exit 0, no offenses |
| `brew audit --strict app-dev-clean` | exit 0 |
| `brew audit --new app-dev-clean` | **exit 1 — not notable enough** |

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

## State of the submission (2026-07-29)

**A PR was opened and is currently closed. Do not open a new one.**

- PR: https://github.com/Homebrew/homebrew-core/pull/296007
- Fork: `latif-essam/homebrew-core`, branch `app-dev-clean` (pushed, still there)
- Local checkout: `$(brew --repository homebrew/core)`, branch `app-dev-clean`
- Commit: `app-dev-clean 0.1.0 (new formula)` — clean subject, no trailers

It was auto-closed within a minute by `github-actions` with:

> Thanks for your pull request. This has been closed because it appears to use an
> incomplete or outdated pull request template. […] This workflow will reopen this
> pull request automatically once the template is complete. **Do not open a new
> pull request for this.**

So the fix is to **edit PR 296007's body** to the completed template. The bot
reopens it on its own. Opening a fresh PR would be the wrong move.

Withdrawn deliberately at this point pending the AI-disclosure decision below —
not because anything was wrong with the formula.

## What the first attempt got wrong

Four things, all now corrected in this document:

1. **homebrew-core's default branch is `main`, not `master`.**
   `git checkout -b app-dev-clean origin/master` fails outright with
   `fatal: 'origin/master' is not a commit`. Base the PR on `main` too.
2. **The audit flag is `--new`, not `--new-formula`.** The old name errors with
   `invalid option: --new-formula`. `--new` implies `--strict` and `--online`.
3. **The PR body must be the repo's template, filled in.** A hand-written
   description — however complete — gets auto-closed. Read
   `.github/PULL_REQUEST_TEMPLATE.md` from the checkout and fill that.
4. **`gh repo fork --remote=false` is invalid.** `--remote` is a boolean flag;
   use `gh repo fork Homebrew/homebrew-core --clone=false` and add the remote by
   hand.

## The AI-usage requirement — read before submitting

The template's last checkbox is not a formality:

> I did not use AI/LLM to create this PR, or I disclosed the tool/model below and
> reviewed its output; I did not attribute commits to AI and **will answer
> maintainer questions and review comments myself without AI/LLM**.

Per `docs.brew.sh/Responsible-AI-Usage`: disclose the tool, explain how the
changes were verified, personally review all AI-generated content *before* asking
a maintainer to look at it, and answer reviewer questions yourself. The template
adds that non-maintainers may have **only one AI-assisted PR open at a time**.

If this formula was drafted with AI assistance, the honest path is the disclosure
branch, not the "did not use AI" branch. Note the commit itself is compliant with
the "did not attribute commits to AI" clause — it carries no trailer (see the
commit convention in `CLAUDE.md`).

**The commitment to handle review personally can only be made by the submitter.**
That is the reason this PR is parked rather than pushed through.

## Submitting (or completing PR 296007)

The formula is validated, so this is template mechanics. Step 1 clones a 1.3 GB
repo — it is the slow part and only needed once.

```
brew tap --force homebrew/core
```
```
cd "$(brew --repository homebrew/core)"
```
```
git fetch origin && git checkout -b app-dev-clean origin/main
```

Write the formula to `Formula/a/app-dev-clean.rb` (core shards by first letter),
copying the ruby block from this document. Then run the gate exactly as the
template words it:

```
HOMEBREW_NO_INSTALL_FROM_API=1 brew install --build-from-source app-dev-clean
```
```
brew test app-dev-clean
```
```
brew audit --new app-dev-clean
```
```
brew style app-dev-clean
```

Note `brew audit --new` is slow — it makes network calls and took over ten
minutes here. Budget for that rather than assuming it hung.

Commit, fork, push:

```
git add Formula/a/app-dev-clean.rb
```
```
git commit -m "app-dev-clean 0.1.0 (new formula)"
```
```
gh repo fork Homebrew/homebrew-core --clone=false
```
```
git remote add fork https://github.com/latif-essam/homebrew-core.git
```
```
git push fork app-dev-clean
```

Then set the PR body to the completed template. To finish the existing PR:

```
gh pr edit 296007 --repo Homebrew/homebrew-core --body-file <your-filled-template.md>
```

The title must be exactly `app-dev-clean 0.1.0 (new formula)` — Homebrew's
automation parses it.

Afterwards, restore this machine, since building from source replaces the tap
install and drops the `adc` alias:

```
brew uninstall app-dev-clean
```
```
brew install latif-essam/tap/app-dev-clean
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
