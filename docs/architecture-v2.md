# ZyroAgentCLI — Arquitectura Decisional v2
> Junio 2026 · Síntesis post-interrogatorio

---

## Contradicción Principal: RESUELTA

| Antes (RFC inflado) | Ahora (decisión) |
|---------------------|------------------|
| ¿Quién lidera el pipeline? | **OpenCode lidera. ZyroCLI configura.** |
| gRPC para 4 scripts de 44-105 líneas | os/exec. Sin cambios. |
| Grafo estático de 1770 nodos inútil | HelixDB vivo, con propósito claro |
| ~1700 líneas de código que no existen | MVP en 3 fases, sin over-engineering |
| Context7 bridge | Se depreca. Usar Context + GitMCP en su lugar. |
| HelixDB MCP como runtime | No existe como producto. En su lugar: **modelo híbrido** HTTP directo + MCP tools propias + ZyroCLI. |

**ZyroCLI = companion/configurador, igual que gentle-ai.**
**OpenCode = agente de código que lidera el flujo de desarrollo.**

---

## El Propósito Real de HelixDB

No es "reemplazar el grafo estático". Es **inyección eficiente de contexto**.

En vez de que cada subagente revise todo el código:

```
Orquestador → consulta HelixDB → "Este nodo es el módulo auth (JWT + refresh).
              Este nodo es el schema User (id, email, role). Este nodo es el
              patrón Repository usado en este proyecto."
              ↓
Subagente recibe solo esos nodos → trabaja sobre lo específico
              ↓
Sin lectura de codebase completa. Sin tokens desperdiciados.
```

El grafo guarda **summaries de nodos**, no código completo. El agente pide el detalle
solo si lo necesita.

---

## Schema HelixDB — Jerárquico + Edges Cruzados

```
Developer
  properties: name, default_tech_stack, culture, playbook_ref
  ↓ HAS_PROJECT
  Project
    properties: project_id, name, description, status, current_phase, repo_path
    ↓ HAS_DOC        → Document   (README, spec, handoff, ADR, decisión)
    ↓ HAS_PATTERN    → Pattern    (patrones encontrados en Phase 0)
    ↓ USES_LIB       → Library    (librerías validadas en Phase 0)
    ↓ REQUIRES_SKILL → Skill      (nodo COMPARTIDO entre proyectos — sin project_id)
    ↓ HAS_CODENODE   → CodeNode   (summary de módulo, NO código completo)
                                 properties: path, summary, hash, language
                                 upsert por (project_id, path)
    ↓ HAS_TASK       → Task       (trabajo actual/pasado)

Skill (pool compartido del Developer — SIN project_id)
  properties: name, type (BE|FE|DevOps), source_url, validated_at, version
  ↑ REQUIRES_SKILL ← múltiples Projects

Task
  properties: description, phase, status, created_at
  ↓ REFERENCES → CodeNode   (edges con REFERENCES a CodeNode)
  ↓ REQUIRES   → Skill
```

### Por qué jerárquico + edges cruzados y no plano

Si TypeScript + Tailwind aparece en 3 proyectos, en grafo plano lo duplicás 3 veces.
Con edges cruzados es **un nodo Skill** compartido al que apuntan los 3 proyectos.
Cuando un skill se actualiza (nueva versión, deprecación), se actualiza en un solo lugar.
Esto es lo que lo hace "escalable a nivel empresarial".

---

## Modelo de Acceso Híbrido a HelixDB

HelixDB (Rust, sub-milisegundo) es el eje central. Se accede de tres formas,
cada una para su propósito:

### Capa 1: HTTP directo (agente autónomo)
- El agente (OpenCode/Gentle AI) consulta HelixDB vía POST /v1/query
- Para reads exploratorios rápidos: "¿qué Skills usa este proyecto?"
- Aprovecha velocidad nativa de Rust (~1ms, 10-1000x vs alternativas)
- No requiere skills externas ni middleware
- Solo reads — writes van por ZyroCLI

### Capa 2: MCP Tools (contexto trazable)
- MCP server propio expone tools: task_context, search_code, search_skills
- Internamente llaman a zyrocli context
- Garantizan: trazabilidad, consistencia en formato, optimización de tokens
- El subagente recibe siempre el mismo formato predecible
- Se depreca zyrocli context como CLI, pero la lógica vive en las MCP tools

