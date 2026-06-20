package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/secko/zyrocli/internal/opencode"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// TUI states
// ---------------------------------------------------------------------------

type tuiState int

const (
	stateSelectAgent    tuiState = iota
	stateSelectProvider
	stateSelectModel
	stateSummary
)

// ---------------------------------------------------------------------------
// TUI styles (same aesthetic as before)
// ---------------------------------------------------------------------------

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED"))

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7C3AED")).
				Padding(0, 1)

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

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#71717A"))

	modelTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B"))

	phaseTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6366F1"))
)

// ---------------------------------------------------------------------------
// Assignment
// ---------------------------------------------------------------------------

// Assignment represents a completed agent→provider→model mapping.
type Assignment struct {
	AgentName  string
	Mode       string
	ProviderID string
	ModelID    string
}

// ---------------------------------------------------------------------------
// TUI model
// ---------------------------------------------------------------------------

type profileTuiModel struct {
	state        tuiState
	agents       []AgentDef
	agentIdx     int
	providers    []opencode.Provider
	providerIdx  int
	modelIdx     int
	assignments  []Assignment
	setAllMode   bool           // true when "Set All" was selected
	setAllProv   string         // provider id for Set All
	setAllModel  string         // model id for Set All
	done         bool
	cancelled    bool
	err          error
	width        int
	height       int
	lastContent  string
	currentAgent *AgentDef // agent currently being configured
}

func newProfileTUIModel(agents []AgentDef, providers []opencode.Provider) profileTuiModel {
	// Sort agents alphabetically, but keep orchestrator first.
	sorted := make([]AgentDef, len(agents))
	copy(sorted, agents)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Name == "zyro-orchestrator" {
			return true
		}
		if sorted[j].Name == "zyro-orchestrator" {
			return false
		}
		return sorted[i].Name < sorted[j].Name
	})

	return profileTuiModel{
		state:       stateSelectAgent,
		agents:      sorted,
		agentIdx:    0,
		providers:   providers,
		providerIdx: 0,
		modelIdx:    0,
		assignments: make([]Assignment, 0, len(sorted)),
	}
}

func (m profileTuiModel) Init() tea.Cmd { return nil }

func (m profileTuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			m.cancelled = true
			return m, tea.Quit
		}

		switch m.state {
		case stateSelectAgent:
			return m.updateAgentSelection(msg)
		case stateSelectProvider:
			return m.updateProviderSelection(msg)
		case stateSelectModel:
			return m.updateModelSelection(msg)
		case stateSummary:
			return m.updateSummary(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// ---------------------------------------------------------------------------
// State update handlers
// ---------------------------------------------------------------------------

func (m profileTuiModel) updateAgentSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	totalItems := len(m.agents) + 1 // +1 for "Set All"

	switch msg.String() {
	case "up", "k":
		if m.agentIdx > 0 {
			m.agentIdx--
		}
	case "down", "j":
		if m.agentIdx < totalItems-1 {
			m.agentIdx++
		}
	case "enter":
		if m.agentIdx == 0 {
			// "Set All" selected
			m.setAllMode = true
			m.state = stateSelectProvider
			m.providerIdx = 0
		} else {
			// Specific agent selected
			m.setAllMode = false
			m.currentAgent = &m.agents[m.agentIdx-1]
			m.state = stateSelectProvider
			m.providerIdx = 0
		}
	}

	return m, nil
}

func (m profileTuiModel) updateProviderSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		if len(m.providers) == 0 || len(m.providers[m.providerIdx].Models) == 0 {
			break
		}
		m.state = stateSelectModel
		m.modelIdx = 0
	case "b": // back to agent selection
		m.state = stateSelectAgent
	}

	return m, nil
}

