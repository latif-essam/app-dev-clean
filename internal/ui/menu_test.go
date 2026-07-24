package ui

import (
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
