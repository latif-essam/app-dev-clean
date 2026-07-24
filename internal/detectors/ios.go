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

func iosLocal(ctx detect.Context) (int64, error) {
	return clean.Remove(ctx.DryRun, iosLocalPaths(ctx.ProjectRoot)...), nil
}

func (ios) Targets() []detect.Target {
	return []detect.Target{{
		Name:  "ios",
		Label: "ios",
		Desc:  "build/, Pods, Podfile.lock, SwiftPM .build",
		Scope: detect.Local,
		Paths: func(ctx detect.Context) []string { return iosLocalPaths(ctx.ProjectRoot) },
		Run:   iosLocal,
	}}
}

func init() { detect.Register(ios{}) }
