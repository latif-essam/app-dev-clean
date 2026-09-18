package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/latif-essam/app-dev-clean/internal/detect"
	"github.com/latif-essam/app-dev-clean/internal/platform"
)

func TestRowsOmitsUnavailableGlobals(t *testing.T) {
	local := []detect.Target{{Name: "js", Label: "js", Scope: detect.Local}}
	globals := []detect.Target{
		{Name: "gradle-global", Label: "gradle", Scope: detect.Global,
			Paths: func(c detect.Context) []string { return []string{"/g"} }},
		{Name: "xcode-dd", Label: "xcode", Scope: detect.Global,
			Paths: func(c detect.Context) []string { return nil }}, // unavailable
	}
	rows := Rows(local, globals, detect.Context{Paths: platform.Paths{}})
	for _, r := range rows {
		if r.Target == "xcode-dd" {
			t.Fatal("unavailable global must be omitted")
		}
	}
}

func TestModelToggleSelects(t *testing.T) {
	m := newModel([]Row{
		{Header: "LOCAL"},
		{Target: "js", Label: "js"},
		{Target: "ios", Label: "ios"},
	})
	m.cursor = 1 // on "js"
	m = m.toggle()
	got := m.selectedTargets()
	if len(got) != 1 || got[0] != "js" {
		t.Fatalf("want [js], got %v", got)
	}
}

func TestSelectAllOnlySelectsLocalTargets(t *testing.T) {
	rows := Rows([]detect.Target{{Name: "js"}, {Name: "metro", Scope: detect.Shared}}, []detect.Target{{Name: "pub-cache", Scope: detect.Global, Paths: func(detect.Context) []string { return []string{"/cache"} }}}, detect.Context{})
	m := newModel(rows)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	got := updated.(model).selectedTargets()
	if len(got) != 1 || got[0] != "js" {
		t.Fatalf("select-all must not include shared caches or combos: %v", got)
	}
}

func TestCombosAreExclusive(t *testing.T) {
	m := newModel([]Row{{Target: "js"}, {Target: "local-all"}, {Target: "nuclear"}})
	m = m.toggle()
	m.cursor = 2
	m = m.toggle()
	if got := m.selectedTargets(); len(got) != 1 || got[0] != "nuclear" {
		t.Fatalf("combo must replace individual selections: %v", got)
	}
	m.cursor = 0
	m = m.toggle()
	if got := m.selectedTargets(); len(got) != 1 || got[0] != "js" {
		t.Fatalf("individual selection must clear combo: %v", got)
	}
}

func TestRowsCarryTargetPaths(t *testing.T) {
	local := []detect.Target{
		{Name: "js", Label: "js", Scope: detect.Local,
			Paths: func(detect.Context) []string { return []string{"/tmp/node_modules"} }},
		{Name: "watchman", Label: "watchman", Scope: detect.Local},
	}
	rows := Rows(local, nil, detect.Context{})
	for _, r := range rows {
		switch r.Target {
		case "js":
			if len(r.Paths) != 1 {
				t.Fatalf("js row must carry its cleanup path, got %v", r.Paths)
			}
		case "watchman":
			if r.Paths != nil {
				t.Fatalf("a target without paths must carry none, got %v", r.Paths)
			}
		}
	}
}

func TestSizeTextReportsProgress(t *testing.T) {
	m := newModel([]Row{
		{Target: "js", Paths: []string{"/tmp/node_modules"}},
		{Target: "watchman"},
	})
	if got := m.sizeText(0); got != "..." {
		t.Fatalf("unmeasured row must show progress, got %q", got)
	}
	if got := m.sizeText(1); got != "" {
		t.Fatalf("a row that cleans nothing must show no size, got %q", got)
	}
	updated, _ := m.Update(sizeMsg{row: 0, size: 2048})
	if got := updated.(model).sizeText(0); got != "2.0 KB" {
		t.Fatalf("want 2.0 KB, got %q", got)
	}
}

func TestInitMeasuresOnlyRowsWithPaths(t *testing.T) {
	m := newModel([]Row{
		{Target: "js", Paths: []string{"/tmp/a"}},
		{Header: "LOCAL"},
		{Target: "watchman"},
	})
	if m.Init() == nil {
		t.Fatal("rows with paths must be measured")
	}
	empty := newModel([]Row{{Target: "watchman"}})
	if empty.Init() != nil {
		t.Fatal("nothing to measure must issue no work")
	}
}

func TestViewAlignsSizesWithoutTrailingSpace(t *testing.T) {
	m := newModel([]Row{
		{Target: "js", Label: "js", Desc: "short", Paths: []string{"/a"}},
		{Target: "android", Label: "android", Desc: "a much longer description than the first", Paths: []string{"/b"}},
		{Target: "watchman", Label: "watchman", Desc: "no paths"},
	})
	m.sizes[0] = 2048
	m.sizes[1] = 4096
	var cols []int
	for _, line := range strings.Split(m.View(), "\n") {
		if line != strings.TrimRight(line, " ") {
			t.Fatalf("trailing whitespace: %q", line)
		}
		if i := strings.LastIndex(line, " KB"); i >= 0 {
			cols = append(cols, i)
		}
	}
	if len(cols) != 2 || cols[0] != cols[1] {
		t.Fatalf("sizes must share one column, got %v", cols)
	}
}
