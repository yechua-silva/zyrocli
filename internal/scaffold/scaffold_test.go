package scaffold

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestScaffoldCreatesAllFiles(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		ProjectName:     "My Cool App",
		Language:        "Go",
		Module:          "github.com/my-cool-app",
		Problem:         "Need something awesome",
		SuccessCriteria: "Works perfectly",
		Version:         "2.0",
		Source:          "holdin-admin",
		ScaffoldDir:     filepath.Join(tmpDir, "output"),
	}

	result, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.TargetDir != cfg.ScaffoldDir {
		t.Errorf("TargetDir = %q, want %q", result.TargetDir, cfg.ScaffoldDir)
	}
	if result.FilesCreated != 9 {
		t.Errorf("FilesCreated = %d, want 9", result.FilesCreated)
	}

	// Verify all expected files exist.
	files := []string{
		"AGENT.md",
		"opencode.json",
		"handoff.yaml",
		".gitignore",
		"README.md",
		"cmd/my-cool-app/main.go",
		"scripts/explorer.py",
		"scripts/test-runner.py",
		"scripts/linter.py",
	}
	for _, f := range files {
		fullPath := filepath.Join(tmpDir, "output", f)
		if _, err := os.Stat(fullPath); err != nil {
			t.Errorf("missing file %s: %v", f, err)
		}
	}

	// Verify empty directories exist.
	dirs := []string{
		"skills",
		"docs/contexto_proyecto",
		"docs/recursos",
		"internal",
	}
	for _, d := range dirs {
		fullPath := filepath.Join(tmpDir, "output", d)
		if info, err := os.Stat(fullPath); err != nil {
			t.Errorf("missing directory %s: %v", d, err)
		} else if !info.IsDir() {
			t.Errorf("%s is not a directory", d)
		}
	}

	// Verify rendered content is non-empty.
	agentContent, err := os.ReadFile(filepath.Join(tmpDir, "output", "AGENT.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(agentContent) == 0 {
		t.Error("AGENT.md is empty")
	}
	if !bytes.Contains(agentContent, []byte("## Tools por Fase")) {
		t.Error("AGENT.md missing ## Tools por Fase section")
	}

	readmeContent, err := os.ReadFile(filepath.Join(tmpDir, "output", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(readmeContent) == "" {
		t.Error("README.md is empty")
	}

	// Verify state file was created
	statePath := filepath.Join(tmpDir, "output", ".zyro", "state.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Errorf("missing state file: %v", err)
	} else {
		stateData, err := os.ReadFile(statePath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(stateData, []byte(`"initialized": true`)) {
			t.Error("state file missing initialized: true")
		}
	}
}

func TestScaffoldNameNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // relative cmd path
	}{
		{
			name:     "spaces and special chars",
			input:    "My Cool App_v1",
			expected: "cmd/my-cool-app-v1/main.go",
		},
		{
			name:     "already kebab",
			input:    "my-app",
			expected: "cmd/my-app/main.go",
		},
		{
			name:     "leading/trailing specials",
			input:    "_  _my_app__  ",
			expected: "cmd/my-app/main.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			cfg := Config{
				ProjectName: tt.input,
				Language:    "Go",
				Module:      "github.com/test",
				ScaffoldDir: filepath.Join(tmpDir, "out"),
			}

			result, err := Run(cfg)
			if err != nil {
				t.Fatalf("Run failed: %v", err)
			}

			if result.TargetDir != filepath.Join(tmpDir, "out") {
				t.Errorf("TargetDir = %q", filepath.Join(tmpDir, "out"))
			}

			cmdPath := filepath.Join(tmpDir, "out", tt.expected)
			if _, err := os.Stat(cmdPath); err != nil {
				t.Errorf("expected cmd at %s: %v", cmdPath, err)
			}
		})
	}
}

func TestScaffoldExistingDir(t *testing.T) {
	tmpDir := t.TempDir()
	existing := filepath.Join(tmpDir, "exists")

	if err := os.MkdirAll(existing, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		ProjectName: "test",
		ScaffoldDir: existing,
	}

	_, err := Run(cfg)
	if err == nil {
		t.Fatal("expected error for existing directory, got nil")
	}
}

func TestScaffoldModuleDefault(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		ProjectName: "test-module",
		ScaffoldDir: filepath.Join(tmpDir, "out"),
		// Module is empty — should default to github.com/test-module
	}

	result, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.TargetDir != filepath.Join(tmpDir, "out") {
		t.Errorf("TargetDir = %q", filepath.Join(tmpDir, "out"))
	}
}

func TestWriteProjectCleansUpOnError(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "out")

	// Create the target directory.
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Place a file where WriteProject expects a directory.
	// The output includes "cmd/<name>/main.go", so creating "cmd" as a file blocks it.
	blockFile := filepath.Join(outDir, "cmd")
	if err := os.WriteFile(blockFile, []byte("blocker"), 0644); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"cmd/test/main.go": "package main",
		"AGENT.md":         "# hello",
	}

	err := WriteProject(outDir, files)
	if err == nil {
		t.Fatal("expected error for path conflict, got nil")
	}

	// Verify cleanup removed the entire target directory.
	if _, statErr := os.Stat(outDir); statErr == nil {
		t.Error("expected target directory to be removed on cleanup")
	}
}

