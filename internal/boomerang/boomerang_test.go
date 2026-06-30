package boomerang

import (
	"context"
	"testing"
	"time"

	"github.com/yechua-silva/zyrocli/internal/boundari"
	"github.com/yechua-silva/zyrocli/internal/memory"
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
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	if o == nil {
		t.Fatal("expected non-nil orchestrator")
	}
	if o.maxIterations != 3 {
		t.Errorf("expected 3 max iterations, got %d", o.maxIterations)
	}
}

func TestMemoryStep(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
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
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ctx := context.Background()

	dag, err := o.ThinkStep(ctx, "F0", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(dag.Tasks) != 3 {
		t.Errorf("F0 expected 3 tasks, got %d", len(dag.Tasks))
	}

	dag, err = o.ThinkStep(ctx, "F3", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(dag.Tasks) != 2 {
		t.Errorf("F3 expected 2 tasks, got %d", len(dag.Tasks))
	}
}

func TestThinkStepInvalidPhase(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ctx := context.Background()

	_, err := o.ThinkStep(ctx, "F9", "", nil)
	if err == nil {
		t.Error("expected error for invalid phase")
	}
}

func TestGitStep(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
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
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ctx := context.Background()

	dag := &TaskDAG{
		ParallelGroups: [][]int{{0}},
		Tasks: []TaskSpec{
			{ID: 1, Name: "test-task", Description: "test", Agent: "nonexistent-agent"},
		},
	}

	result, err := o.DelegateStep(ctx, dag, "F0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestQualityStep(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
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
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ctx := context.Background()

	delegateResult := &DelegateResult{
		NodesCreated: 2,
		TaskResults: map[string]TaskResult{
			"task1": {TaskName: "task1", Success: true, Output: "resultado 1", Nodes: 1},
			"task2": {TaskName: "task2", Success: true, Output: "resultado 2", Nodes: 1},
		},
	}

	result, err := o.SaveStep(ctx, "F0", delegateResult, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.FactsSaved != 2 {
		t.Errorf("expected 2 facts saved, got %d", result.FactsSaved)
	}
}

func TestRunPhase(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
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

// --- T5: Tests unitarios de skip.go ---

func TestStepString(t *testing.T) {
	tests := []struct {
		step Step
		want string
	}{
		{StepMemory, "Memory"},
		{StepThink, "Think"},
		{StepDelegate, "Delegate"},
		{StepGit, "Git"},
		{StepQuality, "Quality"},
		{StepSave, "Save"},
		{Step(-1), "Unknown"},
		{Step(99), "Unknown"},
	}
	for _, tt := range tests {
		got := tt.step.String()
		if got != tt.want {
			t.Errorf("Step(%d).String() = %q, want %q", tt.step, got, tt.want)
		}
	}
}

func TestStepStatusString(t *testing.T) {
	tests := []struct {
		status StepStatus
		want   string
	}{
		{StepPending, "pending"},
		{StepRunning, "running"},
		{StepDone, "done"},
		{StepSkipped, "skipped"},
		{StepFailed, "failed"},
		{StepStatus(-1), "unknown"},
		{StepStatus(99), "unknown"},
	}
	for _, tt := range tests {
		got := tt.status.String()
		if got != tt.want {
			t.Errorf("StepStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestDefaultPhaseMatrix(t *testing.T) {
	m := DefaultPhaseMatrix()

	// Verificar que existen las 5 fases
	expectedPhases := []string{"F0", "F1", "F2", "F3", "F4"}
	for _, phase := range expectedPhases {
		if _, ok := m[phase]; !ok {
			t.Errorf("DefaultPhaseMatrix: missing phase %s", phase)
		}
	}

	// F0: debe tener exactamente 4 steps
	if len(m["F0"]) != 4 {
		t.Errorf("F0: expected 4 steps, got %d", len(m["F0"]))
	}

	// F3: debe tener 6 steps
	if len(m["F3"]) != 6 {
		t.Errorf("F3: expected 6 steps, got %d", len(m["F3"]))
	}

	// F4: debe tener 4 steps (Memory, Delegate, Git, Save)
	if len(m["F4"]) != 4 {
		t.Errorf("F4: expected 4 steps, got %d", len(m["F4"]))
	}
}

func TestDefaultPhaseMatrixValidates(t *testing.T) {
	m := DefaultPhaseMatrix()
	if err := ValidateMatrix(m); err != nil {
		t.Errorf("DefaultPhaseMatrix should be valid, got: %v", err)
	}
}

func TestShouldRun(t *testing.T) {
	m := DefaultPhaseMatrix()

	tests := []struct {
		phase string
		step  Step
		want  bool
	}{
		// F0: Git y Quality no, el resto sí
		{"F0", StepMemory, true},
		{"F0", StepThink, true},
		{"F0", StepDelegate, true},
		{"F0", StepGit, false},
		{"F0", StepQuality, false},
		{"F0", StepSave, true},
		// F1: igual que F0
		{"F1", StepGit, false},
		{"F1", StepQuality, false},
		{"F1", StepMemory, true},
		// F2: igual que F0
		{"F2", StepGit, false},
		{"F2", StepQuality, false},
		// F3: todos true
		{"F3", StepMemory, true},
		{"F3", StepThink, true},
		{"F3", StepDelegate, true},
		{"F3", StepGit, true},
		{"F3", StepQuality, true},
		{"F3", StepSave, true},
		// F4: Think y Quality no
		{"F4", StepMemory, true},
		{"F4", StepThink, false},
		{"F4", StepDelegate, true},
		{"F4", StepGit, true},
		{"F4", StepQuality, false},
		{"F4", StepSave, true},
	}

	for _, tt := range tests {
		got := m.ShouldRun(tt.phase, tt.step)
		if got != tt.want {
			t.Errorf("ShouldRun(%q, %v) = %v, want %v", tt.phase, tt.step, got, tt.want)
		}
	}
}

func TestShouldRunUnknownPhase(t *testing.T) {
	m := DefaultPhaseMatrix()
	// Fase desconocida: todos los steps deben ejecutarse (default seguro)
	for _, step := range AllSteps() {
		if !m.ShouldRun("F99", step) {
			t.Errorf("ShouldRun(F99, %v) = false, want true (unknown phase should run all)", step)
		}
	}
}

func TestActiveSteps(t *testing.T) {
	m := DefaultPhaseMatrix()

	tests := []struct {
		phase string
		want  int
	}{
		{"F0", 4},
		{"F1", 4},
		{"F2", 4},
		{"F3", 6},
		{"F4", 4},
	}

	for _, tt := range tests {
		steps := m.ActiveSteps(tt.phase)
		if len(steps) != tt.want {
			t.Errorf("ActiveSteps(%q) = %d steps, want %d", tt.phase, len(steps), tt.want)
		}
	}
}

func TestActiveStepsUnknownPhase(t *testing.T) {
	m := DefaultPhaseMatrix()
	steps := m.ActiveSteps("F99")
	if len(steps) != 6 {
		t.Errorf("ActiveSteps(F99) = %d steps, want 6", len(steps))
	}
}

func TestActiveStepsImmutable(t *testing.T) {
	m := DefaultPhaseMatrix()
	original := m.ActiveSteps("F0")

	// Mutar la copia
	original[0] = StepGit

	// Verificar que la matriz original no se afectó
	current := m.ActiveSteps("F0")
	if current[0] == StepGit {
		t.Error("ActiveSteps mutó la matriz original: current[0] == StepGit")
	}
}

func TestValidateMatrix(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		m := DefaultPhaseMatrix()
		if err := ValidateMatrix(m); err != nil {
			t.Errorf("expected valid matrix, got: %v", err)
		}
	})

	t.Run("missing phase", func(t *testing.T) {
		m := DefaultPhaseMatrix()
		delete(m, "F0")
		if err := ValidateMatrix(m); err == nil {
			t.Error("expected error for missing phase F0")
		}
	})

	t.Run("empty phase", func(t *testing.T) {
		m := DefaultPhaseMatrix()
		m["F0"] = []Step{}
		if err := ValidateMatrix(m); err == nil {
			t.Error("expected error for empty phase F0")
		}
	})

	t.Run("F4 missing Save", func(t *testing.T) {
		m := DefaultPhaseMatrix()
		m["F4"] = []Step{StepMemory, StepDelegate, StepGit}
		if err := ValidateMatrix(m); err == nil {
			t.Error("expected error for F4 missing Save")
		}
	})

	t.Run("F3 missing Quality", func(t *testing.T) {
		m := DefaultPhaseMatrix()
		m["F3"] = []Step{StepMemory, StepThink, StepDelegate, StepGit, StepSave}
		if err := ValidateMatrix(m); err == nil {
			t.Error("expected error for F3 missing Quality")
		}
	})

	t.Run("F3 missing Git", func(t *testing.T) {
		m := DefaultPhaseMatrix()
		m["F3"] = []Step{StepMemory, StepThink, StepDelegate, StepQuality, StepSave}
		if err := ValidateMatrix(m); err == nil {
			t.Error("expected error for F3 missing Git")
		}
	})

	t.Run("duplicate step", func(t *testing.T) {
		m := DefaultPhaseMatrix()
		m["F0"] = []Step{StepMemory, StepMemory, StepThink}
		if err := ValidateMatrix(m); err == nil {
			t.Error("expected error for duplicate step")
		}
	})

	t.Run("invalid step value", func(t *testing.T) {
		m := DefaultPhaseMatrix()
		m["F0"] = []Step{Step(99)}
		if err := ValidateMatrix(m); err == nil {
			t.Error("expected error for invalid step value")
		}
	})
}

func TestAllSteps(t *testing.T) {
	steps := AllSteps()
	if len(steps) != 6 {
		t.Errorf("AllSteps() = %d steps, want 6", len(steps))
	}
	if steps[0] != StepMemory {
		t.Errorf("AllSteps()[0] = %v, want StepMemory", steps[0])
	}
	if steps[5] != StepSave {
		t.Errorf("AllSteps()[5] = %v, want StepSave", steps[5])
	}
}

// --- T6: Tests de integración RunPhase con skip matrix ---

func TestRunPhaseSkipMatrixF0(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ctx := context.Background()

	config := PhaseConfig{
		Phase:    "F0",
		TaskDesc: "test F0 skip matrix",
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

	// F0: Git se salta → GitStatus debe ser ""
	if result.GitStatus != "" {
		t.Errorf("F0: expected empty GitStatus (skipped), got %q", result.GitStatus)
	}
	// F0: Quality se salta → QualityOK debe ser false
	if result.QualityOK {
		t.Error("F0: expected QualityOK=false (skipped)")
	}
	// F0: Sin Quality ejecutado → Success debe ser true
	if !result.Success {
		t.Error("F0: expected Success=true (skipped Quality)")
	}
}

func TestRunPhaseSkipMatrixF3(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ctx := context.Background()

	config := PhaseConfig{
		Phase:    "F3",
		TaskDesc: "test F3 skip matrix",
	}

	result, err := o.RunPhase(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Phase != "F3" {
		t.Errorf("expected F3, got %s", result.Phase)
	}

	// F3: Git se ejecuta → GitStatus debe tener algún valor
	if result.GitStatus == "" {
		t.Error("F3: expected non-empty GitStatus (executed)")
	}
}

func TestRunPhaseSkipMatrixF4(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ctx := context.Background()

	config := PhaseConfig{
		Phase:    "F4",
		TaskDesc: "test F4 skip matrix",
	}

	result, err := o.RunPhase(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Phase != "F4" {
		t.Errorf("expected F4, got %s", result.Phase)
	}

	// F4: Think se salta → TasksPlanned debe ser 0
	if result.TasksPlanned != 0 {
		t.Errorf("F4: expected TasksPlanned=0 (Think skipped), got %d", result.TasksPlanned)
	}
	// F4: Quality se salta → QualityOK debe ser false
	if result.QualityOK {
		t.Error("F4: expected QualityOK=false (skipped)")
	}
	// F4: Sin Quality ejecutado → Success debe ser true
	if !result.Success {
		t.Error("F4: expected Success=true (skipped Quality)")
	}
}

// --- T7: Tests de backward compatibility ---

func TestRunPhaseLegacySignature(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ctx := context.Background()

	// La firma debe ser: RunPhase(ctx context.Context, config PhaseConfig) (*PhaseResult, error)
	config := PhaseConfig{
		Phase:    "F0",
		TaskDesc: "legacy signature test",
	}

	result, err := o.RunPhase(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("RunPhase must return non-nil PhaseResult")
	}
}

func TestRunPhaseLegacyF0Result(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ctx := context.Background()

	config := PhaseConfig{
		Phase:    "F0",
		TaskDesc: "legacy F0 test",
	}

	result, err := o.RunPhase(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "F0" {
		t.Errorf("expected phase F0, got %s", result.Phase)
	}
	if result.Error != "" {
		t.Errorf("expected no error, got %q", result.Error)
	}
}

// --- Fase 2: PhaseConfigV2 tests ---

func TestPhaseConfigToV2(t *testing.T) {
	legacy := PhaseConfig{
		Phase:    "F3",
		TaskDesc: "test",
		Timeout:  time.Minute,
	}
	v2 := legacy.ToV2()

	if v2.Phase != "F3" {
		t.Errorf("ToV2().Phase = %q, want F3", v2.Phase)
	}
	if v2.TaskDesc != "test" {
		t.Errorf("ToV2().TaskDesc = %q, want test", v2.TaskDesc)
	}
	if v2.Timeout != time.Minute {
		t.Errorf("ToV2().Timeout = %v, want 1m", v2.Timeout)
	}
	if v2.AsyncMode {
		t.Error("ToV2().AsyncMode should be false (legacy default)")
	}
}

func TestDefaultPhaseConfigV2(t *testing.T) {
	cfg := DefaultPhaseConfigV2("F0")

	if cfg.Phase != "F0" {
		t.Errorf("DefaultPhaseConfigV2 phase = %q, want F0", cfg.Phase)
	}
	if cfg.Parallelism != 3 {
		t.Errorf("DefaultPhaseConfigV2 Parallelism = %d, want 3", cfg.Parallelism)
	}
	if cfg.AsyncMode {
		t.Error("DefaultPhaseConfigV2 AsyncMode should be false")
	}
	if cfg.FailurePolicy != FailurePolicyFailFast {
		t.Errorf("DefaultPhaseConfigV2 FailurePolicy = %v, want FailFast", cfg.FailurePolicy)
	}
	if cfg.SkipMatrix != nil {
		t.Error("DefaultPhaseConfigV2 SkipMatrix should be nil")
	}
	if cfg.Steps != nil {
		t.Error("DefaultPhaseConfigV2 Steps should be nil")
	}
}

func TestFailurePolicyString(t *testing.T) {
	tests := []struct {
		policy FailurePolicy
		want   string
	}{
		{FailurePolicyFailFast, "fail_fast"},
		{FailurePolicyContinueOnError, "continue_on_error"},
		{FailurePolicy(99), "unknown"},
	}
	for _, tt := range tests {
		got := tt.policy.String()
		if got != tt.want {
			t.Errorf("FailurePolicy(%d).String() = %q, want %q", tt.policy, got, tt.want)
		}
	}
}

func TestRunPhaseV2WithCustomSteps(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ctx := context.Background()

	// Ejecutar solo Memory + Save (sin Think, Delegate, Git, Quality)
	cfg := PhaseConfigV2{
		Phase:    "F0",
		TaskDesc: "custom steps test",
		Steps:    []Step{StepMemory, StepSave},
	}

	result, err := o.runPhaseV2(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	// Think NO se ejecutó → TasksPlanned debe ser 0
	if result.TasksPlanned != 0 {
		t.Errorf("expected TasksPlanned=0 (Think skipped), got %d", result.TasksPlanned)
	}
}

func TestRunPhaseV2DelegatesToRunPhaseV2(t *testing.T) {
	// Verificar que RunPhase() llama a runPhaseV2 internamente
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, NewTaskManager(0), nil)
	ctx := context.Background()

	legacy := PhaseConfig{
		Phase:    "F0",
		TaskDesc: "delegation test",
	}

	result, err := o.RunPhase(ctx, legacy)
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
