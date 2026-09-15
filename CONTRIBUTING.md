# Contributing

Keep changes focused and include a regression test for fixes involving cleanup,
path handling, command execution, or package-manager selection.

## Checks

```sh
gofmt -w main.go main_test.go e2e_test.go safety_test.go internal/
go test ./...
go vet ./...
go build -o dist/dev/app-dev-clean .
```

CI runs tests on macOS, Linux, and Windows. Test fixtures must stay in temporary
directories. Never exercise destructive targets against a real project or a
machine's shared cache during tests.

## Architecture

`main` calls `internal/cli`, which resolves projects through `internal/detect`.
Detectors in `internal/detectors` provide cleanup paths and commands.
`internal/clean` validates scopes and performs removal and command execution.
`internal/reinstall` selects locked install commands. `internal/platform` locates
cache directories, and `internal/ui` renders the selection menu.

Register new detectors with `detect.Register`. Targets must list their cleanup
paths for preflight and use `clean.Remove` for direct deletion. Return errors
from required commands. Preserve dry-run behavior and dependency lockfiles.
Native target paths are relative to the native project directory, which may be
`android/` or `ios/` beneath the detected mobile app root.

Use Conventional Commits for this repository. Keep commit messages focused on
the change and its verification. Follow the destination project's commit format
when contributing packaging changes elsewhere.

## Releases

See [PUBLISHING.md](PUBLISHING.md). Keep published tags and release assets
immutable. Package-manager submissions have separate requirements; see
[Homebrew submission notes](docs/homebrew-core.md).
