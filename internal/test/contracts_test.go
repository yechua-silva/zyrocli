package test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ContractExecutor — Given/When/Then pipeline
// ---------------------------------------------------------------------------

func TestContractExecutor_Success(t *testing.T) {
	executor := NewContractExecutor()
	contract := Contract{
		Name: "successful-contract",
		Given: func(ctx context.Context) (interface{}, error) {
			return "initial-state", nil
		},
		When: func(ctx context.Context, state interface{}) (interface{}, error) {
			return fmt.Sprintf("result-from-%s", state), nil
		},
		Then: func(ctx context.Context, state, result interface{}) error {
			expected := "result-from-initial-state"
			if result != expected {
				return fmt.Errorf("expected %q, got %v", expected, result)
			}
			return nil
		},
	}

	result := executor.Execute(context.Background(), contract)
	if !result.Passed {
		t.Errorf("expected passed=true, got passed=false with error: %s", result.Error)
	}
	if result.Name != "successful-contract" {
		t.Errorf("expected name 'successful-contract', got %q", result.Name)
	}
}

func TestContractExecutor_GivenFailure(t *testing.T) {
	executor := NewContractExecutor()
	contract := Contract{
		Name: "given-fails",
		Given: func(ctx context.Context) (interface{}, error) {
			return nil, errors.New("setup failed")
		},
		When: func(ctx context.Context, state interface{}) (interface{}, error) {
			t.Error("WHEN should not be called after GIVEN failure")
			return nil, nil
		},
		Then: func(ctx context.Context, state, result interface{}) error {
			t.Error("THEN should not be called after GIVEN failure")
			return nil
		},
	}

	result := executor.Execute(context.Background(), contract)
	if result.Passed {
		t.Error("expected passed=false after GIVEN failure")
	}
	if !strings.Contains(result.Error, "GIVEN failed") {
		t.Errorf("expected 'GIVEN failed' in error, got %q", result.Error)
	}
}

func TestContractExecutor_WhenFailure(t *testing.T) {
	executor := NewContractExecutor()
	contract := Contract{
		Name: "when-fails",
		Given: func(ctx context.Context) (interface{}, error) {
			return "ok", nil
		},
		When: func(ctx context.Context, state interface{}) (interface{}, error) {
			return nil, errors.New("action failed")
		},
		Then: func(ctx context.Context, state, result interface{}) error {
			t.Error("THEN should not be called after WHEN failure")
			return nil
		},
	}

	result := executor.Execute(context.Background(), contract)
	if result.Passed {
		t.Error("expected passed=false after WHEN failure")
	}
	if !strings.Contains(result.Error, "WHEN failed") {
		t.Errorf("expected 'WHEN failed' in error, got %q", result.Error)
	}
}

func TestContractExecutor_ThenFailure(t *testing.T) {
	executor := NewContractExecutor()
	contract := Contract{
		Name: "then-fails",
		Given: func(ctx context.Context) (interface{}, error) {
			return "state", nil
		},
		When: func(ctx context.Context, state interface{}) (interface{}, error) {
			return "result", nil
		},
		Then: func(ctx context.Context, state, result interface{}) error {
			return errors.New("verification failed")
		},
	}

	result := executor.Execute(context.Background(), contract)
	if result.Passed {
		t.Error("expected passed=false after THEN failure")
	}
	if !strings.Contains(result.Error, "THEN failed") {
		t.Errorf("expected 'THEN failed' in error, got %q", result.Error)
	}
}

func TestContractExecutor_NilThen(t *testing.T) {
	executor := NewContractExecutor()
	contract := Contract{
		Name: "no-then",
		Given: func(ctx context.Context) (interface{}, error) {
			return "state", nil
		},
		When: func(ctx context.Context, state interface{}) (interface{}, error) {
			return "result", nil
		},
		Then: nil, // no verification
	}

	result := executor.Execute(context.Background(), contract)
	if !result.Passed {
		t.Errorf("expected passed=true for nil Then, got error: %s", result.Error)
	}
}

