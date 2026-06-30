# Design: ux-tui-zorro-naranja

## Technical Approach

Cuatro requerimientos independientes que se implementan secuencialmente: (1) nuevo logo naranja desde `ascii-art-logo.txt` → asset embebido + `RenderNewLogo()` en `brand.go` + reemplazo de arte en `zorro-logo.tsx`; (2) fix clear screen antes de brand duplicado en `runSetupFlow()` y `runAutostartFlow()`; (3) clear screen sistemático antes de cada menú, flujo y confirmación; (4) nuevo `MenuItem` "Acerca de ZyroCLI" con `runAboutFlow()`. Sin import cycles, sin cambios en API pública exportada, solo funciones nuevas.

## Architecture Decisions

### Decision: Asset embedding pattern

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Leer archivo en runtime | Sencillo pero el path es relativo; se rompe si el binario se mueve | ❌ |
| `//go:embed` en brand.go | Mismo patrón que `brand.txt` y `logo.txt` existentes. Compilación estática, sin dependencias en runtime | ✅ |

**Rationale**: El proyecto ya usa `//go:embed` para `assets/brand.txt` y `assets/logo.txt`. El nuevo logo sigue exactamente el mismo patrón: asset en `assets/logo-new.txt`, variable `logoNewArtRaw string`, sanitización en `init()`, estilo `logoNewStyle` y función `RenderNewLogo()`.

### Decision: logoNewStyle — reusar colorNaranja

**Choice**: `logoNewStyle = lipgloss.NewStyle().Foreground(colorNaranja).Bold(true)` (mismo que `brandStyle`)
**Alternatives considered**: Crear un color naranja diferente o compartir `brandStyle` directamente
**Rationale**: El spec pide naranja como `brandStyle`. `brandStyle` es una variable de paquete que otros tests/consumidores podrían depender. Mejor crear `logoNewStyle` idéntica pero independiente para no acoplar.

### Decision: Clear screen con escape ANSI

**Choice**: `fmt.Print("\033[2J\033[H")` — secuencia POSIX estándar
**Alternatives considered**: Usar `cmd.Clear` de bubbletea, o el paquete `os/exec` con `clear`
**Rationale**: El clear screen ocurre _antes_ de lanzar bubbletea programs (menú, confirmaciones). No hay model en ese punto. El escape ANSI es universal, sin dependencias, y ya se usa en otras partes del ecosistema Go. La secuencia `\033[2J` borra la pantalla y `\033[H` mueve el cursor al inicio.

### Decision: runAboutFlow() en cmd/zyrocli/main.go

**Choice**: Función nueva en `main.go` que llama `tui.RenderBrand()` + texto formateado con lipgloss
**Alternatives considered**: Mover el texto a `internal/tui/` como función exportada
**Rationale**: El texto del about es específico de la UX del CLI, no de la librería TUI. Ponerlo en `main.go` evita inflar `internal/tui/` con copy de UI que solo se usa en un lugar. El branding (logo) ya viene de `tui.RenderBrand()`.

## Data Flow

