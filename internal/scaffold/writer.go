package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WriteProject creates the project directory tree from a map of relative paths to content.
// Keys ending in "/" are treated as directory markers. All file parent directories are
// created automatically. On any error the entire target directory is removed.
func WriteProject(targetDir string, files map[string]string) error {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}

	// Sort: directories before files, then alphabetically.
	sort.Slice(keys, func(i, j int) bool {
		iDir := strings.HasSuffix(keys[i], "/")
		jDir := strings.HasSuffix(keys[j], "/")
		if iDir != jDir {
			return iDir // directories first
		}
		return keys[i] < keys[j]
	})

	for _, key := range keys {
		fullPath := filepath.Join(targetDir, key)

		if strings.HasSuffix(key, "/") {
			// Directory marker.
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				cleanup(targetDir)
				return fmt.Errorf("scaffold: create directory %s: %w", key, err)
			}
			continue
		}

		// Ensure parent directory exists.
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			cleanup(targetDir)
			return fmt.Errorf("scaffold: create parent directory for %s: %w", key, err)
		}

		if err := os.WriteFile(fullPath, []byte(files[key]), 0644); err != nil {
			cleanup(targetDir)
			return fmt.Errorf("scaffold: write file %s: %w", key, err)
		}
	}

	return nil
}

// cleanup removes the entire target directory. Safe to call on non-existent paths.
func cleanup(targetDir string) {
	_ = os.RemoveAll(targetDir)
}
