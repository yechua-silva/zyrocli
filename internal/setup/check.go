package setup

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// DependencyType identifica qué dependencia verificar.
type DependencyType string

const (
	DepGo     DependencyType = "go"
	DepUv     DependencyType = "uv"
	DepDocker DependencyType = "docker"
	DepHelix  DependencyType = "helix"
	DepGit    DependencyType = "git"
)

// CheckResult almacena el resultado de verificar una dependencia.
type CheckResult struct {
	Type      DependencyType `json:"type"`
	Name      string         `json:"name"`
	Installed bool           `json:"installed"`
	Version   string         `json:"version,omitempty"`
	Error     string         `json:"error,omitempty"`
	Fixable   bool           `json:"fixable"`
}

// Checker verifica dependencias del sistema.
type Checker struct {
	// execFunc permite mockear exec.Command en tests.
	execFunc func(name string, arg ...string) *exec.Cmd
}

// NewChecker crea un nuevo Checker.
func NewChecker() *Checker {
	return &Checker{
		execFunc: exec.Command,
	}
}

// Check verifica una dependencia específica.
func (c *Checker) Check(dep DependencyType) *CheckResult {
	switch dep {
	case DepGo:
		return c.checkGo()
	case DepUv:
		return c.checkUv()
	case DepDocker:
		return c.checkDocker()
	case DepHelix:
		return c.checkHelix()
	case DepGit:
		return c.checkGit()
	default:
		return &CheckResult{
			Type:    dep,
			Name:    string(dep),
			Error:   fmt.Sprintf("unknown dependency type: %s", dep),
			Fixable: false,
		}
	}
}

// CheckAll verifica todas las dependencias y retorna los resultados.
func (c *Checker) CheckAll() []*CheckResult {
	deps := []DependencyType{DepGo, DepUv, DepDocker, DepHelix, DepGit}
	results := make([]*CheckResult, 0, len(deps))

	for _, dep := range deps {
		results = append(results, c.Check(dep))
	}

	return results
}

// checkGo verifica la instalación de Go.
func (c *Checker) checkGo() *CheckResult {
	res := &CheckResult{Type: DepGo, Name: "Go", Fixable: false}

	path, err := exec.LookPath("go")
	if err != nil {
		res.Installed = false
		res.Error = "go no encontrado en PATH"
		return res
	}

	cmd := c.execFunc("go", "version")
	out, err := cmd.Output()
	if err != nil {
		res.Installed = true
		res.Error = fmt.Sprintf("go encontrado en %s pero no se pudo obtener versión: %v", path, err)
		return res
	}

	res.Installed = true
	res.Version = strings.TrimSpace(string(out))
	return res
}

// checkUv verifica la instalación de uv.
func (c *Checker) checkUv() *CheckResult {
	res := &CheckResult{Type: DepUv, Name: "uv", Fixable: true}

	path, err := exec.LookPath("uv")
	if err != nil {
		res.Installed = false
		res.Error = "uv no encontrado en PATH"
		return res
	}

	cmd := c.execFunc("uv", "--version")
	out, err := cmd.Output()
	if err != nil {
		res.Installed = true
		res.Error = fmt.Sprintf("uv encontrado en %s pero no se pudo obtener versión: %v", path, err)
		return res
	}

	res.Installed = true
	res.Version = strings.TrimSpace(string(out))
	return res
}

// checkDocker verifica la instalación de Docker.
func (c *Checker) checkDocker() *CheckResult {
	res := &CheckResult{Type: DepDocker, Name: "Docker", Fixable: false}

	path, err := exec.LookPath("docker")
	if err != nil {
		res.Installed = false
		res.Error = "docker no encontrado en PATH"
		return res
	}

	cmd := c.execFunc("docker", "--version")
	out, err := cmd.Output()
	if err != nil {
		res.Installed = true
		res.Error = fmt.Sprintf("docker encontrado en %s pero no se pudo obtener versión: %v", path, err)
		return res
	}

	res.Installed = true
	res.Version = strings.TrimSpace(string(out))
	return res
}

// checkHelix verifica la instalación de HelixDB.
func (c *Checker) checkHelix() *CheckResult {
	res := &CheckResult{Type: DepHelix, Name: "HelixDB", Fixable: true}

	path, err := exec.LookPath("helix")
	if err != nil {
		res.Installed = false
		res.Error = "helix no encontrado en PATH"
		return res
	}

	cmd := c.execFunc("helix", "--version")
	out, err := cmd.Output()
	if err != nil {
		res.Installed = true
		res.Error = fmt.Sprintf("helix encontrado en %s pero no se pudo obtener versión: %v", path, err)
		return res
	}

	res.Installed = true
	res.Version = strings.TrimSpace(string(out))
	return res
}

// checkGit verifica la instalación de Git.
func (c *Checker) checkGit() *CheckResult {
	res := &CheckResult{Type: DepGit, Name: "Git", Fixable: false}

	path, err := exec.LookPath("git")
	if err != nil {
		res.Installed = false
		res.Error = "git no encontrado en PATH"
		return res
	}

	cmd := c.execFunc("git", "--version")
	out, err := cmd.Output()
	if err != nil {
		res.Installed = true
		res.Error = fmt.Sprintf("git encontrado en %s pero no se pudo obtener versión: %v", path, err)
		return res
	}

	res.Installed = true
	res.Version = strings.TrimSpace(string(out))
	return res
}

// PlatformInfo devuelve información del sistema operativo.
func PlatformInfo() (os, arch string) {
	return runtime.GOOS, runtime.GOARCH
}