func (m profileTuiModel) updateModelSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	provider := m.providers[m.providerIdx]

	switch msg.String() {
	case "up", "k":
		if m.modelIdx > 0 {
			m.modelIdx--
		}
	case "down", "j":
		if m.modelIdx < len(provider.Models)-1 {
			m.modelIdx++
		}
	case "enter":
		p := m.providers[m.providerIdx]
		model := p.Models[m.modelIdx]

		if m.setAllMode {
			// Save for Set All
			m.setAllProv = p.ID
			m.setAllModel = model.ID
			// Create assignments for all agents
			m.assignments = make([]Assignment, 0, len(m.agents))
			for _, a := range m.agents {
				m.assignments = append(m.assignments, Assignment{
					AgentName:  a.Name,
					Mode:       a.DefaultMode,
					ProviderID: p.ID,
					ModelID:    model.ID,
				})
			}
			m.state = stateSummary
		} else if m.currentAgent != nil {
			// Save individual assignment
			m.assignments = append(m.assignments, Assignment{
				AgentName:  m.currentAgent.Name,
				Mode:       m.currentAgent.DefaultMode,
				ProviderID: p.ID,
				ModelID:    model.ID,
			})
			// Go back to agent selection
			m.state = stateSelectAgent
		}
	case "b": // back to provider selection
		m.state = stateSelectProvider
	}

	return m, nil
}

func (m profileTuiModel) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.done = true
		return m, tea.Quit
	case "b":
		// Go back: if Set All, go to model; if individual, go to agent
		if m.setAllMode {
			m.state = stateSelectModel
		} else {
			m.state = stateSelectAgent
		}
	}

	return m, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m profileTuiModel) View() string {
	var b strings.Builder

	switch m.state {
	case stateSelectAgent:
		b.WriteString(renderAgentView(m))
	case stateSelectProvider:
		b.WriteString(renderProviderView(m))
	case stateSelectModel:
		b.WriteString(renderModelView(m))
	case stateSummary:
		b.WriteString(renderSummaryView(m))
	}

	content := b.String()
	if content == m.lastContent {
		return m.lastContent
	}
	m.lastContent = borderStyle.Render(content)
	return m.lastContent
}

// ---------------------------------------------------------------------------
// View renderers
// ---------------------------------------------------------------------------

func renderAgentView(m profileTuiModel) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("ZyroCLI — Selector de Modelos"))
	b.WriteString("\n\n")
	b.WriteString("Seleccioná un agente para configurar, o \"Set All\" para asignar el mismo modelo a todos:")
	b.WriteString("\n\n")

	// "Set All" as first option (index 0)
	setAllLabel := "★ Set All — Asignar el mismo modelo a TODOS los agentes"
	if m.agentIdx == 0 {
		b.WriteString(fmt.Sprintf(" ● %s\n", selectedItemStyle.Render(setAllLabel)))
	} else {
		b.WriteString(fmt.Sprintf(" ○  %s\n", setAllLabel))
	}

	// Agent list (index 1..n)
	for i, agent := range m.agents {
		idx := i + 1 // +1 because Set All is index 0
		bullet := "○  "
		style := bulletStyle
		if m.agentIdx == idx {
			bullet = " ● "
			style = selectedItemStyle
		}

		phaseTag := ""
		if agent.Phase != "" {
			phaseTag = fmt.Sprintf("[%s] ", phaseTagStyle.Render(agent.Phase))
		}

		row := fmt.Sprintf("%s %s%s",
			bullet,
			phaseTag,
			style.Render(agent.Name),
		)
		b.WriteString(row)
		b.WriteString("\n")

		// Show description and current model on next line (indented)
		if m.agentIdx == idx {
			// Selected: show description
			b.WriteString(fmt.Sprintf("     %s\n", descStyle.Render(agent.Description)))
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("[Enter] seleccionar  [↑/↓] navegar  [q] cancelar"))
	return b.String()
}

