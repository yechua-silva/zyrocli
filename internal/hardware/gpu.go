package hardware

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// GPUDetector is the interface for detecting the system GPU.
type GPUDetector interface {
	DetectGPU() (*GPUInfo, error)
}

// defaultDetectorFn is initialized by platform-specific init() functions.
// Each OS file (gpu_linux.go, gpu_darwin.go, gpu_windows.go) sets this
// in its init() to return the appropriate implementation.
var defaultDetectorFn func() GPUDetector

// DetectGPU detects the system GPU.
// It is the main public function used by the installer.
func DetectGPU() (*GPUInfo, error) {
	if defaultDetectorFn == nil {
		return &GPUInfo{Platform: runtime.GOOS}, nil
	}
	return defaultDetectorFn().DetectGPU()
}

// CheckOllamaGPUStatus checks if Ollama is currently using the GPU.
func CheckOllamaGPUStatus() BackendStatus {
	driver := os.Getenv("OLLAMA_GPU_DRIVER")
	switch strings.ToLower(driver) {
	case "cuda", "rocm", "vulkan":
		return BackendGPU
	case "cpu":
		return BackendCPUMode
	}

	if hasNvidiaSMI() {
		return BackendGPU
	}

	if hasROCm() {
		return BackendGPU
	}

	return BackendUnknown
}

// hasNvidiaSMI checks if nvidia-smi is available.
func hasNvidiaSMI() bool {
	_, err := execLookPath("nvidia-smi")
	return err == nil
}

// hasROCm checks if ROCm is available (Linux only).
func hasROCm() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if _, err := os.Stat("/dev/kfd"); err == nil {
		return true
	}
	return false
}

// execLookPath is overridable in tests.
var execLookPath = func(file string) (string, error) {
	return "", fmt.Errorf("not implemented: %s", file)
}

// GPUInstructions generates human-readable instructions for Ollama GPU config.
func GPUInstructions(info *GPUInfo, status BackendStatus) []string {
	switch status {
	case BackendGPU:
		return nil
	case BackendCPUMode:
		return generateCPUModeInstructions(info)
	case BackendComplex:
		return generateComplexInstructions(info)
	case BackendNone, BackendUnknown:
		return []string{
			"No dedicated GPU detected. Ollama will run in CPU mode.",
			"Small models (≤3B parameters) work well on CPU.",
			"Recommended: phi4-mini:3.8b or llama3.2:3b",
		}
	default:
		return nil
	}
}

func generateCPUModeInstructions(info *GPUInfo) []string {
	switch info.Vendor {
	case "nvidia":
		return []string{
			fmt.Sprintf("GPU detected: %s (%s)", info.Name, info.VendorName()),
			"",
			"To enable GPU acceleration in Ollama:",
			"",
			"1. Stop Ollama: ollama stop",
			"2. Set environment variable:",
			"   export OLLAMA_GPU_DRIVER=cuda",
			"3. Restart Ollama: ollama serve",
			"",
			"To make this permanent, add to ~/.bashrc or ~/.zshrc:",
			"   echo 'export OLLAMA_GPU_DRIVER=cuda' >> ~/.bashrc",
		}
	case "amd":
		return []string{
			fmt.Sprintf("GPU detected: %s (%s)", info.Name, info.VendorName()),
			"",
			"Recommended: Use Vulkan backend (easier):",
			"   export OLLAMA_GPU_DRIVER=vulkan",
			"",
			"Advanced: Use ROCm backend (more performance):",
			"   export OLLAMA_GPU_DRIVER=rocm",
			"   sudo modprobe amdkfd  (if not loaded)",
			"   sudo usermod -aG render,video $USER",
			"",
			"Restart Ollama after configuration:",
			"   ollama serve",
		}
	case "apple":
		return []string{
			fmt.Sprintf("GPU detected: %s", info.Name),
			"",
			"Apple Silicon uses Metal Performance Shaders automatically.",
			"Ollama should already be using GPU acceleration.",
			"No configuration needed.",
		}
	default:
		return []string{
			"GPU detected but vendor is not specifically supported.",
			"Try setting OLLAMA_GPU_DRIVER to one of: cuda, rocm, vulkan",
		}
	}
}

func generateComplexInstructions(info *GPUInfo) []string {
	switch info.Vendor {
	case "nvidia":
		return []string{
			fmt.Sprintf("GPU detected: %s (%s)", info.Name, info.VendorName()),
			"",
			"Installing GPU drivers can be complex on your system.",
			"Alternative: Run Ollama via Docker with GPU support:",
			"",
			"   docker run -d \\",
			"     --gpus all \\",
			"     -v ollama:/root/.ollama \\",
			"     -p 11434:11434 \\",
			"     --name ollama \\",
			"     ollama/ollama",
			"",
			"Benefits: no system modifications, easy to uninstall:",
			"   docker rm -f ollama",
		}
	case "amd":
		return []string{
			fmt.Sprintf("GPU detected: %s (%s)", info.Name, info.VendorName()),
			"",
			"Run Ollama via Docker with Vulkan support:",
			"",
			"   docker run -d \\",
			"     --device /dev/kfd --device /dev/dri \\",
			"     -v ollama:/root/.ollama \\",
			"     -p 11434:11434 \\",
			"     -e OLLAMA_GPU_DRIVER=vulkan \\",
			"     --name ollama \\",
			"     ollama/ollama",
		}
	default:
		return []string{
			"GPU driver installation is complex on your system.",
			"Consider using Docker with GPU support.",
			"See: https://ollama.com/blog/ollama-docker",
		}
	}
}
