package boomerang

import (
	"context"
	"os/exec"
	"strings"
)

// GitStatus representa el estado del repositorio.
type GitStatus string

const (
	GitClean GitStatus = "clean"
	GitDirty GitStatus = "dirty"
	GitError GitStatus = "error"
)

// GitStep verifica el estado del repositorio Git ejecutando
// `git status --porcelain`. Retorna "clean", "dirty" o "error".
func (o *BoomerangOrchestrator) GitStep(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return string(GitError), nil
	}

	if len(strings.TrimSpace(string(output))) == 0 {
		return string(GitClean), nil
	}

	return string(GitDirty), nil
}
