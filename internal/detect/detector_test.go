package detect

import "testing"

type fake struct{ name string }

func (f fake) Name() string          { return f.name }
func (f fake) Detect(string) bool    { return true }
func (f fake) Targets() []Target     { return nil }

func TestRegistry(t *testing.T) {
	registry = nil
	Register(fake{"a"})
	Register(fake{"b"})
	if len(Detectors()) != 2 {
		t.Fatalf("want 2 detectors, got %d", len(Detectors()))
	}
}

func TestGlobalRegistry(t *testing.T) {
	globals = nil
	RegisterGlobal(Target{Name: "gradle-global", Scope: Global})
	if len(Globals()) != 1 || Globals()[0].Scope != Global {
		t.Fatalf("global target not registered: %+v", Globals())
	}
}
