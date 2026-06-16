# Investigación: Ecosistema OpenCode — Plugins, Multi-Agente, MCP, Skills

> **Fecha**: 2026-06-15
> **Propósito**: Investigación técnica del ecosistema de OpenCode para identificar brechas de integración en ZyroAgentCLI y recomendar mejoras concretas.
> **Autor**: Investigador técnico
> **Estado**: Completo

---

## Resumen Ejecutivo

ZyroAgentCLI es un orquestador Go que actualmente configura OpenCode con **14 skills embebidas**, **6 MCP tools** (helix-integration), y un **pipeline F0→F4** con approval gates via CLI. Sin embargo, la integración tiene 3 brechas críticas:

1. **No usa plugins OpenCode** — ZyroAgentCLI escribe `opencode.jsonc` directamente, sin aprovechar `@opencode-ai/plugin` ni el ecosistema de plugins npm.
2. **No usa el claude-bridge** — los skills se escriben a mano en Go embed en vez de reutilizar skills del ecosistema Claude Code.
3. **No hay lazy-loading de MCP** — los 6 MCP tools se cargan siempre, incluso cuando no se necesitan.
4. **Aprobaciones humanas fragiles** — `PromptApproval()` usa stdin, que no funciona bien cuando OpenCode maneja el flujo.

El ecosistema OpenCode tiene 3 plugins clave (`opencode-multiagent`, `opencode-claude-bridge`, `opencode-lazy-loader`) y un sistema de plugins maduro (`@opencode-ai/plugin`). ZyroAgentCLI debería migrar a un modelo híbrido: **plugin bridge para skills + lazy MCP loading + comando slash nativo**.

---

## Arquitectura de OpenCode

### Estructura de Configuración (`opencode.jsonc`)

Basado en el schema oficial en `https://opencode.ai/config.json`:

```jsonc
{
  "$schema": "https://opencode.ai/config.json",

  // --- Agentes ---
  "agent": {
    "build": {
      "mode": "primary",        // primary | subagent | all
      "model": "provider/model",
      "variant": "default",
      "temperature": 0.7,
      "top_p": 0.9,
      "prompt": "{skill:my-skill}",  // o {file:path/to/file}
      "description": "Cuándo usar este agente",
      "hidden": false,           // solo para subagent
      "color": "#FF5733",        // hex o theme color
      "steps": 100,              // max agentic iterations
      "disable": false,
      "permission": {
        "read": "allow",
        "edit": "deny",
        "glob": "allow",
        "grep": "allow",
        "list": "allow",
        "bash": "deny",
        "task": {
          "*": "deny",
          "subagent-name": "allow"
        },
        "write": "deny",
        "webfetch": "allow",
        "websearch": "allow",
        "question": "allow",
        "skill": "deny",
        "lsp": "deny",
        "external_directory": "allow",
        "todowrite": "allow",
        "doom_loop": "ask"
      }
    }
  },

  // --- Default agent ---
  "default_agent": "build",

  // --- Comandos slash (/command) ---
  "command": {
    "my-command": {
      "template": "<command-instruction>Instrucción del comando</command-instruction>\n\n<user-request>\n$ARGUMENTS\n</user-request>",
      "description": "Descripción mostrada en /",
      "agent": "subagent-name",     // delegar a subagente
      "model": "provider/model",     // o heredar del agente
      "variant": "default",
      "subtask": true               // subtarea sin confirmación
    }
  },

  // --- MCP Servers ---
  "mcp": {
    "my-server": {
      "type": "local",
      "command": ["npx", "-y", "@some/mcp"],
      "cwd": "/path/to/workdir",
      "environment": {
        "API_KEY": "${MY_API_KEY}"
      },
      "enabled": true,
      "timeout": 5000
    },
    "remote-server": {
      "type": "remote",
      "url": "https://mcp.example.com/mcp",
      "headers": {
        "Authorization": "Bearer ${TOKEN}"
      },
      "oauth": {
        "clientId": "...",
        "scope": "read write"
      },
      "timeout": 10000
    }
  },

  // --- Skills ---
  "skills": {
    "paths": [
      "~/.config/opencode/skills",
      ".opencode/skills"
    ],
    "urls": [
      "https://example.com/.well-known/skills/"
    ]
  },

  // --- Plugins ---
  "plugin": [
    "opencode-multiagent",
    ["opencode-lazy-loader", { "option": "value" }],
    "./path/to/local-plugin"
  ],

  // --- Providers ---
  "provider": {
    "my-provider": {
      "name": "My Provider",
      "api": "https://api.example.com/v1",
      "env": ["MY_API_KEY"],
      "models": {
        "model-name": {
          "id": "model-id",
          "name": "Display Name",
          "family": "gpt",
          "cost": { "input": 0.01, "output": 0.03 },
          "limit": { "context": 128000, "output": 4096 },
          "tool_call": true,
          "reasoning": true,
          "status": "active"
        }
      }
    }
  },

  // --- Permisos globales ---
  "permission": {
    "read": "allow",
    "bash": "ask",
    "skill": { "skill-name": "deny" }
  },

  // --- Misc ---
  "model": "provider/default-model",
  "small_model": "provider/small-model",
  "shell": "/bin/bash",
  "logLevel": "INFO",
  "mode": "auto",           // deprecated — usar agent
  "instructions": ["AGENT.md", "CLAUDE.md"],
  "plugin": [],
}
```

