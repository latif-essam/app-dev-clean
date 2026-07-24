package detect

import "github.com/latif-essam/app-dev-clean/internal/platform"

type Scope int

const (
	Local Scope = iota
	Global
)

type Context struct {
	ProjectRoot string
	Paths       platform.Paths
	DryRun      bool
	Yes         bool
	Force       bool // nuclear: run reinstall without prompting
}

type Target struct {
	Name  string
	Label string
	Desc  string
	Scope Scope
	Paths func(ctx Context) []string
	Run   func(ctx Context) (freed int64, err error)
}

type Detector interface {
	Name() string
	Detect(dir string) bool
	Targets() []Target
}

// PostRunner is optionally implemented by detectors that offer post-clean
// actions (e.g. RN reinstall prompts). cli type-asserts for it.
type PostRunner interface {
	PostRun(ctx Context, ran []string) error
}

var registry []Detector

func Register(d Detector)      { registry = append(registry, d) }
func Detectors() []Detector    { return registry }

var globals []Target

func RegisterGlobal(t Target)  { globals = append(globals, t) }
func Globals() []Target        { return globals }
