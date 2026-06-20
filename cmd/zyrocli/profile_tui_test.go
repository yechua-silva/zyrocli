package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/secko/zyrocli/internal/opencode"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// testBaseAgents returns a minimal agent list for predictable tests.
func testBaseAgents() []AgentDef {
	return []AgentDef{
		{Name: "zyro-orchestrator", Description: "Coordinator", Phase: "", DefaultMode: "primary"},
		{Name: "zyro-sdd-apply", Description: "Implementation", Phase: "F3", DefaultMode: "subagent"},
		{Name: "zyro-sdd-verify", Description: "Verification", Phase: "F3", DefaultMode: "subagent"},
	}
}

// testBaseProviders returns a minimal provider list used for predictable tests.
func testBaseProviders() []opencode.Provider {
	return []opencode.Provider{
		{
			ID: "provider-a", Name: "Provider A",
			Models: []opencode.Model{
				{ID: "model-a1", Name: "Model A1"},
				{ID: "model-a2", Name: "Model A2"},
			},
		},
		{
			ID: "provider-b", Name: "Provider B",
			Models: []opencode.Model{
				{ID: "model-b1", Name: "Model B1"},
			},
		},
	}
}

// selectAgent sends Enter on agent selection to select the agent at current agentIdx.
func selectAgent(m profileTuiModel) profileTuiModel {
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(profileTuiModel)
}

// selectProvider sends Enter on provider selection.
func selectProvider(m profileTuiModel) profileTuiModel {
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(profileTuiModel)
}

// selectModel sends Enter on model selection to save the assignment.
func selectModel(m profileTuiModel) profileTuiModel {
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(profileTuiModel)
}

// completeOneAgent does the full flow: select agent → select provider → select model.
func completeOneAgent(m profileTuiModel) profileTuiModel {
	m = selectAgent(m)   // agent → provider
	m = selectProvider(m) // provider → model
	m = selectModel(m)    // model → saved, back to agent
	return m
}

// ---------------------------------------------------------------------------
// 1. Test initial state
// ---------------------------------------------------------------------------

func TestNewProfileTUIModel(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	if m.state != stateSelectAgent {
		t.Errorf("initial state = %d, want %d", m.state, stateSelectAgent)
	}
	if m.agentIdx != 0 {
		t.Errorf("initial agentIdx = %d, want 0", m.agentIdx)
	}
	if len(m.assignments) != 0 {
		t.Errorf("initial assignments = %d, want 0", len(m.assignments))
	}
	if m.done {
		t.Error("done should be false initially")
	}
	if m.cancelled {
		t.Error("cancelled should be false initially")
	}
}

// ---------------------------------------------------------------------------
// 2. Test agent selection transitions
// ---------------------------------------------------------------------------

func TestAgentSelectionSetAll(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// At agentIdx=0 (Set All), Enter → provider selection
	m = selectAgent(m)
	if m.state != stateSelectProvider {
		t.Errorf("state = %d, want %d", m.state, stateSelectProvider)
	}
	if !m.setAllMode {
		t.Error("setAllMode should be true when Set All selected")
	}
}

func TestAgentSelectionIndividual(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// Navigate to first agent (index 1 = zyro-orchestrator)
	m.agentIdx = 1
	m = selectAgent(m)
	if m.state != stateSelectProvider {
		t.Errorf("state = %d, want %d", m.state, stateSelectProvider)
	}
	if m.setAllMode {
		t.Error("setAllMode should be false for individual agent")
	}
	if m.currentAgent == nil {
		t.Fatal("currentAgent should be set")
	}
	if m.currentAgent.Name != "zyro-orchestrator" {
		t.Errorf("currentAgent.Name = %q, want %q", m.currentAgent.Name, "zyro-orchestrator")
	}
}

// ---------------------------------------------------------------------------
// 3. Test provider → model → assignment flow
// ---------------------------------------------------------------------------

