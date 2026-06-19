package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// ── Banner variants ─────────────────────────────────────────────────────

// BannerVariant determines which banner to render based on terminal width.
type BannerVariant int

const (
	// BannerMedium shows ZYRO 3D centered (≥60 cols).
	BannerMedium BannerVariant = iota
	// BannerSmall shows simple text (<60 cols).
	BannerSmall
)

// ResolveBanner determines the banner variant based on terminal width.
func ResolveBanner(width int) BannerVariant {
	if width >= 60 {
		return BannerMedium
	}
	return BannerSmall
}

// RenderBanner renders the appropriate banner for the given variant and width.
// NO usa bordes alrededor del ASCII art. Centra usando el ancho real de terminal.
func RenderBanner(variant BannerVariant, subtitle string, termWidth int) string {
	var content string

	switch variant {
	case BannerMedium:
		content = RenderWelcome(subtitle)
	case BannerSmall:
		content = RenderSmallBanner(subtitle)
	}

	// Si no tenemos ancho real, usamos fallback
	if termWidth <= 0 {
		termWidth = 80
	}

	return lipgloss.Place(termWidth, lipgloss.Height(content),
		lipgloss.Center, lipgloss.Top,
		content,
	)
}

// ── Step list rendering ─────────────────────────────────────────────────

// stepStyle returns a style for a step based on its state.
func stepStyle(state StepState) lipgloss.Style {
	switch state {
	case StepPending:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#52525B"))
	case StepRunning:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F97316")) // Zyro orange
	case StepDone:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")) // Zyro green
	case StepError:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")) // Red
	default:
		return lipgloss.NewStyle()
	}
}

// RenderStepList renders the list of steps with their current states.
// Opcionalmente se puede envolver en un borde con stepGroupStyle.
func RenderStepList(steps []*InstallStep, s spinner.Model) string {
	var b strings.Builder

	for i, step := range steps {
		style := stepStyle(step.state)
		var line string

		switch step.state {
		case StepPending:
			line = fmt.Sprintf("  %s  %s",
				style.Render(" "),
				style.Render(step.Name))

		case StepRunning:
			line = fmt.Sprintf("  %s  %s",
				style.Render(s.View()),
				style.Render(step.Name))

		case StepDone:
			line = fmt.Sprintf("  %s  %s",
				style.Render("✓"),
				style.Render(step.Name))

		case StepError:
			line = fmt.Sprintf("  %s  %s",
				style.Render("✗"),
				style.Render(step.Name))
			if step.err != nil {
				line += fmt.Sprintf("\n    └ %s",
					lipgloss.NewStyle().
						Foreground(lipgloss.Color("#EF4444")).
						Render(step.err.Error()))
			}
		}

		b.WriteString(line)
		if i < len(steps)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// RenderStepGroup renderiza la lista de pasos dentro de un contenedor con borde redondeado.
func RenderStepGroup(steps []*InstallStep, s spinner.Model, termWidth int) string {
	stepsContent := RenderStepList(steps, s)

	// Ancho del borde: 2 de padding horizontal + 2 de border
	borderWidth := 4
	groupWidth := termWidth - borderWidth
	if groupWidth < 40 {
		groupWidth = 40
	}

	groupStyle := stepGroupStyle.Copy().
		Width(groupWidth)

	return groupStyle.Render(stepsContent)
}

// ── Summary ─────────────────────────────────────────────────────────────

// RenderSummary renders a summary table of all completed steps.
func RenderSummary(steps []*InstallStep) string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F97316")).
		Padding(0, 1)

	b.WriteString(titleStyle.Render("Resumen de Instalación"))
	b.WriteString("\n\n")

	// Steps
	for _, step := range steps {
		var icon string
		var iconStyle lipgloss.Style

		switch step.state {
		case StepDone:
			icon = "✓"
			iconStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
		case StepError:
			icon = "✗"
			iconStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
		default:
			icon = "–"
			iconStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#52525B"))
		}

		b.WriteString(fmt.Sprintf("  %s  %s\n",
			iconStyle.Render(icon),
			step.Name,
		))
	}

	// Footer
	b.WriteString("\n")
	successStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#10B981"))

	b.WriteString(successStyle.Render("✔ Instalación completada"))

	return b.String()
}

// ── Style exports for other files ───────────────────────────────────────

var (
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#52525B"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F97316"))
)
