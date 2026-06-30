package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// SetupHelixAutostart configura HelixDB para inicio automático con systemd --user.
func SetupHelixAutostart() string {
	// Verificar si systemd está disponible
	if _, err := exec.LookPath("systemctl"); err != nil {
		return Warning("systemd no disponible en este sistema")
	}

	// Buscar el servicio de HelixDB
	cmd := exec.Command("systemctl", "--user", "list-units", "--full", "--all", "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		return Warning("No se pudo verificar servicios systemd")
	}

	if strings.Contains(string(output), "helix") {
		// Si existe, habilitar
		enable := exec.Command("systemctl", "--user", "enable", "helix.service")
		if err := enable.Run(); err != nil {
			return Warning("No se pudo habilitar HelixDB en systemd")
		}
		start := exec.Command("systemctl", "--user", "start", "helix.service")
		start.Run()
		return Success("HelixDB configurado para inicio automático")
	}

	// Si no existe el servicio, crear un servicio simple via docker
	home, _ := os.UserHomeDir()
	serviceContent := fmt.Sprintf(`[Unit]
Description=HelixDB (ZyroCLI)
After=docker.service

[Service]
Type=simple
ExecStart=docker start -a helix-zyrocli-global-dev
ExecStop=docker stop helix-zyrocli-global-dev
Restart=on-failure

[Install]
WantedBy=default.target
`)

	serviceDir := home + "/.config/systemd/user"
	os.MkdirAll(serviceDir, 0755)
	servicePath := serviceDir + "/helix-zyrocli.service"

	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return Warning("No se pudo crear servicio systemd: " + err.Error())
	}

	// Recargar y habilitar
	exec.Command("systemctl", "--user", "daemon-reload").Run()
	exec.Command("systemctl", "--user", "enable", "helix-zyrocli.service").Run()

	return Success("Servicio HelixDB creado y habilitado en systemd --user")
}

// SetupOllamaAutostart configura Ollama para inicio automático.
func SetupOllamaAutostart() string {
	// Verificar si systemd está disponible
	if _, err := exec.LookPath("systemctl"); err != nil {
		return Warning("systemd no disponible en este sistema")
	}

	// Buscar servicio de ollama
	cmd := exec.Command("systemctl", "--user", "list-units", "--full", "--all", "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		return Warning("No se pudo verificar servicios systemd")
	}

	if strings.Contains(string(output), "ollama") {
		enable := exec.Command("systemctl", "--user", "enable", "ollama.service")
		if err := enable.Run(); err != nil {
			return Warning("No se pudo habilitar Ollama en systemd")
		}
		start := exec.Command("systemctl", "--user", "start", "ollama.service")
		start.Run()
		return Success("Ollama configurado para inicio automático")
	}

	// Intentar servicio del sistema
	if err := exec.Command("systemctl", "enable", "ollama.service").Run(); err == nil {
		exec.Command("systemctl", "start", "ollama.service").Run()
		return Success("Ollama configurado para inicio automático (system)")
	}

	return Warning("No se encontró servicio Ollama. Instálalo primero con 'zyrocli install'")
}
