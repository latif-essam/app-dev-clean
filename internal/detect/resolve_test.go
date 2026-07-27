package detect

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type markerDet struct {
	name   string
	marker string // filename that identifies a root of this type
}

func (m markerDet) Name() string { return m.name }
func (m markerDet) Detect(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, m.marker))
	return err == nil
}
func (m markerDet) Targets() []Target { return nil }

func TestResolveFindsRootFromNested(t *testing.T) {
	registry = []Detector{markerDet{"rn", "package.json"}}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "src", "deep", "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Resolve(nested)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if res.Root != root {
		t.Fatalf("want root %q, got %q", root, res.Root)
	}
	if len(res.Matched) != 1 || res.Matched[0].Name() != "rn" {
		t.Fatalf("want matched [rn], got %+v", res.Matched)
	}
}

func TestResolveReturnsUnion(t *testing.T) {
	registry = []Detector{markerDet{"rn", "package.json"}, markerDet{"android", "settings.gradle"}}
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "package.json"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(root, "settings.gradle"), []byte(""), 0o644)
	res, err := Resolve(root)
	if err != nil || len(res.Matched) != 2 {
		t.Fatalf("want 2 matched, got %+v err=%v", res, err)
	}
}

func TestResolveRefusesOutsideProject(t *testing.T) {
	registry = []Detector{markerDet{"rn", "package.json"}}
	_, err := Resolve(t.TempDir())
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want ErrNotExist, got %v", err)
	}
}
