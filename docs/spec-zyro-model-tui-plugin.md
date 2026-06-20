# Spec F1 — TUI Plugin `/zyro-model`

> **Fecha:** 2026-06-20  
> **Autor:** ZyroCLI Orchestrator  
> **Estado:** SPEC — lista para F2 (Design + Tasks)  
> **Basado en:** `docs/spec-zyro-model-routing.md`

---

## 1. Resumen

Sistema de asignación interactiva de modelos LLM por agente SDD, implementado como **TUI Plugin** para OpenCode (SolidJS/TSX). El plugin se invoca con el comando `/zyro-model` (o `Alt+K`) y presenta una secuencia de tres `DialogSelect` encadenados:

1. **Selector de agente** — los 16 agentes SDD agrupados por fase, más "★ Set All" y "✓ Done"
2. **Selector de proveedor** — proveedores disponibles en OpenCode (con modelos)
3. **Selector de modelo** — modelos del proveedor elegido

Al confirmar un modelo, se escribe en caliente via `api.client.global.config.update()` y se persiste en disco via `zyrocli profile set`. El plugin se integra con el instalador `zyrocli install` mediante embed en Go.

---

## 2. Arquitectura

### 2.1 Diagrama de capas

```
┌────────────────────────────────────────────────────────────────────┐
│                        OpenCode TUI Sandbox                         │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │              TUI Plugin: zyro-model.tsx                       │  │
│  │  ┌──────────┐   ┌──────────┐   ┌──────────┐                │  │
│  │  │DialogSel.#1│→│DialogSel.#2│→│DialogSel.#3│               │  │
│  │  │ Agentes   │   │ Proveed. │   │ Modelos  │               │  │
│  │  └─────┬─────┘   └────┬─────┘   └────┬─────┘               │  │
│  │        │              │              │                       │  │
│  │        ▼              ▼              ▼                       │  │
│  │  ┌──────────────────────────────────────────────────────┐   │  │
│  │  │          Lógica de asignación (assignModel)          │   │  │
│  │  │  api.client.global.config.update() → en caliente      │   │  │
│  │  │  Bun.$ zyrocli profile set ...     → en disco        │   │  │
│  │  │  api.ui.toast()                    → confirmación    │   │  │
│  │  └──────────────────────────────────────────────────────┘   │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │              TUI Plugin: zorro-logo.tsx                       │  │
│  │  (slots.register home_logo — no interfiere)                  │  │
│  └──────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────────┐
│                      opencode.json / opencode.jsonc                 │
│  agent: {                                                          │
│    "zyro-orchestrator": { "model": "opencode-go/deepseek-v4-flash"},│
│    "zyro-sdd-apply":     { "model": "anthropic/claude-sonnet-4"},  │
│    ...                                                              │
│  }                                                                  │
└────────────────────────────────────────────────────────────────────┘
```

### 2.2 Ubicación del plugin

| Elemento | Ruta |
|----------|------|
| Archivo fuente del plugin | `~/.config/opencode/tui-plugins/zyro-model.tsx` |
| Registro en TUI | `~/.config/opencode/tui.json` → `"plugin": ["...zyro-model.tsx"]` |
| Server plugin legacy | `~/.config/opencode/plugins/zyro-model.ts` (a eliminar) |
| Embed en Go (instalador) | `internal/opencode/tui-plugins/zyro-model.tsx` |
| Go embed directive | `//go:embed tui-plugins/zyro-model.tsx` en `tui_plugins.go` |

### 2.3 Registro en tui.json

El plugin se registra en `~/.config/opencode/tui.json` como una entrada en el array `plugin`:

```json
{
  "$schema": "https://opencode.ai/tui.json",
  "plugin": [
    "/home/secko/.config/opencode/tui-plugins/zorro-logo.tsx",
    "/home/secko/.config/opencode/tui-plugins/zyro-model.tsx"
  ]
}
```

---

## 3. Flujo de usuario

### 3.1 Diagrama de flujo completo

