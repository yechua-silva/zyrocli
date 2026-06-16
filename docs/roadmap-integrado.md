# Roadmap Integrado: ZyroAgentCLI v2

> **Fecha:** 2026-06-15
> **Versión:** 2.0.0-plan
> **Integra:** 5 investigaciones + feedback CLI inteligente
> **Base:** docs/explorations/investigacion-01..05 + docs/feedback/cli-inteligente-setup.md + docs/architecture-v2.md + código actual

---

## 1. Filosofía del Sistema

### Principios Rectores

1. **OpenCode lidera el pipeline. ZyroCLI es el configurador/orquestador.**
   - OpenCode es el runtime de agentes. ZyroCLI prepara el terreno (setup, configuración, perfiles de modelo, seed de HelixDB) y luego deja que OpenCode ejecute las fases.
   - El scheduler Go orquesta las fases F0→F4 con approval gates, pero **no** reemplaza a OpenCode como runtime.

2. **Go es el orquestador, Python es el agente (pero no toca HelixDB).**
   - Go escribe/lee HelixDB directamente vía Go SDK oficial.
   - El agente Python (PydanticAI) recibe contexto plano (JSON stdin) desde Go, analiza con LLM, y retorna un PydanticModel validado.
   - El agente Python **NUNCA** tiene acceso directo a tools de escritura de HelixDB. Solo Go escribe.
   - Patrón: **Agent-as-Validator** → el agente opina, el orquestador ejecuta.

3. **Cada fase tiene políticas Boundari distintas.**
   - F0 (descubrimiento): solo lectura del codebase, sin escritura, sin ejecución.
   - F1 (investigación): lectura intensiva + web search con approval.
   - F2 (planificación): escritura de documentos/planos con approval.
   - F3 (implementación): escritura intensiva de código con approval condicional.
   - F4 (revisión): solo lectura otra vez, escritura solo bajo approval explícito.

4. **Memoria causal sobre HelixDB (NO usar Engram MCP Server).**
   - Zyro construye su propio sistema de memoria causal usando nodos `Fact` + aristas causales en HelixDB.
   - No se usa el producto `keggan-std/Engram` (TypeScript + SQLite). Zyro ya tiene HelixDB — duplicar la capa de datos con SQLite introduce complejidad innecesaria.
   - Esquema: 6 tipos de hecho (decision, error, preference, pattern, dependency, observation) + 7 aristas causales (CAUSED, PRECEDES, CONTRADICTS, SUPPORTS, REQUIRES, DERIVES_FROM, REFERENCES).

5. **OpenCode es el runtime, la integración va por plugins, no por escritura directa de JSON.**
   - Reemplazar la escritura manual de `opencode.jsonc` por plugins OpenCode (`@opencode-ai/plugin`).
   - Usar `@sjawhar/opencode-claude-bridge` para skills declarativas en formato `.md`.
   - Usar `opencode-lazy-loader` para lazy-loading de MCP tools (cargar solo cuando se usan).

6. **Protocolo Boomerang en cada fase: Memory → Think → Delegate → Git → Quality → Save.**
   - Cada fase completa 6 pasos: consultar memoria, pensar/analizar, delegar a subagentes, verificar git, validar calidad, guardar decisiones.
   - El ciclo completo asegura que ninguna fase termina sin persistir su contexto.

7. **El CLI debe ser auto-instalable.**
   - `zyro setup` verifica e instala todas las dependencias (uv, go, docker, helixdb, git).
   - `zyro doctor --fix` repara configuraciones rotas.
   - No se asume nada preinstalado. El CLI se levanta a sí mismo.

8. **El agente Python usa PydanticAI correctamente.**
   - La versión actual (`mcp-tools/runner.py`) usa `FastMCP` con tools sueltos, sin agente real.
   - Se refactoriza para crear un `Agent` PydanticAI con `output_type`, `capabilities`, `deferred_tools`.
   - NO se cambia de framework. Se usa PydanticAI como está diseñado.

---

## 2. Sprints de Implementación

### Sprint 0: Fundación — CLI Inteligente (Duración: 1 semana)

**Objetivo:** `zyro setup` auto-instalable, `zyro doctor --fix`, detección de entorno.

#### Archivos a crear

| Archivo | Propósito |
|---------|-----------|
| `cmd/zyrocli/cmd/setup.go` | Comando `zyro setup` — punto de entrada |
| `cmd/zyrocli/cmd/doctor.go` | Comando `zyro doctor` — diagnóstico y reparación |
| `internal/setup/check.go` | Verificadores de dependencias (OS, go, uv, docker, helix, git) |
| `internal/setup/install.go` | Instaladores automáticos (uv, helix, go) |
| `internal/setup/config.go` | Generación de `~/.zyro/config.yaml` |
| `internal/setup/doctor.go` | Reparación de configuraciones rotas |

#### Archivos a modificar

| Archivo | Cambio |
|---------|--------|
| `scripts/install.sh` | Simplificar: que llame a `zyro setup` en lugar de hacer todo manual |
| `cmd/zyrocli/main.go` | Registrar comandos `setup` y `doctor` |

#### Detalle técnico

**`zyro setup` ejecuta esta secuencia:**
1. **Verificar OS** y arquitectura (Linux x86_64/ARM64, macOS).
2. **Comprobar cada dependencia** con mensaje claro:
   - `uv` (gestor Python) → si no está, instalar desde `astral.sh/uv` con `curl -sSL https://astral.sh/uv/install.sh | bash`
   - `go` 1.21+ → si no, avisar con link oficial (Go no se auto-instala por licencia)
   - `docker` (opcional, para sandbox) → si no, avisar
   - `helixdb` → si no está en `PATH`, descargar binario desde GitHub releases de HelixDB
   - `git` (opcional pero recomendado)
3. **Crear entorno virtual Python** con `uv venv` y sincronizar `mcp-tools/pyproject.toml`.
4. **Compilar el binario Go**: `go build -o ~/.local/bin/zyrocli ./cmd/zyrocli`.
5. **Configurar MCP servers** automáticamente (helix-integration, etc.).
6. **Generar `~/.zyro/config.yaml`** con rutas y preferencias.
7. **Ejecutar `zyro doctor --fix`** para reparar problemas residuales.

**`zyro doctor --fix`:**
- Lee `~/.zyro/config.yaml`.
- Verifica cada ruta y dependencia listada.
- Para cada problema detectado, intenta reparar automáticamente o sugiere comando exacto.
- Soporta `--dry-run` para mostrar qué haría sin ejecutar.
- Soporta `--verbose` para logs detallados.
- Soporta `--json` para output machine-readable.

**Idempotencia:** Cada paso de `zyro setup` es idempotente. Si ya está instalado/configurado, hace skip con mensaje `✅ already installed`.

#### Criterio de éxito

```bash
# En una máquina Ubuntu 22.04 LIMPIA (sin go, sin uv, sin helix):
curl -sSL https://install.zyro.dev | bash
# → Sin intervención manual, todo queda listo
zyro doctor --json
# → {"status": "ok", "checks": {"go": true, "uv": true, "helix": true, "docker": false, "git": true}}
```

