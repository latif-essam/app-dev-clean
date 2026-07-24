package detect

import (
	"fmt"
	"os"
	"path/filepath"
)

type Result struct {
	Root    string
	Matched []Detector
}

func Resolve(start string) (*Result, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	for {
		var matched []Detector
		for _, d := range Detectors() {
			if d.Detect(dir) {
				matched = append(matched, d)
			}
		}
		if len(matched) > 0 {
			return &Result{Root: dir, Matched: matched}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir { // reached filesystem root
			return nil, fmt.Errorf("no project root up-tree from %s: %w", start, os.ErrNotExist)
		}
		dir = parent
	}
}