### Campos Clave de Config

| Campo | Tipo | Propósito |
|-------|------|-----------|
| `agent[*].mode` | `primary│subagent│all` | Un `primary` aparece en el tab. Un `subagent` es invocado via `@name` o `agent(name)`. `all` hace ambas. |
| `agent[*].prompt` | string | `{skill:name}` carga el SKILL.md de un skill. `{file:path}` carga contenido de un archivo. |
| `agent[*].permission.task` | object | Controla qué subagentes puede invocar este agente. `"*": "deny"` + lista de allows. |
| `command[*].template` | string | El template se wrappea con `$ARGUMENTS` para el input del usuario. |
| `command[*].subtask` | boolean | Si es `true`, OpenCode no pide confirmación antes de ejecutar. |
| `mcp[*].type` | `local│remote` | Local ejecuta un proceso; remote conecta vía HTTP/SSE. |
| `plugin` | array | Plugins npm o rutas locales. También `["name", {opts}]`. |
| `skills.paths` | string[] | Directorios donde OpenCode busca `*/SKILL.md`. |

### Descubrimiento de Skills

OpenCode escanea (en orden):

1. `.opencode/skills/<name>/SKILL.md` (project-local OpenCode)
2. `~/.config/opencode/skills/<name>/SKILL.md` (global OpenCode)
3. `.claude/skills/<name>/SKILL.md` (project-local Claude compat)
4. `~/.claude/skills/<name>/SKILL.md` (global Claude compat)
5. Cualquier path en `config.skills.paths`
6. Cualquier URL en `config.skills.urls`

### Sistema de Plugins

OpenCode tiene un sistema de plugins via `@opencode-ai/plugin` (npm, >1M descargas/semana). Los plugins se cargan en tiempo de arranque y pueden:

- Modificar `config` via hooks (`config` hook)
- Registrar comandos
- Exponer herramientas MCP
- Proveer skills

Un plugin típico:
```typescript
import { definePlugin } from '@opencode-ai/plugin';

export default definePlugin({
  name: 'my-plugin',
  config: (config, client) => {
    // Modificar config antes de que OpenCode la use
    config.agent['my-agent'] = { ... };
    config.mcp['my-server'] = { ... };
    return config;
  },
});
```

---

## Plugins Disponibles

### 1. `opencode-multiagent` (vaur94)

**npm**: `npm i opencode-multiagent` · v1.0.0 · MIT

**Propósito**: Instala un plano de control multi-agente estructurado con 3 agentes primarios y 11 subagentes.

**Agentes**:
| Agente | Rol |
|--------|-----|
| `brainstormer` | Genera ideas, explora alternativas |
| `planner` | Descompone en tareas, ordena ejecución |
| `executor` | Implementa siguiendo el plan |

**Subagentes**: coding, review, research, documentation, knowledge management.

**Instalación**:
```json
{
  "plugin": ["opencode-multiagent"]
}
```

**Host-side overrides**:
```json
{
  "plugin": ["opencode-multiagent"],
  "agent": {
    "planner": {
      "steps": 300
    }
  }
}
```

**Documentación disponible**:
- `docs/configuration.md` — Flags, profiles, agent settings, MCP defaults
- `docs/usage-guide.md` — Installation, workflows, task board, profiles
- `docs/agents.md` — All primary agents and subagents, routing, models

**Relevancia para ZyroAgentCLI**: Alto. Proporciona una estructura multi-agente prefabricada que ZyroAgentCLI podría usar como base en lugar de definir todos los agentes manualmente.

---

### 2. `@sjawhar/opencode-claude-bridge` (sjawhar)

