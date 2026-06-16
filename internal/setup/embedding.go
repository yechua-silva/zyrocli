package setup

import (
	"fmt"
	"os/exec"
	"runtime"
)

// SetupEmbeddings instala y configura el sistema de embeddings
func SetupEmbeddings(dryRun, verbose bool) error {
	if dryRun {
		fmt.Println("[DRY-RUN] Setup Embeddings:")
		fmt.Println("  1. Detectar GPU...")
		fmt.Println("  2. Instalar Ollama (si no existe)...")
		fmt.Println("  3. Pull model mxbai-embed-large...")
		fmt.Println("  4. Configurar fallback API (opcional)...")
		return nil
	}

	// 1. Detectar GPU
	gpu := detectGPU()
	if verbose {
		fmt.Printf("  GPU detectada: %s\n", gpu)
	}

	// 2. Instalar Ollama
	if !commandExists("ollama") {
		fmt.Println("  Instalando Ollama...")
		if err := installOllama(); err != nil {
			return fmt.Errorf("install ollama: %w", err)
		}
	}

	// 3. Pull model
	fmt.Println("  Descargando modelo mxbai-embed-large (~350MB)...")
	if err := pullModel("mxbai-embed-large"); err != nil {
		return fmt.Errorf("pull model: %w", err)
	}

	fmt.Println("  ✅ Embeddings locales activados (mxbai-embed-large, 768 dims)")
	return nil
}

// detectGPU detecta el tipo de GPU disponible en el sistema
func detectGPU() string {
	// NVIDIA
	if err := exec.Command("nvidia-smi").Run(); err == nil {
		out, _ := exec.Command("nvidia-smi", "--query-gpu=name", "--format=csv,noheader").Output()
		if len(out) > 0 {
			return fmt.Sprintf("NVIDIA %s", string(out))
		}
		return "NVIDIA GPU detectada"
	}

	// AMD ROCm
	if err := exec.Command("rocminfo").Run(); err == nil {
		return "AMD GPU detectada"
	}

	// Apple Silicon
	if runtime.GOOS == "darwin" {
		return "Apple Silicon (Metal)"
	}

	return "CPU"
}

// installOllama instala Ollama usando el script oficial
func installOllama() error {
	cmd := exec.Command("curl", "-fsSL", "https://ollama.com/install.sh")
	return cmd.Run()
}

// pullModel descarga un modelo de Ollama
func pullModel(model string) error {
	cmd := exec.Command("ollama", "pull", model)
	return cmd.Run()
}

// commandExists verifica si un comando está disponible en PATH
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
