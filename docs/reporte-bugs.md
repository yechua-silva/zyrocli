# 🏛️ Auditoría Estática Completa — ZyroAgentCLI

**Fecha:** 2026-06-19
**Tipo:** Solo lectura de código, trazado de flujos, cero ejecución.
**Total:** 12 🔴 Críticos · 12 🟡 Medios · 6 🟢 Cosméticos

---

## 🔴 CRÍTICOS (12) — Rompen funcionalidad en runtime

| # | Área | Archivo:línea | Qué falla | Consecuencia en runtime |
|---|------|---------------|-----------|------------------------|
| **C1** | Memoria | `contradictions.go:109-111` | `deactivateFact` es STUB: `_ = fact`, `return nil`. TODO sin implementar. | Facts contradictorios NUNCA se desactivan en HelixDB. Filtro `is_active` no excluye nada. Contradicciones persisten para siempre. |
| **C2** | Memoria | `scheduler.go:140-141` | `_ = mc` — el contexto de memoria obtenido por `PrePhase` se descarta con blank identifier. | `RunPhase` single-phase arranca SIN contexto de memoria causal. El TODO en línea 140 confirma que nunca se implementó la inyección. |
| **C3** | Orquestación | `scheduler/config.go:70` | `TaskManager` creado con `NewTaskManager(5)` — SIN `apply.Runner`. | `DelegateStep` ejecuta el STUB en `executeTask`: marca `Success: true` inmediato. El pipeline `zyrocli run` nunca ejecuta nada real. |
| **C4** | Orquestación | `task_manager.go:178-183` | `Execute` function es STUB: `return "Tarea despachada...", nil`. | TODO explícito: "for now, marks the task as dispatched". El runner existe pero la tarea no hace nada. |
| **C5** | Orquestación | `config.go:70` vs `mcp_server.go:83` | DOS `TaskManager` separados: uno sin runner (scheduler), otro con runner (MCP). NUNCA se cruzan. | El orquestador Boomerang tiene un TaskManager stub. El MCP server tiene otro con runner real. No conversan. |
| **C6** | Orquestación | `mcp_server.go:423-473` | `wait_phase` sin autenticación. Protocolo JSON-RPC sobre stdio sin auth. | Cualquier skill, subagente o LLM con acceso al proceso puede sincronizar fases sin permiso. |
| **C7** | Conexiones nuevas | `absorb.go:191-197` | `CodeNode` creado con `file_path` en vez de `path`, sin `hash`, sin `project_id`. Schema incorrecto. | Índices no funcionan, `findCodeNodeByPath` nunca encuentra nada, duplicados infinitos en re-ejecuciones, nodos huérfanos sin proyecto. |
| **C8** | Config e Install | `install.go:345-362` | `runner.py` embebido es un **helix MCP server** (FastMCP). El `runner.py` raíz es un **Zyro Agent runner** (PydanticAI). Son programas DISTINTOS. | `zyrocli install` deploya el runner INCORRECTO. El MCP server embebido no coincide con el código fuente actual. |
| **C9** | Config e Install | `mcptools/` vs `mcp-tools/` | Drift severo: 6 archivos `.py` faltan en embebidos (`embedding_harness.py`, `models.py`, `capabilities.py`, `boundari_wrapper.py`, `approval.py`, `agent.py`). `pyproject.toml` es v0.1.0 vs v0.2.0. | Snapshot congelada y obsoleta. El binario deploya código viejo. |
| **C10** | Skills | `.zyro/handoffs/` no existe | Skills dicen "generá handoff" pero NO HAY código Go que lo genere, valide o verifique. El directorio no existe. | Handoffs son humo documental. Si el LLM falla al generar, la siguiente fase arranca sin contexto. Cero enforcement programático. |
| **C11** | Comandos CLI | `run.go:75-81` | PRE-F0 NO está en el pipeline. `runners` arranca directo en F0. | `zyrocli run` salta la alineación de dominio. No hay fase PRE-F0 ejecutable. |
| **C12** | Comandos CLI | `skills_embed.go:29` | Skill `zyro-pre-f0` embebido pero **nunca referenciado** por pipeline ni scheduler. | Asset huérfano: 1 skill + 1 agente que existen pero nadie los llama. |

