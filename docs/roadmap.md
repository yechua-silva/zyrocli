# ZyroAgentCLI — Roadmap & Decision Log

> Junio 2026 · Seguimiento de progreso para sesiones futuras

---

## Estado General

| Dimensión | Estado |
|-----------|--------|
| Fase 1: Companion Funcional | ✅ COMPLETADA |
| Fase 2: HelixDB Integration | ✅ COMPLETADA |
| Fase 3: Multi-Project | ✅ COMPLETADA |
| Fase 4: HelixDB como Eje Central | 🟡 EN PLANIFICACIÓN |
| Tests totales | ~200 pasando |
| Paquetes Go | 17 |

---

## Fase 1 — Companion Funcional ✅

### Implementado
| Componente | Archivos | Tests |
|------------|----------|-------|
| /zyro-model slash command | ~/.config/opencode/commands/zyro-model.md | — |
| zyrocli profile tui (2-pasos) | cmd/zyrocli/profile.go, profile_tui.go, profile_tui_test.go | 30 |
| internal/opencode/ package | internal/opencode/models.go, opencode.go, opencode_test.go | 14 |

### Decisiones
- OpenCode lidera el pipeline, ZyroCLI configura
- Sin gRPC, sin protobuf, sin Python server persistente
- os/exec para scripts Python (suficiente)

---

## Fase 2 — HelixDB Integration ✅

### Implementado
| Componente | Archivos | Tests |
|------------|----------|-------|
| HelixDB client wrapper | internal/db/helix/client.go, schema.go, errors.go | 15 |
| Node CRUD | internal/db/helix/nodes.go | +18 |
| Edge operations | internal/db/helix/edges.go | — |
| Vector/text search | internal/db/helix/search.go | +11 |
| zyrocli db init/status/reset | cmd/zyrocli/db.go | — |
| zyrocli absorb | cmd/zyrocli/absorb.go | — |
| **Total helix** | **6 archivos** | **83 tests** |

### Decisiones
- HelixDB Go SDK v0.1.1, alias `helixsdk` (evita conflicto con package name)
- Tenant injection automática en writes
- Retry 3x en conexión
- Índices idempotentes (CreateIndexIfNotExists)
- API real: `Drop()` en vez de `Del()`, `SourceAnd()` en vez de `NWhere()`

---

## Fase 3 — Multi-Project ✅

### PR 1: Skills cross-project
- Skills sin project_id (globales al developer)
- FindSharedSkills, VectorSearchGlobal, UpsertSkill
- +24 tests (total 66 en helix)

### PR 2: CodeNode summaries
- Nuevo paquete internal/codeparse/ (go/ast)
- ParseFile, ParseDir, GenerateSummary (template-based, sin LLM)
- UpsertCodeNode por (project_id, path)
- +25 tests (total 91 en helix)

### PR 3: Task → CodeNode graph
- Nuevo paquete internal/git/ (git diff --name-status)
- task create/link/list
- LinkTaskToCodeNodes automático
- +20 tests

### PR 4: zyrocli context [task]
- Nuevo paquete internal/taskcontext/
- 3 formatos: text, json, prompt
- +11 tests

**Total Fase 3**: 122 tests en internal/db/helix/

### Decisiones
- `tenant_id` → `project_id` (rename completo, eliminado ErrTenantMismatch)
- Skills son globales (sin project_id), compartidos entre proyectos del mismo dev
- CodeNode summaries con template + AST primero, LLM después si hace falta
- git diff --name-status para detectar renames (no --name-only)
- internal/taskcontext/ (no internal/context/ — evita colisión con Context7 bridge)

---

## Fase 4 — HelixDB como Eje Central 🟡

### Pendiente
| Item | Prioridad | Dependencias |
|------|-----------|-------------|
| MCP server propio (tools: task_context, search_code, search_skills) | 🔴 Alta | — |
| Instalar Helix Skills (helix-query-*) | 🟡 Media | — |
| Reemplazar Context7 bridge por Context + GitMCP | 🟡 Media | — |
| Deprecar zyrocli context (lógica migra a MCP tools) | 🟢 Baja | MCP server |

### Decisiones tomadas (esta sesión)
- **Modelo híbrido**: Agente → HTTP directo a HelixDB (reads exploratorios, ~1ms) + MCP tools (contexto trazable) + ZyroCLI (writes controlados)
- **Context7 se depreca**: Reemplazar por Context + GitMCP
- **HelixDB MCP server oficial no existe**: El "MCP" de helix chef son skills de query-authoring, no un server runtime
- **Híbrido elegido sobre automático puro**: Por trazabilidad, consistencia en contexto de subagentes, y control de writes

### Abierto
- Definir tools específicas del MCP server
- Evaluar si OpenCode necesita config adicional para HTTP directo a HelixDB

---

## Stack Definitivo

| Capa | Tecnología | Puerto |
|------|-----------|--------|
| Orquestación | Go 1.26.3 + Cobra + bubbletea | CLI |
| Base de conocimiento | HelixDB (Rust, v3.0.5, Go SDK v0.1.1) | 6969 |
| Modelos | opencode.json → agents section | — |
| Docs | Context + GitMCP | — |
| Lenguajes | Go (stdlib), Python vía os/exec | — |

---

## Lo Descartado (con motivo)

| Descartado | Motivo |
|------------|--------|
| gRPC + protobuf | os/exec funciona, no hay estado que preservar |
| Python gRPC server | PydanticAI evaluation cuando llegue el momento |
| ConnectRPC | Evaluar si el load lo justifica (hoy no) |
| Reemplazar openspec/ | Funciona, HelixDB lo complementa |
| HelixDB MCP server oficial | No existe como producto |
| Context7 | Limita, reemplazar por Context + GitMCP |
| PostgreSQL, SQLite | HelixDB cubre grafos + vectores + docs |

---

## Notas para la Próxima Sesión

1. **MCP server** es la prioridad más alta de Fase 4
2. Verificar que los tests de integración con HelixDB real funcionen
3. Instalar Helix Skills en el entorno de desarrollo
4. Evaluar rendimiento de HTTP directo del agente vs MCP tools
