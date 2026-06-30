package main

import (
	"testing"
)

func TestProfileTUIShowsHelp(t *testing.T) {
	// Just test that the command exists and has the right name
	if profileTUICmd.Use != "tui" {
		t.Errorf("profileTUICmd.Use = %q, want %q", profileTUICmd.Use, "tui")
	}
}

func TestProfileTUIShortDescription(t *testing.T) {
	if profileTUICmd.Short == "" {
		t.Error("profileTUICmd.Short should not be empty")
	}
}

func TestProfileTUIHasRunE(t *testing.T) {
	if profileTUICmd.RunE == nil {
		t.Error("profileTUICmd.RunE should not be nil")
	}
}
