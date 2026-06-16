package setup

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════════
//  Mocks
// ══════════════════════════════════════════════════════════════════════════

// mockChecker retorna resultados predefinidos sin tocar el sistema real.
type mockChecker struct {
	results []*CheckResult
}

func (m *mockChecker) CheckAll() []*CheckResult {
	return m.results
}

func (m *mockChecker) Check(dep DependencyType) *CheckResult {
	for _, r := range m.results {
		if r.Type == dep {
			return r
		}
	}
	return &CheckResult{
		Type:      dep,
		Name:      string(dep),
		Installed: false,
		Error:     "no mockeado",
	}
}

// mockInstaller registra qué dependencias se le pidió instalar.
type mockInstaller struct {
	installCalls []DependencyType
	returnErr    error
}

func (m *mockInstaller) InstallAll(results []*CheckResult) []error {
	for _, r := range results {
		m.installCalls = append(m.installCalls, r.Type)
	}
	if m.returnErr != nil {
		return []error{m.returnErr}
	}
	return nil
}

func (m *mockInstaller) Install(dep DependencyType) error {
	m.installCalls = append(m.installCalls, dep)
	return m.returnErr
}

// mockHTTPGetter mockea el cliente HTTP para checkHelixHealth.
type mockHTTPGetter struct {
	err    error
	status int
	body   string
}

func (m *mockHTTPGetter) Get(url string) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &http.Response{
		StatusCode: m.status,
		Body:       io.NopCloser(strings.NewReader(m.body)),
	}, nil
}

// ══════════════════════════════════════════════════════════════════════════
//  Helpers
// ══════════════════════════════════════════════════════════════════════════

// captureStdout ejecuta fn y captura todo lo escrito en stdout.
func captureStdout(fn func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	fn()
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stdout = old
	return buf.String()
}

// withTestHooks configura mocks globales y los restaura al final.
func withTestHooks(t *testing.T, mc *mockChecker, mi *mockInstaller, hg httpGetter) {
	t.Helper()

	savedChecker := newCheckerFn
	savedInstaller := newInstallerFn
	savedHTTP := newHTTPClientFn

	newCheckerFn = func() CheckerInterface { return mc }
	newInstallerFn = func(dryRun, verbose, force bool) InstallerInterface { return mi }
	if hg != nil {
		newHTTPClientFn = func() httpGetter { return hg }
	}

	t.Cleanup(func() {
		newCheckerFn = savedChecker
		newInstallerFn = savedInstaller
		newHTTPClientFn = savedHTTP
	})
}

// ══════════════════════════════════════════════════════════════════════════
//  TestSetupIntegration
// ══════════════════════════════════════════════════════════════════════════

func TestSetupIntegration(t *testing.T) {
	// 1. Mock Checker: uv no instalado, helix no instalado, todo lo demás ok
	mc := &mockChecker{
		results: []*CheckResult{
			{Type: DepGo, Name: "Go", Installed: true, Version: "go1.22.0 linux/amd64", Fixable: false},
			{Type: DepUv, Name: "uv", Installed: false, Fixable: true, Error: "uv no encontrado en PATH"},
			{Type: DepDocker, Name: "Docker", Installed: true, Version: "Docker version 24.0.0", Fixable: false},
			{Type: DepHelix, Name: "HelixDB", Installed: false, Fixable: true, Error: "helix no encontrado en PATH"},
			{Type: DepGit, Name: "Git", Installed: true, Version: "git version 2.40.0", Fixable: false},
		},
	}
	mi := &mockInstaller{}

	withTestHooks(t, mc, mi, nil)

	// 3. Ejecuta RunSetup con dryRun=true
	output := captureStdout(func() {
		err := RunSetup(true, false, false)
		if err != nil {
			t.Fatalf("RunSetup(dryRun=true) no debería error: %v", err)
		}
	})

	// 4. Verifica que se mencionan uv y helix en el plan
	if !strings.Contains(output, "uv") {
		t.Errorf("output debería mencionar 'uv', got:\n%s", output)
	}
	if !strings.Contains(output, "HelixDB") {
		t.Errorf("output debería mencionar 'HelixDB', got:\n%s", output)
	}
	if !strings.Contains(output, "dry-run") {
		t.Errorf("output debería mencionar 'dry-run', got:\n%s", output)
	}
	if !strings.Contains(output, "Instalar uv") {
		t.Errorf("output debería contener 'Instalar uv', got:\n%s", output)
	}
	if !strings.Contains(output, "Instalar HelixDB") {
		t.Errorf("output debería contener 'Instalar HelixDB', got:\n%s", output)
	}

	// 5. Verifica que NO se ejecutaron instalaciones (dry-run)
	if len(mi.installCalls) > 0 {
		t.Errorf("No debería haber installs en dry-run, got: %v", mi.installCalls)
	}
}

