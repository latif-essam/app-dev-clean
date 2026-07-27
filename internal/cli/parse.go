package cli

import (
	"fmt"

	"github.com/latif-essam/app-dev-clean/internal/detect"
)

type Options struct {
	Targets    []string
	TypeFilter string
	DryRun     bool
	Yes        bool
	ShowRoot   bool
	Help       bool
	Version    bool
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
		case "--yes", "-y":
			o.Yes = true
		case "--type":
			if i+1 >= len(args) {
				return o, fmt.Errorf("--type needs a value")
			}
			i++
			o.TypeFilter = args[i]
		default:
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
	return name == "nuclear"
}