**npm**: `npm i @sjawhar/opencode-claude-bridge` · v0.5.0 · MIT

**Propósito**: Puente que registra agentes, comandos y skills de Claude Code en OpenCode mediante el hook `config` del plugin system.

**Características clave**:

1. **Bridge de fuentes Claude Code**: Escanea `<dir>/agents/*.md`, `<dir>/commands/*.md`, `<dir>/skills/*/SKILL.md` y los traduce a config de OpenCode.

2. **Registro dual de skills**: Cada skill se registra como skill OpenCode (via `config.skills.paths`) Y como slash-command (via `config.command`).

3. **Traducción agente Claude → OpenCode**:
   | Claude frontmatter | OpenCode config |
   |---|---|
   | `name` / filename | object key |
   | `description` | `description` |
   | `model: opus│sonnet│haiku` | `anthropic/claude-opus-4-6` / etc |
   | `tools: "Read, Edit"` | `{read: true, edit: true}` |
   | `color: <name>` | `color` (si es hex o theme color) |
   | body | `prompt` (sin frontmatter) |
   | (none) | `mode: "subagent"` |

4. **Traducción comando → slash command**:
   | Claude frontmatter | OpenCode config |
   |---|---|
   | filename | object key (sin `.md`) |
   | `description` | `description` |
   | body | wrappeado como `<command-instruction>...</command-instruction>\n\n<user-request>\n$ARGUMENTS\n</user-request>` |
   | `agent` | `agent` |
   | `model` | `model` |
   | `subtask` | `subtask` |
   | `handoffs` | `handoffs` |

5. **Skill MCP desde frontmatter**: OpenCode ignora el bloque `mcp:` en SKILL.md. El bridge lo parsea y registra en `config.mcp`.

6. **Marketplace discovery**: con `claudePlugins: true`, descubre plugins de Claude Code automáticamente.

7. **Control de superficie**: `disable-model-invocation: true` quita el skill del modelo pero mantiene el slash command. `user-invocable: false` quita el slash command pero mantiene el skill visible para el modelo.

**Colisión de nombres**: Usa `${namespace}/${name}` para agentes/commands, `${namespace}-${name}` para MCPs (para mantener compatibilidad con Anthropic tool names).

**Uso típico**:
```typescript
// ~/.config/opencode/plugins/my-bridge.ts
import { createClaudeBridge } from "@sjawhar/opencode-claude-bridge";
import path from "node:path";
import os from "node:os";

export const MyBridge = createClaudeBridge({
  sources: [
    { dir: path.join(os.homedir(), ".dotfiles/plugins/sjawhar"), namespace: "sjawhar" },
    { dir: ".claude" },
  ],
  claudePlugins: true,
});
```

**Relevancia para ZyroAgentCLI**: **Crítica**. ZyroAgentCLI actualmente escribe skills a `~/.config/opencode/skills/` manualmente (Go embed → WriteFile). El bridge permitiría:
- Skills declarativas en `.md` con frontmatter estándar
- Skills discoverables tanto como skill OpenCode como slash-command
- MCP embebido en frontmatter de skill
- Reutilización de skills del ecosistema Claude Code
- Sin código Go para escribir skills

---

### 3. `opencode-lazy-loader` (fka `opencode-embedded-skill-mcp`)

**npm**: `npm i opencode-lazy-loader` · v1.0.3 · MIT

**Propósito**: Plugin que permite que los skills tengan sus propios servidores MCP embebidos, cargados bajo demanda (lazy loading).

**Características**:

1. **MCP embebido en skills**: Define servidores MCP en el frontmatter YAML del skill o en `mcp.json` dentro del directorio del skill.

2. **Lazy loading**: Los servidores MCP se conectan solo al primer uso, no al arranque.

3. **Auto-cleanup**: Desconexión automática tras 5 minutos de inactividad.

4. **Pooling de conexiones**: Por sesión/skill/servidor.

5. **Expansión de variables de entorno**: Soporte para `${VAR}` y `${VAR:-default}`.

**Ejemplo de skill con MCP embebido**:
```markdown
---
name: browser-automation
description: "Browser automation via Playwright MCP"
mcp:
  playwright:
    command: ["npx", "-y", "@playwright/mcp@latest"]
---

# Browser Automation

This skill provides browser automation tools via the `playwright` MCP.
```

**Uso**:
```
skill(name="browser-automation")
skill_mcp(mcp_name="playwright", tool_name="screenshot", arguments='{"url": "https://google.com"}')
```

