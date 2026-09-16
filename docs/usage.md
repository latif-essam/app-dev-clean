# Command reference

[Back to the README](../README.md) · [First cleanup](quickstart.md)

These commands describe the current source. The published `v0.1.0` predates
the safety and reinstall changes; build from source to use them before the next release.

## Usage

```sh
app-dev-clean                           # Open the selection menu
app-dev-clean --root                    # Show the detected project root
app-dev-clean js --dry-run -y           # Preview dependency cleanup
app-dev-clean js -y                     # Remove node_modules; keep lockfiles
app-dev-clean js --reinstall -y         # Clean, then reinstall using the lockfile
app-dev-clean ios --reinstall           # Clean iOS output, then reinstall Pods
app-dev-clean local-all --dry-run -y    # Preview all project-local targets
app-dev-clean --type expo               # Limit the menu to Expo targets
app-dev-clean metro                     # Confirm cleanup of shared Metro caches
```

`-y` runs without prompts and does not imply reinstall. Add `--reinstall` when
needed. Without `-y`, dependency targets offer a reinstall prompt before cleanup.
The menu's `a` key selects project-local targets only.

Shared/global targets require confirmation, or explicit `--allow-shared` when
running unattended. For example:

```sh
app-dev-clean gradle-global --dry-run -y
app-dev-clean gradle-global -y --allow-shared
```

`nuclear` includes local targets, shared caches, available global caches, and
reinstallation. It requires a recognized project and the same shared-cache
consent. Prefer individual targets when diagnosing a problem: global cache
removal can require downloading dependencies again.

## Cleanup targets

| Project | Target | Removed or reset |
|---|---|---|
| React Native, Expo | `js` | `node_modules` |
| React Native, Expo | `metro` | Shared `metro-*`, `haste-map-*`, and `metro-cache` in the temporary directory |
| React Native, Expo | `android` | Native build output under `android/`, plus `gradlew clean` if present |
| React Native, Expo | `ios` | `ios/build`, `ios/Pods`, `ios/.build` |
| React Native | `watchman` | The current project's watch; skipped if Watchman is absent |
| Expo | `expo` | `.expo`, `.expo-shared` |
| Flutter | `flutter` | `build`, `.dart_tool`, plus `flutter clean` |
| Android | `android` | `build`, `app/build`, `.gradle`, `.cxx`, `app/.cxx`, plus `gradlew clean` if present |
| Xcode/SwiftPM | `ios` | `build`, `Pods`, `.build` |

Global targets are `gradle-global`, `xcode-dd`, `pods-cache`, and `pub-cache`.
Xcode and CocoaPods caches are offered only on macOS. Gradle and pub cache
locations respect `GRADLE_USER_HOME` and `PUB_CACHE`.

## Reinstallation

Lockfiles and package download caches are preserved during local cleanup.
The CLI checks `packageManager` and existing lockfiles before choosing an install:

| Package manager | Install command |
|---|---|
| npm | `npm ci --prefer-offline` |
| pnpm | `pnpm install --frozen-lockfile --prefer-offline` |
| Yarn 1 | `yarn install --frozen-lockfile --prefer-offline` |
| Yarn 2+ | `yarn install --immutable` |
| Bun | `bun install --frozen-lockfile` |
| CocoaPods | `pod install --deployment`, or `bundle exec pod install --deployment` when declared in a Gemfile |

These reuse the package manager's cache where possible. Missing packages may
still require network access; no remote cache service is set up automatically.
See the upstream [npm](https://docs.npmjs.com/cli/v11/commands/npm-ci/),
[pnpm](https://pnpm.io/cli/install), [Yarn](https://yarnpkg.com/cli/install), and
[Bun](https://bun.sh/docs/pm/cli/install) documentation for install behavior.

Conflicting lockfiles, missing install tools, and mismatched pinned tool versions
stop the operation before deletion. A lockfile is required for automatic
reinstall. JS cleanup in a shared workspace is refused; use the workspace's
package manager to repair its dependencies explicitly.

## Safety and limits

- Unknown options, invalid targets, and mismatched project types stop cleanup.
- Every selected cleanup path is checked before any selected target runs.
- Cache overrides reject home/project roots, common personal folders, and regular files.
- Direct deletion is confined to the project or selected cache scope. Escaping
  ancestor symlinks are rejected; leaf symlinks are unlinked without following them.
- Local cleanup refuses to delete Git-tracked files in its named output paths.
- Lockfiles, including `package-lock.json` and `Podfile.lock`, are not cleanup targets.
- `--dry-run` does not delete files or invoke cleanup/install commands.
- Command output and warnings stream to the terminal. A failure stops execution
  and returns a nonzero exit code.

This is a cleanup tool, not a backup system. Untracked files in selected output
folders are deleted. External tools such as Gradle, Flutter, and package-manager
scripts retain their own behavior and permissions. A failed reinstall can leave
dependencies missing; the CLI preserves lockfiles and reports the command to retry.
