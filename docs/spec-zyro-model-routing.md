# Spec: Model Routing por Agente SDD

## Fecha
2026-06-20

## Autores
ZyroCLI Orchestrator

## 1. Resumen ejecutivo

Sistema que permite asignar un modelo LLM DISTINTO a cada uno de los 15 agentes SDD de ZyroCLI,
usando proveedores configurados en OpenCode (Anthropic, Google, OpenCode Go, Ollama, etc.).
Implementado como un plugin `/zyro-model` para OpenCode (JS/TS) + CLI `zyrocli profile` (Go).

## 2. Contexto

- ZyroCLI tiene 15 subagentes que ejecutan fases SDD (PRE-F0 → F0 → F1 → F2 → F3 → F4)
- Todos usan actualmente el mismo modelo: `opencode-go/deepseek-v4-flash`
- Se intentó implementar antes con un plugin TUI `zyro-model.js` que quedó huérfano
- El TUI bubbletea de `zyrocli profile tui` usaba nombres de agente INCORRECTOS
- El agente `model-assigner` nunca existió a pesar de estar documentado

## 3. Principios de diseño

1. **ZyroCLI no gestiona proveedores** — OpenCode ya tiene `/connect` para API keys
2. **Validación estricta** — Solo se permiten modelos que existen en los providers conocidos
3. **Herencia de modelo** — Si un subagente no tiene modelo asignado, usa el del agente que lo invocó (comportamiento nativo de OpenCode)
4. **Dos entry points**, un mismo destino: plugin JS/TS + CLI Go escriben al mismo `opencode.json`
5. **Set All** — Opción para asignar el mismo modelo a todos los agentes de una sola vez

## 4. Arquitectura

```
┌──────────────────────────────────────────────────────────────────┐
│                        opencode.json                              │
│  agent: {                                                         │
│    "zyro-orchestrator": { model: "anthropic/claude-sonnet-4", .. }│
│    "zyro-sdd-apply":     { model: "opencode-go/deepseek-v4-flash"}│
│    "zyro-sdd-verify":    { model: "google/gemini-2.5-pro", ... } │
│    ...                                                             │
│  }                                                                 │
└──────────────────────────────┬───────────────────────────────────┘
                               │ escribe
              ┌────────────────┼────────────────┐
              ▼                ▼                ▼
   ┌──────────────────┐ ┌────────────┐ ┌──────────────┐
   │ /zyro-model       │ │ zyrocli    │ │ zyrocli      │
   │ plugin (JS/TS)    │ │ profile set│ │ profile tui  │
   │ dentro de OpenCode│ │ (script)   │ │ (bubbletea)  │
   │ modal interactivo │ │            │ │ fuera de OC  │
   └──────────────────┘ └────────────┘ └──────────────┘
```

## 5. Componentes

### 5.1 Plugin /zyro-model (JS/TS)

**Ubicación:** `~/.config/opencode/plugins/zyro-model.ts`

**API que usa del SDK de OpenCode:**
- `client.config.providers()` — lista proveedores y modelos disponibles
- `client.config.get()` — lee configuración actual (agentes)
- `client.tui.showToast()` — muestra notificación de confirmación
- `client.tui.appendPrompt()` — útil para debugging

**Flujo:**
1. Usuario escribe `/zyro-model` en OpenCode
2. Plugin lista todos los agentes SDD con sus descripciones
3. Opción "Set All" → seleccionar modelo para todos
4. O seleccionar agente individual
5. Mostrar proveedores disponibles (de `client.config.providers()`)
6. Mostrar modelos del proveedor seleccionado
7. Escribir `agent.{name}.model = "provider/model"` en opencode.json
8. Toast: "✓ zyro-sdd-apply → anthropic/claude-sonnet-4"

### 5.2 CLI zyrocli profile (Go)