### Capa 3: ZyroCLI (writes controlados)
- Operaciones que MODIFICAN datos: task create, db init, absorb, sync
- Van por Go SDK de HelixDB
- Controladas, trazables, sin riesgo de writes accidentales del agente

### Diagrama
```
OpenCode/Gentle AI
  │
  ├── POST /v1/query ──→ HelixDB (reads exploratorios, ~1ms, Rust)
  │
  ├── MCP tool: task_context ──→ zyrocli context ──→ HelixDB (trazable)
  │
  └── CLI: ZyroCLI ──→ Go SDK ──→ HelixDB (writes controlados)
```

---

## Flujo Completo

```
┌─────────────────────────────────────────────────────┐
│  HoldingAdmin (fuera del proyecto)                  │
│  Valida idea → genera handoff → cae en .docs/       │
└─────────────────────┬───────────────────────────────┘
                      ↓
           zyrocli init [nombre]
           ├── Crea estructura de carpetas
           ├── Crea .zyro/ (config local del proyecto)
           └── Crea nodos Developer + Project en HelixDB

                      ↓
           Dev abre OpenCode en la carpeta
           (opcional: /zyro-model para asignar modelos por fase)

                      ↓
           Dev escribe en el chat:
           "empecemos con [proyecto]" → OpenCode presenta Phase 0:

┌─────────────────────────────────────────────────────┐
│  PHASE 0 — Investigación (validación humana previa) │
│                                                     │
│  OpenCode: "Voy a consultar HelixDB directamente     │
│  (vía POST /v1/query) para investigación rápida,    │
│  más 3 subagentes en paralelo. ¿Arrancamos?"        │
│  → Dev dice sí                                       │
│                                                     │
│  HelixDB directo (HTTP):                            │
│    → Consulta Skills del Developer pool             │
│    → Consulta CodeNodes existentes del proyecto     │
│    → Consulta Tasks activas para contexto           │
│                                                     │
│  Subagente A: WebResearcher                         │
│    → Busca patrones similares al proyecto           │
│    → Genera Pattern nodes en HelixDB                │
│                                                     │
│  Subagente B: LibraryValidator                      │
│    → git MCP + Cortes → valida librerías vigentes   │
│    → Genera Library nodes en HelixDB                │
│                                                     │
│  Subagente C: SkillDetector                         │
│    → Detecta skills BE + FE necesarias              │
│    → Busca o crea Skill nodes (shared pool)         │
│                                                     │
│  + absorbe .docs/ si existe → Doc nodes             │
│                                                     │
│  Output al dev:                                     │
│    1. Patrones recomendados (con justificación)     │
│    2. Librerías validadas (pocas, precisas)         │
│    3. Skills detectadas (BE + FE)                   │
└─────────────────────────────────────────────────────┘
                      ↓
           Dev valida output → Phase 1-N (SDD normal)
           OpenCode inyecta nodos HelixDB en cada delegación
```

---

## /zyro-model — Implementación

**Nivel**: global (`~/.config/opencode/commands/zyro-model.md`)

```yaml
---
description: Seleccionar modelos por fase SDD
subtask: true
---
!`zyrocli profile tui`
```

`zyrocli profile tui` es un selector interactivo en terminal (igual al TUI de gentle-ai):

```
┌── /zyro-model ─────────────────────────────────────────┐
│  Asignar modelos por fase SDD                          │
│                                                        │
│  sdd-explore  →  [ deepseek/deepseek-v4-flash-free  ] │
│  sdd-onboard  →  [ mimo/mimo-vl-7b-rl-free          ] │
│  sdd-spec     →  [ mimo/mimo-vl-7b-rl-free          ] │
│  sdd-design   →  [ nvidia/nemotron-3-super-free      ] │
│  sdd-propose  →  [ nvidia/nemotron-3-super-free      ] │
│  sdd-apply    →  [ deepseek/deepseek-v4-flash-free   ] │
│  sdd-verify   →  [ deepseek/deepseek-v4-flash-free   ] │
│                                                        │
│  [Enter] confirmar  [Tab] ciclar modelo  [q] cancelar  │
└────────────────────────────────────────────────────────┘
```

Escribe resultado a `~/.config/opencode/profiles/[project-name].json`

---

## ZyroCLI — Comandos MVP

