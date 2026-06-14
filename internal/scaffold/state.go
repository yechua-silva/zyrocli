package scaffold

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// State holds the initialization state of a scaffolded project.
type State struct {
	Initialized bool   `json:"initialized"`
	ProjectName string `json:"project_name"`
	TargetDir   string `json:"target_dir"`
	Version     string `json:"version"`
}

// StateFileName is the relative path of the state file inside the project.
const StateFileName = ".zyro/state.json"

// WriteState saves the initialization state to the project directory.
func WriteState(dir string, s *State) error {
	stateDir := filepath.Join(dir, ".zyro")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, StateFileName), data, 0644)
}

// ReadState reads the initialization state from a project directory.
// Returns nil if no state file exists (project not initialized).
func ReadState(dir string) (*State, error) {
	data, err := os.ReadFile(filepath.Join(dir, StateFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // not initialized
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
