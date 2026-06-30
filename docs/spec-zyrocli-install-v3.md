# Spec F1 — zyrocli install v3: TUI Moderna + GPU Cross-Platform

> Fecha: 17 Junio 2026
> Autor: Fase 0 → F1
> Estado: SPEC — lista para F2 (Design) y F3 (Implementation)
> Formato: C-I-O (Contexto, Interfaz, Output)

---

## C — Contexto

### Problema

El comando `zyrocli install` tiene dos problemas fundamentales:

1. **UX de los 90s**: Usa `fmt.Println` estático, cajas `╔════` que se rompen en terminales chicas, prompts con `fmt.Scanln` que no soportan flechas, y colores ANSI básicos. El proyecto YA tiene bubbletea v1.3.10 y lipgloss v1.1.0 pero no se usan en install.

2. **Detección GPU intrusiva y rota**: El código actual en `internal/setup/tui_launcher.go` y `scripts/install_tui.py` asume Arch Linux, ejecuta `sudo modprobe`, `sudo pacman -S` con `yay`, `pkill ollama`, y escribe archivos en `/etc/modules-load.d/`. Esto excluye al 90% de los usuarios (Debian/Ubuntu/Fedora/macOS/Windows) y deja el sistema en estado potencialmente inestable.

### Objetivos

| Objetivo | Métrica |
|----------|---------|
| TUI moderna con spinners animados | Cada paso crítico muestra spinner → ✓ |
| Layout responsive | Banner se adapta a <80, 80-99, ≥100 columnas |
| Menú interactivo Sí/No | Flechas ↑↓ + Enter, no `fmt.Scanln` |
| Detección GPU cross-platform | Funciona en Linux, macOS, Windows |
| No ejecutar comandos destructivos | Solo comandos read-only (lspci, nvidia-smi) |
| No modificar archivos del usuario | Solo imprimir instrucciones, no escribir .bashrc |
| Docker como fallback | Ofrecer docker run --gpus all cuando sea complejo |

### Alcance

| Incluye | No incluye |
|---------|------------|
| Refactor completo de `cmd/zyrocli/install.go` | Migración a bubbletea v2 (futuro) |
| Nuevo paquete `internal/tui/` con modelo bubbletea | Migración a charms v2 (bubbles, lipgloss) |
| Nuevo paquete `internal/hardware/` para GPU | Interfaz gráfica (GUI) |
| Detección NVIDIA, AMD, Apple Silicon, Intel | Soporte para TPUs o NPUs |
| Pasos: MCP, skills, config, Ollama, HelixDB | Modelos de IA adicionales |
| Resumen final con tabla lipgloss | |

### Dependencias a agregar

```bash
go get github.com/charmbracelet/bubbles@v1.0.0   # spinner, textinput, list
# Opcional (evaluar en F2):
# go get github.com/jaypipes/ghw@latest          # GPU detection
```

**CRÍTICO**: Bubbles v1 (github.com/charmbracelet/bubbles). NO v2. Bubbles v2 requiere migrar bubbletea a `charm.land/bubbletea/v2` y lipgloss a `charm.land/lipgloss/v2`.

---

## I — Interfaz (API y Estructuras)

### I.1 — Paquete `internal/tui` — Modelo Principal

#### `install.go` — Modelo bubbletea multi-step

