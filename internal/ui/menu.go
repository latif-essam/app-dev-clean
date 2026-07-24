package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/latif-essam/app-dev-clean/internal/detect"
)

type Row struct {
	Target string
	Label  string
	Desc   string
	Header string // non-empty => section header (not selectable)
}

func Rows(local, globals []detect.Target, ctx detect.Context) []Row {
	var rows []Row
	if len(local) > 0 {
		rows = append(rows, Row{Header: "LOCAL (project — fast to rebuild)"})
		for _, t := range local {
			rows = append(rows, Row{Target: t.Name, Label: t.Label, Desc: t.Desc})
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
			rows = append(rows, Row{Target: g.Name, Label: g.Label, Desc: g.Desc})
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
}

func newModel(rows []Row) model {
	m := model{rows: rows, checked: map[int]bool{}}
	m.cursor = m.firstSelectable(0, 1)
	return m
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

func (m model) toggle() model {
	if m.cursor >= 0 && m.cursor < len(m.rows) && m.rows[m.cursor].Header == "" {
		m.checked[m.cursor] = !m.checked[m.cursor]
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

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		for i, r := range m.rows {
			if r.Header == "" {
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

func (m model) View() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("  app-dev-clean") + "\n")
	b.WriteString("  up/down move · SPACE toggle · a all · n none · ENTER run · q quit\n\n")
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
		label := fmt.Sprintf("%-14s %s", r.Label, r.Desc)
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
