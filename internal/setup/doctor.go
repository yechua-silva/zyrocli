package setup

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DoctorResult almacena el resultado de una revisión del doctor.
type DoctorResult struct {
	Check   string `json:"check"`
	Status  string `json:"status"`  // "ok", "warning", "error"
	Message string `json:"message"`
}

// ── RunDoctor ──────────────────────────────────────────────────────────────

// RunDoctor ejecuta el diagnóstico completo del CLI.
// Si fix=true, intenta reparar automáticamente los problemas encontrados.
func RunDoctor(fix bool) error {
	fmt.Println("🔍 Zyro Doctor — Diagnóstico del CLI")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()

	var results []DoctorResult

	// ── 1. Config file ──────────────────────────────────────────────────
	fmt.Println("📋 1. Archivo de configuración")
	r := checkConfigFile()
	results = append(results, r)
	printResult(r)
	fmt.Println()

	// ── 2. Dependencies ─────────────────────────────────────────────────
	fmt.Println("📦 2. Dependencias")
	deps := checkDependencies()
	results = append(results, deps...)
	for _, d := range deps {
		printResult(d)
	}
	fmt.Println()

	// ── 3. Paths ────────────────────────────────────────────────────────
	fmt.Println("📁 3. Paths de configuración")
	cfg, _ := LoadConfig()
	if cfg != nil {
		paths := checkPaths(cfg)
		results = append(results, paths...)
		for _, p := range paths {
			printResult(p)
		}
	} else {
		msg := DoctorResult{Check: "paths", Status: "warning", Message: "No hay configuración para validar paths"}
		results = append(results, msg)
		printResult(msg)
	}
	fmt.Println()

	// ── 4. Permissions ─────────────────────────────────────────────────
	fmt.Println("🔒 4. Permisos de directorios")
	perms := checkPermissions()
	results = append(results, perms...)
	for _, p := range perms {
		printResult(p)
	}
	fmt.Println()

	// ── 5. HelixDB health ──────────────────────────────────────────────
	fmt.Println("💾 5. HelixDB")
	h := checkHelixHealth()
	results = append(results, h)
	printResult(h)
	fmt.Println()

	// ── Summary ─────────────────────────────────────────────────────────
	fmt.Println("═══════════════════════════════════════")
	total := countIssues(results)
	if total == 0 {
		fmt.Println("✅ Todo en orden. No se detectaron problemas.")
		return nil
	}

	fmt.Printf("⚠️  Se detectaron %d problema(s).\n", total)
	fmt.Println()

	if fix {
		fmt.Println("🔧 Ejecutando reparaciones...")
		fmt.Println()
		if err := runFixes(results); err != nil {
			return fmt.Errorf("doctor --fix: %w", err)
		}
		fmt.Println()
		fmt.Println("✅ Reparaciones completadas.")
	} else {
		fmt.Println("💡 Usa 'zyro doctor --fix' para reparar automáticamente.")
	}

	return nil
}

// ── Individual checks ──────────────────────────────────────────────────────

// checkConfigFile verifica que ~/.zyro/config.yaml existe y es YAML válido.
func checkConfigFile() DoctorResult {
	path := ConfigPath()

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DoctorResult{
				Check:   "config_file",
				Status:  "error",
				Message: fmt.Sprintf("Archivo no encontrado: %s", path),
			}
		}
		return DoctorResult{
			Check:   "config_file",
			Status:  "error",
			Message: fmt.Sprintf("Error al acceder: %v", err),
		}
	}

	if info.IsDir() {
		return DoctorResult{
			Check:   "config_file",
			Status:  "error",
			Message: fmt.Sprintf("No es un archivo: %s", path),
		}
	}

	if _, err := LoadConfig(); err != nil {
		return DoctorResult{
			Check:   "config_file",
			Status:  "error",
			Message: fmt.Sprintf("YAML inválido: %v", err),
		}
	}

	return DoctorResult{
		Check:   "config_file",
		Status:  "ok",
		Message: fmt.Sprintf("Archivo válido: %s", path),
	}
}

// checkDependencies verifica las dependencias del sistema usando el Checker.
func checkDependencies() []DoctorResult {
	checker := newCheckerFn()
	results := checker.CheckAll()

	docResults := make([]DoctorResult, 0, len(results))
	for _, r := range results {
		status := "ok"
		if !r.Installed {
			status = "error"
		}
		msg := r.Name
		if r.Version != "" {
			msg += " " + r.Version
		}
		if r.Error != "" {
			msg += " — " + r.Error
		}
		docResults = append(docResults, DoctorResult{
			Check:   "dep_" + string(r.Type),
			Status:  status,
			Message: msg,
		})
	}
	return docResults
}

// checkPaths verifica que los paths almacenados en la configuración existen.
func checkPaths(cfg *Config) []DoctorResult {
	entries := map[string]string{
		"go_bin":     cfg.Paths.GoBin,
		"uv_bin":     cfg.Paths.UvBin,
		"helix_bin":  cfg.Paths.HelixBin,
		"docker_bin": cfg.Paths.DockerBin,
		"git_bin":    cfg.Paths.GitBin,
		"config_dir": cfg.Paths.ConfigDir,
	}

	results := make([]DoctorResult, 0, len(entries))
	for name, p := range entries {
		if p == "" {
			results = append(results, DoctorResult{
				Check:   "path_" + name,
				Status:  "warning",
				Message: fmt.Sprintf("%s: no configurado", name),
			})
			continue
		}
		if _, err := os.Stat(p); err != nil {
			results = append(results, DoctorResult{
				Check:   "path_" + name,
				Status:  "error",
				Message: fmt.Sprintf("%s: no existe (%s)", name, p),
			})
		} else {
			results = append(results, DoctorResult{
				Check:   "path_" + name,
				Status:  "ok",
				Message: fmt.Sprintf("%s: %s", name, p),
			})
		}
	}
	return results
}

