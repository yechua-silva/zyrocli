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

// ---------------------------------------------------------------------------
// GuidedApproval detail mode tests
// ---------------------------------------------------------------------------

func TestGuidedApproval_DetailMode(t *testing.T) {
	// Input: "d" to see detail, then "s" to approve
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("d\ns\n"))
	t.Cleanup(func() { stdinReader = old })

	g := NewGuidedApproval(PhaseF1, "test summary").
		WithDetail("Full agent output here\nWith multiple lines")

	approved, err := g.PromptApproval()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true after detail then yes")
	}
}

func TestGuidedApproval_DetailNoOutput(t *testing.T) {
	// Input: "d" to see detail (but no FullOutput set), then "s"
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("d\ns\n"))
	t.Cleanup(func() { stdinReader = old })

	g := NewGuidedApproval(PhaseF1, "test summary")
	// No WithDetail set

	approved, err := g.PromptApproval()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true after detail then yes")
	}
}

func TestGuidedApproval_RejectAfterDetail(t *testing.T) {
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("d\nn\n"))
	t.Cleanup(func() { stdinReader = old })

	g := NewGuidedApproval(PhaseF1, "test summary").
		WithDetail("some detail")

	approved, err := g.PromptApproval()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false after detail then no")
	}
}

func TestGuidedApproval_WithRecommendAndRisk(t *testing.T) {
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("s\n"))
	t.Cleanup(func() { stdinReader = old })

	g := NewGuidedApproval(PhaseF1, "test summary").
		WithRecommend("Continue to next phase").
		WithRisk("High complexity in F3")

	approved, err := g.PromptApproval()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true")
	}
}

func TestGuidedApproval_MultipleInvalidThenApprove(t *testing.T) {
	old := stdinReader
	// Multiple invalid inputs, then approve
	stdinReader = bufio.NewReader(strings.NewReader("x\n?\nfoo\ny\n"))
	t.Cleanup(func() { stdinReader = old })

	g := NewGuidedApproval(PhaseF1, "test")
	approved, err := g.PromptApproval()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true after retries")
	}
}

// ---------------------------------------------------------------------------
// HarnessValidator tests
// ---------------------------------------------------------------------------

func TestHarnessValidator_ValidateTransitionRequiresApproval(t *testing.T) {
	v := NewHarnessValidator(AllPhases)
	v.SetCurrent(PhaseF1)

	err := v.ValidateTransition(PhaseF1, PhaseF2, false)
	if err == nil {
		t.Fatal("expected error for unapproved transition")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("expected 'blocked' in error, got %v", err)
	}
}

func TestHarnessValidator_ValidTransition(t *testing.T) {
	v := NewHarnessValidator(AllPhases)
	v.SetCurrent(PhaseF1)

	err := v.ValidateTransition(PhaseF1, PhaseF2, true)
	if err != nil {
		t.Errorf("expected nil error for valid approved transition, got %v", err)
	}
}

func TestHarnessValidator_CurrentPhase(t *testing.T) {
	v := NewHarnessValidator(AllPhases)
	v.SetCurrent(PhaseF2)

	if v.CurrentPhase() != PhaseF2 {
		t.Errorf("expected F2, got %s", v.CurrentPhase())
	}
}

func TestHarnessValidator_NextPhase(t *testing.T) {
	v := NewHarnessValidator(AllPhases)

	v.SetCurrent(PhaseF1)
	if v.NextPhase() != PhaseF2 {
		t.Errorf("expected F2 after F1, got %s", v.NextPhase())
	}

	v.SetCurrent(PhaseF2)
	if v.NextPhase() != PhaseF3 {
		t.Errorf("expected F3 after F2, got %s", v.NextPhase())
	}

	v.SetCurrent(PhaseF3)
	if v.NextPhase() != PhaseF4 {
		t.Errorf("expected F4 after F3, got %s", v.NextPhase())
	}

	v.SetCurrent(PhaseF4)
	if v.NextPhase() != "" {
		t.Errorf("expected empty after F4, got %s", v.NextPhase())
	}
}

func TestHarnessValidator_InvalidPhase(t *testing.T) {
	v := NewHarnessValidator(AllPhases)
	v.SetCurrent("F5")

	err := v.ValidateTransition("F5", "F6", true)
	if err == nil {
		t.Fatal("expected error for unknown phase")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("expected 'unknown' in error, got %v", err)
	}

	if v.NextPhase() != "" {
		t.Errorf("expected empty next for unknown phase")
	}
}

// ---------------------------------------------------------------------------
// MacroPhaseRunner tests
// ---------------------------------------------------------------------------

func TestMacroPhaseRunner_Success(t *testing.T) {
	runner := NewMacroPhaseRunner(PhaseF1, func(ctx context.Context, cfg *Config) (*Result, error) {
		return &Result{Phase: PhaseF1, Status: StatusSuccess, Summary: "done"}, nil
	})

	result, err := runner.Run(context.Background(), &Config{PhaseTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != StatusSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}
}

func TestMacroPhaseRunner_WithValidatorPass(t *testing.T) {
	runner := NewMacroPhaseRunner(PhaseF1, func(ctx context.Context, cfg *Config) (*Result, error) {
		return &Result{Phase: PhaseF1, Status: StatusSuccess, Summary: "ok"}, nil
	}).WithValidator(DefaultValidator())

	result, err := runner.Run(context.Background(), &Config{PhaseTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != StatusSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}
}

func TestMacroPhaseRunner_WithValidatorFail(t *testing.T) {
	runner := NewMacroPhaseRunner(PhaseF1, func(ctx context.Context, cfg *Config) (*Result, error) {
		return &Result{Phase: PhaseF1, Status: StatusFail, Summary: "failed"}, nil
	}).WithValidator(DefaultValidator())

	result, err := runner.Run(context.Background(), &Config{PhaseTimeout: 5 * time.Second})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if result.Status != StatusFail {
		t.Errorf("expected fail, got %s", result.Status)
	}
}

func TestMacroPhaseRunner_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	runner := NewMacroPhaseRunner(PhaseF1, func(ctx context.Context, cfg *Config) (*Result, error) {
		return &Result{Phase: PhaseF1, Status: StatusSuccess, Summary: "should not run"}, nil
	})

	result, err := runner.Run(ctx, &Config{PhaseTimeout: 5 * time.Second})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if result.Status != StatusFail {
		t.Errorf("expected fail after cancel, got %s", result.Status)
	}
}

func TestMacroPhaseRunner_Name(t *testing.T) {
	runner := NewMacroPhaseRunner(PhaseF3, nil)
	if runner.Name() != PhaseF3 {
		t.Errorf("expected F3, got %s", runner.Name())
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------


