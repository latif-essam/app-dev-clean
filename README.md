# app-dev-clean

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