// ══════════════════════════════════════════════════════════════════════════
//  TestSetupIntegration_WithForce
// ══════════════════════════════════════════════════════════════════════════

func TestSetupIntegration_WithForce(t *testing.T) {
	// 1. Mock Checker: todo instalado
	mc := &mockChecker{
		results: []*CheckResult{
			{Type: DepGo, Name: "Go", Installed: true, Version: "go1.22.0", Fixable: false},
			{Type: DepUv, Name: "uv", Installed: true, Version: "uv 0.4.0", Fixable: true},
			{Type: DepDocker, Name: "Docker", Installed: true, Version: "Docker 24.0.0", Fixable: false},
			{Type: DepHelix, Name: "HelixDB", Installed: true, Version: "helix 3.0.0", Fixable: true},
			{Type: DepGit, Name: "Git", Installed: true, Version: "git 2.40.0", Fixable: false},
		},
	}
	mi := &mockInstaller{}

	withTestHooks(t, mc, mi, nil)

	// 2. Ejecuta RunSetup(dryRun=false, verbose=false, force=true)
	output := captureStdout(func() {
		err := RunSetup(false, false, true)
		if err != nil {
			t.Fatalf("RunSetup(force=true) no debería error: %v", err)
		}
	})

	// 3. Verifica que reinstala aunque exista (force)
	// Installer.InstallAll debería haber sido llamado (installCalls len > 0)
	if len(mi.installCalls) == 0 {
		t.Error("Esperaba que InstallAll fuera llamado con force=true")
	}

	// Con force=true, debería reinstalar uv y helix (los fixables)
	hasUV := false
	hasHelix := false
	for _, call := range mi.installCalls {
		if call == DepUv {
			hasUV = true
		}
		if call == DepHelix {
			hasHelix = true
		}
	}
	if !hasUV {
		t.Errorf("Esperaba reinstalar uv con force, installCalls=%v", mi.installCalls)
	}
	if !hasHelix {
		t.Errorf("Esperaba reinstalar HelixDB con force, installCalls=%v", mi.installCalls)
	}

	if !strings.Contains(output, "Setup completado") {
		t.Errorf("output debería indicar setup completo, got:\n%s", output)
	}
}

// ══════════════════════════════════════════════════════════════════════════
//  TestDoctorIntegration_AllOk
// ══════════════════════════════════════════════════════════════════════════