---

## 🟡 MEDIOS (12) — Comportamiento incorrecto o degradado

| # | Área | Archivo:línea | Qué falla | Consecuencia |
|---|------|---------------|-----------|-------------|
| **M1** | Memoria | `decay.go:26` → `recall.go:196-200` | `GetFactByID` usa `GetCausalChain` con traversal completo para leer un solo fact. | Operación O(n) sobre el grafo para una lectura puntual. Degradación en proyectos grandes. |
| **M2** | Memoria | `scheduler/config.go:67` | `embeddingSvc = nil` en `NewHelixEngramStore`. | Solo BM25. Sin búsqueda vectorial, sin detección de contradicciones. Degradación silenciosa. |
| **M3** | Memoria | `memory_hook.go:66-68` | `factExtractorPath = ""` — nunca usa extractor Python. Solo keyword fallback español (9 keywords). | Extracción de hechos limitada a español. Sin soporte para otros idiomas ni NLP avanzado. |
| **M4** | Memoria | `embedding.go:54` | `sync.Map` sin límite de tamaño ni evicción. `CacheSize=1000` definido pero nunca usado. | Memoria del proceso crece sin bound en producción. |
| **M5** | MCP Layer | `install.go:346-349` | No verifica que `uv` esté instalado antes de registrar `helix-integration`. | MCP tool no arranca si falta `uv`. Error silencioso en logs de OpenCode. |
| **M6** | MCP Layer | `install.go:354-357` | Entry `context` escrito ANTES de instalar `@neuledge/context`. | MCP server registrado pero quebrado si `npm install -g` falla. |
| **M7** | Conexiones nuevas | `task_manager.go:186-206` | `len(results) == 0` cae al fallback stub en vez de marcar error. | Tareas no-ejecutadas (workers skipped por fail-fast) aparecen como completadas exitosamente. |
| **M8** | Conexiones nuevas | `absorb.go:169-186` | `project_id` no se inyecta si `HELIX_PROJECT_ID` no está configurada. | Nodos fuera de scope del proyecto. No recuperables por queries scoped. |
| **M9** | Config e Install | `.gitignore` | `.codex/` no está en `.gitignore`. | Config de Codex CLI puede trackearse accidentalmente. |
| **M10** | Comandos CLI | `internal/context/bridge.go:1,5` | `package context` + `import "context"`. Compila pero es trampa de legibilidad. | Desarrolladores dudan de la validez. Convención Go exige alias. |
| **M11** | Comandos CLI | `absorb.go:70-88, 169-186` | Lógica de conexión HelixDB duplicada en 2 lugares. | Si cambian los defaults/env vars, hay que actualizar ambos. |
| **M12** | Skills | `scheduler.go:39-40` | BUG #5: modo Boomerang salta `PrePhase` porque `Boomerang.RunPhase` hace su propio `MemoryStep`. | Documentado pero sin fix. Doble recall evitado, pero `PrePhase` completamente ignorado en modo Boomerang. |

---

## 🟢 COSMÉTICOS (6) — No afectan runtime