```go
package tui

import (
    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/spinner"
    "github.com/charmbracelet/bubbles/progress"
    "github.com/charmbracelet/lipgloss"
)

// StepState representa el estado de un paso individual.
type StepState int

const (
    StepPending  StepState = iota // "   Instalando skills..."
    StepRunning                   // "⠋ Instalando skills..."
    StepDone                      // "✓ Instalando skills"
    StepError                     // "✗ Instalando skills — error message"
)

// InstallStep define un paso del instalador.
type InstallStep struct {
    // Name es el título descriptivo del paso (ej: "Extrayendo MCP tools")
    Name string

    // Action es la función que ejecuta el trabajo real.
    // Se ejecuta en una goroutine. Retorna error si falla.
    Action func() error

    // state se actualiza internamente: pending → running → done | error
    state StepState

    // err almacena el error si state == StepError
    err error
}

// InstallModel es el modelo principal de bubbletea para el instalador.
type InstallModel struct {
    // steps es la lista ordenada de pasos a ejecutar
    steps  []InstallStep

    // currentIdx es el índice del paso actualmente ejecutándose
    currentIdx int

    // spinner es el componente animado para el paso en ejecución
    spinner spinner.Model

    // progress es la barra de progreso global (opcional)
    progress progress.Model

    // width y height se actualizan vía WindowSizeMsg
    width  int
    height int

    // ollamaConfirm almacena la respuesta del menú interactivo
    // true = usuario quiere configurar Ollama
    ollamaConfirm *bool // nil = no preguntado aún

    // err acumula errores fatales del flujo completo
    err error

    // done indica que el instalador terminó
    done bool
}

// InstalStepMsg se envía cuando un paso termina su ejecución.
type InstalStepMsg struct {
    Index int
    Err   error
}

// --- Funciones públicas ---

// NewInstallModel crea el modelo con los pasos de instalación.
// Los pasos se inyectan para facilitar tests.
func NewInstallModel(steps []InstallStep) InstallModel { ... }

// RunInstall ejecuta el programa bubbletea con el modelo de instalación.
// Es la función principal que llama installCmd.RunE.
// Retorna error si el usuario canceló o hubo error fatal.
func RunInstall(steps []InstallStep) error { ... }
```

#### Ciclo de vida del modelo

```go
// Init: arranca el spinner y ejecuta el primer paso
func (m InstallModel) Init() tea.Cmd {
    m.steps[0].state = StepRunning
    return tea.Batch(m.spinner.Tick(), runStep(m.steps[0], 0))
}

// runStep ejecuta un paso en una goroutine y envía el resultado
func runStep(step InstallStep, index int) tea.Cmd {
    return func() tea.Msg {
        err := step.Action()
        return InstalStepMsg{Index: index, Err: err}
    }
}

// Update maneja los mensajes del programa bubbletea
func (m InstallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.progress.Width = msg.Width - 10

    case spinner.TickMsg:
        var cmd tea.Cmd
        m.spinner, cmd = m.spinner.Update(msg)
        return m, cmd

    case InstalStepMsg:
        if msg.Err != nil {
            m.steps[msg.Index].state = StepError
            m.steps[msg.Index].err = msg.Err
            // No detenemos el flujo — continuamos con el siguiente paso
            // a menos que sea un error fatal
        } else {
            m.steps[msg.Index].state = StepDone
        }
        m.currentIdx++
        if m.currentIdx >= len(m.steps) {
            m.done = true
            return m, tea.Quit
        }
        m.steps[m.currentIdx].state = StepRunning
        return m, runStep(m.steps[m.currentIdx], m.currentIdx)

    case tea.KeyMsg:
        // q o ctrl+c cancelan en cualquier momento
        if msg.String() == "q" || msg.String() == "ctrl+c" {
            m.done = true
            return m, tea.Quit
        }
    }

    return m, nil
}
```

#### `confirm.go` — Menú interactivo Sí/No

```go
package tui

import "github.com/charmbracelet/bubbletea"

// ConfirmResult representa la elección del usuario.
type ConfirmResult int

const (
    ConfirmYes ConfirmResult = iota
    ConfirmNo
)

// ConfirmModel es un modelo bubbletea para un menú Sí/No con flechas.
type ConfirmModel struct {
    // question es el texto a mostrar
    question string

    // cursor: 0 = Sí (default), 1 = No
    cursor int

    // result se setea cuando el usuario presiona Enter
    result ConfirmResult

    // done indica que el usuario ya eligió
    done bool
}

// NewConfirmModel crea un nuevo modelo de confirmación.
// question: texto a mostrar (ej: "¿Configurar Ollama?")
func NewConfirmModel(question string) ConfirmModel { ... }

// RunConfirm ejecuta el programa bubbletea y retorna la elección.
// Es un helper síncrono: bloquea hasta que el usuario elige.
func RunConfirm(question string) (bool, error) { ... }
```