func TestDoctorIntegration_AllOk(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// 1. Crear ~/.zyro/config.yaml temporal con paths válidos
	cfg := &Config{
		Version: "2.0.0",
		Project: ProjectConfig{Name: "TestProject", Root: tmpDir},
		Paths: PathsConfig{
			GoBin:     tmpDir,
			UvBin:     tmpDir,
			HelixBin:  tmpDir,
			DockerBin: tmpDir,
			GitBin:    tmpDir,
			ConfigDir: filepath.Join(tmpDir, ".zyro"),
		},
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// 2. Mock Checker: todo instalado
	mc := &mockChecker{
		results: []*CheckResult{
			{Type: DepGo, Name: "Go", Installed: true, Version: "go1.22.0", Fixable: false},
			{Type: DepUv, Name: "uv", Installed: true, Version: "uv 0.4.0", Fixable: true},
			{Type: DepDocker, Name: "Docker", Installed: true, Version: "Docker 24.0.0", Fixable: false},
			{Type: DepHelix, Name: "HelixDB", Installed: true, Version: "helix 3.0.0", Fixable: true},
			{Type: DepGit, Name: "Git", Installed: true, Version: "git 2.40.0", Fixable: false},
		},
	}
	mi := &mockInstaller{}

	// Mock HTTP: HelixDB responde OK
	hg := &mockHTTPGetter{status: http.StatusOK}

	withTestHooks(t, mc, mi, hg)

	// 3. Ejecuta RunDoctor(fix=false)
	output := captureStdout(func() {
		err := RunDoctor(false)
		if err != nil {
			t.Fatalf("RunDoctor(false) no debería error: %v", err)
		}
	})

	// 4. Verifica que reporta "ok" en todos los checks
	if !strings.Contains(output, "Todo en orden") {
		t.Errorf("output debería contener 'Todo en orden', got:\n%s", output)
	}

	// Verificar checks individuales
	for _, check := range []string{"Archivo válido", "Go", "uv", "Docker", "HelixDB", "Git"} {
		if !strings.Contains(output, check) {
			t.Errorf("output debería mencionar '%s'", check)
		}
	}

	// No debería haber ❌ en el output para este test
	if strings.Contains(output, "❌") {
		t.Errorf("no deberían haber errores (❌) en output all-ok:\n%s", output)
	}
}

// ══════════════════════════════════════════════════════════════════════════
//  TestDoctorIntegration_WithFixes
// ══════════════════════════════════════════════════════════════════════════

func TestDoctorIntegration_WithFixes(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// 1. No crear config.yaml (simula que falta)

	// 2. Mock Checker: uv no instalado
	mc := &mockChecker{
		results: []*CheckResult{
			{Type: DepGo, Name: "Go", Installed: true, Version: "go1.22.0", Fixable: false},
			{Type: DepUv, Name: "uv", Installed: false, Fixable: true, Error: "uv no encontrado en PATH"},
			{Type: DepDocker, Name: "Docker", Installed: true, Version: "Docker 24.0.0", Fixable: false},
			{Type: DepHelix, Name: "HelixDB", Installed: true, Version: "helix 3.0.0", Fixable: true},
			{Type: DepGit, Name: "Git", Installed: true, Version: "git 2.40.0", Fixable: false},
		},
	}
	mi := &mockInstaller{}

	// Mock HTTP: HelixDB responde OK para evitar interferencias
	hg := &mockHTTPGetter{status: http.StatusOK}

	withTestHooks(t, mc, mi, hg)

	// 3. Ejecuta RunDoctor(fix=true)
	output := captureStdout(func() {
		err := RunDoctor(true)
		if err != nil {
			t.Fatalf("RunDoctor(true) no debería error: %v", err)
		}
	})

	// 4. Verifica que se generó config.yaml
	configPath := ConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Se esperaba que config.yaml fuera creado por doctor --fix")
	}

	// La configuración generada debe ser cargable
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("No se pudo cargar config creado: %v", err)
	}
	if loaded.Version != "2.0.0" {
		t.Errorf("Version esperada 2.0.0, got %s", loaded.Version)
	}

	// 5. Verifica que se "instaló" uv
	foundUV := false
	for _, call := range mi.installCalls {
		if call == DepUv {
			foundUV = true
			break
		}
	}
	if !foundUV {
		t.Errorf("Se esperaba installCall para uv, installCalls=%v", mi.installCalls)
	}

	if !strings.Contains(output, "Reparaciones completadas") {
		t.Errorf("output debería indicar reparaciones, got:\n%s", output)
	}
}