| # | Área | Archivo:línea | Hallazgo |
|---|------|---------------|----------|
| **G1** | MCP Layer | `mcp_server.go:367` | `saveTaskToHelix` es fire-and-forget goroutine. Si HelixDB falla, solo log. |
| **G2** | MCP Layer | `task_manager.go:136-138` | IDs de tarea `fmt.Sprintf("%s-%s-%d", ...)` — counter reinicia al reiniciar proceso. Solo importa si se persisten IDs. |
| **G3** | Config e Install | `skills_embed.go:37-42` | `deprecatedSkillDirs` sigue existiendo. 12 `os.Stat` innecesarios por `zyrocli install`. Inocuo. |
| **G4** | Skills | `zyro-sdd-propose/SKILL.md` | Cuerpo placeholder de 8 líneas sin estructura estándar. Legacy no migrado a SDD v2. |
| **G5** | Comandos CLI | `profile_tui.go:217` | `borderStyle.Render()` en cada `View()` — Lip Gloss recalcula borde cada frame. Posible flicker cosmético. |
| **G6** | Memoria | `search.go:207-213` | ✅ **OK:** ValueMap lee los 6 campos. `applyFilters` funciona. `ReinforceSalience` tiene `UpdateNode` real. No hay `_ = props`. No es bug. |

---

## 📊 Resumen por Área

| Área | 🔴 Críticos | 🟡 Medios | 🟢 Cosméticos | Diagnóstico |
|------|:-----------:|:---------:|:-------------:|-------------|
| **1. Memoria Causal** | 2 | 4 | 1 | `deactivateFact` stub + `_ = mc` son los 2 agujeros. El resto funcional pero degradado (nil embedding, cache sin límite). |
| **2. Orquestación** | 4 | 0 | 0 | **La peor área.** DelegateStep es no-op, dos TaskManager sin conexión, wait_phase sin auth. |
| **3. MCP Layer** | 1 | 2 | 2 | Execute stub es el crítico. Dependencias externas (uv, context) no verificadas. |
| **4. Conexiones nuevas** | 1 | 2 | 0 | CodeNode en absorb --code tiene schema incorrecto. Empty results de runner no manejados. |
| **5. Config e Install** | 2 | 1 | 1 | runner.py embebido es otro programa. Drift severo mcptools/ vs mcp-tools/. |
| **6. Skills** | 1 | 1 | 1 | Handoffs son humo. Lo demás en buen estado. |
| **7. Comandos CLI** | 2 | 2 | 2 | PRE-F0 ausente del pipeline. Bridge Neuledge es dead code (635 líneas). |

---

## ⚡ Top 5 prioritarios

```
1. C3+C4  → DelegateStep es no-op. Sin esto, el pipeline "executa" pero no hace nada.
2. C8+C9  → runner.py embebido está mal. zyrocli install deploya código incorrecto.
3. C7     → absorb --code crea nodos HelixDB con schema equivocado → duplicados infinitos.
4. C1     → deactivateFact no persiste → contradicciones nunca se resuelven.
5. C10    → Handoffs no existen → el pipeline no tiene memoria entre fases.
```

---

## Detalle por Área

### Área 1 — Memoria Causal (Engram + HelixDB)

**Archivos:** `internal/db/helix/search.go`, `internal/memory/store.go`, `internal/memory/decay.go`, `internal/memory/contradictions.go`, `internal/memory/recall.go`, `internal/boomerang/engram.go`, `internal/memory/memory_hook.go`, `internal/db/helix/embedding.go`, `internal/scheduler/scheduler.go`

| Ítem | Estado | Detalle |
|------|--------|---------|
| `enrichWithNodeProperties` ValueMap | ✅ OK | 12 campos: `$id, $label, content, salience, confidence, phase, project_id, is_active, is_stale, access_count, decay_rate, created_at, last_accessed_at` |
| `applyFilters` con salience | ✅ OK | `r.Fact.Salience < opts.MinSalience` funciona. Salience propagada correctamente. |
| `ReinforceSalience` | ✅ OK | `UpdateNode` real con `salience`, `access_count`, `last_accessed_at`. No hay `_ = props`. |
| `DecayAndRefresh` | ✅ OK | No es stub. Carga facts, aplica Ebbinghaus, persiste. |
| `deactivateFact` | 🔴 **CRÍTICO** | Stub: `_ = fact`, `return nil`. TODO sin implementar. |
| `NewDefaultConfig` embeddingSvc | 🟡 MEDIO | `nil` pasado explícitamente. Guard en cada método evita panic pero degrada a BM25. |
| `PostPhase` extractor Python | 🟡 MEDIO | `factExtractorPath = ""` → nunca usa Python. Solo 9 keywords español. |
| Embedding cache `sync.Map` | 🟡 MEDIO | Sin límite. `CacheSize` definido pero inerte. |
| Double recall guard | ✅ OK | Guard en scheduler.go:42 evita doble recall. |
| `_ = mc` en RunPhase | 🔴 **CRÍTICO** | Contexto de memoria descartado. |