**Herramientas expuestas**:
- `skill(name)` — Carga un skill y muestra sus capacidades MCP
- `skill_mcp(mcp_name, tool_name, arguments)` — Invoca un tool en un MCP de skill

**Relevancia para ZyroAgentCLI**: Alta. Los 6 MCP tools de helix-integration se cargan siempre. Con lazy-loader, se cargarían solo cuando el skill que los necesita está activo. Esto reduce:
- Consumo de memoria
- Latencia de arranque de OpenCode
- Complejidad de configuración global

---

### 4. `@opencode-ai/plugin`

**npm**: `@opencode-ai/plugin` · v1.17.7 · MIT · >5M descargas/semana

**Propósito**: SDK para construir plugins de OpenCode.

**Dependencia externa** para todos los plugins de OpenCode. No tiene README en npm pero es la base del sistema de plugins. Los plugins se especifican en `config.plugin` como strings (nombre npm) o arrays `["nombre", {opts}]`.

---

## Cómo Registrar Comandos Slash (como `/zyro-model`)

### Método 1: Directo en `opencode.jsonc`

El comando `/zyro-model` actual de ZyroAgentCLI:

```jsonc
{
  "command": {
    "zyro-model": {
      "template": "zyrocli profile tui",
      "description": "Seleccionar modelos por fase SDD",
      "subtask": true,
      "agent": "model-assigner"
    }
  }
}
```

**Campos del comando**:
| Campo | Obligatorio | Descripción |
|-------|-------------|-------------|
| `template` | ✅ | Instrucción + `$ARGUMENTS` placeholder |
| `description` | ❌ | Texto mostrado en el menú `/` |
| `agent` | ❌ | Subagente que ejecuta el comando |
| `model` | ❌ | Modelo específico (sobreescribe el del agente) |
| `variant` | ❌ | Variante del modelo |
| `subtask` | ❌ | `true` = sin confirmación previa |

### Método 2: Via archivo `.md` estilo Claude Code → bridge

```markdown
---
description: Seleccionar modelos por fase SDD
subtask: true
---

!`zyrocli profile tui`
```

Con el bridge, el frontmatter se traduce automáticamente a `config.command`.

### Método 3: Via plugin system

```typescript
export default definePlugin({
  name: 'zyro-commands',
  config: (config) => {
    config.command['zyro-model'] = {
      template: "zyrocli profile tui",
      description: "Seleccionar modelos por fase SDD",
      subtask: true,
      agent: "model-assigner",
    };
    return config;
  },
});
```

### Template Wrapping

OpenCode wrappea el `template` automáticamente:

```
<command-instruction>
[template content]
</command-instruction>

<user-request>
[lo que el usuario escribe después de /command]
</user-request>
```

Para comandos sin argumentos (como `/zyro-model`), el `$ARGUMENTS` se reemplaza con el input del usuario (que puede estar vacío).

---

## Cómo Integrar Aprobaciones Humanas con OpenCode

### Problema Actual

ZyroAgentCLI usa `PromptApproval()` en Go que lee de stdin:
```go
func PromptApproval(phase Phase, summary string) (bool, error) {
    fmt.Printf("--- Phase %s complete ---\n", phase)
    fmt.Print("Approve? (y/n): ")
    input, _ := stdinReader.ReadString('\n')
    // ...
}
```

Esto **no funciona cuando OpenCode maneja el flujo** porque:
- OpenCode captura stdin/stdout
- El proceso Go bloquea esperando input que nunca llega
- No hay integración real entre el scheduler Go y el chat de OpenCode

### Soluciones con OpenCode

#### Opción A: Usar `subtask: true` + delegación a subagente

```jsonc
{
  "command": {
    "zyro-approve-f0": {
      "template": "Verifica que Fase 0 esté completa: patrones, librerías y skills en HelixDB. Pregunta al humano si aprueba. Responde solo 'approved' o 'rejected'.",
      "description": "Aprobar Fase 0",
      "subtask": false,
      "agent": "zyro-sdd-verify"
    }
  }
}
```

#### Opción B: Subagente con `question: "ask"` en permisos

El subagente puede preguntar al humano usando la tool `question`:
```jsonc
{
  "zyro-sdd-verify": {
    "mode": "subagent",
    "permission": {
      "question": "ask"
    }
  }
}
```

Cuando el subagente necesita aprobación, pide al humano via el chat de OpenCode. El permiso `"ask"` significa que OpenCode pregunta al usuario antes de dejar pasar la tool call.

#### Opción C: Ciclo humano-en-el-loop via skill