#### `view.go` — Renderizado del layout responsive

```go
package tui

import "github.com/charmbracelet/lipgloss"

// BannerVariant determina qué banner renderizar según el ancho.
type BannerVariant int

const (
    BannerFull   BannerVariant = iota // ≥100 cols: ZYRO 3D + Zorro + subtítulo
    BannerMedium                      // 80-99 cols: Solo ZYRO 3D + subtítulo
    BannerSmall                       // <80 cols: Texto "ZyroCLI Installer"
)

// ResolveBanner determina la variante según el ancho de terminal.
func ResolveBanner(width int) BannerVariant {
    switch {
    case width >= 100:
        return BannerFull
    case width >= 80:
        return BannerMedium
    default:
        return BannerSmall
    }
}

// RenderBanner renderiza el banner según la variante.
// Usa lipgloss.Place para centrado absoluto.
func RenderBanner(variant BannerVariant, subtitle string) string { ... }

// RenderStepList renderiza la lista de pasos con su estado visual:
//   pending:  "    Instalando skills..."
//   running:  " ⠋ Instalando skills..." (con spinner animado)
//   done:     " ✓ Instalando skills" (verde)
//   error:    " ✗ Instalando skills — error" (rojo)
func RenderStepList(steps []InstallStep, s spinner.Model) string { ... }

// RenderSummary renderiza la tabla resumen final con lipgloss/table.
func RenderSummary(results []InstallStep) string { ... }
```

### I.2 — Paquete `internal/hardware` — Detección GPU

#### `gpu.go` — API pública de detección

```go
package hardware

import "runtime"

// GPUInfo encapsula la información de GPU detectada.
type GPUInfo struct {
    // Detected es true si se encontró GPU compatible.
    Detected bool `json:"detected"`

    // Vendor es el fabricante: "nvidia", "amd", "intel", "apple", "".
    Vendor string `json:"vendor"`

    // Name es el nombre del producto (ej: "GeForce RTX 3060").
    Name string `json:"name"`

    // Driver es el nombre o versión del driver (ej: "nvidia", "545.29.06").
    Driver string `json:"driver"`

    // OllamaUsing indica si Ollama está usando GPU.
    // Valores: "cuda", "rocm", "vulkan", "metal", "cpu", "unknown".
    OllamaBackend string `json:"ollama_backend"`

    // Platform es el SO: "linux", "darwin", "windows".
    Platform string `json:"platform"`
}

// BackendStatus describe el estado del backend GPU en Ollama.
type BackendStatus int

const (
    BackendUnknown  BackendStatus = iota
    BackendGPU                    // GPU ya activa en Ollama (L1)
    BackendCPUMode                // GPU detectada pero Ollama en CPU (L2)
    BackendComplex                // GPU detectada, instalación compleja (L3)
    BackendNone                   // Sin GPU detectada (L4)
)

// --- Funciones públicas ---

// DetectGPU detecta la GPU del sistema de forma read-only y cross-platform.
// No requiere privilegios de root. No modifica el sistema.
//
// Orden de detección por plataforma:
//   Linux:   ghw.GPU() → nvidia-smi → lspci → lshw
//   macOS:   sysctl → IOKit (via CGO, opcional)
//   Windows: ghw.GPU() → WMI
//
// Retorna GPUInfo completo o error si no se pudo detectar.
func DetectGPU() (*GPUInfo, error) { ... }

// CheckOllamaGPUStatus verifica si Ollama está usando GPU.
//
// Métodos de detección:
//   1. Leer OLLAMA_GPU_DRIVER del entorno
//   2. nvidia-smi para CUDA
//   3. journalctl para logs de ollama (Linux)
//   4. /proc/$(pgrep ollama)/maps para librerías ggml cargadas
//
// Retorna BackendStatus indicando el nivel de soporte.
func CheckOllamaGPUStatus() BackendStatus { ... }

// GPUInstructions genera las instrucciones para configurar Ollama con GPU.
// No ejecuta comandos — solo retorna strings con instrucciones.
//
// Dependiendo del BackendStatus:
//   L1 (BackendGPU):     nil (no se necesita hacer nada)
//   L2 (BackendCPUMode): instrucciones para configurar backend
//   L3 (BackendComplex): instrucciones para Docker
//   L4 (BackendNone):    recomendación de CPU mode
func GPUInstructions(info *GPUInfo, status BackendStatus) []string { ... }
```

