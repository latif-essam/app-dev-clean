package detectors

import (
	"path/filepath"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

// expo embeds rn so it inherits rn's PostRun (npm/pod reinstall) behavior.
// Name/Detect/Targets below shadow the embedded rn's methods.
type expo struct{ rn }

func (expo) Name() string { return "expo" }

func (expo) Detect(dir string) bool { return pkgJSONHas(dir, `"expo"`) }

func expoDirPaths(root string) []string {
	return []string{filepath.Join(root, ".expo"), filepath.Join(root, ".expo-shared")}
}

func (expo) Targets() []detect.Target {
	t := []detect.Target{
		{Name: "expo", Label: "expo", Desc: ".expo/ + prebuild caches", Scope: detect.Local,
			Paths: func(c detect.Context) []string { return expoDirPaths(c.ProjectRoot) },
			Run: func(c detect.Context) (int64, error) {
				return clean.Remove(c.DryRun, expoDirPaths(c.ProjectRoot)...), nil
			}},
		{Name: "js", Label: "js", Desc: "node_modules + package-lock.json", Scope: detect.Local,
			Paths: func(c detect.Context) []string {
				return []string{filepath.Join(c.ProjectRoot, "node_modules"), filepath.Join(c.ProjectRoot, "package-lock.json")}
			}, Run: rnJS},
		{Name: "metro", Label: "metro", Desc: "Metro/Haste temp caches", Scope: detect.Local,
			Paths: func(c detect.Context) []string { return nil }, Run: rnMetro},
	}
	// Bare Expo also has native dirs -> include android/ios local cleanups.
	t = append(t,
		detect.Target{Name: "android", Label: "android", Desc: "android build + gradle caches", Scope: detect.Local,
			Paths: func(c detect.Context) []string { return androidLocalPaths(c.ProjectRoot) }, Run: androidLocal},
		detect.Target{Name: "ios", Label: "ios", Desc: "ios build, Pods, Podfile.lock", Scope: detect.Local,
			Paths: func(c detect.Context) []string { return iosLocalPaths(c.ProjectRoot) }, Run: iosLocal},
	)
	return t
}

func init() { detect.Register(expo{}) }
