// Package doc provides documentation tooling: index, search, sync, export,
// and graphify integration. It connects the local filesystem index with
// Engram's persistent memory for traceable SDD artifacts.
package doc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Index types
// ---------------------------------------------------------------------------

// IndexEntry represents a single entry in the documentation index.
// Each entry maps to an Engram topic key with metadata for search and sync.
type IndexEntry struct {
	TopicKey     string `yaml:"topic_key" json:"topic_key"`
	Type         string `yaml:"type" json:"type"`
	ObservationID int   `yaml:"observation_id" json:"observation_id"`
	LastModified string `yaml:"last_modified" json:"last_modified"`
	ChangeName   string `yaml:"change_name,omitempty" json:"change_name,omitempty"`
}

// DocIndex is the complete documentation index stored in .zyro/doc-index.yaml.
type DocIndex struct {
	GeneratedAt string       `yaml:"generated_at" json:"generated_at"`
	Entries     []IndexEntry `yaml:"entries" json:"entries"`
}

// ---------------------------------------------------------------------------
// Paths
// ---------------------------------------------------------------------------

const (
	// DefaultConventionsPath is the default location of conventions.yaml
	// relative to the project root.
	DefaultConventionsPath = ".zyro/conventions.yaml"

	// DefaultIndexPath is the default location of the generated doc index.
	DefaultIndexPath = ".zyro/doc-index.yaml"

	// DefaultProjectRoot is the path used when no explicit root is given.
	DefaultProjectRoot = "."
)

// ---------------------------------------------------------------------------
// GenerateIndex builds a doc index from conventions.yaml and the filesystem.
//
// It reads known topic key patterns from conventions.yaml, resolves active
// changes from the openspec/changes/ directory, and produces a complete
// index of all known Engram entries for the project.
//
// The index is the single source of truth for what documentation exists
// and where it lives in Engram.
func GenerateIndex(projectRoot string) (*DocIndex, error) {
	convPath := filepath.Join(projectRoot, DefaultConventionsPath)
	convData, err := os.ReadFile(convPath)
	if err != nil {
		return nil, fmt.Errorf("doc: reading conventions: %w", err)
	}

	var conv Conventions
	if err := yaml.Unmarshal(convData, &conv); err != nil {
		return nil, fmt.Errorf("doc: parsing conventions: %w", err)
	}

	// Detect project name from conventions or fallback
	project := detectProjectName(conv)

	// Discover active changes from openspec/changes/
	changes, err := discoverChanges(projectRoot)
	if err != nil {
		// Non-fatal: changes directory may not exist yet
		changes = nil
	}

	entries := buildIndexEntries(conv, project, changes)

	return &DocIndex{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Entries:     entries,
	}, nil
}

// ---------------------------------------------------------------------------
// LoadIndex reads a previously saved doc index from disk.
func LoadIndex(projectRoot string) (*DocIndex, error) {
	path := filepath.Join(projectRoot, DefaultIndexPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("doc: loading index: %w", err)
	}

	var idx DocIndex
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("doc: parsing index: %w", err)
	}
	return &idx, nil
}

// SaveIndex writes the doc index to disk.
func SaveIndex(projectRoot string, idx *DocIndex) error {
	path := filepath.Join(projectRoot, DefaultIndexPath)

	// Ensure .zyro directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("doc: creating index dir: %w", err)
	}

	data, err := yaml.Marshal(idx)
	if err != nil {
		return fmt.Errorf("doc: marshaling index: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("doc: writing index: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// Conventions mirrors the .zyro/conventions.yaml structure.
type Conventions struct {
	TopicKeys struct {
		Project map[string]string `yaml:"project"`
		Change  map[string]string `yaml:"change"`
		Graph   map[string]string `yaml:"graph"`
	} `yaml:"topic_keys"`
}

// detectProjectName extracts the project name from conventions or falls back
// to the current directory name.
func detectProjectName(conv Conventions) string {
	// The project context key is "zyro/{project}/context".
	// If it has the pattern, extract the project name.
	ctxKey := conv.TopicKeys.Project["context"]
	if strings.HasPrefix(ctxKey, "zyro/") && strings.HasSuffix(ctxKey, "/context") {
		parts := strings.Split(strings.TrimPrefix(ctxKey, "zyro/"), "/")
		if len(parts) > 0 && parts[0] != "{project}" {
			return parts[0]
		}
	}
	return "unknown"
}

// discoverChanges lists active SDD changes from openspec/changes/.
func discoverChanges(projectRoot string) ([]string, error) {
	changesDir := filepath.Join(projectRoot, "openspec", "changes")
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return nil, err
	}

	var changes []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "archive" {
			changes = append(changes, e.Name())
		}
	}
	return changes, nil
}

// resolveKey replaces {project} and {change} placeholders in a topic key pattern.
func resolveKey(pattern, project, change string) string {
	result := strings.ReplaceAll(pattern, "{project}", project)
	result = strings.ReplaceAll(result, "{change}", change)
	return result
}

// buildIndexEntries creates IndexEntry values from conventions and active changes.
func buildIndexEntries(conv Conventions, project string, changes []string) []IndexEntry {
	var entries []IndexEntry

	// Project-scoped keys
	for name, pattern := range conv.TopicKeys.Project {
		entries = append(entries, IndexEntry{
			TopicKey:     resolveKey(pattern, project, ""),
			Type:         "project/" + name,
			ObservationID: 0,
			LastModified: "",
		})
	}

	// Graph-scoped keys
	for name, pattern := range conv.TopicKeys.Graph {
		entries = append(entries, IndexEntry{
			TopicKey:     resolveKey(pattern, project, ""),
			Type:         "graph/" + name,
			ObservationID: 0,
			LastModified: "",
		})
	}

	// Change-scoped keys — one set per active change
	for _, change := range changes {
		for name, pattern := range conv.TopicKeys.Change {
			entries = append(entries, IndexEntry{
				TopicKey:      resolveKey(pattern, project, change),
				Type:          "change/" + name,
				ObservationID: 0,
				LastModified:  "",
				ChangeName:    change,
			})
		}
	}

	return entries
}