#### `gpu_linux.go` — Implementación Linux

```go
//go:build linux
package hardware

func detectGPU() (*GPUInfo, error) {
    // Método 1: ghw.GPU() — detecta NVIDIA, AMD, Intel
    // Usa /sys/class/drm + PCI database
    //
    // Método 2: nvidia-smi — detecta NVIDIA con driver version
    //   $ nvidia-smi --query-gpu=name,driver_version --format=csv,noheader
    //
    // Método 3: lspci -d 10de: / lspci -d 1002:
    //   Detecta GPU presente sin drivers instalados
    //
    // Método 4: lshw -c display
    //   Fallback si no hay lspci
    ...
}
```

#### `gpu_darwin.go` — Implementación macOS

```go
//go:build darwin
package hardware

func detectGPU() (*GPUInfo, error) {
    // sysctl -n machdep.cpu.brand_string → detecta "Apple"
    // system_profiler SPDisplaysDataType → nombre GPU
    // Apple Silicon siempre tiene Metal automático
    ...
}
```

#### `gpu_windows.go` — Implementación Windows

```go
//go:build windows
package hardware

func detectGPU() (*GPUInfo, error) {
    // ghw.GPU() usa WMI Win32_VideoController
    // nvidia-smi disponible si NVIDIA drivers instalados
    // Ollama desktop app auto-detecta GPU en Windows
    ...
}
```

### I.3 — Integración: `cmd/zyrocli/install.go`

```go
package main

import (
    "github.com/secko/zyrocli/internal/tui"
    "github.com/secko/zyrocli/internal/hardware"
    "github.com/secko/zyrocli/internal/opencode"
    "github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
    Use:   "install",
    Short: "Install ZyroCLI ecosystem (config, skills, agents, MCP)",
    RunE: func(cmd *cobra.Command, args []string) error {
        // 1. Definir pasos de instalación
        steps := []tui.InstallStep{
            {Name: "Extrayendo MCP tools", Action: opencode.WriteMCPTools},
            {Name: "Instalando skills",    Action: opencode.WriteAllSkills},
            {Name: "Escribiendo configuración", Action: buildAndWriteConfig},
            // Paso interactivo: menú Ollama se maneja aparte
            {Name: "Verificando HelixDB",  Action: checkHelixDB},
        }

        // 2. Ejecutar TUI de instalación
        if err := tui.RunInstall(steps); err != nil {
            return fmt.Errorf("install: %w", err)
        }

        // 3. Menú interactivo ¿Configurar Ollama?
        ok, _ := tui.RunConfirm("¿Configurar modelos de IA local (Ollama)?")
        if ok {
            // Detectar GPU
            gpuInfo, _ := hardware.DetectGPU()
            status := hardware.CheckOllamaGPUStatus()

            if gpuInfo != nil && gpuInfo.Detected {
                // Mostrar info GPU + ofrecer configuración
                instrucciones := hardware.GPUInstructions(gpuInfo, status)
                for _, inst := range instrucciones {
                    cmd.Println(inst)
                }
            }

            // Lanzar install_tui.py para configuración detallada
            if err := setup.LaunchTUI(); err != nil {
                cmd.Printf("⚠ TUI: %v\n", err)
            }
        }

        // 4. Self-update + resumen
        // ...
    },
}
```