| Comando | Función | Estado |
|---------|---------|--------|
| `zyrocli init [name]` | Scaffold + .zyro/config.yaml | 🔨 A construir |
| `zyrocli sync` | Sync profiles → OpenCode | 🔨 A construir |
| `zyrocli profile tui` | Selector interactivo modelos 2-pasos | ✅ Implementado |
| `zyrocli db init` | Inicializar schema HelixDB | ✅ Implementado |
| `zyrocli db status` | Verificar estado HelixDB | ✅ Implementado |
| `zyrocli absorb` | Ingesta .docs/ → Doc nodes | ✅ Implementado |
| `zyrocli task create/link/list` | Gestión de tareas + CodeNode graph | ✅ Implementado |
| `zyrocli context [task]` | Contexto para subagentes (deprecado → MCP tool) | ✅ Implementado (en deprecación) |
| MCP server (tools) | task_context, search_code, search_skills | 🔨 A construir |
| Scheduler F1 | Ya funciona | ✅ |
| `/zyro-model` | Slash command global | ✅ Implementado |

---

## Plan de Ejecución MVP

### ✅ Fase 1 — Companion Funcional (COMPLETADA)
- `/zyro-model` slash command
- `zyrocli profile tui` (2-pasos provider→modelo)
- `internal/opencode/` package (lista curada providers + opencode.json reader/writer)
- 44 tests

### ✅ Fase 2 — HelixDB Integration (COMPLETADA)
- `internal/db/helix/` (client, schema, nodes, edges, search, errors)
- `tenant_id` → `project_id` refactor
- `zyrocli db init/status/reset`
- `zyrocli absorb`
- 83 tests

### ✅ Fase 3 — Multi-Project (COMPLETADA)
- Skills cross-project (sin project_id)
- CodeNode summaries via AST Go (`internal/codeparse/`)
- Task → CodeNode graph via git diff (`internal/git/`)
- `zyrocli context [task]` (3 formatos: text/json/prompt)
- 122 tests

### 🟡 Fase 4 — HelixDB como Eje Central (PRÓXIMA)
- MCP server propio (expone tools: task_context, search_code, search_skills)
- Helix Skills instaladas (helix-query-*)
- Context + GitMCP reemplazan Context7 bridge
- Modelo híbrido: HTTP directo + MCP tools + ZyroCLI
- Deprecar `zyrocli context` (mantener lógica en MCP tools)

---

## Lo que NO entra en ninguna fase

| Descartado | Motivo |
|------------|--------|
| gRPC + proto files | os/exec funciona. No hay estado que preservar. |
| Python gRPC server persistente | PydanticAI nativo cuando llegue el momento |
| HelixDB MCP server oficial | No existe. En su lugar: **MCP server propio + modelo híbrido**. |
| Context7 bridge | Reemplazado por Context + GitMCP. Más control, menos limitaciones. |
| Reemplazo de openspec/ | Funciona. HelixDB lo complementa, no lo reemplaza. |
| ConnectRPC ahora | Evaluar recién en Fase 3 si el load lo justifica |

---

## Relación con Arquitectura Actual

```
Lo que YA funciona (no tocar):
  ✅ Scheduler F1 (parseo handoff + approval gates)
  ✅ Handoff parser
  ✅ Scaffold
  ✅ Doc sync (~500 líneas)
  ✅ Context bridge (JSON-RPC over stdio, 266 líneas)
  ✅ Contract testing (given/when/then, 112 líneas)
  ✅ Python scripts con os/exec (stdlib puro)

Lo que se construye en Fase 1:
  🔨 zyrocli profile tui
  🔨 zyrocli sync
  🔨 /zyro-model slash command

Lo que se construye en Fase 2:
  🔨 HelixDB schema + Go SDK client
  🔨 zyrocli absorb
  🔨 Phase 0 AGENTS.md definitions
  🔨 Context injection vía context bridge existente

Stubs que se completan:
  ⚠️  F2-F4 scheduler (completan con HelixDB en Fase 2)
---

## Próximos pasos inmediatos

1. **Crear `/zyro-model`** — `.md` en `~/.config/opencode/commands/`
2. **Crear `zyrocli profile tui`** — subcomando Go con bubbletea o similar
3. **Definir schema HelixDB en HQL** — antes de tocar Go SDK
4. **Validar Go SDK** — hacer una query real a HelixDB local (puerto 6969)
   para confirmar que el JSON AST dinámico es suficiente para el schema propuesto
