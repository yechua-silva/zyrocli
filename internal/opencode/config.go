package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the opencode.jsonc configuration structure.
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

// MCPEntry defines a local MCP server configuration.
type MCPEntry struct {
	Type    string   `json:"type"`
	Command []string `json:"command"`
	CWD     string   `json:"cwd,omitempty"`
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
var OpenCodeConfigPath = "~/.config/opencode/opencode.jsonc"

// expandHome replaces "~" with the user's home directory.
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// WriteGlobalConfig writes the opencode.jsonc config to the standard location.
// Creates parent directories if needed. Returns the path written.
func WriteGlobalConfig(cfg *Config) (string, error) {
	path := expandHome(OpenCodeConfigPath)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("opencode: create config dir %s: %w", dir, err)
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
