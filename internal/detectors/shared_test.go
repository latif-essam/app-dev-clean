package detectors

import (
	"testing"

	"github.com/latif-essam/app-dev-clean/internal/detect"
	"github.com/latif-essam/app-dev-clean/internal/platform"
)

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
