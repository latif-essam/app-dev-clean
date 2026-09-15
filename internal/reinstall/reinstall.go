// Package reinstall plans deterministic installs without discarding download caches.
package reinstall

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/latif-essam/app-dev-clean/internal/clean"
)

type Action struct {
	Target, Dir, Command string
	Args                 []string
	version              string
	yarnMajor            int
}

func (a Action) Run(dryRun bool) error { return clean.Exec(dryRun, a.Dir, a.Command, a.Args...) }

func (a Action) Check() error {
	if err := clean.CheckCommand(a.Command); err != nil {
		return err
	}
	args := []string{"--version"}
	if a.Command == "bundle" {
		args = []string{"exec", "pod", "--version"}
	}
	cmd, err := clean.Command(a.Dir, a.Command, args...)
	if err != nil {
		return err
	}
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("cannot run %s in %s: %w", a.Command, a.Dir, err)
	}
	actual := strings.TrimSpace(string(out))
	if a.version != "" && actual != a.version {
		return fmt.Errorf("%s requires version %s, installed version is %q; activate the project version before cleaning", a.Command, a.version, actual)
	}
	if a.yarnMajor > 0 {
		major, _ := strconv.Atoi(strings.Split(actual, ".")[0])
		if major == 0 || (a.yarnMajor == 1) != (major == 1) {
			return fmt.Errorf("Yarn version %q does not match the lockfile format", actual)
		}
	}
	return nil
}

type manifest struct {
	PackageManager string          `json:"packageManager"`
	Workspaces     json.RawMessage `json:"workspaces"`
}

func readManifest(dir string) (manifest, error) {
	var m manifest
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err == nil {
		err = json.Unmarshal(b, &m)
	}
	return m, err
}
func exists(path string) bool { _, err := os.Stat(path); return err == nil }
func hasWorkspaces(m manifest) bool {
	s := strings.TrimSpace(string(m.Workspaces))
	return s != "" && s != "null" && s != "[]" && s != "{}"
}

// CheckJSRoot refuses shared workspace dependency resets. Workspaces need a
// manager-specific, explicit workspace workflow instead of a single-app reset.
func CheckJSRoot(dir string) error {
	for at := dir; ; at = filepath.Dir(at) {
		m, err := readManifest(at)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("read package.json in %s: %w", at, err)
		}
		if hasWorkspaces(m) || exists(filepath.Join(at, "pnpm-workspace.yaml")) {
			return fmt.Errorf("shared workspace at %s: refusing a whole dependency reset; use its package manager's workspace install/repair workflow", at)
		}
		if exists(filepath.Join(at, ".git")) || filepath.Dir(at) == at {
			return nil
		}
	}
}

func JS(dir string) (Action, error) {
	a := Action{Target: "js", Dir: dir}
	if err := CheckJSRoot(dir); err != nil {
		return a, err
	}
	m, err := readManifest(dir)
	if err != nil {
		return a, err
	}
	locks := map[string][]string{
		"npm":  {"npm-shrinkwrap.json", "package-lock.json"},
		"pnpm": {"pnpm-lock.yaml"}, "yarn": {"yarn.lock"}, "bun": {"bun.lock", "bun.lockb"},
	}
	var manager string
	for name, files := range locks {
		for _, file := range files {
			if exists(filepath.Join(dir, file)) {
				if manager != "" && manager != name {
					return a, fmt.Errorf("conflicting package-manager lockfiles; keep the intended manager's lockfile before reinstalling")
				}
				manager = name
			}
		}
	}
	if manager == "" {
		return a, fmt.Errorf("reinstall requires an existing lockfile; create and review one with the project's package manager first")
	}
	if m.PackageManager != "" {
		name, version, ok := strings.Cut(m.PackageManager, "@")
		if !ok || version == "" {
			return a, fmt.Errorf("invalid packageManager %q", m.PackageManager)
		}
		if name != manager {
			return a, fmt.Errorf("packageManager %s conflicts with the %s lockfile", name, manager)
		}
		a.version = strings.Split(version, "+")[0]
	}
	a.Command = manager
	switch manager {
	case "npm":
		a.Args = []string{"ci", "--prefer-offline"}
	case "pnpm":
		a.Args = []string{"install", "--frozen-lockfile", "--prefer-offline"}
	case "yarn":
		b, err := os.ReadFile(filepath.Join(dir, "yarn.lock"))
		if err != nil {
			return a, err
		}
		if strings.Contains(string(b), "# yarn lockfile v1") {
			a.Args, a.yarnMajor = []string{"install", "--frozen-lockfile", "--prefer-offline"}, 1
		} else if strings.Contains(string(b), "__metadata:") {
			a.Args, a.yarnMajor = []string{"install", "--immutable"}, 2
		} else {
			return a, fmt.Errorf("unrecognized Yarn lockfile; refusing to guess install flags")
		}
	case "bun":
		a.Args = []string{"install", "--frozen-lockfile"}
	}
	return a, nil
}

// Pods returns nil for Swift-only or managed projects without a Podfile.
func Pods(dir, projectRoot string) (*Action, error) {
	if !exists(filepath.Join(dir, "Podfile")) {
		return nil, nil
	}
	if !exists(filepath.Join(dir, "Podfile.lock")) {
		return nil, fmt.Errorf("CocoaPods reinstall requires %s; create and review the lockfile first", filepath.Join(dir, "Podfile.lock"))
	}
	a := &Action{Target: "ios", Dir: dir, Command: "pod", Args: []string{"install", "--deployment"}}
	for at := dir; ; at = filepath.Dir(at) {
		b, err := os.ReadFile(filepath.Join(at, "Gemfile"))
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if err == nil && strings.Contains(string(b), "cocoapods") {
			if !exists(filepath.Join(at, "Gemfile.lock")) {
				return nil, fmt.Errorf("Bundler reinstall requires %s", filepath.Join(at, "Gemfile.lock"))
			}
			a.Command, a.Args = "bundle", []string{"exec", "pod", "install", "--deployment"}
			break
		}
		if at == projectRoot || filepath.Dir(at) == at {
			break
		}
	}
	return a, nil
}