**Comandos:**
- `zyrocli profile list` — muestra asignaciones actuales de todos los agentes
- `zyrocli profile set <agent> <provider/model>` — asigna modelo con validación
- `zyrocli profile tui` — TUI interactivo bubbletea (fuera de OpenCode)

**Validación en `profile set`:**
- Formato `provider/model` obligatorio
- Verificar que provider y model existan en KnownProviders + providers cargados
- Error claro si no existe

**Detección de OpenCode en `profile tui`:**
- Checkear `OPENCODE_SESSION_ID` env var
- Si no, intentar conectar a `localhost:4096` (SDK server)
- Si está dentro de OpenCode → mostrar mensaje: "Usá /zyro-model dentro de OpenCode"
- Si no → lanzar bubbletea TUI

### 5.3 Agentes SDD configurables (16 total)

| # | Agente | Fase | Descripción |
|---|--------|------|-------------|
| 1 | zyro-orchestrator | — | Coordinador principal — solo habla y delega |
| 2 | zyro-pre-f0 | PRE-F0 | Alineación de dominio |
| 3 | zyro-phase-0-patterns | F0 | Búsqueda de patrones |
| 4 | zyro-phase-0-libraries | F0 | Investigación de librerías |
| 5 | zyro-skills-find | F0 | Descubrimiento de skills |
| 6 | zyro-skills-audit | F0 | Validación de skills |
| 7 | zyro-skills-apply | F0 | Instalación de skills |
| 8 | zyro-sdd-explore | F0 | Exploración de codebase |
| 9 | zyro-sdd-spec | F1 | Especificación técnica |
| 10 | zyro-sdd-propose | F2 | Propuestas de cambio |
| 11 | zyro-sdd-design | F2 | Diseño técnico |
| 12 | zyro-sdd-tasks | F2 | Tareas atómicas |
| 13 | zyro-sdd-apply | F3 | Implementación |
| 14 | zyro-sdd-verify | F3 | Verificación |
| 15 | zyro-sdd-archive | F4 | Archivo y cierre |
| 16 | to-issues | — | GitHub Issues desde PRDs |

## 6. Comportamiento de herencia de modelo (OpenCode nativo)

Según la documentación de OpenCode Agents:
- Si un **agente primario** no tiene `model` → usa el modelo global configurado
- Si un **subagente** no tiene `model` → hereda el modelo del agente primario que lo invocó

Esto significa que ZyroCLI solo necesita asignar modelo al `zyro-orchestrator` y a cualquier agente que deba tener un modelo DIFERENTE al del orchestrator.

**Ejemplo de configuración mínima:**
```json
{
  "agent": {
    "zyro-orchestrator": {
      "model": "anthropic/claude-sonnet-4-20250514"
    },
    "zyro-sdd-apply": {
      "model": "opencode-go/deepseek-v4-flash"
    },
    "zyro-sdd-verify": {
      "model": "google/gemini-2.5-pro"
    }
  }
}
```
Los 13 agentes restantes heredarán el modelo del orchestrator automáticamente.

## 7. Providers soportados (validación)

Lista curada en `internal/opencode/models.go`:
- opencode-go: deepseek-v4-flash, deepseek-v4-pro, mimo-v2.5, etc.
- opencode (Free): deepseek-v4-flash-free, mimo-v2.5-free, etc.
- google: gemini-2.5-flash, gemini-2.5-pro
- groq: llama-4-scout-17b
- openrouter: qwen3-coder:free
- cerebras: gpt-oss-120b
- nvidia: (vacío)
- anthropic: claude-sonnet-4-6, claude-opus-4, claude-haiku-3-5

Además se cargan los providers configurados en opencode.json que tengan API keys.

## 8. Detección de entorno OpenCode

```go
func IsInsideOpenCode() bool {
    // Método 1: Variable de entorno
    if os.Getenv("OPENCODE_SESSION_ID") != "" {
        return true
    }
    // Método 2: Probar conexión al SDK
    conn, err := net.DialTimeout("tcp", "localhost:4096", 100*time.Millisecond)
    if err == nil {
        conn.Close()
        return true
    }
    return false
}
```

