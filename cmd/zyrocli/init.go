package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yechua-silva/zyrocli/internal/db/helix"
	"github.com/yechua-silva/zyrocli/internal/handoff"
	"github.com/yechua-silva/zyrocli/internal/opencode"
	"github.com/yechua-silva/zyrocli/internal/scaffold"
	"github.com/spf13/cobra"
)

var (
	dryRun       bool
	useScaffold  bool
	noOpenCode   bool
)

var initCmd = &cobra.Command{
	Use:   "init <handoff.yaml>",
	Short: "Initialize project from handoff and open in OpenCode",
	Long: `Parse a handoff.yaml v2.0 contract, create the project folder,
register it in HelixDB, and launch OpenCode.

By default creates an agnostic project structure (language-independent).
Use --scaffold for a Go-specific template with main.go, AGENT.md, etc.

The global ZyroCLI ecosystem (skills, agents, MCP) must be installed first:
  zyrocli install`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		// ── Parse and validate ──────────────────────────────────────
		payload, err := handoff.Parse(path)
		if err != nil {
			return fmt.Errorf("init: %w", err)
		}
		if err := handoff.Validate(payload); err != nil {
			return fmt.Errorf("init: validation failed:\n%v", err)
		}

		// ── Check ecosystem is installed ───────────────────────────
		if !opencode.IsInstalled() {
			cmd.PrintErrln("⚠ ZyroCLI ecosystem not fully configured. Run for best experience:")
			cmd.PrintErrln("    zyrocli install")
		}

		projectName := normalizeDirName(payload.Project.Name)
		projectDir := projectName

		// ── Dry-run mode: just validate ─────────────────────────────
		if dryRun {
			cmd.Printf("✓ handoff valid for project: %s (%s)\n", payload.Project.Name, payload.Project.Language)
			cmd.Printf("  Target directory: %s/\n", projectDir)
			return nil
		}

		// ── Error if directory exists ───────────────────────────────
		if _, err := os.Stat(projectDir); err == nil {
			return fmt.Errorf("init: directory %q already exists", projectDir)
		}

		// ── Create project directory ────────────────────────────────
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			return fmt.Errorf("init: create dir: %w", err)
		}

		// ── Read raw handoff for copying ───────────────────────────
		rawBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("init: reading handoff: %w", err)
		}

		// ── Create agnostic project structure ──────────────────────
		if err := createAgnosticStructure(projectDir, payload, rawBytes); err != nil {
			return fmt.Errorf("init: %w", err)
		}
		cmd.Printf("✓ Project structure created at %s/\n", projectDir)

		// ── Auto-start HelixDB ──────────────────────────────────────
		helixClient, helixErr := helix.NewClient(context.Background())
		if helixErr == nil {
			if err := helixClient.EnsureStarted(context.Background()); err != nil {
				cmd.PrintErrln("⚠ HelixDB not available (project will work, but MCP tools need it)")
			} else {
				if projectNode, err := helixClient.CreateNode(context.Background(), "Project", map[string]any{
					"name":        payload.Project.Name,
					"repo":        payload.Project.Repository,
					"problem":     payload.ValidatedIdea.Problem,
					"current_phase": "phase0",
				}); err == nil {
					cmd.Printf("  HelixDB project node: %d\n", projectNode)
				}
				_ = helixClient.Close()
			}
		}

		// ── Optional: scaffold Go template ──────────────────────────
		if useScaffold {
			scaffoldCfg := scaffold.Config{
				ProjectName:     payload.Project.Name,
				Language:        payload.Project.Language,
				Module:          payload.Project.Repository,
				Problem:         payload.ValidatedIdea.Problem,
				SuccessCriteria: payload.UserStory.Acceptance,
				ScaffoldDir:     projectDir,
				LaunchOpenCode:  false,
				RawHandoff:      string(rawBytes),
				Version:         payload.Version,
				Source:          payload.Source.System,
			}

			// Target dir already exists, so scaffold into it
			result, err := scaffold.Run(scaffoldCfg)
			if err != nil {
				cmd.PrintErrln("⚠ Scaffold warnings (project structure already created):", err)
			} else {
				cmd.Printf("  Go scaffold: %d files\n", result.FilesCreated)
			}
		}

		// ── Check opencode availability ─────────────────────────────
		if noOpenCode {
			cmd.Printf("  Project created at %s/ (--no-opencode)\n", projectDir)
			return nil
		}

		if _, err := exec.LookPath("opencode"); err != nil {
			cmd.PrintErrln("⚠ opencode not found in PATH. Install it separately.")
			cmd.Printf("  Project created at %s/\n", projectDir)
			return nil
		}

		// ── Launch OpenCode ─────────────────────────────────────────
		cmd.Printf("  Opening OpenCode in %s/ ...\n", projectDir)
		openCmd := exec.Command("opencode", projectDir)
		openCmd.Stdin = os.Stdin
		openCmd.Stdout = os.Stdout
		openCmd.Stderr = os.Stderr
		_ = openCmd.Run()
		cmd.Println("OpenCode session ended. Happy coding!")

		return nil
	},
}

