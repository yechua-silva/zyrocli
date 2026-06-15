# ZyroCLI — Orquestador Go para desarrollo asistido por IA

## ⚠️ Reglas del agente (LEER PRIMERO)
- **NO hacer commit, push ni crear PRs** — salvo orden explícita del humano
- **NO decidir por tu cuenta** — si algo no está especificado en el handoff.yaml o en las fases, PREGUNTAR al humano
- **Solo orquestar** — tu trabajo es coordinar subagentes y fases, NO implementar código directamente
- **Usar SDD** — tienes skills sdd-apply, sdd-verify, sdd-explore, sdd-propose, sdd-spec, sdd-design, sdd-tasks, sdd-archive para delegar trabajo a subagentes. DELEGAR siempre que una tarea cruce los triggers del orquestador (4+ archivos, multi-file write, etc.)
- **NO inflar contexto** — si una tarea requiere leer 4+ archivos o escribir en 2+ archivos no triviales, delegar a subagente. No hacerlo inline.
- **Idioma** — responder siempre en el mismo idioma del humano

## Stack
- Go 1.22+, module: `github.com/secko/zyrocli`
- OpenCode MCP (engram, graphify)
- skills.sh para descubrimiento de skills
- Context (Neuledge) `npm i -g @neuledge/context` + GitMCP fallback

## Estructura del proyecto
```
zyrocli/
├── cmd/zyrocli/main.go        # Entry point (cobra CLI)
├── internal/
│   ├── scheduler/             # State machine de 4 fases
│   │   └── scheduler.go       # DAG executor con goroutines + channels
│   ├── handoff/               # Parsea handoff.yaml
│   │   └── payload.go         # Structs Go del contrato Holdin Admin
│   ├── skilladvisor/          # Go nativo, determinístico
│   │   ├── registry.go        # Carga registry YAML local
│   │   ├── score.go           # ScoreSkill() tags-vector dot product
│   │   └── discover.go        # skills.sh API cache
│   ├── spec/                  # C-I-O DSL
│   │   ├── cio.go             # Structs Contract/Interface/Behavior/Constraints/Operation/Testing
│   │   └── compile.go         # Compilación opcional → OpenAPI/protobuf
│   ├── context/               # Integración Context MCP server
│   │   └── bridge.go          # Arranca `context serve --libs`
│   ├── apply/                 # Implementación contra C-I-O
│   │   └── runner.go          # Task runner con goroutines
│   └── test/                  # Contract testing
│       ├── contracts.go       # given/when/then executor
│       └── report.go          # Reporte con graphify diff
├── .opencode/
│   └── skills/                # Skills instaladas per-repo por npx skills add
├── AGENT.md
├── handoff.yaml               # Input de Holdin Admin
└── zyro-skill-overrides.yaml  # Skills custom NO en skills.sh
```

## Flujo de 4 macro-fases (scheduler state machine)

### F1: Planificación
Hand-off → Exploración (Python) + Skill Advisor (Go) → [VALIDACIÓN HUMANA]

**Hand-off**: `internal/handoff/payload.go` parsea handoff.yaml. Crea repo con `git init`. Indexa workspace. `SaveState(engram, topic: "zyro/{project}/handoff")`

**Skill Advisor**: Corre en paralelo con agentes Python. `internal/skilladvisor/score.go` hace ScoreSkill determinístico:
- language match → +10, framework → +20, project_type → +30
- verified_publisher (anthropics, microsoft, nvidia...) → +50
- Gen Agent Trust Hub = Safe → +25, Socket 0 alerts → +15
- Publica recomendación en `zyro/{project}/exploration`

**Humano**: Aprueba exploración + skills recomendadas. Si rechaza → ajusta parámetros y re-lanza.

### F2: Especificación
Contextualización → Especificación C-I-O → [VALIDACIÓN HUMANA]

**Contextualización**: `internal/context/bridge.go` arranca MCP server de Context o GitMCP para docs de librerías detectadas. graphify mapea estructura.

