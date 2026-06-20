package opencode

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
)

// AgentConfig represents the configuration for a single agent profile.
type AgentConfig struct {
	Model           string `json:"model"`
	Mode            string `json:"mode"`                      // "primary" | "subagent"
	ReasoningEffort string `json:"reasoningEffort,omitempty"` // "low" | "medium" | "high"
}

// OpenCodeConfig models only the sections of opencode.json that we need.
type OpenCodeConfig struct {
	Agent     map[string]AgentConfig `json:"agent,omitempty"`
	Providers []Provider             `json:"providers,omitempty"`
	// Raw holds any other top-level keys we do not explicitly model.
	// Populated during read so WriteAgentConfig can preserve them.
	Raw map[string]json.RawMessage `json:"-"`
}

// ResolveConfigPath returns the default path to the OpenCode configuration file:
// ~/.config/opencode/opencode.json
func ResolveConfigPath() string {
	return GetDefaultPath()
}

// GetDefaultPath returns the default path to the OpenCode configuration file:
// ~/.config/opencode/opencode.json
func GetDefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".config/opencode/opencode.json"
	}
	return filepath.Join(home, ".config", "opencode", "opencode.json")
}

// GetEffectiveConfigPath returns the path to the most relevant opencode.json.
// If a project-level config exists in the current working directory, OpenCode
// reads it with precedence over the global config. This function mirrors that
// logic so "profile set" and "profile list" write to the file that OpenCode
// actually uses.
func GetEffectiveConfigPath() string {
	// Check for project-level config in CWD
	cwd, err := os.Getwd()
	if err == nil {
		projectConfig := filepath.Join(cwd, ".config", "opencode", "opencode.json")
		if _, err := os.Stat(projectConfig); err == nil {
			return projectConfig
		}
	}
	// Fall back to global config
	return GetDefaultPath()
}

// ReadProviders reads providers from opencode.json and merges them with
// KnownProviders. Providers defined in the JSON file take precedence over
// curated providers with the same ID.
//
// If the file does not exist or has no providers section, KnownProviders is
// returned without error.
func ReadProviders(path string) ([]Provider, error) {
	known := KnownProviders()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return known, nil
		}
		return nil, fmt.Errorf("reading opencode config: %w", err)
	}

	var cfg struct {
		Providers []Provider `json:"providers,omitempty"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing opencode config: %w", err)
	}

	if len(cfg.Providers) == 0 {
		return known, nil
	}

	return mergeProviders(known, cfg.Providers), nil
}

// ReadAgentConfigs reads the agent section from the opencode.json file at path.
func ReadAgentConfigs(path string) (map[string]AgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading opencode config: %w", err)
	}

	var cfg OpenCodeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing opencode config: %w", err)
	}

	return cfg.Agent, nil
}

// WriteAgentConfig writes the given agent configs into the agent section of
// the opencode.json file at path. All other sections of the file are preserved.
//
// If the file does not exist, it is created with only the agent section.
func WriteAgentConfig(path, profile string, configs map[string]AgentConfig) error {
	// Read existing config if it exists.
	cfg, err := readFullConfig(path)
	if err != nil {
		return fmt.Errorf("reading config for write: %w", err)
	}

	// Update agent section.
	if cfg.Agent == nil {
		cfg.Agent = make(map[string]AgentConfig)
	}
	for k, v := range configs {
		cfg.Agent[k] = v
	}

	// Rebuild the full JSON preserving raw sections.
	output := make(map[string]json.RawMessage)

	// Write known sections explicitly.
	agentData, err := json.Marshal(cfg.Agent)
	if err != nil {
		return fmt.Errorf("marshalling agent config: %w", err)
	}
	output["agent"] = agentData

	// Copy all raw sections except agent (already handled).
	for k, v := range cfg.Raw {
		if k != "agent" {
			output[k] = v
		}
	}

	// Marshal with 2-space indent.
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling output: %w", err)
	}

	// Ensure parent directory exists.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing opencode config: %w", err)
	}

	_ = profile // reserved for future use

	return nil
}

// readFullConfig reads the complete opencode.json and captures all sections,
// both known (agent) and unknown (raw).
func readFullConfig(path string) (*OpenCodeConfig, error) {
	cfg := &OpenCodeConfig{
		Raw: make(map[string]json.RawMessage),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // return empty config
		}
		return nil, err
	}

	// Unmarshal known fields explicitly.
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Capture everything else via raw map.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	for k, v := range raw {
		if k != "agent" && k != "providers" {
			cfg.Raw[k] = v
		}
	}

	return cfg, nil
}

// mergeProviders merges override providers into the base list, replacing
// providers with the same ID.
func mergeProviders(base, overrides []Provider) []Provider {
	result := make([]Provider, len(base))
	copy(result, base)

	for _, o := range overrides {
		idx := slices.IndexFunc(result, func(p Provider) bool {
			return p.ID == o.ID
		})
		if idx >= 0 {
			result[idx] = o
		} else {
			result = append(result, o)
		}
	}

	return result
}

// FetchOpenCodeProviders connects to the OpenCode SDK API (localhost:4096)
// to get the actual providers the user has configured.
// Returns nil if OpenCode is not running or can't be reached.
func FetchOpenCodeProviders() []Provider {
	// Try to fetch from OpenCode SDK API
	providers, _ := fetchOCProviders()
	return providers
}

// ocProviderResponse mirrors the OpenCode API response for GET /provider.
type ocProviderResponse struct {
	All       []Provider        `json:"all,omitempty"`
	Default   map[string]string `json:"default,omitempty"`
	Connected []string          `json:"connected,omitempty"`
}

func fetchOCProviders() ([]Provider, error) {
	// OpenCode SDK server runs on localhost:4096
	resp, err := http.Get("http://localhost:4096/provider")
	if err != nil {
		return nil, fmt.Errorf("opencode sdk: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opencode sdk: status %d", resp.StatusCode)
	}

	var result ocProviderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("opencode sdk: decode: %w", err)
	}

	// If OpenCode tells us which providers are connected, filter to those
	if len(result.Connected) > 0 {
		connected := make(map[string]bool, len(result.Connected))
		for _, id := range result.Connected {
			connected[id] = true
		}
		var active []Provider
		for _, p := range result.All {
			if connected[p.ID] {
				active = append(active, p)
			}
		}
		return active, nil
	}

	// Otherwise return all providers
	return result.All, nil
}

// RegisterPlugin adds a plugin path to the opencode.json config file.
// It reads the existing config, adds the plugin to the plugin array if not already present,
// and writes the config back.
func RegisterPlugin(configPath, pluginPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If file doesn't exist, create minimal config
		if os.IsNotExist(err) {
			cfg := map[string]any{
				"$schema": "https://opencode.ai/config.json",
				"plugin":  []string{pluginPath},
			}
			out, _ := json.MarshalIndent(cfg, "", "  ")
			return os.WriteFile(configPath, out, 0644)
		}
		return fmt.Errorf("register plugin: read: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("register plugin: parse: %w", err)
	}

	// Get or create plugin array
	var plugins []string
	if p, ok := raw["plugin"]; ok {
		if arr, ok := p.([]any); ok {
			for _, item := range arr {
				if s, ok := item.(string); ok {
					plugins = append(plugins, s)
				}
			}
		}
	}

	// Check if already registered
	for _, p := range plugins {
		if p == pluginPath {
			return nil // already registered
		}
	}

	plugins = append(plugins, pluginPath)
	raw["plugin"] = plugins

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("register plugin: marshal: %w", err)
	}

	return os.WriteFile(configPath, out, 0644)
}