---

## O — Output (UX, Estados y Comportamiento)

### O.1 — Flujo de UX Completo

```
$ zyrocli install
│
├── [0.0s] BANNER RESPONSIVE
│   ┌──────────────────────────────────────────┐
│   │                                          │
│   │    █████   █   █   ████      ███         │
│   │     ░░█░░   █ █ ░  █░░░█    █ ░░█        │
│   │      █░░░░   █ ░ ░ ████░░   █░ ░█░       │
│   │     █ ░ ░    █░ ░  █░░█░ ░  █░░ █░░      │
│   │    █████     █░░   █░░░█░    ███ ░░       │
│   │     ░░░░░     ░░    ░░  ░     ░░░ ░       │
│   │                                          │
│   │         ZyroCLI — Instalación            │
│   │                                          │
│   └──────────────────────────────────────────┘
│
├── [0.5s] PASO 1: ⠋ Extrayendo MCP tools...
│   → [2.0s] ✓ MCP tools extraídas
│
├── [2.0s] PASO 2: ⠋ Instalando skills...
│   → [3.0s] ✓ 12 skills instaladas
│
├── [3.0s] PASO 3: ⠋ Escribiendo configuración...
│   → [3.5s] ✓ Config escrito
│
├── [3.5s] MENÚ INTERACTIVO (↑↓ + Enter)
│   ┌──────────────────────────────────────────┐
│   │  ¿Configurar modelos de IA local         │
│   │  (Ollama)?                               │
│   │                                          │
│   │  ● Sí  (recomendado)                     │
│   │  ○ No                                    │
│   │                                          │
│   │  [Enter] confirmar  [↑/↓] navegar        │
│   └──────────────────────────────────────────┘
│   │
│   ├── [Sí] → Detectar GPU
│   │   ⠋ Detectando GPU...
│   │   ┌──────────────────────────────────────┐
│   │   │  ✓ GPU detectada:                    │
│   │   │    NVIDIA GeForce RTX 3060           │
│   │   │    Driver: 545.29.06                 │
│   │   │                                     │
│   │   │  ⚎ Ollama usa CPU actualmente       │
│   │   │                                     │
│   │   │  Recomendación:                     │
│   │   │  export OLLAMA_GPU_DRIVER=cuda      │
│   │   │  Agrega eso a ~/.bashrc             │
│   │   │                                     │
│   │   │  ¿Quieres configuración detallada?   │
│   │   │  ● Sí (abre asistente)               │
│   │   │  ○ No, ya sé lo que hago             │
│   │   └──────────────────────────────────────┘
│   │   │
│   │   └── [Sí] → Lanzar install_tui.py
│   │       (hereda stdin/stdout, experiencia Rich)
│   │
│   └── [No] → Saltar configuración Ollama
│
├── [4.0s] PASO 4: ⠋ Verificando HelixDB...
│   → [4.5s] ✓ HelixDB reachable
│
├── [4.5s] PASO 5: Self-update
│   → [5.0s] ✓ zyrocli actualizado
│
└── [5.0s] RESUMEN FINAL
    ┌──────────────────────────────────────────┐
    │  Resumen de Instalación                  │
    │                                          │
    │  ✓ Extrayendo MCP tools                  │
    │  ✓ Instalando skills        [12]         │
    │  ✓ Escribiendo configuración             │
    │  ✓ Verificando HelixDB                   │
    │  ✓ self-update                           │
    │                                          │
    │  GPU: NVIDIA GeForce RTX 3060 (CUDA)     │
    │  Ollama: ✓ Configurado                   │
    │                                          │
    │  🎉 Instalación completada               │
    │                                          │
    │  Next steps:                             │
    │    zyro onboard .                        │
    │    zyrocli profile tui                   │
    └──────────────────────────────────────────┘
```

### O.2 — Variantes de Banner Responsive

