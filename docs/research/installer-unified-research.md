# Investigación: Instalador Unificado ZyroCLI

> Fecha: 17 Junio 2026
> Contexto: Unificar zyrocli install, zyrocli setup, install_tui.py y branding en una sola experiencia

## Estado Actual

### Componentes existentes

| Componente | Ubicación | Tecnología | LOC | Estado |
|------------|-----------|------------|-----|--------|
| `zyrocli install` | `cmd/zyrocli/install.go` | Go + Cobra | 291 | ✅ Configura OpenCode |
| `zyrocli setup` | `cmd/zyrocli/setup.go` | Go + Cobra | 88 | ✅ Checkea dependencias |
| `install_tui.py` | `scripts/install_tui.py` | Python + Rich | 1511 | ✅ TUI Ollama/Modelos/GPU |
| Brand ASCII | `zyro-brand-ascii-art.md` | Markdown | 9 | ⬜ Sin usar |
| Logo ASCII | `zyro-logo-ascii-art.md` | Markdown | 34 | ⬜ Sin usar |
| OpenCode theme | `~/.config/opencode/tui.json` | JSON | 7 | ⬜ Casi vacío |
| OpenCode logo plugin | `~/.config/opencode/tui-plugins/gentle-logo.tsx` | TSX + SolidJS | ~60 | ⬜ Existe como ejemplo |

### Análisis de Brechas

1. **Brand no se muestra**: El ZYRO 3D y el Zorro Hacker están en .md pero no se renderizan en ningún lado
2. **TUI separada del install**: El install_tui.py es un script Python que hay que ejecutar aparte
3. **OpenCode sin logo**: tui.json está casi vacío, no tiene el Zorro Hacker configurado
4. **Next steps incompletos**: No mencionan `zyro onboard` para proyectos existentes

## Estado de HelixDB

### Diagnóstico (17 Junio 2026)

| Componente | Estado | Detalle |
|------------|--------|---------|
| CLI binario | ✅ Instalado | `/home/secko/.local/bin/helix` — Helix CLI v3.0.5 |
| Servicio | ✅ Corriendo | Contenedor Docker `helix-zyrocli-global-dev` |
| API | ✅ Respondiendo | `localhost:6969/health` → `{"healthy":true,"service":"gateway"}` |
| zyrocli db status | ✅ Conectado | |

### Cómo se manejará en el instalador

El instalador debe:
1. Verificar si HelixDB responde (`curl localhost:6969/health`)
2. Si no responde, intentar iniciar el contenedor Docker: `docker start helix-zyrocli-global-dev`
3. Si el contenedor no existe, mostrar instrucciones para instalarlo
4. No bloquear si HelixDB no está disponible (graceful degradation)

## Arquitectura Propuesta

### Flujo Unificado

```
zyrocli install (o setup)
  ├── 1. Brand: ZYRO 3D + Zorro Hacker → banner terminal (Go + lipgloss)
  ├── 2. Setup: check dependencias básicas
  ├── 3. TUI Python: lanza install_tui.py para Ollama/modelos/GPU
  ├── 4. Config: skills + agents + MCP en OpenCode (ya existe)
  ├── 5. Theme: Zorro Hacker → OpenCode tui.json + plugin .tsx
  └── 6. Summary: next steps con onboard incluido
```

### Dependencias Tecnológicas

| Dependencia | Uso | Ya en go.mod? |
|-------------|-----|---------------|
| `github.com/charmbracelet/bubbletea` | TUI en Go | ✅ v1.3.10 |
| `github.com/charmbracelet/lipgloss` | Estilos TUI | ✅ v1.1.0 |
| Python + Rich | TUI de Ollama (install_tui.py) | ❌ (externo) |
| OpenCode TUI plugins | Logo Zorro en dashboard | ❌ (externo, SolidJS) |

## Investigación OpenCode Theme

### Cómo se configura el logo en OpenCode

OpenCode NO tiene un campo directo `"logo"` en la configuración. El logo se implementa mediante **plugins TUI** que se registran en slots específicos.

#### Slots disponibles en OpenCode TUI:
- `home_logo` → Logo en la pantalla de inicio ← **Este es el que necesitamos**
- `home_prompt` → Input de prompt en home
- `session_prompt` → Input en sesión activa
- `home_footer` → Footer del home
- `sidebar_title` → Título del sidebar
- `sidebar_content` → Contenido del sidebar

#### Estructura de un plugin TUI:
```tsx
// Basado en gentle-logo.tsx (ejemplo existente funcional)
const tui: TuiPlugin = async (api) => {
  api.slots.register({
    id: "zorro-logo",
    order: 100,
    slots: {
      home_logo(ctx) {
        return <Logo theme={ctx.theme.current} />
      },
    },
  })
}
```

