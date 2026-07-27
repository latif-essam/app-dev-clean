package detectors

import (
	"path/filepath"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

type ios struct{}

func (ios) Name() string { return "ios" }

func (ios) Detect(dir string) bool {
	if exists(filepath.Join(dir, "Package.swift")) {
		return true
	}
	for _, pat := range []string{"*.xcodeproj", "*.xcworkspace"} {
		if m, _ := filepath.Glob(filepath.Join(dir, pat)); len(m) > 0 {
			return true
		}
	}
	return false
}

func iosLocalPaths(root string) []string {
	return []string{
		filepath.Join(root, "build"),
		filepath.Join(root, "Pods"),
		filepath.Join(root, "Podfile.lock"),
		filepath.Join(root, ".build"), // SwiftPM
	}
}

// iosTarget builds the ios cleanup target for a given base dir — the root for a
// native Xcode/SwiftPM repo, <root>/ios for RN/Expo. See shared.go.
func iosTarget(base func(detect.Context) string, desc string) detect.Target {
	return detect.Target{
		Name:  "ios",
		Label: "ios",
		Desc:  desc,
		Scope: detect.Local,
		Paths: func(ctx detect.Context) []string { return iosLocalPaths(base(ctx)) },
		Run: func(ctx detect.Context) (int64, error) {
			return clean.Remove(ctx.DryRun, iosLocalPaths(base(ctx))...), nil
		},
	}
}

func (ios) Targets() []detect.Target {
	return []detect.Target{
		iosTarget(projectRoot, "build/, Pods, Podfile.lock, SwiftPM .build"),
	}
}

func init() { detect.Register(ios{}) }
