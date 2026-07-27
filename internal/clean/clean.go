package clean

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Size(path string) int64 {
	var total int64
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, e := d.Info(); e == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func Remove(dryRun bool, paths ...string) (freed int64) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			continue // absent -> skip silently
		}
		sz := Size(p)
		if dryRun {
			fmt.Printf("  [dry-run] would remove %s (%s)\n", p, Human(sz))
			freed += sz
			continue
		}
		if err := os.RemoveAll(p); err != nil {
			fmt.Fprintf(os.Stderr, "  ! could not remove %s: %v\n", p, err)
			continue
		}
		fmt.Printf("  removed %s (%s)\n", p, Human(sz))
		freed += sz
	}
	return freed
}

func Exec(dryRun bool, dir, name string, args ...string) {
	if dryRun {
		fmt.Printf("  [dry-run] would run: %s %s\n", name, strings.Join(args, " "))
		return
	}
	// A name containing a path separator is an explicit executable path
	// (e.g. an absolute gradlew path) — stat it directly. Otherwise resolve
	// via PATH. Never pass a bare relative path like "./gradlew" to
	// exec.Command: its resolution is NOT relative to cmd.Dir and varies by
	// OS/Go version — callers must pass an absolute path instead.
	if strings.ContainsAny(name, `/\`) {
		if _, err := os.Stat(name); err != nil {
			fmt.Printf("  ! %s not found; skipping\n", name)
			return
		}
	} else if _, err := exec.LookPath(name); err != nil {
		fmt.Printf("  ! %s not found; skipping\n", name)
		return
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		fmt.Printf("  ! %s failed (continuing): %v\n", name, err)
	}
}

func Human(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