#### Archivos de configuración afectados:
- `~/.config/opencode/tui.json` → Lista de plugins activos
- `~/.config/opencode/tui-plugins/zorro-logo.tsx` → Plugin del Zorro Hacker (a crear)

### Assets de Branding

#### Brand (ZYRO 3D) — `zyro-brand-ascii-art.md`:
```
             █████    █   █    ████      ███                    ███     █        ███   
              ░░█░░    █ █ ░   █░░░█    █ ░░█                  █ ░░░    █░        █░░  
               █░░░░    █ ░ ░  ████░░   █░ ░█░      ████       █░ ░░░   █░░       █░░░ 
              █ ░ ░     █░ ░   █░░█░ ░  █░░ █░░      ░░░░      █░░      █░░       █░░  
             █████      █░░    █░░░█░    ███ ░░       ░░░░      ███     █████    ███░  
              ░░░░░      ░░     ░░  ░     ░░░ ░                  ░░░     ░░░░░    ░░░  
               ░░░░░      ░      ░   ░     ░░░                    ░░░     ░░░░░    ░░░ 
```
Letras "ZYRO" en 3D con caracteres █ y ░

#### Logo (Zorro Hacker) — `zyro-logo-ascii-art.md`:
Diseño floral/orgánico de ~34 líneas. Símbolos: # * + - : .
Representa la parte "Gentle" de la marca (contraste con el ZYRO tech).

### Estrategia de Implementación

1. **Go embed**: Embebe los .md de brand y logo en el binario
2. **internal/tui/**: Package Go que renderiza con lipgloss
3. **install.go**: Refactor para mostrar brand + lanzar TUI Python + configurar tema
4. **setup.go**: Refactor para mostrar brand + lanzar TUI Python
5. **theme writer**: Escribir plugin .tsx y tui.json desde Go

## Feedback de Implementación

### 1. Puente Go → Python (install_tui.py)

Al lanzar el script Python desde Go, la TUI debe heredar correctamente la terminal:

```go
cmd := exec.Command("python3", scriptPath)
cmd.Stdin = os.Stdin   // ← necesario para input del teclado
cmd.Stdout = os.Stdout // ← necesario para colores Rich
cmd.Stderr = os.Stderr // ← necesario para errores
cmd.Run()              // bloquea hasta que el TUI termina
```

Sin `cmd.Stdin = os.Stdin`, las flechas del teclado y la navegación Rich no funcionan.
Sin `cmd.Stdout = os.Stdout`, los colores y el renderizado Rich se pierden.

### 2. Plugin TSX de OpenCode (zorro-logo.tsx)

El plugin del Zorro Hacker necesita dos registros:

1. **Archivo físico**: `~/.config/opencode/tui-plugins/zorro-logo.tsx`
2. **Registro en config**: Referencia en `~/.config/opencode/tui.json` bajo `"plugin"`:
   ```json
   {
     "$schema": "https://opencode.ai/tui.json",
     "plugin": [
       "opencode-subagent-statusline",
       "/home/usuario/.config/opencode/tui-plugins/zorro-logo.tsx"
     ]
   }
   ```

Dependiendo de la versión de OpenCode, también puede requerir inyectar la referencia en `opencode.jsonc`.

### 3. Idempotencia

Todo debe funcionar si se ejecuta múltiples veces:

1. **Brand/Logo**: Solo se renderiza, no escribe nada → naturalmente idempotente
2. **TUI Python**: El script install_tui.py ya es idempotente (checkea si ya hay modelos instalados)
3. **OpenCode config** (`opencode.jsonc`): Se sobreescribe completo cada vez → idempotente
4. **TUI theme** (`tui.json`): Reescribir completo cada vez → idempotente
5. **Plugin .tsx**: Reescribir el archivo cada vez → idempotente
6. **Skills**: `WriteAllSkills()` sobreescribe → idempotente
7. **MCP tools**: `WriteMCPTools()` sobreescribe → idempotente

No debe haber assembly de configuraciones (append a arrays) — siempre sobreescribir completa.

## Próximos Pasos

1. P1: Crear `internal/tui/` con renderer de brand + logo (lipgloss)
2. P2: Embed del plugin TSX + theme writer
3. P3: Refactor install.go (brand + TUI + theme + next steps)
4. P4: Refactor setup.go (brand + TUI bridge)
5. P5: Bridge Go → Python TUI
6. P6: Tests
7. P7: Build + verify

### Enlaces de Referencia
- OpenCode TUI plugins: https://opencode.ai/docs/tui-plugins
- Bubbletea: https://github.com/charmbracelet/bubbletea
- Lipgloss: https://github.com/charmbracelet/lipgloss
- Rich (Python TUI): https://rich.readthedocs.io/