---

### Área 2 — Orquestación (Boomerang + Scheduler)

**Archivos:** `internal/boomerang/delegate.go`, `internal/boomerang/think.go`, `internal/boomerang/orchestrator.go`, `internal/boomerang/skip.go`, `internal/scheduler/scheduler.go`, `internal/scheduler/approval.go`, `cmd/zyrocli/mcp_server.go`, `cmd/zyrocli/run.go`

| Ítem | Estado | Detalle |
|------|--------|---------|
| DelegateStep ejecución real | 🔴 **CRÍTICO** | `executeTask` es stub. `Success: true` inmediato. TaskManager sin runner en scheduler. |
| ThinkStep DAG → DelegateStep | ✅ OK | `dag` se pasa correctamente entre steps en `runPhaseV2`. |
| Cadena run → scheduler → boomerang | ✅ OK | Lineal: `run.go` → `scheduler.Run()` → `boomerang.RunPhase()`. |
| Dos TaskManager separados | 🔴 **CRÍTICO** | scheduler/config.go usa `NewTaskManager(5)` sin runner. mcp_server.go usa `NewTaskManagerWithRunner(5, runner)`. No conversan. |
| Approval gates | ✅ OK | `PromptApproval` espera input humano. Sin auto-approval. |
| `wait_phase` sin auth | 🔴 **CRÍTICO** | JSON-RPC sobre stdio. Cualquier skill/LLM puede llamarlo. |
| Skip matrix | ✅ OK | `DefaultPhaseMatrix()` implementada en skip.go. 6 steps por fase, F4 omite Think y Quality. |

---

### Área 3 — MCP Layer

**Archivos:** `cmd/zyrocli/mcp_server.go`, `cmd/zyrocli/install.go`

| Ítem | Estado | Detalle |
|------|--------|---------|
| `saveTaskToHelix` | ✅ OK | Usa SDK `helixClient.CreateNode`. `setup.GetHelixDBURL()`. Sin HTTP raw. |
| dispatch_task → apply.Runner | 🔴 **CRÍTICO** | Execute function es stub. No lanza subagente real. |
| check_task_status | ✅ OK | Lee del mismo `taskManager` global. Thread-safe (copia con RLock). |
| helix-integration sin verificación uv | 🟡 MEDIO | `install.go` no verifica `uv` antes de registrar. |
| context entry antes de install | 🟡 MEDIO | Config escrita antes de `npm install -g @neuledge/context`. |
| saveTaskToHelix fire-and-forget | 🟢 COSMÉTICO | Goroutine sin backpressure. |
| IDs no únicos tras restart | 🟢 COSMÉTICO | Counter reinicia. Solo importa si se persisten. |

---

### Área 4 — Conexiones nuevas (esta sesión)

**Archivos:** `internal/boomerang/task_manager.go`, `internal/context/bridge_pool.go`, `cmd/zyrocli/absorb.go`

| Ítem | Estado | Detalle |
|------|--------|---------|
| executeTask branch runner != nil | ✅ OK | Sí tiene branch y fallback. |
| Empty results runner.Run() | 🟡 MEDIO | Cae al fallback stub en vez de error. |
| SharedBridge sync.Once | ✅ OK | Lazy initialization correcta. |
| bridge Start() si no hay binary | ✅ OK | `exec.CommandContext` falla inmediato. No cuelga. |
| absorb --code flag | ✅ OK | `cmd.Flags()` local. No rompe default. |
| CodeNode schema | 🔴 **CRÍTICO** | `file_path` vs `path`, sin `hash`, sin `project_id`. |
| project_id no inyectado | 🟡 MEDIO | Depende de env var. |

