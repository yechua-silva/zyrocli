package scaffold

import (
	"embed"
	"fmt"
)

//go:embed templates/go-project/scripts/*
var scriptsFS embed.FS

// ReadScript reads a script from the embedded scripts filesystem by name.
// Valid names: "explorer.py", "test-runner.py", "linter.py".
func ReadScript(name string) ([]byte, error) {
	data, err := scriptsFS.ReadFile("templates/go-project/scripts/" + name)
	if err != nil {
		return nil, fmt.Errorf("script %q not found: %w", name, err)
	}
	return data, nil
}