| Variante | Ancho mínimo | Contenido | Origen |
|----------|-------------|-----------|--------|
| `BannerFull` | ≥100 cols | ZYRO 3D (brand.txt) + Zorro Hacker (logo.txt) lado a lado + subtítulo | `RenderFullBanner()` en brand.go |
| `BannerMedium` | 80-99 cols | Solo ZYRO 3D centrado + subtítulo | `RenderWelcome()` en brand.go |
| `BannerSmall` | <80 cols | Texto "ZyroCLI Installer" + subtítulo sin ASCII art | **NUEVO**: texto plano estilizado |

Todas las variantes se renderizan con:
```go
lipgloss.Place(width, bannerHeight, lipgloss.Center, lipgloss.Center, content)
```

### O.3 — Estados de cada paso

```
Visualmente:

PENDING:   "    Instalando skills..."       (gris, sin icono)
RUNNING:   " ⠋ Instalando skills..."        (spinner violeta #7C3AED)
DONE:      " ✓ Instalando skills"           (checkmark verde #10B981)
ERROR:     " ✗ Instalando skills            (X rojo #EF4444)
                └ error message"

Transiciones:
  PENDING → RUNNING (cuando le toca el turno)
  RUNNING → DONE    (cuando Action() retorna nil)
  RUNNING → ERROR   (cuando Action() retorna error)
  ERROR → RUNNING   (NO — seguimos al siguiente paso)
```

### O.4 — Estrategia de Error

| Escenario | Comportamiento |
|-----------|----------------|
| Error en paso (Action retorna error) | Marcar paso como `StepError`, mostrar mensaje en rojo, continuar con siguiente paso |
| Error fatal (no se puede escribir config) | Terminar TUI, retornar error a cobra |
| Usuario presiona `q` o `ctrl+c` | Terminar TUI con `tea.Quit`, NO retornar error |
| GPU no detectada | Mostrar "CPU mode — modelos pequeños funcionan bien" |
| Ollama no instalado | Mostrar instrucciones: `curl -fsSL https://ollama.com/install.sh | sh` |
| Docker no instalado (fallback L3) | Mostrar instrucciones: `No hay Docker. Opción manual: [enlace]` |

---

## Plan de Migración de `install_tui.py`

### Qué se REESCRIBE en Go (pasa a `internal/hardware/gpu.go`)

| Módulo Python | Líneas | Nueva ubicación Go | Estado |
|---------------|--------|-------------------|--------|
| `paso5_gpu()`: detección NVIDIA/AMD/Apple | ~200 | `internal/hardware/gpu.go` | **PASA A GO** |
| `_check_ollama_backend()`: backend detection | ~60 | `internal/hardware/gpu.go` | **PASA A GO** |
| `_check_amdkfd_module()`: ROCm check | ~10 | `internal/hardware/gpu.go` | **PASA A GO** |
| `_extraer_nombre_amd()`: parseo ROCm | ~15 | `internal/hardware/gpu.go` | **PASA A GO** |
| `_detectar_gpu_via_ollama()`: logs parsing | ~30 | `internal/hardware/gpu.go` | **PASA A GO** |
| `_get_aur_helper()`: AUR detection | ~5 | **SE ELIMINA** (no más Arch-only) | ❌ ELIMINAR |
| Lógica de instalación con yay/pacman | ~60 | **SE ELIMINA** (no más sudo) | ❌ ELIMINAR |
| GPU decision tree + instalación | ~40 | `GPUInstructions()` en Go | **PASA A GO** |

### Qué SE MANTIENE en Python (scripts/install_tui.py)

| Módulo Python | Líneas | Razón |
|---------------|--------|-------|
| `paso1_bienvenida()` | ~45 | Se mantiene pero se migrará en futura iteración |
| `paso2_ollama()` | ~115 | Verificación de Ollama vía API HTTP — práctico en Python |
| `paso3_embeddings()` | ~55 | Menú selección modelo + pull vía API Ollama |
| `paso4_chat()` | ~60 | Menú selección modelo chat + pull |
| `paso6_probar_embeddings()` | ~80 | Test de embedding vía API Ollama |
| `paso7_probar_chat()` | ~80 | Test de chat vía API Ollama |
| `paso8_resumen()` | ~140 | Escribir config.yaml + resumen |
| `_instalar_modelo()` | ~60 | Pull de modelo con barra de progreso |
| Helpers HTTP | ~50 | Cliente HTTP para API Ollama |

