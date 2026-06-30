package hardware

import (
	"errors"
	"testing"
)

// mockGPUDetector implements GPUDetector for testing.
type mockGPUDetector struct {
	gpuInfo *GPUInfo
	err     error
}

func (m *mockGPUDetector) DetectGPU() (*GPUInfo, error) {
	return m.gpuInfo, m.err
}

// withMockGPU sets a mock detector for the duration of a test.
func withMockGPU(t *testing.T, info *GPUInfo, err error, fn func()) {
	t.Helper()
	original := defaultDetectorFn
	defaultDetectorFn = func() GPUDetector {
		return &mockGPUDetector{gpuInfo: info, err: err}
	}
	defer func() { defaultDetectorFn = original }()
	fn()
}

// --- Test cases for DetectGPU ---

type detectGPUTestCase struct {
	name     string
	mock     *mockGPUDetector
	wantInfo *GPUInfo
	wantErr  bool
}

func TestDetectGPU(t *testing.T) {
	tests := []detectGPUTestCase{
		{
			name: "Linux with NVIDIA GPU via nvidia-smi",
			mock: &mockGPUDetector{
				gpuInfo: &GPUInfo{
					Detected:      true,
					Vendor:        "nvidia",
					Name:          "NVIDIA GeForce RTX 3060",
					Driver:        "nvidia",
					DriverVersion: "545.29.06",
					Platform:      "linux",
				},
			},
			wantInfo: &GPUInfo{
				Detected:      true,
				Vendor:        "nvidia",
				Name:          "NVIDIA GeForce RTX 3060",
				Driver:        "nvidia",
				DriverVersion: "545.29.06",
				Platform:      "linux",
			},
			wantErr: false,
		},
		{
			name: "Linux with AMD GPU",
			mock: &mockGPUDetector{
				gpuInfo: &GPUInfo{
					Detected: true,
					Vendor:   "amd",
					Name:     "AMD Radeon RX 7900 XTX",
					Driver:   "amdgpu",
					Platform: "linux",
				},
			},
			wantInfo: &GPUInfo{
				Detected: true,
				Vendor:   "amd",
				Name:     "AMD Radeon RX 7900 XTX",
				Driver:   "amdgpu",
				Platform: "linux",
			},
			wantErr: false,
		},
		{
			name: "macOS Apple Silicon",
			mock: &mockGPUDetector{
				gpuInfo: &GPUInfo{
					Detected: true,
					Vendor:   "apple",
					Name:     "Apple M3 Pro",
					Driver:   "metal",
					Platform: "darwin",
				},
			},
			wantInfo: &GPUInfo{
				Detected: true,
				Vendor:   "apple",
				Name:     "Apple M3 Pro",
				Driver:   "metal",
				Platform: "darwin",
			},
			wantErr: false,
		},
		{
			name: "Linux without GPU (CPU only)",
			mock: &mockGPUDetector{
				gpuInfo: &GPUInfo{
					Detected: false,
					Vendor:   "",
					Name:     "",
					Platform: "linux",
				},
			},
			wantInfo: &GPUInfo{
				Detected: false,
				Platform: "linux",
			},
			wantErr: false,
		},
		{
			name: "DetectGPU returns error on permission denied",
			mock: &mockGPUDetector{
				err: errors.New("permission denied accessing /sys/class/drm"),
			},
			wantInfo: nil,
			wantErr:  true,
		},
		{
			name: "Linux with Intel integrated GPU",
			mock: &mockGPUDetector{
				gpuInfo: &GPUInfo{
					Detected: true,
					Vendor:   "intel",
					Name:     "Intel UHD Graphics 770",
					Driver:   "i915",
					Platform: "linux",
				},
			},
			wantInfo: &GPUInfo{
				Detected: true,
				Vendor:   "intel",
				Name:     "Intel UHD Graphics 770",
				Driver:   "i915",
				Platform: "linux",
			},
			wantErr: false,
		},
		{
			name: "Windows with NVIDIA GPU",
			mock: &mockGPUDetector{
				gpuInfo: &GPUInfo{
					Detected:      true,
					Vendor:        "nvidia",
					Name:          "NVIDIA GeForce RTX 4070",
					Driver:        "nvidia",
					DriverVersion: "31.0.15.XXXX",
					Platform:      "windows",
				},
			},
			wantInfo: &GPUInfo{
				Detected:      true,
				Vendor:        "nvidia",
				Name:          "NVIDIA GeForce RTX 4070",
				Driver:        "nvidia",
				DriverVersion: "31.0.15.XXXX",
				Platform:      "windows",
			},
			wantErr: false,
		},
		{
			name: "macOS Intel without dedicated GPU",
			mock: &mockGPUDetector{
				gpuInfo: &GPUInfo{
					Detected: false,
					Platform: "darwin",
				},
			},
			wantInfo: &GPUInfo{
				Detected: false,
				Platform: "darwin",
			},
			wantErr: false,
		},
		{
			name: "Unsupported OS returns empty info",
			mock: &mockGPUDetector{
				gpuInfo: &GPUInfo{
					Platform: "freebsd",
				},
			},
			wantInfo: &GPUInfo{
				Platform: "freebsd",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withMockGPU(t, tt.mock.gpuInfo, tt.mock.err, func() {
				got, err := DetectGPU()

				if tt.wantErr {
					if err == nil {
						t.Error("expected error but got nil")
					}
					return
				}
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if got == nil {
					t.Error("expected GPUInfo but got nil")
					return
				}
				if got.Detected != tt.wantInfo.Detected {
					t.Errorf("Detected = %v, want %v", got.Detected, tt.wantInfo.Detected)
				}
				if got.Vendor != tt.wantInfo.Vendor {
					t.Errorf("Vendor = %q, want %q", got.Vendor, tt.wantInfo.Vendor)
				}
				if got.Name != tt.wantInfo.Name {
					t.Errorf("Name = %q, want %q", got.Name, tt.wantInfo.Name)
				}
				if got.Platform != tt.wantInfo.Platform {
					t.Errorf("Platform = %q, want %q", got.Platform, tt.wantInfo.Platform)
				}
			})
		})
	}
}

