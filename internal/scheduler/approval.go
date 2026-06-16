package scheduler

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// stdinReader is a package-level variable so tests can replace it with a mock.
var stdinReader = bufio.NewReader(os.Stdin)

// GuidedApproval holds phase context for an interactive approval dialog.
// The dialog presents a structured summary, recommendation, and risk warning
// before asking for a decision.
type GuidedApproval struct {
	Phase      Phase
	Summary    string // resumen de lo que se hizo
	Recommend  string // recomendación del orquestador
	Risk       string // advertencia de riesgo
	FullOutput string // output completo del agente (se muestra con "d")
}

// NewGuidedApproval creates a GuidedApproval with required fields.
func NewGuidedApproval(phase Phase, summary string) *GuidedApproval {
	return &GuidedApproval{
		Phase:   phase,
		Summary: summary,
	}
}

// WithRecommend sets the recommendation text.
func (g *GuidedApproval) WithRecommend(text string) *GuidedApproval {
	g.Recommend = text
	return g
}

// WithRisk sets the risk warning text.
func (g *GuidedApproval) WithRisk(text string) *GuidedApproval {
	g.Risk = text
	return g
}

// WithDetail sets the full output for "d" (detail) mode.
func (g *GuidedApproval) WithDetail(output string) *GuidedApproval {
	g.FullOutput = output
	return g
}

// PromptApproval displays the guided approval dialog and reads a response.
// Returns true if approved (s/sí), false if rejected (n/no).
func (g *GuidedApproval) PromptApproval() (bool, error) {
	fmt.Printf("\n─── Fase: %s — Completada ───\n\n", g.Phase)
	fmt.Printf("Resumen: %s\n", g.Summary)

	if g.Recommend != "" {
		fmt.Printf("\n### Recomendación\n%s\n", g.Recommend)
	}
	if g.Risk != "" {
		fmt.Printf("\n### Riesgos\n%s\n", g.Risk)
	}

	fmt.Printf("\n¿Querés ajustar algo o continuamos? (s/n/d): ")

	input, err := stdinReader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read approval: %w", err)
	}
	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "s", "si", "sí", "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	case "d", "detalle":
		g.showDetails()
		return g.PromptApproval()
	default:
		fmt.Printf("Respuesta no reconocida %q. Usá 's' (sí), 'n' (no), o 'd' (detalle).\n", input)
		return g.PromptApproval()
	}
}

// showDetails prints the full agent output for the phase.
func (g *GuidedApproval) showDetails() {
	if g.FullOutput == "" {
		fmt.Println("\n(No hay detalle adicional disponible)")
		return
	}
	fmt.Printf("\n─── Detalle de %s ───\n%s\n\n", g.Phase, g.FullOutput)
}

// ApprovalGate procesa aprobación usando OpenCode si está disponible.
// Si opencode no está disponible, delega a GuidedApproval.
func ApprovalGate(phase Phase, summary string) (bool, error) {
	if opencodeExists() {
		cmd := exec.Command("opencode", "subagent", "zyro-approval-gate",
			"--param", fmt.Sprintf("phase=%s", phase),
			"--param", fmt.Sprintf("summary=%s", summary),
		)
		output, err := cmd.Output()
		if err == nil {
			outputStr := strings.TrimSpace(string(output))
			if strings.Contains(strings.ToLower(outputStr), "approved") ||
				strings.Contains(strings.ToLower(outputStr), `"approved": true`) {
				return true, nil
			}
			return false, nil
		}
	}

	// Fallback: GuidedApproval dialog
	g := NewGuidedApproval(phase, summary)
	return g.PromptApproval()
}

// PromptApproval mantiene compatibilidad (deprecated)
func PromptApproval(phase Phase, summary string) (bool, error) {
	return ApprovalGate(phase, summary)
}

func opencodeExists() bool {
	_, err := exec.LookPath("opencode")
	return err == nil
}
