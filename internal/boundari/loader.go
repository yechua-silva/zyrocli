package boundari

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadPolicy busca y carga el archivo YAML de política para una fase.
// Los archivos se nombran phase{N}-boundari.yaml (ej: phase0-boundari.yaml para F0).
func LoadPolicy(phase string, searchDirs []string) (*Policy, error) {
	phaseNum := strings.TrimPrefix(phase, "F")
	filename := fmt.Sprintf("phase%s-boundari.yaml", phaseNum)
	for _, dir := range searchDirs {
		path := filepath.Join(dir, filename)
		if data, err := os.ReadFile(path); err == nil {
			var p Policy
			if err := yaml.Unmarshal(data, &p); err != nil {
				return nil, fmt.Errorf("boundari: parse %s: %w", path, err)
			}
			if err := ValidatePolicy(&p); err != nil {
				return nil, fmt.Errorf("boundari: validate %s: %w", path, err)
			}
			return &p, nil
		}
	}
	return nil, fmt.Errorf("boundari: policy not found for phase %s", phase)
}

// LoadDefaultPolicy retorna política hardcodeada como fallback
func LoadDefaultPolicy(phase string) *Policy {
	p := &Policy{
		Version: "1.0",
		Phase:   phase,
		Budget:  Budget{MaxToolCalls: 50, MaxRuntimeSecs: 300},
	}
	switch phase {
	case "PRE-F0":
		p.Description = "Alineación de dominio — lectura + .md (fallback)"
		p.Budget = Budget{MaxToolCalls: 30, MaxRuntimeSecs: 300}
		p.Tools = []ToolRule{
			{Name: "read_file", Action: ActionAllow},
			{Name: "search_code", Action: ActionAllow},
			{Name: "search_skills", Action: ActionAllow},
			{Name: "task_context", Action: ActionAllow},
			{Name: "web_search", Action: ActionAllow},
			{Name: "web_fetch", Action: ActionAllow},
			{Name: "glob", Action: ActionAllow},
			{Name: "grep", Action: ActionAllow},
			{Name: "write_file", Action: ActionAllow},
			{Name: "save_to_helix", Action: ActionAllow},
			{Name: "task_create", Action: ActionAllow},
			{Name: "edit_file", Action: ActionDeny},
			{Name: "execute_command", Action: ActionDeny},
		}
	case "F0":
		p.Description = "Investigación — solo lectura (fallback)"
		p.Tools = []ToolRule{
			{Name: "search_code", Action: ActionAllow},
			{Name: "search_skills", Action: ActionAllow},
			{Name: "task_context", Action: ActionAllow},
			{Name: "write_file", Action: ActionDeny},
			{Name: "edit_file", Action: ActionDeny},
			{Name: "execute_command", Action: ActionDeny},
			{Name: "task_create", Action: ActionAllow},
		}
	case "F3":
		p.Description = "Implementación — permisiva (fallback)"
		p.Budget = Budget{MaxToolCalls: 200, MaxRuntimeSecs: 1800}
		p.Tools = []ToolRule{
			{Name: "write_file", Action: ActionAllow},
			{Name: "edit_file", Action: ActionAllow},
			{Name: "execute_command", Action: ActionAllow, RequireApproval: true},
			{Name: "task_create", Action: ActionAllow},
			{Name: "save_to_helix", Action: ActionAllow},
		}
	default:
		p.Description = "Modo lectura (fallback)"
		p.Tools = []ToolRule{
			{Name: "write_file", Action: ActionDeny},
			{Name: "edit_file", Action: ActionDeny},
			{Name: "execute_command", Action: ActionDeny},
			{Name: "task_create", Action: ActionAllow},
			{Name: "save_to_helix", Action: ActionAllow},
		}
	}
	return p
}

// ValidatePolicy verifica que la política sea válida
func ValidatePolicy(p *Policy) error {
	if p.Phase == "" {
		return fmt.Errorf("boundari: phase is required")
	}
	if p.Budget.MaxToolCalls < 0 {
		return fmt.Errorf("boundari: max_tool_calls cannot be negative")
	}
	if p.Budget.MaxRuntimeSecs < 0 {
		return fmt.Errorf("boundari: max_runtime_seconds cannot be negative")
	}
	if len(p.Tools) == 0 {
		return fmt.Errorf("boundari: at least one tool rule required")
	}
	for _, t := range p.Tools {
		if t.Name == "" {
			return fmt.Errorf("boundari: tool name is required")
		}
		validActions := map[Action]bool{ActionAllow: true, ActionDeny: true, ActionApproval: true}
		if !validActions[t.Action] {
			return fmt.Errorf("boundari: invalid action %q for tool %s", t.Action, t.Name)
		}
	}
	return nil
}
