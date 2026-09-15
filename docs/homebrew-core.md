# Homebrew core submission

Status checked on 2026-09-15. Distribution through
`latif-essam/tap/app-dev-clean` works independently of inclusion in `homebrew/core`.

## Existing PR

[PR #296007](https://github.com/Homebrew/homebrew-core/pull/296007) is closed.
The recorded closure reason is an incomplete or outdated PR template. There are
no maintainer reviews rejecting the source because of how it was written.
The closure message says to complete the existing PR, not open a replacement.

The old PR body incorrectly reports that every new-formula audit passed. Earlier
project notes establish that the successful new-formula audit ran in a separate
tap; the core audit failed its notability check. Correct this claim before
resubmission and rerun checks against the actual core formula.

## Eligibility blocker

The repository currently has 0 stars, 0 forks, and 0 watchers. Homebrew's
[package acceptance policy](https://docs.brew.sh/Package-Acceptance-Policy#notability)
normally requires 30 forks, 30 watchers, or 75 stars. For an owner submitting
their own repository, the thresholds are **90 forks, 90 watchers, or 225 stars**.
Exceptions are at Homebrew's discretion. The project does not currently satisfy
these thresholds; source cleanup and passing tests do not remove this blocker.

## Formula

[`packaging/homebrew/app-dev-clean.rb`](../packaging/homebrew/app-dev-clean.rb)
retains the source-build formula from the existing submission. It references
published `v0.1.0` and its existing checksum. It does **not** include the current
unreleased safety fixes. Update its version and checksum only after a new stable
release contains those fixes.

The core formula must build from versioned source with a verified checksum,
declare its build and runtime dependencies, and test real behavior. Keep its
single `app-dev-clean` executable name. The project's separate binary-distribution
tap also installs the optional `adc` alias.

Before packaging the next release, declare Git as a runtime dependency on
platforms that do not supply it: tracked-file protection uses Git when cleaning
a repository. Native build tools and package managers are selected by the user's
project and checked before they are invoked.

## Submission checks

Use Homebrew's current [contribution guide](https://github.com/Homebrew/homebrew-core/blob/main/CONTRIBUTING.md),
[formula cookbook](https://docs.brew.sh/Formula-Cookbook),
[acceptance rules](https://docs.brew.sh/Acceptable-Formulae), and exact
[PR template](https://github.com/Homebrew/homebrew-core/blob/main/.github/PULL_REQUEST_TEMPLATE.md).

Before reopening the existing PR:

- Resolve the eligibility blocker or obtain a documented exception.
- Publish the fixes under a new immutable release tag.
- Update the source URL/checksum and dependencies in the core formula.
- Build with `HOMEBREW_NO_INSTALL_FROM_API=1 brew install --build-from-source app-dev-clean`.
- Run `brew test app-dev-clean`, `brew audit --new app-dev-clean`, and
  `brew style app-dev-clean` in the core checkout. A separate tap's audit does
  not establish core eligibility.
- Use one focused formula commit, titled `app-dev-clean VERSION (new formula)`.
- Fill the current PR template without checking unperformed actions.

Homebrew's [disclosure and review requirements](https://docs.brew.sh/Responsible-AI-Usage)
apply even after removing tool attribution from commits. Disclose assistance in
the PR description, personally review the contribution before asking maintainers
to review it, and answer maintainer questions directly. No tool may be listed as
a commit author, co-author, committer, or signatory. Non-maintainers may have only
one assisted PR open at a time.

## History and releases

Development history may be cleaned up with a verified backup. Preserve human
authorship and do not rewrite published release tags or assets: Homebrew requires
immutable sources, and changing release content can invalidate cached checksums.
Historical published releases therefore retain their original source history.
