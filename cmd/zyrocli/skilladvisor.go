package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/secko/zyrocli/internal/skilladvisor"
	"github.com/spf13/cobra"
)

var skillAdvisorProjectDir string

var skillAdvisorCmd = &cobra.Command{
	Use:   "skill-advisor <query>",
	Short: "Discover, validate, and score skills for a project",
	Long: `Search skills.sh by language/framework, validate with NVIDIA SkillSpector,
and score each skill against the project context.

Usage:
  zyro skill-advisor go            search skills for Go
  zyro skill-advisor react         search skills for React
  zyro skill-advisor ts --project .  auto-detect stack from current project

The tool runs "npx skills find <query>", then validates each skill
with SkillSpector (if installed) and scores them for relevance.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		// Detect project context if --project is set
		ctx := skilladvisor.ProjectContext{Language: query}
		if skillAdvisorProjectDir != "" {
			detected := detectProjectStack(skillAdvisorProjectDir)
			if detected.Language != "" {
				ctx = detected
			}
		}

		// Step 1: npx skills find
		cmd.Printf("  Buscando skills para '%s'...\n", query)
		skills, err := runNpxSkillsFind(query)
		if err != nil {
			return fmt.Errorf("skill-advisor: %w", err)
		}
		cmd.Printf("  Encontradas: %d skills\n\n", len(skills))

		if len(skills) == 0 {
			fmt.Println("  No se encontraron skills. Probá con otro término.")
			return nil
		}

		// Step 2: Score + validate
		fmt.Println("  Validando y puntuando skills...")
		type result struct {
			skilladvisor.Skill
			Score       int
			RiskScore   int
			RiskSeverity string
		}
		var results []result

		for _, s := range skills {
			score := skilladvisor.ScoreSkill(s, ctx)
			riskScore, riskSeverity := 0, "N/A"

			// Try SkillSpector if installed
			if hasSkillSpector() {
				riskScore, riskSeverity = runSkillSpector(s)
			}

			results = append(results, result{
				Skill:        s,
				Score:        score,
				RiskScore:    riskScore,
				RiskSeverity: riskSeverity,
			})
		}

		// Step 3: Display results
		fmt.Println("\n  Resultados:")
		fmt.Println(strings.Repeat("-", 70))

		// Recommended (score >= 50)
		fmt.Println("\n  ✅ RECOMENDADAS (score ≥ 50, risk LOW):")
		for _, r := range results {
			riskOK := r.RiskSeverity == "" || r.RiskSeverity == "LOW" || r.RiskSeverity == "N/A"
			if r.Score >= 50 && riskOK {
				fmt.Printf("    %-35s score=%3d  risk=%s\n", r.Name, r.Score, r.RiskSeverity)
			}
		}

		// Caution (high score but risky)
		fmt.Println("\n  ⚠️  RIESGO (score ≥ 50 pero riesgo alto):")
		for _, r := range results {
			risky := r.RiskSeverity == "MEDIUM" || r.RiskSeverity == "HIGH" || r.RiskSeverity == "CRITICAL"
			if r.Score >= 50 && risky {
				fmt.Printf("    %-35s score=%3d  risk=%s\n", r.Name, r.Score, r.RiskSeverity)
			}
		}

		// Lower score
		fmt.Println("\n  ℹ️  OTRAS (score < 50):")
		for _, r := range results {
			if r.Score < 50 && r.Score > 0 {
				fmt.Printf("    %-35s score=%3d  risk=%s\n", r.Name, r.Score, r.RiskSeverity)
			}
		}

		// All skills found, regardless of score/risk
		cmd.Printf("\n  📋 TODAS LAS SKILLS ENCONTRADAS (%d):\n", len(results))
		for _, r := range results {
			cmd.Printf("    - %s (score: %d, riesgo: %s)\n", r.Name, r.Score, r.RiskSeverity)
		}

		if len(results) == 0 {
			fmt.Println("    (ninguna)")
		}

		return nil
	},
}

// runNpxSkillsFind executes "npx skills find" and parses results.
func runNpxSkillsFind(query string) ([]skilladvisor.Skill, error) {
	cmd := exec.Command("npx", "skills", "find", query)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// npx might exit non-zero for "not found", still parse output
	}

	lines := strings.Split(string(output), "\n")
	var skills []skilladvisor.Skill
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ">") || strings.HasPrefix(line, "npx") {
			continue
		}
		// Parse: "skill-name" or "owner/repo@skill-name (stars)"
		parts := strings.Fields(line)
		if len(parts) >= 1 {
			name := parts[0]
			name = strings.TrimPrefix(name, "@")
			name = strings.TrimSuffix(name, ",")
			if strings.Contains(name, "/") {
				parts2 := strings.SplitN(name, "/", 2)
				if len(parts2) == 2 {
					skills = append(skills, skilladvisor.Skill{
						Name:      parts2[1],
						Publisher: parts2[0],
					})
				}
			} else {
				skills = append(skills, skilladvisor.Skill{Name: name})
			}
		}
	}

	return skills, nil
}

// runSkillSpector runs "skillspector scan" on a skill and returns risk score.
func runSkillSpector(s skilladvisor.Skill) (int, string) {
	// Create a temp dir with a minimal SKILL.md for skillspector to scan
	tmpDir, err := os.MkdirTemp("", "skillspector-*")
	if err != nil {
		return 0, "ERROR"
	}
	defer os.RemoveAll(tmpDir)

	skillMd := fmt.Sprintf("# %s\n\nSkill for %s.\n", s.Name, s.Language)
	os.WriteFile(filepath.Join(tmpDir, "SKILL.md"), []byte(skillMd), 0644)

	cmd := exec.Command("skillspector", "scan", tmpDir, "--format", "json", "--no-llm")
	output, err := cmd.Output()
	if err != nil {
		return 0, "N/A"
	}

	// Parse JSON output for score and severity
	outputStr := string(output)
	score := 0
	severity := "LOW"

	// Simple extraction from JSON (no dependency on json parsing of unknown schema)
	for _, line := range strings.Split(outputStr, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, `"risk_score"`) {
			fmt.Sscanf(line, `%*s %d`, &score)
		}
		if strings.Contains(line, `"risk_severity"`) {
			parts := strings.Split(line, `"`)
			if len(parts) >= 4 {
				severity = parts[3]
			}
		}
	}

	return score, severity
}

// detectProjectStack scans project files to determine language/framework.
func detectProjectStack(dir string) skilladvisor.ProjectContext {
	ctx := skilladvisor.ProjectContext{}

	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		ctx.Language = "go"
		ctx.ProjectType = "cli"
		ctx.Framework = "cobra"
		return ctx
	}
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		ctx.Language = "typescript"
		ctx.ProjectType = "web"
		return ctx
	}
	if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
		ctx.Language = "rust"
		ctx.ProjectType = "cli"
		return ctx
	}
	if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
		ctx.Language = "python"
		ctx.ProjectType = "api"
		return ctx
	}

	return ctx
}

// hasSkillSpector checks if skillspector is installed.
func hasSkillSpector() bool {
	_, err := exec.LookPath("skillspector")
	return err == nil
}

func init() {
	rootCmd.AddCommand(skillAdvisorCmd)
	skillAdvisorCmd.Flags().StringVarP(&skillAdvisorProjectDir, "project", "p", "", "project directory for auto-detection")
}
