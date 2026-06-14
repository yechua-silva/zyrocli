package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testHandoffYAML is a minimal valid handoff v2.0 used across init tests.
const testHandoffYAML = `version: "2.0"
source:
  system: holdin-admin
project:
  name: test-scaffold
  language: go
  repository: github.com/test/app
validated_idea:
  problem: "test problem"
  solution: "test solution"
  rationale: "test rationale"
user_story:
  story: "test story"
  acceptance: "test acceptance criteria"
governance:
  mode: structured
testing:
  strategy: unit
`

// writeTestHandoff writes a handoff YAML to a temp file and returns its path.
func writeTestHandoff(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "handoff.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// runInitCmd executes the root command with the given args.
// It resets flags and output before each run to avoid test pollution.
func runInitCmd(t *testing.T, args []string) (*bytes.Buffer, error) {
	t.Helper()

	// Reset package-level flags so previous tests don't leak state.
	scaffoldFlag = false
	opencodeFlag = false

	buf := new(bytes.Buffer)
	initCmd.SetOut(buf)
	initCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()

	// Clean up to avoid test pollution.
	rootCmd.SetArgs(nil)
	initCmd.SetOut(nil)
	initCmd.SetErr(nil)
	return buf, err
}

// withEmptyPATH runs fn with PATH set to a directory without opencode.
func withEmptyPATH(fn func()) {
	old := os.Getenv("PATH")
	os.Setenv("PATH", "/tmp")
	fn()
	os.Setenv("PATH", old)
}

func TestInitNoFlags(t *testing.T) {
	path := writeTestHandoff(t, testHandoffYAML)

	buf, err := runInitCmd(t, []string{"init", path})
	if err != nil {
		t.Fatalf("init without flags failed: %v", err)
	}
	if !strings.Contains(buf.String(), "OK") {
		t.Errorf("expected OK output for no-flags mode, got: %s", buf.String())
	}
}

func TestInitScaffoldFlag(t *testing.T) {
	path := writeTestHandoff(t, testHandoffYAML)

	// Run in a temp directory so scaffold creates files there.
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	buf, err := runInitCmd(t, []string{"init", path, "--scaffold"})
	if err != nil {
		t.Fatalf("init --scaffold failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Project scaffolded") {
		t.Errorf("expected scaffold output, got: %s", output)
	}
	if !strings.Contains(output, "Files created") {
		t.Errorf("expected files created line, got: %s", output)
	}

	// Verify the scaffold directory was created.
	targetDir := filepath.Join(tmpDir, "test-scaffold")
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		t.Errorf("scaffold directory %s was not created", targetDir)
	}
}

func TestInitOpenCodeWithoutScaffold(t *testing.T) {
	path := writeTestHandoff(t, testHandoffYAML)

	_, err := runInitCmd(t, []string{"init", path, "--opencode"})
	if err == nil {
		t.Fatal("expected error for --opencode without --scaffold, got nil")
	}
	if !strings.Contains(err.Error(), "requires --scaffold") {
		t.Errorf("expected error about requiring --scaffold, got: %v", err)
	}
}

func TestInitScaffoldOpenCode(t *testing.T) {
	path := writeTestHandoff(t, testHandoffYAML)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Clear PATH so exec.LookPath("opencode") fails — avoids launching the TUI.
	var buf *bytes.Buffer
	var err error
	withEmptyPATH(func() {
		buf, err = runInitCmd(t, []string{"init", path, "--scaffold", "--opencode"})
	})
	if err != nil {
		t.Fatalf("init --scaffold --opencode failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Project scaffolded") {
		t.Errorf("expected scaffold output, got: %s", output)
	}
	if !strings.Contains(output, "opencode not found") {
		t.Errorf("expected opencode not found warning, got: %s", output)
	}
}

func TestInitScaffoldFlagsParsed(t *testing.T) {
	// Reset flags to defaults before checking.
	scaffoldFlag = false
	opencodeFlag = false

	flags := initCmd.Flags()

	sf, err := flags.GetBool("scaffold")
	if err != nil {
		t.Fatalf("--scaffold flag not registered: %v", err)
	}
	if sf {
		t.Error("--scaffold should default to false")
	}

	of, err := flags.GetBool("opencode")
	if err != nil {
		t.Fatalf("--opencode flag not registered: %v", err)
	}
	if of {
		t.Error("--opencode should default to false")
	}

	// Verify short flags map correctly.
	if f := flags.Lookup("scaffold"); f == nil || f.Shorthand != "s" {
		t.Error("--scaffold shorthand should be 's'")
	}
	if f := flags.Lookup("opencode"); f == nil || f.Shorthand != "o" {
		t.Error("--opencode shorthand should be 'o'")
	}
}
