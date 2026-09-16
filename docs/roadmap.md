# Next steps

[Back to the README](../README.md)

These are planned work items, not released features or promised dates.
The immediate goal is a tested release of the safety improvements already on
`main`. Feedback from real projects should guide the next features.

## Release the current improvements

- [ ] Review the unreleased changes and choose the next version.
- [ ] Finish packaging readiness, including Git runtime availability and the GoReleaser publisher warning.
- [ ] Run release checks and generate a local release snapshot.
- [ ] Publish a new immutable tag, archives, and checksums.
- [ ] Verify the Homebrew tap and Scoop bucket point to that release.
- [ ] Test fresh installs and upgrades on macOS, Linux, and Windows.
- [ ] Replace the README's prerelease notice and regenerate the demo against the released binary.

Follow [PUBLISHING.md](../PUBLISHING.md) for the release procedure. Official
Homebrew inclusion also has [eligibility and submission requirements](homebrew-core.md);
distribution through the project's tap can proceed independently.

## Make first use easier

- [x] Add a preview-first README, platform installation guide, and text tutorial.
- [x] Capture a small CLI walkthrough with a readable still-image alternative.
- [ ] Record a real-project walkthrough after the release, with visible install logs.
- [ ] Collect reports from React Native, Expo, Flutter, and native-project users.
- [ ] Add a concise issue template for environment details and reproducible failures.

## Investigate after feedback

- [ ] Safer workspace-aware JS cleanup with explicit ownership boundaries.
- [ ] A machine-readable cleanup plan for scripts and CI.
- [ ] Better guidance when an install fails after cleanup.
- [ ] Document package-manager cache configuration for CI before considering any remote-cache integration.

Remote caching is not currently implemented. Any future integration should keep
credentials, private packages, cache ownership, and retention explicit. Reusing
existing local download caches remains the current behavior.

## Ways to help

- Try a dry-run in a disposable clone and check the selected paths.
- Report your OS, CLI version, project type, package manager, and expected behavior.
- Test a fresh installation or upgrade on a machine you can spare.
- Suggest a narrowly scoped improvement with a real example.

Open an [issue](https://github.com/latif-essam/app-dev-clean/issues) or read
[CONTRIBUTING.md](../CONTRIBUTING.md). Remove credentials and private paths from logs.
