package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	helix "github.com/secko/zyrocli/internal/db/helix"
	"github.com/spf13/cobra"
)

var absorbDir string

// absorbCmd scans a docs/ directory and ingests markdown files into HelixDB
// as Document nodes with topic_key, content, and file_path properties.
var absorbCmd = &cobra.Command{
	Use:   "absorb [path]",
	Short: "Ingest markdown files from docs/ into HelixDB",
	Long: `Scan a directory (default: docs/) for .md files, read each one,
and create a Document node in HelixDB with:
  - topic_key:  the relative path without extension
  - content:    the file body (first 10000 bytes)
  - file_path:  the relative path from the scanned directory

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

		// Build HelixDB client.
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
			return fmt.Errorf("absorb: HelixDB connection failed: %w", err)
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

func init() {
	rootCmd.AddCommand(absorbCmd)
	absorbCmd.Flags().StringVarP(&absorbDir, "dir", "d", "docs/", "directory to scan for markdown files")
}