func TestContractExecutor_PhaseOrder(t *testing.T) {
	executor := NewContractExecutor()
	var order []string

	contract := Contract{
		Name: "phase-order",
		Given: func(ctx context.Context) (interface{}, error) {
			order = append(order, "given")
			return "state", nil
		},
		When: func(ctx context.Context, state interface{}) (interface{}, error) {
			order = append(order, "when")
			return "result", nil
		},
		Then: func(ctx context.Context, state, result interface{}) error {
			order = append(order, "then")
			return nil
		},
	}

	executor.Execute(context.Background(), contract)

	expected := []string{"given", "when", "then"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d phases, got %d: %v", len(expected), len(order), order)
	}
	for i, p := range order {
		if p != expected[i] {
			t.Errorf("phase %d: expected %s, got %s", i, expected[i], p)
		}
	}
}

func TestContractExecutor_StatePassThrough(t *testing.T) {
	executor := NewContractExecutor()
	type AppState struct {
		Value string
	}

	contract := Contract{
		Name: "state-flow",
		Given: func(ctx context.Context) (interface{}, error) {
			return AppState{Value: "hello"}, nil
		},
		When: func(ctx context.Context, state interface{}) (interface{}, error) {
			s := state.(AppState)
			return AppState{Value: s.Value + "-world"}, nil
		},
		Then: func(ctx context.Context, state, result interface{}) error {
			s := state.(AppState)
			r := result.(AppState)
			if s.Value != "hello" {
				return fmt.Errorf("expected state.Value='hello', got %q", s.Value)
			}
			if r.Value != "hello-world" {
				return fmt.Errorf("expected result.Value='hello-world', got %q", r.Value)
			}
			return nil
		},
	}

	result := executor.Execute(context.Background(), contract)
	if !result.Passed {
		t.Errorf("expected state flow to pass, got error: %s", result.Error)
	}
}

// ---------------------------------------------------------------------------
// ExecuteBatch
// ---------------------------------------------------------------------------

func TestContractExecutor_ExecuteBatch(t *testing.T) {
	executor := NewContractExecutor()
	contracts := []Contract{
		{
			Name: "pass",
			Given: func(ctx context.Context) (interface{}, error) { return nil, nil },
			When:  func(ctx context.Context, state interface{}) (interface{}, error) { return "ok", nil },
			Then:  func(ctx context.Context, state, result interface{}) error { return nil },
		},
		{
			Name: "fail",
			Given: func(ctx context.Context) (interface{}, error) { return nil, nil },
			When:  func(ctx context.Context, state interface{}) (interface{}, error) { return nil, errors.New("fail") },
		},
		{
			Name: "pass-2",
			Given: func(ctx context.Context) (interface{}, error) { return nil, nil },
			When:  func(ctx context.Context, state interface{}) (interface{}, error) { return "ok-2", nil },
			Then:  func(ctx context.Context, state, result interface{}) error { return nil },
		},
	}

	results := executor.ExecuteBatch(context.Background(), contracts)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if !results[0].Passed {
		t.Errorf("expected contract 0 to pass, got error: %s", results[0].Error)
	}
	if results[1].Passed {
		t.Error("expected contract 1 to fail")
	}
	if !results[2].Passed {
		t.Errorf("expected contract 2 to pass, got error: %s", results[2].Error)
	}
}

func TestContractExecutor_ExecuteBatchNil(t *testing.T) {
	executor := NewContractExecutor()
	results := executor.ExecuteBatch(context.Background(), nil)
	if results != nil {
		t.Errorf("expected nil results for nil contracts, got %d", len(results))
	}
}

