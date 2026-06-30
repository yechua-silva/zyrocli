//go:build darwin

package hardware

import (
	"os/exec"
	"strings"
)

// gpuDetectorDarwin detects GPU on macOS via sysctl and system_profiler.
type gpuDetectorDarwin struct{}

// DetectGPU implements GPUDetector for macOS.
//
// Detection order:
//  1. sysctl machdep.cpu.brand_string — detects Apple Silicon
//  2. system_profiler SPDisplaysDataType — gets GPU name
//
// Apple Silicon uses Metal Performance Shaders automatically in Ollama.
// No additional configuration is needed.
func (d *gpuDetectorDarwin) DetectGPU() (*GPUInfo, error) {
	info := &GPUInfo{Platform: "darwin"}

	// Method 1: sysctl to detect Apple Silicon
	if detected := detectAppleSilicon(); detected != nil {
		return detected, nil
	}

	// Method 2: system_profiler for Intel Macs with dedicated GPU
	if detected := detectViaSystemProfiler(); detected != nil {
		return detected, nil
	}

	// No GPU detected (Intel Mac without dedicated GPU)
	return info, nil
}

// detectAppleSilicon checks if the system has Apple Silicon.
// Apple Silicon has Metal GPU built-in and Ollama uses it automatically.
func detectAppleSilicon() *GPUInfo {
	sysctl, err := exec.LookPath("sysctl")
	if err != nil {
		return nil
	}

	output, err := exec.Command(sysctl, "-n", "machdep.cpu.brand_string").Output()
	if err != nil {
		return nil
	}

	brand := strings.TrimSpace(string(output))
	if brand == "" {
		return nil
	}

	// Apple Silicon chips contain "Apple" in the brand string
	if strings.Contains(strings.ToLower(brand), "apple") {
		// Try to get the exact GPU name via system_profiler
		gpuName := brand
		if gpu := detectGPUNameViaProfiler(); gpu != "" {
			gpuName = gpu
		}

		return &GPUInfo{
			Detected: true,
			Vendor:   "apple",
			Name:     gpuName,
			Driver:   "metal",
			Platform: "darwin",
		}
	}

	return nil
}

// detectGPUNameViaProfiler uses system_profiler to get the GPU name.
// Returns empty string if unavailable.
func detectGPUNameViaProfiler() string {
	profiler, err := exec.LookPath("system_profiler")
	if err != nil {
		return ""
	}

	output, err := exec.Command(profiler, "SPDisplaysDataType").Output()
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Chipset Model:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "Chipset Model:"))
		}
	}

	return ""
}

// detectViaSystemProfiler detects GPUs on Intel Macs.
func detectViaSystemProfiler() *GPUInfo {
	profiler, err := exec.LookPath("system_profiler")
	if err != nil {
		return nil
	}

	output, err := exec.Command(profiler, "SPDisplaysDataType").Output()
	if err != nil {
		return nil
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Chipset Model:") {
		return nil
	}

	// Extract GPU name
	name := detectGPUNameViaProfiler()
	if name == "" {
		return nil
	}

	// Determine vendor from name
	vendor := ""
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "nvidia"):
		vendor = "nvidia"
	case strings.Contains(lower, "amd") || strings.Contains(lower, "radeon"):
		vendor = "amd"
	case strings.Contains(lower, "intel"):
		vendor = "intel"
	case strings.Contains(lower, "apple"):
		vendor = "apple"
	default:
		vendor = "unknown"
	}

	return &GPUInfo{
		Detected: true,
		Vendor:   vendor,
		Name:     name,
		Platform: "darwin",
	}
}

func init() {
	defaultDetectorFn = func() GPUDetector {
		return &gpuDetectorDarwin{}
	}
}