// ══════════════════════════════════════════════════════════════════════════
//  TestDoctorIntegration_HelixDBNotRunning
// ══════════════════════════════════════════════════════════════════════════

func TestDoctorIntegration_HelixDBNotRunning(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// 1. Crear config.yaml válida
	cfg := &Config{
		Version: "2.0.0",
		Project: ProjectConfig{Name: "TestProject", Root: tmpDir},
		Paths: PathsConfig{
			GoBin:     tmpDir,
			UvBin:     tmpDir,
			HelixBin:  tmpDir,
			DockerBin: tmpDir,
			GitBin:    tmpDir,
			ConfigDir: filepath.Join(tmpDir, ".zyro"),
		},
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// 2. Mock Checker: todo instalado
	mc := &mockChecker{
		results: []*CheckResult{
			{Type: DepGo, Name: "Go", Installed: true, Version: "go1.22.0", Fixable: false},
			{Type: DepUv, Name: "uv", Installed: true, Version: "uv 0.4.0", Fixable: true},
			{Type: DepDocker, Name: "Docker", Installed: true, Version: "Docker 24.0.0", Fixable: false},
			{Type: DepHelix, Name: "HelixDB", Installed: true, Version: "helix 3.0.0", Fixable: true},
			{Type: DepGit, Name: "Git", Installed: true, Version: "git 2.40.0", Fixable: false},
		},
	}
	mi := &mockInstaller{}

	// 3. Mock HTTP: HelixDB NO responde
	hg := &mockHTTPGetter{
		err: fmt.Errorf("connection refused"),
	}

	withTestHooks(t, mc, mi, hg)

	// 4. Ejecuta RunDoctor(fix=false)
	output := captureStdout(func() {
		err := RunDoctor(false)
		if err != nil {
			t.Fatalf("RunDoctor(false) no debería error: %v", err)
		}
	})

	// 5. Verifica que reporta "error" en HelixDB
	if !strings.Contains(output, "No responde en localhost:6969") {
		t.Errorf("output debería indicar que HelixDB no responde, got:\n%s", output)
	}

	if !strings.Contains(output, "Se detectaron") {
		t.Errorf("output debería indicar que se detectaron problemas, got:\n%s", output)
	}
}

// ══════════════════════════════════════════════════════════════════════════
//  TestSetupIntegration_AllInstalled_NoForce
// ══════════════════════════════════════════════════════════════════════════

func TestSetupIntegration_AllInstalled_NoForce(t *testing.T) {
	// Cuando todo está instalado y force=false, no se instala nada
	mc := &mockChecker{
		results: []*CheckResult{
			{Type: DepGo, Name: "Go", Installed: true, Version: "go1.22.0", Fixable: false},
			{Type: DepUv, Name: "uv", Installed: true, Version: "uv 0.4.0", Fixable: true},
			{Type: DepDocker, Name: "Docker", Installed: true, Version: "Docker 24.0.0", Fixable: false},
			{Type: DepHelix, Name: "HelixDB", Installed: true, Version: "helix 3.0.0", Fixable: true},
			{Type: DepGit, Name: "Git", Installed: true, Version: "git 2.40.0", Fixable: false},
		},
	}
	mi := &mockInstaller{}

	withTestHooks(t, mc, mi, nil)

	output := captureStdout(func() {
		err := RunSetup(false, false, false)
		if err != nil {
			t.Fatalf("RunSetup() con todo instalado no debería error: %v", err)
		}
	})

	// El installer no debería ser llamado
	if len(mi.installCalls) != 0 {
		t.Errorf("No debería llamar al installer con todo instalado, got: %v", mi.installCalls)
	}

	if !strings.Contains(output, "Setup completado") {
		t.Errorf("output debería indicar setup completo, got:\n%s", output)
	}
}
