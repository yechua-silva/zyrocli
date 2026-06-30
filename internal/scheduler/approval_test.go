package scheduler

import (
	"bufio"
	"strings"
	"testing"

	"github.com/yechua-silva/zyrocli/internal/boomerang"
)

func TestApprovalGateBlocksFailed(t *testing.T) {
	// Save original reader and restore after test
	orig := stdinReader
	defer func() { stdinReader = orig }()

	// Simulate "s" input — but criteria should block before reading
	stdinReader = bufio.NewReader(strings.NewReader("s\n"))

	summary := &boomerang.CriteriaSummary{
		Total:    3,
		Verified: 1,
		Pending:  0,
		Failed:   2,
	}

	approved, err := ApprovalGate("F3", "test summary", summary)
	if err != nil {
		t.Fatal(err)
	}
	if approved {
		t.Error("expected approval to be blocked (Failed > 0)")
	}
}

func TestApprovalGateAllowsVerified(t *testing.T) {
	orig := stdinReader
	defer func() { stdinReader = orig }()

	stdinReader = bufio.NewReader(strings.NewReader("s\n"))

	summary := &boomerang.CriteriaSummary{
		Total:    3,
		Verified: 3,
		Pending:  0,
		Failed:   0,
	}

	approved, err := ApprovalGate("F3", "test summary", summary)
	if err != nil {
		t.Fatal(err)
	}
	if !approved {
		t.Error("expected approval to pass (all verified)")
	}
}

func TestApprovalGateNoCriteria(t *testing.T) {
	orig := stdinReader
	defer func() { stdinReader = orig }()

	stdinReader = bufio.NewReader(strings.NewReader("s\n"))

	approved, err := ApprovalGate("F3", "test summary", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !approved {
		t.Error("expected approval to pass (nil criteria)")
	}
}

func TestPromptApprovalBackwardCompat(t *testing.T) {
	orig := stdinReader
	defer func() { stdinReader = orig }()

	stdinReader = bufio.NewReader(strings.NewReader("s\n"))

	approved, err := PromptApproval("F3", "test summary")
	if err != nil {
		t.Fatal(err)
	}
	if !approved {
		t.Error("expected PromptApproval to pass (backward compat)")
	}
}

func TestGuidedApprovalWithCriteria(t *testing.T) {
	orig := stdinReader
	defer func() { stdinReader = orig }()

	stdinReader = bufio.NewReader(strings.NewReader("s\n"))

	g := NewGuidedApproval("F3", "test summary")
	g.WithCriteria(&boomerang.CriteriaSummary{
		Total:    2,
		Verified: 2,
		Pending:  0,
		Failed:   0,
	})

	approved, err := g.PromptApproval()
	if err != nil {
		t.Fatal(err)
	}
	if !approved {
		t.Error("expected guided approval to pass")
	}
}

func TestGuidedApprovalWithFailedCriteria(t *testing.T) {
	// No need to mock stdin since failed criteria block before reading
	g := NewGuidedApproval("F3", "test summary")
	g.WithCriteria(&boomerang.CriteriaSummary{
		Total:    2,
		Verified: 0,
		Pending:  0,
		Failed:   2,
	})

	approved, err := g.PromptApproval()
	if err != nil {
		t.Fatal(err)
	}
	if approved {
		t.Error("expected guided approval to be blocked (Failed > 0)")
	}
}
