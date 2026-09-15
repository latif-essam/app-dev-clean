package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"testing"

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