// --- Test cases for VendorName ---

func TestGPUInfo_VendorName(t *testing.T) {
	tests := []struct {
		vendor string
		want   string
	}{
		{"nvidia", "NVIDIA"},
		{"amd", "AMD"},
		{"intel", "Intel"},
		{"apple", "Apple Silicon"},
		{"", "Unknown"},
		{"other", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.vendor, func(t *testing.T) {
			g := &GPUInfo{Vendor: tt.vendor}
			if got := g.VendorName(); got != tt.want {
				t.Errorf("VendorName() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- Test cases for IsGPUAvailable ---

func TestGPUInfo_IsGPUAvailable(t *testing.T) {
	tests := []struct {
		name  string
		info  *GPUInfo
		want  bool
	}{
		{
			name: "NVIDIA GPU available",
			info: &GPUInfo{Detected: true, Vendor: "nvidia"},
			want: true,
		},
		{
			name: "AMD GPU available",
			info: &GPUInfo{Detected: true, Vendor: "amd"},
			want: true,
		},
		{
			name: "Intel integrated not available for ML",
			info: &GPUInfo{Detected: true, Vendor: "intel"},
			want: false,
		},
		{
			name: "Apple Silicon available",
			info: &GPUInfo{Detected: true, Vendor: "apple"},
			want: true,
		},
		{
			name: "Not detected not available",
			info: &GPUInfo{Detected: false, Vendor: "nvidia"},
			want: false,
		},
		{
			name: "Empty info not available",
			info: &GPUInfo{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.IsGPUAvailable(); got != tt.want {
				t.Errorf("IsGPUAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Test cases for GPUInstructions ---

func TestGPUInstructions(t *testing.T) {
	tests := []struct {
		name    string
		info    *GPUInfo
		status  BackendStatus
		wantNil bool
		wantLen int // approximate
	}{
		{
			name:    "GPU already active returns nil",
			info:    &GPUInfo{Vendor: "nvidia", Name: "RTX 3060"},
			status:  BackendGPU,
			wantNil: true,
		},
		{
			name:    "NVIDIA CPU mode returns instructions",
			info:    &GPUInfo{Vendor: "nvidia", Name: "RTX 3060"},
			status:  BackendCPUMode,
			wantNil: false,
		},
		{
			name:    "AMD complex returns Docker instructions",
			info:    &GPUInfo{Vendor: "amd", Name: "RX 7900"},
			status:  BackendComplex,
			wantNil: false,
		},
		{
			name:    "No GPU returns CPU recommendation",
			info:    &GPUInfo{Detected: false},
			status:  BackendNone,
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GPUInstructions(tt.info, tt.status)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Error("expected instructions, got nil")
				return
			}
			if len(got) == 0 {
				t.Error("expected non-empty instructions")
			}
		})
	}
}
