# ZyroCLI — Guía para Developers

> Este archivo es **AGENT.md** — contiene las reglas para trabajar EN ZyroCLI.
> El comportamiento del orquestador en runtime está en `internal/opencode/skills/zyro-orchestrator/SKILL.md`.

---

## ⚠️ Reglas para developers

| # | Regla | Severidad |
|---|-------|-----------|
| R1 | **Trabajar SIEMPRE en rama `dev`** — nunca en `main` | 🔴 Bloqueante |
| R2 | **NO hacer push ni crear PRs** — solo commits locales en `dev` | 🔴 Bloqueante |
| R3 | **NO decidir por tu cuenta** — si algo no está en las fases, PREGUNTAR al humano | 🔴 Bloqueante |
| R4 | **NO implementar código directamente** — coordinar subagentes, delegar | 🔴 Bloqueante |
| R5 | **Delegar con SDD** — usar skills `zyro-sdd-*` cuando la tarea cruce triggers | 🟡 Alta |
| R6 | **NO inflar contexto** — si requiere leer 4+ archivos, delegar a subagente | 🟡 Alta |
| R7 | **Responder siempre en el mismo idioma del humano** | 🟢 Media |

---

## Stack

- **Go 1.26+**, module: `github.com/secko/zyrocli`
- **HelixDB** (graph-vector DB, localhost:6969, HTTP API + Go SDK)
- **Python MCP tools** (PydanticAI, httpx) via `uv run`
- **Context** (Neuledge): `npm i -g @neuledge/context`
- **Ollama** (GPU vía Vulkan): `nomic-embed-text` (768d) + `phi4-mini:3.8b`

---

## Comandos útiles

| Comando | Para qué |
|---------|----------|
| `make build` | Compila `zyrocli` con versión y commit |
| `go build ./...` | Compila todo sin generar binary |
| `go vet ./...` | Análisis estático |
| `go test ./...` | Tests |
| `./zyrocli doctor` | Diagnóstico del entorno |
| `./zyrocli install` | Instala el ecosistema global |

---

## Pipeline del orquestador (referencia)

El pipeline SDD v2 completo está documentado en:
`internal/opencode/skills/zyro-orchestrator/SKILL.md`

Resumen: `PRE-F0 → F0 → F1 → F2 → F3 → F4` con approval gates humanos.

---

## Skills

16 skills embebidas en `internal/opencode/skills/`:

| Skill | Fase |
|-------|------|
| `zyro-orchestrator` | — (orquestador) |
| `zyro-pre-f0` | PRE-F0 |
| `zyro-phase-0-patterns` | F0 |
| `zyro-phase-0-libraries` | F0 |
| `zyro-skills-find` | F0 |
| `zyro-skills-audit` | F0 |
| `zyro-skills-apply` | F0 |
| `zyro-sdd-spec` | F1 |
| `zyro-sdd-design` | F2 |
| `zyro-sdd-tasks` | F2 |
| `zyro-sdd-apply` | F3 |
| `zyro-sdd-verify` | F3 |
| `zyro-sdd-explore` | — (exploración) |
| `zyro-sdd-propose` | — (propuestas) |
| `zyro-sdd-archive` | F4 |
| `to-issues` | — (GitHub Issues) |

---

## Seguridad

- Skills solo de verified publishers
- NO hardcodear credenciales — usar env vars
- Para compatibilidad con otros agentes:
  ```bash
  ln -s AGENT.md CLAUDE.md
  ln -s AGENT.md .cursorrules
  ```