// createAgnosticStructure writes a minimal language-agnostic project skeleton.
func createAgnosticStructure(dir string, payload *handoff.Payload, rawHandoff []byte) error {
	// .gitignore
	gitignore := `# Dependencies
node_modules/
vendor/
.venv/
__pycache__/

# Build
dist/
build/
*.exe
*.dll
*.so
*.dylib

# Env
.env
.env.local

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db
`
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignore), 0644); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}

	// handoff.yaml — copy from source
	if len(rawHandoff) > 0 {
		if err := os.WriteFile(filepath.Join(dir, "handoff.yaml"), rawHandoff, 0644); err != nil {
			return fmt.Errorf("write handoff.yaml: %w", err)
		}
	}

	// README.md
	readme := fmt.Sprintf("# %s\n\n%s\n\n## Stack\n\n- **Language**: %s\n- **Repository**: %s\n\n## Getting Started\n\n```bash\n# Install dependencies (language-specific)\n# Write your code\n# Run tests\n```\n",
		payload.Project.Name,
		payload.ValidatedIdea.Problem,
		payload.Project.Language,
		payload.Project.Repository,
	)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0644); err != nil {
		return fmt.Errorf("write README.md: %w", err)
	}

	// skills/ — empty, skills get installed here by Fase 0
	if err := os.MkdirAll(filepath.Join(dir, "skills"), 0755); err != nil {
		return fmt.Errorf("create skills dir: %w", err)
	}

	// Placeholder for skills
	skillsReadme := "# Skills\n\nSkills for this project are discovered and installed by Fase 0.\nRun `zyrocli init` and Fase 0 will populate this directory.\n"
	if err := os.WriteFile(filepath.Join(dir, "skills", "README.md"), []byte(skillsReadme), 0644); err != nil {
		return fmt.Errorf("write skills/README.md: %w", err)
	}

	// docs/
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0755); err != nil {
		return fmt.Errorf("create docs dir: %w", err)
	}

	// src/ — language-agnostic source directory
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0755); err != nil {
		return fmt.Errorf("create src dir: %w", err)
	}

	return nil
}

// normalizeDirName produces a safe directory name from a project name.
func normalizeDirName(name string) string {
	name = strings.ToLower(name)
	name = strings.Join(strings.Fields(name), "-")
	var result strings.Builder
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result.WriteByte(c)
		}
	}
	return strings.Trim(result.String(), "-_")
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "only parse and validate, do not create anything")
	initCmd.Flags().BoolVarP(&useScaffold, "scaffold", "s", false, "generate Go-specific template files too")
	initCmd.Flags().BoolVarP(&noOpenCode, "no-opencode", "", false, "create project but do not open OpenCode")
}
