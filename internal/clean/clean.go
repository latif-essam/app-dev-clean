package clean

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrGitUnavailable reports that git is missing, so tracked-file protection
// could not run.
var ErrGitUnavailable = errors.New("git executable not found")

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

// relative never permits deleting the scope itself or paths outside it.
func relative(root, path string) (string, error) {
	if !filepath.IsAbs(root) || !filepath.IsAbs(path) || filepath.Dir(filepath.Clean(root)) == filepath.Clean(root) {
		return "", fmt.Errorf("cleanup requires an absolute, non-filesystem-root scope: %q", root)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || !filepath.IsLocal(rel) {
		return "", fmt.Errorf("refusing to remove %q outside/beside cleanup scope %q", path, root)
	}
	return rel, nil
}

// ValidatePaths checks all ancestors, including when the final path is missing.
// A leaf symlink is safe: removal unlinks it without following its destination.
func ValidatePaths(root string, paths ...string) error {
	for _, p := range paths {
		rel, err := relative(root, p)
		if err != nil {
			return err
		}
		r, err := os.OpenRoot(root)
		if err != nil {
			return err
		}
		err = validate(r, rel)
		r.Close()
		if err != nil {
			return fmt.Errorf("unsafe/inaccessible cleanup path %s: %w", p, err)
		}
	}
	return nil
}

func validate(r *os.Root, rel string) error {
	parts := strings.Split(rel, string(filepath.Separator))
	for i := 1; i < len(parts); i++ {
		dir := filepath.Join(parts[:i]...)
		info, err := r.Stat(dir) // Root refuses escaping ancestor symlinks.
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", dir)
		}
	}
	_, err := r.Lstat(rel)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Remove uses rooted file operations throughout traversal, so an ancestor
// replaced with an escaping symlink after validation cannot redirect deletion.
// Root.RemoveAll requires Go 1.25; this traversal also supports our Go 1.24 floor.
func Remove(dryRun bool, root string, paths ...string) (freed int64, err error) {
	if err = ValidatePaths(root, paths...); err != nil {
		return 0, err
	}
	if len(paths) == 0 {
		return 0, nil
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return 0, err
	}
	defer r.Close()
	for _, p := range paths {
		rel, err := relative(root, p)
		if err != nil {
			return freed, err
		}
		if _, err := r.Lstat(rel); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return freed, err
		}
		if !dryRun {
			fmt.Printf("  removing %s...\n", p)
		}
		sz, err := removeTree(r, rel, dryRun)
		freed += sz
		if err != nil {
			return freed, fmt.Errorf("remove %s: %w", p, err)
		}
		if dryRun {
			fmt.Printf("  [dry-run] would remove %s (%s)\n", p, Human(sz))
		} else {
			fmt.Printf("  removed %s (%s)\n", p, Human(sz))
		}
	}
	return freed, nil
}

func removeTree(r *os.Root, path string, dryRun bool) (int64, error) {
	info, err := r.Lstat(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		if !dryRun {
			if err := r.Remove(path); err != nil {
				return 0, err
			}
		}
		return info.Size(), nil
	}
	// OpenRoot holds this directory even if its parent entry changes during work.
	sub, err := r.OpenRoot(path)
	if err != nil {
		return 0, err
	}
	defer sub.Close()
	f, err := sub.Open(".")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	var total int64
	for {
		names, readErr := f.Readdirnames(128)
		for _, name := range names {
			n, err := removeTree(sub, name, dryRun)
			total += n
			if err != nil {
				return total, err
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				return total, readErr
			}
			break
		}
	}
	if !dryRun {
		f.Close()
		sub.Close()
		err = r.Remove(path)
	}
	return total, err
}

func CheckCommand(name string) error {
	_, err := Command("", name)
	return err
}

func Exec(dryRun bool, dir, name string, args ...string) error {
	if dryRun {
		fmt.Printf("  [dry-run] would run in %s: %s %s\n", dir, name, strings.Join(args, " "))
		return nil
	}
	cmd, err := Command(dir, name, args...)
	if err != nil {
		return err
	}
	cmd.Dir, cmd.Stdin, cmd.Stdout, cmd.Stderr = dir, os.Stdin, os.Stdout, os.Stderr
	fmt.Printf("==> %s %s\n", filepath.Base(name), strings.Join(args, " "))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return nil
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

// CacheScope anchors home caches at home, keeping symlinked ancestors confined.
// An explicitly configured external cache uses its nearest existing parent.
func CacheScope(path, home, project string) (string, error) {
	if !filepath.IsAbs(path) || filepath.Dir(filepath.Clean(path)) == filepath.Clean(path) {
		return "", fmt.Errorf("unsafe cache directory %q; use an absolute cache path", path)
	}
	path = filepath.Clean(path)
	info, err := os.Lstat(path)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err == nil && !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		return "", fmt.Errorf("cache path must be a directory: %s", path)
	}
	if filepath.IsAbs(home) {
		for _, name := range []string{"Desktop", "Documents", "Downloads", "Pictures", "Music", "Movies", "Videos", "Library", "AppData", "Applications", ".ssh", ".gnupg", ".config", ".local"} {
			if strings.EqualFold(path, filepath.Join(home, name)) {
				return "", fmt.Errorf("refusing to use personal directory %s as a cache", path)
			}
		}
	}
	for _, marker := range []string{".git", "package.json", "pubspec.yaml", "go.mod", "Cargo.toml", "Package.swift", "settings.gradle", "settings.gradle.kts"} {
		if _, err := os.Lstat(filepath.Join(path, marker)); err == nil {
			return "", fmt.Errorf("refusing to use project directory %s as a cache (%s present)", path, marker)
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	for _, protected := range []string{home, project} {
		if protected == "" {
			continue
		}
		rel, err := filepath.Rel(path, protected)
		if err == nil && (rel == "." || filepath.IsLocal(rel)) {
			return "", fmt.Errorf("cache %s contains a protected home/project directory", path)
		}
	}
	if filepath.IsAbs(home) {
		if rel, err := filepath.Rel(home, path); err == nil && rel != "." && filepath.IsLocal(rel) {
			return home, nil
		}
	}
	root := filepath.Dir(path)
	for {
		info, err := os.Stat(root)
		if err == nil && info.IsDir() {
			break
		}
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(root)
		if parent == root {
			break
		}
		root = parent
	}
	if _, err := relative(root, path); err != nil {
		return "", err
	}
	return root, nil
}

// ProtectTracked prevents named output directories from erasing versioned data.
// Git is required only when the project is inside a repository.
func ProtectTracked(root string, paths ...string) error {
	inRepo := false
	for dir := root; ; dir = filepath.Dir(dir) {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			inRepo = true
			break
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	if !inRepo {
		return nil
	}
	gitPath, lookErr := exec.LookPath("git")
	if lookErr != nil {
		return fmt.Errorf("%w: cannot verify that cleanup would spare Git-tracked files in %s; install Git and retry, or clean a project that is not in a Git repository", ErrGitUnavailable, root)
	}
	args := []string{"-C", root, "ls-files", "--cached", "-z", "--"}
	for _, path := range paths {
		rel, err := relative(root, path)
		if err != nil {
			return err
		}
		args = append(args, filepath.ToSlash(rel))
	}
	cmd := exec.Command(gitPath, args...)
	cmd.Env = append(os.Environ(), "GIT_LITERAL_PATHSPECS=1")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("cannot verify tracked-file protection: %w", err)
	}
	if len(out) > 0 {
		return fmt.Errorf("refusing to delete tracked data: %s; move versioned files outside cleanup directories first", strings.Split(string(out), "\x00")[0])
	}
	return nil
}
