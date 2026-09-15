package detect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Result struct {
	Root    string
	Matched []Detector
}

func matches(dir string) []Detector {
	var out []Detector
	for _, d := range Detectors() {
		if d.Detect(dir) {
			out = append(out, d)
		}
	}
	return out
}
func hasType(ds []Detector, names ...string) bool {
	for _, d := range ds {
		for _, name := range names {
			if d.Name() == name {
				return true
			}
		}
	}
	return false
}
func marker(dir string, names ...string) bool {
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func Resolve(start string) (*Result, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	var result *Result
	for {
		matched := matches(dir)
		if len(matched) > 0 {
			if result == nil {
				result = &Result{Root: dir, Matched: matched}
			}
			if hasType(result.Matched, "rn", "expo", "flutter") {
				return result, nil
			}
			rel, _ := filepath.Rel(dir, result.Root)
			nativeSubtree := rel == "android" || rel == "ios" || strings.HasPrefix(rel, "android"+string(filepath.Separator)) || strings.HasPrefix(rel, "ios"+string(filepath.Separator))
			if nativeSubtree && hasType(matched, "rn", "expo", "flutter") {
				return &Result{Root: dir, Matched: matched}, nil
			}
			if hasType(result.Matched, "android") && hasType(matched, "android") && marker(dir, "settings.gradle", "settings.gradle.kts") {
				result = &Result{Root: dir, Matched: matched}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir || marker(dir, ".git") {
			break
		}
		dir = parent
	}
	if result != nil {
		return result, nil
	}
	return nil, fmt.Errorf("no project root up-tree from %s: %w", start, os.ErrNotExist)
}
