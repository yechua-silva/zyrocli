package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Menu item ────────────────────────────────────────────────────────────

// MenuItem represents a single option in the main menu.
type MenuItem struct {
	Key         string
	Label       string
	Description string
}

// ── MenuModel ────────────────────────────────────────────────────────────

// MenuModel is the main menu Bubbletea model.
type MenuModel struct {
	items    []MenuItem
	cursor   int
	width    int
	height   int
	selected string // key of selected item, empty if not yet selected
	done     bool
}

// NewMainMenu creates the main menu with default options.
func NewMainMenu() MenuModel {
	return MenuModel{
		items: []MenuItem{
			{
				Key:   "install",
				Label: "🚀 Instalación completa",
				Description: "Skills, MCP, OpenCode, HelixDB, Ollama,\n" +
					"modelos, GPU y pruebas. Todo en uno.",
			},
			{
				Key:   "setup",
				Label: "⚙️  Configurar servicios",
				Description: "Iniciar/verificar HelixDB y Ollama.\n" +
					"Detectar GPU e instalar backend.",
			},
			{
				Key:   "models",
				Label: "🔧 Configurar modelos IA",
				Description: "Elegir modelos de embeddings y chat.\n" +
					"Probar funcionamiento.",
			},
			{
				Key:   "autostart",
				Label: "🔄 Auto-inicio servicios",
				Description: "Configurar systemd/user para que HelixDB\n" +
					"y Ollama arranquen al prender el PC.",
			},
			{
				Key:         "about",
				Label:       "📖 Acerca de ZyroCLI",
				Description: "¿Qué es ZyroCLI?",
			},
			{
				Key:   "exit",
				Label: "❌ Salir",
				Description: "",
			},
		},
		cursor: 0,
		width:  80,
	}
}

// ── tea.Model implementation ─────────────────────────────────────────────

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			selected := m.items[m.cursor].Key
			m.selected = selected
			m.done = true
			return m, tea.Quit
		case "q", "ctrl+c", "esc":
			m.selected = "exit"
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m MenuModel) View() string {
	var b strings.Builder

	// ── Brand ─────────────────────────────────────────────
	b.WriteString(RenderBrand())
	b.WriteString("\n\n")

	// ── Title ─────────────────────────────────────────────
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F97316")).
		Padding(0, 2)

	b.WriteString(titleStyle.Render("¿Qué quieres hacer?"))
	b.WriteString("\n\n")

	// ── Menu items ────────────────────────────────────────
	for i, item := range m.items {
		// Cursor indicator
		cursor := "  "
		if i == m.cursor {
			cursor = "○ "
		}

		// Style based on selection
		if i == m.cursor {
			// Selected item
			itemStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#F97316")).
				Padding(0, 2)

			descStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D4D4D8")).
				Padding(0, 4)

			b.WriteString(fmt.Sprintf("%s%s\n", cursor, itemStyle.Render(item.Label)))
			if item.Description != "" {
				b.WriteString(descStyle.Render(item.Description))
				b.WriteString("\n")
			}
		} else {
			// Unselected item
			itemStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A1A1AA")).
				Padding(0, 2)

			b.WriteString(fmt.Sprintf("  %s\n", itemStyle.Render(item.Label)))
		}
		b.WriteString("\n")
	}

	// ── Help bar ──────────────────────────────────────────
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#52525B"))

	b.WriteString(helpStyle.Render("[↑/↓] navegar  [Enter] seleccionar  [q] salir"))

	return b.String()
}

// ── RunMenu ──────────────────────────────────────────────────────────────

// RunMainMenu displays the main menu and returns the selected option key.
// Returns "exit" if the user cancels.
func RunMainMenu() string {
	m := NewMainMenu()
	p := tea.NewProgram(m)

	final, err := p.Run()
	if err != nil {
		return "exit"
	}

	model := final.(MenuModel)
	if model.selected != "" {
		return model.selected
	}
	return "exit"
}
