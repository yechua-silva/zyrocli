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

// runInitCmd executes the init command with the given args.
func runInitCmd(t *testing.T, args []string) (*bytes.Buffer, error) {
	t.Helper()

	// Reset package-level flags so previous tests don't leak state.
	useScaffold = false
	noOpenCode = false
	dryRun = false

	buf := new(bytes.Buffer)
	initCmd.SetOut(buf)
	initCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()

	rootCmd.SetArgs(nil)
	initCmd.SetOut(nil)
	initCmd.SetErr(nil)
	return buf, err
}

func TestInitNoFlags(t *testing.T) {
	path := writeTestHandoff(t, testHandoffYAML)

	// Run in a temp directory so init creates files there.
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	buf, err := runInitCmd(t, []string{"init", path, "--no-opencode"})
	if err != nil {
		t.Fatalf("init without flags failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Project structure created") {
		t.Errorf("expected output about project structure, got: %s", buf.String())
	}

	// Verify the project directory was created.
	targetDir := filepath.Join(tmpDir, "test-scaffold")
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		t.Errorf("project directory %s was not created", targetDir)
	}
}

func TestInitScaffoldFlag(t *testing.T) {
	path := writeTestHandoff(t, testHandoffYAML)

	// Run in a temp directory so scaffold creates files there.
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	buf, err := runInitCmd(t, []string{"init", path, "--scaffold", "--no-opencode"})
	if err != nil {
		t.Fatalf("init --scaffold failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Project structure created") {
		t.Errorf("expected project structure output, got: %s", output)
	}

	// Verify the project directory was created.
	targetDir := filepath.Join(tmpDir, "test-scaffold")
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		t.Errorf("project directory %s was not created", targetDir)
	}
}

func TestInitDryRun(t *testing.T) {
	path := writeTestHandoff(t, testHandoffYAML)

	buf, err := runInitCmd(t, []string{"init", path, "--dry-run"})
	if err != nil {
		t.Fatalf("init --dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "handoff valid") {
		t.Errorf("expected handoff valid output, got: %s", output)
	}
}

func TestInitFlagsParsed(t *testing.T) {
	// Reset flags to defaults before checking.
	useScaffold = false
	noOpenCode = false
	dryRun = false

	flags := initCmd.Flags()

	sf, err := flags.GetBool("scaffold")
	if err != nil {
		t.Fatalf("--scaffold flag not registered: %v", err)
	}
	if sf {
		t.Error("--scaffold should default to false")
	}

	nf, err := flags.GetBool("no-opencode")
	if err != nil {
		t.Fatalf("--no-opencode flag not registered: %v", err)
	}
	if nf {
		t.Error("--no-opencode should default to false")
	}

	// Verify short flags map correctly.
	if f := flags.Lookup("scaffold"); f == nil || f.Shorthand != "s" {
		t.Error("--scaffold shorthand should be 's'")
	}
}