#### Depende de

Nada. Es el Sprint 0 — fundación del CLI.

#### Investigación relacionada

`docs/feedback/cli-inteligente-setup.md`

---

### Sprint 1: Harness Inteligente — PydanticAI Agent-as-Validator (Duración: 2 semanas)

**Objetivo:** Reemplazar el MCP server Python actual (tools sueltos en FastMCP) por un verdadero agente PydanticAI con output estructurado, capabilities separadas, y approval gates.

#### Archivos a crear

| Archivo | Propósito |
|---------|-----------|
| `mcp-tools/agent.py` | Agente PydanticAI con `output_type=AgentDecision`, tools de solo lectura |
| `mcp-tools/capabilities.py` | Separación `HelixReadCapability` / `HelixWriteCapability` |
| `mcp-tools/approval.py` | Approval gates con `deferred_tools` + `console_approver` |
| `mcp-tools/models.py` | Pydantic models compartidos: `AgentDecision`, `HelixNodeOutput` |

#### Archivos a modificar

| Archivo | Cambio |
|---------|--------|
| `mcp-tools/runner.py` | REFACTOR COMPLETO: de FastMCP tools sueltos → orquestador que llama al agente PydanticAI |
| `mcp-tools/pyproject.toml` | Agregar `pydantic-graph`, mantener `pydantic-ai`, `httpx` |
| `mcp-tools/helix_client.py` | Mantener como capa de comunicación HTTP con HelixDB (solo para debug/testing interno) |
| `mcp-tools/helix_write.py` | ELIMINAR o convertir en helper interno del orquestador Go |
| `mcp-tools/task_context.py` | Convertir en tool del agente (read-only) |
| `mcp-tools/search_code.py` | Convertir en tool del agente (read-only) |
| `mcp-tools/search_skills.py` | Convertir en tool del agente (read-only) |

#### Patrón Agent-as-Validator

```
┌────────────────────────────────────────────────────────────────────┐
│                        ORQUESTADOR GO                               │
│                                                                     │
│  1. Construye contexto desde HelixDB (Go SDK)                       │
│  2. Serializa a JSON plano                                          │
│  3. Envía al agente Python por stdin                                │
│                                                                     │
└──────────────────────────┬─────────────────────────────────────────┘
                           │ stdin (JSON plano)
                           ▼
┌────────────────────────────────────────────────────────────────────┐
│                        AGENTE PYTHON (PydanticAI)                   │
│                                                                     │
│  1. Recibe contexto plano (nunca toca HelixDB)                      │
│  2. Usa LLM para analizar y generar decisión                        │
│  3. Retorna PydanticModel validado por stdout                       │
│  4. Tools: solo lectura (search_code, task_context, search_skills)  │
│                                                                     │
└──────────────────────────┬─────────────────────────────────────────┘
                           │ stdout (JSON validado)
                           ▼
┌────────────────────────────────────────────────────────────────────┐
│                        ORQUESTADOR GO                               │
│                                                                     │
│  4. Recibe AgentDecision validado por Pydantic                      │
│  5. Si requires_approval → pausa para humano                        │
│  6. Solo tras aprobación: Go escribe a HelixDB                      │
│                                                                     │
└────────────────────────────────────────────────────────────────────┘
```

#### Approval Gates

Dos mecanismos complementarios:

**1. Deferred Tools (PydanticAI nativo):**
```python
@agent.tool(requires_approval=True)
def save_to_helix(label: str, properties: dict) -> str:
    """NUNCA se ejecuta desde el agente. Siempre requiere aprobación."""
    raise ApprovalRequired
```

**2. Approval en el orquestador Go:**
```go
// En el scheduler Go, después de recibir AgentDecision:
if decision.requiresApproval {
    approved := PromptApproval(phase, decision.Summary)
    if !approved { return }
}
// Go escribe a HelixDB
client.Exec(ctx, writeQuery, &out)
```

#### Estructura de modelos Pydantic

```python
class AgentDecision(BaseModel):
    """Output validado del agente. El orquestador ejecuta."""
    action: str  # "create" | "update" | "search" | "skip"
    reasoning: str
    nodes: list[HelixNodeOutput]
    requires_approval: bool = False

class HelixNodeOutput(BaseModel):
    label: str
    properties: dict
    project_id: int | None = None
    requires_approval: bool = False
```

#### Dependencias en pyproject.toml

```toml
[project]
dependencies = [
    "pydantic-ai>=1.95",
    "pydantic-graph",
    "httpx",
]
```

#### Criterio de éxito

```python
# El agente retorna JSON validado:
decision = await run_agent("Buscar patrones Factory Method en Go")
assert decision.action == "search"
assert len(decision.nodes) == 0  # no escribió nada

# Go escribe solo tras validación:
result = await process_decision(decision)
# result = {"status": "ok", "nodes_created": [...]}
```

#### Depende de

Sprint 0 (tener Go compilado y CLI funcionando).

#### Investigación relacionada

`docs/explorations/investigacion-01-pydanticai-harness.md`

---

### Sprint 2: Persistencia Profunda — HelixDB SDK Go (Duración: 2 semanas)

**Objetivo:** Migrar el cliente Go de raw HTTP/JSON al SDK oficial `github.com/helixdb/helix-db/sdks/go`. Implementar búsqueda híbrida (vector + BM25 con RRF), traversals complejos, y pipeline de embeddings.

#### Archivos a crear

| Archivo | Propósito |
|---------|-----------|
| `internal/db/helix/queries.go` | Queries tipadas con el DSL del SDK oficial |
| `internal/db/helix/search.go` | Búsqueda híbrida: vector + BM25 con RRF (app-side) |
| `internal/db/helix/traverse.go` | Traversals complejos: `Repeat`, `Union`, `Choose`, `Coalesce` |
| `internal/db/helix/embedding.go` | Pipeline de embeddings: worker que llama a API de embeddings |
| `internal/db/helix/indexes.go` | Creación de índices con `CreateIndexIfNotExists` |

#### Archivos a modificar

| Archivo | Cambio |
|---------|--------|
| `internal/db/helix/helix.go` | REFACTOR: migrar de `buildV3Envelope` + `doQuery` a `client.Exec(ctx, q, &out)` |
| `internal/db/helix/types.go` | Reemplazar tipos propios por tipos del SDK oficial o wrappers delgados |
| `internal/db/helix/errors.go` | Reemplazar `ErrNotFound`, `ErrConnectionFailed` por `helix.HelixError` |
| `internal/db/helix/helix_test.go` | Tests actualizados para SDK oficial |

#### Arquitectura de migración

```
ANTES (Sprint 0-2):
  internal/db/helix/helix.go
    → buildV3Envelope() → raw map[string]any
    → doQuery() → POST /v1/query
    → parseSingleNode() → json.RawMessage → Node

DESPUÉS (Sprint 2):
  internal/db/helix/queries.go
    → helix.ReadQuery("find_task")
    → q.ParamString("name", "foo")
    → helix.G().NWithLabel("Task").Where(...)
    → client.Exec(ctx, q, &out)
```