El skill le dice al modelo que pregunte al humano:

```markdown
---
name: zyro-approval-gate
description: "Gate de aprobación humana entre fases SDD"
---

# Approval Gate

## Instrucciones
1. Resume lo que se completó en la fase actual
2. PREGUNTA al humano: "¿Aprobás esta fase para continuar?"
3. Espera respuesta del humano
4. Si "sí" → continúa
5. Si "no" → aborta y reporta
```

#### Opción D: Usar permission `doom_loop` y `continue_loop_on_deny`

OpenCode soporta `continue_loop_on_deny: true` en `experimental` y `doom_loop` permission. Esto permite ciclos de approve/retry.

### Recomendación

Migrar de `PromptApproval()` en Go a **subagentes con `question: "ask"`** y skills que definen el protocolo de aprobación. El scheduler Go debe lanzar OpenCode y dejar que los subagentes manejen las interacciones humanas, en vez de intentar leer stdin.

---

## Multi-Agente: Cómo Orquestar Subagentes desde OpenCode

### Mecanismo Nativo de OpenCode

1. **`agent(name)` tool**: El agente orquestador invoca un subagente
   ```
   agent(name="zyro-sdd-explore", args="Explora la estructura actual del proyecto")
   ```

2. **`@name` autocomplete**: En el chat, `@zyro-sdd-apply` selecciona el agente

3. **Delegación con tareas**: El agente primario crea tareas que son ejecutadas por subagentes

4. **Permission `task`**: Controla qué subagentes puede invocar cada agente

### Patrón de Orquestación con Permisos

El orquestador (`zyro-orchestrator`) tiene permisos restrictivos:
```jsonc
{
  "zyro-orchestrator": {
    "mode": "primary",
    "prompt": "{skill:zyro-orchestrator}",
    "permission": {
      "read": "allow",
      "task": {
        "*": "deny",
        "zyro-sdd-explore": "allow",
        "zyro-sdd-apply": "allow",
        "zyro-sdd-verify": "allow"
      },
      "write": "deny",
      "edit": "deny",
      "bash": "deny"
    }
  }
}
```

Esto fuerza al orquestador a delegar siempre — no puede escribir, editar, o ejecutar bash directamente.

### Paralelización

OpenCode **no tiene paralelización nativa de subagentes**. Para ejecutar múltiples subagentes en paralelo (como Fase 0 actual con patterns + libraries + skills-find), se necesita:

1. **Plugin multi-agent** (`opencode-multiagent`) que sí soporta ejecución paralela
2. **Un subagente "coordinador"** que lanza sub-subagentes secuencialmente
3. **MCP server externo** que maneja la paralelización

### Plugin `opencode-multiagent`

Proporciona un control plane con:
- 3 agentes primarios (brainstormer, planner, executor)
- 11 subagentes
- Task board compartido
- File locks
- Telemetría
- MCP defaults
- Profiles

### Recomendación

ZyroAgentCLI debería:
1. Usar `opencode-multiagent` como base para el sistema multi-agente
2. Reemplazar los agentes manuales de `install.go` con agentes del plugin
3. Mantener el orquestador como `default_agent`
4. Para paralelización real, evaluar si el plugin multi-agent lo soporta o construir un MCP coordinator

---

## Protocolo Boomerang (Memory → Think → Delegate → Git → Quality → Save)

> **Nota**: El protocolo Boomerang es un concepto propio de ZyroAgentCLI. **No es parte de OpenCode**. Es el protocolo de 6 pasos que ZyroAgentCLI implementa para el pipeline de desarrollo.

### Descripción Detallada de los 6 Pasos

#### 1. Memory (Memoria)
- **Output**: Archivos `.zyro/`, nodos en HelixDB
- **Implementación actual**: `internal/opencode/skills_embed.go`, `internal/db/helix/`
- **OpenCode equivalente**: Skills con `{skill:name}` prompt
- **Brecha**: No hay persistencia automática de contexto entre sesiones de OpenCode. ZyroAgentCLI escribe a HelixDB, pero OpenCode no lee automáticamente de HelixDB.

#### 2. Think (Pensamiento/Análisis)
- **Output**: Nodos Pattern, Library, Skill en HelixDB
- **Implementación actual**: Subagentes zyro-phase-0-patterns, zyro-phase-0-libraries
- **OpenCode equivalente**: Subagentes con skills de investigación
- **Brecha**: Los subagentes no tienen acceso nativo a HelixDB. Dependen de MCP tools (helix-integration).

