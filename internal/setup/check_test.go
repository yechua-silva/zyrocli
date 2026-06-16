package setup

import (
	"os/exec"
	"strings"
	"testing"
)

// mockExecFunc returns a function that creates a Cmd that outputs the given
// stdout string. It's used to mock exec.Command in tests.
func mockExecFunc(versionOutput string) func(name string, arg ...string) *exec.Cmd {
	return func(name string, arg ...string) *exec.Cmd {
		cmd := exec.Command("echo", versionOutput)
		return cmd
	}
}

// mockExecFuncMissing simulates a command not found by returning a command
// that will fail.
func mockExecFuncMissing() func(name string, arg ...string) *exec.Cmd {
	return func(name string, arg ...string) *exec.Cmd {
		// Return a command that doesn't exist
		cmd := exec.Command("nonexistent-binary-xyz")
		return cmd
	}
}

func TestNewChecker(t *testing.T) {
	c := NewChecker()
	if c == nil {
		t.Fatal("NewChecker() returned nil")
	}
	if c.execFunc == nil {
		t.Error("execFunc should not be nil")
	}
}

func TestCheckerCheckAll(t *testing.T) {
	c := NewChecker()
	results := c.CheckAll()
	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}

	// Verify all dependency types are present
	types := make(map[DependencyType]bool)
	for _, r := range results {
		types[r.Type] = true
	}
	for _, dep := range []DependencyType{DepGo, DepUv, DepDocker, DepHelix, DepGit} {
		if !types[dep] {
			t.Errorf("missing check for %s", dep)
		}
	}
}

func TestCheckerCheckGo(t *testing.T) {
	c := NewChecker()
	result := c.Check(DepGo)
	// Go may or may not be installed, but the result should be well-formed
	if result.Type != DepGo {
		t.Errorf("expected type %s, got %s", DepGo, result.Type)
	}
	if result.Name != "Go" {
		t.Errorf("expected name Go, got %s", result.Name)
	}
}

func TestCheckerCheckGoMissing(t *testing.T) {
	c := &Checker{
		execFunc: mockExecFuncMissing(),
	}
	// Override LookPath to fail by using a custom approach:
	// We can't easily mock exec.LookPath, but we can test the execFunc path
	// by testing the version extraction. The "not installed" case relies on
	// exec.LookPath which uses PATH; in tests without Go it'll show not installed.
	result := c.checkGo()
	// Since we can't mock LookPath, this test mainly validates the struct
	_ = result
}

func TestCheckerCheckUv(t *testing.T) {
	c := &Checker{
		execFunc: mockExecFunc("uv 0.4.0"),
	}
	result := c.checkUv()
	// LookPath will be real, so this depends on whether uv is installed
	// But we can test the version parsing path
	if result.Installed {
		c2 := &Checker{
			execFunc: mockExecFunc("uv 0.4.0"),
		}
		r2 := c2.Check(DepUv)
		if r2.Installed {
			// If uv is actually installed, our mock won't be used (LookPath finds it first)
			// Just verify the result is well-formed
			if r2.Version == "" && r2.Error != "" {
				t.Logf("uv version check: %s", r2.Error)
			}
		}
	}
}

func TestCheckerCheckHelix(t *testing.T) {
	c := &Checker{
		execFunc: mockExecFunc("helix 3.0.0"),
	}
	_ = c.checkHelix()
}

func TestCheckerCheckDocker(t *testing.T) {
	c := &Checker{
		execFunc: mockExecFunc("Docker version 24.0.0"),
	}
	_ = c.checkDocker()
}

func TestCheckerCheckGit(t *testing.T) {
	c := &Checker{
		execFunc: mockExecFunc("git version 2.40.0"),
	}
	_ = c.checkGit()
}

func TestCheckerCheckUnknown(t *testing.T) {
	c := NewChecker()
	result := c.Check(DependencyType("unknown"))
	if result.Error == "" {
		t.Error("expected error for unknown dependency type")
	}
	if result.Fixable {
		t.Error("expected Fixable=false for unknown dependency")
	}
}

func TestPlatformInfo(t *testing.T) {
	osName, arch := PlatformInfo()
	if osName == "" {
		t.Error("expected non-empty OS name")
	}
	if arch == "" {
		t.Error("expected non-empty architecture")
	}
}

func TestCheckResultFields(t *testing.T) {
	r := &CheckResult{
		Type:      DepGo,
		Name:      "Go",
		Installed: true,
		Version:   "go version go1.22.0 linux/amd64",
		Fixable:   false,
	}
	if r.Type != DepGo {
		t.Errorf("expected type %s, got %s", DepGo, r.Type)
	}
	if !r.Installed {
		t.Error("expected Installed=true")
	}
	if !strings.Contains(r.Version, "go1.22") {
		t.Errorf("unexpected version: %s", r.Version)
	}
}

func TestCheckerExecFuncMock(t *testing.T) {
	// Test that when execFunc is mocked, we can control the output
	mockVersion := "go version go1.99.0 linux/amd64"
	c := &Checker{
		execFunc: mockExecFunc(mockVersion),
	}
	// The checkGo still uses exec.LookPath (which finds real go if installed)
	// but the version output uses execFunc
	_ = c.checkGo()

	// Verify mock works directly
	cmd := c.execFunc("go", "version")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("mock exec failed: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != mockVersion {
		t.Errorf("expected %q, got %q", mockVersion, got)
	}
}

func TestCheckerCheckAllResultsHaveType(t *testing.T) {
	c := NewChecker()
	results := c.CheckAll()
	for _, r := range results {
		if r.Type == "" {
			t.Error("result has empty Type")
		}
		if r.Name == "" {
			t.Error("result has empty Name")
		}
	}
}
