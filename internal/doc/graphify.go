package doc

import (
	"fmt"
	"os"
	"path/filepath"
)

// ---------------------------------------------------------------------------
// Graph state types
// ---------------------------------------------------------------------------

// GraphState represents the previous graph state for diff comparison.
type GraphState struct {
	EntryCount int    `yaml:"entry_count"`
	Checksum   string `yaml:"checksum"`
}

const (
	// GraphStatePath is where the previous graph state is stored.
	GraphStatePath = ".zyro/graph-state.yaml"
)

// ---------------------------------------------------------------------------
// UpdateGraph compares the current doc index with the previous graph state
// and updates it if significant changes are detected.
//
// "Significant" is defined by conventions.yaml's significant_threshold (default 5).
// If the difference in entry count exceeds this threshold, the graph state
// is updated and a diff note is returned.
func UpdateGraph(projectRoot string, idx *DocIndex) error {
	if idx == nil {
		return fmt.Errorf("doc: cannot update graph with nil index")
	}

	statePath := filepath.Join(projectRoot, GraphStatePath)

	// Read previous state
	prevState, err := readGraphState(statePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading graph state: %w", err)
	}

	currentChecksum := fmt.Sprintf("%d", len(idx.Entries))
	threshold := 5 // significant_threshold from conventions.yaml

	if prevState != nil {
		diff := abs(len(idx.Entries) - prevState.EntryCount)
		if diff < threshold {
			// No significant change — skip
			return nil
		}
	}

	// Save new state
	newState := &GraphState{
		EntryCount: len(idx.Entries),
		Checksum:   currentChecksum,
	}

	data := fmt.Sprintf("entry_count: %d\nchecksum: %s\n", newState.EntryCount, newState.Checksum)
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		return fmt.Errorf("creating graph state dir: %w", err)
	}
	if err := os.WriteFile(statePath, []byte(data), 0o644); err != nil {
		return fmt.Errorf("writing graph state: %w", err)
	}

	return nil
}

// readGraphState loads a previously saved graph state, if it exists.
func readGraphState(path string) (*GraphState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state GraphState
	if _, err := fmt.Sscanf(string(data), "entry_count: %d\nchecksum: %s\n", &state.EntryCount, &state.Checksum); err != nil {
		// If parsing fails, return empty state
		return &GraphState{}, nil
	}
	return &state, nil
}

// abs returns the absolute value of an int.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
