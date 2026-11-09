package detectors

import (
	"fmt"
	"path/filepath"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

type android struct{}

func (android) Name() string { return "android" }

func (android) Detect(dir string) bool {
	for _, m := range []string{"settings.gradle", "settings.gradle.kts", "build.gradle", "build.gradle.kts"} {
		if exists(filepath.Join(dir, m)) {
			return true
		}
	}
	return false
}

func androidLocalPaths(root string) []string {
	return []string{
		filepath.Join(root, "build"),
		filepath.Join(root, "app", "build"),
		filepath.Join(root, ".gradle"),
		filepath.Join(root, ".cxx"),
		filepath.Join(root, "app", ".cxx"),
	}
}

func gradlewName() string {
	if isWindows() {
		return "gradlew.bat"
	}
	return "gradlew"
}

// androidLocalIn cleans the gradle project rooted at base(ctx) — the root for a
// native repo, <root>/android for RN/Expo. See the base helpers in shared.go.
func androidLocalIn(base func(detect.Context) string) func(detect.Context) (int64, error) {
	return func(ctx detect.Context) (int64, error) {
		dir := base(ctx)
		wrapperPath := filepath.Join(dir, gradlewName())
		if exists(wrapperPath) {
			// gradlew clean is best effort: it commonly fails when node_modules,
			// the JDK or a Gradle plugin is broken, which is exactly when the
			// build directories most need removing. Warn and keep deleting.
			if err := gradlewClean(ctx.DryRun, dir, wrapperPath); err != nil {
				fmt.Fprintf(clean.Out, "  warning: gradlew clean failed (%v); skipping it and removing build directories\n", err)
			}
		}
		return clean.Remove(ctx.DryRun, ctx.ProjectRoot, androidLocalPaths(dir)...)
	}
}

func gradlewClean(dryRun bool, dir, wrapperPath string) error {
	if !dryRun {
		if err := clean.CheckCommand(wrapperPath); err != nil {
			return err
		}
	}
	// Pass the ABSOLUTE wrapper path: bare "./gradlew" does not resolve
	// relative to cmd.Dir in os/exec (see clean.Exec).
	return clean.Exec(dryRun, dir, wrapperPath, "clean")
}

// androidTarget builds the android cleanup target for a given base dir, so the
// native and RN/Expo detectors share one definition.
func androidTarget(base func(detect.Context) string, desc string) detect.Target {
	return detect.Target{
		Name:  "android",
		Label: "android",
		Desc:  desc,
		Scope: detect.Local,
		Paths: func(ctx detect.Context) []string { return androidLocalPaths(base(ctx)) },
		Check: func(ctx detect.Context) error {
			wrapper := filepath.Join(base(ctx), gradlewName())
			if err := clean.ValidatePaths(ctx.ProjectRoot, wrapper); err != nil {
				return err
			}
			if exists(wrapper) {
				// Validate a child to check the executable's own symlink too.
				resolved, err := filepath.EvalSymlinks(wrapper)
				if err != nil {
					return err
				}
				scope, err := filepath.EvalSymlinks(ctx.ProjectRoot)
				if err != nil {
					return err
				}
				// A non-executable wrapper is not fatal: Run skips gradlew
				// clean with a warning and still removes the build output.
				if err := clean.ValidatePaths(scope, resolved); err != nil {
					return err
				}
			}
			return nil
		},
		Run: androidLocalIn(base),
	}
}

func (android) Targets() []detect.Target {
	return []detect.Target{
		androidTarget(projectRoot, "build/, app/build, .gradle, .cxx + gradlew clean"),
	}
}

func init() { detect.Register(android{}) }