func TestContractExecutor_ExecuteBatchEmpty(t *testing.T) {
	executor := NewContractExecutor()
	results := executor.ExecuteBatch(context.Background(), []Contract{})
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty contracts, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// GraphifyDiff
// ---------------------------------------------------------------------------

func TestNewGraphifyDiff_NoChanges(t *testing.T) {
	diff := NewGraphifyDiff(10, 5, 10, 5)
	if diff.TotalDiffs != 0 {
		t.Errorf("expected 0 diffs, got %d", diff.TotalDiffs)
	}
	if diff.Significant {
		t.Error("expected not significant for no changes")
	}
	if diff.Summary() != "No structural changes detected" {
		t.Errorf("unexpected summary: %q", diff.Summary())
	}
}

func TestNewGraphifyDiff_NodesAdded(t *testing.T) {
	diff := NewGraphifyDiff(10, 5, 15, 5)
	if diff.NodesAdded != 5 {
		t.Errorf("expected 5 nodes added, got %d", diff.NodesAdded)
	}
	if diff.NodesRemoved != 0 {
		t.Errorf("expected 0 nodes removed, got %d", diff.NodesRemoved)
	}
	if diff.TotalDiffs != 5 {
		t.Errorf("expected TotalDiffs=5, got %d", diff.TotalDiffs)
	}
}

func TestNewGraphifyDiff_NodesRemoved(t *testing.T) {
	diff := NewGraphifyDiff(10, 5, 7, 5)
	if diff.NodesRemoved != 3 {
		t.Errorf("expected 3 nodes removed, got %d", diff.NodesRemoved)
	}
	if diff.EdgesAdded != 0 {
		t.Errorf("expected 0 edges added, got %d", diff.EdgesAdded)
	}
}

func TestNewGraphifyDiff_EdgesAdded(t *testing.T) {
	diff := NewGraphifyDiff(10, 5, 10, 8)
	if diff.EdgesAdded != 3 {
		t.Errorf("expected 3 edges added, got %d", diff.EdgesAdded)
	}
	if diff.NodesAdded != 0 {
		t.Errorf("expected 0 nodes added, got %d", diff.NodesAdded)
	}
}

func TestNewGraphifyDiff_EdgesRemoved(t *testing.T) {
	diff := NewGraphifyDiff(10, 5, 10, 2)
	if diff.EdgesRemoved != 3 {
		t.Errorf("expected 3 edges removed, got %d", diff.EdgesRemoved)
	}
}

func TestNewGraphifyDiff_Mixed(t *testing.T) {
	diff := NewGraphifyDiff(20, 15, 25, 10)
	if diff.NodesAdded != 5 {
		t.Errorf("expected 5 nodes added, got %d", diff.NodesAdded)
	}
	if diff.EdgesRemoved != 5 {
		t.Errorf("expected 5 edges removed, got %d", diff.EdgesRemoved)
	}
	if diff.TotalDiffs != 10 {
		t.Errorf("expected TotalDiffs=10, got %d", diff.TotalDiffs)
	}
	if !diff.Significant {
		t.Error("expected significant for 10 changes")
	}
}

func TestGraphifyDiff_IsSignificant(t *testing.T) {
	diff := NewGraphifyDiff(10, 5, 10, 5)
	if diff.IsSignificant(0) {
		t.Error("expected not significant for 0 changes with threshold 0")
	}
	if diff.IsSignificant(10) {
		t.Error("expected not significant for 0 changes with threshold 10")
	}

	diff2 := NewGraphifyDiff(10, 5, 15, 8)
	if !diff2.IsSignificant(5) {
		t.Error("expected significant for 8 changes with threshold 5")
	}
	if diff2.IsSignificant(10) {
		t.Error("expected not significant for 8 changes with threshold 10")
	}
}

func TestGraphifyDiff_Summary(t *testing.T) {
	diff := NewGraphifyDiff(10, 5, 15, 5)
	summary := diff.Summary()
	if !strings.Contains(summary, "+5/-0 nodes") {
		t.Errorf("expected '+5/-0 nodes' in summary, got %q", summary)
	}

	diff2 := NewGraphifyDiff(10, 5, 5, 3)
	summary2 := diff2.Summary()
	if !strings.Contains(summary2, "-5 nodes") {
		t.Errorf("expected '-5 nodes' in summary, got %q", summary2)
	}
}

func TestGraphifyDiff_String(t *testing.T) {
	diff := NewGraphifyDiff(10, 5, 15, 8)
	s := diff.String()
	if !strings.Contains(s, "GraphifyDiff") {
		t.Errorf("expected 'GraphifyDiff' in String(), got %q", s)
	}
	if !strings.Contains(s, "node_added") {
		t.Errorf("expected 'node_added' in String(), got %q", s)
	}
	if !strings.Contains(s, "edge_added") {
		t.Errorf("expected 'edge_added' in String(), got %q", s)
	}

	noDiff := NewGraphifyDiff(5, 3, 5, 3)
	s2 := noDiff.String()
	if !strings.Contains(s2, "No structural changes") {
		t.Errorf("expected 'No structural changes' in String(), got %q", s2)
	}
}

// ---------------------------------------------------------------------------
// Report
// ---------------------------------------------------------------------------

func TestNewReport(t *testing.T) {
	results := []ContractResult{
		{Name: "a", Passed: true},
		{Name: "b", Passed: false, Error: "broken"},
		{Name: "c", Passed: true},
	}

	report := NewReport(results)
	if report.Passed != 2 {
		t.Errorf("expected 2 passed, got %d", report.Passed)
	}
	if report.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", report.Failed)
	}
	if len(report.Diffs) != 1 {
		t.Errorf("expected 1 diff entry, got %d", len(report.Diffs))
	}
}

