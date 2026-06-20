package opencode

import (
	"os"
	"testing"
)

func TestWriteGlobalConfig_createsValidFile(t *testing.T) {
	// Use a temp dir so we don't pollute real config
	orig := OpenCodeConfigPath
	OpenCodeConfigPath = "~/.config/opencode-test/opencode.json"
	t.Cleanup(func() {
		OpenCodeConfigPath = orig
		os.RemoveAll(expandHome("~/.config/opencode-test"))
	})

	cfg := &Config{
		Schema: "https://opencode.ai/config.json",
		Agent: map[string]Agent{
			"test-agent": {
				Mode:        "subagent",
				Description: "test",
				Hidden:      true,
			},
		},
	}

	path, err := WriteGlobalConfig(cfg)
	if err != nil {
		t.Fatalf("WriteGlobalConfig failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("config file not created at %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("config file is empty")
	}
}

func TestIsInstalled_detectsMissing(t *testing.T) {
	// Temporarily point to a non-existent path
	origConfig := OpenCodeConfigPath
	origSkills := SkillsDir
	OpenCodeConfigPath = "~/.config/opencode-missing/opencode.json"
	SkillsDir = "~/.config/opencode-missing/skills"
	t.Cleanup(func() {
		OpenCodeConfigPath = origConfig
		SkillsDir = origSkills
	})

	if IsInstalled() {
		t.Error("expected IsInstalled()=false for missing config")
	}
}
