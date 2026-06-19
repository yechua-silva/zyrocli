package scheduler

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// recentArtifacts walks openspec/ and .zyro/handoffs/ looking for files modified
// within the last 30 minutes. Returns their relative paths (sorted) or a message
// if none found. Does NOT read file contents — only stat mtime.
func recentArtifacts(cwd string) []string {
	since := time.Now().Add(-30 * time.Minute)
	dirs := []string{
		filepath.Join(cwd, "openspec"),
		filepath.Join(cwd, ".zyro", "handoffs"),
	}

	var paths []string
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			if info.ModTime().After(since) {
				rel, _ := filepath.Rel(cwd, path)
				paths = append(paths, rel)
			}
			return nil
		})
	}

	sort.Strings(paths)
	return paths
}

// writeHandoff creates .zyro/handoffs/<FASE>-handoff.md after a phase completes.
// The directory is created if it doesn't exist.
func writeHandoff(phaseName string, result *Result, nextPhase Phase) error {
	// Determine project root: walk up from cwd looking for .zyro/
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	zyroDir := filepath.Join(cwd, ".zyro")
	handoffDir := filepath.Join(zyroDir, "handoffs")

	if err := os.MkdirAll(handoffDir, 0755); err != nil {
		return fmt.Errorf("handoff: create dir %s: %w", handoffDir, err)
	}

	statusEmoji := "✅"
	if result.Status == StatusFail {
		statusEmoji = "❌"
	} else if result.Status == StatusAbort {
		statusEmoji = "⏹"
	}

	nextPhaseStr := string(nextPhase)
	if nextPhaseStr == "" {
		nextPhaseStr = "(ninguna — pipeline completo)"
	}

	// Listar artefactos recientes (últimos 30 min)
	artifacts := recentArtifacts(cwd)
	var artifactsBlock string
	if len(artifacts) > 0 {
		artifactsBlock = strings.Join(artifacts, "\n")
	} else {
		artifactsBlock = "(sin artefactos nuevos detectados)"
	}

	content := fmt.Sprintf(`# Handoff — Fase %s

**Generado:** %s
**Estado:** %s %s

---

## Resumen

%s

## Artefactos recientes

%s

## Siguiente fase sugerida

%s

## Instrucciones

Revisar los artefactos listados antes de continuar a la siguiente fase.
Si hay cambios solicitados por el humano, iterar antes de avanzar.
`,
		phaseName,
		time.Now().Format(time.RFC3339),
		statusEmoji, result.Status,
		result.Summary,
		artifactsBlock,
		nextPhaseStr,
	)

	filename := filepath.Join(handoffDir, fmt.Sprintf("%s-handoff.md", phaseName))
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("handoff: write %s: %w", filename, err)
	}

	return nil
}