func TestProviderSelectionGoesToModelList(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// Select Set All → provider selection
	m = selectAgent(m)
	if m.state != stateSelectProvider {
		t.Fatalf("expected provider state, got %d", m.state)
	}

	// Navigate to second provider
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(profileTuiModel)
	if m.providerIdx != 1 {
		t.Fatalf("providerIdx = %d, want 1", m.providerIdx)
	}

	// Enter selects provider-b, transitions to model selection
	m = selectProvider(m)
	if m.state != stateSelectModel {
		t.Fatalf("state = %d, want %d", m.state, stateSelectModel)
	}
}

func TestModelSelectionSavesAssignment(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// Set All: complete one full flow
	m = selectAgent(m)    // agent → provider
	m = selectProvider(m) // provider (default provider-a) → model

	// Check we're in model selection
	if m.state != stateSelectModel {
		t.Fatalf("expected model state, got %d", m.state)
	}

	// Select model-a1 (default)
	m = selectModel(m)

	// Should go to summary with N assignments (one per agent)
	if m.state != stateSummary {
		t.Errorf("state = %d, want %d (summary)", m.state, stateSummary)
	}
	if len(m.assignments) != 3 {
		t.Errorf("assignments = %d, want 3", len(m.assignments))
	}

	// Check first assignment
	a := m.assignments[0]
	if a.ProviderID != "provider-a" {
		t.Errorf("ProviderID = %q, want %q", a.ProviderID, "provider-a")
	}
	if a.ModelID != "model-a1" {
		t.Errorf("ModelID = %q, want %q", a.ModelID, "model-a1")
	}
}

func TestIndividualAssignmentReturnsToAgentSelection(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// Select individual agent (index 1 = zyro-orchestrator)
	m.agentIdx = 1
	m = selectAgent(m)    // agent → provider
	m = selectProvider(m) // provider → model
	m = selectModel(m)    // model → saved, back to agent

	if m.state != stateSelectAgent {
		t.Errorf("state = %d, want %d (should return to agent)", m.state, stateSelectAgent)
	}
	if len(m.assignments) != 1 {
		t.Errorf("assignments = %d, want 1", len(m.assignments))
	}
}

func TestMultipleIndividualAssignments(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// Assign first agent
	m.agentIdx = 1
	m = completeOneAgent(m)
	if len(m.assignments) != 1 {
		t.Fatalf("after first: assignments = %d, want 1", len(m.assignments))
	}

	// Assign second agent
	m.agentIdx = 2
	m = completeOneAgent(m)
	if len(m.assignments) != 2 {
		t.Fatalf("after second: assignments = %d, want 2", len(m.assignments))
	}
}

// ---------------------------------------------------------------------------
// 4. Test summary state
// ---------------------------------------------------------------------------

func TestSummaryStateReachedAfterSetAll(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// Set All flow
	m = selectAgent(m)
	m = selectProvider(m)
	m = selectModel(m)

	if m.state != stateSummary {
		t.Errorf("state = %d, want %d", m.state, stateSummary)
	}
}

func TestSummaryConfirmSetsDone(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// Set All, reach summary
	m = selectAgent(m)
	m = selectProvider(m)
	m = selectModel(m)

	if m.state != stateSummary {
		t.Fatalf("should be in summary, got state %d", m.state)
	}

	// Enter in summary confirms and quits
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(profileTuiModel)

	if !m.done {
		t.Error("done should be true after Enter in summary")
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd after summary confirm")
	}
}

// ---------------------------------------------------------------------------
// 5. Test cancel
// ---------------------------------------------------------------------------

func TestCancelInAgentSelection(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = next.(profileTuiModel)

	if !m.cancelled {
		t.Error("cancelled should be true after q")
	}
	if len(m.assignments) != 0 {
		t.Error("no assignments should exist after cancel")
	}
}

func TestCancelInProviderSelection(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// Go to provider selection
	m = selectAgent(m)

	// Cancel
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = next.(profileTuiModel)

	if !m.cancelled {
		t.Error("cancelled should be true after q in provider selection")
	}
}