func TestNewReport_AllPass(t *testing.T) {
	results := []ContractResult{
		{Name: "a", Passed: true},
		{Name: "b", Passed: true},
	}
	report := NewReport(results)
	if report.Passed != 2 {
		t.Errorf("expected 2 passed, got %d", report.Passed)
	}
	if report.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", report.Failed)
	}
	if len(report.Diffs) != 0 {
		t.Errorf("expected 0 diffs, got %d", len(report.Diffs))
	}
}

func TestNewReport_AllFail(t *testing.T) {
	results := []ContractResult{
		{Name: "a", Passed: false, Error: "err1"},
		{Name: "b", Passed: false, Error: "err2"},
	}
	report := NewReport(results)
	if report.Passed != 0 {
		t.Errorf("expected 0 passed, got %d", report.Passed)
	}
	if report.Failed != 2 {
		t.Errorf("expected 2 failed, got %d", report.Failed)
	}
	if len(report.Diffs) != 2 {
		t.Errorf("expected 2 diffs, got %d", len(report.Diffs))
	}
}

func TestNewReport_Empty(t *testing.T) {
	report := NewReport([]ContractResult{})
	if report.Passed != 0 {
		t.Errorf("expected 0 passed, got %d", report.Passed)
	}
	if report.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", report.Failed)
	}
}

func TestReport_WithGraphDiff(t *testing.T) {
	results := []ContractResult{{Name: "a", Passed: true}}
	diff := NewGraphifyDiff(10, 5, 15, 8)

	report := NewReport(results).WithGraphDiff(diff)
	if report.GraphDiff == nil {
		t.Fatal("expected GraphDiff to be set")
	}
	if !report.GraphDiff.Significant {
		t.Error("expected diff to be significant")
	}
}

func TestReport_Summary(t *testing.T) {
	results := []ContractResult{
		{Name: "a", Passed: true},
		{Name: "b", Passed: false, Error: "err"},
	}
	report := NewReport(results)
	summary := report.Summary()
	if !strings.Contains(summary, "1 passed") {
		t.Errorf("expected '1 passed' in summary, got %q", summary)
	}
	if !strings.Contains(summary, "1 failed") {
		t.Errorf("expected '1 failed' in summary, got %q", summary)
	}

	diff := NewGraphifyDiff(10, 5, 15, 5)
	report2 := NewReport(results).WithGraphDiff(diff)
	summary2 := report2.Summary()
	if !strings.Contains(summary2, "GraphifyDiff") && !strings.Contains(summary2, "structural") {
		t.Errorf("expected graph info in summary, got %q", summary2)
	}
}
