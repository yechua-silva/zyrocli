package planning

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Decomposer tests
// ---------------------------------------------------------------------------

func TestDecomposer_EmptyStory(t *testing.T) {
	d := NewDecomposer(DecomposerConfig{ProjectName: "test"})
	result := d.Decompose("", "")
	if result.HasFeatures() {
		t.Error("expected no features for empty story")
	}
	if len(result.Errors) == 0 {
		t.Error("expected error for empty story")
	}
}

func TestDecomposer_SingleStory(t *testing.T) {
	d := NewDecomposer(DecomposerConfig{ProjectName: "test"})
	result := d.Decompose("I want to parse handoff.yaml", "YAML file is read and validated")
	if !result.HasFeatures() {
		t.Fatal("expected features for valid story")
	}
	if len(result.Features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(result.Features))
	}
	if !strings.Contains(result.Features[0].Name, "parse") {
		t.Errorf("expected feature name to contain 'parse', got %q", result.Features[0].Name)
	}
	if result.Features[0].Acceptance != "YAML file is read and validated" {
		t.Errorf("expected acceptance to be preserved, got %q", result.Features[0].Acceptance)
	}
}

func TestDecomposer_BulletListStory(t *testing.T) {
	d := NewDecomposer(DecomposerConfig{ProjectName: "test"})
	story := `I want to set up the project:
- Initialize Go module
- Create directory structure
- Add CI configuration`
	result := d.Decompose(story, "Project scaffold exists")
	if !result.HasFeatures() {
		t.Fatal("expected features for bullet list story")
	}
	if len(result.Features) != 3 {
		t.Fatalf("expected 3 features, got %d", len(result.Features))
	}
	if result.Features[0].ID != "F1" {
		t.Errorf("expected first feature ID F1, got %q", result.Features[0].ID)
	}
}

func TestDecomposer_NumberedListStory(t *testing.T) {
	d := NewDecomposer(DecomposerConfig{ProjectName: "test"})
	story := `Steps to implement:
1. Parse the input file
2. Validate the data
3. Generate the output`
	result := d.Decompose(story, "Success criteria")
	if !result.HasFeatures() {
		t.Fatal("expected features for numbered list story")
	}
	if len(result.Features) != 3 {
		t.Fatalf("expected 3 features, got %d", len(result.Features))
	}
}

func TestDecomposer_AndConjunction(t *testing.T) {
	d := NewDecomposer(DecomposerConfig{ProjectName: "test"})
	result := d.Decompose("I want to parse the handoff file and generate a task list", "")
	if !result.HasFeatures() {
		t.Fatal("expected features for compound story")
	}
	// "and" split only triggers when each part has >=3 words
	if len(result.Features) < 1 {
		t.Fatal("expected at least 1 feature")
	}
}

func TestDecomposer_FeatureNameCleaning(t *testing.T) {
	d := NewDecomposer(DecomposerConfig{ProjectName: "test"})
	result := d.Decompose("As an operator, I want to run verification checks", "")
	if !result.HasFeatures() {
		t.Fatal("expected features for story")
	}
	name := result.Features[0].Name
	if strings.Contains(name, "as an operator") {
		t.Errorf("name should not contain 'as an operator': %q", name)
	}
}

func TestFeatureByID(t *testing.T) {
	features := []Feature{
		{ID: "F1", Name: "parse"},
		{ID: "F2", Name: "validate"},
	}
	f := FeatureByID(features, "F2")
	if f == nil {
		t.Fatal("expected to find feature F2")
	}
	if f.Name != "validate" {
		t.Errorf("expected name 'validate', got %q", f.Name)
	}

	f = FeatureByID(features, "F99")
	if f != nil {
		t.Error("expected nil for unknown ID")
	}
}

func TestSummarizeFeatures_Empty(t *testing.T) {
	s := SummarizeFeatures(nil)
	if !strings.Contains(s, "No features") {
		t.Error("expected 'No features' summary for empty list")
	}
}

func TestSummarizeFeatures_WithData(t *testing.T) {
	features := []Feature{
		{ID: "F1", Name: "parse", Description: "Parse input", Complexity: "small"},
		{ID: "F2", Name: "validate", Description: "Validate data", Complexity: "medium", Dependencies: []string{"F1"}},
	}
	s := SummarizeFeatures(features)
	if !strings.Contains(s, "Total features: 2") {
		t.Error("expected count in summary")
	}
	if !strings.Contains(s, "F1") {
		t.Error("expected F1 in summary")
	}
	if !strings.Contains(s, "F2") {
		t.Error("expected F2 in summary")
	}
}

// ---------------------------------------------------------------------------
// Scheduler tests
// ---------------------------------------------------------------------------

func TestScheduler_Empty(t *testing.T) {
	sched := NewScheduler(SchedulerConfig{})
	schedule, err := sched.Schedule("test", nil)
	if err == nil {
		t.Error("expected error for nil features")
	}
	if schedule == nil {
		t.Fatal("expected non-nil schedule")
	}
}

func TestScheduler_SingleFeature(t *testing.T) {
	sched := NewScheduler(SchedulerConfig{})
	features := []Feature{
		{ID: "F1", Name: "parse", Description: "Parse input", Complexity: "small"},
	}
	schedule, err := sched.Schedule("test", features)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schedule.PhaseCount() != 1 {
		t.Errorf("expected 1 phase, got %d", schedule.PhaseCount())
	}
	if schedule.TotalEntries() != 1 {
		t.Errorf("expected 1 entry, got %d", schedule.TotalEntries())
	}
}

