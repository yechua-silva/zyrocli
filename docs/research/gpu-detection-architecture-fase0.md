# Fase 0 — Arquitectura GPU Cross-Platform para Ollama

> Fecha: 17 Junio 2026
> Estado: Investigación completa — lista para F1 (Spec) y F2 (Design)

## Resumen Ejecutivo

Rediseñar el módulo de detección y configuración de GPU en `zyrocli install`. El enfoque actual es intrusivo (fuerza Arch Linux, usa sudo/modprobe, asume yay) y deja el sistema en estado roto. Proponemos un diseño seguro, read-only y agnóstico al SO.

## Vulnerabilidades del Enfoque Actual

| # | Problema | Riesgo | Impacto |
|---|----------|--------|---------|
| 1 | `sudo pacman -S ollama-rocm-bin` con `yay` | Solo funciona en Arch, falla en otras distros | Exclusión de 90% de usuarios |
| 2 | `sudo modprobe amdkfd` | Modifica kernel en caliente, puede crashear | Inestabilidad del sistema |
| 3 | `pkill ollama` | Mata procesos del usuario sin preguntar | Pérdida de datos |
| 4 | `echo amdkfd | sudo tee /etc/modules-load.d/` | Modifica configuración del sistema permanentemente | System pollution |
| 5 | Asume `yay` o `paru` presente | Fallo en Debian/Ubuntu/Fedora/macOS | UX broken |
| 6 | No detecta Apple Silicon | macOS siempre en CPU mode | 30% de usuarios sin GPU |

## Principios de Diseño para la Nueva Arquitectura

1. **Read-only First**: Detectar GPU solo con comandos de lectura (`lspci`, `nvidia-smi`, `ls /dev/kfd`)
2. **No modificar sistema**: Nunca escribir archivos del usuario o del sistema automáticamente
3. **Guided Instructions**: Mostrar comandos exactos que el usuario puede copiar-pegar
4. **Graceful Fallback**: Siempre ofrecer CPU mode o Docker como alternativa
5. **Cross-platform**: Linux, macOS y Windows first-class

## Librería Recomendada: `jaypipes/ghw`

**Repo**: `github.com/jaypipes/ghw`
**Stars**: ~1.9k
**Licencia**: Apache 2.0
**Plataformas**: Linux ✅ FULL, Windows ✅ FULL, macOS ⚠️ parcial (GPU no soportada)

### API Principal

```go
gpu, err := ghw.GPU()  // retorna *GPUInfo
for _, card := range gpu.GraphicCards {
    // card.DeviceInfo.Vendor.ID → "10de" (NVIDIA), "1002" (AMD), "8086" (Intel)
    // card.DeviceInfo.Vendor.Name → "NVIDIA Corporation"
    // card.DeviceInfo.Product.Name → "GP107 [GeForce GTX 1050 Ti]"
    // card.DeviceInfo.Driver → "nvidia", "amdgpu", "i915"
}
```

### Identificación de Vendor

```go
switch card.DeviceInfo.Vendor.ID {
case "10de": // NVIDIA
case "1002": // AMD
case "8086": // Intel (incluye Apple Silicon via PCI? No — Apple no usa PCI)
}
```

### Soporte por Plataforma

| Plataforma | ghw GPU | Alternativa |
|------------|---------|-------------|
| Linux | ✅ FULL — sysfs + PCI | `nvidia-smi` para driver version |
| Windows | ✅ FULL — WMI `Win32_VideoController` | También da DriverVersion |
| macOS (Apple Silicon) | ❌ Stub | `sysctl machdep.cpu.brand_string` + IOKit |

### Limitaciones

- En Linux, `Driver` solo da nombre (`"nvidia"`), NO versión (ej: 545.29.06)
- Para versión exacta de driver: `nvidia-smi --query-gpu=driver_version --format=csv,noheader`
- macOS GPU no soportada en ghw — usar detección nativa

## Cómo Detecta GPU el Script Oficial de Ollama

El script `curl -fsSL https://ollama.com/install.sh | sh` usa:

```
1. nvidia-smi → NVIDIA drivers instalados y funcionales
2. lspci -d 10de: → GPU NVIDIA presente (sin drivers)
3. lspci -d 1002: → GPU AMD presente 
4. lshw -c display → fallback si no hay lspci
5. /etc/nv_tegra_release → NVIDIA Jetson
6. WSL2 → nvidia-smi en Windows Subsystem for Linux
```

