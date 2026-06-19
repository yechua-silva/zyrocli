package opencode

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

//go:embed tui-plugins/zorro-logo.tsx
var zorroLogoPlugin string

// ZorroPluginPath retorna la ruta donde se instalará el plugin del logo.
func ZorroPluginPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "opencode", "tui-plugins", "zorro-logo.tsx")
}

// WriteZorroLogo escribe el plugin TSX del Zorro Hacker en el directorio de plugins de OpenCode.
func WriteZorroLogo() (string, error) {
	pluginPath := ZorroPluginPath()

	if err := os.MkdirAll(filepath.Dir(pluginPath), 0755); err != nil {
		return "", fmt.Errorf("opencode: create tui-plugins dir: %w", err)
	}

	if err := os.WriteFile(pluginPath, []byte(zorroLogoPlugin), 0644); err != nil {
		return "", fmt.Errorf("opencode: write zorro logo plugin: %w", err)
	}

	return pluginPath, nil
}

// tuiConfig representa la estructura de ~/.config/opencode/tui.json.
type tuiConfig struct {
	Schema  string   `json:"$schema,omitempty"`
	Plugin  []string `json:"plugin,omitempty"`
}

// UpdateTuiJSON actualiza ~/.config/opencode/tui.json para incluir
// el plugin zorro-logo sin pisar plugins existentes que el usuario
// haya agregado manualmente.
func UpdateTuiJSON() error {
	home, _ := os.UserHomeDir()
	tuiPath := filepath.Join(home, ".config", "opencode", "tui.json")

	if err := os.MkdirAll(filepath.Dir(tuiPath), 0755); err != nil {
		return fmt.Errorf("opencode: create config dir: %w", err)
	}

	// Leer config existente
	cfg := tuiConfig{
		Schema: "https://opencode.ai/tui.json",
	}
	if data, err := os.ReadFile(tuiPath); err == nil {
		json.Unmarshal(data, &cfg)
	}
	if cfg.Plugin == nil {
		cfg.Plugin = []string{}
	}

	// Limpiar plugins stale (ubicaciones antiguas o plugins deprecados)
	stale := []string{
		"opencode-subagent-statusline",
		filepath.Join(home, ".config", "opencode", "plugins", "zyro-model.js"),
	}
	var cleaned []string
	for _, p := range cfg.Plugin {
		if !slices.Contains(stale, p) {
			cleaned = append(cleaned, p)
		}
	}
	cfg.Plugin = cleaned

	// Agregar plugins de Zyro si no están ya registrados
	wanted := []string{
		ZorroPluginPath(),
	}
	for _, p := range wanted {
		if !slices.Contains(cfg.Plugin, p) {
			cfg.Plugin = append(cfg.Plugin, p)
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("opencode: marshal tui.json: %w", err)
	}

	if err := os.WriteFile(tuiPath, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("opencode: write tui.json: %w", err)
	}

	return nil
}