#### 3. Delegate (Delegación)
- **Output**: Tareas asignadas a subagentes
- **Implementación actual**: Permission `task` en opencode.jsonc
- **OpenCode equivalente**: `agent(name)` tool + permission `task`
- **Brecha**: La delegación actual usa solo permisos estáticos. No hay routing dinámico basado en carga o especialización.

#### 4. Git (Control de Versiones)
- **Output**: Commits, branches
- **Implementación actual**: No hay integración git en ZyroAgentCLI (explícitamente: "NO hacer commit, push ni crear PRs")
- **OpenCode equivalente**: OpenCode tiene integración git nativa (Read, Edit, diff)
- **Brecha**: ZyroAgentCLI no orquesta git. Queda a criterio del humano.

#### 5. Quality (Calidad)
- **Output**: Nodos Review en HelixDB
- **Implementación actual**: Subagente zyro-sdd-verify, loop apply→verify
- **OpenCode equivalente**: Ciclo `agent(apply)` → `agent(verify)` con reintentos
- **Brecha**: La verificación depende de que el subagente cree un nodo Review en HelixDB. No hay validación automática de código.

#### 6. Save (Archivo)
- **Output**: Nodo Archive en HelixDB, Project.status = "archived"
- **Implementación actual**: F4Runner en Go
- **OpenCode equivalente**: Subagente zyro-sdd-archive
- **Brecha**: El archivado es manual (correr `zyrocli run --phase F4`). No hay "done" detection.

### Mapa Boomerang ↔ OpenCode

| Paso | ZyroAgentCLI Actual | OpenCode Nativo | Plugin Necesario |
|------|--------------------|-----------------|------------------|
| Memory | HelixDB + Skills | `{file:...}`, `{skill:...}` | claude-bridge |
| Think | Subagentes F0 | Agent delegation | multiagent |
| Delegate | Permission task | `agent(name)` + task | multiagent |
| Git | No orquestado | Nativo en OpenCode | — |
| Quality | Apply→Verify loop | Subtask + permission | — |
| Save | F4Runner Go | Agent archive | — |

---

## `opencode.jsonc` — Estructura Completa y Campos Clave

Basado en el schema oficial (`https://opencode.ai/config.json`):

### Top-Level

```typescript
interface Config {
  $schema?: string;
  shell?: string;                              // Default shell
  logLevel?: "DEBUG" | "INFO" | "WARN" | "ERROR";
  server?: ServerConfig;                       // HTTP server for web/ssh
  command?: Record<string, CommandConfig>;     // Slash commands
  skills?: SkillsConfig;                       // Skills paths/urls
  references?: Record<string, ReferenceConfig>;// Named git/local refs
  watcher?: { ignore: string[] };
  snapshot?: boolean;                          // Enable/disable file snapshots
  plugin?: (string | [string, object])[];      // Plugins
  share?: "manual" | "auto" | "disabled";
  autoupdate?: boolean | "notify";
  disabled_providers?: string[];
  enabled_providers?: string[];
  model?: string;                              // Default model
  small_model?: string;                        // Model for lightweight tasks
  default_agent?: string;                      // Primary agent por defecto
  username?: string;
  mode?: Record<string, AgentConfig>;          // DEPRECATED, use agent
  agent?: Record<string, AgentConfig>;         // Agent definitions
  provider?: Record<string, ProviderConfig>;   // Custom providers
  mcp?: Record<string, McpConfig>;             // MCP servers
  formatter?: boolean | Record<string, FormatterConfig>;
  lsp?: boolean | Record<string, LSPConfig>;
  instructions?: string[];                     // Additional instruction files
  permission?: PermissionConfig;               // Global permissions
  tools?: Record<string, boolean>;             // Enable/disable tools
  attachment?: AttachmentConfig;               // Image processing
  enterprise?: { url: string };
  tool_output?: { max_lines: number; max_bytes: number };
  compaction?: CompactionConfig;               // Context management
  experimental?: ExperimentalConfig;
}
```

### AgentConfig

```typescript
interface AgentConfig {
  model?: string;              // provider/model
  variant?: string;            // Model variant
  temperature?: number;
  top_p?: number;
  prompt?: string;             // {skill:name} or {file:path}
  tools?: Record<string, boolean>;  // DEPRECATED, use permission
  disable?: boolean;
  description?: string;        // When to use this agent
  mode?: "subagent" | "primary" | "all";
  hidden?: boolean;            // Hide from autocomplete
  options?: object;
  color?: string;              // Hex #RRGGBB or theme color
  steps?: number;              // Max agentic iterations
  maxSteps?: number;           // DEPRECATED, use steps
  permission?: PermissionConfig;
}
```