### Plan en 3 Fases

```
Fase 1 (esta):  TUI Go + GPU detection en Go + install_tui.py simplificado
                └── install.go usa bubbletea, llama a Python solo para config Ollama

Fase 2 (futura): Migrar modelo selection (pasos 3,4) a Go con bubbles/list
                 └── install_tui.py se reduce a helpers Ollama API

Fase 3 (futura): Migrar pull + test de modelos a Go con bubbles/progress
                 └── install_tui.py se elimina por completo
```

### Dependencias entre pasos

```
         ┌──────────────┐
         │  extractMCP  │ (independiente)
         └──────┬───────┘
                │
         ┌──────▼───────┐
         │ installSkills │ (independiente)
         └──────┬───────┘
                │
         ┌──────▼───────┐
         │  writeConfig  │ (requiere skills escritas)
         └──────┬───────┘
                │
         ┌──────▼───────┐
         │  menuOllama   │ (interactivo, pregunta al usuario)
         └──────┬───────┘
         ┌──────▼───────┐
         │ detectGPU     │ (solo si menuOllama == Sí)
         └──────┬───────┘
         ┌──────▼───────┐
         │ launchTUI     │ (solo si usuario quiere config detallada)
         └──────┬───────┘
                │
         ┌──────▼───────┐
         │  verifyHelix  │ (independiente)
         └──────┬───────┘
                │
         ┌──────▼───────┐
         │  selfUpdate   │ (independiente)
         └──────┬───────┘
                │
         ┌──────▼───────┐
         │   summary     │ (todo completado)
         └──────────────┘
```

---

## Archivos Resultantes

| Archivo | Acción | LOC estimado |
|---------|--------|-------------|
| `cmd/zyrocli/install.go` | REFACTOR — usar `tui.RunInstall()` | ~100 |
| `internal/tui/install.go` | NUEVO — modelo bubbletea multi-step | ~200 |
| `internal/tui/confirm.go` | NUEVO — menú interactivo Sí/No | ~100 |
| `internal/tui/view.go` | NUEVO — renderizado responsive + step list | ~150 |
| `internal/hardware/gpu.go` | NUEVO — API pública de detección GPU | ~100 |
| `internal/hardware/gpu_linux.go` | NUEVO — impl Linux | ~150 |
| `internal/hardware/gpu_darwin.go` | NUEVO — impl macOS | ~50 |
| `internal/hardware/gpu_windows.go` | NUEVO — impl Windows | ~50 |
| `internal/setup/tui_launcher.go` | SIMPLIFICAR — quitar GPU detection | ~30 |
| `scripts/install_tui.py` | LIMPIAR — quitar pasos 5 (GPU) | ~-200 |
| `go.mod` | AGREGAR `bubbles v1.0.0` | +1 línea |
| **Total neto** | | **~730 nuevas líneas Go** |

---

## Criterios de Aceptación

1. `zyrocli install` muestra banner responsive que se adapta al ancho de terminal
2. Cada paso muestra spinner animado → checkmark al completar
3. El prompt "¿Configurar Ollama?" se navega con flechas ↑↓ y Enter
4. `internal/hardware.DetectGPU()` detecta NVIDIA, AMD y Apple Silicon sin sudo
5. No se ejecuta `sudo`, `modprobe`, ni se escriben archivos del sistema
6. Si el usuario elige "Sí" en el menú Ollama, se lanza `install_tui.py` con stdin/stdout heredados
7. Al final se muestra resumen con tabla lipgloss
8. `go test ./internal/tui/...` pasa
9. `go test ./internal/hardware/...` pasa
