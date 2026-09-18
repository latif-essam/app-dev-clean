package cli

import (
	"testing"

	"github.com/latif-essam/app-dev-clean/internal/detect"
)

var gateTargets = []detect.Target{
	{Name: "js", Scope: detect.Local},
	{Name: "metro", Scope: detect.Shared},
}

func TestSharedGate(t *testing.T) {
	cases := []struct {
		name     string
		opts     Options
		selected []string
		want     gate
	}{
		{"local only", Options{}, []string{"js"}, gateProceed},
		{"local only non-interactive", Options{Yes: true}, []string{"js"}, gateProceed},
		{"shared interactive asks", Options{}, []string{"metro"}, gateAsk},
		{"shared with -y refuses", Options{Yes: true}, []string{"metro"}, gateRefuse},
		{"shared with -y and --allow-shared", Options{Yes: true, AllowShared: true}, []string{"metro"}, gateProceed},
		{"shared allowed interactively", Options{AllowShared: true}, []string{"metro"}, gateProceed},
		{"dry run never gates", Options{DryRun: true, Yes: true}, []string{"metro"}, gateProceed},
		{"mixed selection gates", Options{Yes: true}, []string{"js", "metro"}, gateRefuse},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sharedGate(c.opts, c.selected, gateTargets); got != c.want {
				t.Fatalf("want %v, got %v", c.want, got)
			}
		})
	}
}

func TestReinstallDecision(t *testing.T) {
	cases := []struct {
		name        string
		opts        Options
		selected    []string
		nuclear     bool
		wantInstall bool
		wantAsk     bool
	}{
		{"flag forces install", Options{Reinstall: true}, []string{"js"}, false, true, false},
		{"nuclear forces install", Options{}, nil, true, true, false},
		{"js asks interactively", Options{}, []string{"js"}, false, false, true},
		{"ios asks interactively", Options{}, []string{"ios"}, false, false, true},
		{"android never asks", Options{}, []string{"android"}, false, false, false},
		{"-y never asks", Options{Yes: true}, []string{"js"}, false, false, false},
		{"dry run never asks", Options{DryRun: true}, []string{"js"}, false, false, false},
		{"--reinstall wins over -y", Options{Reinstall: true, Yes: true}, []string{"js"}, false, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			install, ask := reinstallDecision(c.opts, c.selected, c.nuclear)
			if install != c.wantInstall || ask != c.wantAsk {
				t.Fatalf("want install=%v ask=%v, got install=%v ask=%v", c.wantInstall, c.wantAsk, install, ask)
			}
		})
	}
}
