package handoff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validYAMLv20 is a complete, valid handoff.yaml v2.0 fixture used across tests.
const validYAMLv20 = `version: "2.0"
source:
  system: holdin-admin
  url: ""
project:
  name: TestProject
  language: go
  repository: github.com/test/project
validated_idea:
  problem: "test problem"
  solution: "test solution"
  rationale: "test rationale"
user_story:
  story: "test story"
  acceptance: "test acceptance"
mvp:
  scope: "test scope"
  features:
    - "feature one"
    - "feature two"
governance:
  mode: structured
  module: github.com/test/project
  go_version: "1.26"
  strict_tdd: false
testing:
  strategy: table-driven
  golden: false
  mock: "stdlib interfaces"
limits:
  max_tasks: 15
  max_lines: 400
  max_loops: 5
  phase_timeout: 10m
  chained_prs: false
`

// requiredOnlyYAML is the minimal valid v2.0 handoff file.
const requiredOnlyYAML = `version: "2.0"
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

// writeTempYAML writes content to a temporary file and returns the path.
func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "handoff.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// --- Parser tests ---

func TestParseFile(t *testing.T) {
	path := writeTempYAML(t, validYAMLv20)

	got, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse(%q) unexpected error: %v", path, err)
	}
	if got == nil {
		t.Fatal("Parse returned nil payload")
	}
	if got.Version != "2.0" {
		t.Errorf("expected version 2.0, got %q", got.Version)
	}
	if got.Source.System != "holdin-admin" {
		t.Errorf("expected source.system 'holdin-admin', got %q", got.Source.System)
	}
	if got.Project.Name != "TestProject" {
		t.Errorf("expected project.name 'TestProject', got %q", got.Project.Name)
	}
	if got.Project.Language != "go" {
		t.Errorf("expected project.language 'go', got %q", got.Project.Language)
	}
	if got.Governance.Mode != "structured" {
		t.Errorf("expected governance.mode 'structured', got %q", got.Governance.Mode)
	}
	if got.Governance.Module != "github.com/test/project" {
		t.Errorf("expected governance.module 'github.com/test/project', got %q", got.Governance.Module)
	}
	if got.Governance.GoVersion != "1.26" {
		t.Errorf("expected governance.go_version '1.26', got %q", got.Governance.GoVersion)
	}
	if got.Testing.Strategy != "table-driven" {
		t.Errorf("expected testing.strategy 'table-driven', got %q", got.Testing.Strategy)
	}
	if got.MVP.Scope != "test scope" {
		t.Errorf("expected mvp.scope 'test scope', got %q", got.MVP.Scope)
	}
	if len(got.MVP.Features) != 2 {
		t.Errorf("expected 2 mvp.features, got %d", len(got.MVP.Features))
	}
	if got.Limits.MaxTasks != 15 {
		t.Errorf("expected limits.max_tasks 15, got %d", got.Limits.MaxTasks)
	}
	if got.Limits.MaxLines != 400 {
		t.Errorf("expected limits.max_lines 400, got %d", got.Limits.MaxLines)
	}
	if got.Limits.MaxLoops != 5 {
		t.Errorf("expected limits.max_loops 5, got %d", got.Limits.MaxLoops)
	}
	if got.Limits.PhaseTimeout != "10m" {
		t.Errorf("expected limits.phase_timeout '10m', got %q", got.Limits.PhaseTimeout)
	}
}

func TestParseFileNotFound(t *testing.T) {
	_, err := Parse("/nonexistent/handoff.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "cannot find") {
		t.Logf("error message: %v", err)
	}
}

func TestParseInvalidYAML(t *testing.T) {
	path := writeTempYAML(t, `version: "2.0"
source:
  system: holdin-admin