func TestCancelInModelSelection(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// Go to model selection
	m = selectAgent(m)
	m = selectProvider(m)

	// Cancel
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = next.(profileTuiModel)

	if !m.cancelled {
		t.Error("cancelled should be true after q in model selection")
	}
}

func TestCancelInSummary(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// Reach summary
	m = selectAgent(m)
	m = selectProvider(m)
	m = selectModel(m)

	// Cancel
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = next.(profileTuiModel)

	if !m.cancelled {
		t.Error("cancelled should be true after q in summary")
	}
}

// ---------------------------------------------------------------------------
// 6. Test navigation boundaries
// ---------------------------------------------------------------------------

func TestAgentNavigationBoundaries(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())
	totalItems := len(m.agents) + 1 // +1 for Set All

	// Up at first item (Set All) should stay
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = next.(profileTuiModel)
	if m.agentIdx != 0 {
		t.Errorf("up at first: agentIdx = %d, want 0", m.agentIdx)
	}

	// Navigate to last item
	for i := 1; i < totalItems; i++ {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = next.(profileTuiModel)
	}
	if m.agentIdx != totalItems-1 {
		t.Errorf("after going down: agentIdx = %d, want %d", m.agentIdx, totalItems-1)
	}

	// Down at last should stay
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(profileTuiModel)
	if m.agentIdx != totalItems-1 {
		t.Errorf("down at last: agentIdx = %d, want %d", m.agentIdx, totalItems-1)
	}
}

func TestProviderNavigationBoundaries(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// Go to provider selection
	m = selectAgent(m)

	// Up at first provider should stay
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = next.(profileTuiModel)
	if m.providerIdx != 0 {
		t.Errorf("up at first: providerIdx = %d, want 0", m.providerIdx)
	}

	// Down should move
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(profileTuiModel)
	if m.providerIdx != 1 {
		t.Errorf("down: providerIdx = %d, want 1", m.providerIdx)
	}

	// Down at last should stay
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(profileTuiModel)
	if m.providerIdx != 1 {
		t.Errorf("down at last: providerIdx = %d, want 1", m.providerIdx)
	}
}

func TestModelNavigationBoundaries(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// Go to model selection
	m = selectAgent(m)
	m = selectProvider(m)

	// Up at first model should stay
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = next.(profileTuiModel)
	if m.modelIdx != 0 {
		t.Errorf("up at first: modelIdx = %d, want 0", m.modelIdx)
	}

	// Down should move
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(profileTuiModel)
	if m.modelIdx != 1 {
		t.Errorf("down: modelIdx = %d, want 1", m.modelIdx)
	}

	// Down at last should stay (provider-a has 2 models)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(profileTuiModel)
	if m.modelIdx != 1 {
		t.Errorf("down at last: modelIdx = %d, want 1", m.modelIdx)
	}
}

// ---------------------------------------------------------------------------
// 7. Test back navigation (b key)
// ---------------------------------------------------------------------------

func TestBackFromProviderToAgent(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	m = selectAgent(m) // agent → provider
	if m.state != stateSelectProvider {
		t.Fatalf("expected provider state, got %d", m.state)
	}

	// Press b to go back
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	m = next.(profileTuiModel)
	if m.state != stateSelectAgent {
		t.Errorf("state = %d, want %d (agent)", m.state, stateSelectAgent)
	}
}

func TestBackFromModelToProvider(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	m = selectAgent(m)
	m = selectProvider(m) // provider → model
	if m.state != stateSelectModel {
		t.Fatalf("expected model state, got %d", m.state)
	}

	// Press b to go back
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	m = next.(profileTuiModel)
	if m.state != stateSelectProvider {
		t.Errorf("state = %d, want %d (provider)", m.state, stateSelectProvider)
	}
}

// ---------------------------------------------------------------------------
// 8. Test view rendering
// ---------------------------------------------------------------------------

func TestViewRendersAgentView(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	view := m.View()
	checks := []string{
		"Selector de Modelos",
		"Set All",
		"zyro-orchestrator",
		"zyro-sdd-apply",
		"zyro-sdd-verify",
		"[Enter] seleccionar",
		"[q] cancelar",
	}
	for _, c := range checks {
		if !strings.Contains(view, c) {
			t.Errorf("agent view should contain %q", c)
		}
	}
}

