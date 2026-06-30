package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── SelectOption ─────────────────────────────────────────────────────────

// SelectOption represents an option in a selection list.
type SelectOption struct {
	Key         string
	Label       string
	Description string
	Detail      string // right-aligned detail (e.g. "1024 dims", "2.0 GB")
}

// ── SelectModel ──────────────────────────────────────────────────────────

// SelectModel is a generic list selector for choosing from options.
type SelectModel struct {
	title    string
	subtitle string
	options  []SelectOption
	cursor   int
	width    int
	selected string
	done     bool
}

// NewSelectModel creates a new selection list.
func NewSelectModel(title, subtitle string, options []SelectOption) SelectModel {
	return SelectModel{
		title:    title,
		subtitle: subtitle,
		options:  options,
		cursor:   0,
		width:    80,
	}
}

func (m SelectModel) Init() tea.Cmd {
	return nil
}

func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.options[m.cursor].Key
			m.done = true
			return m, tea.Quit
		case "q", "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m SelectModel) View() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F97316")).
		Padding(0, 1)
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n")

	if m.subtitle != "" {
		subStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#52525B")).
			Italic(true).
			Padding(0, 1)
		b.WriteString(subStyle.Render(m.subtitle))
		b.WriteString("\n\n")
	}

	// Options
	for i, opt := range m.options {
		cursor := "  "
		if i == m.cursor {
			cursor = "○ "
		}

		if i == m.cursor {
			// Selected
			labelStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#F97316")).
				Padding(0, 2)

			detailStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A1A1AA")).
				Padding(0, 2)

			descStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D4D4D8")).
				Padding(0, 4)

			line := fmt.Sprintf("%s%s", cursor, labelStyle.Render(opt.Label))
			if opt.Detail != "" {
				line += detailStyle.Render(opt.Detail)
			}
			b.WriteString(line)
			b.WriteString("\n")
			if opt.Description != "" {
				b.WriteString(descStyle.Render(opt.Description))
				b.WriteString("\n")
			}
		} else {
			labelStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A1A1AA")).
				Padding(0, 2)

			detailStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#52525B")).
				Padding(0, 2)

			line := fmt.Sprintf("  %s", labelStyle.Render(opt.Label))
			if opt.Detail != "" {
				line += detailStyle.Render(opt.Detail)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#52525B"))
	b.WriteString(helpStyle.Render("[↑/↓] navegar  [Enter] seleccionar  [q] cancelar"))

	return b.String()
}

// ── RunSelect ────────────────────────────────────────────────────────────

// RunSelect displays a selection list and returns the selected key.
// Returns empty string if cancelled.
func RunSelect(title, subtitle string, options []SelectOption) string {
	m := NewSelectModel(title, subtitle, options)
	p := tea.NewProgram(m)

	final, err := p.Run()
	if err != nil {
		return ""
	}

	model := final.(SelectModel)
	if model.selected != "" {
		return model.selected
	}
	return ""
}
