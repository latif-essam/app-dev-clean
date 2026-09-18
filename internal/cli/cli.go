package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
	"github.com/latif-essam/app-dev-clean/internal/platform"
	"github.com/latif-essam/app-dev-clean/internal/reinstall"
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
  app-dev-clean --json         machine-readable report on stdout (no prompts)
  app-dev-clean --root         print resolved root + detected type(s)
  app-dev-clean --version      print version
  app-dev-clean --help         this help
`

func Run(args []string, version string) (code int) {
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

	// JSON mode keeps stdout for the report alone; progress moves to stderr.
	out := io.Writer(os.Stdout)
	var report *jsonReport
	var runErr error
	if o.JSON {
		out, clean.Out = os.Stderr, os.Stderr
		report = newReport(version, o.DryRun)
		defer func() {
			report.fail(runErr)
			report.emit()
		}()
	}
	fail := func(c int, err error) int {
		runErr = err
		fmt.Fprintln(os.Stderr, "error:", err)
		return c
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
			runErr = err
			fmt.Fprintln(os.Stderr, "✗ not inside a recognized project.")
			fmt.Fprintln(os.Stderr, "  refusing local cleanup so nothing is deleted in the wrong place.")
			fmt.Fprintln(os.Stderr, "  cd into a project, or run a global target (e.g. gradle-global).")
			return 1
		}
		ctx.ProjectRoot = res.Root
		if o.TypeFilter != "" && typeNames(res, o.TypeFilter) == "" {
			return fail(2, fmt.Errorf("project does not match --type %s", o.TypeFilter))
		}
		if report != nil {
			report.Project = &jsonProject{Root: res.Root, Types: typeList(res, o.TypeFilter)}
		}
		fmt.Fprintf(out, "==> project: %s (%s)\n", res.Root, typeNames(res, o.TypeFilter))
	}

	if o.ShowRoot {
		if res == nil {
			return fail(1, fmt.Errorf("not a project"))
		}
		fmt.Fprintln(out, res.Root)
		fmt.Fprintln(out, "types:", typeNames(res, ""))
		return 0
	}

	targets := collectTargets(res, o.TypeFilter)

	// --json with nothing named reports what could be cleaned, and deletes nothing.
	if report != nil && len(o.Targets) == 0 {
		report.Targets = planTargets(ctx, targets)
		return 0
	}

	// Gather raw selections (menu rows OR CLI args, either may include combos).
	var raw []string
	if len(o.Targets) == 0 {
		rows := ui.Rows(targets, detect.Globals(), ctx)
		raw = ui.Run(rows)
		if len(raw) == 0 {
			fmt.Fprintln(out, "nothing selected")
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
		return fail(1, fmt.Errorf("cleanup cancelled before deletion: %w", err))
	}
	switch sharedGate(o, selected, targets) {
	case gateRefuse:
		return fail(2, fmt.Errorf("shared/global cleanup affects all projects; add --allow-shared explicitly or run interactively"))
	case gateAsk:
		fmt.Fprintf(out, "Shared/global targets selected: %s\n", strings.Join(selected, " "))
		if !promptYes(input, "  These affect ALL projects. Proceed? [y/N] ") {
			fmt.Fprintln(out, "aborted")
			return 0
		}
	}
	install, ask := reinstallDecision(o, selected, nuclear)
	if ask {
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
			return fail(1, fmt.Errorf("reinstall preflight failed; nothing deleted: %w", err))
		}
		if report != nil {
			for _, a := range actions {
				report.Reinstall = append(report.Reinstall, jsonReinstall{Target: a.Target, Dir: a.Dir, Command: a.Command, Args: a.Args})
			}
		}
	}
	results, freed, err := runTargets(ctx, selected, targets, out)
	if report != nil {
		report.Targets, report.FreedTotal, report.Executed = results, freed, true
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nCleanup stopped after reclaiming ~%s\n", clean.Human(freed))
		return fail(1, err)
	}
	for _, action := range actions {
		if err := action.Run(o.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "\nCleanup reclaimed ~%s. Lockfiles and download caches were preserved; retry the displayed install command.\n", clean.Human(freed))
			return fail(1, fmt.Errorf("reinstall failed: %w", err))
		}
	}
	if o.DryRun {
		fmt.Fprintf(out, "\nDry run complete. Would reclaim ~%s\n", clean.Human(freed))
	} else {
		fmt.Fprintf(out, "\nDone. Reclaimed ~%s\n", clean.Human(freed))
	}
	return 0
}

// planTargets measures every available target without deleting anything.
func planTargets(ctx detect.Context, local []detect.Target) []jsonTarget {
	out := []jsonTarget{}
	all := append(append([]detect.Target{}, local...), detect.Globals()...)
	for _, tg := range all {
		var paths []string
		if tg.Paths != nil {
			paths = tg.Paths(ctx)
		}
		if len(paths) == 0 {
			continue
		}
		var size int64
		for _, p := range paths {
			size += clean.Size(p)
		}
		out = append(out, jsonTarget{Name: tg.Name, Scope: scopeName(tg.Scope), Paths: paths, Size: size})
	}
	return out
}

func typeList(res *detect.Result, filter string) []string {
	names := []string{}
	if res == nil {
		return names
	}
	for _, d := range res.Matched {
		if filter == "" || d.Name() == filter {
			names = append(names, d.Name())
		}
	}
	return names
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
		err := clean.ProtectTracked(ctx.ProjectRoot, localPaths...)
		// A dry run deletes nothing, so warn rather than block the preview.
		if errors.Is(err, clean.ErrGitUnavailable) && ctx.DryRun {
			fmt.Fprintln(os.Stderr, "warning: git is unavailable, so Git-tracked files were not checked; this dry run deletes nothing, but install Git before cleaning for real")
			return nil
		}
		return err
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

// gate is what the shared/global selection requires before anything is deleted.
type gate int

const (
	gateProceed gate = iota
	gateRefuse
	gateAsk
)

func sharedGate(o Options, selected []string, local []detect.Target) gate {
	if o.DryRun || o.AllowShared || !needsConfirm(selected, local) {
		return gateProceed
	}
	if o.Yes || o.JSON {
		return gateRefuse
	}
	return gateAsk
}

// reinstallDecision reports whether to reinstall, and whether to ask first.
func reinstallDecision(o Options, selected []string, nuclear bool) (install, ask bool) {
	if o.Reinstall || nuclear {
		return true, false
	}
	if o.Yes || o.DryRun || o.JSON {
		return false, false
	}
	return false, containsStr(selected, "js") || containsStr(selected, "ios")
}

func needsConfirm(selected []string, local []detect.Target) bool {
	for _, name := range selected {
		if tg, ok := targetByName(name, local); ok && tg.Scope != detect.Local {
			return true
		}
	}
	return false
}

func runTargets(ctx detect.Context, selected []string, local []detect.Target, out io.Writer) ([]jsonTarget, int64, error) {
	var freed int64
	results := []jsonTarget{}
	for _, name := range selected {
		tg, ok := targetByName(name, local)
		if !ok {
			return results, freed, fmt.Errorf("unknown target %s", name)
		}
		fmt.Fprintf(out, "==> %s: %s\n", tg.Label, tg.Desc)
		f, err := tg.Run(ctx)
		freed += f
		entry := jsonTarget{Name: tg.Name, Scope: scopeName(tg.Scope), Paths: []string{}, Size: f}
		if tg.Paths != nil {
			if p := tg.Paths(ctx); p != nil {
				entry.Paths = p
			}
		}
		if !ctx.DryRun {
			entry.Freed = f
		}
		results = append(results, entry)
		if err != nil {
			return results, freed, fmt.Errorf("%s: %w", name, err)
		}
	}
	return results, freed, nil
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
