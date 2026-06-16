package spec

import (
	"fmt"
	"strings"
)

// CIO defines a Contract / Interface / Behavior / Constraint / Operation / Testing
// specification unit, following the clean agent-spec pattern described in AGENT.md.
type CIO struct {
	Contract   Contract
	Interface  []IOMethod
	Behavior   []Rule
	Constraint Constraint
	Operation  Operation
	Testing    Testing
}

// Contract is the top-level agreement between agent and system.
type Contract struct {
	ID          string
	Name        string
	Description string
}

// IOMethod describes a single input/output method in the interface.
type IOMethod struct {
	Name     string
	Input    string
	Output   string
}

// Rule defines a behavioral rule with preconditions and postconditions.
type Rule struct {
	Description    string
	Precondition   string
	Postcondition  string
}

// Constraint captures limitations or invariants.
type Constraint struct {
	Limitations []string
	Invariants  []string
}

// Operation defines an executable operation.
type Operation struct {
	Steps []string
}

// Testing describes the test strategy for this spec unit.
type Testing struct {
	Approach string
	Scopes   []string
}

// ToMarkdown serializes the CIO struct as a markdown document.
// Fields with zero values are omitted.
func (c *CIO) ToMarkdown() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# CIO: %s\n\n", c.Contract.Name))
	b.WriteString(fmt.Sprintf("**ID**: %s\n", c.Contract.ID))
	b.WriteString(fmt.Sprintf("**Description**: %s\n\n", c.Contract.Description))

	if len(c.Interface) > 0 {
		b.WriteString("## Interface\n\n")
		for _, m := range c.Interface {
			b.WriteString(fmt.Sprintf("- **%s**: `%s` → `%s`\n", m.Name, m.Input, m.Output))
		}
		b.WriteString("\n")
	}

	if len(c.Behavior) > 0 {
		b.WriteString("## Behavior\n\n")
		for _, r := range c.Behavior {
			b.WriteString(fmt.Sprintf("- **%s**\n", r.Description))
			if r.Precondition != "" {
				b.WriteString(fmt.Sprintf("  - Pre: %s\n", r.Precondition))
			}
			if r.Postcondition != "" {
				b.WriteString(fmt.Sprintf("  - Post: %s\n", r.Postcondition))
			}
		}
		b.WriteString("\n")
	}

	if len(c.Constraint.Limitations) > 0 {
		b.WriteString("## Constraints\n\n")
		b.WriteString("### Limitations\n")
		for _, l := range c.Constraint.Limitations {
			b.WriteString(fmt.Sprintf("- %s\n", l))
		}
		b.WriteString("\n")
	}

	if len(c.Constraint.Invariants) > 0 {
		b.WriteString("### Invariants\n")
		for _, inv := range c.Constraint.Invariants {
			b.WriteString(fmt.Sprintf("- %s\n", inv))
		}
		b.WriteString("\n")
	}

	if len(c.Operation.Steps) > 0 {
		b.WriteString("## Operation\n\n")
		for i, step := range c.Operation.Steps {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
		b.WriteString("\n")
	}

	if c.Testing.Approach != "" || len(c.Testing.Scopes) > 0 {
		b.WriteString("## Testing\n\n")
		if c.Testing.Approach != "" {
			b.WriteString(fmt.Sprintf("**Approach**: %s\n", c.Testing.Approach))
		}
		if len(c.Testing.Scopes) > 0 {
			b.WriteString("**Scopes**: " + strings.Join(c.Testing.Scopes, ", ") + "\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}
