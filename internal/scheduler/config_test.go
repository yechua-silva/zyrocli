package scheduler

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// validHandoffYAML is a minimal valid handoff.yaml with all scheduler-relevant fields.
const validHandoffYAML = `version: "2.0"
source:
  system: holdin-admin
project:
  name: TestProject
  language: go
governance:
  mode: structured
  module: github.com/yechua-silva/zyrocli
  go_version: "1.26"
testing:
  strategy: table-driven
limits:
  max_tasks: 15
  max_lines: 400
  max_loops: 3
  phase_timeout: 15m
`

// minimalHandoffYAML has no optional limits — tests defaults.
const minimalHandoffYAML = `version: "2.0"
source:
  system: holdin-admin
project:
  name: TestProject
  language: go
governance:
  mode: structured
testing:
  strategy: unit
`

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "handoff.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfig_Valid(t *testing.T) {
	path := writeTempYAML(t, validHandoffYAML)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q) unexpected error: %v", path, err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig returned nil config")
	}

	if cfg.Mode != "structured" {
		t.Errorf("expected Mode 'structured', got %q", cfg.Mode)
	}
	if cfg.Module != "github.com/yechua-silva/zyrocli" {
		t.Errorf("expected Module 'github.com/yechua-silva/zyrocli', got %q", cfg.Module)
	}
	if cfg.GoVersion != "1.26" {
		t.Errorf("expected GoVersion '1.26', got %q", cfg.GoVersion)
	}
	if cfg.MaxTasks != 15 {
		t.Errorf("expected MaxTasks 15, got %d", cfg.MaxTasks)
	}
	if cfg.MaxLines != 400 {
		t.Errorf("expected MaxLines 400, got %d", cfg.MaxLines)
	}
	if cfg.MaxLoops != 3 {
		t.Errorf("expected MaxLoops 3, got %d", cfg.MaxLoops)
	}
	if cfg.PhaseTimeout != 15*time.Minute {
		t.Errorf("expected PhaseTimeout 15m, got %v", cfg.PhaseTimeout)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	path := writeTempYAML(t, minimalHandoffYAML)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q) unexpected error: %v", path, err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig returned nil config")
	}

	// max_loops should default to 5
	if cfg.MaxLoops != 5 {
		t.Errorf("expected default MaxLoops 5, got %d", cfg.MaxLoops)
	}

	// phase_timeout should default to 10 minutes
	if cfg.PhaseTimeout != 10*time.Minute {
		t.Errorf("expected default PhaseTimeout 10m, got %v", cfg.PhaseTimeout)
	}

	// zero-value fields should be 0
	if cfg.MaxTasks != 0 {
		t.Errorf("expected MaxTasks 0, got %d", cfg.MaxTasks)
	}
	if cfg.MaxLines != 0 {
		t.Errorf("expected MaxLines 0, got %d", cfg.MaxLines)
	}
}

func TestLoadConfig_CustomValues(t *testing.T) {
	const customYAML = `version: "2.0"
source:
  system: holdin-admin
project:
  name: CustomProject
  language: python
governance:
  mode: agile
  module: custom/module
  go_version: "1.22"
testing:
  strategy: bdd
limits:
  max_tasks: 99
  max_lines: 9999
  max_loops: 10
  phase_timeout: 30s
`
	path := writeTempYAML(t, customYAML)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q) unexpected error: %v", path, err)
	}

	if cfg.Mode != "agile" {
		t.Errorf("expected Mode 'agile', got %q", cfg.Mode)
	}
	if cfg.Module != "custom/module" {
		t.Errorf("expected Module 'custom/module', got %q", cfg.Module)
	}
	if cfg.GoVersion != "1.22" {
		t.Errorf("expected GoVersion '1.22', got %q", cfg.GoVersion)
	}
	if cfg.MaxTasks != 99 {
		t.Errorf("expected MaxTasks 99, got %d", cfg.MaxTasks)
	}
	if cfg.MaxLines != 9999 {
		t.Errorf("expected MaxLines 9999, got %d", cfg.MaxLines)
	}
	if cfg.MaxLoops != 10 {
		t.Errorf("expected MaxLoops 10, got %d", cfg.MaxLoops)
	}
	if cfg.PhaseTimeout != 30*time.Second {
		t.Errorf("expected PhaseTimeout 30s, got %v", cfg.PhaseTimeout)
	}
}

func TestLoadConfig_InvalidTimeout(t *testing.T) {
	const badTimeoutYAML = `version: "2.0"
source:
  system: holdin-admin
project:
  name: BadProject
  language: go
governance:
  mode: structured
testing:
  strategy: unit
limits:
  phase_timeout: not-a-duration
`
	path := writeTempYAML(t, badTimeoutYAML)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid phase_timeout, got nil")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/handoff.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}
