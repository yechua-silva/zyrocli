// DEPRECATED: Skills and MCP tools now load via OpenCode plugins (claude-bridge,
// embedded-skill-mcp). This file is kept for backward compatibility but should
// not be used for new configurations. See internal/boomerang/ for the new
// orchestration pipeline.
package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the opencode.json configuration structure.
type Config struct {
	Schema        string              `json:"$schema,omitempty"`
	DefaultAgent  string              `json:"default_agent,omitempty"`
	Agent         map[string]Agent    `json:"agent,omitempty"`
	MCP           map[string]MCPEntry `json:"mcp,omitempty"`
	Skills        *SkillsConfig       `json:"skills,omitempty"`
	Command       map[string]Command  `json:"command,omitempty"`
}

// Agent defines an OpenCode agent (primary or subagent).
type Agent struct {
	Mode        string         `json:"mode"`
	Description string         `json:"description,omitempty"`
	Prompt      string         `json:"prompt,omitempty"`
	Model       string         `json:"model,omitempty"`
	Hidden      bool           `json:"hidden,omitempty"`
	Permission  map[string]any `json:"permission,omitempty"`
}

// MCPEntry defines a local or remote MCP server configuration.
type MCPEntry struct {
	Type    string   `json:"type"`
	Command []string `json:"command,omitempty"`
	URL     string   `json:"url,omitempty"`
	CWD     string   `json:"cwd,omitempty"`
	Enabled *bool    `json:"enabled,omitempty"`
}

// SkillsConfig defines additional skill paths.
type SkillsConfig struct {
	Paths []string `json:"paths,omitempty"`
}

// Command defines a slash command.
type Command struct {
	Template    string `json:"template"`
	Description string `json:"description,omitempty"`
	Subtask     bool   `json:"subtask,omitempty"`
	Agent       string `json:"agent,omitempty"`
}

// OpenCodeConfigPath is the default global config path.
var OpenCodeConfigPath = "~/.config/opencode/opencode.json"

// expandHome replaces "~" with the user's home directory.
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// WriteGlobalConfig writes the opencode.json config to the standard location.
// Creates parent directories if needed. MERGES with existing config if present,
// so user's manually added MCP servers and agents are preserved.
// Returns the path written.
func WriteGlobalConfig(cfg *Config) (string, error) {
	path := expandHome(OpenCodeConfigPath)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("opencode: create config dir %s: %w", dir, err)
	}

	// Read existing config if present and merge
	existing := &Config{}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, existing); err == nil {
			// Merge MCP servers: user's existing entries take precedence
			if existing.MCP == nil {
				existing.MCP = make(map[string]MCPEntry)
			}
			for k, v := range cfg.MCP {
				if _, exists := existing.MCP[k]; !exists {
					existing.MCP[k] = v
				}
			}
			cfg.MCP = existing.MCP

			// Merge agents: cfg provee estructura + modelo default,
			// existing preserva model si el usuario lo personalizó vía profile tui
			if existing.Agent != nil {
				for k, v := range existing.Agent {
					if _, exists := cfg.Agent[k]; exists {
						// Agente existe en ambos — preservar model del existing
						// si el usuario lo cambió (distinto al default)
						if v.Model != "" && v.Model != cfg.Agent[k].Model {
							a := cfg.Agent[k]
							a.Model = v.Model
							cfg.Agent[k] = a
						}
					} else {
						// Agente solo en existing (ej: agregado manualmente)
						cfg.Agent[k] = v
					}
				}
			}
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("opencode: marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("opencode: write %s: %w", path, err)
	}

	return path, nil
}
