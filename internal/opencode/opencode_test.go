package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// -- KnownProviders tests --

func TestKnownProvidersNotEmpty(t *testing.T) {
	providers := KnownProviders()
	if len(providers) == 0 {
		t.Fatal("KnownProviders() returned empty slice")
	}
}

func TestKnownProvidersHasIDs(t *testing.T) {
	for _, p := range KnownProviders() {
		if p.ID == "" {
			t.Errorf("provider %q has empty ID", p.Name)
		}
		for _, m := range p.Models {
			if m.ID == "" {
				t.Errorf("model in provider %q has empty ID", p.ID)
			}
		}
	}
}

// -- ReadProviders tests --

func TestReadProviders_FileNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	got, err := ReadProviders(path)
	if err != nil {
		t.Fatalf("ReadProviders should not error on missing file: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected KnownProviders as fallback, got empty")
	}
	// Verify it's the curated list.
	if got[0].ID != "opencode-go" {
		t.Errorf("expected first provider to be opencode-go, got %q", got[0].ID)
	}
}

func TestReadProviders_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")

	// Write a config with one custom provider.
	input := `{
		"providers": [
			{"id": "custom-provider", "name": "Custom", "models": [{"id": "custom-model", "name": "Custom Model"}]}
		]
	}`
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadProviders(path)
	if err != nil {
		t.Fatalf("ReadProviders failed: %v", err)
	}

	// Should contain both known and custom providers.
	foundCustom := false
	for _, p := range got {
		if p.ID == "custom-provider" {
			foundCustom = true
			break
		}
	}
	if !foundCustom {
		t.Error("custom provider not found in merged result")
	}

	// Known providers should still be present.
	foundKnown := false
	for _, p := range got {
		if p.ID == "opencode-go" {
			foundKnown = true
			break
		}
	}
	if !foundKnown {
		t.Error("known provider opencode-go missing from merged result")
	}
}

func TestReadProviders_ProvidersOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")

	// Override the opencode-go provider with different models.
	input := `{
		"providers": [
			{"id": "opencode-go", "name": "Overridden", "models": [{"id": "override-model", "name": "Override"}]}
		]
	}`
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadProviders(path)
	if err != nil {
		t.Fatalf("ReadProviders failed: %v", err)
	}

	for _, p := range got {
		if p.ID == "opencode-go" {
			if p.Name != "Overridden" {
				t.Errorf("expected overridden name 'Overridden', got %q", p.Name)
			}
			if len(p.Models) != 1 || p.Models[0].ID != "override-model" {
				t.Errorf("expected overridden models, got %+v", p.Models)
			}
			return
		}
	}
	t.Error("opencode-go provider not found after override")
}

func TestReadProviders_NoProvidersSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")

	// Config without providers section.
	input := `{"agent": {"test": {"model": "x", "mode": "primary"}}}`
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadProviders(path)
	if err != nil {
		t.Fatalf("ReadProviders failed: %v", err)
	}

	// Should return the full curated list.
	if len(got) != len(KnownProviders()) {
		t.Errorf("expected %d providers, got %d", len(KnownProviders()), len(got))
	}
}

// -- WriteAgentConfig tests --