#### Detalle técnico

**SDK oficial — patrón general:**
```go
import helix "github.com/helixdb/helix-db/sdks/go"

type TaskRow struct {
    ID   int64  `json:"$id"`
    Name string `json:"name"`
}

func FindTask(name string) helix.Request {
    q := helix.ReadQuery("find_task")
    param := q.ParamString("name", name)
    return q.
        VarAs("task",
            helix.G().
                NWithLabel("Task").
                Where(helix.PredEq("name", param)).
                ValueMap("$id", "name"),
        ).
        Returning("task")
}

// Uso:
client, _ := helix.NewClient("http://localhost:6969")
var out struct { Task []TaskRow }
client.Exec(ctx, FindTask("auth"), &out)
```

**Búsqueda híbrida — RRF app-side:**
- Ejecutar `VectorSearchNodes` y `TextSearchNodes` en paralelo.
- Fusionar con Reciprocal Rank Fusion (k=60).
- Implementar en `search.go`.

**Traversals complejos:**
```go
// Cross-project skill discovery
helix.G().
    NWithLabel("Skill").
    Where(helix.PredEq("name", "typescript")).
    In("REQUIRES_SKILL").
    Out("USES_LIB").
    Dedup().
    ValueMap("$id", "name", "version")
```

**Pipeline de embeddings:**
- Worker en `embedding.go` que llama a API de embeddings (OpenAI `text-embedding-3-small` → 1536-dim).
- Cache de embeddings para evitar recalcular.
- Fallback a Ollama (`nomic-embed-text`) si no hay API key.
- Batch async para no bloquear el flujo principal.

#### Criterio de éxito

```go
// SDK oficial funciona
client, _ := helix.NewClient("http://localhost:6969")
var out FindTaskResponse
err := client.Exec(ctx, FindTask("auth"), &out)
assert.NoError(t, err)
assert.Len(t, out.Task, 1)

// Búsqueda híbrida devuelve resultados
results, err := HybridSearch(ctx, "jwt authentication", embedding, 10)
assert.Greater(t, len(results), 0)

// Traversal complejo funciona
nodes, err := traverseCrossProjectSkills(ctx, "typescript")
assert.Greater(t, len(nodes), 0)
```

#### Depende de

Sprint 0 (HelixDB corriendo, Go compilado).

#### Investigación relacionada

`docs/explorations/investigacion-02-helixdb-deep-integration.md`

---

### Sprint 3: Seguridad por Fase — Boundari (Duración: 1 semana)

**Objetivo:** Implementar políticas Boundari por fase para control granular de herramientas del agente. Cada fase (F0-F4) carga su propio `boundari.yaml` con allow/deny/approval por tool.

#### Archivos a crear

| Archivo | Propósito |
|---------|-----------|
| `boundari/phase0-boundari.yaml` | F0 — solo lectura: read_file, search_code, list_directory, git_log |
| `boundari/phase1-boundari.yaml` | F1 — lectura + web_fetch con approval |
| `boundari/phase2-boundari.yaml` | F2 — escritura de planos con approval |
| `boundari/phase3-boundari.yaml` | F3 — implementación intensiva, approval condicional |
| `boundari/phase4-boundari.yaml` | F4 — solo lectura otra vez |
| `internal/boundari/loader.go` | Carga política YAML según fase activa |
| `internal/boundari/enforcer.go` | Wrapper Go que valida acciones contra política actual |
| `mcp-tools/boundari_wrapper.py` | Wrapper Python que envuelve tools del agente con `boundary.wrap_tool()` |

#### Archivos a modificar

| Archivo | Cambio |
|---------|--------|
| `mcp-tools/agent.py` | Integrar `Boundary.from_file()` antes del loop principal del agente |

#### Mapa de políticas por fase

| Fase | Tools Permitidas | Tools Bloqueadas | Approval Requerido |
|------|-----------------|-------------------|-------------------|
| **F0** | read_file, search_code, list_directory, git_log, git_diff | write_file, delete_file, shell_exec, git_commit, git_push, network_request, npm_install | — |
| **F1** | read_file, search_code, grep_search, web_fetch, pypi_search, github_search | write_file, shell_exec, network_request | web_fetch (si URL externa), github_search (si repo no trusted) |
| **F2** | read_file, write_file, create_directory, shell_exec (comandos seguros), git_commit | delete_file, git_push, npm_install | Toda escritura, shell_exec (si comando no seguro) |
| **F3** | read_file, write_file, create_directory, shell_exec, npm_install, pip_install, git_diff | delete_file, git_commit, git_push | write_file (si no es en src/), shell_exec, pip_install |
| **F4** | read_file, search_code, git_log, git_diff, write_file (con approval) | shell_exec, delete_file, npm_install, pip_install, network_request | write_file, git_commit, git_push |

#### Budgets por fase

| Fase | max_tool_calls | max_runtime_seconds | max_cost_usd |
|------|---------------|---------------------|--------------|
| F0 | 30 | 300 | $0.10 |
| F1 | 40 | 600 | $0.25 |
| F2 | 50 | 600 | $0.35 |
| F3 | 150 | 1800 | $1.00 |
| F4 | 30 | 300 | $0.10 |

#### Flujo de integración

```python
# El scheduler Go pasa la fase como argumento:
#   python agent.py --phase F0

phase = sys.argv[1]  # "F0"
boundary = Boundary.from_file(f"boundari/phase{phase}-boundari.yaml", 
                              approver=console_approver)

# Envolver todas las MCP tools del agente
for tool_name, tool_func in mcp_tools_registry.items():
    mcp_tools_registry[tool_name] = boundary.wrap_tool(
        tool_name, tool_func, raise_on_denied=True
    )

# El agente ejecuta con todas las tools envueltas
agent.run()
```

#### Auditoría

- Cada fase escribe un archivo JSONL de auditoría: `~/.zyro/audit/<phase>-<timestamp>.jsonl`.
- Formato: `{tool, args, allowed, reason, timestamp, phase}`.
- Go puede leer estos archivos post-fase para verificar cumplimiento.

#### Criterio de éxito

```bash
# F0: el agente NO puede escribir archivos
zyro run --phase F0  # tools de escritura → Decision(allowed=False)

# F2: el agente necesita approval para shell_exec
zyro run --phase F2  # shell_exec("rm -rf /") → "approval_required"

# F3: approval condicional para write_file fuera de src/
zyro run --phase F3  # write_file("README.md") → auto-allow
                      # write_file("/etc/passwd") → "approval_required"

# Auditoría existe
ls ~/.zyro/audit/
# → phase0-20260615T120000.jsonl
```

#### Depende de

Sprint 1 (tener el agente Python funcionando con PydanticAI).

#### Investigación relacionada

`docs/explorations/investigacion-03-boundari-politicas-seguridad.md`

