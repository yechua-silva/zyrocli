package scheduler

import (
	"bufio"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Mock PhaseRunner for scheduler tests
// ---------------------------------------------------------------------------

type mockRunner struct {
	name   Phase
	result *Result
	err    error
	delay  time.Duration // if > 0, simulates work; ctx.Done aborts it
}

func (m *mockRunner) Run(ctx context.Context, cfg *Config) (*Result, error) {
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return &Result{Phase: m.name, Status: StatusFail, Summary: "timeout"}, ctx.Err()
		case <-time.After(m.delay):
		}
	}
	return m.result, m.err
}

func (m *mockRunner) Name() Phase { return m.name }

// ---------------------------------------------------------------------------
// PromptApproval tests
// ---------------------------------------------------------------------------

func TestPromptApproval_ApproveY(t *testing.T) {
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("y\n"))
	t.Cleanup(func() { stdinReader = old })

	approved, err := PromptApproval(PhaseF1, "test summary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true for 'y'")
	}
}

func TestPromptApproval_ApproveYes(t *testing.T) {
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("yes\n"))
	t.Cleanup(func() { stdinReader = old })

	approved, err := PromptApproval(PhaseF1, "test summary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true for 'yes'")
	}
}

func TestPromptApproval_ApproveS(t *testing.T) {
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("s\n"))
	t.Cleanup(func() { stdinReader = old })

	approved, err := PromptApproval(PhaseF1, "test summary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true for 's'")
	}
}

func TestPromptApproval_ApproveSi(t *testing.T) {
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("si\n"))
	t.Cleanup(func() { stdinReader = old })

	approved, err := PromptApproval(PhaseF1, "test summary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true for 'si'")
	}
}

func TestPromptApproval_RejectN(t *testing.T) {
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("n\n"))
	t.Cleanup(func() { stdinReader = old })

	approved, err := PromptApproval(PhaseF1, "test summary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false for 'n'")
	}
}

func TestPromptApproval_RejectNo(t *testing.T) {
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("no\n"))
	t.Cleanup(func() { stdinReader = old })

	approved, err := PromptApproval(PhaseF1, "test summary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false for 'no'")
	}
}

func TestPromptApproval_InvalidRetryThenApprove(t *testing.T) {
	old := stdinReader
	// First input is invalid ("maybe"), then valid approve ("y")
	stdinReader = bufio.NewReader(strings.NewReader("maybe\ny\n"))
	t.Cleanup(func() { stdinReader = old })

	approved, err := PromptApproval(PhaseF1, "test summary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true after retry")
	}
}

func TestPromptApproval_InvalidRetryThenReject(t *testing.T) {
	old := stdinReader
	// First invalid ("xyz"), then reject ("n")
	stdinReader = bufio.NewReader(strings.NewReader("xyz\nn\n"))
	t.Cleanup(func() { stdinReader = old })

	approved, err := PromptApproval(PhaseF1, "test summary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false after retry and reject")
	}
}

// ---------------------------------------------------------------------------
// Scheduler Run tests (uses mock runners, no stdin via PromptApproval mock)
// ---------------------------------------------------------------------------

func makeSuccessRunners() []PhaseRunner {
	return []PhaseRunner{
		&mockRunner{name: PhaseF1, result: &Result{Phase: PhaseF1, Status: StatusSuccess, Summary: "F1 done"}},
		&mockRunner{name: PhaseF2, result: &Result{Phase: PhaseF2, Status: StatusSuccess, Summary: "F2 done"}},
		&mockRunner{name: PhaseF3, result: &Result{Phase: PhaseF3, Status: StatusSuccess, Summary: "F3 done"}},
		&mockRunner{name: PhaseF4, result: &Result{Phase: PhaseF4, Status: StatusSuccess, Summary: "F4 done"}},
	}
}

func TestScheduler_AllSucceed(t *testing.T) {
	cfg := &Config{PhaseTimeout: time.Minute}
	s := NewScheduler(cfg, makeSuccessRunners())

	// Override stdinReader to auto-approve all 4 phases
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("y\ny\ny\ny\n"))
	t.Cleanup(func() { stdinReader = old })

	results, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Status != StatusSuccess {
			t.Errorf("expected StatusSuccess for %s, got %s", r.Phase, r.Status)
		}
	}
}

func TestScheduler_AbortOnFailure(t *testing.T) {
	cfg := &Config{PhaseTimeout: time.Minute}
	runners := []PhaseRunner{
		&mockRunner{name: PhaseF1, result: &Result{Phase: PhaseF1, Status: StatusSuccess, Summary: "F1 done"}},
		&mockRunner{name: PhaseF2, result: &Result{Phase: PhaseF2, Status: StatusFail, Summary: "F2 failed"}},
		&mockRunner{name: PhaseF3, result: &Result{Phase: PhaseF3, Status: StatusSuccess, Summary: "should not run"}},
		&mockRunner{name: PhaseF4, result: &Result{Phase: PhaseF4, Status: StatusSuccess, Summary: "should not run"}},
	}
	s := NewScheduler(cfg, runners)

	// Auto-approve F1 before it fails
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("y\n"))
	t.Cleanup(func() { stdinReader = old })

	results, err := s.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for phase failure")
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (F1+F2), got %d", len(results))
	}
	if results[0].Phase != PhaseF1 || results[0].Status != StatusSuccess {
		t.Errorf("expected F1 success, got %s=%s", results[0].Phase, results[0].Status)
	}
	if results[1].Phase != PhaseF2 || results[1].Status != StatusFail {
		t.Errorf("expected F2 fail, got %s=%s", results[1].Phase, results[1].Status)
	}
}

