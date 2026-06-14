package handoff

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Parse reads a YAML handoff file and unmarshals it into a Payload.
// If path is "-", it reads from os.Stdin.
func Parse(path string) (*Payload, error) {
	var data []byte
	var err error

	switch path {
	case "-":
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("parse stdin: %w", err)
		}
	default:
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("parse (%s): %w", path, err)
		}
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("parse: empty input")
	}

	var payload Payload
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	return &payload, nil
}
