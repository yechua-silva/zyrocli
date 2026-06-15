# ZyroCLI — Orquestador Go para desarrollo asistido por IA

ZyroCLI configura y orquesta el pipeline de desarrollo F0→F1→F2→F3→F4 usando agentes de IA sobre OpenCode, con HelixDB como fuente de verdad.

## Requisitos previos

Antes de instalar, necesitás tener instalado en tu sistema:

- **Go 1.26+** — para compilar el binario (`go version`)
- **OpenCode** — agente de IA (`npx opencode --version` o instalado globalmente)
- **uv** — gestor de proyectos Python (`uv --version`)
- **Docker** — para HelixDB (`docker --version`)
- **Node.js 18+** — para npx y skills.sh (`node --version`)

## Instalación

```bash
# 1. Clonar el repositorio
git clone https://github.com/secko/zyrocli
cd zyrocli

# 2. Compilar e instalar todo
./scripts/install.sh
```

### Qué esperar

Durante la instalación vas a ver:

```
🚀 Installing ZyroCLI...
  1. Building binary...                  ✅ zyrocli en ~/.local/bin/
  2. Installing HelixDB...               ✅ Docker container (localhost:6969)
  3. Installing ZyroCLI ecosystem...     ✅ 14 skills + 6 MCP tools + agentes
  4. Installing find-skills skill...     ✅ descubrimiento global de skills
```

### Post-instalación

Verificá que todo funciona:

```bash
zyrocli --help           # Ver comandos disponibles
zyrocli profile list     # Ver 14 agentes configurados
```

## Primeros pasos

```bash
# 1. Creá un handoff.yaml con la descripción de tu proyecto
# 2. Inicializá el proyecto
zyrocli init docs/examples/test-frontend-handoff.yaml

# 3. Se abre OpenCode. Escribí "iniciemos"
# 4. Fase 0 corre automáticamente (3 subagentes en paralelo)
# 5. Revisá resultados, aprobá skills, seguí con F1→F4
```

## Stack

- **Go 1.26+** — CLI (Cobra), scheduler, state-gating
- **OpenCode** — runtime de agentes (MCP stdio)
- **HelixDB** — grafo+vector persistente (localhost:6969)
- **Python** — MCP tools (PydanticAI + httpx) vía `uv run`
- **NVIDIA SkillSpector** — validación de seguridad de skills
- **skills.sh** — descubrimiento de skills comunitarias

## Instalación

```bash
git clone https://github.com/secko/zyrocli
cd zyrocli && ./scripts/install.sh
```

Esto:
1. Compila el binario `zyrocli` y lo instala en `~/.local/bin/`
2. Instala HelixDB (Docker) si no está presente
3. Ejecuta `zyrocli install` para configurar el ecosistema global

Si preferís hacerlo paso a paso:

```bash
cd zyrocli && go build -o ~/.local/bin/zyrocli ./cmd/zyrocli
zyrocli install
```

## Comandos

| Comando | Descripción |
|---------|-------------|
| `zyrocli install` | (Re)configura el ecosistema global (skills, agentes, MCP) |
| `zyrocli init <handoff.yaml>` | Crea proyecto desde handoff y abre OpenCode |
| `zyrocli run [--phase F0\|F1\|F2\|F3\|F4]` | Ejecuta el pipeline completo o una fase específica |
| `zyrocli profile list\|set\|tui` | Gestiona modelos por agente/asignación |
| `zyrocli context <id>` | Obtiene contexto de tarea o proyecto desde HelixDB |
| `zyrocli skill-advisor <query>` | Busca skills, valida con SkillSpector, puntúa |

## Pipeline F0→F4

Cada fase requiere **aprobación humana explícita** antes de continuar. No hay modo automático.

```
F0: Investigación — patrones, librerías, skills
  ↓ [¿Aprobás?]
F1: Especificación técnica — arquitectura, módulos, dependencias
  ↓ [¿Aprobás?]
F2: Diseño + planificación — componentes, tareas atómicas
  ↓ [¿Aprobás?]
F3: Implementación — apply ↔ verify (loop, máx 5 intentos)
  ↓ [¿Aprobás?]
F4: Cierre — archive, limpieza
```