```
Usuario escribe /zyro-model o presiona Alt+K
│
├── ¿OpenCode cargado? ──No──→ Toast: "OpenCode no está disponible"
│   │
│   Sí
│   ▼
├── [DialogSelect #1 — Agentes]
│   ┌────────────────────────────────────────────────────────┐
│   │  ★ Set All (asignar mismo modelo a todos los agentes) │
│   │                                                        │
│   │  — PRE-F0 —                                            │
│   │  ○ zyro-pre-f0           Alineación de dominio         │
│   │                                                        │
│   │  — F0 —                                               │
│   │  ○ zyro-phase-0-patterns Búsqueda de patrones         │
│   │  ○ zyro-phase-0-libraries Investigación de librerías  │
│   │  ○ zyro-skills-find       Descubrimiento de skills    │
│   │  ○ zyro-skills-audit      Validación de skills        │
│   │  ○ zyro-skills-apply      Instalación de skills       │
│   │  ○ zyro-sdd-explore       Exploración de codebase     │
│   │                                                        │
│   │  — F1 —                                               │
│   │  ○ zyro-sdd-spec          Especificación técnica      │
│   │                                                        │
│   │  — F2 —                                               │
│   │  ○ zyro-sdd-propose       Propuestas de cambio        │
│   │  ○ zyro-sdd-design        Diseño técnico              │
│   │  ○ zyro-sdd-tasks         Tareas atómicas             │
│   │                                                        │
│   │  — F3 —                                               │
│   │  ○ zyro-sdd-apply         Implementación              │
│   │  ○ zyro-sdd-verify        Verificación               │
│   │                                                        │
│   │  — F4 —                                               │
│   │  ○ zyro-sdd-archive       Archivo y cierre            │
│   │                                                        │
│   │  — —                                                  │
│   │  ○ zyro-orchestrator      Coordinador principal       │
│   │  ○ to-issues              GitHub Issues desde PRDs    │
│   │                                                        │
│   │  ✓ Done                                               │
│   └───────────────────────────┬────────────────────────────┘
│                               │
│               ┌───────────────┼───────────────┐
│               ▼               ▼               ▼
│         "★ Set All"     agente ind'    "✓ Done"
│               │               │               │
│               ▼               ▼               └──→ cierra diálogos
│         [DialogSelect #2]  [DialogSelect #2]       return
│               │               │
│               ▼               ▼
│         [DialogSelect #3]  [DialogSelect #3]
│               │               │
│               ▼               ▼
│         asigna a todos   asigna a 1 agente
│         (loop 16 veces)
│               │               │
│               ▼               ▼
│         [DialogSelect #1]  [DialogSelect #1]
│         (vuelve a agentes) (vuelve a agentes)
│
├── Toast: "✓ zyro-sdd-apply → anthropic/claude-sonnet-4"
│
└── Usuario elige "✓ Done" → fin
```

### 3.2 Paso a paso detallado

| Paso | Acción | API | Salida |
|------|--------|-----|--------|
| 1 | Usuario invoca `/zyro-model` o `Alt+K` | `api.command.register()`, `api.keymap.registerLayer()` | Se abre DialogSelect #1 |
| 2 | Usuario navega agentes con ↑↓ | — | Cursor se mueve entre opciones |
| 3a | Usuario selecciona agente → Enter | — | Se abre DialogSelect #2 con proveedores |
| 3b | Usuario selecciona "★ Set All" → Enter | — | Se abre DialogSelect #2 (Set All mode) |
| 3c | Usuario selecciona "✓ Done" → Enter | — | Se cierran diálogos, fin |
| 4 | Usuario selecciona proveedor → Enter | `api.state.provider` | Se abre DialogSelect #3 con modelos |
| 5 | Usuario selecciona modelo → Enter | — | Se llama a `assignModel(agent, provider/model)` |
| 6 | Asignación en caliente | `api.client.global.config.update()` | Config se actualiza en memoria |
| 7 | Persistencia en disco | `Bun.$ zyrocli profile set <agent> <model>` | Config se escribe en opencode.json(c) |
| 8 | Toast de confirmación | `api.ui.toast()` | "✓ zyro-sdd-apply → anthropic/claude-sonnet-4" |
| 9 | Loop: vuelve a DialogSelect #1 | — | Usuario puede seguir asignando |

### 3.3 Caso especial: Set All

