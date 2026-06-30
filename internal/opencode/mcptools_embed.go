// DEPRECATED: MCP tools now load via opencode-lazy-loader plugin.
// This file is kept for backward compatibility but should not be used
// for new configurations. See internal/boomerang/ for the new pipeline.
package opencode

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed mcptools/*
var mcptoolsFS embed.FS

// ZyroDir is the ZyroCLI data directory where extracted tools live.
var ZyroDir = "~/.config/zyrocli"

// WriteMCPTools extracts the embedded MCP Python tools to the ZyroCLI data dir.
// Returns the mcp-tools directory path.
func WriteMCPTools() (string, error) {
	zyroDir := expandHome(ZyroDir)
	mcpDir := filepath.Join(zyroDir, "mcp-tools")

	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		return "", fmt.Errorf("opencode: create mcp dir %s: %w", mcpDir, err)
	}

	err := fs.WalkDir(mcptoolsFS, "mcptools", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// path = "mcptools/runner.py" → extract "runner.py"
		relPath := strings.TrimPrefix(path, "mcptools/")

		// Skip .venv files (shouldn't be embedded, but just in case)
		if strings.Contains(relPath, ".venv/") {
			return nil
		}

		outPath := filepath.Join(mcpDir, relPath)

		// Crear directorio padre (defensivo, para archivos anidados)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return fmt.Errorf("opencode: create dir %s: %w", filepath.Dir(outPath), err)
		}

		content, err := mcptoolsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("opencode: read embedded %s: %w", path, err)
		}

		if err := os.WriteFile(outPath, content, 0644); err != nil {
			return fmt.Errorf("opencode: write %s: %w", outPath, err)
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return mcpDir, nil
}
