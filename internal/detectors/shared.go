package detectors

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isWindows() bool { return runtime.GOOS == "windows" }

// Base dirs for the native cleanups. The android/ios path builders are relative
// to the dir that actually holds the gradle project / Xcode project, which is
// NOT always the detected project root:
//
//	native Android or Xcode repo -> the root IS that dir      -> projectRoot
//	React Native / bare Expo     -> it is <root>/android, <root>/ios
//
// Passing projectRoot for RN made every native target resolve to a
// non-existent path and silently clean nothing.
func projectRoot(ctx detect.Context) string { return ctx.ProjectRoot }

func androidSubdir(ctx detect.Context) string {
	return filepath.Join(ctx.ProjectRoot, "android")
}

func iosSubdir(ctx detect.Context) string {
	return filepath.Join(ctx.ProjectRoot, "ios")
}

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
