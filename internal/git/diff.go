package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// ChangedFile representa un archivo modificado
type ChangedFile struct {
	Path    string // path actual
	Status  string // M(odified), A(dded), D(eleted), R100(enamed)
	OldPath string // path anterior (solo para renames)
}

// ChangedFiles obtiene archivos modificados vs un ref usando git diff --name-status
// ref: "HEAD", "HEAD~1", "main...HEAD", etc.
func ChangedFiles(ref string, dir string) ([]ChangedFile, error) {
	cmd := exec.Command("git", "diff", "--name-status", ref)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	return parseDiffOutput(string(output)), nil
}

// ChangedFilesBetween obtiene archivos modificados entre dos refs
func ChangedFilesBetween(from, to string, dir string) ([]ChangedFile, error) {
	ref := from + "..." + to
	return ChangedFiles(ref, dir)
}

func parseDiffOutput(output string) []ChangedFile {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return nil
	}

	lines := strings.Split(trimmed, "\n")

	var files []ChangedFile
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		path := parts[1]

		cf := ChangedFile{
			Status: status,
			Path:   path,
		}

		// Detectar renames (R100, R090, etc.)
		if strings.HasPrefix(status, "R") && len(parts) >= 3 {
			cf.OldPath = path
			cf.Path = parts[2]
		}

		files = append(files, cf)
	}

	return files
}

// IsModified retorna true si el archivo fue modificado o agregado
func (cf ChangedFile) IsModified() bool {
	return cf.Status == "M" || cf.Status == "A"
}

// IsRename retorna true si el archivo fue renombrado
func (cf ChangedFile) IsRename() bool {
	return strings.HasPrefix(cf.Status, "R")
}

// IsDeleted retorna true si el archivo fue borrado
func (cf ChangedFile) IsDeleted() bool {
	return cf.Status == "D"
}