func TestScheduler_TopologicalOrder(t *testing.T) {
	sched := NewScheduler(SchedulerConfig{})
	// F3 depends on F2, F2 depends on F1
	features := []Feature{
		{ID: "F1", Name: "foundation", Description: "Base layer", Complexity: "medium"},
		{ID: "F2", Name: "middleware", Description: "Middleware layer", Complexity: "medium", Dependencies: []string{"F1"}},
		{ID: "F3", Name: "application", Description: "App layer", Complexity: "large", Dependencies: []string{"F2"}},
	}
	schedule, err := sched.Schedule("test", features)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// F1 should be phase 1, F2 phase 2, F3 phase 3
	if schedule.PhaseCount() != 3 {
		t.Fatalf("expected 3 phases, got %d", schedule.PhaseCount())
	}
	if len(schedule.Phases[0]) != 1 || schedule.Phases[0][0].Feature.ID != "F1" {
		t.Errorf("expected F1 in phase 1")
	}
	if len(schedule.Phases[1]) != 1 || schedule.Phases[1][0].Feature.ID != "F2" {
		t.Errorf("expected F2 in phase 2")
	}
	if len(schedule.Phases[2]) != 1 || schedule.Phases[2][0].Feature.ID != "F3" {
		t.Errorf("expected F3 in phase 3")
	}
}

func TestScheduler_ParallelFeatures(t *testing.T) {
	sched := NewScheduler(SchedulerConfig{})
	features := []Feature{
		{ID: "F1", Name: "parse", Description: "Parse input", Complexity: "small"},
		{ID: "F2", Name: "validate", Description: "Validate data", Complexity: "medium"},
		{ID: "F3", Name: "generate", Description: "Generate output", Complexity: "small"},
	}
	schedule, err := sched.Schedule("test", features)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All independent features should be in phase 1
	if schedule.PhaseCount() != 1 {
		t.Fatalf("expected 1 phase for no dependencies, got %d", schedule.PhaseCount())
	}
	if schedule.TotalEntries() != 3 {
		t.Errorf("expected 3 entries, got %d", schedule.TotalEntries())
	}
}

func TestScheduler_UnknownDependency(t *testing.T) {
	sched := NewScheduler(SchedulerConfig{})
	features := []Feature{
		{ID: "F1", Name: "main", Dependencies: []string{"F99"}},
	}
	_, err := sched.Schedule("test", features)
	if err == nil {
		t.Error("expected error for unknown dependency")
	}
	if !strings.Contains(err.Error(), "F99") {
		t.Errorf("expected error to mention F99, got: %v", err)
	}
}

func TestScheduler_PhaseLimit(t *testing.T) {
	sched := NewScheduler(SchedulerConfig{MaxPhases: 2})
	features := []Feature{
		{ID: "F1", Name: "a", Dependencies: []string{}},
		{ID: "F2", Name: "b", Dependencies: []string{"F1"}},
		{ID: "F3", Name: "c", Dependencies: []string{"F2"}},
	}
	schedule, err := sched.Schedule("test", features)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With MaxPhases=2 and 3-level chain, F3 won't be scheduled
	if schedule.PhaseCount() > 2 {
		t.Errorf("expected at most 2 phases, got %d", schedule.PhaseCount())
	}
}

// ---------------------------------------------------------------------------
// ValidateNoCircularDeps tests
// ---------------------------------------------------------------------------

func TestValidateNoCircularDeps_NoCycle(t *testing.T) {
	features := []Feature{
		{ID: "F1", Dependencies: []string{}},
		{ID: "F2", Dependencies: []string{"F1"}},
	}
	err := ValidateNoCircularDeps(features)
	if err != nil {
		t.Errorf("unexpected error for no cycle: %v", err)
	}
}

func TestValidateNoCircularDeps_DirectCycle(t *testing.T) {
	features := []Feature{
		{ID: "F1", Dependencies: []string{"F2"}},
		{ID: "F2", Dependencies: []string{"F1"}},
	}
	err := ValidateNoCircularDeps(features)
	if err == nil {
		t.Fatal("expected error for circular dependency")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected 'circular' in error, got: %v", err)
	}
}

func TestValidateNoCircularDeps_Empty(t *testing.T) {
	err := ValidateNoCircularDeps(nil)
	if err != nil {
		t.Errorf("unexpected error for empty: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Schedule utility tests
// ---------------------------------------------------------------------------

func TestSchedule_Markdown(t *testing.T) {
	schedule := &Schedule{
		Project: "test",
		Phases: [][]ScheduleEntry{
			{
				{Feature: Feature{ID: "F1", Name: "parse", Complexity: "small"}, Priority: 1, Phase: 1},
			},
		},
	}
	md := schedule.Markdown()
	if !strings.Contains(md, "Schedule: test") {
		t.Error("expected schedule header")
	}
	if !strings.Contains(md, "parse") {
		t.Error("expected feature name in markdown")
	}
}

func TestSchedule_TotalEntries(t *testing.T) {
	s := &Schedule{
		Phases: [][]ScheduleEntry{
			{{Feature: Feature{ID: "F1"}}, {Feature: Feature{ID: "F2"}}},
			{{Feature: Feature{ID: "F3"}}},
		},
	}
	if s.TotalEntries() != 3 {
		t.Errorf("expected 3 total entries, got %d", s.TotalEntries())
	}
}

func TestPhaseBoundaryDescription(t *testing.T) {
	desc := PhaseBoundaryDescription([]ScheduleEntry{
		{Feature: Feature{Name: "parse"}},
		{Feature: Feature{Name: "validate"}},
	})
	if !strings.Contains(desc, "parse") || !strings.Contains(desc, "validate") {
		t.Errorf("expected both feature names in description, got: %s", desc)
	}

	empty := PhaseBoundaryDescription(nil)
	if !strings.Contains(empty, "empty") {
		t.Error("expected 'empty phase' for nil input")
	}
}
