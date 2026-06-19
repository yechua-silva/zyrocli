package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/secko/zyrocli/internal/opencode"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Zyro-SDD phase definitions
// ---------------------------------------------------------------------------

// phaseTmpl is the immutable template for a Zyro-SDD phase.
type phaseTmpl struct {
	Phase   string // "zyro-sdd-planning"
	AgentID string // "sdd-orchestrator-zyro"
	Mode    string // "primary" | "subagent"
	Effort  string // "low" | "medium" | "high"
}

// zyroPhases lists all Zyro-SDD phases in order.
var zyroPhases = []phaseTmpl{
	{Phase: "zyro-sdd-explorer-stack", AgentID: "sdd-explore-zyro", Mode: "subagent", Effort: "high"},
	{Phase: "zyro-sdd-planning", AgentID: "sdd-orchestrator-zyro", Mode: "primary", Effort: "high"},
	{Phase: "zyro-sdd-implement", AgentID: "sdd-implement-zyro", Mode: "subagent", Effort: "low"},
	{Phase: "zyro-sdd-verify", AgentID: "sdd-verify-zyro", Mode: "subagent", Effort: "medium"},
}

// tuiState represents the current screen in the 2-step TUI flow.
type tuiState int

const (
	stateSelectProvider tuiState = iota
	stateSelectModel
	stateSummary
)

// PhaseAssignment represents a completed phase assignment with provider+model.
type PhaseAssignment struct {
	Phase      string
	AgentID    string
	Mode       string
	ProviderID string
	ModelID    string
	Effort     string
}

// profileTuiModel is the bubbletea model for the 2-step profile TUI.
type profileTuiModel struct {
	state       tuiState
	phases      []phaseTmpl
	phaseIdx    int
	providers   []opencode.Provider
	providerIdx int
	modelIdx    int
	assignments []PhaseAssignment
	width       int
	height      int
	done        bool
	cancelled   bool
	err         error
	lastContent string // cached inner content for View() optimization (G5)
}

// ---------------------------------------------------------------------------
// Lipgloss styles
// ---------------------------------------------------------------------------

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7C3AED")).
				Padding(0, 1)

	selectedBulletStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true)

	phaseLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#52525B"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(1, 2)

	bulletStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A1A1AA"))
)

// ---------------------------------------------------------------------------
// Model lifecycle
// ---------------------------------------------------------------------------

func newProfileTUIModel(providers []opencode.Provider) profileTuiModel {
	phases := make([]phaseTmpl, len(zyroPhases))
	copy(phases, zyroPhases)
	return profileTuiModel{
		state:       stateSelectProvider,
		phases:      phases,
		phaseIdx:    0,
		providers:   providers,
		providerIdx: 0,
		modelIdx:    0,
		assignments: make([]PhaseAssignment, 0, len(phases)),
	}
}

func (m profileTuiModel) Init() tea.Cmd {
	return nil
}