func TestWriteAgentConfig_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")

	configs := map[string]AgentConfig{
		"test-agent": {
			Model: "deepseek-v4-flash",
			Mode:  "primary",
		},
	}

	if err := WriteAgentConfig(path, "default", configs); err != nil {
		t.Fatalf("WriteAgentConfig failed: %v", err)
	}

	// Verify file exists and can be read back.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file was not created: %v", err)
	}

	var result struct {
		Agent map[string]AgentConfig `json:"agent"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON written: %v", err)
	}

	cfg, ok := result.Agent["test-agent"]
	if !ok {
		t.Fatal("test-agent not found in written config")
	}
	if cfg.Model != "deepseek-v4-flash" {
		t.Errorf("expected model deepseek-v4-flash, got %q", cfg.Model)
	}
	if cfg.Mode != "primary" {
		t.Errorf("expected mode primary, got %q", cfg.Mode)
	}
}

func TestWriteAgentConfig_PreservesOtherSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")

	// Pre-create a file with multiple sections.
	existing := `{
		"mcp": {
			"server1": {"enabled": true}
		},
		"agent": {
			"old-agent": {"model": "x", "mode": "subagent"}
		},
		"permission": {
			"read": {"*": "allow"}
		}
	}`
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	configs := map[string]AgentConfig{
		"new-agent": {
			Model: "gemini-2.5-flash",
			Mode:  "subagent",
		},
	}

	if err := WriteAgentConfig(path, "default", configs); err != nil {
		t.Fatalf("WriteAgentConfig failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// mcp section should be preserved.
	if _, ok := raw["mcp"]; !ok {
		t.Error("mcp section was lost")
	}
	// permission section should be preserved.
	if _, ok := raw["permission"]; !ok {
		t.Error("permission section was lost")
	}

	// Agent section should contain both old and new.
	var agent map[string]AgentConfig
	if err := json.Unmarshal(raw["agent"], &agent); err != nil {
		t.Fatalf("invalid agent section: %v", err)
	}
	if _, ok := agent["old-agent"]; !ok {
		t.Error("old-agent was removed from agent section")
	}
	if _, ok := agent["new-agent"]; !ok {
		t.Error("new-agent was not added to agent section")
	}
}

func TestWriteAgentConfig_AgentSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")

	configs := map[string]AgentConfig{
		"my-agent": {
			Model:           "claude-sonnet-4-6",
			Mode:            "primary",
			ReasoningEffort: "high",
		},
	}

	if err := WriteAgentConfig(path, "default", configs); err != nil {
		t.Fatalf("WriteAgentConfig failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)

	// Verify format: 2-space indentation.
	if !strings.Contains(content, "  ") {
		t.Error("expected 2-space indentation")
	}

	// Verify structure.
	var parsed struct {
		Agent map[string]AgentConfig `json:"agent"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	cfg := parsed.Agent["my-agent"]
	if cfg.Model != "claude-sonnet-4-6" {
		t.Errorf("model = %q", cfg.Model)
	}
	if cfg.Mode != "primary" {
		t.Errorf("mode = %q", cfg.Mode)
	}
	if cfg.ReasoningEffort != "high" {
		t.Errorf("reasoningEffort = %q", cfg.ReasoningEffort)
	}
}

func TestWriteAgentConfig_FileNotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "dir", "opencode.json")

	configs := map[string]AgentConfig{
		"test": {
			Model: "deepseek-v4-flash",
			Mode:  "primary",
		},
	}

	if err := WriteAgentConfig(path, "default", configs); err != nil {
		t.Fatalf("WriteAgentConfig should create directory and file: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("file was not created")
	}
}

// -- Config path tests --

func TestResolveConfigPath(t *testing.T) {
	path := ResolveConfigPath()
	if !strings.HasSuffix(path, "opencode.json") {
		t.Errorf("expected path to end with opencode.json, got %q", path)
	}
	if !strings.Contains(path, ".config/opencode") {
		t.Errorf("expected path to contain .config/opencode, got %q", path)
	}
}

func TestGetDefaultPath(t *testing.T) {
	path := GetDefaultPath()
	if !strings.HasSuffix(path, "opencode.json") {
		t.Errorf("expected path to end with opencode.json, got %q", path)
	}
	if !strings.Contains(path, ".config/opencode") {
		t.Errorf("expected path to contain .config/opencode, got %q", path)
	}
}

// -- ReadAgentConfigs tests --

func TestReadAgentConfigs_FileNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	got, err := ReadAgentConfigs(path)
	if err != nil {
		t.Fatalf("ReadAgentConfigs should not error on missing file: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for missing file")
	}
}

func TestReadAgentConfigs_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")

	input := `{
		"agent": {
			"agent-a": {"model": "m1", "mode": "primary"},
			"agent-b": {"model": "m2", "mode": "subagent", "reasoningEffort": "low"}
		}
	}`
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadAgentConfigs(path)
	if err != nil {
		t.Fatalf("ReadAgentConfigs failed: %v", err)
	}

	if len(got) != 2 {
		t.Errorf("expected 2 agents, got %d", len(got))
	}

	a, ok := got["agent-a"]
	if !ok {
		t.Fatal("agent-a not found")
	}
	if a.Model != "m1" || a.Mode != "primary" {
		t.Errorf("agent-a = %+v", a)
	}

	b, ok := got["agent-b"]
	if !ok {
		t.Fatal("agent-b not found")
	}
	if b.Model != "m2" || b.Mode != "subagent" || b.ReasoningEffort != "low" {
		t.Errorf("agent-b = %+v", b)
	}
}
