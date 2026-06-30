---
description: >-
  ZyroCLI Orchestrator — Pure coordinator. Receives user requests, decides phase,
  delegates to specialized sub-agents, waits for results, asks for approval.
  NEVER reads files, NEVER investigates, NEVER implements code.
mode: primary
model: opencode-go/deepseek-v4-flash
temperature: 0.1
tools:
  read: false
  write: false
  edit: false
  bash: true
  glob: false
  grep: false
  task: true
  skill: false
permission:
  read: deny
  edit: deny
  bash:
    "zyrocli *": allow
    "*": ask
  task: allow
  skill: deny
  external_directory: deny
---

# ZyroCLI Orchestrator

> ⚠️ Este archivo es la **definición técnica del agente** (frontmatter).
> Las **instrucciones de comportamiento** están en `AGENT.md` (raíz del proyecto)
> y en `internal/opencode/skills/zyro-orchestrator/SKILL.md`.

## Permisos

El frontmatter define los permisos del agente orquestador:

- **read/write/edit**: denegados — el orquestador solo coordina
- **bash**: `zyrocli *` permitido sin preguntar, todo lo demás requiere aprobación
- **task**: permitido — puede delegar a subagentes
- **skill**: denegado — no carga skills directamente

## Pipeline

Ver `AGENT.md` en la raíz del proyecto para la descripción completa del pipeline SDD v2.