func (m profileTuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global quit keys — work in any state.
		switch msg.String() {
		case "q", "esc":
			m.cancelled = true
			return m, tea.Quit
		}

		switch m.state {
		case stateSelectProvider:
			switch msg.String() {
			case "up", "k":
				if m.providerIdx > 0 {
					m.providerIdx--
				}
			case "down", "j":
				if m.providerIdx < len(m.providers)-1 {
					m.providerIdx++
				}
			case "enter":
				if len(m.providers) == 0 {
					break
				}
				provider := m.providers[m.providerIdx]
				if len(provider.Models) == 0 {
					break
				}
				m.state = stateSelectModel
				m.modelIdx = 0
			}

		case stateSelectModel:
			switch msg.String() {
			case "up", "k":
				if m.modelIdx > 0 {
					m.modelIdx--
				}
			case "down", "j":
				provider := m.providers[m.providerIdx]
				if m.modelIdx < len(provider.Models)-1 {
					m.modelIdx++
				}
			case "enter":
				p := m.providers[m.providerIdx]
				current := m.phases[m.phaseIdx]
				m.assignments = append(m.assignments, PhaseAssignment{
					Phase:      current.Phase,
					AgentID:    current.AgentID,
					Mode:       current.Mode,
					Effort:     current.Effort,
					ProviderID: p.ID,
					ModelID:    p.Models[m.modelIdx].ID,
				})
				m.phaseIdx++
				if m.phaseIdx >= len(m.phases) {
					m.state = stateSummary
				} else {
					m.state = stateSelectProvider
					m.providerIdx = 0
					m.modelIdx = 0
				}
			}

		case stateSummary:
			switch msg.String() {
			case "enter":
				m.done = true
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m profileTuiModel) View() string {
	var b strings.Builder

	switch m.state {
	case stateSelectProvider:
		b.WriteString(renderProviderView(m))
	case stateSelectModel:
		b.WriteString(renderModelView(m))
	case stateSummary:
		b.WriteString(renderSummaryView(m))
	}

	content := b.String()
	if content == m.lastContent {
		// Content unchanged — return cached rendered view
		return m.lastContent
	}

	m.lastContent = borderStyle.Render(content)
	return m.lastContent
}

// ---------------------------------------------------------------------------
// View renderers
// ---------------------------------------------------------------------------

func renderProviderView(m profileTuiModel) string {
	var b strings.Builder

	phase := m.phases[m.phaseIdx]
	b.WriteString(titleStyle.Render("Profile TUI — Paso 1: Provider"))
	b.WriteString("\n\n")
	b.WriteString(phaseLabelStyle.Render(phase.Phase))
	b.WriteString(" → ")
	b.WriteString("¿Qué proveedor?")
	b.WriteString("\n\n")

	if len(m.providers) == 0 {
		b.WriteString(helpStyle.Render("No hay proveedores disponibles."))
		b.WriteString("\n")
	} else {
		for i, p := range m.providers {
			style := bulletStyle
			bullet := "○  "
			if i == m.providerIdx {
				bullet = " ● "
				style = selectedItemStyle
			}
			row := fmt.Sprintf("%s %s  %s",
				bullet,
				style.Render(p.ID),
				p.Name,
			)
			b.WriteString(row)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("[Enter] seleccionar  [↑/↓] navegar  [q] cancelar"))

	return b.String()
}

func renderModelView(m profileTuiModel) string {
	var b strings.Builder

	phase := m.phases[m.phaseIdx]
	provider := m.providers[m.providerIdx]
	b.WriteString(titleStyle.Render("Profile TUI — Paso 2: Modelo"))
	b.WriteString("\n\n")
	b.WriteString(phaseLabelStyle.Render(phase.Phase))
	b.WriteString(" → ")
	b.WriteString(provider.ID)
	b.WriteString("\n\n")

	for i, model := range provider.Models {
		style := bulletStyle
		bullet := "○  "
		if i == m.modelIdx {
			bullet = " ● "
			style = selectedItemStyle
		}
		row := fmt.Sprintf("%s %s  %s",
			bullet,
			style.Render(model.ID),
			model.Name,
		)
		b.WriteString(row)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("[Enter] seleccionar  [↑/↓] navegar  [q] cancelar"))

	return b.String()
}

func renderSummaryView(m profileTuiModel) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Profile TUI — Resumen"))
	b.WriteString("\n\n")

	if len(m.assignments) == 0 {
		b.WriteString(helpStyle.Render("No hay asignaciones."))
		b.WriteString("\n")
	} else {
		for _, a := range m.assignments {
			modelLabel := a.ProviderID + "/" + a.ModelID
			row := fmt.Sprintf("%s  %s",
				phaseLabelStyle.Render(a.Phase),
				modelLabel,
			)
			b.WriteString(row)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("[Enter] confirmar y guardar  [q] cancelar"))

	return b.String()
}

// ---------------------------------------------------------------------------
// Public API: runProfileTUI is the cobra RunE for "profile tui".
// ---------------------------------------------------------------------------

func runProfileTUI(cmd *cobra.Command, args []string) error {
	providers, err := opencode.ReadProviders(opencode.GetDefaultPath())
	if err != nil {
		return fmt.Errorf("profile tui: loading providers: %w", err)
	}

	m := newProfileTUIModel(providers)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return fmt.Errorf("profile tui: running TUI: %w", err)
	}

	m = final.(profileTuiModel)
	if m.cancelled || !m.done {
		cmd.Println("Profile cancelled, no changes saved.")
		return nil
	}

	// Build agent configs and write to opencode.json.
	configs := make(map[string]opencode.AgentConfig, len(m.assignments))
	for _, a := range m.assignments {
		configs[a.AgentID] = opencode.AgentConfig{
			Model:           a.ProviderID + "/" + a.ModelID,
			Mode:            a.Mode,
			ReasoningEffort: a.Effort,
		}
	}

	path := opencode.GetDefaultPath()
	if err := opencode.WriteAgentConfig(path, "default", configs); err != nil {
		return fmt.Errorf("profile tui: writing config: %w", err)
	}

	cmd.Println("✓ Model assignments saved to opencode.json")
	return nil
}