func TestViewRendersProviderView(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	m = selectAgent(m) // agent → provider

	view := m.View()
	checks := []string{
		"Proveedor",
		"provider-a",
		"Provider A",
		"provider-b",
		"Provider B",
		"[Enter] seleccionar",
		"[b] volver",
	}
	for _, c := range checks {
		if !strings.Contains(view, c) {
			t.Errorf("provider view should contain %q", c)
		}
	}
}

func TestViewRendersModelView(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	m = selectAgent(m)
	m = selectProvider(m) // provider → model

	view := m.View()
	checks := []string{
		"Modelo",
		"provider-a",
		"model-a1",
		"model-a2",
		"Model A1",
		"[b] volver",
	}
	for _, c := range checks {
		if !strings.Contains(view, c) {
			t.Errorf("model view should contain %q", c)
		}
	}
}

func TestViewRendersSummary(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// Set All
	m = selectAgent(m)
	m = selectProvider(m)
	m = selectModel(m)

	view := m.View()
	checks := []string{
		"Resumen",
		"provider-a/model-a1",
		"[Enter] confirmar",
	}
	for _, c := range checks {
		if !strings.Contains(view, c) {
			t.Errorf("summary view should contain %q", c)
		}
	}
}

func TestViewEmptySummary(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())
	m.state = stateSummary
	m.assignments = nil

	view := m.View()
	if !strings.Contains(view, "No hay asignaciones") {
		t.Errorf("empty summary should show message, got: %s", view)
	}
}

// ---------------------------------------------------------------------------
// 9. Test confirm writes to opencode.json
// ---------------------------------------------------------------------------

func TestConfirmWritesConfigToFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// Set All: complete flow
	m = selectAgent(m)
	m = selectProvider(m)
	m = selectModel(m)

	// Build agent configs (same logic as runProfileTUI)
	configs := make(map[string]opencode.AgentConfig, len(m.assignments))
	for _, a := range m.assignments {
		configs[a.AgentName] = opencode.AgentConfig{
			Model: a.ProviderID + "/" + a.ModelID,
			Mode:  a.Mode,
		}
	}

	path := opencode.GetDefaultPath()
	if err := opencode.WriteAgentConfig(path, "default", configs); err != nil {
		t.Fatalf("WriteAgentConfig error: %v", err)
	}

	// Verify file exists and is valid JSON
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading config file: %v", err)
	}

	var parsed struct {
		Agent map[string]opencode.AgentConfig `json:"agent"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON written: %v", err)
	}

	// Verify each expected agent config
	for _, a := range m.assignments {
		cfg, ok := parsed.Agent[a.AgentName]
		if !ok {
			t.Errorf("agent %q not found in written config", a.AgentName)
			continue
		}
		wantModel := a.ProviderID + "/" + a.ModelID
		if cfg.Model != wantModel {
			t.Errorf("agent %q model = %q, want %q", a.AgentName, cfg.Model, wantModel)
		}
		if cfg.Mode != a.Mode {
			t.Errorf("agent %q mode = %q, want %q", a.AgentName, cfg.Mode, a.Mode)
		}
	}
}

// ---------------------------------------------------------------------------
// 10. Test empty providers
// ---------------------------------------------------------------------------

func TestEmptyProvidersList(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), []opencode.Provider{})

	if m.state != stateSelectAgent {
		t.Fatalf("state = %d, want %d", m.state, stateSelectAgent)
	}

	// Select agent (index 1)
	m.agentIdx = 1
	m = selectAgent(m)
	if m.state != stateSelectProvider {
		t.Fatalf("expected provider state, got %d", m.state)
	}

	// View should show error message
	view := m.View()
	if !strings.Contains(view, "No hay proveedores disponibles") {
		t.Errorf("empty providers view should contain error message, got: %s", view)
	}

	// Enter should not change state (guarded by len check)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(profileTuiModel)
	if m.state != stateSelectProvider {
		t.Errorf("state should remain stateSelectProvider when empty, got %d", m.state)
	}
}

// ---------------------------------------------------------------------------
// 11. Test navigation with j/k keys
// ---------------------------------------------------------------------------

func TestNavigationWithJKKeys(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// j moves down
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = next.(profileTuiModel)
	if m.agentIdx != 1 {
		t.Errorf("j: agentIdx = %d, want 1", m.agentIdx)
	}

	// k moves up
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = next.(profileTuiModel)
	if m.agentIdx != 0 {
		t.Errorf("k: agentIdx = %d, want 0", m.agentIdx)
	}
}

// ---------------------------------------------------------------------------
// 12. Test WindowSizeMsg
// ---------------------------------------------------------------------------

func TestWindowSizeMsg(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = next.(profileTuiModel)

	if m.width != 100 {
		t.Errorf("width = %d, want 100", m.width)
	}
	if m.height != 30 {
		t.Errorf("height = %d, want 30", m.height)
	}
}

// ---------------------------------------------------------------------------
// 13. Test model selection with specific model
// ---------------------------------------------------------------------------

func TestModelSelectionWithNavigation(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())

	// Agent → provider → model
	m = selectAgent(m)
	m = selectProvider(m)

	// Navigate to second model
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(profileTuiModel)
	if m.modelIdx != 1 {
		t.Fatalf("modelIdx = %d, want 1", m.modelIdx)
	}

	// Select model-a2
	m = selectModel(m)

	if len(m.assignments) == 0 {
		t.Fatalf("assignments = 0, expected some")
	}
	if m.assignments[0].ModelID != "model-a2" {
		t.Errorf("ModelID = %q, want %q", m.assignments[0].ModelID, "model-a2")
	}
}

// ---------------------------------------------------------------------------
// 14. Test escape key exits
// ---------------------------------------------------------------------------

func TestEscapeKeyExitsAllStates(t *testing.T) {
	tests := []struct {
		name  string
		setup func() profileTuiModel
	}{
		{
			name:  "esc in agent selection",
			setup: func() profileTuiModel { return newProfileTUIModel(testBaseAgents(), testBaseProviders()) },
		},
		{
			name: "esc in provider selection",
			setup: func() profileTuiModel {
				m := newProfileTUIModel(testBaseAgents(), testBaseProviders())
				return selectAgent(m)
			},
		},
		{
			name: "esc in model selection",
			setup: func() profileTuiModel {
				m := newProfileTUIModel(testBaseAgents(), testBaseProviders())
				m = selectAgent(m)
				return selectProvider(m)
			},
		},
		{
			name: "esc in summary",
			setup: func() profileTuiModel {
				m := newProfileTUIModel(testBaseAgents(), testBaseProviders())
				m = selectAgent(m)
				m = selectProvider(m)
				return selectModel(m)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()

			next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
			got := next.(profileTuiModel)

			if !got.cancelled {
				t.Error("cancelled should be true after esc")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 15. Test provider with no models does not advance
// ---------------------------------------------------------------------------

func TestProviderWithNoModelsDoesNotAdvance(t *testing.T) {
	providers := []opencode.Provider{
		{
			ID: "empty-provider", Name: "Empty Provider",
			Models: []opencode.Model{},
		},
	}
	m := newProfileTUIModel(testBaseAgents(), providers)

	// Select Set All → provider
	m = selectAgent(m)

	// Enter should not advance because provider has no models
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(profileTuiModel)
	if m.state != stateSelectProvider {
		t.Errorf("state = %d, want %d (should stay in provider)", m.state, stateSelectProvider)
	}
}

// ---------------------------------------------------------------------------
// 16. Test View includes bordered box
// ---------------------------------------------------------------------------

func TestViewIncludesBorderedBox(t *testing.T) {
	m := newProfileTUIModel(testBaseAgents(), testBaseProviders())
	view := m.View()

	if !strings.Contains(view, "─") {
		t.Error("view should contain border characters (rounded border)")
	}
}
