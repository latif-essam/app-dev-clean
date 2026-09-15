package detectors

import (
	"path/filepath"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

type expo struct{}

func (expo) Name() string { return "expo" }

func (expo) Detect(dir string) bool { return pkgJSONHas(dir, "expo") }

func expoDirPaths(root string) []string {
	return []string{filepath.Join(root, ".expo"), filepath.Join(root, ".expo-shared")}
}

func (expo) Targets() []detect.Target {
	t := []detect.Target{
		{Name: "expo", Label: "expo", Desc: ".expo/ + prebuild caches", Scope: detect.Local,
			Paths: func(c detect.Context) []string { return expoDirPaths(c.ProjectRoot) },
			Run: func(c detect.Context) (int64, error) {
				return clean.Remove(c.DryRun, c.ProjectRoot, expoDirPaths(c.ProjectRoot)...)
			}},
		jsTarget(), metroTarget(),
	}
	// Bare Expo also has native dirs (post-prebuild) -> include android/ios
	// local cleanups, rooted at those subdirs exactly as for RN. In a managed
	// project the dirs are absent and clean.Remove skips them.
	t = append(t,
		androidTarget(androidSubdir, "android/ build, app/build, .gradle, .cxx + gradlew clean"),
		iosTarget(iosSubdir, "ios/ build, Pods, .build (keep Podfile.lock)"),
	)
	return t
}

func init() { detect.Register(expo{}) }
