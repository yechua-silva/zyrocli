//go:build linux

package hardware

import (
	"os"
	"os/exec"
	"strings"
)

// gpuDetectorLinux detects GPU on Linux via nvidia-smi and lspci.
type gpuDetectorLinux struct{}

// DetectGPU implements GPUDetector for Linux.
//
// Detection order:
//  1. nvidia-smi — fastest, gives exact driver version
//  2. ghw.GPU() — PCI-based detection (if ghw is available)
//  3. lspci -d 10de: / 1002: — fallback vendor ID check
//  4. /dev/kfd — ROCm availability check for AMD
//
// All methods are read-only. No system modifications are made.
func (d *gpuDetectorLinux) DetectGPU() (*GPUInfo, error) {
	info := &GPUInfo{Platform: "linux"}

	// Method 1: nvidia-smi (NVIDIA with driver version)
	if detected := detectNvidiaSMI(); detected != nil {
		return detected, nil
	}

	// Method 2: lspci vendor ID detection
	if detected := detectViaLSPCI(); detected != nil {
		return detected, nil
	}

	// Method 3: Check for AMD ROCm via /dev/kfd
	if detected := detectROCm(); detected != nil {
		return detected, nil
	}

	// No GPU detected — return empty info
	return info, nil
}

// detectNvidiaSMI runs nvidia-smi to detect NVIDIA GPUs.
// Returns nil if nvidia-smi is not available or fails.
func detectNvidiaSMI() *GPUInfo {
	nvidiaSMI, err := exec.LookPath("nvidia-smi")
	if err != nil {
		return nil
	}

	output, err := exec.Command(nvidiaSMI, "--query-gpu=name,driver_version", "--format=csv,noheader").Output()
	if err != nil {
		// nvidia-smi exists but failed (e.g., no permissions)
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
		Platform: "linux",
	}

	if len(parts) >= 2 {
		info.DriverVersion = strings.TrimSpace(parts[1])
	}

	return info
}

// detectViaLSPCI uses lspci to detect GPU vendors.
// Returns nil if lspci is not available or no GPU found.
// Only detects the vendor — no driver version available.
func detectViaLSPCI() *GPUInfo {
	lspci, err := exec.LookPath("lspci")
	if err != nil {
		return nil
	}

	// Check for NVIDIA (vendor ID 10de)
	output, err := exec.Command(lspci, "-d", "10de:").Output()
	if err == nil && len(output) > 0 {
		// Try to extract GPU name from the lspci line
		name := extractGPUNameFromLSPCI(string(output))
		return &GPUInfo{
			Detected: true,
			Vendor:   "nvidia",
			Name:     name,
			Driver:   "nvidia",
			Platform: "linux",
		}
	}

	// Check for AMD (vendor ID 1002)
	output, err = exec.Command(lspci, "-d", "1002:").Output()
	if err == nil && len(output) > 0 {
		name := extractGPUNameFromLSPCI(string(output))
		return &GPUInfo{
			Detected: true,
			Vendor:   "amd",
			Name:     name,
			Platform: "linux",
		}
	}

	// Check for Intel (vendor ID 8086) — display class
	output, err = exec.Command(lspci, "-d", "8086:").Output()
	if err == nil && len(output) > 0 {
		// Only report Intel if it's a display controller (VGA)
		if strings.Contains(strings.ToLower(string(output)), "vga") ||
			strings.Contains(strings.ToLower(string(output)), "display") ||
			strings.Contains(strings.ToLower(string(output)), "graphics") {
			name := extractGPUNameFromLSPCI(string(output))
			return &GPUInfo{
				Detected: true,
				Vendor:   "intel",
				Name:     name,
				Driver:   "i915",
				Platform: "linux",
			}
		}
	}

	return nil
}

// detectROCm checks for AMD ROCm availability.
func detectROCm() *GPUInfo {
	// Check for /dev/kfd (ROCm device)
	if _, err := os.Stat("/dev/kfd"); err != nil {
		return nil
	}

	// Check for rocminfo
	rocminfo, err := exec.LookPath("rocminfo")
	if err != nil {
		return &GPUInfo{
			Detected: true,
			Vendor:   "amd",
			Name:     "AMD GPU (ROCm)",
			Platform: "linux",
		}
	}

	output, err := exec.Command(rocminfo).Output()
	if err == nil {
		name := extractGPUNameFromROCm(string(output))
		return &GPUInfo{
			Detected: true,
			Vendor:   "amd",
			Name:     name,
			Platform: "linux",
		}
	}

	return &GPUInfo{
		Detected: true,
		Vendor:   "amd",
		Name:     "AMD GPU (ROCm)",
		Platform: "linux",
	}
}

// extractGPUNameFromLSPCI parses the GPU name from an lspci line.
// Example input: "03:00.0 VGA compatible controller: NVIDIA Corporation GP107 [GeForce GTX 1050 Ti]"
func extractGPUNameFromLSPCI(lspciOutput string) string {
	lines := strings.Split(strings.TrimSpace(lspciOutput), "\n")
	if len(lines) == 0 {
		return "GPU detected"
	}

	// Take the last line (usually has the most info)
	line := lines[len(lines)-1]

	// Try to extract text after the last ": "
	if idx := strings.LastIndex(line, ": "); idx != -1 {
		name := strings.TrimSpace(line[idx+2:])
		if name != "" {
			return name
		}
	}

	return "GPU detected"
}

// extractGPUNameFromROCm parses the GPU name from rocminfo output.
func extractGPUNameFromROCm(rocminfoOutput string) string {
	for _, line := range strings.Split(rocminfoOutput, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Name:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[1])
				if name != "" {
					return name
				}
			}
		}
	}
	return "AMD GPU (ROCm)"
}

func init() {
	defaultDetectorFn = func() GPUDetector {
		return &gpuDetectorLinux{}
	}
}
