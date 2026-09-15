package detect

import "github.com/latif-essam/app-dev-clean/internal/platform"

type Scope int

const (
	Local Scope = iota
	Global
	Shared
)

type Context struct {
	ProjectRoot string
	Paths       platform.Paths
	DryRun      bool
	Yes         bool
}

type Target struct {
	Name  string
	Label string
	Desc  string
	Scope Scope
	Paths func(ctx Context) []string
	Check func(ctx Context) error // preflight before any selected target runs
	Run   func(ctx Context) (freed int64, err error)
}

type Detector interface {
	Name() string
	Detect(dir string) bool
	Targets() []Target
}

var registry []Detector

func Register(d Detector)   { registry = append(registry, d) }
func Detectors() []Detector { return registry }

var globals []Target

func RegisterGlobal(t Target) { globals = append(globals, t) }
func Globals() []Target       { return globals }
