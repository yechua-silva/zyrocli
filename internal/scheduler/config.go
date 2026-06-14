package scheduler

import (
	"fmt"
	"time"

	"github.com/secko/zyrocli/internal/handoff"
)

// LoadConfig reads handoff.yaml and extracts scheduler configuration.
func LoadConfig(path string) (*Config, error) {
	payload, err := handoff.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	timeout := 10 * time.Minute // default
	if payload.Limits.PhaseTimeout != "" {
		timeout, err = time.ParseDuration(payload.Limits.PhaseTimeout)
		if err != nil {
			return nil, fmt.Errorf("load config: invalid phase_timeout %q: %w", payload.Limits.PhaseTimeout, err)
		}
	}

	maxLoops := payload.Limits.MaxLoops
	if maxLoops == 0 {
		maxLoops = 5 // default
	}

	return &Config{
		Mode:         payload.Governance.Mode,
		Module:       payload.Governance.Module,
		GoVersion:    payload.Governance.GoVersion,
		MaxTasks:     payload.Limits.MaxTasks,
		MaxLines:     payload.Limits.MaxLines,
		MaxLoops:     maxLoops,
		PhaseTimeout: timeout,
	}, nil
}