---

### Área 5 — Config e Install

**Archivos:** `cmd/zyrocli/install.go`, `internal/opencode/skills_embed.go`, `.gitignore`, `.config/opencode/opencode.json`, `internal/opencode/mcptools/`

| Ítem | Estado | Detalle |
|------|--------|---------|
| MCP servers en buildInstallConfig | ✅ OK | `helix-integration`, `zyro-task-board`, `context`, `gitmcp`. |
| git MCP name | ✅ OK | `gitmcp`, consistente. |
| runner.py embebido vs raíz | 🔴 **CRÍTICO** | Son programas distintos. El embebido es helix MCP server; el raíz es Zyro Agent runner. |
| deprecatedSkillDirs | 🟢 COSMÉTICO | 12 Stat innecesarios. Inocuo. |
| .codex/ en .gitignore | 🟡 MEDIO | No está. |
| .helix/ en .gitignore | ✅ OK | Sí está. |
| opencode.json local | ✅ OK | zyro-orchestrator primary, 16 agentes, gentle-orchestrator eliminado. |
| Archivos embebidos mcptools/ | 🔴 **CRÍTICO** | Faltan 6 archivos .py. Snapshot congelada vs raíz. |

---

### Área 6 — Skills y Agentes

**Archivos:** `internal/opencode/skills/*/SKILL.md`, `.zyro/handoffs/`

| Ítem | Estado | Detalle |
|------|--------|---------|
| zyro-sdd-explore 3 pasos | ✅ OK | READ DOCS → INTERVIEW → OUTPUT completos. |
| zyro-sdd-spec template PRD | ✅ OK | 10 secciones: Problema, Solución, Stories, Acceptance, Deep Modules, etc. |
| zyro-pre-f0 sub-fases | ✅ OK | 4 sub-fases: grill-me, domain-model, triage (opt), improve-arch (opt). |
| Frontmatter 16 skills | ✅ OK | Todos con `name:` y `description:` válidos. |
| zyro-sdd-propose placeholder | 🟢 COSMÉTICO | 8 líneas, sin estructura estándar. |
| gentle-orchestrator en skills | ✅ OK | Solo en `docs/research/`, no en skills activos. |
| .zyro/handoffs/ existe | 🔴 **CRÍTICO** | **No existe.** Skills dicen generarlo pero no hay código Go que lo haga. |

---

### Área 7 — Comandos CLI

**Archivos:** `cmd/zyrocli/context.go`, `cmd/zyrocli/absorb.go`, `cmd/zyrocli/run.go`, `cmd/zyrocli/profile_tui.go`, `internal/context/bridge.go`

| Ítem | Estado | Detalle |
|------|--------|---------|
| context CLI vs bridge | ✅ OK | Ortogonales. Uno lee HelixDB, otro es bridge Neuledge. Sin colisión. |
| Bridge Neuledge usado | 🔴 **CRÍTICO** | **Dead code.** 635+ líneas nunca importadas. |
| absorb --code no rompe default | ✅ OK | Early return cuando `--code`. Flujo markdown intacto. |
| PRE-F0 en pipeline | 🔴 **CRÍTICO** | No está. `runners` arranca en F0. |
| zyro-pre-f0 huérfano | 🔴 **CRÍTICO** | Embebido pero no referenciado. |
| profile_tui logo | ✅ OK | No hay logo corrupto. El logo vive en `internal/tui/`, no en profile_tui. |
| Reset estado entre pasos | ✅ OK | `providerIdx` y `modelIdx` resetean. |
| package context + import context | 🟡 MEDIO | Confuso pero compila. |
| Conexión HelixDB duplicada | 🟡 MEDIO | 2 copias de la misma lógica en absorb.go. |

---

*Auditoría generada el 2026-06-19. 30 hallazgos totales.*
