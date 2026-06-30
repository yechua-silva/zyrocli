package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Confirm type ────────────────────────────────────────────────────────

// ConfirmResult represents the user's choice.
type ConfirmResult int

const (
	// ConfirmYes means the user selected "Sí".
	ConfirmYes ConfirmResult = iota
	// ConfirmNo means the user selected "No".
	ConfirmNo
)

// ── ConfirmModel ────────────────────────────────────────────────────────

// ConfirmModel is a bubbletea model for interactive Yes/No prompts.
// The user navigates with ↑↓ arrows and confirms with Enter.
type ConfirmModel struct {
	// question is the text to display.
	question string

	// affirmative is the "Yes" button text (default: "Sí").
	affirmative string

	// negative is the "No" button text (default: "No").
	negative string

	// cursor: 0 = Yes (default), 1 = No.
	cursor int

	// result is set when the user presses Enter.
	result ConfirmResult

	// done indicates the user has made a choice.
	done bool

	// width for layout.
	width int
}

// NewConfirmModel creates a new confirm model.
func NewConfirmModel(question string) ConfirmModel {
	return ConfirmModel{
		question:    question,
		affirmative: "Sí",
		negative:    "No",
		cursor:      0, // default: Sí
		width:       80,
	}
}

// ── tea.Model implementation ────────────────────────────────────────────

func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k", "down", "j":
			// Toggle between 0 (Yes) and 1 (No)
			m.cursor = (m.cursor + 1) % 2
			return m, nil

		case "enter":
			if m.cursor == 0 {
				m.result = ConfirmYes
			} else {
				m.result = ConfirmNo
			}
			m.done = true
			return m, tea.Quit

		case "q", "esc", "ctrl+c":
			// Cancel = No
			m.result = ConfirmNo
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m ConfirmModel) View() string {
	var b strings.Builder

	// Question
	questionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF"))

	b.WriteString(questionStyle.Render(m.question))
	b.WriteString("\n\n")

	// Options
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#F97316")).
		Padding(0, 2)

	unselectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A1A1AA")).
		Padding(0, 2)

	// "Sí" option
	if m.cursor == 0 {
		b.WriteString(fmt.Sprintf("  %s\n", selectedStyle.Render(fmt.Sprintf("● %s (recomendado)", m.affirmative))))
	} else {
		b.WriteString(fmt.Sprintf("  %s\n", unselectedStyle.Render(fmt.Sprintf("○ %s", m.affirmative))))
	}

	// "No" option
	if m.cursor == 1 {
		b.WriteString(fmt.Sprintf("  %s\n", selectedStyle.Render(fmt.Sprintf("● %s", m.negative))))
	} else {
		b.WriteString(fmt.Sprintf("  %s\n", unselectedStyle.Render(fmt.Sprintf("○ %s", m.negative))))
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#52525B"))
	b.WriteString(helpStyle.Render("[↑/↓] navegar  [Enter] confirmar  [q] cancelar"))

	return b.String()
}

// ── RunConfirm ──────────────────────────────────────────────────────────

// RunConfirm displays an interactive Yes/No prompt and returns the choice.
// It blocks until the user makes a selection.
func RunConfirm(question string) (bool, error) {
	m := NewConfirmModel(question)
	p := tea.NewProgram(m)

	final, err := p.Run()
	if err != nil {
		return false, fmt.Errorf("confirm TUI: %w", err)
	}

	model := final.(ConfirmModel)
	return model.result == ConfirmYes, nil
}
