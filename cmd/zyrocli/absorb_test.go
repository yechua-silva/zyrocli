package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestAbsorbHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	absorbCmd.SetOut(buf)
	absorbCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"absorb", "--help"})
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil)
	absorbCmd.SetOut(nil)
	absorbCmd.SetErr(nil)

	if err != nil {
		t.Fatalf("absorb --help failed: %v", err)
	}
	if !strings.Contains(buf.String(), "markdown") {
		t.Errorf("expected 'markdown' in help, got: %s", buf.String())
	}
}

func TestAbsorbDirFlag(t *testing.T) {
	if f := absorbCmd.Flags().Lookup("dir"); f == nil {
		t.Error("--dir flag not found")
	}
	if f := absorbCmd.Flags().Lookup("dir"); f != nil && f.Shorthand != "d" {
		t.Error("--dir shorthand should be 'd'")
	}
}
