# Spec ZyroAgentCLI v2 — Fase 1

> **Versión:** 2.0.0-spec  
> **Fecha:** 2026-06-15  
> **Pipeline:** SDD Fase 1  
> **Sprints:** S0–S5  
> **Archivo:** docs/spec-zyrov2.md  

---

## 1. Contexto

### 1.1 Problema que Resuelve Zyro v2

ZyroAgentCLI v1 (MVP actual) es un companion funcional que configura modelos OpenCode por fase, administra un grafo de proyecto en HelixDB (vía HTTP raw), y ejecuta un scheduler F1 con parseo de handoff + approval gates. Sin embargo, enfrenta limitaciones críticas:

| Problema | Impacto |
|----------|---------|
| **CLI frágil** — `scripts/install.sh` asume Go, uv, HelixDB preinstalados | Incorporación de nuevos developers bloqueada |
| **Agente Python sin estructura** — tools sueltos en FastMCP sin Agent PydanticAI real | El orquestador no puede confiar en el output |
| **Cliente HelixDB raw** — construye JSON a mano con `buildV3Envelope()` | Sin type safety, sin traversals, sin búsqueda híbrida |
| **Sin seguridad por fase** — cualquier tool del agente se ejecuta en cualquier fase | Riesgo: F0 podría ejecutar comandos peligrosos |
| **Sin memoria persistente** — decisiones de F0 se pierden al llegar a F1 | Cada fase empieza desde cero |
| **Skills y MCP tools embedidas en Go** — sin flexibilidad ni lazy-loading | Todo atado al binario |
| **Aprobaciones por stdin** — `PromptApproval()` bloquea el flujo | Rompe UX de OpenCode basada en chat |

Zyro v2 resuelve todo esto con 6 sprints incrementalmente dependientes que transforman el CLI en un pipeline SDD completo, auto-instalable, seguro, con memoria causal, y profundamente integrado con OpenCode.

### 1.2 Aprendizaje de las 5 Investigaciones

**Investigación 1 — PydanticAI Harness (Agent-as-Validator):**
- El agente debe retornar un `PydanticModel` validado, no un string libre.
- Separación `HelixReadCapability` vs `HelixWriteCapability` (solo Go escribe).
- `@tool(requires_approval=True)` marca gates que nunca escalan desde el agente.
- `PydanticGraph` para state machines formales (plan→execute→review→persist).
- Decisión: NO cambiar de framework — PydanticAI es el correcto, hay que usarlo bien.

**Investigación 2 — HelixDB Deep Integration:**
- SDK oficial `github.com/helixdb/helix-db/sdks/go` reemplaza raw HTTP.
- Embeddings app-side (OpenAI text-embedding-3-small 1536-dim, fallback Ollama).
- Búsqueda híbrida: Vector ANN + BM25 fusionados con RRF (k=60).
- Traversals complejos: `Repeat`, `Union`, `Choose`, `Coalesce`.
- `ForEachParam` para batch writes, `varAsIf` para upsert condicional.

**Investigación 3 — Boundari Security Policies:**
- Políticas-as-código en YAML, una por fase (F0–F4).
- Cada política define: `allow`, `deny`, `approval` por tool + budgets.
- Auditoría en JSONL. Version pinning + fallback Go hardcodeado.
- Boundari en Python envuelve tools del agente; Go tiene su propio `enforcer.go`.

**Investigación 4 — Engram/Memoria Causal (Custom sobre HelixDB):**
- NO usar `keggan-std/Engram` — Zyro ya tiene HelixDB, duplicar con SQLite no tiene sentido.
- Sistema custom: nodos `Fact` con 6 tipos + 7 aristas causales.
- Tipos: `decision`, `error`, `preference`, `pattern`, `dependency`, `observation`.
- Aristas: `CAUSED`, `PRECEDES`, `CONTRADICTS`, `SUPPORTS`, `REQUIRES`, `DERIVES_FROM`, `REFERENCES`.
- Pre-fase: inyectar memoria (salience > 0.2) en prompt del agente.
- Post-fase: extractor LLM parsea log y genera hechos con embeddings.
- Ebbinghaus: `salience(t) = salience_0 * e^(-decay_rate * days_since_access)`.

**Investigación 5 — OpenCode Ecosystem + Plugins:**
- Bridge `@sjhawar/opencode-claude-bridge` reemplaza Go embed → skills como `.md`.
- `opencode-lazy-loader` carga MCP servers bajo demanda.
- Approval gates vía subagentes con `question: "ask"`.
- Boomerang: 6 pasos por fase (Memory→Think→Delegate→Git→Quality→Save).
- `/zyro-approve` slash command para aprobación humana nativa.

### 1.3 Feedback del Mundo Real (PC del hermano)

La prueba en PC limpio reveló: `install.sh` asume `uv` preinstalado, HelixDB no se descarga automáticamente, Go build falla si no hay Go, y no se genera configuración automática.

**Soluciones:**
- `zyro setup` auto-instala uv, HelixDB, compila binario Go, genera config.
- `zyro doctor --fix` verifica y repara configuraciones rotas.
- Instalación atómica: `curl https://install.zyro.dev | bash` → todo listo.
- Cada paso es idempotente: si ya existe, salta con mensaje claro.

### 1.4 Arquitectura Actual vs Deseada

**Estado actual (v1 MVP):**
```
Terminal → ZyroCLI (Go raw HTTP → HelixDB)
            → OpenCode (sin plugins, skills embed)
              → MCP Python (FastMCP tools sueltos)
```

Componentes existentes que NO se tocan:
- ✅ Scheduler F1 (parseo handoff + approval gates)
- ✅ Handoff parser, Scaffold, Doc sync, Context bridge, Contract testing

Componentes existentes a REFACTORIZAR:
- ⚠️ `internal/db/helix/` — migrar de raw HTTP al SDK oficial Go
- ⚠️ `mcp-tools/runner.py` — refactor completo a Agent PydanticAI
- ⚠️ `internal/opencode/mcptools_embed.go` + `skills_embed.go` — deprecar
- ⚠️ `internal/scheduler/approval.go` — reemplazar stdin por subagentes

**Estado deseado (v2):**
```
Terminal → ZyroCLI (Go SDK oficial)
            → HelixDB (Rust: graph + vector + text search)
            → OpenCode (plugins: bridge + lazy-loader + multiagent)
              → Skills declarativas en .md
              → MCP tools bajo demanda
              → Approval gates en chat
                → Agente Python Agent-as-Validator (PydanticAI)
                  → Read tools envueltas en Boundari
                    → Output validado → Go escribe a DB
```

---

## 2. Inputs

### 2.1 Código Fuente Base

| Componente | Ruta | Líneas | Propósito |
|-----------|------|--------|-----------|
| Scheduler Go | `internal/scheduler/scheduler.go` | ~400 | Orquestador de fases F0-F4 |
| Cliente HelixDB | `internal/db/helix/helix.go` | ~250 | Raw HTTP vía buildV3Envelope + doQuery |
| Config OpenCode | `internal/opencode/config.go` | ~180 | Reader/writer de opencode.jsonc |
| Perfiles | `internal/opencode/model.go` | ~120 | Lista curada de providers + modelos |
| MCP runner | `mcp-tools/runner.py` | ~150 | FastMCP server con tools sueltos |
| Tools Python | `mcp-tools/*.py` | ~350 | Herramientas MCP no estructuradas |
| CLI commands | `cmd/zyrocli/` | ~200 | Comandos existentes |
| Instalador | `scripts/install.sh` | ~80 | Script manual frágil |
| Tests | `*_test.go` | ~249 | Tests existentes |

### 2.2 Documentos de Investigación

