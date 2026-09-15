# app-dev-clean

Clean development caches for React Native, Expo, Flutter, Android, and
Xcode/SwiftPM projects on macOS, Linux, and Windows. Run from anywhere inside a
supported project; the CLI locates its root and shows the selected cleanup work.

## Install

Homebrew on macOS or Linux, using the project's tap:

```sh
brew install latif-essam/tap/app-dev-clean
```

Scoop on Windows:

```powershell
scoop bucket add latif-essam https://github.com/latif-essam/scoop-bucket
scoop install app-dev-clean
```

With Go:

```sh
go install github.com/latif-essam/app-dev-clean@latest
```

Prebuilt binaries are available on [GitHub Releases](https://github.com/latif-essam/app-dev-clean/releases).
Homebrew and Scoop also install the `adc` alias. The project uses its own tap;
it is not included in `homebrew/core`.

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

## Development

Requires Go 1.24 or newer.

```sh
go test ./...
go vet ./...
go build -o dist/dev/app-dev-clean .
```

The commands documented here describe the current source. Published packages
receive these changes when a new version is released.

## License

[MIT](LICENSE) © Latif Essam