func TestScheduler_AbortOnRunnerError(t *testing.T) {
	cfg := &Config{PhaseTimeout: time.Minute}
	runners := []PhaseRunner{
		&mockRunner{name: PhaseF1, result: &Result{Phase: PhaseF1, Status: StatusSuccess, Summary: "F1 done"}},
		&mockRunner{name: PhaseF2, result: nil, err: errors.New("unexpected crash")},
		&mockRunner{name: PhaseF3, result: &Result{Phase: PhaseF3, Status: StatusSuccess, Summary: "should not run"}},
	}
	s := NewScheduler(cfg, runners)

	// Auto-approve F1
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("y\n"))
	t.Cleanup(func() { stdinReader = old })

	results, err := s.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for runner error")
	}
	if !strings.Contains(err.Error(), "unexpected crash") {
		t.Errorf("expected error to mention 'unexpected crash', got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (F1+F2 fail), got %d", len(results))
	}
}

func TestScheduler_EmptyRunners(t *testing.T) {
	cfg := &Config{PhaseTimeout: time.Minute}
	s := NewScheduler(cfg, []PhaseRunner{})

	results, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error with empty runners: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// RunPhase tests
// ---------------------------------------------------------------------------

func TestRunPhase_Isolation(t *testing.T) {
	runners := []PhaseRunner{
		&mockRunner{name: PhaseF1, result: &Result{Phase: PhaseF1, Status: StatusSuccess, Summary: "F1"}},
		&mockRunner{name: PhaseF2, result: &Result{Phase: PhaseF2, Status: StatusSuccess, Summary: "F2"}},
	}
	s := NewScheduler(&Config{PhaseTimeout: time.Minute}, runners)

	// Only run F2; verify it returns without running F1
	result, err := s.RunPhase(context.Background(), PhaseF2)
	if err != nil {
		t.Fatalf("RunPhase F2 unexpected error: %v", err)
	}
	if result.Phase != PhaseF2 || result.Status != StatusSuccess {
		t.Errorf("expected F2 success, got %s=%s", result.Phase, result.Status)
	}
}

func TestRunPhase_PhaseNotFound(t *testing.T) {
	s := NewScheduler(&Config{PhaseTimeout: time.Minute}, makeSuccessRunners())

	result, err := s.RunPhase(context.Background(), "F5")
	if err == nil {
		t.Fatal("expected error for unknown phase")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Phase timeout tests
// ---------------------------------------------------------------------------

func TestPhaseTimeout(t *testing.T) {
	cfg := &Config{PhaseTimeout: 5 * time.Millisecond}
	runners := []PhaseRunner{
		&mockRunner{
			name:   PhaseF1,
			result: &Result{Phase: PhaseF1, Status: StatusSuccess, Summary: "too slow"},
			delay:  200 * time.Millisecond, // far exceeds timeout
		},
	}
	s := NewScheduler(cfg, runners)

	results, err := s.Run(context.Background())
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("expected deadline exceeded error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusFail {
		t.Errorf("expected StatusFail after timeout, got %s", results[0].Status)
	}
}

func TestPhaseTimeout_SubsequentPhasesSkipped(t *testing.T) {
	cfg := &Config{PhaseTimeout: 5 * time.Millisecond}
	runners := []PhaseRunner{
		&mockRunner{
			name:   PhaseF1,
			result: &Result{Phase: PhaseF1, Status: StatusSuccess, Summary: "ok"},
		},
		&mockRunner{
			name:   PhaseF2,
			result: &Result{Phase: PhaseF2, Status: StatusSuccess, Summary: "too slow"},
			delay:  200 * time.Millisecond, // will time out
		},
		&mockRunner{
			name:   PhaseF3,
			result: &Result{Phase: PhaseF3, Status: StatusSuccess, Summary: "should not run"},
		},
	}
	s := NewScheduler(cfg, runners)

	// Auto-approve F1 before the timeout on F2
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("y\n"))
	t.Cleanup(func() { stdinReader = old })

	results, err := s.Run(context.Background())
	if err == nil {
		t.Fatal("expected timeout error")
	}
	// F1 should succeed, F2 should time out, F3 should be skipped
	if len(results) != 2 {
		t.Fatalf("expected 2 results (F1+F2 timeout), got %d", len(results))
	}
	if results[0].Phase != PhaseF1 || results[0].Status != StatusSuccess {
		t.Errorf("expected F1 success, got %s=%s", results[0].Phase, results[0].Status)
	}
	if results[1].Phase != PhaseF2 || results[1].Status != StatusFail {
		t.Errorf("expected F2 fail, got %s=%s", results[1].Phase, results[1].Status)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// captureStdout runs fn while capturing stdout to avoid test noise.
// In this implementation we simply run fn; output goes to test log.
func captureStdout(t *testing.T, fn func()) {
	t.Helper()
	fn()
}