| Documento | Decisiones Clave |
|-----------|-----------------|
| `docs/explorations/investigacion-01-pydanticai-harness.md` | Agent-as-Validator, deferred_tools, PydanticGraph |
| `docs/explorations/investigacion-02-helixdb-deep-integration.md` | SDK Go oficial, RRF, traversals, embeddings |
| `docs/explorations/investigacion-03-boundari-politicas-seguridad.md` | Políticas YAML por fase, auditoría JSONL |
| `docs/explorations/investigacion-04-engram-memoria-causal.md` | 6 Fact types, 7 causal edges, Ebbinghaus |
| `docs/explorations/investigacion-05-opencode-ecosistema-plugins.md` | Bridge, lazy-loader, Boomerang |
| `docs/feedback/cli-inteligente-setup.md` | Setup auto-instalable, doctor --fix |
| `docs/roadmap-integrado.md` | 6 sprints, 39 archivos nuevos, 18 modificados, ~4400 líneas |
| `docs/architecture-v2.md` | Arquitectura decisional, schema HelixDB, modelo híbrido |

---

## 3. Outputs (La Especificación)

### A. CLI Inteligente (Sprint 0)

**Objetivo:** `zyro setup` + `zyro doctor --fix` con auto-instalación idempotente.

#### 3.A.1 Comando `zyro setup`

**Ruta:** `cmd/zyrocli/cmd/setup.go`

```go
var SetupCmd = &cobra.Command{
    Use:   "setup",
    Short: "Auto-install all dependencies and configure Zyro",
    RunE:  runSetup,
}

type SetupFlags struct {
    DryRun    bool   // --dry-run: mostrar qué haría sin ejecutar
    Verbose   bool   // --verbose: logs detallados
    SkipGo    bool   // --skip-go: no compilar Go (útil en CI)
    TargetDir string // --target-dir: directorio de instalación
}
```

**Secuencia de ejecución:**
```
setup.Run():
  1. VerifyOS()           → linux (x86_64|arm64) o darwin
  2. CheckDependencies()  → uv, go, docker, helixdb, git
  3. InstallUV()          → curl astral.sh/uv/install.sh | bash
  4. InstallHelixDB()     → descargar release de GitHub, extraer a PATH
  5. CreateVenv()         → uv venv && uv sync (mcp-tools/pyproject.toml)
  6. BuildGo()            → go build -o ~/.local/bin/zyrocli ./cmd/zyrocli
  7. GenerateConfig()     → ~/.zyro/config.yaml con rutas
  8. RegisterMCP()        → configurar MCP servers en opencode.jsonc
  9. DoctorFix()          → reparar problemas residuales
```

**Config generada (`~/.zyro/config.yaml`):**
```yaml
version: "2.0"
paths:
  helix_binary: /home/user/.local/bin/helixdb
  zyro_binary: /home/user/.local/bin/zyrocli
  venv: /home/user/.local/share/zyro/venv
  mcp_tools: /home/user/.local/share/zyro/mcp-tools
  config_dir: /home/user/.config/zyro
  skills_dir: /home/user/.config/zyro/skills
  audit_dir: /home/user/.config/zyro/audit
helixdb:
  host: localhost
  port: 6969
  auto_start: true
mcp:
  auto_register: true
  lazy_loader: true
```

#### 3.A.2 Comando `zyro doctor`

**Ruta:** `cmd/zyrocli/cmd/doctor.go`

```go
type Check struct {
    Name    string   `json:"name"`
    Status  Status   `json:"status"`   // ok | warning | error | missing
    Version string   `json:"version,omitempty"`
    Detail  string   `json:"detail,omitempty"`
    Fix     *FixStep `json:"fix,omitempty"`
}

type DoctorReport struct {
    Timestamp string  `json:"timestamp"`
    Status    string  `json:"status"`   // ok | issues_found | fixed
    Checks    []Check `json:"checks"`
}
```

#### 3.A.3 Módulos Internos

**Ruta:** `internal/setup/check.go`
```go
type DependencyChecker struct {
    results  map[Dependency]*CheckResult
    platform PlatformInfo
}

func (dc *DependencyChecker) CheckAll(ctx context.Context) map[Dependency]*CheckResult
func (dc *DependencyChecker) CheckOne(ctx context.Context, dep Dependency) *CheckResult
```

**Ruta:** `internal/setup/install.go`
```go
type Installer struct {
    DryRun  bool
    Verbose bool
}

func (inst *Installer) InstallUV(ctx context.Context) error
func (inst *Installer) InstallHelixDB(ctx context.Context) error
func (inst *Installer) CreateVenv(ctx context.Context, toolsDir string) error
func (inst *Installer) BuildGoBinary(ctx context.Context, outputPath string) error
```

Cada método es idempotente: verifica si ya existe, salta con `✅ already installed`.

#### 3.A.4 Archivos Modificados

- **`cmd/zyrocli/main.go`**: agregar `rootCmd.AddCommand(SetupCmd)` y `rootCmd.AddCommand(DoctorCmd)`.
- **`scripts/install.sh`**: simplificar a `curl -sSL https://install.zyro.dev | bash`.

#### 3.A.5 Criterios de Éxito

```bash
# Máquina Ubuntu 22.04 LIMPIA — sin go, sin uv, sin helix
curl -sSL https://install.zyro.dev | bash
# → ✅ uv instalado, ✅ HelixDB corriendo en :6969
# → ✅ zyro compilado en ~/.local/bin/, ✅ config generada

zyro doctor --json
# → {"status": "ok", "checks": {"go": true, "uv": true, "helix": true, ...}}

zyro setup --dry-run
# → "Everything is already set up. Nothing to do."
```

---

### B. Harness Agent-as-Validator (Sprint 1)

**Objetivo:** Reemplazar `mcp-tools/runner.py` (FastMCP tools sueltos) por un agente PydanticAI con output estructurado, capabilities separadas y approval gates.

#### 3.B.1 Arquitectura

```
ORQUESTADOR GO                     AGENTE PYTHON
┌─────────────────────┐            ┌──────────────────────┐
│ scheduler.go        │  stdin     │ agent.py             │
│ 1. Construye        │────JSON───►│ 1. Recibe contexto   │
│    contexto desde   │            │    plano (JSON)      │
│    HelixDB (Go SDK) │            │ 2. LLM analiza       │
│ 2. Serializa a JSON │            │ 3. Tools READ-ONLY:  │
│    plano            │            │    search_code       │
│ 3. Envía a Python   │            │    search_skills     │
│    por stdin        │            │    task_context      │
│                     │  stdout    │ 4. Retorna           │
│ 4. Recibe           │◄───JSON────│    AgentDecision     │
│    AgentDecision    │            │    validado          │
│ 5. Si requiere      │            └──────────────────────┘
│    approval → gate  │
│ 6. Go escribe a     │
│    HelixDB          │
└─────────────────────┘
```

#### 3.B.2 Modelos Pydantic

**Ruta:** `mcp-tools/models.py`

```python
from pydantic import BaseModel, Field
from enum import Enum
from typing import Optional

class Action(str, Enum):
    create = "create"
    update = "update"
    search = "search"
    skip = "skip"

class HelixNodeOutput(BaseModel):
    label: str
    properties: dict = Field(default_factory=dict)
    project_id: Optional[int] = None
    requires_approval: bool = False

class AgentDecision(BaseModel):
    action: Action
    reasoning: str = Field(..., min_length=10)
    nodes: list[HelixNodeOutput] = Field(default_factory=list)
    requires_approval: bool = False
    metadata: dict = Field(default_factory=dict)
```

#### 3.B.3 Capabilities

**Ruta:** `mcp-tools/capabilities.py`

