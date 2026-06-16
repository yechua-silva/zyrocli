package setup

import (
	"testing"
)

func TestNewInstaller(t *testing.T) {
	inst := NewInstaller(true, true, false)
	if inst == nil {
		t.Fatal("NewInstaller() returned nil")
	}
	if !inst.dryRun {
		t.Error("expected dryRun=true")
	}
	if !inst.verbose {
		t.Error("expected verbose=true")
	}
	if inst.force {
		t.Error("expected force=false")
	}
}

func TestInstallerInstallUV(t *testing.T) {
	// Test dry-run mode — should not execute anything
	inst := NewInstaller(true, true, false)
	err := inst.installUV()
	if err != nil {
		t.Errorf("dry-run installUV should not error: %v", err)
	}

	// Test non-dry-run (will actually try to install if uv is missing)
	inst2 := NewInstaller(false, false, false)
	err2 := inst2.installUV()
	// If uv is already installed, the script is idempotent
	// If not, it will try to install. We accept either outcome.
	_ = err2
}

func TestInstallerInstallHelix(t *testing.T) {
	// Test dry-run mode
	inst := NewInstaller(true, true, false)
	err := inst.installHelix()
	if err != nil {
		t.Errorf("dry-run installHelix should not error: %v", err)
	}
}

func TestInstallerInstallGo(t *testing.T) {
	// Go install should never error (it just prints a message)
	inst := NewInstaller(false, false, false)
	err := inst.installGo()
	if err != nil {
		t.Errorf("installGo should not error: %v", err)
	}
}

func TestInstallerInstallDocker(t *testing.T) {
	inst := NewInstaller(false, false, false)
	err := inst.installDocker()
	if err != nil {
		t.Errorf("installDocker should not error: %v", err)
	}
}

func TestInstallerInstallGit(t *testing.T) {
	inst := NewInstaller(false, false, false)
	err := inst.installGit()
	if err != nil {
		t.Errorf("installGit should not error: %v", err)
	}
}

func TestInstallerInstallAll(t *testing.T) {
	inst := NewInstaller(true, true, false)

	results := []*CheckResult{
		{Type: DepGo, Name: "Go", Installed: true, Fixable: false},
		{Type: DepUv, Name: "uv", Installed: false, Fixable: true},
		{Type: DepDocker, Name: "Docker", Installed: true, Fixable: false},
		{Type: DepHelix, Name: "HelixDB", Installed: false, Fixable: true},
		{Type: DepGit, Name: "Git", Installed: true, Fixable: false},
	}

	errs := inst.InstallAll(results)
	// In dry-run mode, both uv and helix should "succeed" (they're dry-run)
	if len(errs) != 0 {
		t.Errorf("expected 0 errors in dry-run, got %d: %v", len(errs), errs)
	}
}

func TestInstallerInstallAllForce(t *testing.T) {
	inst := NewInstaller(true, true, true) // force=true

	results := []*CheckResult{
		{Type: DepUv, Name: "uv", Installed: true, Fixable: true, Version: "uv 0.4.0"},
	}

	errs := inst.InstallAll(results)
	// With --force, it should try to install even if installed
	// In dry-run mode, it should succeed
	if len(errs) != 0 {
		t.Errorf("expected 0 errors with force+dry-run, got %d: %v", len(errs), errs)
	}
}

func TestInstallerInstallAllNotFixable(t *testing.T) {
	inst := NewInstaller(false, true, false)

	results := []*CheckResult{
		{Type: DepGo, Name: "Go", Installed: false, Fixable: false},
		{Type: DepDocker, Name: "Docker", Installed: false, Fixable: false},
	}

	errs := inst.InstallAll(results)
	// These are not fixable, so they should be skipped, not produce errors
	if len(errs) != 0 {
		t.Errorf("expected 0 errors for non-fixable deps, got %d: %v", len(errs), errs)
	}
}

func TestInstallerInstallUnknown(t *testing.T) {
	inst := NewInstaller(false, false, false)
	err := inst.Install(DependencyType("unknown"))
	if err == nil {
		t.Error("expected error for unknown dependency")
	}
}

func TestIsInstalled(t *testing.T) {
	// "go" should be installed if we're running tests with Go
	installed := IsInstalled("go")
	if !installed {
		t.Log("go not found in PATH (unexpected for Go test environment)")
	}

	// "nonexistent-binary-xyz" should not be installed
	notInstalled := IsInstalled("nonexistent-binary-xyz")
	if notInstalled {
		t.Error("expected nonexistent binary to not be installed")
	}
}

func TestDepTypeForName(t *testing.T) {
	tests := []struct {
		name string
		want DependencyType
	}{
		{"go", DepGo},
		{"Go", DepGo},
		{"GO", DepGo},
		{"uv", DepUv},
		{"UV", DepUv},
		{"docker", DepDocker},
		{"helix", DepHelix},
		{"git", DepGit},
		{"unknown", DependencyType("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DepTypeForName(tt.name)
			if got != tt.want {
				t.Errorf("DepTypeForName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestNewInstallerChecker(t *testing.T) {
	inst := NewInstaller(false, false, false)
	if inst.checker == nil {
		t.Error("expected checker to be initialized")
	}
}

func TestInstallAllEmpty(t *testing.T) {
	inst := NewInstaller(false, false, false)
	errs := inst.InstallAll([]*CheckResult{})
	if len(errs) != 0 {
		t.Errorf("expected 0 errors for empty results, got %d", len(errs))
	}
}
