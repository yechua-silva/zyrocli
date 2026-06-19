package hardware

// GPUInfo encapsulates detected GPU information.
// This struct is returned by DetectGPU() and consumed by the installer TUI.
type GPUInfo struct {
	// Detected is true if a compatible GPU was found.
	Detected bool `json:"detected"`

	// Vendor is the manufacturer: "nvidia", "amd", "intel", "apple", "".
	Vendor string `json:"vendor"`

	// Name is the product name (e.g. "GeForce RTX 3060").
	Name string `json:"name"`

	// Driver is the driver name (e.g. "nvidia", "amdgpu", "i915", "metal").
	Driver string `json:"driver"`

	// DriverVersion is the exact driver version when available (e.g. "545.29.06").
	// This is only populated when nvidia-smi or equivalent is available.
	DriverVersion string `json:"driver_version"`

	// Platform is the OS: "linux", "darwin", "windows".
	Platform string `json:"platform"`
}

// BackendStatus indicates the level of GPU support in Ollama.
type BackendStatus int

const (
	// BackendUnknown means we couldn't determine the backend.
	BackendUnknown BackendStatus = iota

	// BackendGPU means Ollama is already using the GPU (L1).
	BackendGPU

	// BackendCPUMode means GPU is detected but Ollama is running on CPU (L2).
	BackendCPUMode

	// BackendComplex means GPU is detected but installation is complex (L3).
	// Recommended fallback: Docker.
	BackendComplex

	// BackendNone means no GPU was detected (L4).
	// Recommendation: CPU mode with small models.
	BackendNone
)

// VendorName returns a human-readable vendor name.
func (g *GPUInfo) VendorName() string {
	switch g.Vendor {
	case "nvidia":
		return "NVIDIA"
	case "amd":
		return "AMD"
	case "intel":
		return "Intel"
	case "apple":
		return "Apple Silicon"
	default:
		return "Unknown"
	}
}

// IsGPUAvailable returns true if a dedicated GPU was detected.
// Integrated Intel GPUs are not considered "available" for ML workloads.
func (g *GPUInfo) IsGPUAvailable() bool {
	if !g.Detected {
		return false
	}
	// Intel integrated GPUs are not suitable for ML
	if g.Vendor == "intel" {
		return false
	}
	return true
}