```
handleMenu() loop (c/ iteración):
  │
  ├─ fmt.Print("\033[2J\033[H")          ← clear screen (REQ-3)
  │
  ├─ choice := tui.RunMainMenu()
  │
  └─ switch choice:
       ├─ "install"  → runInstallFlow()
       │                  ├─ fmt.Print("\033[2J\033[H")   ← clear (REQ-3)
       │                  ├─ installCmd.RunE(...)
       │                  ├─ tui.PrintSuccess(...)
       │                  ├─ fmt.Print("\033[2J\033[H")   ← clear antes de confirm (REQ-3)
       │                  ├─ tui.RunConfirm("¿Configurar modelos?")
       │                  ├─ [if ok] tui.RunModelsFlow()
       │                  └─ tui.GPUSummary()
       │
       ├─ "setup"    → runSetupFlow()
       │                  ├─ fmt.Print("\033[2J\033[H")   ← clear (REQ-2 + REQ-3)
       │                  ├─ fmt.Println(tui.RenderBrand())
       │                  ├─ checks (HelixDB, Ollama, GPU)
       │                  └─ tui.RunConfirm("¿Iniciar servicios?")
       │
       ├─ "models"   → runModelsFlow()
       │                  ├─ fmt.Print("\033[2J\033[H")   ← clear (REQ-3)
       │                  ├─ check Ollama
       │                  ├─ tui.RunModelsFlow()
       │                  ├─ fmt.Print("\033[2J\033[H")   ← clear antes de confirm test (REQ-3)
       │                  ├─ tui.RunConfirm("¿Probar modelos?")
       │                  ├─ fmt.Print("\033[2J\033[H")   ← clear antes de tests (REQ-3)
       │                  ├─ tui.TestEmbedding(...)
       │                  └─ tui.TestChat(...)
       │
       ├─ "autostart" → runAutostartFlow()
       │                  ├─ fmt.Print("\033[2J\033[H")   ← clear (REQ-2 + REQ-3)
       │                  ├─ fmt.Println(tui.RenderBrand())
       │                  ├─ tui.SetupHelixAutostart()
       │                  └─ tui.SetupOllamaAutostart()
       │
       ├─ "about"    → runAboutFlow()     ← NUEVO (REQ-4)
       │                  ├─ fmt.Println(tui.RenderBrand())
       │                  ├─ texto descriptivo formateado con lipgloss
       │                  └─ pausa/espera antes de volver al menú
       │
       └─ "exit"     → fmt.Println("👋 Hasta luego!")
                        return

RenderNewLogo() (REQ-1):
  ascii-art-logo.txt (raíz)
       │
       ├─ copiado a → internal/tui/assets/logo-new.txt
       │
       ├─ //go:embed en brand.go → logoNewArtRaw string
       │
       ├─ init(): sanitizeArt(logoNewArtRaw) → logoNewArt
       │
       └─ RenderNewLogo(): centeredBlock(logoNewArt, logoNewStyle) → string
                            (logoNewStyle: colorNaranja + Bold)

zorro-logo.tsx (REQ-1):
  const zorroArt = [...]  ← reemplazar con 38 líneas del nuevo arte
  const compactArt = "  🦊 ZyroCLI — Zorro Naranja  "
  Render: <text fg={theme.accent}>  ← ya usa accent, el color se ve naranja
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/assets/logo-new.txt` | **Create** | Copia de `ascii-art-logo.txt` (raíz del proyecto) — 38 líneas de ASCII art del zorro naranja |
| `internal/tui/brand.go` | Modify | +`//go:embed assets/logo-new.txt`, `logoNewArtRaw`, `init()` sanitization, `logoNewStyle` (naranja), `RenderNewLogo()` |
| `internal/tui/brand_test.go` | Modify | +`TestRenderNewLogo()` — verifica non-empty y contenido ASCII |
| `internal/opencode/tui-plugins/zorro-logo.tsx` | Modify | Reemplazar array `zorroArt` (31→38 líneas) y `compactArt` text |
| `cmd/zyrocli/main.go` | Modify | +8 `fmt.Print("\033[2J\033[H")` en puntos de clear screen, +`case "about"`, +`runAboutFlow()` |
| `internal/tui/menu.go` | Modify | +`MenuItem{Key:"about", Label:"Acerca de ZyroCLI", Description:"..."}` |

## Interfaces / Contracts

No se modifican interfaces públicas. Solo se agregan:

```go
// brand.go — nuevas exportaciones
func RenderNewLogo() string              // renderiza el nuevo logo naranja centrado

// main.go — nueva función (package-private)
func runAboutFlow()                       // muestra "Acerca de ZyroCLI" con branding + texto descriptivo
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `RenderNewLogo()` non-empty, contiene caracteres ASCII | `TestRenderNewLogo()` en `brand_test.go` |
| Build | Todo el proyecto compila sin errores | `go build ./...` |
| Manual | Menú muestra "Acerca de ZyroCLI", navegación fluida, clear screen funciona | Ejecutar `zyrocli` y probar cada flujo |

## Migration / Rollout

No migration required. Todos los cambios son aditivos:
- `RenderNewLogo()` es función nueva, nada existente la invoca por defecto
- `MenuItem` "Acerca de ZyroCLI" se agrega a la lista sin modificar los items existentes
- Los clear screens solo afectan la presentación, no la lógica de negocio
- El asset `logo-new.txt` es nuevo, no reemplaza assets existentes

## Implementation Order

1. **REQ-1**: `ascii-art-logo.txt` → `internal/tui/assets/logo-new.txt`, brand.go (embed + style + RenderNewLogo), brand_test.go, zorro-logo.tsx (reemplazar arte)
2. **REQ-4**: menu.go (MenuItem "about"), main.go (case "about" + runAboutFlow())
3. **REQ-2 + REQ-3**: main.go (8 clear screen inserts)

## Open Questions

- [ ] Verificar que el logo de 38 líneas se ve bien en OpenCode (el TSX condiciona por `term.height >= zorroArt.length + 6` — antes con 31 líneas, ahora con 38 líneas el threshold aumenta)
- [ ] Confirmar el texto descriptivo exacto para el item "Acerca de ZyroCLI"
