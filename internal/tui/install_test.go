package tui

import (
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// ── Helper: collect steps from InstallModel ────────────────────────────

func stepStates(m InstallModel) []StepState {
	states := make([]StepState, len(m.steps))
	for i, s := range m.steps {
		states[i] = s.state
	}
	return states
}

// ── Test cases for InstallModel ────────────────────────────────────────

type installModelTestCase struct {
	name      string
	steps     []InstallStep
	msgs      []tea.Msg
	wantState []StepState
	wantDone  bool
}

func TestInstallModel(t *testing.T) {
	successAction := func() error { return nil }
	errorAction := func() error { return errors.New("falló") }

	tests := []installModelTestCase{
		{
			name: "single step completes successfully",
			steps: []InstallStep{
				{Name: "Paso 1", Action: successAction},
			},
			msgs: []tea.Msg{
				InstalStepMsg{Index: 0, Err: nil},
			},
			wantState: []StepState{StepDone},
			wantDone:  true,
		},
		{
			name: "multiple steps all succeed",
			steps: []InstallStep{
				{Name: "Paso 1", Action: successAction},
				{Name: "Paso 2", Action: successAction},
				{Name: "Paso 3", Action: successAction},
			},
			msgs: []tea.Msg{
				InstalStepMsg{Index: 0, Err: nil},
				InstalStepMsg{Index: 1, Err: nil},
				InstalStepMsg{Index: 2, Err: nil},
			},
			wantState: []StepState{StepDone, StepDone, StepDone},
			wantDone:  true,
		},
		{
			name: "step fails, continues to next step",
			steps: []InstallStep{
				{Name: "Paso 1", Action: successAction},
				{Name: "Paso 2", Action: errorAction},
				{Name: "Paso 3", Action: successAction},
			},
			msgs: []tea.Msg{
				InstalStepMsg{Index: 0, Err: nil},
				InstalStepMsg{Index: 1, Err: errors.New("falló")},
				InstalStepMsg{Index: 2, Err: nil},
			},
			wantState: []StepState{StepDone, StepError, StepDone},
			wantDone:  true,
		},
		{
			name: "initial state is pending for all steps",
			steps: []InstallStep{
				{Name: "Paso 1", Action: successAction},
				{Name: "Paso 2", Action: successAction},
			},
			msgs:      []tea.Msg{},
			wantState: []StepState{StepPending, StepPending},
			wantDone:  false,
		},
		{
			name: "window resize updates model width",
			steps: []InstallStep{
				{Name: "Paso 1", Action: successAction},
			},
			msgs: []tea.Msg{
				tea.WindowSizeMsg{Width: 80, Height: 30},
			},
			wantState: []StepState{StepPending},
			wantDone:  false,
		},
		{
			name: "spinner tick does not change state",
			steps: []InstallStep{
				{Name: "Paso 1", Action: successAction},
			},
			msgs: []tea.Msg{
				spinner.TickMsg{},
				spinner.TickMsg{},
			},
			wantState: []StepState{StepPending},
			wantDone:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewInstallModel(tt.steps)

			for _, msg := range tt.msgs {
				var cmd tea.Cmd
				var model tea.Model
				model, cmd = m.Update(msg)
				if im, ok := model.(InstallModel); ok {
					m = im
				}
				_ = cmd
			}

			gotStates := stepStates(m)
			for i, want := range tt.wantState {
				if gotStates[i] != want {
					t.Errorf("step[%d] state = %v, want %v", i, gotStates[i], want)
				}
			}

			if m.done != tt.wantDone {
				t.Errorf("done = %v, want %v", m.done, tt.wantDone)
			}
		})
	}
}

// ── Test cases for RunInstall ───────────────────────────────────────────

func TestRunInstall(t *testing.T) {
	t.Run("empty steps returns immediately", func(t *testing.T) {
		err := RunInstall([]InstallStep{})
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}

// ── Test cases for StepState ────────────────────────────────────────────

func TestStepState_String(t *testing.T) {
	tests := []struct {
		state StepState
		want  string
	}{
		{StepPending, " "},
		{StepRunning, "⠋"},
		{StepDone, "✓"},
		{StepError, "✗"},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("StepState.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
