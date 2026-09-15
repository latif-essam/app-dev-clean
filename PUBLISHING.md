# Publishing

The source repository is `latif-essam/app-dev-clean`. Releases contain binaries
for macOS, Linux, and Windows on amd64 and arm64, plus checksums. The release
workflow also updates `latif-essam/homebrew-tap` and `latif-essam/scoop-bucket`.

## Release checklist

1. Review the changes and update `CHANGELOG.md`.
2. Run formatting, tests, vet, and cross-compilation for all supported targets.
3. Run `goreleaser check` and inspect a local release snapshot.
4. Push the reviewed source commit and wait for the full CI matrix to pass.
5. Create a new semantic-version tag and push that tag to trigger the release
   workflow. Do not move or reuse a published tag.
6. Verify the release archives and checksums, then confirm the tap and bucket
   reference the new version.
7. Install and exercise the published packages on the supported systems.

A source commit alone does not update installed Homebrew or Scoop packages.
Publish a new release for users to receive source changes.

The current GoReleaser `brews` publisher remains compatible but is deprecated;
`goreleaser check` reports that existing warning as a nonzero result. Its formula
is for the project tap only. Review publisher migration before upgrading the
release tool; do not substitute a binary cask for the core source formula.

## Credentials

`GITHUB_TOKEN` publishes the release in this repository.
`HOMEBREW_TAP_GITHUB_TOKEN` must be a fine-grained token with Contents read/write
access to `homebrew-tap` and `scoop-bucket`. Keep it in Actions secrets and renew
it before expiry. Do not commit tokens or copy them into logs.

The release configuration uses the maintainer's name and public GitHub email for
packaging commits. Homebrew's official repository uses a separate source-build
formula; the generated tap formula is not a core submission.

## Failed releases

Inspect the workflow failure before retrying. If a version has already been
published, fix it in a new version rather than replacing its sources, archives,
or checksums. Existing consumers may have cached the original release.

For command lookup problems, inspect `command -v app-dev-clean` (or
`Get-Command app-dev-clean` in PowerShell) for an older install earlier in PATH.

## Official package repositories

The project currently distributes through its own Homebrew tap and Scoop bucket.
Official repository inclusion requires a separate accepted submission. See
[Homebrew requirements and current blockers](docs/homebrew-core.md).