```python
from dataclasses import dataclass

@dataclass
class HelixReadCapability:
    max_results: int = 10
    allowed_nodes: tuple[str, ...] = (
        "Pattern", "Library", "Skill", "CodeNode", "Task", "Fact", "Document"
    )

@dataclass
class AgentDependencies:
    read_cap: HelixReadCapability
    phase: str
    task_description: str
    memory_context: str
```

#### 3.B.4 Agent Principal

**Ruta:** `mcp-tools/agent.py`

```python
from pydantic_ai import Agent, RunContext
from models import AgentDecision, AgentDependencies, HelixReadInput, HelixSearchResult

zyro_agent = Agent[AgentDependencies, AgentDecision](
    output_type=AgentDecision,
    system_prompt="""Eres un agente de análisis para el pipeline SDD.
Tu función es ANALIZAR y RECOMENDAR, nunca ejecutar.
REGLAS:
- NUNCA intentes escribir en HelixDB directamente
- NUNCA solicites tools de escritura
- Si detectas contradicción con memoria causal, inclúyelo en reasoning""",
)

@zyro_agent.tool
async def search_code(ctx: RunContext[AgentDependencies], input: HelixReadInput) -> list[HelixSearchResult]:
    """Buscar código en HelixDB por similitud semántica. Solo lectura."""

@zyro_agent.tool
async def search_skills(ctx: RunContext[AgentDependencies], input: HelixReadInput) -> list[HelixSearchResult]:
    """Buscar skills en el pool compartido. Solo lectura."""

@zyro_agent.tool
async def task_context(ctx: RunContext[AgentDependencies], task_id: int) -> str:
    """Obtener contexto de una tarea. Solo lectura."""

@zyro_agent.tool(requires_approval=True)
async def save_to_helix(ctx: RunContext[AgentDependencies], label: str, properties: dict) -> str:
    """NUNCA se ejecuta desde el agente. Marcador para el orquestador Go."""
    raise RuntimeError("save_to_helix debe ser interceptada por Go")
```

#### 3.B.5 Runner Refactorizado

**Ruta:** `mcp-tools/runner.py` (REFACTOR COMPLETO)

```python
import sys, json, asyncio
from agent import zyro_agent
from capabilities import AgentDependencies, HelixReadCapability

async def main():
    input_raw = sys.stdin.read()
    input_data = json.loads(input_raw)

    deps = AgentDependencies(
        read_cap=HelixReadCapability(),
        phase=input_data["phase"],
        task_description=input_data["task"],
        memory_context=input_data.get("memory", ""),
    )

    result = await zyro_agent.run(input_data["task"], deps=deps)

    print(result.data.model_dump_json(indent=2))

if __name__ == "__main__":
    asyncio.run(main())
```

#### 3.B.6 PydanticGraph State Machine (Opcional)

Para flujos que requieren múltiples pasos secuenciales:

```python
from pydantic_graph import Graph, End, BaseNode
from dataclasses import dataclass

@dataclass
class PlanState(BaseNode[AgentDependencies, None, AgentDecision]):
    async def run(self, ctx) -> "ExecuteState | ReviewState":
        # Planificar según fase
        ...

@dataclass
class ExecuteState(BaseNode[AgentDependencies, None, AgentDecision]):
    plan: dict
    async def run(self, ctx) -> "ReviewState":
        # Ejecutar plan
        ...

@dataclass
class ReviewState(BaseNode[AgentDependencies, None, AgentDecision]):
    async def run(self, ctx) -> "PersistState | ExecuteState":
        # Revisar, posiblemente iterar
        ...

@dataclass
class PersistState(BaseNode[AgentDependencies, None, AgentDecision]):
    async def run(self, ctx) -> End[AgentDecision]:
        return End(self.decision)
```

#### 3.B.7 Archivos Eliminados

- `mcp-tools/helix_write.py` → eliminado (writes solo por Go SDK)
- `mcp-tools/task_context.py` → reemplazado por tool del agente
- `mcp-tools/search_code.py` → reemplazado por tool del agente
- `mcp-tools/search_skills.py` → reemplazado por tool del agente
- `mcp-tools/helix_client.py` → mantenido solo para debug/testing interno

#### 3.B.8 Criterios de Éxito

```python
decision = await run_agent("Buscar patrones Factory Method en Go", phase="F0")
assert decision.action == "search"
assert len(decision.nodes) == 0
assert len(decision.reasoning) >= 10

# Go escribe solo tras validación
result = await process_decision(decision, client)
assert result["status"] == "ok"
```

---

### C. HelixDB SDK Go (Sprint 2)

**Objetivo:** Migrar de raw HTTP (`buildV3Envelope` + `doQuery`) al SDK oficial `github.com/helixdb/helix-db/sdks/go`.

#### 3.C.1 Arquitectura de Migración

```
ANTES:
  internal/db/helix/helix.go → buildV3Envelope() → doQuery() → parseSingleNode()

DESPUÉS:
  internal/db/helix/helix.go  → wrapper delgado sobre client.Exec()
  internal/db/helix/queries.go  → queries tipadas con DSL fluido
  internal/db/helix/search.go   → búsqueda híbrida (vector + BM25 + RRF)
  internal/db/helix/traverse.go → Repeat, Union, Choose, Coalesce
  internal/db/helix/embedding.go→ pipeline embeddings (OpenAI/Ollama)
  internal/db/helix/indexes.go  → CreateIndexIfNotExists
```

#### 3.C.2 Client Wrapper

**Ruta:** `internal/db/helix/helix.go` (REFACTOR)

```go
package helix

import (
    helix "github.com/helixdb/helix-db/sdks/go"
)

type Client struct {
    inner *helix.Client
    opts  ClientOptions
}

type ClientOptions struct {
    BaseURL    string        // default: "http://localhost:6969"
    Timeout    time.Duration // default: 30s
    MaxRetries int           // default: 3
}

func NewClient(opts ClientOptions) (*Client, error) {
    inner, err := helix.NewClient(opts.BaseURL)
    if err != nil {
        return nil, fmt.Errorf("helix new client: %w", err)
    }
    return &Client{inner: inner, opts: opts}, nil
}

func (c *Client) Exec(ctx context.Context, q helix.Request, out any) error {
    var lastErr error
    for i := 0; i <= c.opts.MaxRetries; i++ {
        err := c.inner.Exec(ctx, q, out)
        if err == nil { return nil }
        if errors.Is(err, helix.ErrConnectionFailed) {
            time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
            lastErr = err; continue
        }
        return err
    }
    return fmt.Errorf("helix exec after %d retries: %w", c.opts.MaxRetries, lastErr)
}

func (c *Client) HealthCheck(ctx context.Context) error {
    q := helix.ReadQuery("health")
    var out struct{ Result int }
    return c.Exec(ctx, q, &out)
}
```

#### 3.C.3 Queries Tipadas

**Ruta:** `internal/db/helix/queries.go`

```go
package helix

import helix "github.com/helixdb/helix-db/sdks/go"

type TaskRow struct {
    ID          int64     `json:"$id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Phase       string    `json:"phase"`
    Status      string    `json:"status"`
    CreatedAt   time.Time `json:"created_at"`
}

type CodeNodeRow struct {
    ID       int64  `json:"$id"`
    Path     string `json:"path"`
    Summary  string `json:"summary"`
    Language string `json:"language"`
    Hash     string `json:"hash"`
}

type FactRow struct {
    ID         int64   `json:"$id"`
    Type       string  `json:"type"`
    Content    string  `json:"content"`
    Salience   float64 `json:"salience"`
    Confidence float64 `json:"confidence"`
    Phase      string  `json:"phase"`
    IsActive   bool    `json:"is_active"`
}