project: [invalid: yaml: broken
`)

	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
	if !strings.Contains(err.Error(), "yaml:") && !strings.Contains(err.Error(), "cannot") {
		t.Logf("error message: %v", err)
	}
}

func TestParseStdin(t *testing.T) {
	content := validYAMLv20
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	// Write content and close writer
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	w.Close()

	// Replace os.Stdin
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	got, err := Parse("-")
	if err != nil {
		t.Fatalf("Parse(\"-\") unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("Parse returned nil payload")
	}
	if got.Version != "2.0" {
		t.Errorf("expected version 2.0 from stdin, got %q", got.Version)
	}
	if got.Project.Name != "TestProject" {
		t.Errorf("expected project.name 'TestProject' from stdin, got %q", got.Project.Name)
	}
}

func TestParseStdinEmpty(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	w.Close() // no data written

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	_, err = Parse("-")
	if err == nil {
		t.Fatal("expected error for empty stdin, got nil")
	}
	if !strings.Contains(err.Error(), "empty input") {
		t.Errorf("expected 'empty input' error, got: %v", err)
	}
}

// --- Validate tests ---

func TestValidateValid(t *testing.T) {
	payload := &Payload{
		Version: "2.0",
		Source: Source{
			System: "holdin-admin",
		},
		Project: Project{
			Name:     "TestProject",
			Language: "go",
		},
		Governance: Governance{
			Mode: "structured",
		},
		Testing: Testing{
			Strategy: "unit",
		},
	}

	if err := Validate(payload); err != nil {
		t.Fatalf("Validate(valid payload) unexpected error: %v", err)
	}
}

func TestValidateMissingFields(t *testing.T) {
	tests := []struct {
		name     string
		payload  *Payload
		expected string // substring expected in error
	}{
		{
			name: "missing source.system",
			payload: &Payload{
				Version: "2.0",
				Source:  Source{System: ""},
				Project: Project{Name: "Test", Language: "go"},
				Governance: Governance{Mode: "structured"},
				Testing:    Testing{Strategy: "unit"},
			},
			expected: "source.system",
		},
		{
			name: "missing project.name",
			payload: &Payload{
				Version: "2.0",
				Source:  Source{System: "holdin-admin"},
				Project: Project{Name: "", Language: "go"},
				Governance: Governance{Mode: "structured"},
				Testing:    Testing{Strategy: "unit"},
			},
			expected: "project.name",
		},
		{
			name: "missing project.language",
			payload: &Payload{
				Version: "2.0",
				Source:  Source{System: "holdin-admin"},
				Project: Project{Name: "Test", Language: ""},
				Governance: Governance{Mode: "structured"},
				Testing:    Testing{Strategy: "unit"},
			},
			expected: "project.language",
		},
		{
			name: "missing governance.mode",
			payload: &Payload{
				Version: "2.0",
				Source:  Source{System: "holdin-admin"},
				Project: Project{Name: "Test", Language: "go"},
				Governance: Governance{Mode: ""},
				Testing:    Testing{Strategy: "unit"},
			},
			expected: "governance.mode",
		},
		{
			name: "missing testing.strategy",
			payload: &Payload{
				Version: "2.0",
				Source:  Source{System: "holdin-admin"},
				Project: Project{Name: "Test", Language: "go"},
				Governance: Governance{Mode: "structured"},
				Testing:    Testing{Strategy: ""},
			},
			expected: "testing.strategy",
		},
		{
			name: "wrong version",
			payload: &Payload{
				Version: "1.0",
				Source:  Source{System: "holdin-admin"},
				Project: Project{Name: "Test", Language: "go"},
				Governance: Governance{Mode: "structured"},
				Testing:    Testing{Strategy: "unit"},
			},
			expected: "version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.payload)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expected) {
				t.Errorf("expected error containing %q, got: %v", tt.expected, err)
			}
		})
	}
}

func TestValidateMultiError(t *testing.T) {
	payload := &Payload{
		Version:     "1.0",
		Source:      Source{System: ""},
		Project:     Project{Name: "", Language: ""},
		Governance:  Governance{Mode: ""},
		Testing:     Testing{Strategy: ""},
	}

	err := Validate(payload)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errStr := err.Error()
	// We should have 6 violations: version, source.system, project.name, project.language, governance.mode, testing.strategy
	for _, field := range []string{"version", "source.system", "project.name", "project.language", "governance.mode", "testing.strategy"} {
		if !strings.Contains(errStr, field) {
			t.Errorf("expected error to contain %q, got: %v", field, errStr)
		}
	}
}

// --- Integration tests ---

func TestIntegrationParseValidateFull(t *testing.T) {
	path := writeTempYAML(t, validYAMLv20)

	payload, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if err := Validate(payload); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
}

func TestIntegrationParseValidateMinimal(t *testing.T) {
	path := writeTempYAML(t, requiredOnlyYAML)

	payload, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if err := Validate(payload); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// Verify zero-values for optional fields
	if payload.ValidatedIdea.Problem != "" {
		t.Errorf("expected empty validated_idea.problem, got %q", payload.ValidatedIdea.Problem)
	}
	if payload.UserStory.Story != "" {
		t.Errorf("expected empty user_story.story, got %q", payload.UserStory.Story)
	}
	if payload.MVP.Scope != "" {
		t.Errorf("expected empty mvp.scope, got %q", payload.MVP.Scope)
	}
	if payload.Limits.MaxLoops != 0 {
		t.Errorf("expected default limits.max_loops 0, got %d", payload.Limits.MaxLoops)
	}
	if payload.Limits.PhaseTimeout != "" {
		t.Errorf("expected empty limits.phase_timeout, got %q", payload.Limits.PhaseTimeout)
	}
}
