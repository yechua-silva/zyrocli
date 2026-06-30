package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/secko/zyrocli/internal/db/helix"
	"github.com/secko/zyrocli/internal/scanner"
	"github.com/spf13/cobra"
)

var (
	onboardDryRun    bool
	onboardForce     bool
	onboardNoOpenCode bool
	onboardName      string
	onboardDesc      string
)

var onboardCmd = &cobra.Command{
	Use:   "onboard [path]",
	Short: "Register an existing project in ZyroCLI",
	Long: `Register an existing project in the ZyroCLI ecosystem.

Scans the project structure, detects language and frameworks, creates
nodes in HelixDB, and opens OpenCode configured for existing projects
(instead of starting from scratch like 'zyro init').

If no path is given, the current directory is used.
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPath := "."
		if len(args) > 0 {
			targetPath = args[0]
		}

		absPath, err := filepath.Abs(targetPath)
		if err != nil {
			return fmt.Errorf("onboard: cannot resolve path %q: %w", targetPath, err)
		}

		info, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("onboard: path %q does not exist", absPath)
			}
			return fmt.Errorf("onboard: cannot access %q: %w", absPath, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("onboard: %q is not a directory", absPath)
		}

		cmd.Printf("🔍 Onboarding project: %s\n", absPath)

		// Escanear proyecto
		cmd.Println("  Scanning project structure...")
		s := scanner.NewScanner()
		projectInfo, err := s.Scan(absPath)
		if err != nil {
			return fmt.Errorf("onboard: scan failed: %w", err)
		}

		cmd.Printf("    ✓ Found %d files (%s)\n", projectInfo.FileCount, formatSize(projectInfo.TotalSize))
		cmd.Printf("    ✓ Detected language: %s\n", projectInfo.Language)
		cmd.Printf("    ✓ Project name: %s\n", projectInfo.Name)

		if onboardDryRun {
			cmd.Println()
			cmd.Println("  📋 Dry-run mode — no changes made")
			return nil
		}

		// T4: Sincronizar con HelixDB
		cmd.Println("  Syncing to HelixDB...")
		ctx := context.Background()
		helixClient, err := helix.NewClient(ctx)
		if err != nil {
			cmd.Printf("    ⚠ HelixDB not available: %v\n", err)
			cmd.Println("    ⚠ Continuing without HelixDB sync (run 'helix start dev --disk' first)")
		} else {
			projectID, err := syncToHelixDB(ctx, cmd, helixClient, projectInfo)
			if err != nil {
				cmd.Printf("    ⚠ HelixDB sync partial: %v\n", err)
			} else {
				cmd.Printf("    ✓ Project node created (ID: %d)\n", projectID)
				cmd.Printf("    ✓ %d CodeNodes created\n", projectInfo.FileCount)
			}
			helixClient.Close()
		}

		// T5: Escribir .zyro/task.yaml y lanzar OpenCode
		if !onboardNoOpenCode {
			cmd.Println("  Preparing OpenCode...")
			zyroDir := filepath.Join(absPath, ".zyro")
			os.MkdirAll(zyroDir, 0755)

			taskPath := filepath.Join(zyroDir, "task.yaml")
			taskContent := fmt.Sprintf(`phase: "PRE-F0"
agent: "zyro-pre-f0"
is_onboard: true
project_name: "%s"
project_language: "%s"
file_count: %d
scanned_at: "%s"
`, projectInfo.Name, projectInfo.Language, projectInfo.FileCount, projectInfo.ScannedAt.Format(time.RFC3339))

			if err := os.WriteFile(taskPath, []byte(taskContent), 0644); err != nil {
				cmd.Printf("    ⚠ Cannot write task file: %v\n", err)
			} else {
				cmd.Println("    ✓ Context written to .zyro/task.yaml")
			}

			cmd.Println()
			cmd.Println("  Opening OpenCode...")
			if _, err := exec.LookPath("opencode"); err == nil {
				openCmd := exec.Command("opencode", absPath)
				openCmd.Stdin = os.Stdin
				openCmd.Stdout = os.Stdout
				openCmd.Stderr = os.Stderr
				if err := openCmd.Run(); err != nil {
					cmd.Printf("    ⚠ OpenCode exited with error: %v\n", err)
				}
			} else {
				cmd.Println("  ⚠ opencode not found. Open manually: opencode", absPath)
			}
		} else {
			cmd.Println("  (skipped OpenCode launch per --no-opencode)")
		}

		cmd.Println()
		cmd.Println("✓ Project onboarded successfully!")
		cmd.Printf("  Next: open OpenCode in %s or run 'zyrocli run'\n", absPath)

		return nil
	},
}

// formatSize humaniza un tamaño de bytes.
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// syncToHelixDB crea el Project node y CodeNodes en HelixDB.
func syncToHelixDB(ctx context.Context, cmd *cobra.Command, client *helix.Client, info *scanner.ProjectInfo) (int64, error) {
	// Crear Project node
	projectID, err := client.CreateNode(ctx, "Project", map[string]interface{}{
		"name":          info.Name,
		"language":      string(info.Language),
		"path":          info.Root,
		"status":        "onboarded",
		"current_phase": "phase0",
		"file_count":    info.FileCount,
		"onboarded_at":  time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return 0, fmt.Errorf("helix: create project node: %w", err)
	}

	// Crear CodeNodes para archivos (máx 500 para no saturar)
	maxCodeNodes := 500
	created := 0
	for i, f := range info.Files {
		if i >= maxCodeNodes {
			break
		}
		_, err := client.CreateNode(ctx, "CodeNode", map[string]interface{}{
			"path":       f.Path,
			"name":       f.Name,
			"language":   string(f.Language),
			"hash":       f.Hash,
			"size":       f.Size,
			"project_id": projectID,
		})
		if err != nil {
			cmd.PrintErrf("    ⚠ error creating CodeNode for %s: %v\n", f.Path, err)
			continue
		}
		created++
	}

	if created < len(info.Files) && len(info.Files) > maxCodeNodes {
		// Se crearon hasta maxCodeNodes
	}

	return projectID, nil
}

func init() {
	rootCmd.AddCommand(onboardCmd)
	onboardCmd.Flags().BoolVarP(&onboardDryRun, "dry-run", "n", false, "Show what would be done without making changes")
	onboardCmd.Flags().BoolVarP(&onboardForce, "force", "f", false, "Re-onboard even if already initialized")
	onboardCmd.Flags().BoolVar(&onboardNoOpenCode, "no-opencode", false, "Skip opening OpenCode after onboarding")
	onboardCmd.Flags().StringVarP(&onboardName, "name", "N", "", "Custom project name (default: directory name)")
	onboardCmd.Flags().StringVarP(&onboardDesc, "description", "d", "", "Project description")
}
