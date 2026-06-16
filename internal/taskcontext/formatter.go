package taskcontext

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormatText formatea el contexto en texto legible
func (tc *TaskContext) FormatText() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Context for Task #%d", tc.TaskID))
	if tc.Description != "" {
		b.WriteString(fmt.Sprintf(": %s", tc.Description))
	}
	b.WriteString("\n\n")

	writeSection(&b, "Skills", tc.Skills, func(item ContextItem) string {
		return fmt.Sprintf("  \u2022 %s", item.Name)
	})

	writeSection(&b, "CodeNodes", tc.CodeNodes, func(item ContextItem) string {
		if item.Summary != "" {
			return fmt.Sprintf("  \u2022 %s \u2014 %s", item.Name, item.Summary)
		}
		return fmt.Sprintf("  \u2022 %s", item.Name)
	})

	writeSection(&b, "Documents", tc.Documents, func(item ContextItem) string {
		s := fmt.Sprintf("  \u2022 %s", item.Name)
		if item.Type != "" {
			s += fmt.Sprintf(" (%s)", item.Type)
		}
		return s
	})

	writeSection(&b, "Patterns", tc.Patterns, func(item ContextItem) string {
		return fmt.Sprintf("  \u2022 %s", item.Name)
	})

	return b.String()
}

func writeSection(b *strings.Builder, title string, items []ContextItem, format func(ContextItem) string) {
	if len(items) == 0 {
		fmt.Fprintf(b, "%s (0):\n  (none)\n\n", title)
		return
	}
	fmt.Fprintf(b, "%s (%d):\n", title, len(items))
	for _, item := range items {
		b.WriteString(format(item))
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

// FormatJSON formatea el contexto como JSON
func (tc *TaskContext) FormatJSON() (string, error) {
	data, err := json.MarshalIndent(tc, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FormatPrompt formatea el contexto como prompt listo para inyectar en subagente
func (tc *TaskContext) FormatPrompt() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("## Context for Task #%d", tc.TaskID))
	if tc.Description != "" {
		b.WriteString(fmt.Sprintf(": %s", tc.Description))
	}
	b.WriteString("\n\n")

	if len(tc.Skills) > 0 {
		b.WriteString("### Skills Required\n")
		for _, s := range tc.Skills {
			b.WriteString(fmt.Sprintf("- %s\n", s.Name))
		}
		b.WriteString("\n")
	}

	if len(tc.CodeNodes) > 0 {
		b.WriteString("### Affected Code\n")
		for _, c := range tc.CodeNodes {
			b.WriteString(fmt.Sprintf("- **%s**", c.Name))
			if c.Type != "" {
				b.WriteString(fmt.Sprintf(" (%s)", c.Type))
			}
			if c.Summary != "" {
				b.WriteString(fmt.Sprintf(": %s", c.Summary))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if len(tc.Documents) > 0 {
		b.WriteString("### Reference Documents\n")
		for _, d := range tc.Documents {
			b.WriteString(fmt.Sprintf("- %s", d.Name))
			if d.Type != "" {
				b.WriteString(fmt.Sprintf(" (%s)", d.Type))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if len(tc.Patterns) > 0 {
		b.WriteString("### Patterns\n")
		for _, p := range tc.Patterns {
			b.WriteString(fmt.Sprintf("- %s\n", p.Name))
		}
		b.WriteString("\n")
	}

	return b.String()
}
