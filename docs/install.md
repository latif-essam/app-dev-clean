# Install app-dev-clean

[Back to the README](../README.md) · [First cleanup](quickstart.md)

The published release is currently **v0.1.0**. It predates the safety and reinstall
changes documented on `main`. Use a source build to try those changes today.

## Package managers

| System | Prerequisite | Install |
| --- | --- | --- |
| macOS | [Homebrew](https://brew.sh/) | `brew install latif-essam/tap/app-dev-clean` |
| Linux / WSL | [Homebrew on Linux](https://docs.brew.sh/Homebrew-on-Linux) | `brew install latif-essam/tap/app-dev-clean` |
| Windows | [Scoop](https://scoop.sh/) | Add the bucket, then install as below |

```powershell
scoop bucket add latif-essam https://github.com/latif-essam/scoop-bucket
scoop install app-dev-clean
app-dev-clean --version
```

The Homebrew tap and Scoop bucket are maintained by this project. Inclusion in
Homebrew's official repository is a separate process; see the
[submission status](homebrew-core.md).

Go users can install the latest **published** version:

```sh
go install github.com/latif-essam/app-dev-clean@latest
```

Go is only needed to compile or use `go install`, not to run a prebuilt binary.
Install Git for tracked-file checks when using the current source inside a Git
repository. Project tools such as npm, Flutter, Java/Gradle, or CocoaPods must be
installed separately for the operations that use them. The CLI does not install
SDKs or make macOS-only tools available on Windows/Linux.

## Build the current source

Requires [Go 1.24+](https://go.dev/doc/install) and Git. This checks out the current
development branch, not a stable release.

macOS / Linux:

```sh
git clone https://github.com/latif-essam/app-dev-clean.git
cd app-dev-clean
go build -o dist/dev/app-dev-clean .
./dist/dev/app-dev-clean --version
mkdir -p "$HOME/.local/bin"
install -m 0755 dist/dev/app-dev-clean "$HOME/.local/bin/app-dev-clean"
export PATH="$HOME/.local/bin:$PATH"
app-dev-clean --help
```

The install command replaces any existing `app-dev-clean` in `~/.local/bin`.
The `export` applies to this shell. Add that PATH entry to your shell configuration
if you want it in new terminals. This may take precedence over a Homebrew install;
use `command -v app-dev-clean` to check which binary you are running.

Windows PowerShell:

```powershell
git clone https://github.com/latif-essam/app-dev-clean.git
Set-Location app-dev-clean
go build -o dist/dev/app-dev-clean.exe .
.\dist\dev\app-dev-clean.exe --version
$adcBin = (Resolve-Path .\dist\dev).Path
$env:Path = "$adcBin;$env:Path"
app-dev-clean --help
```

That PATH change lasts for this PowerShell session. For a permanent setup, add
the resolved `dist\dev` folder to your **user** Path in Windows Environment
Variables, or copy the executable to a folder already on Path. Verify with
`Get-Command app-dev-clean`.

Now `cd` into your mobile project and start with `app-dev-clean --root` followed
by a dry-run. The source repository itself is not a supported mobile project.
The [tutorial](quickstart.md) gives examples for each project type.

## Manual installation

Open [Releases](https://github.com/latif-essam/app-dev-clean/releases/latest), then
download the archive matching your OS and CPU **and that release's `checksums.txt`**.

| System / CPU | Archive |
| --- | --- |
| macOS, Apple Silicon | `app-dev-clean_darwin_arm64.tar.gz` |
| macOS, Intel | `app-dev-clean_darwin_amd64.tar.gz` |
| Linux, ARM64 | `app-dev-clean_linux_arm64.tar.gz` |
| Linux, x86-64 | `app-dev-clean_linux_amd64.tar.gz` |
| Windows, ARM64 | `app-dev-clean_windows_arm64.zip` |
| Windows, x86-64 | `app-dev-clean_windows_amd64.zip` |

On macOS/Linux, `uname -m` reports the CPU architecture. `x86_64` maps to `amd64`;
`arm64` and `aarch64` map to `arm64`. On Windows, check **Settings → System → About → System type**.

Before extracting, compute the archive's SHA-256 and compare it with the exact
filename's entry in `checksums.txt`. For example, for Linux x86-64:

```sh
sha256sum app-dev-clean_linux_amd64.tar.gz
```

On macOS, use `shasum -a 256` with your archive filename. On Windows:

```powershell
Get-FileHash .\app-dev-clean_windows_amd64.zip -Algorithm SHA256
```

Only continue if the digest matches. Checksums check download integrity; obtain
both files from the intended release. Extract into a new folder, then put
`app-dev-clean` (or `app-dev-clean.exe`) on PATH. Example on Linux x86-64, from
the download folder:

```sh
mkdir app-dev-clean-release
tar -xzf app-dev-clean_linux_amd64.tar.gz -C app-dev-clean-release
mkdir -p "$HOME/.local/bin"
install -m 0755 app-dev-clean-release/app-dev-clean "$HOME/.local/bin/app-dev-clean"
export PATH="$HOME/.local/bin:$PATH"
app-dev-clean --version
```

On macOS, substitute the matching `darwin` archive. On Windows, extract the ZIP
with Explorer, put the executable in a dedicated folder, add that folder to your
user Path, and reopen the terminal. Raw archives do not create the `adc` alias.

## Update or uninstall

Use the same package manager you installed with:

| Route | Update | Uninstall |
| --- | --- | --- |
| Homebrew | `brew update`, then `brew upgrade latif-essam/tap/app-dev-clean` | `brew uninstall latif-essam/tap/app-dev-clean` |
| Scoop | `scoop update`, then `scoop update app-dev-clean` | `scoop uninstall app-dev-clean` |
| Go | Repeat `go install github.com/latif-essam/app-dev-clean@latest` | Remove the binary from your Go binary directory |
| Manual archive | Verify and replace with a newer release binary | Remove the binary you installed |
| Source build | Pull the reviewed source and rebuild | Remove the source-built binary / its PATH entry |

`brew upgrade` cannot install fixes that have not been released yet. Uninstalling
the CLI does not restore files removed by previous cleanup operations.

## Command not found, or wrong version?

1. Open a new terminal after changing PATH.
2. Check `command -v app-dev-clean` on macOS/Linux or `Get-Command app-dev-clean -All` on Windows.
3. Run `app-dev-clean --version` and `app-dev-clean --help`.
4. If a new flag is unavailable, check for a v0.1.0 binary earlier on PATH.

For Go installs, `go env GOBIN GOPATH` shows the configured locations. An empty
`GOBIN` means the executable normally lives in the `bin` directory under `GOPATH`.