**C-I-O DSL**: `internal/spec/cio.go`. Convierte HU del handoff.yaml a contratos:
```go
type CIO struct {
    Contract   Contract   // name, description, language, deps
    Interface  []IOMethod // name, input, output, errors (SIN HTTP)
    Behavior   []Rule     // reglas de negocio
    Constraint Constraint // latency, security, storage
    Operation  Operation  // workflows + state_machine
    Testing    Testing    // contracts with given/when/then
}
```
Agente de aplicación NUNCA ve HU original.

**Generación del AGENT.md del proyecto destino**: ZyroCLI crea un `AGENT.md` ultra-condensado (~350 caracteres) en la raíz del proyecto destino. Describe su stack, flujo de 4 fases, y decisiones clave. Máxima economía de tokens para que el agente del proyecto arranque sin contexto innecesario.

**Humano**: Aprueba contratos. Si skills fueron recomendadas, se instalan ahora: `npx skills add <source>` en raíz del repo.

### F3: Implementación
Aplicación (paralelo por tarea) → Testing de Contratos → loop hasta pasar → [VALIDACIÓN HUMANA]

**Aplicación**: `internal/apply/runner.go`. Goroutine pool implementa tareas contra C-I-O. Cada tarea = una función/método.

**Testing**: `internal/test/contracts.go`. given/when/then contra C-I-O. Si falla → `internal/test/report.go` genera diff con graphify. Humano revisa, corrige, y se re-ejecuta.

**Loop**: max_loops del governance en handoff.yaml (default 5).

### F4: Cierre
Archivo (Engram) + Automática (lint/build) + Revisión Final (opcional)

**Archivo**: `mem_save` a Engram con topic_key `zyro/{project}/archive-report`. Limpia topics temporales.

## Contrato handoff.yaml (input de Holdin Admin)
```yaml
version: "2.0"
source: { system: "holdin-admin" }
project: { name, language, repository }
validated_idea: { problem, success_criteria }
user_stories:
  - id: "US-001"  # Solo para trazabilidad. Agente nunca ve esto.
mvp: { strict_boundaries, excluded }
governance:
  mode: "always_approve"
  approval_points:
    - { phase: "F1", required: true, max_loops: 3 }
testing: { strategy: "contract_testing", tdd_mode: true, max_fix_loops: 5 }
limits: { max_parallel_agents: 5, phase_timeout: "10m" }
```

## Comandos CLI
- `zyrocli init [handoff.yaml]` — crea repo, indexa workspace
- `zyrocli run` — ejecuta F1→F2→F3→F4 con pausas en cada validación humana(esto lo hae internamente opencode)
- `zyrocli phase F2` — ejecuta solo F2 (Contextualización + Especificación)
- `zyrocli skill-advisor` — analiza proyecto y recomienda skillsw
- `zyrocli context-serve --libs "$(cat go.mod | grep -oP '^\s+\S+' | tr '\n' ' ')"` — arranca Context MCP server

## Patrones Go
- **Concurrencia**: goroutine pool con `chan Result`. `context.WithTimeout` por agente. `select` para fan-in de resultados y detección de timeouts parciales.
- **MCP Bridge**: `os/exec` para subprocess Python. Stdout parseado como JSON. Timeout por agente.
- **State Machine**: Scheduler interno con DAG de dependencias. Cada macrófase es un nodo.
- **Engram MCP**: `mem_save` y `mem_search` via MCP client para persistencia.

## Seguridad
- Skills: solo verified publishers y audit passes (skills.sh/audits)
- Context: local-first, sin telemetría, Apache 2.0
- GitMCP: no almacena queries, self-hostable, solo repos públicos
- No hardcodear credenciales. Usar env vars o secrets manager.
- Symlinks para compatibilidad (cuando exista repo):
  ```bash
  ln -s AGENT.md CLAUDE.md
  ln -s AGENT.md .cursorrules
  ```
