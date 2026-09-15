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
	// Missing home must never produce a relative cleanup path.
	joinHome := func(parts ...string) string {
		if home == "" {
			return ""
		}
		return filepath.Join(append([]string{home}, parts...)...)
	}
	p.GradleCache = joinHome(".gradle", "caches")
	switch goos {
	case "windows":
		if local := env("LOCALAPPDATA"); local != "" {
			p.PubCache = filepath.Join(local, "Pub", "Cache")
		}
		p.TmpDir = env("TEMP")
	case "darwin":
		p.XcodeDD = joinHome("Library", "Developer", "Xcode", "DerivedData")
		p.CocoaPods = joinHome("Library", "Caches", "CocoaPods")
		p.PubCache = joinHome(".pub-cache")
		p.TmpDir = tmpOr(env, "/tmp")
	default: // linux and others
		p.PubCache = joinHome(".pub-cache")
		p.TmpDir = tmpOr(env, "/tmp")
	}
	if gradle := env("GRADLE_USER_HOME"); gradle != "" {
		p.GradleCache = filepath.Join(gradle, "caches")
	}
	if pub := env("PUB_CACHE"); pub != "" {
		p.PubCache = pub
	}
	return p
}

func tmpOr(env func(string) string, def string) string {
	if t := env("TMPDIR"); t != "" {
		return t
	}
	return def
}
