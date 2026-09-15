package detectors

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

func pkgJSONHas(dir, name string) bool {
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return false
	}
	var pkg struct{ Dependencies, DevDependencies, PeerDependencies map[string]json.RawMessage }
	if json.Unmarshal(b, &pkg) != nil {
		return false
	}
	for _, deps := range []map[string]json.RawMessage{pkg.Dependencies, pkg.DevDependencies, pkg.PeerDependencies} {
		if _, ok := deps[name]; ok {
			return true
		}
	}
	return false
}

type rn struct{}

func (rn) Name() string { return "rn" }
func (rn) Detect(dir string) bool {
	return pkgJSONHas(dir, "react-native") && (exists(filepath.Join(dir, "android")) || exists(filepath.Join(dir, "ios")))
}

func jsTarget() detect.Target {
	paths := func(c detect.Context) []string { return []string{filepath.Join(c.ProjectRoot, "node_modules")} }
	return detect.Target{Name: "js", Label: "js", Desc: "node_modules (keep lockfiles and download caches)", Scope: detect.Local, Paths: paths,
		Run: func(c detect.Context) (int64, error) { return clean.Remove(c.DryRun, c.ProjectRoot, paths(c)...) }}
}

func metroPaths(c detect.Context) []string {
	if c.Paths.TmpDir == "" {
		return nil
	}
	var paths []string
	for _, pat := range []string{"metro-*", "haste-map-*", "metro-cache"} {
		matches, _ := filepath.Glob(filepath.Join(c.Paths.TmpDir, pat))
		paths = append(paths, matches...)
	}
	return paths
}

func metroTarget() detect.Target {
	return detect.Target{Name: "metro", Label: "metro", Desc: "Metro/Haste temp caches (all projects)", Scope: detect.Shared, Paths: metroPaths,
		Check: func(c detect.Context) error {
			if !filepath.IsAbs(c.Paths.TmpDir) {
				return fmt.Errorf("Metro requires an absolute temporary directory")
			}
			return nil
		},
		Run: func(c detect.Context) (int64, error) { return clean.Remove(c.DryRun, c.Paths.TmpDir, metroPaths(c)...) }}
}

func rnWatchman(ctx detect.Context) (int64, error) {
	if !ctx.DryRun {
		if _, err := exec.LookPath("watchman"); err != nil {
			fmt.Println("  watchman unavailable; skipping optional watch reset")
			return 0, nil
		}
	}
	if err := clean.Exec(ctx.DryRun, ctx.ProjectRoot, "watchman", "watch-del", ctx.ProjectRoot); err != nil {
		return 0, err
	}
	return 0, clean.Exec(ctx.DryRun, ctx.ProjectRoot, "watchman", "watch-project", ctx.ProjectRoot)
}

func (rn) Targets() []detect.Target {
	return []detect.Target{
		androidTarget(androidSubdir, "android/ build, app/build, .gradle, .cxx + gradlew clean"),
		iosTarget(iosSubdir, "ios/ build, Pods, .build (keep Podfile.lock)"),
		jsTarget(), metroTarget(),
		{Name: "watchman", Label: "watchman", Desc: "reset this project's watch", Scope: detect.Local, Run: rnWatchman},
	}
}
func init() { detect.Register(rn{}) }