// TestRenderFuncs verifies the template FuncMap works correctly.
func TestScaffoldScriptsExist(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		ProjectName: "scripts-test",
		ScaffoldDir: filepath.Join(tmpDir, "output"),
	}

	result, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.FilesCreated != 9 {
		t.Errorf("FilesCreated = %d, want 9", result.FilesCreated)
	}

	scripts := []string{
		"scripts/explorer.py",
		"scripts/test-runner.py",
		"scripts/linter.py",
	}
	for _, s := range scripts {
		fullPath := filepath.Join(tmpDir, "output", s)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Errorf("missing script %s: %v", s, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("script %s is empty", s)
		}
	}
}

func TestScriptsExecutable(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		ProjectName: "exec-test",
		ScaffoldDir: filepath.Join(tmpDir, "output"),
	}

	_, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	scripts := []string{
		"scripts/explorer.py",
		"scripts/test-runner.py",
		"scripts/linter.py",
	}
	for _, s := range scripts {
		fullPath := filepath.Join(tmpDir, "output", s)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("failed to read %s: %v", s, err)
			continue
		}
		if len(content) < 20 {
			t.Errorf("%s too short (%d bytes)", s, len(content))
		}
		// Verify shebang line.
		const shebang = "#!/usr/bin/env python3"
		if len(content) < len(shebang) || string(content[:len(shebang)]) != shebang {
			t.Errorf("%s missing shebang line or wrong shebang", s)
		}
	}
}

func TestPythonExplorerRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Python runtime test in short mode")
	}

	// Check if python3 is available.
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not found in PATH")
	}

	tmpDir := t.TempDir()

	// Scaffold into the temp directory so scripts/explorer.py exists.
	cfg := Config{
		ProjectName: "explorer-test",
		Language:    "Go",
		Module:      "github.com/test/explorer",
		ScaffoldDir: filepath.Join(tmpDir, "output"),
	}
	_, err := Run(cfg)
	if err != nil {
		t.Fatalf("scaffold Run failed: %v", err)
	}

	// Run explorer.py on the scaffold output directory.
	cmd := exec.Command("python3", "scripts/explorer.py", "--path", ".")
	cmd.Dir = filepath.Join(tmpDir, "output")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("explorer.py failed: %v\noutput: %s", err, string(output))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("invalid JSON from explorer.py: %v\nraw: %s", err, string(output))
	}

	if _, ok := result["total_files"]; !ok {
		t.Error("output missing 'total_files' key")
	}
	if _, ok := result["languages"]; !ok {
		t.Error("output missing 'languages' key")
	}
	if total, ok := result["total_files"].(float64); !ok || total < 1 {
		t.Errorf("expected total_files >= 1, got %v", result["total_files"])
	}
}

func TestRenderFuncs(t *testing.T) {
	r := NewRenderer()

	// Register a test template with all funcs.
	cfg := Config{
		ProjectName: "Test Project",
		Module:      "github.com/test-project",
	}

	result, err := r.Render("templates/go-project/AGENT.md.tmpl", cfg)
	if err != nil {
		t.Fatalf("Render AGENT.md failed: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("rendered AGENT.md is empty")
	}

	// Verify opencode.json renders (static template, no directives).
	jsonResult, err := r.Render("templates/go-project/opencode.json.tmpl", cfg)
	if err != nil {
		t.Fatalf("Render opencode.json failed: %v", err)
	}
	if len(jsonResult) == 0 {
		t.Fatal("rendered opencode.json is empty")
	}
}

func TestScaffoldWithLaunchOpenCode(t *testing.T) {
	// LaunchOpenCode is handled by the CLI layer, not by scaffold.Run.
	// This test verifies that setting it to true does not cause errors.
	tmpDir := t.TempDir()

	cfg := Config{
		ProjectName:     "open-app",
		Language:        "Go",
		Module:          "github.com/test/open-app",
		Problem:         "test problem",
		SuccessCriteria: "test criteria",
		Version:         "2.0",
		Source:          "holdin-admin",
		ScaffoldDir:     filepath.Join(tmpDir, "open-app"),
		LaunchOpenCode:  true,
	}

	result, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run with LaunchOpenCode=true failed: %v", err)
	}

	if result.TargetDir != cfg.ScaffoldDir {
		t.Errorf("TargetDir = %q, want %q", result.TargetDir, cfg.ScaffoldDir)
	}
	if result.FilesCreated != 9 {
		t.Errorf("FilesCreated = %d, want 9", result.FilesCreated)
	}

	// Verify open-app/main.go was created correctly.
	cmdPath := filepath.Join(tmpDir, "open-app", "cmd", "open-app", "main.go")
	if _, err := os.Stat(cmdPath); err != nil {
		t.Errorf("expected cmd at %s: %v", cmdPath, err)
	}
}

func TestReadScript(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		wantErr bool
	}{
		{name: "valid explorer.py", script: "explorer.py", wantErr: false},
		{name: "valid test-runner.py", script: "test-runner.py", wantErr: false},
		{name: "valid linter.py", script: "linter.py", wantErr: false},
		{name: "invalid name", script: "nonexistent.py", wantErr: true},
		{name: "empty name", script: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := ReadScript(tt.script)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadScript(%q) failed: %v", tt.script, err)
			}
			if len(data) == 0 {
				t.Errorf("ReadScript(%q) returned empty content", tt.script)
			}
			if string(data[:len("#!/usr/bin/env python3")]) != "#!/usr/bin/env python3" {
				t.Errorf("ReadScript(%q) missing shebang line", tt.script)
			}
		})
	}
}
