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

// advancePhase sends Enter on provider selection, then Enter on model selection.
// It returns the model after completing one full phase.
func advancePhase(m profileTuiModel) profileTuiModel {
	// Enter on provider selection transitions to model selection.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(profileTuiModel)
	// Enter on model selection saves the assignment and advances.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(profileTuiModel)
}

// ---------------------------------------------------------------------------
// 1. Test state transitions (init → selectProvider → selectModel → next phase)
// ---------------------------------------------------------------------------

func TestNewProfileTUIModel(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	if m.state != stateSelectProvider {
		t.Errorf("initial state = %d, want %d", m.state, stateSelectProvider)
	}
	if m.phaseIdx != 0 {
		t.Errorf("initial phaseIdx = %d, want 0", m.phaseIdx)
	}
	if len(m.phases) != 4 {
		t.Errorf("len(phases) = %d, want 4", len(m.phases))
	}
	if len(m.assignments) != 0 {
		t.Errorf("initial assignments = %d, want 0", len(m.assignments))
	}
	if m.providerIdx != 0 {
		t.Errorf("initial providerIdx = %d, want 0", m.providerIdx)
	}
	if m.done {
		t.Error("done should be false initially")
	}
	if m.cancelled {
		t.Error("cancelled should be false initially")
	}

	// Verify correct phase names in order.
	wantPhases := []string{
		"zyro-sdd-explorer-stack",
		"zyro-sdd-planning",
		"zyro-sdd-implement",
		"zyro-sdd-verify",
	}
	for i, p := range m.phases {
		if p.Phase != wantPhases[i] {
			t.Errorf("phases[%d].Phase = %q, want %q", i, p.Phase, wantPhases[i])
		}
	}
}

func TestStateTransitions(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	// Initial state.
	if m.state != stateSelectProvider {
		t.Errorf("initial state = %d, want %d", m.state, stateSelectProvider)
	}

	// Enter on provider selection → stateSelectModel.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(profileTuiModel)
	if m.state != stateSelectModel {
		t.Errorf("after provider Enter: state = %d, want %d", m.state, stateSelectModel)
	}

	// Enter on model selection → saves assignment, advances phase.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(profileTuiModel)
	if m.state != stateSelectProvider {
		t.Errorf("after model Enter: state = %d, want %d", m.state, stateSelectProvider)
	}
	if m.phaseIdx != 1 {
		t.Errorf("after phase 0: phaseIdx = %d, want 1", m.phaseIdx)
	}
	if len(m.assignments) != 1 {
		t.Errorf("after phase 0: len(assignments) = %d, want 1", len(m.assignments))
	}
}

// ---------------------------------------------------------------------------
// 2. Test provider selection updates model list
// ---------------------------------------------------------------------------

func TestProviderSelectionGoesToModelList(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	// Navigate to second provider.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(profileTuiModel)
	if m.providerIdx != 1 {
		t.Fatalf("providerIdx = %d, want 1", m.providerIdx)
	}

	// Enter selects provider-b, transitions to model selection.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(profileTuiModel)
	if m.state != stateSelectModel {
		t.Fatalf("state = %d, want %d", m.state, stateSelectModel)
	}

	// Verify model list belongs to provider-b (1 model).
	provider := m.providers[m.providerIdx]
	if len(provider.Models) != 1 {
		t.Errorf("provider-b models = %d, want 1", len(provider.Models))
	}
	if provider.ID != "provider-b" {
		t.Errorf("provider ID = %q, want %q", provider.ID, "provider-b")
	}
}

// ---------------------------------------------------------------------------
// 3. Test model selection saves assignment
// ---------------------------------------------------------------------------

func TestModelSelectionSavesAssignment(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	// Complete phase 0 with provider-a / model-a1 (defaults).
	m = advancePhase(m)

	if len(m.assignments) != 1 {
		t.Fatalf("assignments = %d, want 1", len(m.assignments))
	}

	a := m.assignments[0]
	if a.Phase != "zyro-sdd-explorer-stack" {
		t.Errorf("Phase = %q, want %q", a.Phase, "zyro-sdd-explorer-stack")
	}
	if a.ProviderID != "provider-a" {
		t.Errorf("ProviderID = %q, want %q", a.ProviderID, "provider-a")
	}
	if a.ModelID != "model-a1" {
		t.Errorf("ModelID = %q, want %q", a.ModelID, "model-a1")
	}
	if a.Mode != "subagent" {
		t.Errorf("Mode = %q, want %q", a.Mode, "subagent")
	}
}