Para AMD, descarga automática: `ollama-linux-${ARCH}-rocm` (binario compilado con ROCm).
Para NVIDIA, instala drivers CUDA vía gestor de paquetes del sistema (apt, yum, dnf).

## AMD ROCm vs Vulkan — Estado Actual

| Aspecto | ROCm | Vulkan |
|---------|------|--------|
| Rendimiento | ⭐ Alto | 👍 Bueno (10-20% menos) |
| Facilidad de instalación | ❌ Compleja | ✅ Simple |
| Dependencias | amdkfd, HIP, rocminfo | Drivers Vulkan existentes |
| Configuración | `HSA_OVERRIDE_GFX_VERSION` | Automático |
| Soporte Ollama | ≥0.5.x | ≥0.5.x |
| `OLLAMA_GPU_DRIVER` | `rocm` | `vulkan` |

**Recomendación**: Vulkan como default, ROCm como opción avanzada (solo si usuario explícitamente pide más rendimiento).

## Apple Silicon

- Ollama en macOS usa **Metal Performance Shaders** automáticamente
- No requiere configuración adicional
- Detectar con: `sysctl -n machdep.cpu.brand_string` contiene "Apple"
- ghw NO soporta GPU en macOS
- No instalar nada extra

## Árbol de Decisión Completo

```
DETECTAR GPU
│
├── macOS
│   ├── sysctl → "Apple" → Metal ✅ (automático, no hacer nada)
│   └── Intel → CPU ⚠️ (sin GPU dedicada)
│
├── Linux
│   ├── L1: GPU Ready
│   │   ├── nvidia-smi funciona → CUDA ✅
│   │   └── rocminfo + /dev/kfd → ROCm ✅
│   │
│   ├── L2: GPU Detectada, Ollama en CPU
│   │   ├── NVIDIA → "Instala drivers CUDA: [comandos para tu distro]"
│   │   ├── AMD → "1) Vulkan (fácil): export OLLAMA_GPU_DRIVER=vulkan"
│   │   │           "2) ROCm (avanzado): export OLLAMA_GPU_DRIVER=rocm"
│   │   │           "   + sudo modprobe amdkfd (si no está)"
│   │   │           "   + sudo usermod -aG render,video $USER"
│   │   └── Intel → CPU mode
│   │
│   ├── L3: GPU Detectada, instalación compleja → Docker
│   │   ├── NVIDIA: docker run --gpus all
│   │   └── AMD: docker run --device /dev/kfd --device /dev/dri
│   │
│   └── L4: Sin GPU detectada → CPU mode
│
└── Windows
    └── Ollama app auto-detecta → solo verificar
```

## Flujo de UX Propuesto

```
PASO 5: Detectar GPU

  ┌─────────────────────────────────────┐
  │  Detectando hardware de GPU...      │
  │                                     │
  │  ─────────────────────────────────  │
  │                                     │
  │  ⠋ Buscando GPU...                  │
  │                                     │
  └─────────────────────────────────────┘

  ┌─────────────────────────────────────┐
  │  ✓ GPU detectada:                   │
  │    NVIDIA GeForce RTX 3060          │
  │    Driver: 545.29.06                │
  │                                     │
  │  ¿Quieres configurar Ollama para    │
  │  usar esta GPU?                     │
  │                                     │
  │  ● Sí, configurar automáticamente   │
  │  ○ No, usar CPU                     │
  │  ○ Mostrar instrucciones manuales   │
  │                                     │
  │  [Enter] seleccionar  [↑/↓] navegar │
  └─────────────────────────────────────┘

  ┌─────────────────────────────────────┐
  │  ✓ Ollama configurado con GPU       │
  │  (CUDA backend activo)              │
  │                                     │
  │  💡 Si cambias de terminal,         │
  │  agrega esto a tu ~/.bashrc:        │
  │  export OLLAMA_GPU_DRIVER=cuda      │
  └─────────────────────────────────────┘
```

## Estrategias de Fallback (4 Niveles)

| Nivel | Condición | Acción |
|-------|-----------|--------|
| **L1** | GPU detectada + Ollama ya la usa | ✅ Mostrar "GPU activa" |
| **L2** | GPU detectada, Ollama en CPU | Ofrecer configuración guiada (read-only) |
| **L3** | GPU detectada, instalación compleja | Ofrecer Docker como alternativa |
| **L4** | Sin GPU detectada | CPU mode, recomendar modelos pequeños |

