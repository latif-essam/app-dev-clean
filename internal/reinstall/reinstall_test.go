package reinstall

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func put(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
func TestJSInstallPlans(t *testing.T) {
	for _, tc := range []struct {
		name, lock, body, manager string
		args                      []string
	}{
		{"npm", "package-lock.json", "{}", "npm@10.8.2", []string{"ci", "--prefer-offline"}},
		{"npm shrinkwrap", "npm-shrinkwrap.json", "{}", "", []string{"ci", "--prefer-offline"}},
		{"pnpm", "pnpm-lock.yaml", "lockfileVersion: '9.0'", "pnpm@9.15.0", []string{"install", "--frozen-lockfile", "--prefer-offline"}},
		{"Yarn classic", "yarn.lock", "# yarn lockfile v1", "yarn@1.22.22", []string{"install", "--frozen-lockfile", "--prefer-offline"}},
		{"Yarn modern", "yarn.lock", "__metadata:\n  version: 8", "yarn@4.5.0", []string{"install", "--immutable"}},
		{"Bun", "bun.lock", "{}", "bun@1.3.0", []string{"install", "--frozen-lockfile"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			put(t, dir, "package.json", `{"packageManager":"`+tc.manager+`"}`)
			put(t, dir, tc.lock, tc.body)
			a, err := JS(dir)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(a.Args, tc.args) {
				t.Fatalf("wrong deterministic install: %+v", a)
			}
			if tc.manager != "" && a.Command != strings.Split(tc.manager, "@")[0] {
				t.Fatalf("wrong package manager: %+v", a)
			}
		})
	}
}

func TestJSRejectsAmbiguousOrMissingLocks(t *testing.T) {
	for _, tc := range []struct {
		name, manifest string
		files          []string
	}{
		{"no lockfile", `{}`, nil},
		{"conflicting locks", `{}`, []string{"package-lock.json", "pnpm-lock.yaml"}},
		{"mismatched declaration", `{"packageManager":"pnpm@9.0.0"}`, []string{"package-lock.json"}},
		{"unknown Yarn format", `{}`, []string{"yarn.lock"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			put(t, dir, "package.json", tc.manifest)
			for _, file := range tc.files {
				put(t, dir, file, "{}")
			}
			if _, err := JS(dir); err == nil {
				t.Fatal("unsafe reinstall accepted")
			}
		})
	}
}

func TestWorkspaceGuard(t *testing.T) {
	for _, kind := range []string{"npm", "pnpm"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			nested := filepath.Join(dir, "apps", "mobile")
			if err := os.MkdirAll(nested, 0o755); err != nil {
				t.Fatal(err)
			}
			put(t, nested, "package.json", `{}`)
			if kind == "npm" {
				put(t, dir, "package.json", `{"workspaces":["apps/*"]}`)
			} else {
				put(t, dir, "pnpm-workspace.yaml", "packages:\n - apps/*")
			}
			if err := CheckJSRoot(nested); err == nil {
				t.Fatal("nested workspace reset must be refused")
			}
		})
	}
}

func TestPodPlansPreserveLocks(t *testing.T) {
	dir := t.TempDir()
	ios := filepath.Join(dir, "ios")
	if err := os.Mkdir(ios, 0o755); err != nil {
		t.Fatal(err)
	}
	if a, err := Pods(ios, dir); a != nil || err != nil {
		t.Fatalf("Swift-only project must skip pods: %v %v", a, err)
	}
	put(t, ios, "Podfile", "platform :ios, '15.0'")
	if _, err := Pods(ios, dir); err == nil {
		t.Fatal("missing Podfile.lock must block reinstall")
	}
	put(t, ios, "Podfile.lock", "COCOAPODS: 1.16.2")
	a, err := Pods(ios, dir)
	if err != nil || !reflect.DeepEqual(a.Args, []string{"install", "--deployment"}) {
		t.Fatalf("unexpected pod action: %+v %v", a, err)
	}
	put(t, dir, "Gemfile", "gem 'cocoapods'")
	if _, err := Pods(ios, dir); err == nil {
		t.Fatal("missing Gemfile.lock must block Bundler reinstall")
	}
	put(t, dir, "Gemfile.lock", "GEM")
	a, err = Pods(ios, dir)
	if err != nil || a.Command != "bundle" || !reflect.DeepEqual(a.Args, []string{"exec", "pod", "install", "--deployment"}) {
		t.Fatalf("Bundler must be respected: %+v %v", a, err)
	}
}