func FindTask(name string) helix.Request {
    q := helix.ReadQuery("find_task")
    param := q.ParamString("name", name)
    return q.
        VarAs("task",
            helix.G().
                NWithLabel("Task").
                Where(helix.PredEq("name", param)).
                ValueMap("$id", "name", "description", "phase", "status", "created_at"),
        ).
        Returning("task")
}

func UpsertCodeNode(projectID int64, path, summary, language, hash string) helix.Request {
    q := helix.WriteQuery("upsert_code_node")
    pathParam := q.ParamString("path", path)
    summaryParam := q.ParamString("summary", summary)
    return q.
        VarAsIf("existing",
            helix.G().NWithLabel("CodeNode").
                Where(helix.PredEq("project_id", projectID),
                      helix.PredEq("path", pathParam)),
        ).
        VarAs("node",
            helix.IfElse(
                helix.Exists("existing"),
                helix.S().N("existing").Set("summary", summaryParam).Set("hash", hash),
                helix.S().Create("CodeNode", map[string]any{
                    "project_id": projectID, "path": path,
                    "summary": summary, "language": language, "hash": hash,
                }),
            ),
        ).
        Returning("node")
}
```

#### 3.C.4 Búsqueda Híbrida con RRF

**Ruta:** `internal/db/helix/search.go`

```go
package helix

type SearchResult struct {
    ID      int64   `json:"$id"`
    Label   string  `json:"label"`
    Content string  `json:"content"`
    Score   float64 `json:"score"`
    Source  string  `json:"source"`  // "vector" | "text" | "hybrid"
}

type HybridSearchOptions struct {
    MaxResults int     // default: 10
    RRFFusionK int     // default: 60
    NodeLabels []string
}

func HybridSearch(ctx context.Context, client *Client, query string,
    embedding []float32, opts HybridSearchOptions) ([]SearchResult, error) {

    // Ejecutar vector search y BM25 en paralelo
    type partialResult struct {
        results []SearchResult
        err     error
    }
    ch := make(chan partialResult, 2)
    var wg sync.WaitGroup
    wg.Add(2)

    go func() {
        defer wg.Done()
        r, e := vectorSearch(ctx, client, embedding, opts)
        ch <- partialResult{r, e}
    }()
    go func() {
        defer wg.Done()
        r, e := textBM25Search(ctx, client, query, opts)
        ch <- partialResult{r, e}
    }()
    wg.Wait(); close(ch)

    var vectorResults, textResults []SearchResult
    for pr := range ch {
        if pr.err != nil { return nil, pr.err }
        if len(pr.results) > 0 {
            if pr.results[0].Source == "vector" {
                vectorResults = pr.results
            } else {
                textResults = pr.results
            }
        }
    }

    return fuseRRF(vectorResults, textResults, opts.RRFFusionK, opts.MaxResults), nil
}

// fuseRRF: Reciprocal Rank Fusion
// RRFScore = sum(1 / (k + rank_i(d)))
func fuseRRF(vector, text []SearchResult, k, maxResults int) []SearchResult {
    // Fusiona rankings de ambas búsquedas
    // Normaliza scores a [0,1]
    // Retorna top maxResults
}
```

#### 3.C.5 Traversals Complejos

**Ruta:** `internal/db/helix/traverse.go`

```go
// DiscoverCrossProjectSkills: Skill ← REQUIRES_SKILL ← Project → USES_LIB → Library
func DiscoverCrossProjectSkills(ctx context.Context, client *Client, skillName string) ([]LibraryInfo, error) {
    q := helix.ReadQuery("cross_project_skills")
    name := q.ParamString("name", skillName)
    var out struct { Libraries []LibraryInfo `json:"libs"` }
    err := client.Exec(ctx,
        q.VarAs("libs",
            helix.G().NWithLabel("Skill").
                Where(helix.PredEq("name", name)).
                In("REQUIRES_SKILL").Out("USES_LIB").Dedup().
                ValueMap("$id", "name", "version"),
        ).Returning("libs"), &out)
    return out.Libraries, err
}

// TraverseCausalChain: navega cadena causal con Repeat
func TraverseCausalChain(ctx context.Context, client *Client, factID int64, maxDepth int) ([]FactWithPath, error) {
    q := helix.ReadQuery("causal_chain")
    var out struct { Chain []FactWithPath `json:"chain"` }
    err := client.Exec(ctx,
        q.VarAs("chain",
            helix.G().NWithLabel("Fact").
                Where(helix.PredEq("$id", factID)).
                Repeat(func(r *helix.RepeatBuilder) {
                    r.Out("CAUSED", "PRECEDES", "DERIVES_FROM").
                      MaxDepth(maxDepth).Dedup()
                }).
                ValueMap("$id", "type", "content", "phase", "confidence"),
        ).Returning("chain"), &out)
    return out.Chain, err
}
```

#### 3.C.6 Pipeline de Embeddings

**Ruta:** `internal/db/helix/embedding.go`

```go
type EmbeddingProvider string
const (
    ProviderOpenAI EmbeddingProvider = "openai"
    ProviderOllama EmbeddingProvider = "ollama"
)

type EmbeddingConfig struct {
    Provider  EmbeddingProvider
    Model     string   // "text-embedding-3-small" | "nomic-embed-text"
    Dims      int      // 1536 | 768
    APIKey    string
    BatchSize int      // default: 20
    CacheSize int      // default: 1000 (LRU)
}

type EmbeddingService struct {
    config EmbeddingConfig
    client *http.Client
    cache  *sync.Map
}

func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
    // Check cache → try Provider → store cache → return
}
```

#### 3.C.7 Criterios de Éxito

```go
client, _ := NewClient(ClientOptions{BaseURL: "http://localhost:6969"})
var out struct { Task []TaskRow }
err := client.Exec(ctx, FindTask("auth"), &out)
assert.NoError(t, err)
assert.Len(t, out.Task, 1)

results, err := HybridSearch(ctx, client, "jwt auth", embedding, HybridSearchOptions{MaxResults: 10})
assert.Greater(t, len(results), 0)
```

---

### D. Boundari por Fase (Sprint 3)

**Objetivo:** Implementar políticas Boundari por fase (F0-F4) con allow/deny/approval por tool, budgets, y auditoría JSONL.

#### 3.D.1 Mapa de Políticas

| Fase | Tools Permitidas | Tools Bloqueadas | Approval Requerido |
|------|-----------------|-------------------|-------------------|
| **F0** | read_file, search_code, list_directory, git_log, git_diff | write_file, delete_file, shell_exec, git_commit, git_push, network_request, npm_install | — |
| **F1** | read_file, search_code, grep_search, web_fetch, pypi_search, github_search | write_file, shell_exec, network_request | web_fetch (si URL externa) |
| **F2** | read_file, write_file, create_directory, shell_exec (comandos seguros), git_commit | delete_file, git_push, npm_install | Toda escritura, shell_exec (si no seguro) |
| **F3** | read_file, write_file, create_directory, shell_exec, npm_install, pip_install, git_diff | delete_file, git_commit, git_push | write_file (si no es src/), shell_exec, pip_install |
| **F4** | read_file, search_code, git_log, git_diff, write_file (con approval) | shell_exec, delete_file, npm_install, pip_install, network_request | write_file, git_commit, git_push |

#### 3.D.2 Budgets por Fase

| Fase | max_tool_calls | max_runtime_seconds | max_cost_usd |
|------|---------------|---------------------|--------------|
| F0 | 30 | 300 | $0.10 |
| F1 | 40 | 600 | $0.25 |
| F2 | 50 | 600 | $0.35 |
| F3 | 150 | 1800 | $1.00 |
| F4 | 30 | 300 | $0.10 |

#### 3.D.3 Ejemplo de Política YAML

**Ruta:** `boundari/phase0-boundari.yaml`

```yaml
version: "1.0"
phase: "F0"
description: "Discovery phase — read only, no execution"
budget:
  max_tool_calls: 30
  max_runtime_seconds: 300
  max_cost_usd: 0.10
