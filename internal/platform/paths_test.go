package platform

import "testing"

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestDetectForDarwin(t *testing.T) {
	p := detectFor("darwin", envFrom(map[string]string{"HOME": "/Users/x"}))
	if p.XcodeDD == "" || p.CocoaPods == "" {
		t.Fatalf("darwin must have xcode+cocoapods paths, got %+v", p)
	}
	if p.GradleCache != "/Users/x/.gradle/caches" {
		t.Fatalf("gradle path wrong: %q", p.GradleCache)
	}
}

func TestDetectForLinux(t *testing.T) {
	p := detectFor("linux", envFrom(map[string]string{"HOME": "/home/x"}))
	if p.XcodeDD != "" || p.CocoaPods != "" {
		t.Fatalf("linux must NOT have xcode/cocoapods, got %+v", p)
	}
	if p.PubCache != "/home/x/.pub-cache" {
		t.Fatalf("pub path wrong: %q", p.PubCache)
	}
}

func TestDetectForWindows(t *testing.T) {
	p := detectFor("windows", envFrom(map[string]string{
		"USERPROFILE": `C:\Users\x`, "LOCALAPPDATA": `C:\Users\x\AppData\Local`, "TEMP": `C:\Temp`,
	}))
	if p.XcodeDD != "" || p.CocoaPods != "" {
		t.Fatalf("windows must NOT have xcode/cocoapods")
	}
	if p.TmpDir != `C:\Temp` {
		t.Fatalf("windows tmp wrong: %q", p.TmpDir)
	}
}
