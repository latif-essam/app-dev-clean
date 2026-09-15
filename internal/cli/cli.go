package cli

import (
	"bufio"
	"fmt"
	"github.com/latif-essam/app-dev-clean/internal/reinstall"
	"os"
	"path/filepath"
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
  app-dev-clean --reinstall    reinstall selected JS/Pods using existing lockfiles
  app-dev-clean --allow-shared allow shared/global cleanup with -y
  app-dev-clean -y, --yes      non-interactive cleanup (no implicit reinstall)
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
		if o.TypeFilter != "" && typeNames(res, o.TypeFilter) == "" {
			fmt.Fprintf(os.Stderr, "error: project does not match --type %s\n", o.TypeFilter)
			return 2
		}
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

	input := bufio.NewReader(os.Stdin)
	nuclear := containsStr(raw, "nuclear")
	selected := expandCombos(raw, targets, ctx)
	// Validate the entire plan before confirmations, commands, or deletion.
	if err := preflight(ctx, selected, targets); err != nil {
		fmt.Fprintln(os.Stderr, "error: cleanup cancelled before deletion:", err)
		return 1
	}
	if !o.DryRun && needsConfirm(selected, targets) && !o.AllowShared {
		if o.Yes {
			fmt.Fprintln(os.Stderr, "error: shared/global cleanup affects all projects; add --allow-shared explicitly or run interactively")
			return 2
		}
		fmt.Printf("Shared/global targets selected: %s\n", strings.Join(selected, " "))
		if !promptYes(input, "  These affect ALL projects. Proceed? [y/N] ") {
			fmt.Println("aborted")
			return 0
		}
	}
	install := o.Reinstall || nuclear
	if !install && !o.Yes && !o.DryRun && (containsStr(selected, "js") || containsStr(selected, "ios")) {
		install = promptYes(input, "Reinstall selected dependencies from existing lockfiles after cleanup? [y/N] ")
	}
	var actions []reinstall.Action
	if install {
		actions, err = planReinstalls(ctx, selected, targets)
		if err == nil && !o.DryRun {
			for _, action := range actions {
				if err = action.Check(); err != nil {
					break
				}
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: reinstall preflight failed; nothing deleted:", err)
			return 1
		}
	}
	freed, err := runTargets(ctx, selected, targets)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nCleanup stopped after reclaiming ~%s: %v\n", clean.Human(freed), err)
		return 1
	}
	for _, action := range actions {
		if err := action.Run(o.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "\nCleanup reclaimed ~%s, but reinstall failed: %v\nLockfiles and download caches were preserved; retry the displayed install command.\n", clean.Human(freed), err)
			return 1
		}
	}
	if o.DryRun {
		fmt.Printf("\nDry run complete. Would reclaim ~%s\n", clean.Human(freed))
	} else {
		fmt.Printf("\nDone. Reclaimed ~%s\n", clean.Human(freed))
	}
	return 0
}

func preflight(ctx detect.Context, selected []string, local []detect.Target) error {
	var localPaths []string
	for _, name := range selected {
		tg, ok := targetByName(name, local)
		if !ok {
			return fmt.Errorf("unknown/unavailable target %q; see --help", name)
		}
		var paths []string
		if tg.Paths != nil {
			paths = tg.Paths(ctx)
		}
		root := ctx.ProjectRoot
		if tg.Scope == detect.Global {
			if len(paths) == 0 {
				return fmt.Errorf("%s is unavailable on this platform or cache configuration", name)
			}
			var err error
			root, err = clean.CacheScope(paths[0], ctx.Paths.Home, mustCwd())
			if err != nil {
				return err
			}
		} else if tg.Scope == detect.Shared {
			root = ctx.Paths.TmpDir
		} else {
			localPaths = append(localPaths, paths...)
		}
		if err := clean.ValidatePaths(root, paths...); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if name == "js" {
			if err := reinstall.CheckJSRoot(ctx.ProjectRoot); err != nil {
				return err
			}
		}
		if tg.Check != nil {
			if err := tg.Check(ctx); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	if len(localPaths) > 0 {
		return clean.ProtectTracked(ctx.ProjectRoot, localPaths...)
	}
	return nil
}

func planReinstalls(ctx detect.Context, selected []string, local []detect.Target) ([]reinstall.Action, error) {
	var actions []reinstall.Action
	// CocoaPods autolinking may load modules installed by the JS action.
	for _, name := range []string{"js", "ios"} {
		if !containsStr(selected, name) {
			continue
		}
		switch name {
		case "js":
			a, err := reinstall.JS(ctx.ProjectRoot)
			if err != nil {
				return nil, err
			}
			actions = append(actions, a)
		case "ios":
			tg, _ := targetByName(name, local)
			for _, path := range tg.Paths(ctx) {
				if filepath.Base(path) != "Pods" {
					continue
				}
				a, err := reinstall.Pods(filepath.Dir(path), ctx.ProjectRoot)
				if err != nil {
					return nil, err
				}
				if a != nil {
					actions = append(actions, *a)
				}
			}
		}
	}
	return actions, nil
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

func expandCombos(requested []string, local []detect.Target, contexts ...detect.Context) []string {
	var out []string
	seen := map[string]bool{}
	add := func(name string) {
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	for _, r := range requested {
		switch r {
		case "local-all", "nuclear":
			for _, tg := range local {
				if r == "nuclear" || tg.Scope == detect.Local {
					add(tg.Name)
				}
			}
			if r == "nuclear" {
				for _, g := range detect.Globals() {
					if len(contexts) > 0 && (g.Paths == nil || len(g.Paths(contexts[0])) == 0) {
						continue
					}
					add(g.Name)
				}
			}
		default:
			add(r)
		}
	}
	return out
}

func needsConfirm(selected []string, local []detect.Target) bool {
	for _, name := range selected {
		if tg, ok := targetByName(name, local); ok && tg.Scope != detect.Local {
			return true
		}
	}
	return false
}

func runTargets(ctx detect.Context, selected []string, local []detect.Target) (int64, error) {
	var freed int64
	for _, name := range selected {
		tg, ok := targetByName(name, local)
		if !ok {
			return freed, fmt.Errorf("unknown target %s", name)
		}
		fmt.Printf("==> %s: %s\n", tg.Label, tg.Desc)
		f, err := tg.Run(ctx)
		freed += f
		if err != nil {
			return freed, fmt.Errorf("%s: %w", name, err)
		}
	}
	return freed, nil
}

func promptYes(input *bufio.Reader, msg string) bool {
	fmt.Print(msg)
	line, _ := input.ReadString('\n')
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