tools:
  read_file:        { allow: true, deny: false, approval: false }
  search_code:      { allow: true, deny: false, approval: false }
  list_directory:   { allow: true, deny: false, approval: false }
  git_log:          { allow: true, deny: false, approval: false }
  git_diff:         { allow: true, deny: false, approval: false }
  write_file:       { allow: false, deny: true, approval: false }
  delete_file:      { allow: false, deny: true, approval: false }
  shell_exec:       { allow: false, deny: true, approval: false }
  git_commit:       { allow: false, deny: true, approval: false }
  git_push:         { allow: false, deny: true, approval: false }
  network_request:  { allow: false, deny: true, approval: false }
  npm_install:      { allow: false, deny: true, approval: false }
```

#### 3.D.4 Loader en Go

**Ruta:** `internal/boundari/loader.go`

```go
type Policy struct {
    Version     string              `yaml:"version"`
    Phase       string              `yaml:"phase"`
    Budget      Budget              `yaml:"budget"`
    Tools       map[string]ToolPolicy `yaml:"tools"`
}

type ToolPolicy struct {
    Allow    bool            `yaml:"allow"`
    Deny     bool            `yaml:"deny"`
    Approval *ApprovalPolicy `yaml:"approval,omitempty"`
}

func LoadPolicy(phase string, searchDirs ...string) (*Policy, error) {
    for _, dir := range searchDirs {
        path := filepath.Join(dir, fmt.Sprintf("phase%s-boundari.yaml", phase))
        data, err := os.ReadFile(path)
        if err == nil {
            var p Policy
            yaml.Unmarshal(data, &p)
            return &p, nil
        }
    }
    return nil, fmt.Errorf("policy for phase %s not found", phase)
}
```

#### 3.D.5 Enforcer en Go

**Ruta:** `internal/boundari/enforcer.go`

```go
type Enforcer struct {
    policy  *Policy
    usage   BudgetUsage
    startAt time.Time
}

func (e *Enforcer) CheckTool(toolName string, args map[string]any) EnforcementResult {
    // 1. Budget check (tool_calls, runtime)
    // 2. Deny check (prioridad sobre allow)
    // 3. Allow check
    // 4. Approval condition evaluation
    ...
}
```

#### 3.D.6 Wrapper Python

**Ruta:** `mcp-tools/boundari_wrapper.py`

```python
class BoundariWrapper:
    def __init__(self, phase: str, policies_dir: str = None):
        self.phase = phase
        self.policy = self._load_policy()
        self.tool_calls = 0
        self.audit_log = []

    def wrap_tool(self, tool_name: str, tool_func, raise_on_denied=True):
        async def wrapped(*args, **kwargs):
            # Budget check
            # Policy check (allow/deny/approval)
            # Execute or raise
            ...
        return wrapped
```

#### 3.D.7 Criterios de Éxito

```bash
zyro run --phase F0
# write_file intentado → Decision(allowed=False, reason="denied")

zyro run --phase F3
# write_file("src/app.ts") → Decision(allowed=True)  # auto-allow
# write_file("/etc/hosts") → Decision(allowed=True, need_approval=True)

ls ~/.zyro/audit/
# → phase0-20260615T120000.jsonl
```

---

### E. Memoria Causal (Sprint 4)

**Objetivo:** Sistema de memoria causal sobre HelixDB con 6 tipos de Fact + 7 aristas causales, extracción LLM post-fase e inyección pre-fase.

#### 3.E.1 Flujo Completo de Memoria

```
Pre-Fase:
  1. memory_hook.PrePhase(phase, task)
  2. EngramStore.RecallMemories(query, phase, limit=10)
  3. Formatear como "MEMORIA CAUSAL:" → inyectar en prompt del agente

Fase:
  4. Agente recibe contexto con memoria incluida
  5. Agente ejecuta con tools envueltas (Boundari)
  6. Agente retorna AgentDecision → Go valida

Post-Fase:
  7. memory_hook.PostPhase(phase, conversationLog)
  8. fact_extractor.py parsea log → extrae hechos con embeddings
  9. EngramStore.SaveFacts(facts)
  10. EngramStore.ResolveContradictions()
  11. EngramStore.ReinforceSalience(accessedFactIDs)
```

#### 3.E.2 Schema de Facts

**Ruta:** `internal/memory/schema.go`

```go
type FactType string
const (
    FactDecision    FactType = "decision"
    FactError       FactType = "error"
    FactPreference  FactType = "preference"
    FactPattern     FactType = "pattern"
    FactDependency  FactType = "dependency"
    FactObservation FactType = "observation"
)

type CausalEdgeType string
const (
    EdgeCaused      CausalEdgeType = "CAUSED"
    EdgePrecedes    CausalEdgeType = "PRECEDES"
    EdgeContradicts CausalEdgeType = "CONTRADICTS"
    EdgeSupports    CausalEdgeType = "SUPPORTS"
    EdgeRequires    CausalEdgeType = "REQUIRES"
    EdgeDerivesFrom CausalEdgeType = "DERIVES_FROM"
    EdgeReferences  CausalEdgeType = "REFERENCES"
)

type Fact struct {
    ID             int64     `json:"$id"`
    Type           FactType  `json:"type"`
    Content        string    `json:"content"`
    Embedding      []float32 `json:"embedding,omitempty"`
    Salience       float64   `json:"salience"`       // 0.0-1.0
    Confidence     float64   `json:"confidence"`     // 0.0-1.0
    Source         string    `json:"source"`         // "agent:F0" | "extractor:llm"
    Phase          string    `json:"phase"`
    CreatedAt      time.Time `json:"created_at"`
    LastAccessedAt time.Time `json:"last_accessed_at"`
    AccessCount    int64     `json:"access_count"`
    DecayRate      float64   `json:"decay_rate"`     // default 0.05
    ExpiresAt      time.Time `json:"expires_at"`
    IsActive       bool      `json:"is_active"`
    ProjectID      string    `json:"project_id"`
    Metadata       map[string]any `json:"metadata,omitempty"`
}
```

#### 3.E.3 EngramStore Interface

**Ruta:** `internal/memory/memory.go`

```go
type EngramStore interface {
    SaveFact(ctx context.Context, fact *Fact) (int64, error)
    SaveFactsBatch(ctx context.Context, facts []*Fact) ([]int64, error)
    AddCausalEdge(ctx context.Context, edge *CausalEdge) error
    RecallMemories(ctx context.Context, query string, embedding []float32, opts RecallOpts) ([]*Fact, error)
    DetectContradictions(ctx context.Context, projectID string) ([]ContradictionPair, error)
    ResolveContradiction(ctx context.Context, pair ContradictionPair, strategy ContradictionStrategy) error
    ReinforceSalience(ctx context.Context, factIDs []int64) error
    DecayAndRefresh(ctx context.Context) error
}
```

#### 3.E.4 Store Implementation

**Ruta:** `internal/memory/store.go`

```go
type HelixEngramStore struct {
    client          *db.Client
    embeddingSvc    *db.EmbeddingService
    defaultDecayRate float64
}

