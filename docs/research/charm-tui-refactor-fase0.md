# Fase 0 — Refactor TUI: Integración Charmbracelet para `zyrocli install`

> Fecha: 17 Junio 2026
> Estado: Investigación completa — lista para F1 (Spec) y F2 (Design)

## Resumen Ejecutivo

Refactorizar `zyrocli install` de `fmt.Println` estático a una experiencia TUI moderna con el stack Charmbracelet (bubbletea + lipgloss + bubbles). El proyecto YA tiene bubbletea v1.3.10 y lipgloss v1.1.0 como dependencias.

## Stack Actual vs Propuesto

| Aspecto | Actual (v2) | Propuesto (v3) |
|---------|-------------|----------------|
| Output | `fmt.Println` estático | `bubbletea` TUI con modelo/update/view |
| Pasos | Texto plano "Extracting..." | Spinner animado `⠋ → ✓` con bubbles/spinner |
| Menú Y/n | `fmt.Scanln` síncrono | Menú interactivo ↑↓ flechas con Enter |
| Banner | Cajas `╔════` feas | Lipgloss centrado responsive |
| Layout | Fijo, se rompe en terminal chica | Responsive: full banner ≥100cols, small <80cols |
| Colores | ANSI básico | Paleta Zyro: Violeta #7C3AED, Verde #10B981 |
| Python TUI | `install_tui.py` con Rich | Go nativo + llama a Python solo para configuración Ollama |

## Dependencias

### Ya instaladas (go.mod)
- `github.com/charmbracelet/bubbletea v1.3.10` — framework TUI
- `github.com/charmbracelet/lipgloss v1.1.0` — estilos
- `golang.org/x/sys v0.36.0` (indirecta) — syscalls

### A agregar
- `github.com/charmbracelet/bubbles v1.0.0` — componentes UI (spinner, textinput, list, progress)
  - **CRÍTICO**: Usar v1, NO v2. Bubbles v2 requiere bubbletea v2 (`charm.land/bubbletea/v2`) y lipgloss v2 (`charm.land/lipgloss/v2`)
- `golang.org/x/term` — detección de ancho de terminal (alternativa: `tea.WindowSizeMsg` incluido en bubbletea)

### Opcionales (evaluar en F1)
- `charm.land/huh/v2` — formularios y confirm (requeriría migrar a bubbletea v2)
- `github.com/jaypipes/ghw` — detección de hardware GPU

## APIs Clave de Bubbles v1

### Spinner
```go
import "github.com/charmbracelet/bubbles/spinner"

s := spinner.New()
s.Spinner = spinner.Dot
s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))

// Init: return tea.Batch(m.spinner.Tick())
// Update: case spinner.TickMsg: s, cmd = s.Update(msg)
// View: s.View() → "⠋ ", "⠙ ", etc.
```

Spinners built-in: `Line`, `Dot`, `MiniDot`, `Jump`, `Pulse`, `Points`, `Globe`, `Moon`, `Monkey`, `Meter`, `Hamburger`, `Ellipsis`

### TextInput
```go
ti := textinput.New()
ti.Placeholder = "Escribe algo..."
ti.Prompt = "→ "
ti.CharLimit = 100
ti.Width = 40
ti.Focus()  // tea.Cmd
// Validate, SetSuggestions, EchoMode, Styles completos
```

### List (menú seleccionable)
```go
delegate := list.NewDefaultDelegate()
l := list.New(items, delegate, width, height)
l.Title = "Selecciona una opción"
// Navegación built-in: ↑↓, k/j, g/G, pgup/pgdn, / para filtrar
// l.SelectedItem(), l.Index(), l.SetItems()
```

### Progress Bar
```go
p := progress.New(
    progress.WithDefaultGradient(),
    progress.WithWidth(50),
)
cmd := p.SetPercent(0.75) // animado
p.View()   // animado
p.ViewAs(0.75) // estático
```

### Help
```go
h := help.New()
h.Width = 80
h.View(myKeyMap{})
// key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
```

## Patrón de Instalador Multi-Step con Spinner

Basado en `charmbracelet/bubbletea/examples/package-manager`:

