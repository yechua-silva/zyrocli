package boomerang

import (
	"context"
	"testing"

	"github.com/secko/zyrocli/internal/boundari"
	"github.com/secko/zyrocli/internal/memory"
)

// mockStore implementa EngramStore para tests
type mockStore struct{}

func (m *mockStore) SaveFact(ctx context.Context, fact *memory.Fact) (int64, error) {
	return 1, nil
}
func (m *mockStore) SaveFactsBatch(ctx context.Context, facts []*memory.Fact) ([]int64, error) {
	return []int64{1}, nil
}
func (m *mockStore) AddCausalEdge(ctx context.Context, edge *memory.CausalEdge) error {
	return nil
}
func (m *mockStore) RecallMemories(ctx context.Context, opts memory.RecallOpts) ([]*memory.MemoryResult, error) {
	return nil, nil
}
func (m *mockStore) GetCausalChain(ctx context.Context, factID int64, maxDepth int) ([]*memory.Fact, error) {
	return nil, nil
}
func (m *mockStore) GetFactByID(ctx context.Context, factID int64) (*memory.Fact, error) {
	return nil, nil
}
func (m *mockStore) DetectContradictions(ctx context.Context, projectID string, threshold float64) ([]memory.ContradictionPair, error) {
	return nil, nil
}
func (m *mockStore) ResolveContradiction(ctx context.Context, pair memory.ContradictionPair, strategy memory.ContradictionStrategy) error {
	return nil
}
func (m *mockStore) ReinforceSalience(ctx context.Context, factIDs []int64) error {
	return nil
}
func (m *mockStore) DecayAndRefresh(ctx context.Context, projectID string) error {
	return nil
}

func mockBoundariLoader(phase string) (*boundari.Policy, error) {
	return boundari.LoadDefaultPolicy(phase), nil
}

func TestNewBoomerangOrchestrator(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader)
	if o == nil {
		t.Fatal("expected non-nil orchestrator")
	}
	if o.maxIterations != 3 {
		t.Errorf("expected 3 max iterations, got %d", o.maxIterations)
	}
}

func TestMemoryStep(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader)
	ctx := context.Background()

	result, err := o.MemoryStep(ctx, "F0", "test task")
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Error("expected empty memory result for mock store")
	}
}

func TestThinkStep(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader)
	ctx := context.Background()

	dag, err := o.ThinkStep(ctx, "F0", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(dag.Tasks) != 3 {
		t.Errorf("F0 expected 3 tasks, got %d", len(dag.Tasks))
	}

	dag, err = o.ThinkStep(ctx, "F3", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(dag.Tasks) != 2 {
		t.Errorf("F3 expected 2 tasks, got %d", len(dag.Tasks))
	}
}

func TestThinkStepInvalidPhase(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader)
	ctx := context.Background()

	_, err := o.ThinkStep(ctx, "F9", "")
	if err == nil {
		t.Error("expected error for invalid phase")
	}
}

func TestGitStep(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader)
	ctx := context.Background()

	status, err := o.GitStep(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// En CI o test, puede ser clean o dirty
	if status != "clean" && status != "dirty" && status != "error" {
		t.Errorf("unexpected git status: %s", status)
	}
}

func TestDelegateStep(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader)
	ctx := context.Background()

	dag := &TaskDAG{
		ParallelGroups: [][]int{{0}},
		Tasks: []TaskSpec{
			{ID: 1, Name: "test-task", Description: "test", Agent: "nonexistent-agent"},
		},
	}

	result, err := o.DelegateStep(ctx, dag, "F0")
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestQualityStep(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader)
	ctx := context.Background()

	dag := &TaskDAG{Tasks: []TaskSpec{{ID: 1, Name: "test"}}}
	delegateResult := &DelegateResult{
		TaskResults: map[string]TaskResult{
			"test": {TaskName: "test", Success: true},
		},
	}

	ok, err := o.QualityStep(ctx, "F0", dag, delegateResult)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected quality to pass for F0")
	}
}

func TestSaveStep(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader)
	ctx := context.Background()

	delegateResult := &DelegateResult{
		NodesCreated: 2,
		TaskResults: map[string]TaskResult{
			"task1": {TaskName: "task1", Success: true, Output: "resultado 1", Nodes: 1},
			"task2": {TaskName: "task2", Success: true, Output: "resultado 2", Nodes: 1},
		},
	}

	result, err := o.SaveStep(ctx, "F0", delegateResult, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.FactsSaved != 2 {
		t.Errorf("expected 2 facts saved, got %d", result.FactsSaved)
	}
}

func TestRunPhase(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader)
	ctx := context.Background()

	config := PhaseConfig{
		Phase:    "F0",
		TaskDesc: "test run",
	}

	result, err := o.RunPhase(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Phase != "F0" {
		t.Errorf("expected F0, got %s", result.Phase)
	}
}