func (s *HelixEngramStore) SaveFact(ctx context.Context, fact *Fact) (int64, error) {
    if len(fact.Embedding) == 0 {
        emb, err := s.embeddingSvc.Embed(ctx, fact.Content)
        if err != nil { return 0, err }
        fact.Embedding = emb
    }
    q := db.CreateFact(db.FactRow{
        Type: string(fact.Type), Content: fact.Content,
        Salience: fact.Salience, Confidence: fact.Confidence,
        Source: fact.Source, Phase: fact.Phase,
    }, fact.Embedding)
    var out struct { Fact struct { ID int64 `json:"$id"` } }
    if err := s.client.Exec(ctx, q, &out); err != nil {
        return 0, fmt.Errorf("save fact: %w", err)
    }
    return out.Fact.ID, nil
}
```

#### 3.E.5 Sistema de Embeddings (Integración con zyro setup)

**Objetivo:** Sistema de generación de embeddings para búsqueda semántica (vector ANN), detección de contradicciones (cosine similarity), y ranking de relevancia (hybrid search vector + BM25 + RRF).

**Modelo local default:** `mxbai-embed-large` (Ollama, 334M params, 768 dims, CPU/GPU) — mejor calidad/speed en CPU según MTEB.

**Fallback API gratuito:** Scaleway (`qwen3-embedding-8b`, 1M tokens gratis, 8B params).

**Fallback terciario:** GitHub Models (OpenAI Embedding 3 si tiene Copilot), Cohere, o NVIDIA NIM.

**Degradación graceful:** Si no hay embeddings disponibles → solo BM25 text search. El sistema funciona completamente sin vectores.

**Embedding Harness:** MCP server separado (`zyro-embedding-harness`) que corre en background como herramienta MCP. No interrumpe el agente principal.

**Cache:** LRU en disco en `~/.zyro/embedding-cache/` (SQLite o JSONL). Embeddings duplicados no se regeneran.

**Pipeline de prioridad (degradación graceful):**
```
┌─ ¿Ollama + mxbai-embed-large disponible?
│   ✅ → Usar (local, CPU/GPU, 768 dims)
├─ ¿No? → ¿Scaleway API configurado?
│   ✅ → Usar (qwen3-embedding-8b, 1M tokens gratis)
├─ ¿No? → ¿GitHub Models / Cohere configurado?
│   ✅ → Fallback terciario
└─ ¿Nada disponible?
    → BM25 puro (sin vectores, sin errores)
```

**Instalación interactiva:** `zyro setup` pregunta si instalar Ollama + modelo, detecta GPU (`nvidia-smi` / `rocminfo`), instala Ollama y hace pull del modelo. Luego pregunta si configurar fallback API (Scaleway, GitHub Models, Cohere). Guarda configuración en `~/.zyro/config.yaml`.

**Tools del Embedding Harness:**
- `embed(text) → vector`: genera embedding para un texto.
- `embed_batch(texts) → vectors`: genera embeddings para múltiples textos en batch.
- `status() → dict`: reporta proveedor activo, modelo, tamaño de caché.

#### 3.E.6 Resolución de Contradicciones

**Ruta:** `internal/memory/contradictions.go`

```go
type ContradictionStrategy string
const (
    StrategyNewestWins        ContradictionStrategy = "newest_wins"
    StrategyHighestConfidence ContradictionStrategy = "highest_confidence"
    StrategyKeepBoth          ContradictionStrategy = "keep_both"
)

func (s *HelixEngramStore) ResolveContradiction(
    ctx context.Context, pair ContradictionPair, strategy ContradictionStrategy,
) error {
    switch strategy {
    case StrategyNewestWins:
        if pair.FactA.CreatedAt.After(pair.FactB.CreatedAt) {
            return s.deactivateFact(ctx, pair.FactB.ID, "superseded by newer")
        }
        return s.deactivateFact(ctx, pair.FactA.ID, "superseded by newer")
    case StrategyHighestConfidence:
        if pair.FactA.Confidence >= pair.FactB.Confidence {
            return s.deactivateFact(ctx, pair.FactB.ID, "superseded by higher confidence")
        }
        return s.deactivateFact(ctx, pair.FactA.ID, "superseded by higher confidence")
    case StrategyKeepBoth:
        return nil
    }
    return nil
}
```

#### 3.E.7 Curva de Olvido (Ebbinghaus)

**Ruta:** `internal/memory/decay.go`

```go
// salience(t) = salience_0 * e^(-decay_rate * days_since_access)
func (s *HelixEngramStore) DecayAndRefresh(ctx context.Context) error {
    // Query todos los Facts activos
    // Calcular nuevo salience con fórmula de Ebbinghaus
    // Si salience < 0.15 → marcar como stale
    // Si expires_at < now → marcar como inactivo
}

func (s *HelixEngramStore) ReinforceSalience(ctx context.Context, factIDs []int64) error {
    // salience += 0.3 * (1 - salience)
    // = 0.7 * salience + 0.3
    // Incrementar access_count, actualizar last_accessed_at
}
```

#### 3.E.8 Extractor de Hechos (Python)

**Ruta:** `agents/fact_extractor.py`

```python
#!/usr/bin/env python3
"""
Extractor de hechos: analiza logs de conversación y extrae Facts estructurados.

Uso: python fact_extractor.py --input <log.json> --phase F1
"""

import json, sys, argparse, re

FACT_PATTERNS = {
    "decision": [r"(?:vamos a usar|usamos|decidimos) (.+?)(?:\.|$)",
                 r"(?:decisión|decision):?\s*(.+?)(?:\.|$)"],
    "error":    [r"(?:error|bug|problema|fallo):?\s*(.+?)(?:\.|$)"],
    "preference":[r"(?:prefiero|preferimos|mejor usar) (.+?)(?:\.|$)"],
    "pattern":  [r"(?:patrón|pattern|arquitectura)\s*(.+?)(?:\.|$)"],
    "dependency":[r"(?:dependemos de|necesitamos|requiere) (.+?)(?:\.|$)"],
    "observation":[r"(?:observo|noto|veo que|detecto) (.+?)(?:\.|$)"],
}

def extract_facts(log_text: str, phase: str) -> list[dict]:
    facts = []
    for fact_type, patterns in FACT_PATTERNS.items():
        for pattern in patterns:
            for match in re.finditer(pattern, log_text, re.IGNORECASE):
                content = match.group(1).strip()
                if len(content) < 10:
                    continue
                facts.append({
                    "type": fact_type, "content": content,
                    "salience": 0.7, "confidence": 0.8,
                    "source": "extractor:llm", "phase": phase,
                    "decay_rate": 0.05,
                })
    return facts

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True)
    parser.add_argument("--phase", required=True, choices=["F0","F1","F2","F3","F4"])
    args = parser.parse_args()
    with open(args.input) as f:
        log_data = json.load(f)
    log_text = log_data.get("conversation", "")
    facts = extract_facts(log_text, args.phase)
    print(json.dumps({"facts": facts}, indent=2))

if __name__ == "__main__":
    main()
```

#### 3.E.9 Hooks en el Scheduler

**Ruta:** `internal/scheduler/memory_hook.go`

```go
type MemoryHooks struct {
    store memory.EngramStore
}

func (h *MemoryHooks) PrePhase(ctx context.Context, phase string, taskDesc string) (string, error) {
    facts, err := h.store.RecallMemories(ctx, taskDesc, nil, memory.RecallOpts{
        MaxResults: 10, MinSalience: 0.2,
    })
    if err != nil { return "", err }
    if len(facts) == 0 { return "", nil }
    return formatMemoryForPrompt(facts), nil
}

func (h *MemoryHooks) PostPhase(ctx context.Context, phase string, logData []byte) error {
    // Guardar log temporal → ejecutar fact_extractor.py → parsear output
    // → SaveFactsBatch → DetectContradictions → ResolveContradiction
}
```

#### 3.E.10 Criterios de Éxito

```bash
zyro run --phase F0  # se ejecuta, extrae hechos
zyro run --phase F1  # el prompt incluye decisiones de F0