## 9. Plugin /zyro-model — Estructura del código

Archivo: `~/.config/opencode/plugins/zyro-model.ts`

```typescript
import type { Plugin } from "@opencode-ai/plugin";

interface AgentInfo {
  name: string;
  description: string;
  currentModel: string;
}

export const ZyroModelPlugin: Plugin = async ({ client }) => {
  return {
    command: {
      "zyro-model": async (args) => {
        // 1. Leer proveedores y agentes
        const { providers } = await client.config.providers();
        const config = await client.config.get();
        const agents = getZyroAgents(config.agent || {});
        
        // 2. Mostrar selector: Set All o agente individual
        const choice = await showMainMenu(agents);
        
        if (choice === "SET_ALL") {
          // 3a. Set All: elegir provider+modelo una vez
          const provider = await showProviderPicker(providers);
          const model = await showModelPicker(provider);
          // 4a. Escribir a todos los agentes
          for (const agent of agents) {
            await writeModel(agent.name, `${provider.id}/${model.id}`);
          }
        } else {
          // 3b. Agente individual
          const provider = await showProviderPicker(providers);
          const model = await showModelPicker(provider);
          // 4b. Escribir al agente
          await writeModel(choice, `${provider.id}/${model.id}`);
        }
        
        // 5. Confirmación
        await client.tui.showToast({
          message: "✓ Modelos actualizados en opencode.json",
          variant: "success"
        });
      }
    }
  };
};

function getZyroAgents(agentConfig: Record<string, any>): AgentInfo[] {
  const ZYRO_AGENTS = [
    { name: "zyro-orchestrator", desc: "Coordinador — solo habla y delega" },
    { name: "zyro-pre-f0", desc: "PRE-F0: Alineación de dominio" },
    { name: "zyro-phase-0-patterns", desc: "F0: Búsqueda de patrones" },
    { name: "zyro-phase-0-libraries", desc: "F0: Investigación de librerías" },
    { name: "zyro-skills-find", desc: "F0: Descubrimiento de skills" },
    { name: "zyro-skills-audit", desc: "F0: Validación de skills" },
    { name: "zyro-skills-apply", desc: "F0: Instalación de skills" },
    { name: "zyro-sdd-explore", desc: "F0: Exploración de codebase" },
    { name: "zyro-sdd-spec", desc: "F1: Especificación técnica" },
    { name: "zyro-sdd-propose", desc: "F2: Propuestas de cambio" },
    { name: "zyro-sdd-design", desc: "F2: Diseño técnico" },
    { name: "zyro-sdd-tasks", desc: "F2: Tareas atómicas" },
    { name: "zyro-sdd-apply", desc: "F3: Implementación" },
    { name: "zyro-sdd-verify", desc: "F3: Verificación" },
    { name: "zyro-sdd-archive", desc: "F4: Archivo y cierre" },
    { name: "to-issues", desc: "GitHub Issues desde PRDs" },
  ];
  
  return ZYRO_AGENTS.map(a => ({
    name: a.name,
    description: a.desc,
    currentModel: agentConfig[a.name]?.model || "(hereda del orchestrator)"
  }));
}
```

## 10. Migración desde el estado actual

1. Eliminar `tui-plugins/zyro-model.js` stale
2. Eliminar plugin `zyro-model.js` de `tui.json`
3. Eliminar referencias a `model-assigner` en docs
4. Crear `plugins/zyro-model.ts`
5. Modificar `internal/opencode/tui_plugins.go`: eliminar `zyro-model.js` de stale list
6. Reemplazar `profile_tui.go`: corregir agent IDs, añadir descripciones
7. Agregar `validateModel()` en `profile.go`
8. Agregar `IsInsideOpenCode()` en un nuevo archivo