---

### Sprint 4: Memoria Causal — Engram Custom sobre HelixDB (Duración: 3 semanas)

**Objetivo:** Construir sistema de memoria causal persistente sobre HelixDB usando nodos `Fact` con 6 tipos + 7 aristas causales. El agente recuerda decisiones entre fases, se detectan contradicciones, y se aplica curva de olvido.

#### Arquitectura

```
NO se usa "Engram MCP Server" (keggan-std/Engram). Es un sistema CUSTOM sobre HelixDB.

Razones:
  - HelixDB ya está integrado en Zyro. Agregar SQLite + Node.js duplica infraestructura.
  - El grafo causal (nodos Fact + edges CAUSED/PRECEDES/CONTRADICTS) no es posible en SQLite plano.
  - La búsqueda semántica vectorial es nativa en HelixDB.
  - Control total sobre el modelo de datos y comportamiento.
```

#### Archivos a crear

| Archivo | Propósito |
|---------|-----------|
| `internal/memory/schema.go` | Esquema de nodos Fact (6 tipos) + aristas causales (7 tipos) |
| `internal/memory/store.go` | Guardar hechos con embeddings en HelixDB |
| `internal/memory/recall.go` | Consultar memoria relevante por similitud + navegación causal |
| `internal/memory/contradictions.go` | Detectar y resolver contradicciones entre hechos |
| `internal/memory/decay.go` | Curva de olvido (Ebbinghaus) con decaimiento configurable |
| `internal/memory/memory.go` | Structs exportados, interfaz EngramStore |
| `agents/fact_extractor.py` | Agente Python que extrae hechos de conversaciones usando LLM local |
| `internal/scheduler/memory_hook.go` | Hook pre-fase (inyectar memoria) y post-fase (extraer hechos) |

#### Archivos a modificar

| Archivo | Cambio |
|---------|--------|
| `internal/scheduler/scheduler.go` | Agregar hooks de memoria antes/después de cada fase |
| `internal/scheduler/phase.go` | Agregar campos de contexto de memoria a Config/Result |

#### Esquema de nodos Fact

```
Nodo Fact (la unidad de memoria):
  fact_id:      UUID v7
  type:         "decision" | "error" | "preference" | "pattern" | "dependency" | "observation"
  content:      Texto descriptivo del hecho
  embedding:    []float32 (1536-dim)
  salience:     float64 (0.0-1.0, importancia actual)
  confidence:   float64 (0.0-1.0, certeza del hecho)
  source:       "agent:F0" | "user:input" | "extractor:llm"
  phase:        "F0" | "F1" | "F2" | "F3" | "F4"
  created_at:   datetime
  last_accessed_at: datetime
  access_count: int64
  decay_rate:   float64 (por día, default 0.05)
  expires_at:   datetime
  is_active:    bool
  tenant_id:    string
  metadata:     json
```

#### Aristas causales

| Edge Type | Significado | Ejemplo |
|-----------|-------------|---------|
| `CAUSED` | A causó directamente B | Decisión A → Error B |
| `PRECEDES` | A ocurrió antes que B | Fase F0 → Fase F1 |
| `CONTRADICTS` | A contradice a B | Pref "usa GORM" → Dec "usamos SQLC" |
| `SUPPORTS` | A soporta o refuerza B | Pattern repo → Dec usar interfaces |
| `REQUIRES` | A requiere B para ser válido | Dec migrar SDK → Dep Go SDK instalado |
| `DERIVES_FROM` | A se deriva o infiere de B | Pattern A → Observation B |
| `REFERENCES` | A referencia a B (relación débil) | Fact → CodeNode |

#### Flujo de memoria

**Pre-fase (antes de ejecutar F1, F2, etc.):**
```
1. El scheduler Go construye query con: fase actual, descripción, objetivos
2. Llama a EngramStore.RecallMemories() con búsqueda híbrida (vector + BM25)
3. Filtra por salience > 0.2, is_active = true
4. Formatea como contexto inyectable en el prompt del agente:
   ─── MEMORIA CAUSAL (fase actual: F1) ───
   Decisiones activas:
     • Usamos Go SDK oficial de HelixDB (F0, confianza 0.95)
     • Preferencia del usuario: SQLC para queries (F0, confianza 0.88)
   Errores documentados:
     • Cliente raw JSON devolvía 404 por type mismatch en $id → resuelto
```

**Post-fase (después de ejecutar cada fase):**
```
1. El orquestador toma el log de la conversación de la fase
2. Llama al extractor Python: python fact_extractor.py --input <log> --phase F1
3. El extractor (LLM local con Ollama) parsea y extrae hechos
4. Calcula embeddings para cada hecho
5. Envía hechos al orquestador Go vía HTTP POST /api/v1/facts
6. Go guarda en HelixDB: store.SaveFact() + AddCausalEdge()
7. Go ejecuta resolución de contradicciones
8. Go refuerza salience de hechos accedidos
```

#### Resolución de contradicciones

```go
type ContradictionStrategy string

const (
    StrategyNewestWins        ContradictionStrategy = "newest_wins"
    StrategyHighestConfidence ContradictionStrategy = "highest_confidence"
    StrategyKeepBoth          ContradictionStrategy = "keep_both"
)
```

Cuando se detecta un hecho que contradice a otro (embedding similarity > 0.85 + tipos opuestos):
1. Crear edge `CONTRADICTS` entre ambos.
2. Aplicar estrategia configurable (default: `newest_wins`).
3. El hecho perdedor se marca como `is_active: false, status: "superseded_by_conflict"`.
4. Ambos se exponen en consultas de memoria para transparencia.

#### Curva de olvido (Ebbinghaus)

```
salience(t) = salience_0 * e^(-decay_rate * days_since_access)
```

- Cron job diario (`DecayAndRefresh`) recorre todos los Facts activos.
- Si `salience < threshold` (0.15), marca como `is_stale: true`.
- Si `expires_at < now`, marca como `is_active: false`.
- Cada acceso refuerza: `salience += 0.3 * (1 - salience)`.

#### Criterio de éxito

```bash
# Después de F0, inyectar contexto en F1:
zyro run --phase F0  # se ejecuta, extrae hechos
zyro run --phase F1  # el prompt incluye decisiones de F0

# Verificar memoria:
zyro context --memory
# → Muestra: "Se encontraron 12 hechos activos para el contexto actual"
# → "3 decisiones, 2 errores, 4 preferencias, 2 patrones, 1 dependencia"

# Contradicción resuelta:
# Si F0 dice "usar GORM" y F1 dice "usar SQLC"
# → Se crea edge CONTRADICTS, el más nuevo (newest_wins) prevalece
```

#### Depende de

Sprint 2 (HelixDB con SDK Go para operaciones de escritura y traversals causales).

#### Investigación relacionada

`docs/explorations/investigacion-04-engram-memoria-causal.md`

---

### Sprint 5: Ecosistema OpenCode + Protocolo Boomerang (Duración: 2 semanas)