```
"★ Set All" → DialogSelect #2 (proveedores) → DialogSelect #3 (modelo)
→ for each agent in ZYRO_AGENTS:
    api.client.global.config.update({ agent: { [agent.name]: { model: selectedModel }}})
    Bun.$ zyrocli profile set <agent.name> <selectedModel>
→ Toast: "✓ Set All: Todos los agentes → anthropic/claude-sonnet-4"
→ Loop: vuelve a DialogSelect #1
```

---

## 4. Componentes

### 4.1 Archivo del plugin

**Archivo:** `~/.config/opencode/tui-plugins/zyro-model.tsx`

**Estructura del código:**

```typescript
// @ts-nocheck
/** @jsxImportSource @opentui/solid */
import type { TuiPlugin } from "@opencode-ai/plugin/tui"
import { DialogSelect } from "@opencode-ai/plugin/tui"
import { createSignal } from "solid-js"

// ─── Constantes ───────────────────────────────────────────────────
const PLUGIN_ID = "zyro-model"

interface AgentInfo {
  name: string
  description: string
  phase: string
  currentModel: string
}

const ZYRO_AGENTS: AgentInfo[] = [
  // 16 agentes ordenados por fase
  // Ver sección 7 para la lista completa
]

// ─── Plugin ────────────────────────────────────────────────────────
const tui: TuiPlugin = async (api) => {
  // Registrar comando /zyro-model
  api.command.register({
    id: PLUGIN_ID,
    name: "zyro-model",
    description: "Asignar modelos LLM por agente SDD",
    handler: () => start(api),
  })

  // Registrar keymap Alt+K (solo cuando no hay input activo)
  api.keymap.registerLayer({
    id: `${PLUGIN_ID}-keymap`,
    keymap: {
      "Alt+K": {
        name: "Zyro: Asignar modelo a agente",
        command: PLUGIN_ID,
      },
    },
  })
}

// ─── Flujo principal ──────────────────────────────────────────────
async function start(api: any) {
  // Leer proveedores disponibles
  const providers = await getAvailableProviders(api)

  // Leer configuración actual de agentes
  const config = await api.client.global.config.get()
  const agents = buildAgentList(config.agent || {})

  // DialogSelect #1 — Lista de agentes
  const agentChoice = await showAgentSelector(api, agents)
  if (!agentChoice || agentChoice === "DONE") return
  if (agentChoice === "SET_ALL") {
    await handleSetAll(api, providers, agents)
    return
  }

  // DialogSelect #2 — Lista de proveedores
  const providerChoice = await showProviderSelector(api, providers)
  if (!providerChoice || providerChoice === "BACK") {
    return start(api) // reinicia
  }

  // DialogSelect #3 — Lista de modelos
  const modelChoice = await showModelSelector(api, providerChoice)
  if (!modelChoice || modelChoice === "BACK") {
    return start(api) // reinicia
  }

  // Asignar modelo
  await assignModel(api, agentChoice, `${providerChoice.id}/${modelChoice.id}`)

  // Loop: volver a agentes
  return start(api)
}

// ─── Selectores ────────────────────────────────────────────────────
async function showAgentSelector(api: any, agents: AgentInfo[]) {
  // Construye opciones del DialogSelect con:
  //   - "★ Set All" al inicio
  //   - Grupos por fase (PRE-F0, F0, F1, F2, F3, F4, —)
  //   - Cada agente: "nombre — descripción (modelo actual)"
  //   - "✓ Done" al final
  return DialogSelect(api, { ... })
}

async function showProviderSelector(api: any, providers: Provider[]) {
  // Filtra proveedores con modelos disponibles
  // Muestra: "proveedor (N modelos)"
  // Opción "← Volver"
  return DialogSelect(api, { ... })
}

async function showModelSelector(api: any, provider: Provider) {
  // Muestra modelos del proveedor
  // Opción "← Volver"
  return DialogSelect(api, { ... })
}

// ─── Asignación ────────────────────────────────────────────────────
async function assignModel(api: any, agentName: string, modelStr: string) {
  // 1. En caliente: api.client.global.config.update()
  await api.client.global.config.update({
    agent: { [agentName]: { model: modelStr } },
  })

  // 2. En disco: Bun.$ zyrocli profile set ...
  try {
    const result = await Bun.$`zyrocli profile set ${agentName} ${modelStr}`
    if (result.exitCode !== 0) {
      throw new Error(result.stderr.toString())
    }
  } catch (e) {
    // Toast error si falla CLI
    await api.ui.toast({
      message: `⚠ Error al persistir: ${e.message}`,
      variant: "error",
    })
    return
  }

  // 3. Toast confirmación
  await api.ui.toast({
    message: `✓ ${agentName} → ${modelStr}`,
    variant: "success",
  })
}

async function handleSetAll(api: any, providers: Provider[], agents: AgentInfo[]) {
  const provider = await showProviderSelector(api, providers)
  if (!provider || provider === "BACK") return

  const model = await showModelSelector(api, provider)
  if (!model || model === "BACK") return

  const modelStr = `${provider.id}/${model.id}`
  for (const agent of agents) {
    await api.client.global.config.update({
      agent: { [agent.name]: { model: modelStr } },
    })
    try {
      await Bun.$`zyrocli profile set ${agent.name} ${modelStr}`
    } catch (e) {
      // continuar con el siguiente aunque uno falle
    }
  }

  await api.ui.toast({
    message: `✓ Set All: Todos los agentes → ${modelStr}`,
    variant: "success",
  })
}

// ─── Helpers ───────────────────────────────────────────────────────
async function getAvailableProviders(api: any) {
  const state = await api.state.provider()
  // Filtra solo proveedores con modelos disponibles
  return state.providers.filter((p: any) => p.models?.length > 0)
}

function buildAgentList(agentConfig: Record<string, any>): AgentInfo[] {
  return ZYRO_AGENTS.map(a => ({
    ...a,
    currentModel: agentConfig[a.name]?.model || "(hereda del orchestrator)",
  }))
}

export default { id: PLUGIN_ID, tui }
```

