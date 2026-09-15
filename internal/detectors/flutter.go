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
		Check: func(c detect.Context) error {
			if c.DryRun {
				return nil
			}
			return clean.CheckCommand("flutter")
		},
		Run: func(c detect.Context) (int64, error) {
			if err := clean.Exec(c.DryRun, c.ProjectRoot, "flutter", "clean"); err != nil {
				return 0, err
			}
			return clean.Remove(c.DryRun, c.ProjectRoot, flutterLocalPaths(c.ProjectRoot)...)
		},
	}}
}

func init() { detect.Register(flutter{}) }