### Fase 0 en detalle

Ejecuta 5 subagentes atómicos en paralelo:

| Subagente | Qué hace | Output en HelixDB |
|-----------|----------|-------------------|
| `zyro-phase-0-patterns` | Busca proyectos similares + patrones | Nodos `Pattern` |
| `zyro-phase-0-libraries` | Investiga librerías recomendadas | Nodos `Library` |
| `zyro-skills-find` | Busca skills en skills.sh | Nodos `Skill` |
| `zyro-skills-audit` | Valida seguridad (Gen Agent, Socket, Snyk) | Update `Skill` |
| `zyro-skills-apply` | Instala skills aprobadas por el humano | Edge `REQUIRES_SKILL` |

Go verifica en HelixDB que los nodos se hayan creado antes de dar la fase por completa.

## Arquitectura

```
Humano ↔ zyro-orchestrator (default, plan mode)
              │
              ├── zyro-sdd-explore    → leer código (solo lectura)
              ├── zyro-sdd-spec       → especificación técnica
              ├── zyro-sdd-design     → diseño de componentes
              ├── zyro-sdd-tasks      → planificación de tareas
              ├── zyro-sdd-apply      → implementación (escribe código)
              ├── zyro-sdd-verify     → verificación contra specs
              ├── zyro-sdd-archive    → cierre del proyecto
              ├── zyro-phase-0-*      → Fase 0
              └── model-assigner      → /zyro-model (cambiar modelos)
```

Los agentes se comunican mediante **HelixDB (nodos + edges)**, no texto libre. Go verifica el estado en HelixDB entre cada fase. Si un agente no escribió los nodos esperados, la fase falla.

## Skills embebidas (14)

Todas las skills están compiladas dentro del binario. Se instalan en `~/.config/opencode/skills/` al ejecutar `zyrocli install`.

| Skill | Propósito |
|-------|-----------|
| `zyro-orchestrator` | Orquestador principal (default) |
| `zyro-phase-0-patterns` | Busca patrones similares |
| `zyro-phase-0-libraries` | Investiga librerías |
| `zyro-skills-find` | Descubre skills en skills.sh |
| `zyro-skills-audit` | Valida seguridad de skills |
| `zyro-skills-apply` | Instala skills aprobadas |
| `zyro-sdd-spec` | Especificación técnica |
| `zyro-sdd-design` | Diseño técnico |
| `zyro-sdd-tasks` | Planificación de tareas |
| `zyro-sdd-apply` | Implementación de código |
| `zyro-sdd-verify` | Verificación contra specs |
| `zyro-sdd-explore` | Investigación de código |
| `zyro-sdd-propose` | Propuesta de cambios |
| `zyro-sdd-archive` | Cierre y archive |

## MCP Tools (6)

Servidor MCP en Python que expone herramientas para interactuar con HelixDB:

| Tool | Descripción |
|------|-------------|
| `task_context(id)` | Contexto completo de tarea (skills, code, docs, patterns) |
| `search_code(query)` | Búsqueda de código en HelixDB |
| `search_skills(query)` | Búsqueda de skills en HelixDB |
| `save_to_helix(label, properties)` | Crea nodo con validación de campos required |
| `link_to_project(project_id, target, edge)` | Crea edge desde Project a otro nodo |
| `find_project(name)` | Busca proyecto por nombre |

## Configuración global

El ecosistema se configura automáticamente en:

```
~/.config/opencode/opencode.jsonc   → 14 agentes, 6 MCP tools, /zyro-model
~/.config/opencode/skills/           → 14 skills
~/.config/zyrocli/mcp-tools/         → MCP server Python
```

## Desarrollo

```bash
# Compilar
go build -o ~/.local/bin/zyrocli ./cmd/zyrocli

# Tests
go test ./... -timeout 120s

# Re-instalar ecosistema después de cambios
zyrocli install
```

## Licencia

MIT