```go
type installStep struct {
    Name   string
    Action func() error
}

type installModel struct {
    steps   []installStep
    index   int
    spinner spinner.Model
    done    bool
}

func (m installModel) Init() tea.Cmd {
    return tea.Batch(m.spinner.Tick(), runStep(m.steps[0]))
}

func (m installModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case stepDoneMsg:
        if msg.err != nil { /* error handling */ }
        m.index++
        if m.index >= len(m.steps) { m.done = true; return m, tea.Quit }
        return m, runStep(m.steps[m.index])
    case spinner.TickMsg:
        var cmd tea.Cmd
        m.spinner, cmd = m.spinner.Update(msg)
        return m, cmd
    }
    return m, nil
}

func runStep(step installStep) tea.Cmd {
    return func() tea.Msg {
        return stepDoneMsg{err: step.Action()}
    }
}
```

## Layout Responsive con Lipgloss

Detectar ancho vía `tea.WindowSizeMsg`:

```go
case tea.WindowSizeMsg:
    m.width = msg.Width
```

Tres variantes de banner:

| Ancho | Banner | Contenido |
|-------|--------|-----------|
| ≥100 | Full | ZYRO 3D + Zorro Hacker lado a lado + subtítulo |
| 80-99 | Medium | Solo ZYRO 3D + subtítulo |
| <80 | Small | Texto "ZyroCLI Installer" + subtítulo |

Centrado siempre con:
```go
lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
```

Ya existe en `internal/tui/brand.go`:
- `RenderFullBanner(subtitle)` — ZYRO 3D + Zorro Hacker
- `RenderWelcome(subtitle)` — Solo ZYRO 3D
- Estilos: colorVioleta (#7C3AED), colorVerde (#10B981), colorGris (#52525B)

## Menú Interactivo Sí/No

Sin dependencias extra (solo bubbletea):

```go
type choice int
const (
    choiceYes choice = iota
    choiceNo
)

type confirmModel struct {
    cursor int  // 0 = Sí, 1 = No
    choice choice
    done   bool
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k", "down", "j":
            m.cursor = (m.cursor + 1) % 2
        case "enter":
            if m.cursor == 0 { m.choice = choiceYes } else { m.choice = choiceNo }
            m.done = true
            return m, tea.Quit
        }
    }
    return m, nil
}
```

## Flujo de UX Propuesto para `zyrocli install`

```
1. BANNER RESPONSIVE (ZYRO 3D + Zorro o texto según ancho)
2. PASO 1: ⠋ Extrayendo MCP tools... → ✓ MCP tools extraídas
3. PASO 2: ⠋ Instalando skills... → ✓ N skills instaladas
4. PASO 3: ⠋ Escribiendo configuración... → ✓ Config escrito
5. MENÚ: ¿Configurar Ollama? [Sí/No] (flechas ↑↓ + Enter)
   ├── No → skip
   └── Sí → Lanzar install_tui.py (hereda stdin/stdout)
6. PASO 4: ⠋ Verificando HelixDB... → ✓ HelixDB reachable
7. RESUMEN con tabla lipgloss
```

## Archivos a Modificar/Crear

| Archivo | Acción |
|---------|--------|
| `cmd/zyrocli/install.go` | Refactorizar RunE para usar bubbletea Program |
| `internal/tui/install.go` | **NUEVO** — modelo bubbletea para instalación multi-step |
| `internal/tui/spinner.go` | **NUEVO** — helpers de spinner para pasos |
| `internal/tui/confirm.go` | **NUEVO** — menú interactivo Sí/No |
| `internal/tui/brand.go` | Mejorar: integrar WindowSizeMsg para responsive |
| `internal/tui/assets/brand.txt` | Crear variante small (texto ZYRO simple) |
| `go.mod` | Agregar `github.com/charmbracelet/bubbles v1.0.0` |
| `scripts/install_tui.py` | Limpiar: quitar duplicación de GPU detection (pasa a Go) |

## Migración a bubbletea v2 (Futuro)

Si se decide migrar todo el stack a v2:
- Cambiar imports: `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`
- `tea.KeyMsg` → `tea.KeyPressMsg`
- Getters/setters en lugar de fields públicos
- Manejo explícito de dark/light mode
- `tea.NewProgram` cambia firma ligeramente
