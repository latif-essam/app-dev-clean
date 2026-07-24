package detectors

import (
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

func androidLocal(ctx detect.Context) (int64, error) {
	wrapperName := "gradlew"
	if isWindows() {
		wrapperName = "gradlew.bat"
	}
	wrapperPath := filepath.Join(ctx.ProjectRoot, wrapperName)
	if exists(wrapperPath) {
		// Pass the ABSOLUTE wrapper path: bare "./gradlew" does not resolve
		// relative to cmd.Dir in os/exec (see clean.Exec).
		clean.Exec(ctx.DryRun, ctx.ProjectRoot, wrapperPath, "clean")
	}
	return clean.Remove(ctx.DryRun, androidLocalPaths(ctx.ProjectRoot)...), nil
}

func (android) Targets() []detect.Target {
	return []detect.Target{{
		Name:  "android",
		Label: "android",
		Desc:  "build/, app/build, .gradle, .cxx + gradlew clean",
		Scope: detect.Local,
		Paths: func(ctx detect.Context) []string { return androidLocalPaths(ctx.ProjectRoot) },
		Run:   androidLocal,
	}}
}

func init() { detect.Register(android{}) }