func TestModelSelectionRespectsProviderChoice(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	// Navigate to second provider.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(profileTuiModel)

	// Select provider-b.
	m = advancePhase(m)

	if len(m.assignments) != 1 {
		t.Fatalf("assignments = %d, want 1", len(m.assignments))
	}

	a := m.assignments[0]
	if a.ProviderID != "provider-b" {
		t.Errorf("ProviderID = %q, want %q", a.ProviderID, "provider-b")
	}
	if a.ModelID != "model-b1" {
		t.Errorf("ModelID = %q, want %q", a.ModelID, "model-b1")
	}
}

func TestModelSelectionPreservesEffortAndAgentID(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	// Phase 0: zyro-sdd-explorer-stack → sdd-explore-zyro, subagent, high.
	m = advancePhase(m)

	a := m.assignments[0]
	if a.AgentID != "sdd-explore-zyro" {
		t.Errorf("AgentID = %q, want %q", a.AgentID, "sdd-explore-zyro")
	}
	if a.Effort != "high" {
		t.Errorf("Effort = %q, want %q", a.Effort, "high")
	}

	// Phase 1: zyro-sdd-planning → sdd-orchestrator-zyro, primary, high.
	m = advancePhase(m)
	a = m.assignments[1]
	if a.AgentID != "sdd-orchestrator-zyro" {
		t.Errorf("AgentID = %q, want %q", a.AgentID, "sdd-orchestrator-zyro")
	}
	if a.Mode != "primary" {
		t.Errorf("Mode = %q, want %q", a.Mode, "primary")
	}
	if a.Effort != "high" {
		t.Errorf("Effort = %q, want %q", a.Effort, "high")
	}

	// Phase 2: zyro-sdd-implement → sdd-implement-zyro, subagent, low.
	m = advancePhase(m)
	a = m.assignments[2]
	if a.AgentID != "sdd-implement-zyro" {
		t.Errorf("AgentID = %q, want %q", a.AgentID, "sdd-implement-zyro")
	}
	if a.Mode != "subagent" {
		t.Errorf("Mode = %q, want %q", a.Mode, "subagent")
	}
	if a.Effort != "low" {
		t.Errorf("Effort = %q, want %q", a.Effort, "low")
	}

	// Phase 3: zyro-sdd-verify → sdd-verify-zyro, subagent, medium.
	m = advancePhase(m)
	a = m.assignments[3]
	if a.AgentID != "sdd-verify-zyro" {
		t.Errorf("AgentID = %q, want %q", a.AgentID, "sdd-verify-zyro")
	}
	if a.Mode != "subagent" {
		t.Errorf("Mode = %q, want %q", a.Mode, "subagent")
	}
	if a.Effort != "medium" {
		t.Errorf("Effort = %q, want %q", a.Effort, "medium")
	}
}

// ---------------------------------------------------------------------------
// 4. Test summary renders all phases
// ---------------------------------------------------------------------------

func TestSummaryStateReachedAfterFourPhases(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	for i := 0; i < 4; i++ {
		m = advancePhase(m)
	}

	if m.state != stateSummary {
		t.Errorf("state = %d, want %d (stateSummary)", m.state, stateSummary)
	}
	if len(m.assignments) != 4 {
		t.Errorf("assignments = %d, want 4", len(m.assignments))
	}
}

func TestSummaryViewRendersAllPhases(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())
	for i := 0; i < 4; i++ {
		m = advancePhase(m)
	}

	view := m.View()

	checks := []string{
		"zyro-sdd-explorer-stack",
		"zyro-sdd-planning",
		"zyro-sdd-implement",
		"zyro-sdd-verify",
		"provider-a/model-a1",
		"Resumen",
		"[Enter] confirmar",
	}
	for _, c := range checks {
		if !strings.Contains(view, c) {
			t.Errorf("summary view should contain %q", c)
		}
	}
}

// ---------------------------------------------------------------------------
// 5. Test confirm writes to opencode.json (con archivo temporal)
// ---------------------------------------------------------------------------

func TestConfirmWritesConfigToFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	m := newProfileTUIModel(testBaseProviders())

	// Complete all 4 phases.
	for i := 0; i < 4; i++ {
		m = advancePhase(m)
	}

	// Build agent configs (same logic as runProfileTUI).
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
		t.Fatalf("WriteAgentConfig error: %v", err)
	}

	// Verify file exists and is valid JSON.
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

	// Verify each expected agent config.
	for _, a := range m.assignments {
		cfg, ok := parsed.Agent[a.AgentID]
		if !ok {
			t.Errorf("agent %q not found in written config", a.AgentID)
			continue
		}
		wantModel := a.ProviderID + "/" + a.ModelID
		if cfg.Model != wantModel {
			t.Errorf("agent %q model = %q, want %q", a.AgentID, cfg.Model, wantModel)
		}
		if cfg.Mode != a.Mode {
			t.Errorf("agent %q mode = %q, want %q", a.AgentID, cfg.Mode, a.Mode)
		}
		if cfg.ReasoningEffort != a.Effort {
			t.Errorf("agent %q reasoningEffort = %q, want %q", a.AgentID, cfg.ReasoningEffort, a.Effort)
		}
	}
}

