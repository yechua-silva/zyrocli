package boomerang

import (
	"context"
	"testing"
)

func TestEvaluateCriteriaAllPass(t *testing.T) {
	dag := &TaskDAG{
		Tasks: []TaskSpec{
			{
				ID: 1, Name: "implement",
				AcceptanceCriteria: []AcceptanceCriteria{
					{ID: "AC-001", Status: CriteriaPending},
					{ID: "AC-002", Status: CriteriaPending},
				},
			},
		},
	}
	delegateResult := &DelegateResult{
		TaskResults: map[string]TaskResult{
			"implement": {TaskName: "implement", Success: true, Output: "changes applied"},
		},
	}

	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ok := o.evaluateCriteria(context.Background(), dag, delegateResult)
	if !ok {
		t.Error("expected all criteria to pass")
	}

	// Verify statuses
	for _, task := range dag.Tasks {
		for _, c := range task.AcceptanceCriteria {
			if c.Status != CriteriaVerified {
				t.Errorf("criteria %s: expected Verified, got %s", c.ID, c.Status)
			}
		}
	}
}

func TestEvaluateCriteriaFail(t *testing.T) {
	dag := &TaskDAG{
		Tasks: []TaskSpec{
			{
				ID: 1, Name: "implement",
				AcceptanceCriteria: []AcceptanceCriteria{
					{ID: "AC-001", Status: CriteriaPending},
				},
			},
		},
	}
	delegateResult := &DelegateResult{
		TaskResults: map[string]TaskResult{
			"implement": {TaskName: "implement", Success: false, Output: ""},
		},
	}

	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ok := o.evaluateCriteria(context.Background(), dag, delegateResult)
	if ok {
		t.Error("expected criteria to fail when delegate fails")
	}

	// Verify status
	for _, task := range dag.Tasks {
		for _, c := range task.AcceptanceCriteria {
			if c.Status != CriteriaFailed {
				t.Errorf("criteria %s: expected Failed, got %s", c.ID, c.Status)
			}
		}
	}
}

func TestEvaluateCriteriaEmpty(t *testing.T) {
	dag := &TaskDAG{
		Tasks: []TaskSpec{
			{ID: 1, Name: "implement", AcceptanceCriteria: []AcceptanceCriteria{}},
		},
	}
	delegateResult := &DelegateResult{
		TaskResults: map[string]TaskResult{
			"implement": {TaskName: "implement", Success: true, Output: "ok"},
		},
	}

	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ok := o.evaluateCriteria(context.Background(), dag, delegateResult)
	if !ok {
		t.Error("expected empty criteria to pass")
	}
}

func TestEvaluateCriteriaNoDAG(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ok := o.evaluateCriteria(context.Background(), nil, nil)
	if !ok {
		t.Error("expected nil DAG to pass")
	}
}

func TestEvaluateCriteriaMixed(t *testing.T) {
	dag := &TaskDAG{
		Tasks: []TaskSpec{
			{
				ID: 1, Name: "implement",
				AcceptanceCriteria: []AcceptanceCriteria{
					{ID: "AC-001", Status: CriteriaPending},
					{ID: "AC-002", Status: CriteriaPending},
				},
			},
		},
	}
	delegateResult := &DelegateResult{
		TaskResults: map[string]TaskResult{
			"implement": {TaskName: "implement", Success: true, Output: "output"},
		},
	}

	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ok := o.evaluateCriteria(context.Background(), dag, delegateResult)
	if !ok {
		t.Error("expected mix to pass when all succeed")
	}
}

func TestEvaluateCriteriaAlreadyVerified(t *testing.T) {
	dag := &TaskDAG{
		Tasks: []TaskSpec{
			{
				ID: 1, Name: "implement",
				AcceptanceCriteria: []AcceptanceCriteria{
					{ID: "AC-001", Status: CriteriaVerified}, // already verified
					{ID: "AC-002", Status: CriteriaPending},
				},
			},
		},
	}
	delegateResult := &DelegateResult{
		TaskResults: map[string]TaskResult{
			"implement": {TaskName: "implement", Success: true, Output: "output"},
		},
	}

	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ok := o.evaluateCriteria(context.Background(), dag, delegateResult)
	if !ok {
		t.Error("expected criteria to pass with pre-verified")
	}

	// AC-001 should remain verified (not re-evaluated)
	for _, task := range dag.Tasks {
		for _, c := range task.AcceptanceCriteria {
			if c.ID == "AC-001" && c.Status != CriteriaVerified {
				t.Errorf("AC-001 should remain Verified, got %s", c.Status)
			}
			if c.ID == "AC-002" && c.Status != CriteriaVerified {
				t.Errorf("AC-002 should be Verified, got %s", c.Status)
			}
		}
	}
}

func TestEvaluateCriteriaAlreadyFailed(t *testing.T) {
	dag := &TaskDAG{
		Tasks: []TaskSpec{
			{
				ID: 1, Name: "implement",
				AcceptanceCriteria: []AcceptanceCriteria{
					{ID: "AC-001", Status: CriteriaFailed}, // already failed
				},
			},
		},
	}
	delegateResult := &DelegateResult{
		TaskResults: map[string]TaskResult{
			"implement": {TaskName: "implement", Success: true, Output: "output"},
		},
	}

	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ok := o.evaluateCriteria(context.Background(), dag, delegateResult)
	if ok {
		t.Error("expected criteria to fail when already failed")
	}
}