## Docker como Alternativa Segura

**NVIDIA:**
```bash
docker run -d --gpus all \
  -v ollama:/root/.ollama \
  -p 11434:11434 \
  --name ollama \
  ollama/ollama
```

**AMD (Vulkan):**
```bash
docker run -d \
  --device /dev/kfd --device /dev/dri \
  -v ollama:/root/.ollama \
  -p 11434:11434 \
  -e OLLAMA_GPU_DRIVER=vulkan \
  --name ollama \
  ollama/ollama
```

**Ventajas**: No modifica el sistema, fácil de desinstalar (`docker rm -f ollama`), funciona en cualquier distro.

## Variables de Entorno

```bash
# Forzar backend GPU en Ollama
export OLLAMA_GPU_DRIVER=cuda     # NVIDIA
export OLLAMA_GPU_DRIVER=rocm     # AMD ROCm
export OLLAMA_GPU_DRIVER=vulkan   # AMD/Intel Vulkan
export OLLAMA_GPU_DRIVER=cpu      # Forzar CPU

# Seleccionar GPU específica (multi-GPU)
export CUDA_VISIBLE_DEVICES=0     # Solo primera GPU NVIDIA
export HIP_VISIBLE_DEVICES=0      # Solo primera GPU AMD

# AMD ROCm override (para GPUs no soportadas oficialmente)
export HSA_OVERRIDE_GFX_VERSION=8.0.3  # ROCm v7+ para RDNA3
```

**Regla**: Nunca modificar .bashrc/.zshrc del usuario automáticamente. Imprimir instrucciones y dejar que el usuario decida.

## Código Go de Detección

```go
type GPUInfo struct {
    Detected   bool   `json:"detected"`
    Vendor     string `json:"vendor"` // "nvidia", "amd", "intel", "apple", ""
    Name       string `json:"name"`
    Driver     string `json:"driver"` // nombre o versión
    OllamaUsing string `json:"ollama_using"` // backend actual
}

func DetectGPU() GPUInfo {
    // Intentar con ghw
    if gpu, err := ghw.GPU(); err == nil {
        for _, card := range gpu.GraphicsCards {
            if card.DeviceInfo == nil { continue }
            info := GPUInfo{
                Detected: true,
                Name: card.DeviceInfo.Product.Name,
                Driver: card.DeviceInfo.Driver,
            }
            switch card.DeviceInfo.Vendor.ID {
            case "10de": info.Vendor = "nvidia"
            case "1002": info.Vendor = "amd"
            case "8086": info.Vendor = "intel"
            }
            return info
        }
    }

    // Fallback macOS
    if runtime.GOOS == "darwin" {
        if output, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
            if strings.Contains(strings.ToLower(string(output)), "apple") {
                return GPUInfo{Detected: true, Vendor: "apple", Name: "Apple Silicon"}
            }
        }
    }

    // Fallback nvidia-smi
    if output, err := exec.Command("nvidia-smi", "--query-gpu=name,driver_version", "--format=csv,noheader").Output(); err == nil {
        parts := strings.Split(strings.TrimSpace(string(output)), ",")
        if len(parts) >= 2 {
            return GPUInfo{Detected: true, Vendor: "nvidia", Name: parts[0], Driver: parts[1]}
        }
    }

    // Fallback lspci
    // ...

    return GPUInfo{Detected: false}
}

func CheckOllamaBackend() string {
    // Leer OLLAMA_GPU_DRIVER de entorno
    if driver := os.Getenv("OLLAMA_GPU_DRIVER"); driver != "" {
        return driver
    }
    // Verificar logs de ollama
    // ...
    return "unknown"
}
```

## Archivos a Modificar/Crear

| Archivo | Acción |
|---------|--------|
| `internal/tui/gpu.go` | **NUEVO** — modelo bubbletea para detección GPU |
| `internal/tui/gpu_detect.go` | **NUEVO** — lógica de detección cross-platform |
| `internal/setup/tui_launcher.go` | Simplificar: quitar GPU detection (pasa a Go) |
| `scripts/install_tui.py` | Limpiar: pasos 5 (GPU) se manejan en Go ahora |
| `go.mod` | Evaluar agregar `github.com/jaypipes/ghw` |