func renderProviderView(m profileTuiModel) string {
	var b strings.Builder

	context := ""
	if m.setAllMode {
		context = "★ Set All"
	} else if m.currentAgent != nil {
		context = m.currentAgent.Name
	}

	b.WriteString(titleStyle.Render("Paso 1: Proveedor"))
	b.WriteString("\n\n")
	b.WriteString(phaseLabelStyle.Render(context))
	b.WriteString(" → ¿Qué proveedor?")
	b.WriteString("\n\n")

	if len(m.providers) == 0 {
		b.WriteString(helpStyle.Render("No hay proveedores disponibles. Configurá tus API keys en OpenCode con /connect"))
		b.WriteString("\n")
	} else {
		for i, p := range m.providers {
			style := bulletStyle
			bullet := "○  "
			if i == m.providerIdx {
				bullet = " ● "
				style = selectedItemStyle
			}
			modelCount := fmt.Sprintf("%d modelos", len(p.Models))
			row := fmt.Sprintf("%s %s  %s  %s",
				bullet,
				style.Render(p.ID),
				p.Name,
				helpStyle.Render(modelCount),
			)
			b.WriteString(row)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("[Enter] seleccionar  [↑/↓] navegar  [b] volver  [q] cancelar"))
	return b.String()
}

func renderModelView(m profileTuiModel) string {
	var b strings.Builder

	context := ""
	if m.setAllMode {
		context = "★ Set All"
	} else if m.currentAgent != nil {
		context = m.currentAgent.Name
	}

	provider := m.providers[m.providerIdx]
	b.WriteString(titleStyle.Render("Paso 2: Modelo"))
	b.WriteString("\n\n")
	b.WriteString(phaseLabelStyle.Render(context))
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
	b.WriteString(helpStyle.Render("[Enter] seleccionar  [↑/↓] navegar  [b] volver  [q] cancelar"))
	return b.String()
}

func renderSummaryView(m profileTuiModel) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Resumen de asignaciones"))
	b.WriteString("\n\n")

	if len(m.assignments) == 0 {
		b.WriteString(helpStyle.Render("No hay asignaciones."))
		b.WriteString("\n")
	} else {
		// Group by provider/model for cleaner display
		grouped := make(map[string][]string)
		for _, a := range m.assignments {
			key := a.ProviderID + "/" + a.ModelID
			grouped[key] = append(grouped[key], a.AgentName)
		}

		for modelStr, agentNames := range grouped {
			b.WriteString(fmt.Sprintf("  %s\n", modelTagStyle.Render(modelStr)))
			for _, name := range agentNames {
				b.WriteString(fmt.Sprintf("    • %s\n", name))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("[Enter] confirmar y guardar  [b] volver  [q] cancelar"))
	return b.String()
}

// ---------------------------------------------------------------------------
// Public API: runProfileTUI is the cobra RunE for "profile tui".
// ---------------------------------------------------------------------------

func runProfileTUI(cmd *cobra.Command, args []string) error {
	// Check if running inside OpenCode first
	if IsInsideOpenCode() {
		cmd.Println("Estás dentro de OpenCode. Usá /zyro-model para el selector interactivo.")
		cmd.Println("O ejecutá 'zyrocli profile list' para ver asignaciones actuales.")
		return nil
	}

	providers, err := opencode.ReadProviders(opencode.GetDefaultPath())
	if err != nil {
		return fmt.Errorf("profile tui: loading providers: %w", err)
	}

	if len(providers) == 0 {
		providers = opencode.KnownProviders()
	}

	m := newProfileTUIModel(zyroAgents, providers)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return fmt.Errorf("profile tui: running TUI: %w", err)
	}

	m = final.(profileTuiModel)
	if m.cancelled || !m.done {
		cmd.Println("Perfil cancelado, no se guardaron cambios.")
		return nil
	}

	// Build agent configs and write to opencode.json.
	configs := make(map[string]opencode.AgentConfig, len(m.assignments))
	for _, a := range m.assignments {
		configs[a.AgentName] = opencode.AgentConfig{
			Model: a.ProviderID + "/" + a.ModelID,
			Mode:  a.Mode,
		}
	}

	path := opencode.GetDefaultPath()
	if err := opencode.WriteAgentConfig(path, "default", configs); err != nil {
		return fmt.Errorf("profile tui: writing config: %w", err)
	}

	cmd.Println("✓ Asignaciones de modelos guardadas en opencode.json")
	return nil
}
