package boomerang

// ExportTaskStatusString expone TaskStatus.String() para tests externos.
func ExportTaskStatusString(s TaskStatus) string {
	return s.String()
}

// NewTaskForTest crea un Task con estado controlado para tests.
func NewTaskForTest(id TaskID, status TaskStatus) *Task {
	return &Task{
		ID:     id,
		Name:   string(id),
		Agent:  "test-agent",
		Phase:  "F0",
		Status: status,
	}
}
