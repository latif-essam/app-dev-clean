# Publishing app-dev-clean — runbook

Do this when you're ready to ship. Nothing here has been done yet.

## Accounts you need

**Only GitHub** — you already have it (`latif-essam`, and `gh` is authed on this
machine). No other signups:

- **Homebrew** and **Scoop** are NOT services you register with. They're just
  conventions over ordinary GitHub repos. A "tap" / "bucket" is a normal repo
  that holds a package definition file.
- **goreleaser** needs no account — it's a CLI that runs in GitHub Actions.
- **Cost: $0.** Public repos + GitHub Actions free tier cover everything.

## Repos to create (3, all public, under `latif-essam`)

| Repo | Purpose | Name is fixed? |
|---|---|---|
| `app-dev-clean` | the tool: source + GitHub Releases (prebuilt binaries) | free choice (this is the product name) |
| `homebrew-tap` | holds the auto-generated Homebrew **formula** | **yes** — Homebrew maps `brew tap latif-essam/tap` → repo `homebrew-tap`. Must start with `homebrew-`. |
| `scoop-bucket` | holds the auto-generated Scoop **manifest** (Windows) | conventional; any name works but `scoop-bucket` is standard |

**Why three?** goreleaser publishes to Homebrew/Scoop by *committing a package
file into a separate repo* (the tap / bucket) — not into the code repo. `brew`
and `scoop` then read those repos to find installable packages. The code repo
only hosts the source and the release archives.

## The one credential — a Personal Access Token (PAT)

**Why:** the GitHub Actions release job runs *inside* `app-dev-clean`. Its
built-in token can only write to that same repo. But the release must **push the
formula into `homebrew-tap` and the manifest into `scoop-bucket`** — different
repos. Cross-repo writes need a token with wider scope. This is the only step
nobody but you can do (minting credentials).

**How:**
1. GitHub → Settings → Developer settings → Personal access tokens →
   **Fine-grained tokens** → Generate new token.
2. **Repository access:** select `homebrew-tap` and `scoop-bucket` (or "All repos").
3. **Permissions:** Repository permissions → **Contents: Read and write**.
4. Generate, copy the token.
5. Store it as a secret in the code repo:
   `gh secret set HOMEBREW_TAP_GITHUB_TOKEN --repo latif-essam/app-dev-clean`
   (paste when prompted) — or via repo Settings → Secrets and variables →
   Actions → New repository secret, name `HOMEBREW_TAP_GITHUB_TOKEN`.

The workflow (`.github/workflows/release.yml`) already reads this secret name.

## Steps, in order (who does what, and why)

1. **Create repos** — *me, via `gh`.*
   `gh repo create latif-essam/app-dev-clean --public --source=. --remote=origin`
   plus `homebrew-tap` and `scoop-bucket` (`--public --add-readme`).
2. **Push + PR + merge** — *me* pushes `go-rewrite`, opens a PR; *you* review;
   merge to `main`; then remove the legacy bash tree (`bin/ lib/ apps/ Formula/
   tests/run.sh`) and the old plan doc. Keeps `main` clean, history preserved.
3. **Mint the PAT + set the secret** — *you only* (see above). Must exist before
   step 5 or Homebrew/Scoop publishing fails.
4. **Tag the release** — *me*: `git tag v0.1.0 && git push origin v0.1.0`.
   The tag push triggers `release.yml` → goreleaser: builds all 6 binaries,
   creates the GitHub Release with archives + checksums, commits the formula to
   `homebrew-tap`, the manifest to `scoop-bucket`.
5. **Verify** — install on each OS:
   - mac/Linux: `brew install latif-essam/tap/app-dev-clean && app-dev-clean --version`
   - Windows: `scoop bucket add latif-essam https://github.com/latif-essam/scoop-bucket; scoop install app-dev-clean`
   - anywhere: `go install github.com/latif-essam/app-dev-clean@latest`

## Minimal vs full publish

- **Minimal** (repo + tag, no PAT): you immediately get **GitHub Releases**, the
  **curl/irm install scripts**, and **`go install`**. These need nothing but the
  public repo + the release tag.
- **Full** (adds Homebrew + Scoop): also needs the `homebrew-tap` +
  `scoop-bucket` repos and the PAT secret.

So if you want to ship fast, minimal works on day one; add brew/scoop whenever.
