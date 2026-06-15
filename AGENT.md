# ZyroCLI — Orquestador Go para desarrollo asistido por IA

## ⚠️ Reglas del agente (LEER PRIMERO)
- **NO hacer commit, push ni crear PRs** — salvo orden explícita del humano
- **NO decidir por tu cuenta** — si algo no está especificado en el handoff.yaml o en las fases, PREGUNTAR al humano
- **Solo orquestar** — tu trabajo es coordinar subagentes y fases, NO implementar código directamente
- **Usar SDD** — tienes skills `zyro-sdd-apply`, `zyro-sdd-verify`, `zyro-sdd-explore`, `zyro-sdd-propose`, `zyro-sdd-spec`, `zyro-sdd-design`, `zyro-sdd-tasks`, `zyro-sdd-archive` para delegar. DELEGAR siempre que una tarea cruce los triggers (4+ archivos, multi-file write).
- **NO inflar contexto** — si una tarea requiere leer 4+ archivos o escribir en 2+ archivos no triviales, delegar a subagente. No hacerlo inline.
- **Idioma** — responder siempre en el mismo idioma del humano

## Stack
- Go 1.26+, module: `github.com/secko/zyrocli`
- HelixDB (graph-vector DB, localhost:6969, HTTP API + Go SDK)
- Python MCP tools (PydanticAI, httpx) via `uv run`
- Context (Neuledge) `npm i -g @neuledge/context` + GitMCP fallback

## Estructura del proyecto
```
zyrocli/
├── cmd/zyrocli/
│   ├── main.go              # Entry point (cobra CLI)
│   ├── init.go              # zyrocli init <handoff.yaml>
│   ├── run.go               # zyrocli run (F0→F1→F2→F3→F4)
│   ├── install.go           # zyrocli install (config global)
│   ├── profile.go           # zyrocli profile list/set/tui
│   ├── context.go           # zyrocli context <task>
│   ├── skilladvisor.go      # zyro skill-advisor <query>
│   └── absorb.go            # zyrocli absorb docs/
├── internal/
│   ├── db/helix/             # Go SDK client para HelixDB
│   ├── scheduler/            # State machine F0-F4 + approval gates
│   ├── handoff/              # Parseo de handoff.yaml v2.0
│   ├── scaffold/             # Generación de proyecto
│   ├── opencode/             # Skills .md embebidas + config writer
│   ├── taskcontext/          # Contexto de tarea desde HelixDB
│   └── skilladvisor/         # Scoring de skills (ScoreSkill)
├── mcp-tools/                # Python MCP tools
│   ├── runner.py             # FastMCP server entry point
│   ├── helix_client.py       # httpx wrapper HelixDB
│   ├── helix_write.py        # save_to_helix, link_to_project, find_project
│   ├── task_context.py       # task_context(id)
│   ├── search_code.py        # search_code(query)
│   ├── search_skills.py      # search_skills(query)
│   └── pyproject.toml        # uv project config
└── AGENT.md
```

## Flujo de 5 macro-fases (scheduler + approval gates)

### F0: Investigación
Subagentes: `zyro-phase-0-patterns`, `zyro-phase-0-libraries`, `zyro-skills-find`, `zyro-skills-audit`, `zyro-skills-apply`
Output: nodos Pattern, Library, Skill en HelixDB
Aprobación humana: skills a instalar

### F1: Especificación
Subagente: `zyro-sdd-spec`
Output: nodo Spec en HelixDB (architecture, modules, dependencies, testing_strategy)
Aprobación humana: spec aprobada

### F2: Diseño + Tareas
Subagentes: `zyro-sdd-design` → `zyro-sdd-tasks`
Output: nodo Design + nodos Task en HelixDB
Aprobación humana: diseño + tareas aprobadas

### F3: Implementación
Subagentes: `zyro-sdd-apply` ↔ `zyro-sdd-verify` (loop, máx 5 intentos)
Output: nodos CodeModule + Review en HelixDB
Aprobación humana: implementación aprobada

### F4: Cierre
Subagente: `zyro-sdd-archive`
Output: nodo Archive en HelixDB, Project.status = "archived"
Aprobación humana: proyecto cerrado

## Contrato handoff.yaml (input de Holdin Admin)
```yaml
version: "2.0"
source: { system: "holdin-admin" }
project: { name, language, repository }
validated_idea: { problem, success_criteria }
user_story:
  story: "..."
  acceptance: "..."
mvp: { scope, features }
governance:
  mode: "always_approve"
testing:
  strategy: "unit"
limits:
  max_loops: 5
  phase_timeout: "10m"
```

## Comandos CLI
- `zyrocli install` — configura ecosistema global (skills, agentes, MCP)
- `zyrocli init <handoff.yaml>` — crea proyecto + abre OpenCode
- `zyrocli run [--phase F0|F1|F2|F3|F4]` — ejecuta pipeline con approval gates
- `zyrocli profile list|set|tui` — gestión de modelos por agente
- `zyrocli context <task-id|project-name>` — contexto desde HelixDB
- `zyrocli skill-advisor <query>` — busca + valida + puntúa skills
- `zyrocli absorb <dir>` — ingesta docs/ a HelixDB

## Patrones Go
- **Concurrencia**: goroutine pool con `chan Result`. `context.WithTimeout` por agente.
- **State Machine**: Scheduler con DAG de fases + approval gates obligatorios.
- **HelixDB**: writes controlados via Go SDK, reads via MCP tools.
- **Skills**: embebidas en binario, 14 skills, reglas first.

## Seguridad
- Skills: solo verified publishers y audit passes (skills.sh/audits)
- Permisos por agente: bash/write/edit deny donde no se necesita
- Validación: NVIDIA SkillSpector (`skillspector scan`)
- No hardcodear credenciales. Usar env vars o secrets manager.
- Symlinks para compatibilidad:
  ```bash
  ln -s AGENT.md CLAUDE.md
  ln -s AGENT.md .cursorrules
  ```
