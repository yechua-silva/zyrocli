package setup

import (
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config representa ~/.zyro/config.yaml
type Config struct {
	Version     string            `yaml:"version"`
	Project     ProjectConfig     `yaml:"project"`
	Paths       PathsConfig       `yaml:"paths"`
	Preferences PreferencesConfig `yaml:"preferences"`
}

// ProjectConfig contiene la configuración del proyecto.
type ProjectConfig struct {
	Name string `yaml:"name"`
	Root string `yaml:"root"`
}

// PathsConfig contiene las rutas a los binarios del sistema.
type PathsConfig struct {
	GoBin     string `yaml:"go_bin"`
	UvBin     string `yaml:"uv_bin"`
	HelixBin  string `yaml:"helix_bin"`
	DockerBin string `yaml:"docker_bin"`
	GitBin    string `yaml:"git_bin"`
	ConfigDir string `yaml:"config_dir"`
}

// PreferencesConfig contiene las preferencias del CLI.
type PreferencesConfig struct {
	Verbose bool `yaml:"verbose"`
	DryRun  bool `yaml:"dry_run"`
}

// ── Helpers ────────────────────────────────────────────────────────────────

// homeDir retorna el directorio home del usuario.
func homeDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return dir
}

// ConfigPath retorna la ruta al archivo de configuración ~/.zyro/config.yaml.
func ConfigPath() string {
	return filepath.Join(homeDir(), ".zyro", "config.yaml")
}

// configDir retorna el directorio de configuración ~/.zyro/.
func configDir() string {
	return filepath.Join(homeDir(), ".zyro")
}

// getCWD retorna el directorio de trabajo actual.
func getCWD() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

// findBin busca un binario en PATH y retorna su ruta completa,
// o cadena vacía si no se encuentra.
func findBin(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

// ── Default / Load / Save ──────────────────────────────────────────────────

// DefaultConfig genera una configuración por defecto detectando paths
// de los binarios disponibles en el sistema.
func DefaultConfig() *Config {
	return &Config{
		Version: "2.0.0",
		Project: ProjectConfig{
			Name: "ZyroAgentCLI",
			Root: getCWD(),
		},
		Paths: PathsConfig{
			GoBin:     findBin("go"),
			UvBin:     findBin("uv"),
			HelixBin:  findBin("helix"),
			DockerBin: findBin("docker"),
			GitBin:    findBin("git"),
			ConfigDir: configDir(),
		},
		Preferences: PreferencesConfig{
			Verbose: false,
			DryRun:  false,
		},
	}
}

// LoadConfig carga la configuración desde ~/.zyro/config.yaml.
// Retorna error si el archivo no existe o contiene YAML inválido.
func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveConfig guarda la configuración en ~/.zyro/config.yaml.
// Crea el directorio ~/.zyro/ si no existe (con permisos 0755).
func SaveConfig(cfg *Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath(), data, 0644)
}
