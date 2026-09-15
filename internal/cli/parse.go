package cli

import (
	"fmt"
	"strings"

	"github.com/latif-essam/app-dev-clean/internal/detect"
)

type Options struct {
	Targets     []string
	TypeFilter  string
	DryRun      bool
	Yes         bool
	ShowRoot    bool
	Help        bool
	AllowShared bool
	Reinstall   bool
	Version     bool
}

func parse(args []string) (Options, error) {
	var o Options
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--help", "-h":
			o.Help = true
		case "--version", "-v":
			o.Version = true
		case "--root":
			o.ShowRoot = true
		case "--dry-run":
			o.DryRun = true
		case "--allow-shared":
			o.AllowShared = true
		case "--reinstall":
			o.Reinstall = true
		case "--yes", "-y":
			o.Yes = true
		case "--type":
			if i+1 >= len(args) {
				return o, fmt.Errorf("--type needs a value")
			}
			i++
			o.TypeFilter = args[i]
			switch o.TypeFilter {
			case "rn", "expo", "android", "ios", "flutter":
			default:
				return o, fmt.Errorf("unknown project type %q", o.TypeFilter)
			}
		default:
			if strings.HasPrefix(a, "-") {
				return o, fmt.Errorf("unknown option %q; see --help", a)
			}
			o.Targets = append(o.Targets, a)
		}
	}
	return o, nil
}

func isGlobalName(name string) bool {
	for _, g := range detect.Globals() {
		if g.Name == name {
			return true
		}
	}
	return false
}
