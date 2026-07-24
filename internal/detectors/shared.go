package detectors

import (
	"os"
	"runtime"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isWindows() bool { return runtime.GOOS == "windows" }

// globalTarget builds a Global-scope target cleaning a single machine-wide
// cache path chosen by pick. If pick returns "", the target offers no path
// (e.g. macOS-only caches on Windows/Linux) and cleaning is a no-op.
func globalTarget(name, label, desc string, pick func(detect.Context) string) detect.Target {
	pathsFn := func(ctx detect.Context) []string {
		if p := pick(ctx); p != "" {
			return []string{p}
		}
		return nil
	}
	return detect.Target{
		Name:  name,
		Label: label,
		Desc:  desc,
		Scope: detect.Global,
		Paths: pathsFn,
		Run: func(ctx detect.Context) (int64, error) {
			return clean.Remove(ctx.DryRun, pathsFn(ctx)...), nil
		},
	}
}

func init() {
	detect.RegisterGlobal(globalTarget("gradle-global", "gradle cache", "~/.gradle/caches (all projects)",
		func(c detect.Context) string { return c.Paths.GradleCache }))
	detect.RegisterGlobal(globalTarget("xcode-dd", "xcode dd", "Xcode DerivedData (macOS)",
		func(c detect.Context) string { return c.Paths.XcodeDD }))
	detect.RegisterGlobal(globalTarget("pods-cache", "pods cache", "CocoaPods cache (macOS)",
		func(c detect.Context) string { return c.Paths.CocoaPods }))
	detect.RegisterGlobal(globalTarget("pub-cache", "pub cache", "Flutter pub cache",
		func(c detect.Context) string { return c.Paths.PubCache }))
}
