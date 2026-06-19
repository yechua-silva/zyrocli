//go:build windows

package hardware

// gpuDetectorWindows detects GPU on Windows via nvidia-smi and WMI.
type gpuDetectorWindows struct{}

import (
	"os/exec"
	"strings"
)

// DetectGPU implements GPUDetector for Windows.
//
// Detection methods:
//  1. ghw.GPU() — uses WMI Win32_VideoController (if ghw is imported)
//  2. nvidia-smi — for NVIDIA GPUs with driver version
//
// On Windows, the Ollama desktop app handles GPU detection automatically.
func (d *gpuDetectorWindows) DetectGPU() (*GPUInfo, error) {
	info := &GPUInfo{Platform: "windows"}

	// Method 1: nvidia-smi for NVIDIA GPUs
	if detected := detectNvidiaSMIWindows(); detected != nil {
		return detected, nil
	}

	// Method 2: Check for AMD via DirectX or OpenCL
	if detected := detectAMDWindows(); detected != nil {
		return detected, nil
	}

	// No GPU detected
	return info, nil
}

// detectNvidiaSMIWindows checks for NVIDIA GPUs via nvidia-smi on Windows.
func detectNvidiaSMIWindows() *GPUInfo {
	nvidiaSMI, err := exec.LookPath("nvidia-smi")
	if err != nil {
		return nil
	}

	output, err := exec.Command(nvidiaSMI, "--query-gpu=name,driver_version", "--format=csv,noheader").Output()
	if err != nil {
		return nil
	}

	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) < 1 {
		return nil
	}

	info := &GPUInfo{
		Detected: true,
		Vendor:   "nvidia",
		Name:     strings.TrimSpace(parts[0]),
		Driver:   "nvidia",
		Platform: "windows",
	}

	if len(parts) >= 2 {
		info.DriverVersion = strings.TrimSpace(parts[1])
	}

	return info
}

// detectAMDWindows checks for AMD GPUs on Windows.
// Uses basic WMI query via PowerShell as fallback.
func detectAMDWindows() *GPUInfo {
	// Attempt to detect AMD GPU via PowerShell WMI query
	powershell, err := exec.LookPath("powershell")
	if err != nil {
		return nil
	}

	cmd := `Get-WmiObject Win32_VideoController | Where-Object { $_.Name -like "*AMD*" -or $_.Name -like "*Radeon*" -or $_.Name -like "*Radeon*" } | Select-Object -ExpandProperty Name`
	output, err := exec.Command(powershell, "-Command", cmd).Output()
	if err != nil {
		return nil
	}

	name := strings.TrimSpace(string(output))
	if name == "" {
		return nil
	}

	return &GPUInfo{
		Detected: true,
		Vendor:   "amd",
		Name:     name,
		Platform: "windows",
	}
}

func init() {
	defaultDetectorFn = func() GPUDetector {
		return &gpuDetectorWindows{}
	}
}