### CommanderConfig

```typescript
interface CommandConfig {
  template: string;            // Instruction template with $ARGUMENTS
  description?: string;        // Shown in / menu
  agent?: string;              // Delegate to subagent
  model?: string;              // Override model
  variant?: string;
  subtask?: boolean;           // Skip confirmation
}
```

### MCP Config

```typescript
interface McpLocalConfig {
  type: "local";
  command: string[];           // Command + args
  cwd?: string;                // Working directory
  environment?: Record<string, string>;
  enabled?: boolean;
  timeout?: number;            // ms, default 5000
}

interface McpRemoteConfig {
  type: "remote";
  url: string;
  enabled?: boolean;
  headers?: Record<string, string>;
  oauth?: McpOAuthConfig | false;
  timeout?: number;
}
```

### PermissionConfig

```typescript
interface PermissionConfig {
  read?: PermissionRule;       // "ask" | "allow" | "deny" | Record<string, action>
  edit?: PermissionRule;
  glob?: PermissionRule;
  grep?: PermissionRule;
  list?: PermissionRule;
  bash?: PermissionRule;
  task?: PermissionRule;       // Subagent delegation
  external_directory?: PermissionRule;
  todowrite?: PermissionAction;
  question?: PermissionAction;
  webfetch?: PermissionAction;
  websearch?: PermissionAction;
  lsp?: PermissionRule;
  doom_loop?: PermissionAction;
  skill?: PermissionRule;      // Skill loading
  write?: PermissionRule;      // Write files
  [key: string]: PermissionRule | undefined;
}
```

---

## Recomendaciones Concretas para ZyroAgentCLI

### 🔴 Prioridad Alta — Brechas Críticas

#### 1. Migrar a Plugin System (reemplazar escritura directa de JSON)

**Problema**: `install.go` escribe `opencode.jsonc` manualmente. No se pueden actualizar configuraciones sin reinstalar.

**Solución**: Crear un plugin OpenCode que modifique `config` vía hook.

```typescript
// ~/.config/opencode/plugins/zyrocli.ts
import { definePlugin } from '@opencode-ai/plugin';
import { createClaudeBridge } from '@sjawhar/opencode-claude-bridge';
import path from 'path';
import os from 'os';

export default createClaudeBridge({
  sources: [
    { dir: path.join(os.homedir(), '.config/zyrocli/skills'), namespace: 'zyro' },
  ],
  claudePlugins: true,
});
```

Esto reemplaza completamente `internal/opencode/config.go` + `install.go` para la escritura de config.

#### 2. Usar `opencode-claude-bridge` para skills

**Problema**: 14 skills embebidas en Go (`skills_embed.go`) escritas a `~/.config/opencode/skills/` manualmente.

**Solución**: Migrar skills a formato Claude Code estándar + bridge. Los skills vivirían como archivos `.md` en `~/.config/zyrocli/skills/` y el bridge los registraría automáticamente.

**Beneficios**:
- Skills discoverables como slash-commands
- Frontmatter `mcp:` para MCP embebido
- `user-invocable: false` y `disable-model-invocation: true` para control granular
- Sin código Go para deploy de skills
- Idempotente — bridge maneja caché y limpieza

#### 3. Implementar lazy-loading de MCP tools

**Problema**: 6 MCP tools de helix-integration se cargan siempre, incluso cuando no se usan.

**Solución**: Migrar MCP tools a `opencode-lazy-loader`:
```markdown
---
name: helix-integration
description: "HelixDB integration — search code, skills, task context"
mcp:
  helix:
    command: ["uv", "run", "--directory", "~/.config/zyrocli/mcp-tools", "runner.py"]
---
```

O usar `config.permission.skill` para denegar skills específicas hasta que se necesiten.

### 🟡 Prioridad Media — Mejoras Significativas

#### 4. Reemplazar aprobaciones stdin con subagentes

**Problema**: `PromptApproval()` en `internal/scheduler/approval.go` usa stdin, incompatible con OpenCode.

**Solución**: 
- Migrar approval gates a un skill `zyro-approval-gate`
- El subagente verifica precondiciones y pregunta al humano via `question` tool
- El scheduler Go deja de manejar aprobaciones y solo lanza OpenCode
- Usar `permission.question = "ask"` para gates críticos

#### 5. Evaluar `opencode-multiagent` como base

