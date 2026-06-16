package setup

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Installer instala dependencias faltantes.
type Installer struct {
	checker *Checker
	dryRun  bool
	verbose bool
	force   bool
}

// NewInstaller crea un nuevo Installer.
func NewInstaller(dryRun, verbose, force bool) *Installer {
	return &Installer{
		checker: NewChecker(),
		dryRun:  dryRun,
		verbose: verbose,
		force:   force,
	}
}

// Install instala una dependencia específica.
func (inst *Installer) Install(dep DependencyType) error {
	switch dep {
	case DepUv:
		return inst.installUV()
	case DepHelix:
		return inst.installHelix()
	case DepGo:
		return inst.installGo()
	case DepDocker:
		return inst.installDocker()
	case DepGit:
		return inst.installGit()
	default:
		return fmt.Errorf("no se sabe instalar: %s", dep)
	}
}

// InstallAll intenta instalar todas las dependencias que fallaron.
// Retorna una lista de errores (uno por cada dependencia que falló).
func (inst *Installer) InstallAll(results []*CheckResult) []error {
	var errs []error

	for _, r := range results {
		if r.Installed && !inst.force {
			if inst.verbose {
				fmt.Printf("  ✔ %s ya instalado (%s)\n", r.Name, r.Version)
			}
			continue
		}

		if !r.Fixable {
			if inst.verbose {
				fmt.Printf("  ⚠ %s no se puede instalar automáticamente\n", r.Name)
			}
			continue
		}

		if inst.verbose {
			fmt.Printf("  → Instalando %s...\n", r.Name)
		}

		if err := inst.Install(r.Type); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", r.Name, err))
		} else {
			if inst.verbose {
				fmt.Printf("  ✔ %s instalado correctamente\n", r.Name)
			}
		}
	}

	return errs
}

// installUV instala uv usando el script oficial de astral.sh.
func (inst *Installer) installUV() error {
	if inst.dryRun {
		fmt.Println("  · dry-run: ejecutaría: curl -LsSf https://astral.sh/uv/install.sh | sh")
		return nil
	}

	if inst.verbose {
		fmt.Println("  · Descargando e instalando uv desde astral.sh...")
	}

	cmd := exec.Command("sh", "-c", "curl -LsSf https://astral.sh/uv/install.sh | sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error instalando uv: %w", err)
	}

	return nil
}

// installHelix instala HelixDB usando el script oficial.
func (inst *Installer) installHelix() error {
	if inst.dryRun {
		fmt.Println("  · dry-run: ejecutaría: curl -sSL https://install.helix-db.com | bash")
		return nil
	}

	if inst.verbose {
		fmt.Println("  · Descargando e instalando HelixDB...")
	}

	cmd := exec.Command("sh", "-c", "curl -sSL https://install.helix-db.com | bash")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error instalando HelixDB: %w", err)
	}

	return nil
}

// installGo muestra instrucciones para instalar Go manualmente.
func (inst *Installer) installGo() error {
	fmt.Println("  ⚠ Go no se puede instalar automáticamente.")
	fmt.Println("    Instálalo manualmente desde: https://go.dev/dl/")
	fmt.Println("    O usando el gestor de paquetes de tu sistema:")
	fmt.Println("      · Ubuntu/Debian:  sudo apt install golang-go")
	fmt.Println("      · macOS:          brew install go")
	fmt.Println("      · Arch:           sudo pacman -S go")
	return nil
}

// installDocker muestra instrucciones para instalar Docker manualmente.
func (inst *Installer) installDocker() error {
	fmt.Println("  ⚠ Docker no se puede instalar automáticamente.")
	fmt.Println("    Instálalo manualmente desde: https://docs.docker.com/get-docker/")
	return nil
}

// installGit muestra instrucciones para instalar Git manualmente.
func (inst *Installer) installGit() error {
	fmt.Println("  ⚠ Git no se puede instalar automáticamente.")
	fmt.Println("    Instálalo manualmente desde: https://git-scm.com/downloads")
	fmt.Println("    O usando el gestor de paquetes de tu sistema:")
	fmt.Println("      · Ubuntu/Debian:  sudo apt install git")
	fmt.Println("      · macOS:          brew install git")
	fmt.Println("      · Arch:           sudo pacman -S git")
	return nil
}

// IsInstalled verifica rápidamente si una dependencia está en PATH.
func IsInstalled(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// DepTypeForName retorna el DependencyType para un nombre de binario conocido.
func DepTypeForName(name string) DependencyType {
	switch strings.ToLower(name) {
	case "go":
		return DepGo
	case "uv":
		return DepUv
	case "docker":
		return DepDocker
	case "helix":
		return DepHelix
	case "git":
		return DepGit
	default:
		return DependencyType(name)
	}
}
