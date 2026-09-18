package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/latif-essam/app-dev-clean/internal/detect"
)

type jsonProject struct {
	Root  string   `json:"root"`
	Types []string `json:"types"`
}

type jsonTarget struct {
	Name  string   `json:"name"`
	Scope string   `json:"scope"`
	Paths []string `json:"paths"`
	Size  int64    `json:"size"`
	Freed int64    `json:"freed"`
}

type jsonReinstall struct {
	Target  string   `json:"target"`
	Dir     string   `json:"dir"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// jsonReport is the contract for --json. Renaming a field breaks callers.
type jsonReport struct {
	Version    string          `json:"version"`
	DryRun     bool            `json:"dryRun"`
	Executed   bool            `json:"executed"`
	Project    *jsonProject    `json:"project"`
	Targets    []jsonTarget    `json:"targets"`
	FreedTotal int64           `json:"freedTotal"`
	Reinstall  []jsonReinstall `json:"reinstall"`
	Error      *string         `json:"error"`
}

func newReport(version string, dryRun bool) *jsonReport {
	return &jsonReport{
		Version:   version,
		DryRun:    dryRun,
		Targets:   []jsonTarget{},
		Reinstall: []jsonReinstall{},
	}
}

func scopeName(s detect.Scope) string {
	switch s {
	case detect.Global:
		return "global"
	case detect.Shared:
		return "shared"
	default:
		return "local"
	}
}

func (r *jsonReport) fail(err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	r.Error = &msg
}

func (r *jsonReport) emit() {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: cannot encode report:", err)
		return
	}
	fmt.Fprintln(os.Stdout, string(b))
}
