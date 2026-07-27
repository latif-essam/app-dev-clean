package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
	"github.com/latif-essam/app-dev-clean/internal/platform"
	"github.com/latif-essam/app-dev-clean/internal/ui"
)

const usage = `app-dev-clean - cross-platform dev-cache cleaner

  app-dev-clean                interactive menu (inside a known project)
  app-dev-clean <target>...    run named targets (e.g. ios js metro)
  app-dev-clean local-all      all local targets for detected type(s)
  app-dev-clean nuclear        local-all + global caches + reinstall (confirmed)
  app-dev-clean --type <t>     scope to one detector (rn|android|ios|flutter|expo)
  app-dev-clean --dry-run      show what would be freed; delete nothing
  app-dev-clean -y, --yes      skip confirmation prompts
  app-dev-clean --root         print resolved root + detected type(s)
  app-dev-clean --version      print version
  app-dev-clean --help         this help
`

func Run(args []string, version string) int {
	o, err := parse(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}
	switch {
	case o.Help:
		fmt.Print(usage)
		return 0
	case o.Version:
		fmt.Println("app-dev-clean", version)
		return 0
	}

	paths := platform.Detect()
	ctx := detect.Context{Paths: paths, DryRun: o.DryRun, Yes: o.Yes}

	// global-only invocation is allowed without a project.
	onlyGlobals := len(o.Targets) > 0
	for _, t := range o.Targets {
		if !isGlobalName(t) {
			onlyGlobals = false
		}
	}

	var res *detect.Result
	if !onlyGlobals || len(o.Targets) == 0 {
		res, err = detect.Resolve(mustCwd())
		if err != nil {
			fmt.Fprintln(os.Stderr, "✗ not inside a recognized project.")
			fmt.Fprintln(os.Stderr, "  refusing local cleanup so nothing is deleted in the wrong place.")
			fmt.Fprintln(os.Stderr, "  cd into a project, or run a global target (e.g. gradle-global).")
			return 1
		}
		ctx.ProjectRoot = res.Root
		fmt.Printf("==> project: %s (%s)\n", res.Root, typeNames(res, o.TypeFilter))
	}

	if o.ShowRoot {
		if res == nil {
			fmt.Fprintln(os.Stderr, "✗ not a project")
			return 1
		}
		fmt.Println(res.Root)
		fmt.Println("types:", typeNames(res, ""))
		return 0
	}

	targets := collectTargets(res, o.TypeFilter)

	// Gather raw selections (menu rows OR CLI args, either may include combos).
	var raw []string
	if len(o.Targets) == 0 {
		rows := ui.Rows(targets, detect.Globals(), ctx)
		raw = ui.Run(rows)
		if len(raw) == 0 {
			fmt.Println("nothing selected")
			return 0
		}
	} else {
		raw = o.Targets
	}

	// Detect nuclear BEFORE expansion (expandCombos replaces the token); nuclear
	// forces unconditional reinstall. Then expand once for both paths.
	nuclear := containsStr(raw, "nuclear")
	ctx.Force = nuclear
	selected := expandCombos(raw, targets)

	if !o.Yes && needsConfirm(selected) {
		fmt.Printf("About to clean shared/global caches: %s\n", strings.Join(selected, " "))
		if !promptYes("  These affect ALL projects. Proceed? [y/N] ") {
			fmt.Println("aborted")
			return 0
		}
	}

	freed := runTargets(ctx, res, selected, targets)
	fmt.Printf("\nDone. Reclaimed ~%s\n", clean.Human(freed))
	return 0
}

func mustCwd() string {
	d, err := os.Getwd()
	if err != nil {
		return "."
	}
	return d
}

func typeNames(res *detect.Result, filter string) string {
	if res == nil {
		return ""
	}
	var names []string
	for _, d := range res.Matched {
		if filter == "" || d.Name() == filter {
			names = append(names, d.Name())
		}
	}
	return strings.Join(names, "+")
}

// collectTargets returns the union of local targets from matched detectors
// (deduped by name), honoring an optional --type filter.
func collectTargets(res *detect.Result, filter string) []detect.Target {
	var out []detect.Target
	seen := map[string]bool{}
	if res != nil {
		for _, d := range res.Matched {
			if filter != "" && d.Name() != filter {
				continue
			}
			for _, tg := range d.Targets() {
				if !seen[tg.Name] {
					seen[tg.Name] = true
					out = append(out, tg)
				}
			}
		}
	}
	return out
}

func targetByName(name string, local []detect.Target) (detect.Target, bool) {
	for _, tg := range local {
		if tg.Name == name {
			return tg, true
		}
	}
	for _, g := range detect.Globals() {
		if g.Name == name {
			return g, true
		}
	}
	return detect.Target{}, false
}

func expandCombos(requested []string, local []detect.Target) []string {
	var out []string
	for _, r := range requested {
		switch r {
		case "local-all":
			for _, tg := range local {
				out = append(out, tg.Name)
			}
		case "nuclear":
			for _, tg := range local {
				out = append(out, tg.Name)
			}
			for _, g := range detect.Globals() {
				out = append(out, g.Name)
			}
		default:
			out = append(out, r)
		}
	}
	return out
}

func needsConfirm(selected []string) bool {
	for _, s := range selected {
		if isGlobalName(s) {
			return true
		}
	}
	return false
}

// runTargets executes each selected target (already combo-expanded), summing
// reclaimed bytes, then runs each matched detector's PostRun hook. PostRun
// itself decides reinstall behavior from ctx (DryRun/Force/Yes). Unknown names
// warn and are skipped.
func runTargets(ctx detect.Context, res *detect.Result, selected []string, local []detect.Target) int64 {
	var freed int64
	for _, name := range selected {
		tg, ok := targetByName(name, local)
		if !ok {
			fmt.Printf("  ! unknown target: %s\n", name)
			continue
		}
		f, err := tg.Run(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ! %s: %v\n", name, err)
		}
		freed += f
	}
	if res != nil {
		for _, d := range res.Matched {
			if pr, ok := d.(detect.PostRunner); ok {
				_ = pr.PostRun(ctx, selected)
			}
		}
	}
	return freed
}

func promptYes(msg string) bool {
	fmt.Print(msg)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

func containsStr(sl []string, s string) bool {
	for _, x := range sl {
		if x == s {
			return true
		}
	}
	return false
}
