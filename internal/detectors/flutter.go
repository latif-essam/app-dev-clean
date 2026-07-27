package detectors

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

type flutter struct{}

func (flutter) Name() string { return "flutter" }

func (flutter) Detect(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, "pubspec.yaml"))
	if err != nil {
		return false
	}
	return strings.Contains(string(b), "flutter")
}

func flutterLocalPaths(root string) []string {
	return []string{filepath.Join(root, "build"), filepath.Join(root, ".dart_tool")}
}

func (flutter) Targets() []detect.Target {
	return []detect.Target{{
		Name:  "flutter",
		Label: "flutter",
		Desc:  "build/, .dart_tool/ + flutter clean",
		Scope: detect.Local,
		Paths: func(c detect.Context) []string { return flutterLocalPaths(c.ProjectRoot) },
		Run: func(c detect.Context) (int64, error) {
			clean.Exec(c.DryRun, c.ProjectRoot, "flutter", "clean")
			return clean.Remove(c.DryRun, flutterLocalPaths(c.ProjectRoot)...), nil
		},
	}}
}

func init() { detect.Register(flutter{}) }
