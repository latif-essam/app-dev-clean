package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

type Paths struct {
	Home        string
	GradleCache string
	XcodeDD     string // "" when unavailable on this OS
	CocoaPods   string // "" when unavailable on this OS
	PubCache    string
	TmpDir      string
}

func Detect() Paths { return detectFor(runtime.GOOS, os.Getenv) }

func detectFor(goos string, env func(string) string) Paths {
	home := env("HOME")
	if goos == "windows" {
		home = env("USERPROFILE")
	}
	p := Paths{Home: home}
	p.GradleCache = filepath.Join(home, ".gradle", "caches")
	switch goos {
	case "windows":
		p.PubCache = filepath.Join(env("LOCALAPPDATA"), "Pub", "Cache")
		p.TmpDir = env("TEMP")
	case "darwin":
		p.XcodeDD = filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData")
		p.CocoaPods = filepath.Join(home, "Library", "Caches", "CocoaPods")
		p.PubCache = filepath.Join(home, ".pub-cache")
		p.TmpDir = tmpOr(env, "/tmp")
	default: // linux and others
		p.PubCache = filepath.Join(home, ".pub-cache")
		p.TmpDir = tmpOr(env, "/tmp")
	}
	return p
}

func tmpOr(env func(string) string, def string) string {
	if t := env("TMPDIR"); t != "" {
		return t
	}
	return def
}
