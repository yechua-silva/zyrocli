package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runTestHandoffYAML is a minimal valid handoff used for run tests that need
// handoff.yaml present.
const runTestHandoffYAML = `version: "2.0"
source:
  system: holdin-admin
project:
  name: TestRun
  language: go
  repository: github.com/secko/test
validated_idea:
  problem: "test"
  solution: "test"
  rationale: "test"
user_story:
  story: "test"
  acceptance: "test"
governance:
  mode: structured
testing:
  strategy: unit
limits:
  max_tasks: 15
  max_lines: 400
  max_loops: 5
  phase_timeout: "10m"
`

// writeHandoffInTemp creates a temp dir with a handoff.yaml and returns the dir path.
func writeHandoffInTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "handoff.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// runCmdDirect calls runCmd.RunE directly with the given setup.
// It resets flags, optionally creates a handoff.yaml, and invokes RunE.
func runCmdDirect(t *testing.T, handoffDir string, setup func()) (string, error) {
	t.Helper()

	// Reset package-level flags
	runPhase = ""

	// Change to handoffDir (must exist even if no handoff.yaml — for test isolation)
	origDir, _ := os.Getwd()
	os.Chdir(handoffDir)

	// Run optional setup (flag overrides, etc.)
	if setup != nil {
		setup()
	}

	buf := new(bytes.Buffer)
	runCmd.SetOut(buf)
	runCmd.SetErr(buf)
	err := runCmd.RunE(runCmd, []string{})
	runCmd.SetOut(nil)
	runCmd.SetErr(nil)

	os.Chdir(origDir)
	return buf.String(), err
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRunHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	runCmd.SetOut(buf)
	runCmd.SetErr(buf)
	err := runCmd.Help()
	runCmd.SetOut(nil)
	runCmd.SetErr(nil)

	if err != nil {
		t.Fatalf("Help() failed: %v", err)
	}
	output := buf.String()

	// The Short description appears in the help output via the command list
	if !strings.Contains(output, "F1→F2→F3→F4") && !strings.Contains(output, "SDD pipeline") {
		t.Errorf("expected help output to describe pipeline, got: %s", output)
	}
	if !strings.Contains(output, "--phase") {
		t.Errorf("expected help output to describe --phase, got: %s", output)
	}
}

func TestRunMissingHandoff(t *testing.T) {
	// Temp dir with NO handoff.yaml
	tmpDir := t.TempDir()
	output, err := runCmdDirect(t, tmpDir, nil)

	if err == nil {
		t.Fatal("expected error for missing handoff.yaml, got nil")
	}
	if !strings.Contains(err.Error(), "handoff.yaml not found") {
		t.Errorf("expected error about missing handoff.yaml, got: %v", err)
	}
	if output != "" {
		t.Errorf("expected empty output on error, got: %s", output)
	}
}

func TestRunInvalidPhase(t *testing.T) {
	handoffDir := writeHandoffInTemp(t, runTestHandoffYAML)

	_, err := runCmdDirect(t, handoffDir, func() {
		runPhase = "F5"
	})
	if err == nil {
		t.Fatal("expected error for invalid phase F5, got nil")
	}
	if !strings.Contains(err.Error(), "invalid phase") {
		t.Errorf("expected error about invalid phase, got: %v", err)
	}
	if !strings.Contains(err.Error(), "F5") {
		t.Errorf("expected error to mention 'F5', got: %v", err)
	}
}

func TestRunFlagsParsed(t *testing.T) {
	// Reset flags to defaults.
	runPhase = ""

	flags := runCmd.Flags()

	pf, err := flags.GetString("phase")
	if err != nil {
		t.Fatalf("--phase flag not registered: %v", err)
	}
	if pf != "" {
		t.Errorf("--phase should default to empty string, got %q", pf)
	}

	if f := flags.Lookup("phase"); f == nil || f.Shorthand != "p" {
		t.Error("--phase shorthand should be 'p'")
	}
}

func TestRunPhaseFlagValid(t *testing.T) {
	handoffDir := writeHandoffInTemp(t, runTestHandoffYAML)

	output, err := runCmdDirect(t, handoffDir, func() {
		runPhase = "F2"
	})
	if err != nil {
		t.Fatalf("run --phase F2 failed: %v", err)
	}
	if !strings.Contains(output, "Running phase F2") {
		t.Errorf("expected output to mention 'Running phase F2', got: %s", output)
	}
	if !strings.Contains(output, "Results") {
		t.Errorf("expected output to show Results, got: %s", output)
	}
}