### 4.2 Dependencias

| Dependencia | Propósito | Origen |
|-------------|-----------|--------|
| `@opencode-ai/plugin/tui` | Tipos `TuiPlugin`, componente `DialogSelect` | OpenCode SDK |
| `@opentui/solid` | JSX runtime para TUI (`/** @jsxImportSource @opentui/solid */`) | OpenCode SDK |
| `solid-js` | `createSignal`, reactividad | OpenCode SDK Bundled |
| `zyrocli` | Comando `zyrocli profile set` para persistencia | Go binary en PATH |

### 4.3 Build

No requiere build externo. OpenCode carga el `.tsx` directamente con su runtime interno de SolidJS + TypeScript. El `// @ts-nocheck` evita errores de tipo si el usuario no tiene los tipos instalados localmente.

### 4.4 Registro

El plugin se registra automáticamente en `~/.config/opencode/tui.json` durante `zyrocli install` (ver sección 8).

---

## 5. API de OpenCode utilizada

### 5.1 `api.command.register()`

Registra el comando `/zyro-model` en OpenCode. El comando aparece en autocompletado al escribir `/`.

```typescript
api.command.register({
  id: "zyro-model",
  name: "zyro-model",
  description: "Asignar modelos LLM por agente SDD",
  handler: () => start(api),
})
```

### 5.2 `api.keymap.registerLayer()`

Registra un keymap global `Alt+K` que ejecuta el mismo comando. Solo se activa cuando no hay un input activo (para no interferir con edición).

```typescript
api.keymap.registerLayer({
  id: "zyro-model-keymap",
  keymap: {
    "Alt+K": {
      name: "Zyro: Asignar modelo a agente",
      command: "zyro-model",
    },
  },
})
```

### 5.3 `api.state.provider`

Obtiene el estado actual de proveedores y modelos desde el SDK de OpenCode. Retorna los proveedores que el usuario ha configurado (con API keys).

```typescript
const state = await api.state.provider()
// state.providers: [{ id: "anthropic", name: "Anthropic", models: [...] }]
```

### 5.4 `DialogSelect` (desde `@opencode-ai/plugin/tui`)

Componente de diálogo de selección nativo de OpenCode TUI. Muestra una lista con opciones navegables con ↑↓ y Enter.

```typescript
import { DialogSelect } from "@opencode-ai/plugin/tui"

const choice = await DialogSelect(api, {
  title: "Seleccionar agente",
  options: [
    { id: "SET_ALL", label: "★ Set All" },
    { id: "zyro-sdd-apply", label: "zyro-sdd-apply — Implementación" },
    { id: "DONE", label: "✓ Done" },
  ],
})
```

### 5.5 `api.client.global.config.update()`

