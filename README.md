# app-dev-clean ~ adc

Cross-platform dev-cache cleaner for mobile & native projects. Run it from
**anywhere inside a project** — it walks up to the real project root, detects the
project type(s), and cleans the right caches. Refuses to touch anything if you're
not inside a recognized project.

Supports **React Native, Expo, Flutter, native Android (Gradle), and native
iOS/macOS (Xcode/SwiftPM)** on **Windows, macOS, and Linux**.

## Install

**Homebrew (macOS/Linux):**

```bash
brew install latif-essam/tap/app-dev-clean
```

No separate `brew tap` needed — the `owner/tap/formula` form taps automatically.

**Scoop (Windows):**

```powershell
scoop bucket add latif-essam https://github.com/latif-essam/scoop-bucket
scoop install app-dev-clean
```

**Install script (macOS/Linux):**

```bash
curl -fsSL https://raw.githubusercontent.com/latif-essam/app-dev-clean/main/install.sh | bash
```

**Install script (Windows PowerShell):**

```powershell
irm https://raw.githubusercontent.com/latif-essam/app-dev-clean/main/install.ps1 | iex
```

**Go:**

```bash
go install github.com/latif-essam/app-dev-clean@latest
```

Or grab a prebuilt binary from [Releases](https://github.com/latif-essam/app-dev-clean/releases).
An `adc` alias is installed alongside the full name.

## Usage

```bash
app-dev-clean                interactive menu (inside a known project)
app-dev-clean ios js         run named targets, no prompt
app-dev-clean local-all      all local targets for the detected type(s)
app-dev-clean nuclear        local-all + global caches + reinstall (confirmed)
app-dev-clean --type flutter scope to one detector
app-dev-clean --dry-run      show what would be freed; delete nothing
app-dev-clean -y             skip confirmation prompts (CI)
app-dev-clean --root         print resolved root + detected type(s)
adc ios js                   same, via the short alias
```

## What it cleans

Local targets, by detected project type. A project can match several types at
once (a bare Expo app is both `expo` and `rn`); you get the deduped union.

| Type | Target | Paths |
|---|---|---|
| `rn`, `expo` | `js` | `node_modules`, `package-lock.json` |
| `rn`, `expo` | `metro` | `metro-*`, `haste-map-*`, `metro-cache` in the OS temp dir |
| `rn`, `expo` | `android` | `android/build`, `android/app/build`, `android/.gradle`, `android/.cxx`, `android/app/.cxx` + `android/gradlew clean` |
| `rn`, `expo` | `ios` | `ios/build`, `ios/Pods`, `ios/Podfile.lock` |
| `rn` | `watchman` | resets a stale watch (no deletion) |
| `expo` | `expo` | `.expo`, `.expo-shared` |
| `flutter` | `flutter` | `build/`, `.dart_tool/` + `flutter clean` |
| `android` | `android` | `build/`, `app/build`, `.gradle`, `.cxx` + `gradlew clean` |
| `ios` | `ios` | `build/`, `Pods`, `Podfile.lock`, SwiftPM `.build` |

Native paths are resolved relative to the dir that actually holds the build — the
project root for a native Gradle/Xcode repo, `android/` and `ios/` for RN and Expo.

Global targets (machine-wide, always confirmed): `gradle-global`
(`~/.gradle/caches`), `xcode-dd` (Xcode DerivedData, macOS), `pods-cache`
(CocoaPods, macOS), `pub-cache` (Flutter pub cache).

## Safety

- Local targets only ever act on the resolved project root — never your cwd blindly.
- Outside a recognized project, local cleanup is refused (non-zero exit).
- Global caches (`~/.gradle`, Xcode DerivedData, CocoaPods, pub cache) always
  prompt before deletion because they affect every project on the machine.

## Adding a detector

Create `internal/detectors/<type>.go` implementing `detect.Detector`
(`Name`, `Detect(dir)`, `Targets()`) and call `detect.Register(...)` in `init()`.
Detection, the menu, and CLI dispatch pick it up automatically. See existing
detectors for the pattern.

## License

MIT © Latif Essam
