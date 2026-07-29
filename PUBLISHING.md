# Publishing app-dev-clean — runbook

## Where things stand (2026-07-29)

**v0.1.0 is released.** `brew install latif-essam/tap/app-dev-clean` works.

| Step | Status |
|---|---|
| Three repos created, all public under `latif-essam` | ✅ |
| Go rewrite merged to `main` (PR #1, `c1f7ad7`) | ✅ |
| Legacy bash tree deleted | ✅ |
| CI green on ubuntu + macOS + Windows | ✅ |
| `HOMEBREW_TAP_GITHUB_TOKEN` Actions secret set | ✅ 2026-07-29 |
| Repo descriptions + topics (incl. `scoop-bucket` topic) | ✅ |
| Tag `v0.1.0` pushed → release workflow succeeded | ✅ run 30489056200 |
| GitHub Release: 6 archives + `checksums.txt` | ✅ |
| Formula committed to `homebrew-tap` | ✅ |
| Manifest committed to `scoop-bucket` | ✅ |
| Verified `brew install` + `adc` alias on macOS arm64 | ✅ reports `0.1.0` |
| Verified `go install …@v0.1.0` | ✅ reports `0.1.0` |
| Verify Scoop install + `adc` shim on real Windows | ⬜ **unverified** |
| homebrew-core submission (Tier 2) | ⬜ see `docs/homebrew-core.md` (formula ready, sha256 filled) |
| winget packaging | ⬜ deferred |

Two things observed during the first release that aren't in the automated checks:

- **`go install …@latest` lags the tag.** `proxy.golang.org` caches the module's
  version list, so immediately after tagging, `@latest` still resolves to the
  previous pseudo-version while `@v0.1.0` works. It corrects itself once the
  proxy indexes the tag. Don't chase this as a bug.
- **Homebrew now asks users to trust third-party taps.** Installing from our tap
  surfaces `brew trust` guidance (see docs.brew.sh/Tap-Trust). It's a real
  friction point for Tier 1 that homebrew-core wouldn't have.

## Cost and accounts

**GitHub only.** Nothing else to sign up for, and **$0** — public repos plus the
Actions free tier cover all of it.

Homebrew and Scoop are not services you register with. They're conventions over
ordinary git repos: a "tap" or "bucket" is a normal GitHub repo holding one
package-definition file. goreleaser is a CLI, not a service.

## The three repos

| Repo | Holds | Name fixed? |
|---|---|---|
| `app-dev-clean` | source + GitHub Releases (prebuilt binaries) | free choice — the product name |
| `homebrew-tap` | the generated Homebrew **formula** (`.rb`) | **yes** — `brew tap latif-essam/tap` maps to repo `homebrew-tap`; must start with `homebrew-` |
| `scoop-bucket` | the generated Scoop **manifest** (`.json`, Windows) | conventional, any name works, but this is standard |

**Why three?** goreleaser publishes to Homebrew and Scoop by *committing a
package file into a separate repo*, not into the code repo. `brew` and `scoop`
then read those repos to find installable packages. The code repo only hosts
source and release archives.

## The one credential — the PAT

Already done, kept here because you'll need it again when the token expires.

### What a PAT is

A **Personal Access Token** is a password substitute for automation: a long
random string GitHub recognises as "acting for Latif, but only for these specific
things." Three properties matter:

- **Scoped** — you choose which repos it touches and what it may do. A token that
  can write files to `homebrew-tap` cannot read your private code or delete repos.
- **Revocable** — if it leaks you delete that one token; the rest of your account
  is untouched. Unlike a password leak.
- **Expiring** — you set a lifetime; after that it's dead whether or not anyone
  noticed.

**Fine-grained** tokens (what we use) name individual repos and permissions.
**Classic** tokens use coarse scopes where `repo` means full control of *every*
repository you own, forever. Prefer fine-grained.

### Why the release needs one

GitHub Actions automatically hands each workflow run a token called
`GITHUB_TOKEN` — already referenced at `release.yml:20`, you never create it. It
is deliberately limited to **the repo the workflow lives in**, so a compromised
workflow can't pivot across your account.

Our release has to write to two *other* repos:

```
app-dev-clean  (workflow runs here)
  ├── create the GitHub Release            ✅ GITHUB_TOKEN can do this
  ├── commit app-dev-clean.rb   → homebrew-tap    ❌ different repo
  └── commit app-dev-clean.json → scoop-bucket    ❌ different repo
```

That cross-repo write is the whole reason a second credential exists. It's why
`.goreleaser.yaml:37` and `:49` read `{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}`.

Minting it is an account action, so it's the one step only the repo owner can do.

### Re-minting it (when it expires)

The release job will fail with an auth error against the tap. Then:

1. **github.com/settings/personal-access-tokens/new** (avatar → Settings →
   Developer settings → Personal access tokens → **Fine-grained tokens**)
2. **Name:** `app-dev-clean-release`. **Expiration:** 90 days is reasonable.
3. **Repository access:** *Only select repositories* → `homebrew-tap` **and**
   `scoop-bucket`. Do **not** add `app-dev-clean` — `GITHUB_TOKEN` already covers
   it, and including it widens the token for no gain.
4. **Permissions:** *Repository permissions* → **Contents: Read and write**.
   Leave everything else at "No access".
5. Generate and copy — GitHub shows it once.
6. Store it:
   ```bash
   gh secret set HOMEBREW_TAP_GITHUB_TOKEN --repo latif-essam/app-dev-clean
   ```
   Paste when prompted. The name must match exactly; `release.yml:21` reads that
   string. Actions secrets are encrypted and cannot be read back, only overwritten.

Blast radius if it leaks: someone commits junk to two package-definition repos.
Revoke the token, force-push the repos, done.

## Cutting a release

Everything below is automatic once the tag lands. Same procedure for every future
version — only the number changes.

```bash
git checkout main && git pull
go test ./... && go vet ./... && gofmt -l main.go main_test.go e2e_test.go internal/
git tag v0.1.0
git push origin v0.1.0
```

The tag push triggers `.github/workflows/release.yml` → goreleaser, which:

1. builds all 6 binaries (linux/darwin/windows × amd64/arm64)
2. creates the GitHub Release with archives + `checksums.txt`
3. commits the formula to `latif-essam/homebrew-tap`
4. commits the manifest to `latif-essam/scoop-bucket`

Watch it:

```bash
gh run watch --repo latif-essam/app-dev-clean
```

**Tag only a commit whose CI is already green.** The release workflow does not
re-run the test matrix — it builds and publishes. A red `main` will still release.

## Verifying a release

```bash
# macOS / Linux — no separate `brew tap` needed, owner/tap/formula auto-taps
brew install latif-essam/tap/app-dev-clean
app-dev-clean --version && adc --version

# anywhere with Go
go install github.com/latif-essam/app-dev-clean@latest

# Windows
scoop bucket add latif-essam https://github.com/latif-essam/scoop-bucket
scoop install app-dev-clean
app-dev-clean --version; adc --version
```

Two gotchas that have already bitten:

- **`go install` puts the binary in `$(go env GOPATH)/bin`**, which is often not
  on `PATH` — the install succeeds and the command still isn't found. Check with
  `ls $(go env GOPATH)/bin` before concluding anything is broken.
- **A dir earlier on `PATH` shadows Homebrew.** `~/.local/bin` precedes
  `/opt/homebrew/bin`, so a symlink left there wins over `brew install`. Remove
  stale symlinks before testing a brew install, or you'll test the wrong binary.

**Still unverified on real hardware:** the Scoop `adc` alias. Scoop manifests
have no alias field, so it's created by a `post_install` block
(`.goreleaser.yaml:57-59`), written to be non-fatal — if it fails the alias is
simply absent and `scoop install` still succeeds. Confirm `adc --version` on a
real Windows machine. Fallbacks if absent: `scoop alias`, or the `install.ps1`
path, which writes `adc.cmd` directly.

## If a release goes wrong

Releases are recoverable, but the tag and the GitHub Release are separate objects
and **both** must go:

```bash
gh release delete v0.1.0 --repo latif-essam/app-dev-clean --yes
git push origin :refs/tags/v0.1.0
git tag -d v0.1.0
```

If goreleaser got far enough to commit to the tap or bucket, revert there too —
otherwise `brew` serves a formula pointing at a release that no longer exists.
Then fix, re-tag, re-push.

Common failures:

| Symptom | Cause |
|---|---|
| release job fails at the brews/scoops stage with a 401/403 | PAT expired or missing Contents:RW |
| GitHub Release exists but tap is empty | job died mid-run — clean up per above before re-tagging |
| `--version` prints `dev` | binary built without the ldflag *and* without module info; `go install` from a tag is fine (see `resolveVersion` in `main.go`) |
| goreleaser warns about `brews` being deprecated | expected and deliberate — see `.goreleaser.yaml` comments |

## Reach: own tap vs. being "in Homebrew"

These are different, and worth not conflating.

**Tier 1 — our own tap (what this runbook delivers).**
`brew install latif-essam/tap/app-dev-clean` genuinely works. But it is **not
discoverable**: `brew search` only covers homebrew-core plus taps the user has
already added. You have to hand people the command.

**Tier 2 — homebrew-core.** Plain `brew install app-dev-clean`, plus a
formulae.brew.sh page that Google indexes. Needs a PR to
`Homebrew/homebrew-core`, and a core formula **builds from source** on their CI —
so it's a hand-written formula, not goreleaser's output. Eligibility check,
ready formula, and submission checklist are in **`docs/homebrew-core.md`**.

Scoop has the same split: our bucket vs. `ScoopInstaller/Main`.

Google discoverability is a third, separate thing, driven by the repo description
and topics (already set), the README, and the auto-generated
`pkg.go.dev/github.com/latif-essam/app-dev-clean` page — not by Homebrew.

## Deferred

- **winget** (Windows Package Manager) packaging — not started.
- **Code signing.** The macOS binary is unsigned, which is why `.goreleaser.yaml`
  uses `brews` (a formula) rather than `homebrew_casks`: casks quarantine
  unsigned binaries under Gatekeeper, while formulae install clean and support
  the `adc` symlink. Revisit the cask question only if the binary gets signed.