**Problema**: ZyroAgentCLI define 13 subagentes manualmente, sin routing dinámico.

**Solución**: Evaluar si el plugin `opencode-multiagent` puede reemplazar la definición manual de agentes. Si los patrones de ZyroAgentCLI (SDD phases) encajan en brainstormer/planner/executor, usarlo. Si no, crear un plugin custom.

#### 6. Pipeline F0-F4 como comandos slash

**Problema**: El pipeline se ejecuta via `zyrocli run [--phase]`, que es un proceso Go externo.

**Solución**: Registrar cada fase como comando slash:
```jsonc
{
  "command": {
    "zyro-phase-0": {
      "template": "Ejecuta Fase 0: investigación de patrones, librerías y skills. Usa agent(name) para delegar a zyro-phase-0-patterns, zyro-phase-0-libraries, zyro-skills-find. Reporta resultados y pide aprobación.",
      "description": "F0: Investigación inicial",
      "subtask": false,
      "agent": "zyro-orchestrator"
    }
  }
}
```

### 🟢 Prioridad Baja — Optimizaciones

#### 7. Skills como repositorio externo

Extraer las 14 skills de Go embed a un repositorio git externo (`zyro-skills`) instalable via `zyrocli install --from-repo`. Esto permite:
- Actualizar skills sin recompilar Go
- Community contributions
- Versionado semántico de skills

#### 8. Telemetría y monitoreo de plugins

Agregar logging estructurado para saber qué plugins/skills se usan y cómo. El bridge de Claude ya soporta logging via `client.app.log()`.

#### 9. Simplificar `internal/opencode/`

El paquete `internal/opencode/` actualmente tiene 3 responsabilidades (config, skills embed, mcptools embed). Con el bridge y lazy-loader, esto se reduce a:
- `config.go` → solo lectura/escritura de perfiles de modelos (para `/zyro-model`)
- Los skills y MCP tools los manejan los plugins

---

## Diagrama de Arquitectura Propuesta

```
┌──────────────────────────────────────────────────────────┐
│                   OpenCode                               │
│                                                          │
│  ┌──────────────┐  ┌──────────────────┐  ┌────────────┐ │
│  │ zyro-orches- │  │  opencode-       │  │ opencode-  │ │
│  │ trator       │──│  claude-bridge   │──│ lazy-loader│ │
│  │ (primary)    │  │  (plugin)        │  │ (plugin)   │ │
│  └──────┬───────┘  └────────┬─────────┘  └─────┬──────┘ │
│         │                   │                    │        │
│         │ delegates to      │ bridges            │ lazy   │
│         ▼                   ▼                    ▼ loads │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐│
│  │ 13 subagentes│  │ Claude Code  │  │ Helix MCP tools ││
│  │ SDD + Phase0 │  │ Skills .md   │  │ (bajo demanda)  ││
│  └──────────────┘  └──────────────┘  └─────────────────┘│
└──────────────────────────────────────────────────────────┘
         │
         │ zyrocli install/configura
         ▼
┌──────────────────────────────────────────────────────────┐
│                   ZyroCLI (Go)                            │
│                                                          │
│  ┌─────────────┐  ┌──────────┐  ┌──────────────────────┐│
│  │ zyrocli     │  │ Scheduler│  │ HelixDB Go SDK       ││
│  │ init/run/   │──│ (F0-F4)  │──│ (writes controlados) ││
│  │ install     │  │          │  │                      ││
│  └─────────────┘  └──────────┘  └──────────────────────┘│
└──────────────────────────────────────────────────────────┘
```

---

## Referencias

1. **Schema oficial OpenCode**: `https://opencode.ai/config.json` — Schema JSON completo de configuración
2. **opencode-multiagent**: `https://www.npmjs.com/package/opencode-multiagent` — Plugin multi-agente
3. **opencode-claude-bridge**: `https://www.npmjs.com/package/@sjawhar/opencode-claude-bridge` — Bridge Claude→OpenCode
4. **opencode-lazy-loader**: `https://www.npmjs.com/package/opencode-lazy-loader` — Lazy MCP para skills
5. **@opencode-ai/plugin**: `https://www.npmjs.com/package/@opencode-ai/plugin` — SDK de plugins OpenCode
6. **ZyroAgentCLI**: `https://github.com/secko/zyrocli` — Repo actual (privado/local)
7. **OpenCode (Tencent Source)**: `https://github.com/tencent-source/opencode` — Repo oficial (404 verificado 2026-06-15, posible renombre o migración)

---

*Fin del informe de investigación*
