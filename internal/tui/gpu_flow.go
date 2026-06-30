package tui

import (
	"fmt"

	"github.com/yechua-silva/zyrocli/internal/hardware"
)

// GPUSummary returns a formatted string with GPU detection results.
func GPUSummary() string {
	gpuInfo, err := hardware.DetectGPU()
	if err != nil {
		return Warning("Error detectando GPU: " + err.Error())
	}

	if gpuInfo == nil || !gpuInfo.Detected {
		return Info("No se detectó GPU dedicada — modo CPU")
	}

	status := hardware.CheckOllamaGPUStatus()
	lines := fmt.Sprintf("GPU detectada: %s (%s)", gpuInfo.VendorName(), gpuInfo.Name)
	if gpuInfo.DriverVersion != "" {
		lines += fmt.Sprintf("\nDriver: %s", gpuInfo.DriverVersion)
	}

	switch status {
	case hardware.BackendGPU:
		lines = Success(lines)
	case hardware.BackendCPUMode:
		lines = Success(lines) + "\n" + Warning("GPU detectada pero Ollama corre en CPU")
		instructions := hardware.GPUInstructions(gpuInfo, status)
		for _, inst := range instructions {
			lines += "\n" + Info(inst)
		}
	case hardware.BackendComplex:
		lines = Success(lines) + "\n" + Warning("Configuración GPU compleja")
		instructions := hardware.GPUInstructions(gpuInfo, status)
		for _, inst := range instructions {
			lines += "\n" + Info(inst)
		}
	case hardware.BackendNone, hardware.BackendUnknown:
		lines = Info("No se detectó GPU dedicada — modo CPU")
	}

	return lines
}