Actualiza la configuración de OpenCode en caliente (en memoria). El cambio es inmediato para el runtime actual.

```typescript
await api.client.global.config.update({
  agent: { "zyro-sdd-apply": { model: "anthropic/claude-sonnet-4" } },
})
```

### 5.6 `Bun.$` (shell)

Ejecuta comandos del sistema desde el plugin. Se usa para persistir la configuración en disco a través de `zyrocli profile set`.

```typescript
const result = await Bun.$`zyrocli profile set zyro-sdd-apply anthropic/claude-sonnet-4`
if (result.exitCode === 0) {
  // éxito
}
```

### 5.7 `api.ui.toast()`

Muestra una notificación toast en la interfaz de OpenCode.

```typescript
await api.ui.toast({
  message: "✓ zyro-sdd-apply → anthropic/claude-sonnet-4",
  variant: "success", // success | error | info | warning
})
```

---

## 6. Criterios de aceptación

### 6.1 Funcionales (10)

| ID | Criterio | Verificación |
|----|----------|-------------|
| CA1 | El comando `/zyro-model` aparece en autocompletado de OpenCode | Escribir `/z` → ver sugerencia |
| CA2 | `Alt+K` abre el selector de agentes | Presionar Alt+K → se abre DialogSelect |
| CA3 | El DialogSelect #1 muestra los 16 agentes agrupados por fase con sus descripciones | Verificar visualmente |
| CA4 | El DialogSelect #2 muestra SOLO los proveedores configurados en OpenCode | Comparar con `/connect` |
| CA5 | Las opciones "← Volver" en selectores #2 y #3 regresan al selector anterior | Navegar y verificar |
| CA6 | "★ Set All" asigna el mismo modelo a todos los 16 agentes | Verificar opencode.json después |
| CA7 | "✓ Done" cierra todos los diálogos | Seleccionar → diálogos desaparecen |
| CA8 | Después de asignar un modelo (individual o Set All), el flujo vuelve al selector de agentes (loop) | Verificar que se puede asignar otro |
| CA9 | La asignación se escribe en caliente (memoria) y en disco (archivo) | `api.client.global.config.get()` + leer opencode.json |
| CA10 | Aparece toast de confirmación con el nombre del agente y el modelo asignado | Verificar visualmente |

### 6.2 Regresión (5)

| ID | Criterio | Verificación |
|----|----------|-------------|
| R1 | Si no hay proveedores configurados, el plugin muestra mensaje "No hay proveedores. Configurá uno con /connect" y un botón "← Volver" | Desconectar todos los providers y probar |
| R2 | Si `zyrocli` no está en PATH, el plugin asigna en caliente igual pero muestra toast de advertencia | `mv $(which zyrocli) /tmp/` y probar |
| R3 | El plugin no interfiere con otros TUI plugins registrados (ej: zorro-logo.tsx) | Verificar que zorro-logo sigue funcionando |
| R4 | El plugin funciona tanto en opencode.json como opencode.jsonc | Probar con ambos formatos |
| R5 | Si `zyrocli profile set` falla (permisos, archivo corrupto), se muestra toast de error y la asignación en caliente se pierde | Corromper opencode.json y probar |

### 6.3 Consideraciones UX

- El loop post-asignación debe ser natural: asignar un modelo y volver a la lista de agentes permite asignar otro sin reinvocar `/zyro-model`
- El orden de los agentes en DialogSelect #1 debe coincidir con el orden SDD (PRE-F0 → F4), no alfabético
- "★ Set All" va PRIMERO, "✓ Done" va ÚLTIMO
- Cada agente muestra el modelo actual entre paréntesis: `zyro-sdd-apply — Implementación (opencode-go/deepseek-v4-flash)`
- Los proveedores sin modelos (lista vacía) se excluyen del DialogSelect #2
- El keymap Alt+K solo debe activarse cuando OpenCode no esté en modo edición de texto

---

## 7. Definición de los 16 agentes

### 7.1 Tabla completa

