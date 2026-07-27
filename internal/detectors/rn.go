package detectors

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

// pkgJSONHas reports whether dir/package.json contains needle (substring match,
// same coarse check the bash version used via grep).
func pkgJSONHas(dir, needle string) bool {
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return false
	}
	return strings.Contains(string(b), needle)
}

type rn struct{}

func (rn) Name() string { return "rn" }

func (rn) Detect(dir string) bool {
	if !pkgJSONHas(dir, `"react-native"`) {
		return false
	}
	return exists(filepath.Join(dir, "android")) || exists(filepath.Join(dir, "ios"))
}

func rnJS(ctx detect.Context) (int64, error) {
	return clean.Remove(ctx.DryRun,
		filepath.Join(ctx.ProjectRoot, "node_modules"),
		filepath.Join(ctx.ProjectRoot, "package-lock.json"),
	), nil
}

func rnMetro(ctx detect.Context) (int64, error) {
	// Metro/Haste temp caches live in the OS temp dir as metro-* / haste-map-*.
	var freed int64
	patterns := []string{"metro-*", "haste-map-*", "metro-cache"}
	for _, pat := range patterns {
		matches, _ := filepath.Glob(filepath.Join(ctx.Paths.TmpDir, pat))
		freed += clean.Remove(ctx.DryRun, matches...)
	}
	return freed, nil
}

func rnWatchman(ctx detect.Context) (int64, error) {
	// Reset stale watch; non-destructive, skipped in dry-run.
	if !ctx.DryRun {
		clean.Exec(false, ctx.ProjectRoot, "watchman", "watch-del", ctx.ProjectRoot)
		clean.Exec(false, ctx.ProjectRoot, "watchman", "watch-project", ctx.ProjectRoot)
	}
	return 0, nil
}

func (rn) Targets() []detect.Target {
	return []detect.Target{
		// Native output lives under android/ and ios/, not at the project root.
		androidTarget(androidSubdir, "android/ build, app/build, .gradle, .cxx + gradlew clean"),
		iosTarget(iosSubdir, "ios/ build, Pods, Podfile.lock"),
		{Name: "js", Label: "js", Desc: "node_modules + package-lock.json", Scope: detect.Local,
			Paths: func(c detect.Context) []string {
				return []string{filepath.Join(c.ProjectRoot, "node_modules"), filepath.Join(c.ProjectRoot, "package-lock.json")}
			}, Run: rnJS},
		{Name: "metro", Label: "metro", Desc: "Metro/Haste temp caches", Scope: detect.Local,
			Paths: func(c detect.Context) []string { return nil }, Run: rnMetro},
		{Name: "watchman", Label: "watchman", Desc: "reset stale watch", Scope: detect.Local,
			Paths: func(c detect.Context) []string { return nil }, Run: rnWatchman},
	}
}

// PostRun handles reinstall after a js/ios clean:
//   - dry-run: never reinstall
//   - Force (nuclear): reinstall unconditionally, no prompt
//   - Yes (non-interactive, non-nuclear): skip reinstall
//   - otherwise (interactive): prompt per action
func (rn) PostRun(ctx detect.Context, ran []string) error {
	if ctx.DryRun {
		return nil
	}
	doJS := containsStr(ran, "js")
	doIOS := containsStr(ran, "ios")
	if ctx.Force {
		if doJS {
			clean.Exec(false, ctx.ProjectRoot, "npm", "install")
		}
		if doIOS {
			clean.Exec(false, filepath.Join(ctx.ProjectRoot, "ios"), "pod", "install", "--repo-update")
		}
		return nil
	}
	if ctx.Yes {
		return nil
	}
	if doJS && promptYes("  Run 'npm install' now? [y/N] ") {
		clean.Exec(false, ctx.ProjectRoot, "npm", "install")
	}
	if doIOS && promptYes("  Run 'pod install' now? [y/N] ") {
		clean.Exec(false, filepath.Join(ctx.ProjectRoot, "ios"), "pod", "install", "--repo-update")
	}
	return nil
}

func promptYes(msg string) bool {
	fmt.Print(msg)
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

func containsStr(sl []string, s string) bool {
	for _, x := range sl {
		if x == s {
			return true
		}
	}
	return false
}

func init() { detect.Register(rn{}) }