**Objetivo:** Integrar todo el sistema con OpenCode vía plugins: bridge de skills, lazy-loading MCP, aprobaciones humanas nativas, y el ciclo Boomerang completo.

#### Archivos a crear

| Archivo | Propósito |
|---------|-----------|
| `internal/boomerang/orchestrator.go` | Orquestador del ciclo Boomerang (6 pasos por fase) |
| `internal/boomerang/memory.go` | Paso 1: consultar memoria causal en HelixDB |
| `internal/boomerang/think.go` | Paso 2: planificar DAG de tareas para la fase |
| `internal/boomerang/delegate.go` | Paso 3: repartir tareas a subagentes OpenCode |
| `internal/boomerang/git.go` | Paso 4: verificar estado del repo (diff, status) |
| `internal/boomerang/quality.go` | Paso 5: ejecutar linters, tests, validaciones |
| `internal/boomerang/save.go` | Paso 6: guardar decisiones y hechos en HelixDB |
| `internal/opencode/plugin.go` | Gestión de plugins OpenCode (bridge, lazy-loader) |

#### Archivos a modificar

| Archivo | Cambio |
|---------|--------|
| `internal/opencode/config.go` | Simplificar: solo perfiles de modelos + plugins |
| `internal/opencode/mcptools_embed.go` | Deprecar: MCP tools van por lazy-loader ahora |
| `internal/opencode/skills_embed.go` | Deprecar: skills van por claude-bridge ahora |
| `internal/scheduler/scheduler.go` | Integrar ciclo Boomerang en cada fase |
| `internal/scheduler/approval.go` | Reemplazar stdin por subagentes con `question: "ask"` |

#### Detalle técnico

**1. Plugin de Bridge (Claude → OpenCode):**

Crear plugin en `~/.config/opencode/plugins/zyrocli.ts`:

```typescript
import { createClaudeBridge } from "@sjawhar/opencode-claude-bridge";
import path from "path";
import os from "os";

export default createClaudeBridge({
  sources: [
    { dir: path.join(os.homedir(), ".config/zyrocli/skills"), namespace: "zyro" },
  ],
  claudePlugins: true,
});
```

Esto reemplaza la escritura manual de skills por Go embed. Los skills viven como archivos `.md` en `~/.config/zyrocli/skills/` y el bridge los registra automáticamente como skills + slash-commands.

**2. Lazy-loading de MCP tools:**

Configurar `opencode-lazy-loader` para que los 6 MCP tools de helix-integration se carguen solo cuando el skill que los necesita está activo:

```markdown
---
name: helix-integration
description: "HelixDB integration — search code, skills, task context"
mcp:
  helix:
    command: ["uv", "run", "--directory", "~/.config/zyrocli/mcp-tools", "runner.py"]
---

# HelixDB Integration
This skill provides HelixDB tools for code search, skill search, and context.
```

**3. Reemplazar aprobaciones stdin:**

El scheduler Go DEJA de usar `PromptApproval()` (stdin). En su lugar:
- Cada fase lanza un subagente OpenCode con `permission.question = "ask"`.
- El subagente resume lo completado y pregunta al humano via el chat de OpenCode.
- El scheduler Go monitorea el resultado del subagente (approved/rejected).

```jsonc
{
  "zyro-approval-gate": {
    "mode": "subagent",
    "description": "Gate de aprobación entre fases",
    "prompt": "{skill:zyro-approval-gate}",
    "permission": {
      "read": "allow",
      "question": "ask"
    }
  }
}
```

**4. Protocolo Boomerang — ciclo completo por fase:**

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CICLO BOOMERANG (por fase)                        │
│                                                                     │
│  ┌─────────┐    ┌─────────┐    ┌───────────┐    ┌──────┐          │
│  │ MEMORY  │───►│  THINK  │───►│ DELEGATE  │───►│ GIT  │          │
│  │ Consulta │    │ Planea  │    │ Reparte a  │    │ Verif │          │
│  │ memoria  │    │ el DAG  │    │ subagentes │    │ estado│          │
│  │ causal   │    │ de tareas│   │            │    │ repo  │          │
│  └─────────┘    └─────────┘    └───────────┘    └──────┘          │
│                                                      │              │
│  ┌─────────┐    ┌─────────┐                          │              │
│  │  SAVE   │◄───│ QUALITY │◄─────────────────────────┘              │
│  │ Guarda  │    │ Linters, │                                        │
│  │ decisión│    │ tests    │                                        │
│  │ en DB   │    │          │                                        │
│  └─────────┘    └─────────┘                                        │
└─────────────────────────────────────────────────────────────────────┘
```

Cada fase ejecuta el ciclo completo. Si QUALITY falla, vuelve a DELEGATE (máx 3 iteraciones).

**5. Comando `/zyro-approve`:**

```jsonc
{
  "command": {
    "zyro-approve": {
      "template": "Actúa como gate de aprobación. Revisa el estado actual de la fase {phase}. Resume lo completado, los archivos tocados, y los nodos creados en HelixDB. Pregunta al humano: '¿Aprobás esta fase para continuar?'. Responde solo 'approved' o 'rejected'.",
      "description": "Aprobar fase actual del pipeline SDD",
      "subtask": false,
      "agent": "zyro-approval-gate"
    }
  }
}
```

#### Criterio de éxito

```bash
# Fase 0 corre con ciclo Boomerang completo:
zyro run --phase F0
# → [Memory] Consulta memoria causal... 0 hechos (primera ejecución)
# → [Think] Planea DAG: patterns + libraries + skills en paralelo
# → [Delegate] Lanza 3 subagentes en paralelo
# → [Git] Verifica estado del repo... limpio
# → [Quality] Verifica que nodos se crearon en HelixDB... OK
# → [Save] Guarda decisiones en HelixDB... 5 hechos guardados

# Los skills se cargan via bridge (no Go embed):
ls ~/.config/opencode/skills/
# → zyro-orchestrator  zyro-sdd-apply  zyro-sdd-verify  ...
# → Cada uno es un .md, no un .go embed

# MCP tools cargan bajo demanda:
# → Sin lazy-loader: 6 MCP tools siempre cargados
# → Con lazy-loader: solo se cargan cuando el skill helix-integration se activa

# Aprobaciones humanas funcionan sin stdin:
# → OpenCode pregunta en el chat, no bloquea stdin
```

#### Depende de

Sprint 3 (Boundari — políticas por fase) + Sprint 4 (Memoria Causal — contexto entre fases).

#### Investigación relacionada

`docs/explorations/investigacion-05-opencode-ecosistema-plugins.md`

---

## 3. Arquitectura Final (Diagrama ASCII)

```
┌────────────────────────────────────────────────────────────────────────────┐
│                              TERMINAL                                      │
│   zyro setup | zyro init | zyro run | zyro doctor --fix                   │
└──────────────────────────────┬─────────────────────────────────────────────┘
                               │
