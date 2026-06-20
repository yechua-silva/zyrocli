package boomerang

import (
	"encoding/json"
	"testing"
)

func TestNewCriteriaSummary(t *testing.T) {
	// Nil slice
	s := NewCriteriaSummary(nil)
	if s == nil {
		t.Fatal("NewCriteriaSummary(nil) returned nil")
	}
	if s.Total != 0 || s.Pending != 0 || s.Verified != 0 || s.Failed != 0 {
		t.Errorf("nil: expected all zeros, got %+v", s)
	}

	// Empty slice
	s = NewCriteriaSummary([]AcceptanceCriteria{})
	if s.Total != 0 || s.Pending != 0 || s.Verified != 0 || s.Failed != 0 {
		t.Errorf("empty: expected all zeros, got %+v", s)
	}
}

func TestCriteriaSummaryCounts(t *testing.T) {
	criteria := []AcceptanceCriteria{
		{ID: "AC-001", Status: CriteriaVerified},
		{ID: "AC-002", Status: CriteriaPending},
		{ID: "AC-003", Status: CriteriaFailed},
		{ID: "AC-004", Status: CriteriaVerified},
		{ID: "AC-005", Status: CriteriaFailed},
	}

	s := NewCriteriaSummary(criteria)
	if s.Total != 5 {
		t.Errorf("Total = %d, want 5", s.Total)
	}
	if s.Verified != 2 {
		t.Errorf("Verified = %d, want 2", s.Verified)
	}
	if s.Pending != 1 {
		t.Errorf("Pending = %d, want 1", s.Pending)
	}
	if s.Failed != 2 {
		t.Errorf("Failed = %d, want 2", s.Failed)
	}
}

func TestAcceptanceCriteriaJSON(t *testing.T) {
	c := AcceptanceCriteria{
		ID:          "AC-001",
		Description: "Test criterion",
		Phase:       "F1",
		Status:      CriteriaPending,
		Source:      "spec",
		TaskID:      "implement-auth",
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}

	var decoded AcceptanceCriteria
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.ID != c.ID || decoded.Description != c.Description || decoded.Status != c.Status {
		t.Errorf("JSON roundtrip: got %+v, want %+v", decoded, c)
	}
}

func TestAcceptanceCriteriaJSONOmitEmpty(t *testing.T) {
	c := AcceptanceCriteria{
		ID:          "AC-001",
		Description: "Test",
		Phase:       "F1",
		Status:      CriteriaPending,
		Source:      "spec",
		// TaskID intentionally empty
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}

	// TaskID should not appear in JSON when empty (omitempty)
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, exists := raw["task_id"]; exists {
		t.Error("task_id should be omitted when empty")
	}
}

func TestExtractCriteriaFromDAG(t *testing.T) {
	// Nil DAG
	result := ExtractCriteriaFromDAG(nil)
	if result != nil {
		t.Errorf("nil DAG: expected nil, got %v", result)
	}

	// DAG with no criteria
	dag := &TaskDAG{
		Tasks: []TaskSpec{
			{ID: 1, Name: "task1"},
			{ID: 2, Name: "task2"},
		},
	}
	result = ExtractCriteriaFromDAG(dag)
	if len(result) != 0 {
		t.Errorf("empty criteria: expected 0, got %d", len(result))
	}

	// DAG with criteria
	dag = &TaskDAG{
		Tasks: []TaskSpec{
			{
				ID: 1, Name: "task1",
				AcceptanceCriteria: []AcceptanceCriteria{
					{ID: "AC-001", Status: CriteriaVerified},
				},
			},
			{
				ID: 2, Name: "task2",
				AcceptanceCriteria: []AcceptanceCriteria{
					{ID: "AC-002", Status: CriteriaPending},
					{ID: "AC-003", Status: CriteriaFailed},
				},
			},
		},
	}
	result = ExtractCriteriaFromDAG(dag)
	if len(result) != 3 {
		t.Errorf("with criteria: expected 3, got %d", len(result))
	}
}