zyro context --memory
# → "12 hechos activos: 3 decisiones, 2 errores, 4 preferencias, ..."

# Contradicción: F0 dice "usar GORM", F1 dice "usar SQLC"
# → Edge CONTRADICTS creado, newest_wins: prevalece "SQLC"

# Ebbinghaus: fact con salience 0.7, decay 0.05, 30d sin acceso
# → salience = 0.7 * e^(-0.05*30) = 0.156 < 0.15 → stale
```

---

### F. OpenCode + Boomerang (Sprint 5)

**Objetivo:** Integrar todo con OpenCode via plugins, protocolo Boomerang de 6 pasos, approval gates nativos.

#### 3.F.1 Protocolo Boomerang

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CICLO BOOMERANG (por fase)                        │
│                                                                     │
│  ┌─────────┐    ┌─────────┐    ┌───────────┐    ┌──────┐          │
│  │ MEMORY  │───►│  THINK  │───►│ DELEGATE  │───►│ GIT  │          │
│  │ Consulta │    │ Planea  │    │ Reparte a │    │ Verif│          │
│  │ memoria  │    │ el DAG  │    │ subagentes│    │ estado│         │
│  │ causal   │    │ de tareas│   │            │    │ repo  │         │
│  └─────────┘    └─────────┘    └───────────┘    └──────┘          │
│                                                      │              │
│  ┌─────────┐    ┌─────────┐                          │              │
│  │  SAVE   │◄───│ QUALITY │◄─────────────────────────┘              │
│  │ Guarda  │    │ Linters,│                                         │
│  │ decisión│    │ tests   │                                         │
│  │ en DB   │    │         │                                         │
│  └─────────┘    └─────────┘                                         │
└─────────────────────────────────────────────────────────────────────┘
```

Si QUALITY falla, vuelve a DELEGATE (máx 3 iteraciones).

#### 3.F.2 Orquestador

**Ruta:** `internal/boomerang/orchestrator.go`

```go
type BoomerangOrchestrator struct {
    memoryStore memory.EngramStore
    gitChecker  GitChecker
    qualityGate QualityGate
    delegateSvc DelegateService
}

func (o *BoomerangOrchestrator) Run(ctx context.Context, config PhaseConfig) (*PhaseResult, error) {
    // 1. MEMORY — consultar memoria causal
    memory, _ := o.memoryStore.RecallMemories(...)

    // 2. THINK — planificar DAG de tareas
    dag := planDAG(config.Phase, memory)

    // 3. DELEGATE — lanzar subagentes
    result, _ := o.delegateSvc.Delegate(ctx, config.Phase, TaskSpec{...})

    // 4. GIT — verificar estado del repo
    status, _ := o.gitChecker.Status(ctx)

    // 5. QUALITY + RETRY LOOP (max 3 iteraciones)
    for iteration := 0; iteration < maxIter; iteration++ {
        ok, issues, _ := o.qualityGate.Check(ctx, config.Phase)
        if ok { break }
        // redelegate con feedback de issues
    }

    // 6. SAVE — guardar decisiones y hechos
    // Extraer hechos → SaveFactsBatch
    return &PhaseResult{Success: true, Iterations: iteration, ...}, nil
}
```

#### 3.F.3 Plugin de Bridge

**Ruta de instalación:** `~/.config/opencode/plugins/zyrocli.ts`

```typescript
import { createClaudeBridge } from "@sjawhar/opencode-claude-bridge";
import path from "path";

export default createClaudeBridge({
  sources: [
    { dir: path.join(os.homedir(), ".config/zyrocli/skills"), namespace: "zyro" },
  ],
  claudePlugins: true,
});
```

Skills como `.md` en `~/.config/zyrocli/skills/`:

```markdown
---
name: zyro-approval-gate
description: "Gate de aprobación entre fases SDD"
mcp:
  helix:
    command: ["uv", "run", "--directory", "~/.config/zyrocli/mcp-tools", "agent.py"]
---

# Zyro Approval Gate

Actúa como gate de aprobación. Revisa el estado actual de la fase.
Resume lo completado, los archivos tocados, y los nodos creados en HelixDB.
Pregunta al humano: '¿Aprobás esta fase para continuar?'
```

#### 3.F.4 Lazy-Loading MCP

```jsonc
{
  "plugin": [
    "@sjawhar/opencode-claude-bridge",
    "opencode-lazy-loader",
    "opencode-multiagent"
  ],
  "mcp_servers": {
    "helix-integration": {
      "command": ["uv", "run", "--directory", "~/.config/zyrocli/mcp-tools", "agent.py"],
      "lazy": true,
      "env": { "HELIX_DB_URL": "http://localhost:6969", "ZYRO_PHASE": "F0" }
    }
  }
}
```

#### 3.F.5 Approval Gates por Subagentes

**Ruta:** `internal/scheduler/approval.go` (REFACTOR)

```go
func ApprovalGate(ctx context.Context, phase string, summary string) (bool, error) {
    cmd := exec.CommandContext(ctx, "opencode", "subagent", "zyro-approval-gate",
        "--param", fmt.Sprintf("phase=%s", phase),
        "--param", fmt.Sprintf("summary=%s", summary),
    )
    output, err := cmd.Output()
    if err != nil { return false, err }
    var result struct { Approved bool `json:"approved"` }
    json.Unmarshal(output, &result)
    return result.Approved, nil
}
```

#### 3.F.6 Comando `/zyro-approve`

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

#### 3.F.7 Criterios de Éxito

```bash
zyro run --phase F0
# → [1/6 MEMORY] Consulta memoria causal... 0 hechos (primera ejecución)
# → [2/6 THINK] Planea DAG: patterns + libraries + skills en paralelo
# → [3/6 DELEGATE] Lanza 3 subagentes en paralelo
# → [4/6 GIT] Verifica estado del repo... limpio
# → [5/6 QUALITY] Verifica que nodos se crearon en HelixDB... OK
# → [6/6 SAVE] Guarda decisiones en HelixDB... 5 hechos guardados

ls ~/.config/opencode/skills/
# → zyro-orchestrator  zyro-sdd-apply  zyro-sdd-verify  zyro-approval-gate

# En chat de OpenCode: /zyro-approve
# → Subagente gate revisa estado, pregunta al humano
# → Humano responde en el chat
```

---

## 4. Restricciones Técnicas

### 4.1 Dependencias Externas

| Dependencia | Versión Mínima | Propósito | Riesgo |
|------------|---------------|-----------|--------|
| Go | 1.21+ | Orquestador | Bajo |
| Python | 3.11+ | Agente PydanticAI | Bajo |
| `uv` | 0.4+ | Gestor Python | Medio |
| `pydantic-ai` | 1.95+ | Framework de agente | Bajo |
| `pydantic-graph` | última | State machines | Medio |
| `github.com/helixdb/helix-db/sdks/go` | v3+ | SDK oficial HelixDB | Medio |
| `boundari` | 0.1.0 (pinned) | Políticas de seguridad | **Alto** — alpha |
| `@sjhawar/opencode-claude-bridge` | última | Bridge de skills | Medio |
| `opencode-lazy-loader` | última | Lazy-loading MCP | Medio |
| `opencode-multiagent` | última | Subagentes paralelos | Medio |

### 4.2 Restricciones de Arquitectura

