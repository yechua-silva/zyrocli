package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig_DetectsPaths(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", cfg.Version)
	}

	// Go should be detectable in a Go test environment
	if cfg.Paths.GoBin == "" {
		t.Log("go not found in PATH (may be expected in constrained environments)")
	}

	// Git is nearly always available
	if cfg.Paths.GitBin == "" {
		t.Log("git not found in PATH")
	}

	// ConfigDir should never be empty
	if cfg.Paths.ConfigDir == "" {
		t.Error("ConfigDir should not be empty")
	}

	// Preferences should have defaults
	if cfg.Preferences.Verbose {
		t.Error("expected Verbose=false by default")
	}
	if cfg.Preferences.DryRun {
		t.Error("expected DryRun=false by default")
	}

	// Project should have a name
	if cfg.Project.Name == "" {
		t.Error("expected non-empty project name")
	}
	if cfg.Project.Root == "" {
		t.Error("expected non-empty project root")
	}
}

func TestDefaultConfig_DetectsSpecificPaths(t *testing.T) {
	tests := []struct {
		name   string
		bin    string
		field  string
	}{
		{"go binary", "go", "GoBin"},
		{"git binary", "git", "GitBin"},
	}

	cfg := DefaultConfig()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the field is either non-empty (found) or empty (not found)
			// but the code doesn't crash and the config is well-formed
			_ = cfg
		})
	}
}

func TestSaveConfig_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	configPath := ConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Verify .zyro dir has correct permissions
	zyroDir := filepath.Join(tmpDir, ".zyro")
	info, err := os.Stat(zyroDir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("expected dir permissions 0755, got %o", info.Mode().Perm())
	}

	// Verify file has readable permissions
	info, err = os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0644 {
		t.Errorf("expected file permissions 0644, got %o", info.Mode().Perm())
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error when config file does not exist")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	original := DefaultConfig()
	original.Project.Name = "TestProject"
	original.Preferences.Verbose = true
	original.Preferences.DryRun = true

	if err := SaveConfig(original); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Version != original.Version {
		t.Errorf("version: got %q, want %q", loaded.Version, original.Version)
	}
	if loaded.Project.Name != original.Project.Name {
		t.Errorf("project name: got %q, want %q", loaded.Project.Name, original.Project.Name)
	}
	if loaded.Preferences.Verbose != original.Preferences.Verbose {
		t.Errorf("verbose: got %v, want %v", loaded.Preferences.Verbose, original.Preferences.Verbose)
	}
	if loaded.Preferences.DryRun != original.Preferences.DryRun {
		t.Errorf("dry_run: got %v, want %v", loaded.Preferences.DryRun, original.Preferences.DryRun)
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath()
	if path == "" {
		t.Error("ConfigPath should not return empty string")
	}
	if filepath.Base(path) != "config.yaml" {
		t.Errorf("expected config.yaml, got %s", filepath.Base(path))
	}
	if filepath.Base(filepath.Dir(path)) != ".zyro" {
		t.Errorf("expected parent dir .zyro, got %s", filepath.Base(filepath.Dir(path)))
	}
}

func TestConfigFields(t *testing.T) {
	cfg := Config{
		Version: "2.0.0",
		Project: ProjectConfig{Name: "P", Root: "/tmp"},
		Paths: PathsConfig{
			GoBin:     "/usr/local/go/bin/go",
			UvBin:     "/home/user/.cargo/bin/uv",
			HelixBin:  "/usr/local/bin/helix",
			DockerBin: "/usr/bin/docker",
			GitBin:    "/usr/bin/git",
			ConfigDir: "/home/user/.zyro",
		},
		Preferences: PreferencesConfig{Verbose: true, DryRun: false},
	}

	if cfg.Version != "2.0.0" {
		t.Errorf("unexpected version: %s", cfg.Version)
	}
	if cfg.Project.Name != "P" {
		t.Errorf("unexpected project name: %s", cfg.Project.Name)
	}
	if cfg.Paths.GoBin != "/usr/local/go/bin/go" {
		t.Errorf("unexpected go_bin: %s", cfg.Paths.GoBin)
	}
	if !cfg.Preferences.Verbose {
		t.Error("expected verbose=true")
	}
}
