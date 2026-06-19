package boomerang

import (
	"encoding/json"
	"testing"
)

// ── TaskStatus serialization tests ──────────────────────────────────────

func TestTaskStatus_MarshalJSON(t *testing.T) {
	tests := []struct {
		status TaskStatus
		want   string
	}{
		{TaskPending, `"pending"`},
		{TaskRunning, `"running"`},
		{TaskDone, `"done"`},
		{TaskFailed, `"failed"`},
		{TaskCancelled, `"cancelled"`},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatalf("MarshalJSON error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("MarshalJSON(%v) = %s, want %s", tt.status, string(got), tt.want)
			}
		})
	}
}

func TestTaskStatus_UnmarshalJSON_String(t *testing.T) {
	tests := []struct {
		input string
		want  TaskStatus
	}{
		{`"pending"`, TaskPending},
		{`"running"`, TaskRunning},
		{`"done"`, TaskDone},
		{`"failed"`, TaskFailed},
		{`"cancelled"`, TaskCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var s TaskStatus
			if err := json.Unmarshal([]byte(tt.input), &s); err != nil {
				t.Fatalf("UnmarshalJSON error: %v", err)
			}
			if s != tt.want {
				t.Errorf("UnmarshalJSON(%s) = %v, want %v", tt.input, s, tt.want)
			}
		})
	}
}

func TestTaskStatus_UnmarshalJSON_Number(t *testing.T) {
	tests := []struct {
		input string
		want  TaskStatus
	}{
		{`0`, TaskPending},
		{`1`, TaskRunning},
		{`2`, TaskDone},
		{`3`, TaskFailed},
		{`4`, TaskCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var s TaskStatus
			if err := json.Unmarshal([]byte(tt.input), &s); err != nil {
				t.Fatalf("UnmarshalJSON error: %v", err)
			}
			if s != tt.want {
				t.Errorf("UnmarshalJSON(%s) = %v, want %v", tt.input, s, tt.want)
			}
		})
	}
}

// ── Task struct serialization tests ─────────────────────────────────────

func TestTask_MarshalJSON_StatusIsString(t *testing.T) {
	task := &Task{
		ID:     "F0-test-1",
		Name:   "test",
		Agent:  "zyro-phase-0-patterns",
		Phase:  "F0",
		Status: TaskRunning,
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Marshal task error: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal task result error: %v", err)
	}

	status, ok := decoded["status"].(string)
	if !ok {
		t.Fatalf("status field is not a string, got %T (%v)", decoded["status"], decoded["status"])
	}
	if status != "running" {
		t.Errorf("status = %q, want %q", status, "running")
	}
}

func TestTask_UnmarshalJSON_StatusString(t *testing.T) {
	input := `{"id":"F0-test-1","name":"test","status":"done"}`
	var task Task
	if err := json.Unmarshal([]byte(input), &task); err != nil {
		t.Fatalf("Unmarshal task error: %v", err)
	}
	if task.Status != TaskDone {
		t.Errorf("Status = %v, want %v", task.Status, TaskDone)
	}
}

func TestTask_UnmarshalJSON_StatusNumber(t *testing.T) {
	input := `{"id":"F0-test-2","name":"test2","status":1}`
	var task Task
	if err := json.Unmarshal([]byte(input), &task); err != nil {
		t.Fatalf("Unmarshal task error: %v", err)
	}
	if task.Status != TaskRunning {
		t.Errorf("Status = %v, want %v", task.Status, TaskRunning)
	}
}

// ── Round-trip test ─────────────────────────────────────────────────────

func TestTaskStatus_RoundTrip(t *testing.T) {
	original := TaskRunning

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded TaskStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded != original {
		t.Errorf("Round-trip: got %v, want %v", decoded, original)
	}
}

// ── Export test ─────────────────────────────────────────────────────────

func TestExportTaskStatusString(t *testing.T) {
	tests := []struct {
		status TaskStatus
		want   string
	}{
		{TaskPending, "pending"},
		{TaskRunning, "running"},
		{TaskDone, "done"},
		{TaskFailed, "failed"},
		{TaskCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := ExportTaskStatusString(tt.status)
			if got != tt.want {
				t.Errorf("ExportTaskStatusString(%v) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}
