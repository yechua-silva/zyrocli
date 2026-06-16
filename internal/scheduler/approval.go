package scheduler

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// stdinReader para tests
var stdinReader = bufio.NewReader(os.Stdin)

// ApprovalGate procesa aprobación usando OpenCode si está disponible
func ApprovalGate(phase Phase, summary string) (bool, error) {
	// Intentar con OpenCode subagent
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

	// Fallback: stdin (prompt humano directo)
	for {
		fmt.Printf("\n=== Aprobación requerida — Fase %s ===\n", phase)
		fmt.Printf("Resumen: %s\n", summary)
		fmt.Print("¿Aprobar? (y/n): ")

		input, err := stdinReader.ReadString('\n')
		if err != nil {
			return false, fmt.Errorf("read approval: %w", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "y", "yes", "s", "si":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Printf("Entrada inválida %q. Ingresá y/n.\n", input)
			continue
		}
	}
}

// PromptApproval mantiene compatibilidad (deprecated)
func PromptApproval(phase Phase, summary string) (bool, error) {
	return ApprovalGate(phase, summary)
}

func opencodeExists() bool {
	_, err := exec.LookPath("opencode")
	return err == nil
}
