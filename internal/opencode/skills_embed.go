// DEPRECATED: Skills now load via claude-bridge as .md files with YAML frontmatter.
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

//go:embed skills/zyro-orchestrator/SKILL.md
//go:embed skills/zyro-phase-0-patterns/SKILL.md
//go:embed skills/zyro-phase-0-libraries/SKILL.md
//go:embed skills/zyro-skills-find/SKILL.md
//go:embed skills/zyro-skills-audit/SKILL.md
//go:embed skills/zyro-skills-apply/SKILL.md
//go:embed skills/zyro-sdd-apply/SKILL.md
//go:embed skills/zyro-sdd-verify/SKILL.md
//go:embed skills/zyro-sdd-explore/SKILL.md
//go:embed skills/zyro-sdd-propose/SKILL.md
//go:embed skills/zyro-sdd-spec/SKILL.md
//go:embed skills/zyro-sdd-design/SKILL.md
//go:embed skills/zyro-sdd-tasks/SKILL.md
//go:embed skills/zyro-sdd-archive/SKILL.md
//go:embed skills/zyro-pre-f0/SKILL.md
//go:embed skills/to-issues/SKILL.md
var skillsFS embed.FS

// SkillsDir is the global skills directory.
var SkillsDir = "~/.config/opencode/skills"


// WriteAllSkills writes all embedded skills to the global skills directory.
// The directory structure is: skills/<skill-name>/SKILL.md
// Returns the number of skills written and any error.
func WriteAllSkills() (int, error) {
	baseDir := expandHome(SkillsDir)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return 0, fmt.Errorf("opencode: create skills dir %s: %w", baseDir, err)
	}

	var count int

	err := fs.WalkDir(skillsFS, "skills", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "SKILL.md" {
			return nil
		}

		// path = "skills/<skill-name>/SKILL.md"
		// Extract skill name from the parent directory
		parts := strings.SplitN(path, "/", 3)
		if len(parts) < 2 {
			return nil
		}
		skillName := parts[1]

		content, err := skillsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("opencode: read embedded %s: %w", path, err)
		}

		skillDir := filepath.Join(baseDir, skillName)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return fmt.Errorf("opencode: create skill dir %s: %w", skillDir, err)
		}

		outPath := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(outPath, content, 0644); err != nil {
			return fmt.Errorf("opencode: write %s: %w", outPath, err)
		}

		count++
		return nil
	})

	return count, err
}

// IsInstalled checks if ZyroCLI global config and skills are installed.
func IsInstalled() bool {
	configPath := expandHome(OpenCodeConfigPath)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false
	}

	skillsDir := expandHome(SkillsDir)
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return false
	}

	return true
}
