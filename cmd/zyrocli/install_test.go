package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestInstallHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	installCmd.SetOut(buf)
	installCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"install", "--help"})
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil)
	installCmd.SetOut(nil)
	installCmd.SetErr(nil)

	if err != nil {
		t.Fatalf("install --help failed: %v", err)
	}
	if !strings.Contains(buf.String(), "ecosystem") {
		t.Errorf("expected help text, got: %s", buf.String())
	}
}

func TestInstallFlags(t *testing.T) {
	if f := installCmd.Flags().Lookup("mcp-dir"); f != nil {
		if f.Shorthand != "m" {
			t.Error("--mcp-dir shorthand should be 'm'")
		}
	}
}
