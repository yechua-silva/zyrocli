package main

import (
	"strings"
	"testing"
)

func TestProfileSubcommands(t *testing.T) {
	cmds := profileCmd.Commands()
	names := make([]string, len(cmds))
	for i, c := range cmds {
		names[i] = c.Name()
	}
	combined := strings.Join(names, " ")
	if !strings.Contains(combined, "list") {
		t.Error("expected 'list' subcommand, got:", combined)
	}
	if !strings.Contains(combined, "set") {
		t.Error("expected 'set' subcommand, got:", combined)
	}
	if !strings.Contains(combined, "tui") {
		t.Error("expected 'tui' subcommand, got:", combined)
	}
}

func TestProfileSetInvalidAgent(t *testing.T) {
	if profileSetCmd.Args == nil {
		t.Error("set command should have Args validator")
	}
}

func TestProfileSetRequiresTwoArgs(t *testing.T) {
	err := profileSetCmd.Args(nil, []string{"only-one"})
	if err == nil {
		t.Error("expected error for 1 arg, got nil")
	}
}