func TestConfirmDoesNotWriteWhenCancelled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Set state to summary as if all phases completed.
	m := newProfileTUIModel(testBaseProviders())
	m.state = stateSummary

	// Cancel instead of confirm.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = next.(profileTuiModel)

	if !m.cancelled {
		t.Error("cancelled should be true")
	}

	// No config should have been written.
	path := opencode.GetDefaultPath()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("config file should not exist after cancel")
	}
}

// ---------------------------------------------------------------------------
// 6. Test cancel returns without writing
// ---------------------------------------------------------------------------

func TestCancelInProviderSelection(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	// Cancel with q.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = next.(profileTuiModel)

	if !m.cancelled {
		t.Error("cancelled should be true after q")
	}
	if m.done {
		t.Error("done should be false after cancel")
	}
	if len(m.assignments) != 0 {
		t.Error("no assignments should exist after cancel")
	}
}

func TestCancelInModelSelection(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	// Go to model selection.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(profileTuiModel)

	// Cancel with q.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = next.(profileTuiModel)

	if !m.cancelled {
		t.Error("cancelled should be true after q in model selection")
	}
	if len(m.assignments) != 0 {
		t.Error("no assignments should exist after cancel in model selection")
	}
}

func TestCancelInSummary(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	// Complete all 4 phases to get to summary.
	for i := 0; i < 4; i++ {
		m = advancePhase(m)
	}

	if m.state != stateSummary {
		t.Fatalf("should be in summary state, got %d", m.state)
	}

	// Cancel with q.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = next.(profileTuiModel)

	if !m.cancelled {
		t.Error("cancelled should be true after q in summary")
	}
}

// ---------------------------------------------------------------------------
// 7. Test navigation boundaries (↑ at first, ↓ at last)
// ---------------------------------------------------------------------------

func TestProviderNavigationBoundaries(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	// At first provider (idx=0), up should stay at 0.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = next.(profileTuiModel)
	if m.providerIdx != 0 {
		t.Errorf("up at first provider: providerIdx = %d, want 0", m.providerIdx)
	}

	// Down should move to 1 (we have 2 providers).
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(profileTuiModel)
	if m.providerIdx != 1 {
		t.Errorf("down from first: providerIdx = %d, want 1", m.providerIdx)
	}

	// Down at last provider should stay.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(profileTuiModel)
	if m.providerIdx != 1 {
		t.Errorf("down at last: providerIdx = %d, want 1", m.providerIdx)
	}
}

func TestModelNavigationBoundaries(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	// Go to model selection (provider-a has 2 models).
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(profileTuiModel)
	if m.state != stateSelectModel {
		t.Fatalf("expected stateSelectModel, got %d", m.state)
	}

	// At first model (idx=0), up should stay at 0.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = next.(profileTuiModel)
	if m.modelIdx != 0 {
		t.Errorf("up at first model: modelIdx = %d, want 0", m.modelIdx)
	}

	// Down should move to 1.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(profileTuiModel)
	if m.modelIdx != 1 {
		t.Errorf("down from first: modelIdx = %d, want 1", m.modelIdx)
	}

	// Down at last model should stay.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(profileTuiModel)
	if m.modelIdx != 1 {
		t.Errorf("down at last: modelIdx = %d, want 1", m.modelIdx)
	}
}

// ---------------------------------------------------------------------------
// 8. Test q key exits
// ---------------------------------------------------------------------------

func TestQKeyExitsAllStates(t *testing.T) {
	tests := []struct {
		name  string
		setup func() profileTuiModel
	}{
		{
			name:  "q in provider selection",
			setup: func() profileTuiModel { return newProfileTUIModel(testBaseProviders()) },
		},
		{
			name: "q in model selection",
			setup: func() profileTuiModel {
				base := newProfileTUIModel(testBaseProviders())
				next, _ := base.Update(tea.KeyMsg{Type: tea.KeyEnter})
				return next.(profileTuiModel)
			},
		},
		{
			name: "q in summary",
			setup: func() profileTuiModel {
				base := newProfileTUIModel(testBaseProviders())
				for i := 0; i < 4; i++ {
					base = advancePhase(base)
				}
				return base
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()

			next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
			got := next.(profileTuiModel)

			if !got.cancelled {
				t.Error("cancelled should be true after q")
			}
		})
	}
}