| # | Agente | Fase | Descripción | Orden en selector |
|---|--------|------|-------------|-------------------|
| 1 | `zyro-pre-f0` | PRE-F0 | Alineación de dominio — grill-me, domain-model, triage, improve-arch | 1 |
| 2 | `zyro-phase-0-patterns` | F0 | Búsqueda de patrones similares en internet | 2 |
| 3 | `zyro-phase-0-libraries` | F0 | Investigación de librerías con Context + GitMCP | 3 |
| 4 | `zyro-skills-find` | F0 | Descubrimiento de skills en skills.sh | 4 |
| 5 | `zyro-skills-audit` | F0 | Validación de skills descubiertas | 5 |
| 6 | `zyro-skills-apply` | F0 | Instalación de skills aprobadas | 6 |
| 7 | `zyro-sdd-explore` | F0 | Exploración de codebase y requerimientos | 7 |
| 8 | `zyro-sdd-spec` | F1 | Especificación técnica basada en hallazgos F0 | 8 |
| 9 | `zyro-sdd-propose` | F2 | Propuestas de cambio con intento, alcance y enfoque | 9 |
| 10 | `zyro-sdd-design` | F2 | Diseño técnico basado en Spec | 10 |
| 11 | `zyro-sdd-tasks` | F2 | División del diseño en tareas atómicas | 11 |
| 12 | `zyro-sdd-apply` | F3 | Implementación siguiendo specs, design y tasks | 12 |
| 13 | `zyro-sdd-verify` | F3 | Verificación contra specs, design y tasks | 13 |
| 14 | `zyro-sdd-archive` | F4 | Archivo de cambios completados y cierre de ciclo | 14 |
| 15 | `zyro-orchestrator` | — | Coordinador principal — solo habla y delega, nunca toca código | 15 |
| 16 | `to-issues` | — | Generación de GitHub Issues desde PRDs y tasks | 16 |

### 7.2 Agrupación por fase

```
★ Set All
────────────────────
— PRE-F0 —
zyro-pre-f0
────────────────────
— F0 —
zyro-phase-0-patterns
zyro-phase-0-libraries
zyro-skills-find
zyro-skills-audit
zyro-skills-apply
zyro-sdd-explore
────────────────────
— F1 —
zyro-sdd-spec
────────────────────
— F2 —
zyro-sdd-propose
zyro-sdd-design
zyro-sdd-tasks
────────────────────
— F3 —
zyro-sdd-apply
zyro-sdd-verify
────────────────────
— F4 —
zyro-sdd-archive
────────────────────
— — (sin fase)
zyro-orchestrator
to-issues
────────────────────
✓ Done
```

### 7.3 Orden lógico

El orden sigue el pipeline SDD: primero los agentes de alineación (PRE-F0), luego los de investigación (F0), especificación (F1), diseño y tareas (F2), implementación y verificación (F3), archivo (F4), y por último los agentes transversales (orchestrator, to-issues). Esto reproduce el flujo natural de un ciclo de desarrollo.

---

## 8. Integración con instalador

### 8.1 Embed en Go

El plugin se embeberá en el binario Go de `zyrocli` usando la directiva `//go:embed`:

```go
// internal/opencode/tui_plugins.go

//go:embed tui-plugins/zyro-model.tsx
var zyroModelPlugin string
```

El archivo fuente se coloca en `internal/opencode/tui-plugins/zyro-model.tsx`.

### 8.2 Extensión de `tui_plugins.go`

Se agregan dos funciones:

```go
// ZyroModelPluginPath retorna la ruta de instalación del plugin.
func ZyroModelPluginPath() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".config", "opencode", "tui-plugins", "zyro-model.tsx")
}

// WriteZyroModelPlugin escribe el plugin en el directorio de TUI plugins.
func WriteZyroModelPlugin() (string, error) {
    pluginPath := ZyroModelPluginPath()
    if err := os.MkdirAll(filepath.Dir(pluginPath), 0755); err != nil {
        return "", fmt.Errorf("opencode: create tui-plugins dir: %w", err)
    }
    if err := os.WriteFile(pluginPath, []byte(zyroModelPlugin), 0644); err != nil {
        return "", fmt.Errorf("opencode: write zyro-model plugin: %w", err)
    }
    return pluginPath, nil
}
```

### 8.3 Actualización de `UpdateTuiJSON()`

La función `UpdateTuiJSON()` en `tui_plugins.go` debe:

