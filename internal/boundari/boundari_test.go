package boundari

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPolicy(t *testing.T) {
	p, err := LoadPolicy("F0", []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	if p.Phase != "F0" {
		t.Errorf("expected F0, got %s", p.Phase)
	}
}

func TestLoadPolicyNotFound(t *testing.T) {
	_, err := LoadPolicy("F9", []string{"/nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent policy")
	}
}

func TestLoadDefaultPolicy(t *testing.T) {
	t.Run("F0", func(t *testing.T) {
		p := LoadDefaultPolicy("F0")
		if p.Phase != "F0" {
			t.Errorf("expected F0, got %s", p.Phase)
		}
		if len(p.Tools) == 0 {
			t.Error("expected at least one tool rule")
		}
	})

	t.Run("PRE-F0", func(t *testing.T) {
		p := LoadDefaultPolicy("PRE-F0")
		if p.Phase != "PRE-F0" {
			t.Errorf("expected PRE-F0, got %s", p.Phase)
		}
		if p.Budget.MaxToolCalls != 30 {
			t.Errorf("expected MaxToolCalls=30, got %d", p.Budget.MaxToolCalls)
		}
		if p.Budget.MaxRuntimeSecs != 300 {
			t.Errorf("expected MaxRuntimeSecs=300, got %d", p.Budget.MaxRuntimeSecs)
		}
		// write_file debe ser allow
		writeRule := p.GetRule("write_file")
		if writeRule == nil || writeRule.Action != ActionAllow {
			t.Error("PRE-F0 should allow write_file")
		}
		// task_create debe ser allow
		dispatchRule := p.GetRule("task_create")
		if dispatchRule == nil || dispatchRule.Action != ActionAllow {
			t.Error("PRE-F0 should allow task_create")
		}
		// edit_file debe ser deny
		editRule := p.GetRule("edit_file")
		if editRule == nil || editRule.Action != ActionDeny {
			t.Error("PRE-F0 should deny edit_file")
		}
		// execute_command debe ser deny
		execRule := p.GetRule("execute_command")
		if execRule == nil || execRule.Action != ActionDeny {
			t.Error("PRE-F0 should deny execute_command")
		}
	})
}

func TestValidatePolicy(t *testing.T) {
	tests := []struct {
		name    string
		policy  *Policy
		wantErr bool
	}{
		{"valid", &Policy{Version: "1.0", Phase: "F0", Budget: Budget{MaxToolCalls: 10, MaxRuntimeSecs: 100}, Tools: []ToolRule{{Name: "test", Action: ActionAllow}}}, false},
		{"no phase", &Policy{Version: "1.0", Tools: []ToolRule{{Name: "test", Action: ActionAllow}}}, true},
		{"negative budget", &Policy{Phase: "F0", Budget: Budget{MaxToolCalls: -1}, Tools: []ToolRule{{Name: "test", Action: ActionAllow}}}, true},
		{"empty tools", &Policy{Phase: "F0", Tools: []ToolRule{}}, true},
		{"invalid action", &Policy{Phase: "F0", Tools: []ToolRule{{Name: "test", Action: "invalid"}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePolicy(tt.policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnforcerCheckTool(t *testing.T) {
	p := LoadDefaultPolicy("F0")
	e := NewEnforcer(p)

	// F0 debe denegar write_file
	r := e.CheckTool("write_file", nil)
	if r.Allowed {
		t.Error("F0 should deny write_file")
	}

	// F0 debe permitir search_code
	r = e.CheckTool("search_code", nil)
	if !r.Allowed {
		t.Error("F0 should allow search_code")
	}

	// Tool no listada debe denegarse
	r = e.CheckTool("nonexistent_tool", nil)
	if r.Allowed {
		t.Error("unlisted tool should be denied")
	}
}

func TestEnforcerPhase3AllowsWrite(t *testing.T) {
	p := LoadDefaultPolicy("F3")
	e := NewEnforcer(p)
	r := e.CheckTool("write_file", nil)
	if !r.Allowed {
		t.Error("F3 should allow write_file")
	}
}

func TestEnforcerBudgetExceeded(t *testing.T) {
	p := LoadDefaultPolicy("F0")
	p.Budget.MaxToolCalls = 2
	e := NewEnforcer(p)

	// Agotar budget
	e.CheckTool("search_code", nil)
	e.CheckTool("search_code", nil)

	if !e.IsBudgetExceeded() {
		t.Error("budget should be exceeded after 2 calls")
	}

	// Tercera llamada debe fallar
	r := e.CheckTool("search_code", nil)
	if r.Allowed {
		t.Error("should deny when budget exceeded")
	}
}

func TestEnforcerApprovalRequired(t *testing.T) {
	p := &Policy{
		Phase: "F3",
		Budget: Budget{MaxToolCalls: 100, MaxRuntimeSecs: 1000},
		Tools: []ToolRule{
			{Name: "execute_command", Action: ActionApproval, RequireApproval: true},
		},
	}
	e := NewEnforcer(p)
	r := e.CheckTool("execute_command", nil)
	if r.Allowed {
		t.Error("execute_command should require approval")
	}
}

func TestEnforcerSaveAuditLog(t *testing.T) {
	ClearAuditLog()
	LogAudit(AuditEvent{Phase: "F0", Tool: "test", Allowed: true, Reason: "test"})
	LogAudit(AuditEvent{Phase: "F0", Tool: "test2", Allowed: false, Reason: "denied"})

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "audit.jsonl")
	if err := SaveAuditLog(path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("audit file should not be empty")
	}
}

func TestAllPoliciesLoad(t *testing.T) {
	phases := []string{"PRE-F0", "F0", "F1", "F2", "F3", "F4"}
	for _, phase := range phases {
		p, err := LoadPolicy(phase, []string{"."})
		if err != nil {
			t.Errorf("phase %s: %v", phase, err)
			continue
		}
		if p.Phase != phase {
			t.Errorf("phase %s: expected phase %s, got %s", phase, phase, p.Phase)
		}
	}
}

func TestEnforcerReset(t *testing.T) {
	p := LoadDefaultPolicy("F0")
	e := NewEnforcer(p)
	e.CheckTool("search_code", nil)
	if e.Usage().ToolCalls == 0 {
		t.Error("should have 1 tool call")
	}
	e.Reset()
	if e.Usage().ToolCalls != 0 {
		t.Error("should have 0 tool calls after reset")
	}
}