func TestEscapeKeyExitsAllStates(t *testing.T) {
	tests := []struct {
		name  string
		setup func() profileTuiModel
	}{
		{
			name: "esc in provider selection",
			setup: func() profileTuiModel { return newProfileTUIModel(testBaseProviders()) },
		},
		{
			name: "esc in model selection",
			setup: func() profileTuiModel {
				base := newProfileTUIModel(testBaseProviders())
				next, _ := base.Update(tea.KeyMsg{Type: tea.KeyEnter})
				return next.(profileTuiModel)
			},
		},
		{
			name: "esc in summary",
			setup: func() profileTuiModel {
				base := newProfileTUIModel(testBaseProviders())
				for i := 0; i < 4; i++ {
					base = advancePhase(base)
				}
				return base
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
// 9. Test all 4 phases complete in order
// ---------------------------------------------------------------------------

func TestAllPhasesCompleteInOrder(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	expectedPhases := []string{
		"zyro-sdd-explorer-stack",
		"zyro-sdd-planning",
		"zyro-sdd-implement",
		"zyro-sdd-verify",
	}

	for i := 0; i < 4; i++ {
		m = advancePhase(m)

		// After phases 0-2, we go back to provider selection.
		// After phase 3, we go to summary.
		if i < 3 {
			if m.state != stateSelectProvider {
				t.Errorf("after phase %d: state = %d, want %d (stateSelectProvider)",
					i, m.state, stateSelectProvider)
			}
			if m.phaseIdx != i+1 {
				t.Errorf("after phase %d: phaseIdx = %d, want %d",
					i, m.phaseIdx, i+1)
			}
		} else {
			if m.state != stateSummary {
				t.Errorf("after phase %d: state = %d, want %d (stateSummary)",
					i, m.state, stateSummary)
			}
		}

		if len(m.assignments) != i+1 {
			t.Errorf("after phase %d: assignments = %d, want %d",
				i, len(m.assignments), i+1)
		}

		// Verify the latest assignment has the correct phase name.
		a := m.assignments[i]
		if a.Phase != expectedPhases[i] {
			t.Errorf("assignment[%d].Phase = %q, want %q",
				i, a.Phase, expectedPhases[i])
		}
	}
}

func TestSummaryConfirmSetsDone(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	// Complete all 4 phases to reach summary.
	for i := 0; i < 4; i++ {
		m = advancePhase(m)
	}

	if m.state != stateSummary {
		t.Fatalf("should be in summary, got state %d", m.state)
	}

	// Enter in summary confirms and quits.
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(profileTuiModel)

	if !m.done {
		t.Error("done should be true after Enter in summary")
	}
	if m.cancelled {
		t.Error("cancelled should be false after Enter in summary")
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd after summary confirm")
	}
}

// ---------------------------------------------------------------------------
// 10. Test empty providers list shows error state
// ---------------------------------------------------------------------------

func TestEmptyProvidersList(t *testing.T) {
	m := newProfileTUIModel([]opencode.Provider{})

	if m.state != stateSelectProvider {
		t.Fatalf("state = %d, want %d", m.state, stateSelectProvider)
	}

	// View should show error message.
	view := m.View()
	if !strings.Contains(view, "No hay proveedores disponibles") {
		t.Errorf("empty providers view should contain error message, got: %s", view)
	}

	// Enter should not change state (guarded by len check).
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(profileTuiModel)
	if m.state != stateSelectProvider {
		t.Errorf("state should remain stateSelectProvider when empty, got %d", m.state)
	}
	if len(m.assignments) != 0 {
		t.Errorf("assignments should be 0 when empty, got %d", len(m.assignments))
	}
}

func TestEmptyProvidersNilSlice(t *testing.T) {
	m := newProfileTUIModel(nil)

	if m.state != stateSelectProvider {
		t.Fatalf("state = %d, want %d", m.state, stateSelectProvider)
	}

	// View should show error message.
	view := m.View()
	if !strings.Contains(view, "No hay proveedores disponibles") {
		t.Errorf("empty providers view should contain error message, got: %s", view)
	}

	// Navigation should not crash.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = next.(profileTuiModel)
	if m.providerIdx != 0 {
		t.Errorf("providerIdx should stay 0 when empty, got %d", m.providerIdx)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(profileTuiModel)
	if m.providerIdx != 0 {
		t.Errorf("providerIdx should stay 0 when empty, got %d", m.providerIdx)
	}
}

// ---------------------------------------------------------------------------
// Additional: navigation with j/k keys
// ---------------------------------------------------------------------------

func TestNavigationWithJKKeys(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	// j moves down.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = next.(profileTuiModel)
	if m.providerIdx != 1 {
		t.Errorf("j: providerIdx = %d, want 1", m.providerIdx)
	}

	// k moves up.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = next.(profileTuiModel)
	if m.providerIdx != 0 {
		t.Errorf("k: providerIdx = %d, want 0", m.providerIdx)
	}
}

// ---------------------------------------------------------------------------
// Additional: model selection picks specific model index
// ---------------------------------------------------------------------------

func TestModelSelectionWithNavigation(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	// Go to model selection (provider-a).
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(profileTuiModel)

	// Navigate to second model.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(profileTuiModel)
	if m.modelIdx != 1 {
		t.Fatalf("modelIdx = %d, want 1", m.modelIdx)
	}

	// Select model-a2.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(profileTuiModel)

	if len(m.assignments) != 1 {
		t.Fatalf("assignments = %d, want 1", len(m.assignments))
	}
	if m.assignments[0].ModelID != "model-a2" {
		t.Errorf("ModelID = %q, want %q", m.assignments[0].ModelID, "model-a2")
	}
}

// ---------------------------------------------------------------------------
// Additional: WindowSizeMsg
// ---------------------------------------------------------------------------

func TestWindowSizeMsg(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

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
// Additional: View rendering checks
// ---------------------------------------------------------------------------

func TestViewRendersProviderView(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	view := m.View()

	checks := []string{
		"Paso 1: Provider",
		"zyro-sdd-explorer-stack",
		"¿Qué proveedor?",
		"provider-a",
		"provider-b",
		"Provider A",
		"Provider B",
		"[Enter] seleccionar",
		"[↑/↓] navegar",
		"[q] cancelar",
	}
	for _, c := range checks {
		if !strings.Contains(view, c) {
			t.Errorf("provider view should contain %q", c)
		}
	}

	// First provider should be selected (has " ● " marker).
	if !strings.Contains(view, " ● ") {
		t.Error("provider view should show a selected item marker")
	}
}

func TestViewRendersModelView(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())

	// Go to model selection for provider-a (2 models).
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(profileTuiModel)

	view := m.View()

	checks := []string{
		"Paso 2: Modelo",
		"zyro-sdd-explorer-stack",
		"provider-a",
		"model-a1",
		"model-a2",
		"Model A1",
		"Model A2",
		"[Enter] seleccionar",
	}
	for _, c := range checks {
		if !strings.Contains(view, c) {
			t.Errorf("model view should contain %q", c)
		}
	}
}

func TestViewRendersSummaryAfterConfirm(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())
	for i := 0; i < 4; i++ {
		m = advancePhase(m)
	}

	view := m.View()

	checks := []string{
		"Resumen",
		"provider-a/model-a1",
		"[Enter] confirmar y guardar",
		"[q] cancelar",
	}
	for _, c := range checks {
		if !strings.Contains(view, c) {
			t.Errorf("summary view should contain %q", c)
		}
	}
}

func TestViewIncludesBorderedBox(t *testing.T) {
	m := newProfileTUIModel(testBaseProviders())
	view := m.View()

	// The border style uses rounded corners (╭─, ╰─, etc.).
	if !strings.Contains(view, "─") {
		t.Error("view should contain border characters (rounded border)")
	}
}

func TestViewEmptySummary(t *testing.T) {
	// If summary state is reached with no assignments (edge case),
	// should show a helpful message.
	m := newProfileTUIModel(testBaseProviders())
	m.state = stateSummary
	m.assignments = nil

	view := m.View()
	if !strings.Contains(view, "No hay asignaciones") {
		t.Errorf("empty summary should show message, got: %s", view)
	}
}

// ---------------------------------------------------------------------------
// Additional: provider with no models should not advance
// ---------------------------------------------------------------------------

func TestProviderWithNoModelsDoesNotAdvance(t *testing.T) {
	providers := []opencode.Provider{
		{
			ID: "empty-provider", Name: "Empty Provider",
			Models: []opencode.Model{},
		},
	}
	m := newProfileTUIModel(providers)

	// Enter should not advance because provider has no models.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(profileTuiModel)
	if m.state != stateSelectProvider {
		t.Errorf("state = %d, want %d (should stay in provider selection)", m.state, stateSelectProvider)
	}
}
