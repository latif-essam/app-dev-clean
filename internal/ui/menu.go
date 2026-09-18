package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/latif-essam/app-dev-clean/internal/clean"
	"github.com/latif-essam/app-dev-clean/internal/detect"
)

type Row struct {
	Scope  detect.Scope
	Target string
	Label  string
	Desc   string
	Header string // non-empty => section header (not selectable)
	Paths  []string
}

func targetPaths(t detect.Target, ctx detect.Context) []string {
	if t.Paths == nil {
		return nil
	}
	return t.Paths(ctx)
}

func Rows(local, globals []detect.Target, ctx detect.Context) []Row {
	var rows []Row
	for _, section := range []struct {
		scope detect.Scope
		title string
	}{{detect.Local, "LOCAL (this project)"}, {detect.Shared, "SHARED (all projects)"}} {
		added := false
		for _, t := range local {
			if t.Scope != section.scope {
				continue
			}
			if !added {
				rows = append(rows, Row{Header: section.title})
				added = true
			}
			rows = append(rows, Row{Target: t.Name, Label: t.Label, Desc: t.Desc, Scope: t.Scope, Paths: targetPaths(t, ctx)})
		}
	}
	var avail []detect.Target
	for _, g := range globals {
		if g.Paths != nil && len(g.Paths(ctx)) > 0 {
			avail = append(avail, g)
		}
	}
	if len(avail) > 0 {
		rows = append(rows, Row{Header: "GLOBAL (shared across ALL projects)"})
		for _, g := range avail {
			rows = append(rows, Row{Target: g.Name, Label: g.Label, Desc: g.Desc, Scope: g.Scope, Paths: targetPaths(g, ctx)})
		}
	}
	rows = append(rows, Row{Header: "COMBOS"})
	rows = append(rows, Row{Target: "local-all", Label: "local-all", Desc: "all local targets"})
	rows = append(rows, Row{Target: "nuclear", Label: "nuclear", Desc: "everything + reinstall"})
	return rows
}

type model struct {
	rows    []Row
	cursor  int
	checked map[int]bool
	done    bool
	quit    bool
	sizes   map[int]int64
}

func newModel(rows []Row) model {
	m := model{rows: rows, checked: map[int]bool{}, sizes: map[int]int64{}}
	m.cursor = m.firstSelectable(0, 1)
	return m
}

// sizeMsg carries a row's measured size back to the menu.
type sizeMsg struct {
	row  int
	size int64
}

// measure walks a row's paths off the render loop; a large cache can take
// seconds and the menu stays usable meanwhile.
func measure(row int, paths []string) tea.Cmd {
	return func() tea.Msg {
		var total int64
		for _, p := range paths {
			total += clean.Size(p)
		}
		return sizeMsg{row: row, size: total}
	}
}

func (m model) firstSelectable(from, dir int) int {
	i := from
	for i >= 0 && i < len(m.rows) {
		if m.rows[i].Header == "" {
			return i
		}
		i += dir
	}
	return from
}

func isCombo(name string) bool { return name == "local-all" || name == "nuclear" }

func (m model) toggle() model {
	if m.cursor >= 0 && m.cursor < len(m.rows) && m.rows[m.cursor].Header == "" {
		checked := !m.checked[m.cursor]
		if checked {
			if isCombo(m.rows[m.cursor].Target) {
				m.checked = map[int]bool{}
			} else {
				for i, row := range m.rows {
					if isCombo(row.Target) {
						delete(m.checked, i)
					}
				}
			}
		}
		m.checked[m.cursor] = checked
	}
	return m
}

func (m model) selectedTargets() []string {
	var out []string
	for i, r := range m.rows {
		if r.Header == "" && m.checked[i] {
			out = append(out, r.Target)
		}
	}
	return out
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for i, r := range m.rows {
		if len(r.Paths) > 0 {
			cmds = append(cmds, measure(i, r.Paths))
		}
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sz, ok := msg.(sizeMsg); ok {
		m.sizes[sz.row] = sz.size
		return m, nil
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "q", "ctrl+c":
		m.quit = true
		return m, tea.Quit
	case "up", "k":
		m.cursor = m.move(-1)
	case "down", "j":
		m.cursor = m.move(1)
	case " ":
		m = m.toggle()
	case "a":
		m.checked = map[int]bool{}
		for i, r := range m.rows {
			if r.Header == "" && r.Scope == detect.Local && !isCombo(r.Target) {
				m.checked[i] = true
			}
		}
	case "n":
		m.checked = map[int]bool{}
	case "enter":
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m model) move(dir int) int {
	i := m.cursor + dir
	for i >= 0 && i < len(m.rows) {
		if m.rows[i].Header == "" {
			return i
		}
		i += dir
	}
	return m.cursor
}

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

// sizeText is blank for rows that clean no paths, and "..." until measured.
func (m model) sizeText(row int) string {
	if len(m.rows[row].Paths) == 0 {
		return ""
	}
	size, ok := m.sizes[row]
	if !ok {
		return "..."
	}
	return clean.Human(size)
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("  app-dev-clean") + "\n")
	b.WriteString("  up/down move · SPACE toggle · a local only · n none · ENTER run · q quit\n\n")
	desc := 0
	for _, r := range m.rows {
		if r.Header == "" && len(r.Desc) > desc {
			desc = len(r.Desc)
		}
	}
	for i, r := range m.rows {
		if r.Header != "" {
			b.WriteString("\n  " + headerStyle.Render(r.Header) + "\n")
			continue
		}
		mark := " "
		if m.checked[i] {
			mark = "x"
		}
		pointer := "  "
		label := strings.TrimRight(fmt.Sprintf("%-14s %-*s %9s", r.Label, desc, r.Desc, m.sizeText(i)), " ")
		if i == m.cursor {
			pointer = cursorStyle.Render("> ")
			label = cursorStyle.Render(label)
		}
		b.WriteString(fmt.Sprintf("  %s[%s] %s\n", pointer, mark, label))
	}
	return b.String()
}

func Run(rows []Row) []string {
	m := newModel(rows)
	p := tea.NewProgram(m)
	res, err := p.Run()
	if err != nil {
		return nil
	}
	fm := res.(model)
	if fm.quit || !fm.done {
		return nil
	}
	return fm.selectedTargets()
}
