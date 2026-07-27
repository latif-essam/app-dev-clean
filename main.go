package main

import (
	"os"
	"runtime/debug"
	"strings"

	"github.com/latif-essam/app-dev-clean/internal/cli"
	_ "github.com/latif-essam/app-dev-clean/internal/detectors" // register detectors
)

// version is injected by goreleaser via -X main.version for release builds
// (brew, scoop, the install scripts, GitHub Release archives).
var version = "dev"

// resolveVersion reports the version to print for --version.
//
// `go install github.com/…/app-dev-clean@v0.1.0` passes no ldflags, so the
// injected value stays "dev" on that path — but Go records the module version in
// the binary, so read it from there instead. The ldflag wins when present.
//
// The two sources disagree on format: goreleaser's {{.Version}} is "0.1.0" while
// module versions are "v0.1.0", so the prefix is stripped to keep --version
// identical however the binary was installed.
func resolveVersion(injected string, readBuildInfo func() (*debug.BuildInfo, bool)) string {
	if injected != "" && injected != "dev" {
		return injected
	}
	fallback := injected
	if fallback == "" {
		fallback = "dev"
	}
	info, ok := readBuildInfo()
	if !ok || info == nil {
		return fallback
	}
	// "(devel)" is what a plain `go build` in a work tree records — no more
	// informative than "dev", so don't surface it.
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return strings.TrimPrefix(v, "v")
	}
	return fallback
}

func main() {
	os.Exit(cli.Run(os.Args[1:], resolveVersion(version, debug.ReadBuildInfo)))
}
