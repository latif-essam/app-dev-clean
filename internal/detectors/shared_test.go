package detectors

import (
	"testing"

	"github.com/latif-essam/app-dev-clean/internal/detect"
	"github.com/latif-essam/app-dev-clean/internal/platform"
)

// targetByName finds a local target in a detector's Targets() slice.
func targetByName(t *testing.T, targets []detect.Target, name string) detect.Target {
	t.Helper()
	for _, tg := range targets {
		if tg.Name == name {
			return tg
		}
	}
	t.Fatalf("target %q not offered; got %+v", name, targets)
	return detect.Target{}
}

// assertPaths checks that tg offers every path in want and none in unwanted.
// Build expected values with filepath.Join — the host separator differs on the
// Windows runner.
func assertPaths(t *testing.T, tg detect.Target, ctx detect.Context, want, unwanted []string) {
	t.Helper()
	got := map[string]bool{}
	for _, p := range tg.Paths(ctx) {
		got[p] = true
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("%s: missing path %q; got %v", tg.Name, w, tg.Paths(ctx))
		}
	}
	for _, u := range unwanted {
		if got[u] {
			t.Errorf("%s: must not offer root-level path %q", tg.Name, u)
		}
	}
}

func globalByName(t *testing.T, name string) detect.Target {
	t.Helper()
	for _, g := range detect.Globals() {
		if g.Name == name {
			return g
		}
	}
	t.Fatalf("global target %q not registered", name)
	return detect.Target{}
}

func TestGlobalsRegistered(t *testing.T) {
	for _, n := range []string{"gradle-global", "xcode-dd", "pods-cache", "pub-cache"} {
		g := globalByName(t, n)
		if g.Scope != detect.Global {
			t.Fatalf("%s must be Global scope", n)
		}
	}
}

func TestGradleGlobalUsesPlatformPath(t *testing.T) {
	g := globalByName(t, "gradle-global")
	ctx := detect.Context{Paths: platform.Paths{GradleCache: "/home/x/.gradle/caches"}}
	got := g.Paths(ctx)
	if len(got) != 1 || got[0] != "/home/x/.gradle/caches" {
		t.Fatalf("gradle-global path wrong: %+v", got)
	}
}

func TestXcodeGlobalEmptyOnNonMac(t *testing.T) {
	g := globalByName(t, "xcode-dd")
	// XcodeDD "" (as on win/linux) -> no paths offered
	if got := g.Paths(detect.Context{Paths: platform.Paths{XcodeDD: ""}}); len(got) != 0 {
		t.Fatalf("xcode-dd must offer no path when XcodeDD empty, got %+v", got)
	}
}
