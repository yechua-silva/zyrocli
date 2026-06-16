package spec

import (
	"strings"
	"testing"
)

func TestCompile_NilCIO(t *testing.T) {
	entries, err := Compile(nil, "test-change")
	if err == nil {
		t.Fatal("expected error for nil CIO")
	}
	if entries != nil {
		t.Errorf("expected nil entries, got %d", len(entries))
	}
}

func TestCompile_FullCIO(t *testing.T) {
	cio := &CIO{
		Contract: Contract{
			ID:          "cio-001",
			Name:        "auth-model",
			Description: "Authentication and authorization model",
		},
		Interface: []IOMethod{
			{Name: "Login", Input: "Credentials", Output: "Token"},
			{Name: "Logout", Input: "Token", Output: "Status"},
		},
		Behavior: []Rule{
			{Description: "Login requires valid credentials", Precondition: "User exists", Postcondition: "Session created"},
		},
		Constraint: Constraint{
			Limitations: []string{"Rate limit: 10 req/min"},
			Invariants:  []string{"Token must be JWT"},
		},
		Operation: Operation{
			Steps: []string{"Validate input", "Check credentials", "Issue token"},
		},
		Testing: Testing{
			Approach: "unit + integration",
			Scopes:   []string{"auth", "session"},
		},
	}

	entries, err := Compile(cio, "scheduler-harness")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	expectedKey := "sdd/scheduler-harness/cio-auth-model"
	if entry.TopicKey != expectedKey {
		t.Errorf("expected topic_key %q, got %q", expectedKey, entry.TopicKey)
	}

	if !strings.Contains(entry.Content, "auth-model") {
		t.Errorf("expected content to contain 'auth-model', got:\n%s", entry.Content)
	}
	if !strings.Contains(entry.Content, "Login") {
		t.Errorf("expected content to contain 'Login', got:\n%s", entry.Content)
	}
	if !strings.Contains(entry.Content, "unit + integration") {
		t.Errorf("expected content to contain testing approach, got:\n%s", entry.Content)
	}
}

func TestCompile_EmptyCIO(t *testing.T) {
	cio := &CIO{
		Contract: Contract{
			Name: "empty-test",
		},
	}

	entries, err := Compile(cio, "test-change")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if !strings.Contains(entries[0].Content, "empty-test") {
		t.Errorf("expected content to contain contract name")
	}
}

func TestCompile_ZeroValueSafety(t *testing.T) {
	cio := &CIO{}
	// Should not panic
	entries, err := Compile(cio, "test-change")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	// Topic key uses "unnamed" fallback for empty contract name
	if entries[0].TopicKey != "sdd/test-change/cio-unnamed" {
		t.Errorf("expected topic_key with 'unnamed', got %q", entries[0].TopicKey)
	}
}

func TestGenerateTopicKey_Stable(t *testing.T) {
	cio := &CIO{
		Contract: Contract{Name: "auth-model"},
	}

	key1 := GenerateTopicKey(cio, "scheduler-harness")
	key2 := GenerateTopicKey(cio, "scheduler-harness")

	if key1 != key2 {
		t.Errorf("expected stable key, got %q vs %q", key1, key2)
	}
}

func TestGenerateTopicKey_EmptyContract(t *testing.T) {
	cio := &CIO{}
	key := GenerateTopicKey(cio, "test-change")
	expected := "sdd/test-change/cio-unnamed"
	if key != expected {
		t.Errorf("expected %q, got %q", expected, key)
	}
}

func TestToMarkdown_AllSections(t *testing.T) {
	cio := &CIO{
		Contract: Contract{
			Name: "test-model",
			ID:   "cio-x",
		},
		Interface: []IOMethod{
			{Name: "Get", Input: "ID", Output: "Item"},
		},
		Behavior: []Rule{
			{Description: "Rule 1", Precondition: "pre", Postcondition: "post"},
		},
		Constraint: Constraint{
			Limitations: []string{"limitation A"},
			Invariants:  []string{"invariant B"},
		},
		Operation: Operation{
			Steps: []string{"Step 1", "Step 2"},
		},
		Testing: Testing{
			Approach: "bdd",
			Scopes:   []string{"unit", "e2e"},
		},
	}

	md := cio.ToMarkdown()
	checks := []string{"test-model", "cio-x", "Get", "`ID` → `Item`", "Rule 1",
		"limitation A", "invariant B", "Step 1", "bdd", "unit, e2e"}
	for _, c := range checks {
		if !strings.Contains(md, c) {
			t.Errorf("expected markdown to contain %q", c)
		}
	}
}

func TestToMarkdown_ZeroValue(t *testing.T) {
	cio := &CIO{}
	// Must not panic
	md := cio.ToMarkdown()
	if md == "" {
		t.Error("expected non-empty markdown even for zero-value CIO")
	}
}
