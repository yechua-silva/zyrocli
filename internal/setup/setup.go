package setup

import (
	"fmt"
	"os"
	"runtime"
)

// ── Interfaces para testabilidad ────────────────────────────────────────────

// CheckerInterface abstrae la verificación de dependencias.
type CheckerInterface interface {
	CheckAll() []*CheckResult
	Check(dep DependencyType) *CheckResult
}

// InstallerInterface abstrae la instalación de dependencias.
type InstallerInterface interface {
	InstallAll(results []*CheckResult) []error
	Install(dep DependencyType) error
}

// ── Test hooks ──────────────────────────────────────────────────────────────
// Estas variables permiten inyectar mocks desde los tests sin modificar
// el sistema real. Por defecto usan las implementaciones reales.

// newCheckerFn es la fábrica de Checker. Reemplazable en tests.
var newCheckerFn = func() CheckerInterface { return NewChecker() }

// newInstallerFn es la fábrica de Installer. Reemplazable en tests.
var newInstallerFn = func(dryRun, verbose, force bool) InstallerInterface {
	return NewInstaller(dryRun, verbose, force)
}

// ── RunSetup ────────────────────────────────────────────────────────────────

// RunSetup ejecuta el ciclo completo: check → install.
// Retorna nil si todo salió bien, o un error resumen si algo falló.
func RunSetup(dryRun, verbose, force bool) error {
	// ── Banner ──────────────────────────────────────────────────
	if verbose {
		fmt.Println("═══════════════════════════════════════════")
		fmt.Println("  ZyroCLI Setup — v2.0.0")
		fmt.Printf("  OS: %s / Arch: %s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Println("═══════════════════════════════════════════")
		fmt.Println()
	}

	// ── Step 1: Detectar OS ────────────────────────────────────
	osName, arch := PlatformInfo()
	if verbose || dryRun {
		fmt.Printf("📦 Sistema: %s %s\n", osName, arch)
	}

	// Solo soportamos linux y darwin
	if osName != "linux" && osName != "darwin" {
		fmt.Fprintf(os.Stderr, "❌ Sistema operativo no soportado: %s\n", osName)
		fmt.Fprintf(os.Stderr, "   Solo se soportan Linux y macOS.\n")
		fmt.Fprintf(os.Stderr, "   Reporta un issue en: https://github.com/yechua-silva/zyrocli/issues\n")
		return fmt.Errorf("OS no soportado: %s", osName)
	}

	// ── Step 2: Verificar dependencias ──────────────────────────
	if verbose {
		fmt.Println()
		fmt.Println("🔎 Verificando dependencias...")
	}

	checker := newCheckerFn()
	results := checker.CheckAll()

	var allInstalled = true
	for _, r := range results {
		status := "✔"
		if !r.Installed {
			status = "✖"
			allInstalled = false
		}
		if verbose || !r.Installed {
			version := ""
			if r.Version != "" {
				version = " (" + r.Version + ")"
			}
			fmt.Printf("  %s %s%s\n", status, r.Name, version)
			if !r.Installed && r.Error != "" && verbose {
				fmt.Printf("       └ %s\n", r.Error)
			}
		}
	}

	// ── Step 3: Instalar dependencias faltantes ─────────────────
	if !allInstalled || force {
		if dryRun {
			fmt.Println()
			fmt.Println("📋 Plan de instalación (dry-run):")
			for _, r := range results {
				if !r.Installed {
					if r.Fixable {
						fmt.Printf("  · Instalar %s automáticamente\n", r.Name)
					} else {
						fmt.Printf("  · %s requiere instalación manual\n", r.Name)
					}
				} else if force {
					fmt.Printf("  · Reinstalar %s (--force)\n", r.Name)
				}
			}
			fmt.Println()
			fmt.Println("✅ Dry-run completado. Usa 'zyro setup' sin --dry-run para ejecutar.")
			return nil
		}

		if verbose {
			fmt.Println()
			fmt.Println("⬇ Instalando dependencias faltantes...")
		}

		installer := newInstallerFn(dryRun, verbose, force)
		errs := installer.InstallAll(results)

		if len(errs) > 0 {
			fmt.Println()
			fmt.Println("❌ Algunas dependencias no se pudieron instalar:")
			for _, err := range errs {
				fmt.Printf("  · %v\n", err)
			}
			return fmt.Errorf("setup: %d error(es) durante la instalación", len(errs))
		}
	}

	// ── Generar config.yaml ─────────────────────────────
	if !dryRun {
		if err := SaveConfig(DefaultConfig()); err != nil {
			return fmt.Errorf("setup: save config: %w", err)
		}
		if verbose {
			fmt.Println("  ✔ Configuración generada en ~/.zyro/config.yaml")
		}
	}

	// ── Resumen final ───────────────────────────────────────────
	fmt.Println()
	if dryRun {
		fmt.Println("✅ Dry-run completado.")
	} else {
		fmt.Println("✅ Setup completado. Todo en orden.")
	}

	return nil
}
