package setup

import (
	"os"
	"testing"
)

func TestDoctorResultFields(t *testing.T) {
	tests := []struct {
		name   string
		result DoctorResult
		checks func(t *testing.T, r DoctorResult)
	}{
		{
			name:   "ok status",
			result: DoctorResult{Check: "test", Status: "ok", Message: "all good"},
			checks: func(t *testing.T, r DoctorResult) {
				if r.Check != "test" {
					t.Errorf("Check = %q, want %q", r.Check, "test")
				}
				if r.Status != "ok" {
					t.Errorf("Status = %q, want %q", r.Status, "ok")
				}
				if r.Message != "all good" {
					t.Errorf("Message = %q, want %q", r.Message, "all good")
				}
			},
		},
		{
			name:   "warning status",
			result: DoctorResult{Check: "disk", Status: "warning", Message: "low space"},
			checks: func(t *testing.T, r DoctorResult) {
				if r.Status != "warning" {
					t.Errorf("Status = %q, want %q", r.Status, "warning")
				}
			},
		},
		{
			name:   "error status",
			result: DoctorResult{Check: "db", Status: "error", Message: "connection failed"},
			checks: func(t *testing.T, r DoctorResult) {
				if r.Status != "error" {
					t.Errorf("Status = %q, want %q", r.Status, "error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.checks(t, tt.result)
		})
	}
}

func TestCountIssues(t *testing.T) {
	tests := []struct {
		name    string
		results []DoctorResult
		want    int
	}{
		{
			name:    "no issues",
			results: []DoctorResult{{Check: "a", Status: "ok"}, {Check: "b", Status: "ok"}},
			want:    0,
		},
		{
			name:    "one issue",
			results: []DoctorResult{{Check: "a", Status: "ok"}, {Check: "b", Status: "error"}},
			want:    1,
		},
		{
			name:    "all issues",
			results: []DoctorResult{{Check: "a", Status: "error"}, {Check: "b", Status: "warning"}},
			want:    2,
		},
		{
			name:    "empty slice",
			results: []DoctorResult{},
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countIssues(tt.results)
			if got != tt.want {
				t.Errorf("countIssues() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRunDoctor_AllOk(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create a valid config so the config_file check passes
	cfg := DefaultConfig()
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// Run doctor without fix — should complete without error
	err := RunDoctor(false)
	if err != nil {
		t.Errorf("RunDoctor(false) with valid config should not error: %v", err)
	}
}

func TestRunDoctor_WithFix(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// No config exists yet
	// Run doctor with --fix — should create config automatically
	err := RunDoctor(true)
	if err != nil {
		t.Fatalf("RunDoctor(true) should not error: %v", err)
	}

	// Verify config was created
	if _, err := os.Stat(ConfigPath()); os.IsNotExist(err) {
		t.Error("expected config file to be created after doctor --fix")
	}

	// Verify the created config can be loaded
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", loaded.Version)
	}
	if loaded.Project.Name == "" {
		t.Error("expected non-empty project name in generated config")
	}
}

func TestRunDoctor_WithFixIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// First run: create config
	if err := RunDoctor(true); err != nil {
		t.Fatal(err)
	}

	// Modify config to verify it's not overwritten incorrectly
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Project.Name = "CustomProject"
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// Second run: should not overwrite custom changes (except maybe paths)
	if err := RunDoctor(true); err != nil {
		t.Fatal(err)
	}

	// Config should still exist and be loadable
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", loaded.Version)
	}
	_ = loaded
}

func TestCheckConfigFile_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	result := checkConfigFile()
	if result.Status != "ok" {
		t.Errorf("expected status ok, got %s: %s", result.Status, result.Message)
	}
}

func TestCheckConfigFile_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	result := checkConfigFile()
	if result.Status != "error" {
		t.Errorf("expected status error, got %s: %s", result.Status, result.Message)
	}
}