1. Agregar `ZyroModelPluginPath()` a la lista `wanted` (junto con `ZorroPluginPath()`)
2. Remover `zyro-model.js` de la lista `stale` (si existe)
3. La lista `stale` actualizada incluye:
   - `"opencode-subagent-statusline"`
   - `"zyro-model"` (referencia antigua sin extensión)

### 8.4 Integración en `zyrocli install`

El flujo de instalación (`cmd/zyrocli/install.go`) debe incluir:

```go
steps := []tui.InstallStep{
    {Name: "Instalando plugin Zorro Logo", Action: func() error {
        _, err := opencode.WriteZorroLogo()
        return err
    }},
    {Name: "Instalando plugin Zyro Model", Action: func() error {
        _, err := opencode.WriteZyroModelPlugin()
        return err
    }},
    {Name: "Registrando plugins en tui.json", Action: func() error {
        return opencode.UpdateTuiJSON()
    }},
    // ... resto de pasos
}
```

---

## 9. Estado actual y migración

### 9.1 Estado actual

| Componente | Estado | Problema |
|------------|--------|----------|
| `~/.config/opencode/plugins/zyro-model.ts` | 🟡 Stale | Server plugin (event-based), depende de `client.config.providers()` que no funciona en TUI sandbox. Solo hace toasts, no tiene UI interactiva. |
| `~/.config/opencode/tui-plugins/zorro-logo.tsx` | ✅ Activo | TUI plugin funcional, slot `home_logo`. Sirve como referencia de API. |
| `cmd/zyrocli/profile_tui.go` | 🟡 Stale | Muestra mensaje "Usá /zyro-model en OpenCode". Bubbletea TUI planeado pero no implementado. |
| `cmd/zyrocli/profile.go` | ✅ Activo | Comandos `list` y `set` funcionales. `validateModel()` implementado. |
| `internal/opencode/tui_plugins.go` | ✅ Activo | Solo maneja zorro-logo. Hay que extenderlo. |

### 9.2 Plan de migración

```
Paso 1: Crear tui-plugins/zyro-model.tsx  (NUEVO)
Paso 2: Agregar embed en tui_plugins.go    (MODIFICAR)
Paso 3: Extender UpdateTuiJSON()           (MODIFICAR)
Paso 4: Agregar paso en install.go         (MODIFICAR)
Paso 5: Eliminar plugins/zyro-model.ts     (ELIMINAR)
Paso 6: Limpiar referencias a model-assigner (LIMPIAR)
Paso 7: Actualizar profile_tui.go si es necesario  (EVALUAR)
```

### 9.3 Archivos resultantes

| Archivo | Acción | LOC estimado |
|---------|--------|-------------|
| `~/.config/opencode/tui-plugins/zyro-model.tsx` | NUEVO | ~250 |
| `internal/opencode/tui-plugins/zyro-model.tsx` | NUEVO (embed copy) | ~250 |
| `internal/opencode/tui_plugins.go` | MODIFICAR (+30 líneas) | ~30 |
| `internal/opencode/tui_plugins_test.go` | MODIFICAR (+tests) | ~50 |
| `cmd/zyrocli/install.go` | MODIFICAR (+paso) | ~10 |
| `~/.config/opencode/tui.json` | MODIFICAR (automático) | +1 línea |
| `~/.config/opencode/plugins/zyro-model.ts` | ELIMINAR | -40 |
| **Total neto** | | **~300 nuevas líneas** |

---

## 10. Referencias

- [Spec: Model Routing por agente SDD](docs/spec-zyro-model-routing.md) — especificación previa, define agentes, providers, herencia de modelo
- [TUI Plugin: zorro-logo.tsx](~/.config/opencode/tui-plugins/zorro-logo.tsx) — referencia de API TUI plugin funcional
- [Server Plugin: zyro-model.ts](~/.config/opencode/plugins/zyro-model.ts) — código stale a reemplazar
- [Go Embed: tui_plugins.go](internal/opencode/tui_plugins.go) — patrón de embed + registro en tui.json
- [Spec: zyrocli install v3](docs/spec-zyrocli-install-v3.md) — instalador con pasos bubbletea
- [OpenCode Plugin API](https://opencode.ai/docs/plugins) — documentación de API de plugins
- [OpenCode TUI Plugin API](https://opencode.ai/docs/tui-plugins) — documentación de TUI plugins (DialogSelect, keymap, state)
