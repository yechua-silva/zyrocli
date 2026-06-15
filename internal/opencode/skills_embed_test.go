package opencode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAllSkills_createsFiles(t *testing.T) {
	orig := SkillsDir
	SkillsDir = "~/.config/opencode-test-skills/skills"
	t.Cleanup(func() {
		SkillsDir = orig
		os.RemoveAll(expandHome("~/.config/opencode-test-skills"))
	})

	count, err := WriteAllSkills()
	if err != nil {
		t.Fatalf("WriteAllSkills failed: %v", err)
	}
	if count == 0 {
		t.Fatal("expected at least 1 skill, got 0")
	}

	// Verify a skill was created
	base := expandHome(SkillsDir)
	entries, _ := os.ReadDir(base)
	if len(entries) == 0 {
		t.Fatal("no skill directories created")
	}
}

func TestWriteAllSkills_removesDeprecated(t *testing.T) {
	orig := SkillsDir
	SkillsDir = "~/.config/opencode-test-dep/skills"
	t.Cleanup(func() {
		SkillsDir = orig
		os.RemoveAll(expandHome("~/.config/opencode-test-dep"))
	})

	// Create a deprecated skill directory
	base := expandHome(SkillsDir)
	oldDir := filepath.Join(base, "sdd-apply")
	os.MkdirAll(oldDir, 0755)
	os.WriteFile(filepath.Join(oldDir, "SKILL.md"), []byte("old"), 0644)

	// Should be removed by WriteAllSkills
	_, err := WriteAllSkills()
	if err != nil {
		t.Fatalf("WriteAllSkills failed: %v", err)
	}

	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Error("deprecated sdd-apply directory was not removed")
	}
}

func TestWriteMCPTools_createsFiles(t *testing.T) {
	orig := ZyroDir
	ZyroDir = "~/.config/zyrocli-test"
	t.Cleanup(func() {
		ZyroDir = orig
		os.RemoveAll(expandHome("~/.config/zyrocli-test"))
	})

	mcpDir, err := WriteMCPTools()
	if err != nil {
		t.Fatalf("WriteMCPTools failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(mcpDir, "runner.py")); os.IsNotExist(err) {
		t.Error("runner.py not created")
	}
	if _, err := os.Stat(filepath.Join(mcpDir, "helix_write.py")); os.IsNotExist(err) {
		t.Error("helix_write.py not created")
	}
}
