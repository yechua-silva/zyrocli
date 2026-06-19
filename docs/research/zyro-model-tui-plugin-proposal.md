# Propuesta: TUI Plugin /zyro-model para OpenCode

## Fecha
2026-06-19

## Objetivo
Que el usuario escriba `/zyro-model` en OpenCode y se abra un modal similar a `/connect`
donde pueda seleccionar proveedor → modelo para cada agente SDD de ZyroAgentCLI.

## Agentes a configurar
- Orchestrator (zyro-orchestrator)
- PRE-F0 (zyro-pre-f0)
- F0 (zyro-phase-0-libraries, zyro-phase-0-patterns)
- F1 (zyro-sdd-spec)
- F2 (zyro-sdd-design, zyro-sdd-tasks)
- F3 (zyro-sdd-apply, zyro-sdd-verify)
- F4 (zyro-sdd-archive)

## UX Flow
```
Usuario escribe: /zyro-model
  ↓
Modal 1: "Seleccioná un agente para configurar"
  [Orchestrator]    ← modelo actual: opencode/deepseek-v4-flash
  [PRE-F0]          ← modelo actual: default
  [F0]              ← modelo actual: default
  [F1]
  [F2]
  [F3]
  [F4]
  [Set all phases]  ← opción para asignar el mismo modelo a todo
  ↓
Modal 2: "Seleccioná un provider para {agente}"
  [opencode-go]     ← 8 modelos
  [anthropic]       ← 3 modelos
  [google]          ← 2 modelos
  [ollama]          ← 4 modelos (phi4-mini, nomic-embed-text, qwen3.5...)
  ↓
Modal 3: "Seleccioná un modelo de {provider} para {agente}"
  [claude-sonnet-4-20250514]  ← 200K context
  [claude-haiku-4-20250514]   ← 200K context
  [claude-opus-4-20250514]    ← 200K context
  ↓
Toast: "Modelo asignado: {provider}/{model} para {agente} ✅"
Se cierra el modal. Vuelve al Modal 1 (siguiente agente).
```

## Implementación

### Fase 1: Scaffold
Crear `scripts/npm/zyro-model-plugin/` con:
- `package.json` (peerDeps: @opencode-ai/plugin, @opentui/core, @opentui/solid, solid-js)
- `tsup.config.ts` (build con esbuild-plugin-solid)
- `tsconfig.json`
- `src/index.tsx` (punto de entrada con registro de comando)

### Fase 2: MVP básico
- Registrar `/zyro-model` vía `api.command.register()`
- Mostrar DialogSelect con agentes
- Al seleccionar agente: DialogSelect con providers de `api.state.provider`
- Al seleccionar provider: DialogSelect con modelos
- Al seleccionar modelo: escribir via `api.client.global.config.update()`

### Fase 3: Integración con instalador
- `zyrocli install` copia el plugin compilado a `~/.config/opencode/plugins/zyro-model.js`
- O registra como npm package en `tui.json`

## APIs clave a utilizar
| Propósito | API |
|---|---|
| Registrar comando | `api.command.register()` con `slash: { name: "zyro-model" }` |
| Mostrar selector de opciones | `api.ui.DialogSelect` |
| Leer providers | `api.state.provider` |
| Leer config actual | `api.state.config.agent` |
| Escribir config en runtime | `api.client.global.config.update()` |
| Notificación | `api.ui.toast()` |
| Registrar atajo teclado | `api.keymap.registerLayer({ bindings: [{ key: "alt+m", cmd: ":zyro-model" }] })` |

## Referencias
- Plugin de referencia: `opencode-sdd-engram-manage` (npm) / `j0k3r-dev-rgl/sdd-engram-plugin` (GitHub)
- TUI Plugin API: `@opencode-ai/plugin/tui`
- build tooling: tsup + esbuild-plugin-solid
