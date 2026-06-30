package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestContextHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	contextCmd.SetOut(buf)
	contextCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"context", "--help"})
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil)
	contextCmd.SetOut(nil)
	contextCmd.SetErr(nil)

	if err != nil {
		t.Fatalf("context --help failed: %v", err)
	}
	if !strings.Contains(buf.String(), "full context") {
		t.Errorf("expected help text containing 'full context', got: %s", buf.String())
	}
}

func TestContextFormatFlag(t *testing.T) {
	if f := contextCmd.Flags().Lookup("format"); f == nil {
		t.Error("--format flag not found")
	}
	if f := contextCmd.Flags().Lookup("format"); f != nil && f.Shorthand != "f" {
		t.Error("--format shorthand should be 'f'")
	}
}