┌──────────────────────────────▼─────────────────────────────────────────────┐
│                         ZYROCLI (Go 1.26+)                                  │
│                                                                             │
│  ┌─────────────────┐  ┌────────────────┐  ┌────────────────────────┐       │
│  │ setup/           │  │ scheduler/     │  │ db/helix/ (SDK Go)     │       │
│  │ check.go         │  │ scheduler.go  │  │ queries.go (oficial)   │       │
│  │ install.go       │  │ phase.go      │  │ search.go (híbrida)    │       │
│  │ config.go        │  │ approval.go   │  │ traverse.go            │       │
│  │ doctor.go        │  │ memory_hook   │  │ embedding.go           │       │
│  └─────────────────┘  └───────┬────────┘  │ indexes.go             │       │
│                               │            └────────────────────────┘       │
│  ┌─────────────────┐  ┌───────▼────────┐  ┌────────────────────────┐       │
│  │ boomerang/       │  │ memory/        │  │ boundari/              │       │
│  │ orchestrator.go  │  │ schema.go      │  │ loader.go              │       │
│  │ think.go         │  │ store.go       │  │ enforcer.go            │       │
│  │ delegate.go      │  │ recall.go      │  │ policies/*.yaml        │       │
│  │ git.go           │  │ contradictions │  └────────────────────────┘       │
│  │ quality.go       │  │ decay.go       │                                    │
│  │ save.go          │  └────────────────┘                                    │
│  └─────────────────┘                                                         │
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────┐       │
│  │ opencode/                                                       │       │
│  │  - config.go (solo perfiles + plugins)                          │       │
│  │  - plugin.go (gestiona bridge + lazy-loader)                    │       │
│  │  - skills_embed.go → DEPRECADO (ahora son .md via bridge)       │       │
│  │  - mcptools_embed.go → DEPRECADO (ahora lazy-loading)           │       │
│  └──────────────────────────────────────────────────────────────────┘       │
└──────────────────────────────┬─────────────────────────────────────────────┘
                               │
          ┌────────────────────┼────────────────────────────┐
          │  stdin/json (Agent  │  Go SDK (writes)           │
          │  -as-Validator)     │  HTTP (reads)              │
          ▼                    ▼                             ▼
┌────────────────────┐ ┌──────────────────┐ ┌────────────────────────┐
│  Agente Python     │ │   OpenCode       │ │   HelixDB (Rust)       │
│  (PydanticAI)      │ │   (Runtime IA)   │ │                        │
│                    │ │                  │ │   ┌──────────────────┐ │
│  Agent-as-Validator│ │   Plugins:       │ │   │  Graph (nodes+  │ │
│  → output validado │ │   • claude-bridge│ │   │  edges)         │ │
│                    │ │   • lazy-loader  │ │   ├──────────────────┤ │
│  Tools READ-ONLY:  │ │   • multiagent   │ │   │  Vector (ANN)   │ │
│  • search_code     │ │                  │ │   ├──────────────────┤ │
│  • search_skills   │ │   Skills (14):   │ │   │  Text (BM25)    │ │
│  • task_context    │ │   declarativas   │ │   ├──────────────────┤ │
│                    │ │   en .md         │ │   │  Memoria Causal │ │
│  Boundari wrapper  │ │                  │ │   │  (Facts + edges)│ │
│  (wrap_tool)       │ │   13 subagentes  │ │   └──────────────────┘ │
└────────────────────┘ └──────────────────┘ └────────────────────────┘
```

### Flujo de datos (resumen)

```
1. HUMANO ejecuta: zyro run --phase F0
2. SCHEDULER Go inicia fase:
   a. Boomerang.Memory() → consulta memoria causal en HelixDB
   b. Boomerang.Think() → planea DAG de tareas
   c. Boomerang.Delegate() → lanza subagentes en OpenCode
3. OPENCODE ejecuta subagentes con:
   - Skills cargadas via claude-bridge
   - MCP tools cargadas via lazy-loader
   - Políticas Boundari aplicadas (wrap_tool)
   - Agent-as-Validator: agente retorna JSON, Go escribe a HelixDB
4. POST-FASE:
   a. Boomerang.Quality() → verifica nodos creados en HelixDB
   b. Boomerang.Save() → extrae hechos, guarda memoria causal
   c. Approval gate → subagente pregunta al humano
5. HUMANO aprueba → SCHEDULER pasa a siguiente fase
```

---

## 4. Dependencias Entre Sprints

```
Sprint 0 (Setup + Doctor)
  │
  ├──► Sprint 1 (PydanticAI Agent-as-Validator)
  │       │
  │       └──► Sprint 3 (Boundari: políticas por fase)
  │
  └──► Sprint 2 (HelixDB SDK Go + búsqueda híbrida)
          │
          └──► Sprint 4 (Memoria Causal sobre HelixDB)
                  │
                  └──► Sprint 5 (OpenCode Plugins + Boomerang)
                          ▲
                          │
                  (Requiere Sprint 3 + Sprint 4)
```

| Dependencia | Razón |
|-------------|-------|
| **Sprint 1 → Sprint 0** | El agente Python necesita que el CLI Go esté compilado y funcionando para recibir/sender datos |
| **Sprint 2 → Sprint 0** | HelixDB SDK Go necesita HelixDB corriendo (verificado por `zyro doctor`) |
| **Sprint 3 → Sprint 1** | Boundari envuelve tools del agente PydanticAI — necesita el agente funcionando |
| **Sprint 4 → Sprint 2** | Memoria causal usa traversals de HelixDB SDK (Repeat, Union) + búsqueda híbrida |
| **Sprint 5 → Sprint 3** | El plugin bridge necesita Boundari para control de herramientas |
| **Sprint 5 → Sprint 4** | Boomerang necesita memoria causal para el paso Memory del ciclo |

**Sprint 1 y Sprint 2 son paralelos** — el harness del agente y la persistencia profunda son independientes.

---

## 5. Riesgos y Mitigaciones

| # | Riesgo | Impacto | Probabilidad | Mitigación |
|---|--------|---------|--------------|------------|
| 1 | `uv` no disponible en arquitectura ARM | Alto | Baja | Fallback a `pip` + venv clásico. Verificar en CI para ARM64. |
| 2 | HelixDB SDK Go cambia API entre versiones | Medio | Media | Versionar dependencia exacta en `go.mod`. Tests de integración en CI que validen queries reales contra HelixDB. |
| 3 | Boundari v0.1.0 alpha — cambios de API | Alto | Media | Congelar versión exacta en `pyproject.toml`: `boundari==0.1.0`. Implementar wrapper Go que pueda funcionar SIN Boundari (fallback a políticas hardcodeadas en Go). |
| 4 | Embeddings locales muy lentos (Ollama en CPU) | Medio | Alta | Cache de embeddings (LRU de 1000 entradas). Batch asíncrono para no bloquear el flujo principal. Usar API de OpenAI si está disponible. |
| 5 | OpenCode plugin system inestable o cambia API | Alto | Media | Mantener la escritura directa de `opencode.jsonc` como fallback. El plugin es mejora, no requisito. |
| 6 | El agente Python se cuelga o tarda demasiado | Alto | Media | Timeout configurable en Go (default 120s). Kill + restart del proceso Python. Logs de timeout para debugging. |
| 7 | LLM local del extractor de hechos produce falsos positivos | Medio | Alta | El extractor asigna `confidence` a cada hecho. Umbral mínimo configurable (default 0.6). Revisión humana opcional post-extracción. |
| 8 | El grafo causal crece sin control | Bajo | Media | Límite de hechos activos por tenant (default 10000). Decaimiento diario. Archivo de hechos stale cada 90 días. |
| 9 | Contradicciones no detectadas por embedding similarity | Medio | Media | Además de embedding, usar reglas explícitas: mismo tipo + mismo source_phase + contenido opuesto por LLM. El humano puede marcar contradicciones manualmente. |
| 10 | OpenCode no soporta paralelización nativa de subagentes | Medio | Alta | El plugin `opencode-multiagent` puede ayudar. Si no, el orquestador Go lanza múltiples procesos OpenCode en paralelo y consolida resultados. |

---

## 6. Criterios de Éxito — Pruebas de Integración

### Prueba 1: Máquina limpia (Ubuntu 22.04)

```bash
# Estado inicial: solo curl + bash
which go     # → not found
which uv     # → not found
which helix  # → not found

# Instalación automática
curl -sSL https://install.zyro.dev | bash
# ✅ uv instalado desde astral.sh
# ✅ HelixDB instalado y corriendo (localhost:6969)
# ✅ zyro compilado en ~/.local/bin/
# ✅ ~/.zyro/config.yaml generado

# Verificación
zyro doctor
# → ✅ go 1.26.0
# → ✅ uv 0.6.0
# → ✅ helix running (localhost:6969)
# → ⚠️ docker not found (optional)
# → ✅ git 2.34.1

zyro setup --dry-run
# → "Everything is already set up. Nothing to do."
```

### Prueba 2: Pipeline completo F0→F4

```bash
zyro init docs/examples/test-handoff.yaml
# → Estructura de proyecto creada
# → Nodos Developer + Project en HelixDB

zyro run --phase F0
# → Boomerang ejecuta ciclo completo
# → 3 subagentes en paralelo (patterns, libraries, skills)
# → Nodos Pattern, Library, Skill creados en HelixDB
# → Approval gate: subagente pregunta "¿Aprobás?"
# → Humano dice sí

zyro run --phase F1
# → Pre-fase: memoria causal inyecta decisiones de F0
# → "Usamos Go SDK oficial de HelixDB (decisión, F0)"
# → "Preferencia del usuario: SQLC para queries (F0)"
# → El agente F1 recibe contexto automáticamente
```

### Prueba 3: Seguridad

```bash
# F0: tools de escritura bloqueadas
zyro run --phase F0
# El agente intenta write_file("test.txt") → Decision(allowed=False)
# El agente intenta shell_exec("rm -rf /") → Decision(allowed=False)

# F3: approval condicional
zyro run --phase F3
# write_file("src/app.ts") → auto-allow (está en src/)
# write_file("/etc/hosts") → "approval_required"

# Presupuestos
# F0: después de 30 tool calls → "budget_exceeded"
# F3: después de $1.00 en costo → "budget_exceeded"
```

### Prueba 4: Memoria causal

```bash
# Ejecutar F0
zyro run --phase F0
# → Extrae hechos

# Ver memoria
zyro context --memory
# → 5 hechos activos:
#   • decision: "Usar Go SDK oficial de HelixDB" (F0, confianza 0.95)
#   • preference: "SQLC para queries" (F0, confianza 0.88)
#   • pattern: "Repository pattern con interfaces" (F0, confianza 0.80)
#   • error: "404 por type mismatch en $id" (F0, confianza 0.75)
#   • dependency: "Fase F2 requiere SDK Go" (F0, confianza 0.90)

# Ejecutar F1 con contexto automático
zyro run --phase F1
# El prompt del agente F1 incluye:
# "CONTEXTO DE MEMORIA CAUSAL:
#  • Usar Go SDK oficial de HelixDB (decisión aprobada)
#  • Preferencia: SQLC para queries (preferencia del usuario)
#  • Patrón: Repository pattern (patrón identificado)"

# Contradicción:
# Si F1 dice "usar GORM", se detecta contradicción con pref "SQLC"
# → Edge CONTRADICTS creado
# → newest_wins: prevalece "SQLC" (de F1)
# → "usar GORM" marcado como superseded
```

### Prueba 5: Ecosistema OpenCode

```bash
# Skills cargadas via bridge (no Go embed)
ls ~/.config/opencode/skills/
# → zyro-orchestrator/  zyro-sdd-apply/  zyro-sdd-verify/  ...
# → Cada directorio tiene SKILL.md

# opencode.jsonc tiene plugins
cat ~/.config/opencode/opencode.jsonc
# → "plugin": [
#     "@sjawhar/opencode-claude-bridge",
#     "opencode-lazy-loader",
#     "opencode-multiagent"
#   ]

# Slash command /zyro-approve funciona
# En chat de OpenCode: /zyro-approve
# → Subagente gate revisa estado, pregunta al humano

# MCP tools cargan bajo demanda
# Sin el skill helix-integration activo: 0 MCP tools en memoria
# Con skill activo: 6 MCP tools disponibles
```

---

## 7. Resumen de Archivos a Crear/Modificar

### Sprint 0 — CLI Inteligente (~800 líneas)

| Tipo | Archivo | Acción |
|------|---------|--------|
| NEW | `cmd/zyrocli/cmd/setup.go` | Crear |
| NEW | `cmd/zyrocli/cmd/doctor.go` | Crear |
| NEW | `internal/setup/check.go` | Crear |
| NEW | `internal/setup/install.go` | Crear |
| NEW | `internal/setup/config.go` | Crear |
| NEW | `internal/setup/doctor.go` | Crear |
| MOD | `scripts/install.sh` | Simplificar (que llame a `zyro setup`) |
| MOD | `cmd/zyrocli/main.go` | Agregar comandos setup/doctor |

### Sprint 1 — Agent-as-Validator (~600 líneas)

| Tipo | Archivo | Acción |
|------|---------|--------|
| NEW | `mcp-tools/agent.py` | Crear |
| NEW | `mcp-tools/capabilities.py` | Crear |
| NEW | `mcp-tools/approval.py` | Crear |
| NEW | `mcp-tools/models.py` | Crear |
| MOD | `mcp-tools/runner.py` | Refactor completo |
| MOD | `mcp-tools/pyproject.toml` | Agregar dependencias |
| MOD | `mcp-tools/helix_write.py` | Eliminar o convertir en helper |
| MOD | `mcp-tools/task_context.py` | Convertir en tool read-only |
| MOD | `mcp-tools/search_code.py` | Convertir en tool read-only |
| MOD | `mcp-tools/search_skills.py` | Convertir en tool read-only |

### Sprint 2 — HelixDB SDK Go (~700 líneas)

| Tipo | Archivo | Acción |
|------|---------|--------|
| NEW | `internal/db/helix/queries.go` | Crear |
| NEW | `internal/db/helix/search.go` | Crear |
| NEW | `internal/db/helix/traverse.go` | Crear |
| NEW | `internal/db/helix/embedding.go` | Crear |
| NEW | `internal/db/helix/indexes.go` | Crear |
| MOD | `internal/db/helix/helix.go` | Refactor a SDK oficial |
| MOD | `internal/db/helix/types.go` | Reemplazar tipos |
| MOD | `internal/db/helix/errors.go` | Reemplazar errores |
| MOD | `internal/db/helix/helix_test.go` | Actualizar tests |

### Sprint 3 — Boundari (~400 líneas)

| Tipo | Archivo | Acción |
|------|---------|--------|
| NEW | `boundari/phase0-boundari.yaml` | Crear |
| NEW | `boundari/phase1-boundari.yaml` | Crear |
| NEW | `boundari/phase2-boundari.yaml` | Crear |
| NEW | `boundari/phase3-boundari.yaml` | Crear |
| NEW | `boundari/phase4-boundari.yaml` | Crear |
| NEW | `internal/boundari/loader.go` | Crear |
| NEW | `internal/boundari/enforcer.go` | Crear |
| NEW | `mcp-tools/boundari_wrapper.py` | Crear |
| MOD | `mcp-tools/agent.py` | Integrar Boundary.from_file() |

### Sprint 4 — Memoria Causal (~900 líneas)

| Tipo | Archivo | Acción |
|------|---------|--------|
| NEW | `internal/memory/schema.go` | Crear |
| NEW | `internal/memory/store.go` | Crear |
| NEW | `internal/memory/recall.go` | Crear |
| NEW | `internal/memory/contradictions.go` | Crear |
| NEW | `internal/memory/decay.go` | Crear |
| NEW | `internal/memory/memory.go` | Crear |
| NEW | `agents/fact_extractor.py` | Crear |
| NEW | `internal/scheduler/memory_hook.go` | Crear |
| MOD | `internal/scheduler/scheduler.go` | Agregar hooks de memoria |
| MOD | `internal/scheduler/phase.go` | Agregar campos de contexto |

### Sprint 5 — OpenCode Plugins + Boomerang (~1000 líneas)

| Tipo | Archivo | Acción |
|------|---------|--------|
| NEW | `internal/boomerang/orchestrator.go` | Crear |
| NEW | `internal/boomerang/memory.go` | Crear |
| NEW | `internal/boomerang/think.go` | Crear |
| NEW | `internal/boomerang/delegate.go` | Crear |
| NEW | `internal/boomerang/git.go` | Crear |
| NEW | `internal/boomerang/quality.go` | Crear |
| NEW | `internal/boomerang/save.go` | Crear |
| NEW | `internal/opencode/plugin.go` | Crear |
| MOD | `internal/opencode/config.go` | Simplificar |
| MOD | `internal/scheduler/scheduler.go` | Integrar Boomerang |
| MOD | `internal/scheduler/approval.go` | Reemplazar stdin |

### Totales

| Sprint | Archivos Nuevos | Archivos Modificados | Líneas Estimadas |
|--------|----------------|---------------------|-------------------|
| Sprint 0 | 6 | 2 | ~800 |
| Sprint 1 | 4 | 6 | ~600 |
| Sprint 2 | 5 | 4 | ~700 |
| Sprint 3 | 8 | 1 | ~400 |
| Sprint 4 | 8 | 2 | ~900 |
| Sprint 5 | 8 | 3 | ~1000 |
| **Total** | **39** | **18** | **~4400** |

---

## 8. Glosario de Términos

| Término | Definición |
|---------|------------|
| **Agent-as-Validator** | Patrón donde el agente LLM retorna un PydanticModel validado y el orquestador (Go) ejecuta las acciones. El agente nunca escribe directamente. |
| **Boomerang** | Protocolo de 6 pasos (Memory→Think→Delegate→Git→Quality→Save) ejecutado en cada fase del pipeline SDD. |
| **Boundari** | Librería Python de políticas-as-código para control granular de tools de agentes por fase. |
| **Engram (Zyro)** | Sistema de memoria causal CUSTOM sobre HelixDB usando nodos Fact + aristas causales. NO es el MCP server `keggan-std/Engram`. |
| **HelixDB SDK Go** | SDK oficial `github.com/helixdb/helix-db/sdks/go` con DSL fluido para queries, traversals, y escrituras. |
| **Lazy-loading MCP** | Plugin OpenCode que carga servidores MCP solo cuando se usan, no al arranque. |
| **claude-bridge** | Plugin OpenCode que traduce skills/agentes/comandos de Claude Code a formato OpenCode. |
| **RRF** | Reciprocal Rank Fusion — algoritmo para fusionar rankings de búsqueda vectorial y BM25. |
| **Fact** | Nodo atómico de memoria causal. 6 tipos: decision, error, preference, pattern, dependency, observation. |
| **Salience** | Métrica de importancia de un hecho (0.0-1.0). Decae con el tiempo según curva de Ebbinghaus. |

---

## 9. Referencias

### Investigaciones
- `docs/explorations/investigacion-01-pydanticai-harness.md` — PydanticAI Agent-as-Validator
- `docs/explorations/investigacion-02-helixdb-deep-integration.md` — HelixDB SDK Go + búsqueda híbrida
- `docs/explorations/investigacion-03-boundari-politicas-seguridad.md` — Boundari por fase
- `docs/explorations/investigacion-04-engram-memoria-causal.md` — Memoria causal sobre HelixDB
- `docs/explorations/investigacion-05-opencode-ecosistema-plugins.md` — OpenCode plugins + Boomerang

### Feedback
- `docs/feedback/cli-inteligente-setup.md` — CLI auto-instalable

### Arquitectura
- `docs/architecture-v2.md` — Arquitectura decisional v2
- `docs/helixdb-integration.md` — Integración HelixDB actual

### Código existente
- `internal/scheduler/` — Scheduler Go con fases F0-F4
- `internal/db/helix/` — Cliente HelixDB actual (raw HTTP)
- `internal/opencode/` — Configuración OpenCode
- `mcp-tools/` — MCP server Python
- `cmd/zyrocli/` — CLI commands
- `scripts/install.sh` — Script de instalación actual

### Dependencias externas
- **PydanticAI**: https://ai.pydantic.dev/
- **HelixDB Go SDK**: `github.com/helixdb/helix-db/sdks/go`
- **Boundari**: `pip install boundari`
- **opencode-claude-bridge**: `npm i @sjawhar/opencode-claude-bridge`
- **opencode-lazy-loader**: `npm i opencode-lazy-loader`
- **opencode-multiagent**: `npm i opencode-multiagent`
- **@opencode-ai/plugin**: `npm i @opencode-ai/plugin`
