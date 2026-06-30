package scheduler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yechua-silva/zyrocli/internal/boomerang"
)

func TestWriteHandoffWithCriteria(t *testing.T) {
	// Use a temp dir to avoid polluting the real project
	tmpDir, err := os.MkdirTemp("", "handoff-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .zyro/handoffs dir
	handoffDir := filepath.Join(tmpDir, ".zyro", "handoffs")
	if err := os.MkdirAll(handoffDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Save and restore working dir
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	result := &Result{
		Phase:   "F3",
		Status:  StatusSuccess,
		Summary: "Boomerang: 2 tasks, quality=true, facts=1",
	}

	criteria := &boomerang.CriteriaSummary{
		Total:    3,
		Verified: 2,
		Pending:  0,
		Failed:   1,
	}

	if err := writeHandoff("F3", result, "F4", criteria); err != nil {
		t.Fatal(err)
	}

	// Read back and verify
	content, err := os.ReadFile(filepath.Join(handoffDir, "F3-handoff.md"))
	if err != nil {
		t.Fatal(err)
	}

	body := string(content)
	if !strings.Contains(body, "## Acceptance Criteria") {
		t.Error("handoff should contain 'Acceptance Criteria' section")
	}
	if !strings.Contains(body, "✅ Verified") {
		t.Error("handoff should contain Verified row")
	}
	if !strings.Contains(body, "❌ Failed") {
		t.Error("handoff should contain Failed row")
	}
	if !strings.Contains(body, "2") {
		t.Error("handoff should contain count 2 (Verified)")
	}
	if !strings.Contains(body, "1") {
		t.Error("handoff should contain count 1 (Failed)")
	}
}

func TestWriteHandoffWithoutCriteria(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "handoff-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	handoffDir := filepath.Join(tmpDir, ".zyro", "handoffs")
	if err := os.MkdirAll(handoffDir, 0755); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	result := &Result{
		Phase:   "F0",
		Status:  StatusSuccess,
		Summary: "F0 completed",
	}

	if err := writeHandoff("F0", result, "F1", nil); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(handoffDir, "F0-handoff.md"))
	if err != nil {
		t.Fatal(err)
	}

	body := string(content)
	if strings.Contains(body, "Acceptance Criteria") {
		t.Error("handoff should NOT contain 'Acceptance Criteria' section when nil")
	}
}
