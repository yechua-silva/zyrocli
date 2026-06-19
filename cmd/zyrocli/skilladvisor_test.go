package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSkillAdvisorHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	skillAdvisorCmd.SetOut(buf)
	skillAdvisorCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"skill-advisor", "--help"})
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil)
	skillAdvisorCmd.SetOut(nil)
	skillAdvisorCmd.SetErr(nil)

	if err != nil {
		t.Fatalf("skill-advisor --help failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Search skills.sh") {
		t.Errorf("expected help text, got: %s", buf.String())
	}
}

func TestDetectProjectStack(t *testing.T) {
	// Test with go.mod
	dir := t.TempDir()
	os.WriteFile(dir+"/go.mod", []byte("module test"), 0644)
	ctx := detectProjectStack(dir)
	if ctx.Language != "go" {
		t.Errorf("expected go, got %s", ctx.Language)
	}

	// Test with package.json
	dir2 := t.TempDir()
	os.WriteFile(dir2+"/package.json", []byte("{}"), 0644)
	ctx2 := detectProjectStack(dir2)
	if ctx2.Language != "typescript" {
		t.Errorf("expected typescript, got %s", ctx2.Language)
	}

	// Test with no project files
	dir3 := t.TempDir()
	ctx3 := detectProjectStack(dir3)
	if ctx3.Language != "" {
		t.Errorf("expected empty, got %s", ctx3.Language)
	}
}

func TestHasSkillSpector(t *testing.T) {
	// This is a platform-dependent test, just verify it doesn't crash
	_ = hasSkillSpector()
}
