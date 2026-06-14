package scheduler

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// stdinReader is a package-level variable so tests can replace it with a mock.
var stdinReader = bufio.NewReader(os.Stdin)

// PromptApproval prompts the user to approve or reject a phase result.
// Returns true if approved, false if rejected.
// In auto mode, callers skip this function entirely.
func PromptApproval(phase Phase, summary string) (bool, error) {
	fmt.Printf("\n--- Phase %s complete ---\n", phase)
	fmt.Printf("Summary: %s\n", summary)
	fmt.Print("Approve? (y/n): ")

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
		fmt.Printf("Invalid input %q. Enter y/n.\n", input)
		return PromptApproval(phase, summary)
	}
}
