package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Step state machine ──────────────────────────────────────────────────

// StepState represents the current state of an installation step.
type StepState int

const (
	// StepPending means the step hasn't started yet.
	StepPending StepState = iota
	// StepRunning means the step is currently executing (spinner visible).
	StepRunning
	// StepDone means the step completed successfully.
	StepDone
	// StepError means the step failed.
	StepError
)

// String returns a visual representation of the step state.
func (s StepState) String() string {
	switch s {
	case StepPending:
		return " "
	case StepRunning:
		return "⠋"
	case StepDone:
		return "✓"
	case StepError:
		return "✗"
	default:
		return "?"
	}
}

// ── InstallStep ─────────────────────────────────────────────────────────

// InstallStep defines a single step in the installation process.
type InstallStep struct {
	// Name is the descriptive title (e.g. "Extracting MCP tools").
	Name string

	// Action is the function that performs the actual work.
	// It runs in a goroutine. Returns error on failure.
	Action func() error

	// state is updated internally: pending → running → done | error.
	state StepState

	// err stores the error if state == StepError.
	err error
}

// ── InstallModel ────────────────────────────────────────────────────────

// InstallModel is the main bubbletea model for the multi-step installer.
type InstallModel struct {
	// steps is the ordered list of installation steps.
	steps []*InstallStep

	// currentIdx is the index of the currently running step.
	currentIdx int

	// spinner is the animated spinner component.
	spinner spinner.Model

	// progress is the optional global progress bar.
	progress progress.Model

	// width and height are updated via WindowSizeMsg.
	width  int
	height int

	// err accumulates fatal errors.
	err error

	// done indicates the installer has finished.
	done bool
}

// InstalStepMsg is sent when a step completes execution.
type InstalStepMsg struct {
	// Index is the step index in the steps slice.
	Index int
	// Err is nil on success, or the error returned by Action.
	Err error
}

// NewInstallModel creates a new InstallModel with the given steps.
func NewInstallModel(steps []InstallStep) InstallModel {
	// Convert to pointers for mutable state
	ptrs := make([]*InstallStep, len(steps))
	for i := range steps {
		ptrs[i] = &steps[i]
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F97316")) // Zyro orange

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(50),
	)

	return InstallModel{
		steps:      ptrs,
		currentIdx: 0,
		spinner:    s,
		progress:   p,
		width:      80,
		height:     24,
	}
}

// ── tea.Model implementation ────────────────────────────────────────────

// Init starts the first step and the spinner.
func (m InstallModel) Init() tea.Cmd {
	if len(m.steps) == 0 {
		m.done = true
		return tea.Quit
	}

	m.steps[0].state = StepRunning
	return tea.Batch(
		spinner.Tick,
		runStep(m.steps[0], 0),
	)
}

// Update handles messages and updates the model.
func (m InstallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// ── Terminal resize ─────────────────────────────────────────
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 10
		return m, nil

	// ── Spinner tick ────────────────────────────────────────────
	case spinner.TickMsg:
		if m.done {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	// ── Step completed ──────────────────────────────────────────
	case InstalStepMsg:
		if msg.Err != nil {
			m.steps[msg.Index].state = StepError
			m.steps[msg.Index].err = msg.Err
		} else {
			m.steps[msg.Index].state = StepDone
		}

		m.currentIdx++
		if m.currentIdx >= len(m.steps) {
			m.done = true
			return m, tea.Quit
		}

		m.steps[m.currentIdx].state = StepRunning
		return m, tea.Batch(
			spinner.Tick,
			runStep(m.steps[m.currentIdx], m.currentIdx),
		)

	// ── Progress bar ────────────────────────────────────────────
	case progress.FrameMsg:
		var cmd tea.Cmd
		var p tea.Model
		p, cmd = m.progress.Update(msg)
		m.progress = p.(progress.Model)
		return m, cmd

	// ── Key press ───────────────────────────────────────────────
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the current state of the installer.
func (m InstallModel) View() string {
	var b strings.Builder

	// ── Banner (sin bordes, responsive) ─────────────────────────
	variant := ResolveBanner(m.width)
	b.WriteString(RenderBanner(variant, "ZyroCLI — Instalación", m.width))
	b.WriteString("\n\n")

	// ── Steps dentro de un contenedor con borde ─────────────────
	b.WriteString(RenderStepGroup(m.steps, m.spinner, m.width))
	b.WriteString("\n")

	// ── Progress bar (if in progress) ───────────────────────────
	if !m.done && m.currentIdx > 0 {
		pct := float64(m.currentIdx) / float64(len(m.steps))
		b.WriteString("\n\n")
		b.WriteString(m.progress.ViewAs(pct))
	}

	// ── Help ────────────────────────────────────────────────────
	if !m.done {
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press q to cancel"))
	}

	content := b.String()

	// ── Scroll limit ──────────────────────────────────────────
	if m.height > 0 {
		lines := strings.Split(content, "\n")
		maxLines := m.height - 3
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			lines = append(lines, helpStyle.Render("  [... más ...]"))
			content = strings.Join(lines, "\n")
		}
	}

	return content
}

// ── Step runner ─────────────────────────────────────────────────────────

// runStep executes a step in a goroutine and sends the result back.
func runStep(step *InstallStep, index int) tea.Cmd {
	return func() tea.Msg {
		err := step.Action()
		return InstalStepMsg{Index: index, Err: err}
	}
}

// ── RunInstall ──────────────────────────────────────────────────────────

// RunInstall executes the installation TUI and returns any fatal error.
func RunInstall(steps []InstallStep) error {
	m := NewInstallModel(steps)
	p := tea.NewProgram(m)

	final, err := p.Run()
	if err != nil {
		return fmt.Errorf("install TUI: %w", err)
	}

	model := final.(InstallModel)
	if model.done && model.err != nil {
		return model.err
	}

	return nil
}
