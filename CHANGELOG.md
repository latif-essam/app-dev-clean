# Changelog

## Unreleased

- Stream cleanup and install output, including warnings, to the terminal.
- Reject unknown options, invalid targets, and mismatched project types before cleanup.
- Validate all selected paths before deletion and confine direct removal to its scope.
- Refuse cleanup of Git-tracked files in named project output directories.
- Preserve dependency lockfiles and download caches during local cleanup.
- Plan reinstall once per target, with JavaScript dependencies installed before Pods.
- Detect npm, pnpm, Yarn, and Bun; use locked installs and check tool versions first.
- Refuse shared-workspace JS resets and ambiguous package-manager configurations.
- Resolve native subdirectories to their owning mobile project.
- Exclude Metro from local-only selections and require explicit shared-cache consent.
- Stop and return a nonzero status when a cleanup or reinstall command fails.
- Support Windows batch wrappers and package-manager shims explicitly.
- Use the Go version declared in go.mod for CI and releases.
