package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// PluginConfig configura los plugins de OpenCode
type PluginConfig struct {
	ClaudeBridge bool           `json:"claude_bridge"`
	LazyLoader   bool           `json:"lazy_loader"`
	MultiAgent   bool           `json:"multi_agent"`
	Sources      []SourceConfig `json:"sources,omitempty"`
}

// SourceConfig fuente de skills para el bridge
type SourceConfig struct {
	Dir       string `json:"dir"`
	Namespace string `json:"namespace"`
}

// EnsurePluginsConfig asegura que los plugins estén configurados
func EnsurePluginsConfig(opencodeDir string, config PluginConfig) error {
	configPath := filepath.Join(opencodeDir, "opencode.json")

	var cfg map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &cfg)
	} else {
		cfg = make(map[string]interface{})
	}

	// Configurar plugins
	plugins := make([]map[string]interface{}, 0)
	if config.ClaudeBridge {
		plugins = append(plugins, map[string]interface{}{
			"name":    "@sjawhar/opencode-claude-bridge",
			"sources": config.Sources,
		})
	}
	if config.LazyLoader {
		plugins = append(plugins, map[string]interface{}{
			"name": "opencode-embedded-skill-mcp",
		})
	}
	if config.MultiAgent {
		plugins = append(plugins, map[string]interface{}{
			"name": "opencode-multiagent",
		})
	}

	if len(plugins) > 0 {
		cfg["plugins"] = plugins
	}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(configPath, data, 0644)
}