1. **Go es el único que escribe en HelixDB.** El agente Python NUNCA tiene acceso directo a tools de escritura.
2. **OpenCode lidera el pipeline. ZyroCLI configura.** Go no reemplaza a OpenCode como runtime.
3. **Boundari es un wrapper, no un reemplazo.** Si Boundari falla, Go debe tener políticas hardcodeadas de fallback.
4. **No usar Engram MCP Server.** Sistema de memoria causal es custom sobre HelixDB.
5. **Cada sprint es independientemente desplegable.** Con stubs donde haya dependencias.
6. **Embeddings son app-side.** HelixDB v3 no tiene `Embed()` nativa.
7. **Boomerang es un protocolo, no un framework.** Es código Go que orquesta el flujo.

### 4.3 Restricciones de Rendimiento

| Métrica | Límite | Cómo se asegura |
|---------|--------|----------------|
| Setup inicial | < 60s | Instalación paralela, binarios pre-compilados |
| Primera query HelixDB | < 100ms | localhost, sub-milisegundo |
| Búsqueda híbrida | < 500ms | Paralelización vector + BM25, cache | 
| Extracción hechos | < 30s | LLM local, batch async |
| Lazy-loading MCP | < 200ms | Bajo demanda, no en startup |
| Memoria por fact | ~500 bytes | Límite 10000 facts/proyecto |

### 4.4 Restricciones de Seguridad

1. El agente Python corre como subproceso de Go — stdin/stdout, sin puerto abierto.
2. Boundari se aplica en dos capas: Python `wrap_tool()` + Go `enforcer.go`.
3. API keys van en `~/.zyro/config.yaml` o entorno del subproceso, no en el agente.
4. Cada fase tiene tool calls y budget de costo — al excederse se aborta.
5. Auditoría JSONL inmutable por cada llamada a tool.

---

## 5. Dependencias Entre Sprints

### 5.1 Mapa de Dependencias

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

### 5.2 Justificación

| Dependencia | Razón |
|------------|-------|
| S1 → S0 | El agente Python necesita el CLI Go compilado para recibir contexto |
| S2 → S0 | HelixDB SDK Go necesita HelixDB corriendo (verificado por doctor) |
| S3 → S1 | Boundari envuelve tools del agente PydanticAI |
| S4 → S2 | Memoria causal usa traversals del SDK (Repeat, Union, Choose) + búsqueda híbrida |
| S5 → S3 | Boomerang necesita Boundari para control de herramientas |
| S5 → S4 | Boomerang necesita memoria causal para paso Memory del ciclo |

**Sprint 1 y Sprint 2 son paralelos** — el harness del agente y la persistencia profunda son independientes.

### 5.3 Resumen de Archivos

| Sprint | Archivos Nuevos | Modificados | Líneas |
|--------|----------------|-------------|--------|
| Sprint 0 | 6 | 2 | ~800 |
| Sprint 1 | 4 | 6 | ~600 |
| Sprint 2 | 5 | 4 | ~700 |
| Sprint 3 | 8 | 1 | ~400 |
| Sprint 4 | 8 | 2 | ~900 |
| Sprint 5 | 8 | 3 | ~1000 |
| **Total** | **39** | **18** | **~4400** |

---

## 6. Criterios de Aceptación

### 6.1 Prueba: Máquina limpia (Ubuntu 22.04)

```bash
which go     # → not found
which uv     # → not found
which helix  # → not found

curl -sSL https://install.zyro.dev | bash
# ✅ uv instalado desde astral.sh
# ✅ HelixDB instalado y corriendo (localhost:6969)
# ✅ zyro compilado en ~/.local/bin/
# ✅ ~/.zyro/config.yaml generado

zyro doctor
# → ✅ go 1.26.0  ✅ uv 0.6.0  ✅ helix running  ⚠️ docker not found  ✅ git 2.34.1
```

### 6.2 Prueba: Pipeline completo F0→F4

```bash
zyro init docs/examples/test-handoff.yaml
# → Estructura creada, nodos Developer + Project en HelixDB

zyro run --phase F0
# → Boomerang: 3 subagentes paralelos, approval gate en chat

zyro run --phase F1
# → Memoria causal inyecta decisiones de F0 en prompt
```

### 6.3 Prueba: Seguridad

```bash
zyro run --phase F0
# write_file("test.txt") → Decision(allowed=False)

zyro run --phase F3
# write_file("src/app.ts") → auto-allow
# write_file("/etc/hosts") → "approval_required"

# Después de 30 tool calls en F0 → "budget_exceeded"
```

### 6.4 Prueba: Memoria causal

```bash
zyro run --phase F0  # extrae hechos
zyro context --memory
# → 5 hechos: 1 decision, 1 preference, 1 pattern, 1 error, 1 dependency

zyro run --phase F1  # prompt incluye contexto automático
```

### 6.5 Prueba: Ecosistema OpenCode

```bash
ls ~/.config/opencode/skills/
# → zyro-orchestrator/  zyro-sdd-apply/  zyro-sdd-verify/  ...

# En chat: /zyro-approve → subagente revisa y pregunta al humano
```

---

## 7. Riesgos y Mitigaciones

| # | Riesgo | Impacto | Prob. | Mitigación |
|---|--------|---------|-------|------------|
| 1 | `uv` no disponible en ARM64 | Alto | Baja | Fallback a `pip` + venv clásico |
| 2 | HelixDB SDK Go cambia API | Medio | Media | Version pinning + tests de integración |
| 3 | Boundari v0.1.0 alpha — cambios API | Alto | Media | Congelar `boundari==0.1.0` + fallback Go hardcodeado |
| 4 | Embeddings locales lentos (CPU) | Medio | Alta | Cache LRU 1000 entradas + batch async + fallback OpenAI |
| 5 | OpenCode plugin system inestable | Alto | Media | Mantener escritura directa de opencode.jsonc como fallback |
| 6 | Agente Python se cuelga/timeout | Alto | Media | Timeout 120s en Go + kill/restart |
| 7 | Extractor LLM produce falsos positivos | Medio | Alta | Umbral confidence configurable (default 0.6) |
| 8 | Grafo causal crece sin control | Bajo | Media | Límite 10000 facts/proyecto, decaimiento diario, archivo a 90 días |
| 9 | Contradicciones no detectadas | Medio | Media | Embedding + reglas explícitas + humano puede marcar manualmente |
| 10 | OpenCode sin paralelización nativa | Medio | Alta | plugin multiagent + orquestador Go lanza procesos paralelos |

---

## 8. Glosario

| Término | Definición |
|---------|-----------|
| **Agent-as-Validator** | Patrón donde el agente LLM retorna un PydanticModel validado y el orquestador (Go) ejecuta. |
| **Boomerang** | Protocolo de 6 pasos (Memory→Think→Delegate→Git→Quality→Save) por fase del pipeline SDD. |
| **Boundari** | Librería Python de políticas-as-código para control granular de tools por fase. |
| **Engram (Zyro)** | Sistema de memoria causal custom sobre HelixDB usando nodos Fact + aristas causales. NO es `keggan-std/Engram`. |
| **HelixDB SDK Go** | SDK oficial `github.com/helixdb/helix-db/sdks/go` con DSL fluido para queries. |
| **Lazy-loading MCP** | Plugin OpenCode que carga MCP servers solo cuando se usan, no al arranque. |
| **claude-bridge** | Plugin que traduce skills Claude Code a formato OpenCode. |
| **RRF** | Reciprocal Rank Fusion — algoritmo para fusionar rankings vectoriales y BM25. |
| **Fact** | Nodo atómico de memoria causal. 6 tipos: decision, error, preference, pattern, dependency, observation. |
| **Salience** | Métrica de importancia de un hecho (0.0-1.0). Decae según curva de Ebbinghaus. |
| **PydanticGraph** | State machine formal de PydanticAI con nodos (PlanState→ExecuteState→ReviewState→PersistState). |
| **PydanticAI** | Framework Python para agentes LLM con output tipado y validado por Pydantic. |