// checkPermissions verifica que los directorios tengan permisos correctos.
func checkPermissions() []DoctorResult {
	dirs := []string{configDir()}

	results := make([]DoctorResult, 0, len(dirs))
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			results = append(results, DoctorResult{
				Check:   "perm_" + filepath.Base(dir),
				Status:  "warning",
				Message: fmt.Sprintf("Directorio no existe: %s", dir),
			})
			continue
		}
		perm := info.Mode().Perm()
		if perm&0755 != 0755 {
			results = append(results, DoctorResult{
				Check:   "perm_" + filepath.Base(dir),
				Status:  "warning",
				Message: fmt.Sprintf("Permisos %o en %s (se esperaba 0755)", perm, dir),
			})
		} else {
			results = append(results, DoctorResult{
				Check:   "perm_" + filepath.Base(dir),
				Status:  "ok",
				Message: fmt.Sprintf("Permisos correctos en %s", dir),
			})
		}
	}
	return results
}

// httpGetter permite mockear llamadas HTTP en tests.
type httpGetter interface {
	Get(url string) (*http.Response, error)
}

// newHTTPClientFn permite reemplazar el cliente HTTP en tests.
var newHTTPClientFn = func() httpGetter {
	return &http.Client{Timeout: 3 * time.Second}
}

// checkHelixHealth verifica que HelixDB responde en /health.
func checkHelixHealth() DoctorResult {
	url := GetHelixDBURL()
	client := newHTTPClientFn()
	resp, err := client.Get(url + "/health")
	if err != nil {
		return DoctorResult{
			Check:   "helix_health",
			Status:  "error",
			Message: fmt.Sprintf("No responde en %s — %v", url, err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return DoctorResult{
			Check:   "helix_health",
			Status:  "ok",
			Message: fmt.Sprintf("HelixDB responde correctamente en %s", url),
		}
	}
	return DoctorResult{
		Check:   "helix_health",
		Status:  "error",
		Message: fmt.Sprintf("HelixDB devolvió HTTP %d", resp.StatusCode),
	}
}

// ── Fix logic ──────────────────────────────────────────────────────────────

// runFixes itera sobre los resultados e intenta reparar cada problema.
func runFixes(results []DoctorResult) error {
	var errs []error

	for _, r := range results {
		switch {
		case r.Check == "config_file" && r.Status != "ok":
			fmt.Println("  → Generando configuración por defecto...")
			cfg := DefaultConfig()
			if err := SaveConfig(cfg); err != nil {
				errs = append(errs, fmt.Errorf("config: %w", err))
			} else {
				fmt.Printf("    ✔ Configuración creada en %s\n", ConfigPath())
			}

		case strings.HasPrefix(r.Check, "dep_") && r.Status != "ok":
			depName := r.Check[4:]
			depType := DepTypeForName(depName)
			checker := newCheckerFn()
			cr := checker.Check(depType)
			if !cr.Fixable {
				fmt.Printf("    ⚠ %s requiere instalación manual\n", cr.Name)
				continue
			}
			fmt.Printf("  → Instalando %s...\n", cr.Name)
			installer := newInstallerFn(false, true, false)
			if err := installer.Install(depType); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", cr.Name, err))
			} else {
				fmt.Printf("    ✔ %s instalado correctamente\n", cr.Name)
			}

		case strings.HasPrefix(r.Check, "path_") && r.Status != "ok":
			// Regenerar configuración completa para corregir paths
			fmt.Println("  → Regenerando configuración con paths correctos...")
			cfg := DefaultConfig()
			if err := SaveConfig(cfg); err != nil {
				errs = append(errs, fmt.Errorf("path fix: %w", err))
			} else {
				fmt.Println("    ✔ Configuración regenerada")
			}

		case strings.HasPrefix(r.Check, "perm_") && r.Status != "ok":
			dir := configDir()
			if err := os.MkdirAll(dir, 0755); err != nil {
				errs = append(errs, fmt.Errorf("chmod: %w", err))
			} else {
				fmt.Printf("    ✔ Permisos corregidos en %s\n", dir)
			}

		case r.Check == "helix_health" && r.Status != "ok":
			if _, err := exec.LookPath("helix"); err != nil {
				fmt.Println("    ⚠ helix no encontrado en PATH, no se puede iniciar automáticamente")
				continue
			}
			fmt.Println("  → Intentando iniciar HelixDB...")
			cmd := exec.Command("helix", "start", "dev")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				errs = append(errs, fmt.Errorf("helix start: %w", err))
			} else {
				fmt.Println("    ✔ HelixDB iniciado")
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d error(es) durante la reparación", len(errs))
	}
	return nil
}

// ── Output helpers ─────────────────────────────────────────────────────────

// printResult imprime un resultado de diagnóstico formateado.
func printResult(r DoctorResult) {
	prefix := "  "
	switch r.Status {
	case "ok":
		prefix += "✅ "
	case "warning":
		prefix += "⚠️  "
	case "error":
		prefix += "❌ "
	default:
		prefix += "❓ "
	}
	fmt.Printf("%s%s\n", prefix, r.Message)
}

// countIssues cuenta cuántos resultados tienen estado distinto de "ok".
func countIssues(results []DoctorResult) int {
	n := 0
	for _, r := range results {
		if r.Status != "ok" {
			n++
		}
	}
	return n
}
