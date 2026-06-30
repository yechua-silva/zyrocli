package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	helix "github.com/secko/zyrocli/internal/db/helix"
	"github.com/secko/zyrocli/internal/codeparse"
	"github.com/spf13/cobra"
)

var (
	absorbDir  string
	absorbCode bool
)

// absorbCmd scans a docs/ directory and ingests markdown files into HelixDB
// as Document nodes with topic_key, content, and file_path properties.
var absorbCmd = &cobra.Command{
	Use:   "absorb [path]",
	Short: "Ingest docs or Go source code into HelixDB",
	Long: `Scan a directory (default: docs/) and create nodes in HelixDB.

Mode 1 — Markdown docs (default): scan .md files, create Document nodes with:
  - topic_key:  the relative path without extension
  - content:    the file body (first 10000 bytes)
  - file_path:  the relative path from the scanned directory

Mode 2 — Go source code (--code flag): scan .go files, parse with go/ast,
create CodeNode entries with:
  - topic_key:  the file name without extension
  - content:    summary of package, exported functions, types, deps
  - file_path:  the file path
  - language:   "go"
  - node_type:  "CodeNode"

Connects to HelixDB at localhost:6969 (override via HELIX_URL env var).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine the base directory.
		baseDir := absorbDir
		if len(args) > 0 {
			baseDir = args[0]
		}

		// Resolve to absolute path for scanning.
		absBase, err := filepath.Abs(baseDir)
		if err != nil {
			return fmt.Errorf("absorb: resolve path %q: %w", baseDir, err)
		}

		// Check that the directory exists.
		info, err := os.Stat(absBase)
		if err != nil {
			return fmt.Errorf("absorb: path %q: %w", baseDir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("absorb: %q is not a directory", baseDir)
		}

		// If --code flag is set, parse Go source files
		if absorbCode {
			return absorbCodeFiles(cmd, absBase)
		}

		// Build HelixDB client
		client, err := newHelixClient()
		if err != nil {
			return err
		}
		defer client.Close()

		// Walk the directory tree looking for .md files.
		var ingested int
		var errors []string

		err = filepath.WalkDir(absBase, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
				return nil
			}

			// Read file content.
			content, err := os.ReadFile(path)
			if err != nil {
				errors = append(errors, fmt.Sprintf("read %s: %v", path, err))
				return nil // skip, continue with other files
			}

			// Truncate content to 10000 characters.
			body := string(content)
			if len(body) > 10000 {
				body = body[:10000]
			}

			// Compute relative path.
			relPath, err := filepath.Rel(absBase, path)
			if err != nil {
				errors = append(errors, fmt.Sprintf("relative path %s: %v", path, err))
				return nil
			}

			// topic_key is the relative path without extension.
			topicKey := strings.TrimSuffix(relPath, filepath.Ext(relPath))

			props := map[string]any{
				"topic_key": topicKey,
				"content":   body,
				"file_path": relPath,
			}

			if _, err := client.CreateNode(context.Background(), "Document", props); err != nil {
				errors = append(errors, fmt.Sprintf("create node for %s: %v", relPath, err))
				return nil
			}

			ingested++
			return nil
		})
		if err != nil {
			return fmt.Errorf("absorb: walk %q: %w", baseDir, err)
		}

		// Report results.
		cmd.Printf("✓ Ingested %d document(s) from %s/\n", ingested, baseDir)
		if len(errors) > 0 {
			for _, e := range errors {
				cmd.PrintErrf("  ⚠ %s\n", e)
			}
		}

		return nil
	},
}

// newHelixClient creates a HelixDB client from environment configuration.
// Reads HELIX_URL and HELIX_PROJECT_ID env vars with sensible defaults.
func newHelixClient() (*helix.Client, error) {
	helixURL := os.Getenv("HELIX_URL")
	if helixURL == "" {
		helixURL = "http://localhost:6969"
	}
	opts := []helix.Option{helix.WithBaseURL(helixURL)}
	if pid := os.Getenv("HELIX_PROJECT_ID"); pid != "" {
		opts = append(opts, helix.WithProjectID(pid))
	}

	client, err := helix.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("absorb: HelixDB connection failed: %w", err)
	}

	if err := client.EnsureStarted(context.Background()); err != nil {
		client.Close()
		return nil, fmt.Errorf("cannot connect to HelixDB: %w", err)
	}

	return client, nil
}

// absorbCodeFiles parses Go source files in the given directory and creates
// CodeNode entries in HelixDB with parsed package info, functions, types, and imports.
func absorbCodeFiles(cmd *cobra.Command, absBase string) error {
	results, err := codeparse.ParseDir(absBase)
	if err != nil {
		return fmt.Errorf("absorb: parse code: %w", err)
	}

	if len(results) == 0 {
		cmd.Printf("⚠ No Go files found in %s/\n", absBase)
		return nil
	}

	// Build HelixDB client
	client, err := newHelixClient()
	if err != nil {
		return err
	}
	defer client.Close()

	var ingested int
	for _, result := range results {
		summary := codeparse.GenerateSummary(result)
		props := map[string]any{
			"path":     result.File,
			"name":     filepath.Base(result.File),
			"summary":  summary,
			"language": "go",
			"hash":     "", // hash no disponible sin leer el archivo completo
		}
		if _, err := client.CreateNode(context.Background(), "CodeNode", props); err != nil {
			cmd.PrintErrf("  ⚠ error ingesting %s: %v\n", result.File, err)
			continue
		}
		ingested++
	}

	cmd.Printf("✓ Ingested %d Go file(s) from %s/\n", ingested, absBase)
	return nil
}

func init() {
	rootCmd.AddCommand(absorbCmd)
	absorbCmd.Flags().StringVarP(&absorbDir, "dir", "d", "docs/", "directory to scan for markdown files")
	absorbCmd.Flags().BoolVarP(&absorbCode, "code", "c", false, "parse Go source files instead of markdown docs")
}
