# Diseño Técnico — ZyroAgentCLI v2

> **Fecha:** 2026-06-15  
> **Basado en:** docs/spec-zyrov2.md + 5 investigaciones Fase 0 + feedback + roadmap  
> **Versión:** 2.0.0-design  
> **Sprints:** S0–S5  
> **Archivos diseñados:** 57 nuevos, 18 modificados  
> **Funciones/métodos diseñados:** ~210  

---

## 1. Arquitectura General

### 1.1 Diagrama de Componentes


```
┌──────────────────────────────────────────────────────────────────────────────┐
│                               TERMINAL                                       │
│  zyro setup | zyro doctor | zyro init | zyro run --phase F{N}               │
└────────────────────────────────┬─────────────────────────────────────────────┘
                                 │
┌────────────────────────────────▼─────────────────────────────────────────────┐
│                          ZYROCLI (Go 1.21+)                                    │
│                                                                                │
│  ┌───────────────────┐  ┌──────────────────┐  ┌──────────────────────────┐    │
│  │ cmd/zyrocli/       │  │ internal/setup/  │  │ internal/db/helix/       │    │
│  │  ├── main.go       │  │  ├── check.go    │  │  ├── helix.go (wrapper)  │    │
│  │  ├── cmd/setup.go  │  │  ├── install.go  │  │  ├── queries.go (tipadas)│    │
│  │  ├── cmd/doctor.go │  │  ├── config.go   │  │  ├── search.go (RRF)     │    │
│  │  └── cmd/run.go   │  │  └── doctor.go   │  │  ├── traverse.go          │    │
│  └───────────────────┘  └──────────────────┘  │  ├── embedding.go         │    │
│  ┌───────────────────┐  ┌──────────────────┐  │  └── indexes.go           │    │
│  │ internal/scheduler/ │  │ internal/memory/ │  └──────────────────────────┘    │
│  │  ├── scheduler.go  │  │  ├── memory.go   │  ┌──────────────────────────┐    │
│  │  ├── phase.go      │  │  ├── schema.go   │  │ internal/boundari/        │    │
│  │  ├── approval.go   │  │  ├── store.go    │  │  ├── loader.go           │    │
│  │  ├── memory_hook   │  │  ├── recall.go   │  │  └── enforcer.go         │    │
│  │  └── runner.go     │  │  ├── contra.go   │  └──────────────────────────┘    │
│  └───────────────────┘  │  └── decay.go     │  ┌──────────────────────────┐    │
│  ┌───────────────────┐  └──────────────────┘  │ internal/boomerang/        │    │
│  │ internal/opencode/  │                       │  ├── orchestrator.go      │    │
│  │  └── config.go     │                       │  ├── think.go             │    │
│  └───────────────────┘                       │  ├── delegate.go           │    │
│                                                │  ├── git.go               │    │
│                                                │  ├── quality.go           │    │
│                                                │  └── save.go              │    │
│                                                └──────────────────────────┘    │
└────────────────────────────────┬─────────────────────────────────────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │ stdin/json        │ Go SDK writes     │
              │ (Agent-as-Val.)   │ + HTTP reads      │
              ▼                   ▼                   ▼
┌──────────────────────┐ ┌──────────────┐ ┌─────────────────────────┐
│  Agente Python        │ │  OpenCode    │ │  HelixDB (Rust v3)     │
│  (PydanticAI)         │ │  (Runtime)   │ │                        │
│                       │ │              │ │  Graph + Vector + Text │
│  Agent-as-Validator   │ │  Plugins:    │ │  + Memoria Causal      │
│  → AgentDecision      │ │  • bridge    │ │  (Facts + 7 edges)     │
│                       │ │  • lazy-load │ │                        │
│  Tools READ-ONLY      │ │  • multiagt  │ │                        │
│  Boundari wrapper     │ │  13 subags   │ │                        │
└──────────────────────┘ └──────────────┘ └─────────────────────────┘
```

### 1.2 Contrato Go-Python (stdin/stdout)

```
INPUT (Go → Python, stdin):
{
  "protocol": "zyro-agent-v2",
  "version": "2.0.0",
  "request_id": "uuid-v7",
  "phase": "F0",
  "task": "Analizar patrones de diseno en el proyecto",
  "memory_context": "--- MEMORIA CAUSAL ---...",
  "boundari_phase": "F0",
  "timeout_seconds": 120
}

OUTPUT (Python → Go, stdout):
{
  "protocol": "zyro-agent-v2",
  "version": "2.0.0",
  "request_id": "uuid-v7",
  "action": "search",
  "reasoning": "Se identificaron 3 patrones Factory...",
  "nodes": [
    {"label": "Pattern", "properties": {...},
     "project_id": 1005, "requires_approval": false}
  ],
  "requires_approval": false,
  "metadata": {"tokens_used": 4500, "model": "gpt-5.2"}
}
```

---

## 2. Sprint 0: CLI Inteligente

### 2.1 Archivos
```

cmd/zyrocli/
├── main.go                   (MOD)  ← +SetupCmd + DoctorCmd
└── cmd/
    ├── setup.go              (NEW)  ← zyro setup
    └── doctor.go             (NEW)  ← zyro doctor

internal/setup/
├── check.go                  (NEW)  ← DependencyChecker, Checker interface
├── install.go                (NEW)  ← Installer (uv, helix, go, venv)
├── config.go                 (NEW)  ← Config struct, ~/.zyro/config.yaml
└── doctor.go                 (NEW)  ← Doctor (Fix, Report)

scripts/install.sh            (MOD)  ← simplificar a curl→zyro setup

```
### 2.2 Funciones y Tipos

#### cmd/zyrocli/cmd/setup.go
```

var SetupCmd = &cobra.Command{Use: "setup", Short: "Auto-install", RunE: runSetup}

type SetupFlags struct {
    DryRun    bool    // --dry-run
    Verbose   bool    // --verbose
    SkipGo    bool    // --skip-go
    TargetDir string  // --target-dir
    JSON      bool    // --json output
}

func runSetup(cmd *cobra.Command) error
func init()

```
#### cmd/zyrocli/cmd/doctor.go
```

type DoctorFlags struct {
    Fix     bool   // --fix
    DryRun  bool   // --dry-run
    Verbose bool   // --verbose
    JSON    bool   // --json output
}

var DoctorCmd = &cobra.Command{Use: "doctor", RunE: runDoctor}
func runDoctor(cmd *cobra.Command, args []string) error

```
#### internal/setup/check.go
```

type Dependency string
const (
    DepGo      Dependency = "go"
    DepUV      Dependency = "uv"
    DepHelixDB Dependency = "helixdb"
    DepDocker  Dependency = "docker"
    DepGit     Dependency = "git"
)

type Status string
const ( StatusOK Status = "ok"; StatusWarning Status = "warning"
        StatusError Status = "error"; StatusMissing Status = "missing" )

type CheckResult struct {
    Name    Dependency ` + "`" + `json:"name"` + "`" + `
    Status  Status     ` + "`" + `json:"status"` + "`" + `
    Version string     ` + "`" + `json:"version,omitempty"` + "`" + `
    Detail  string     ` + "`" + `json:"detail,omitempty"` + "`" + `
    Fix     *FixStep   ` + "`" + `json:"fix,omitempty"` + "`" + `
}

type PlatformInfo struct {
    OS, Arch, HomeDir, LocalBinDir, ConfigDir string
}

type Checker interface {
    CheckAll(ctx context.Context) map[Dependency]*CheckResult
    CheckOne(ctx context.Context, dep Dependency) *CheckResult
    Platform() PlatformInfo
}

type DependencyChecker struct {
    results  map[Dependency]*CheckResult
    platform PlatformInfo
}
func NewDependencyChecker() *DependencyChecker
func (dc *DependencyChecker) CheckAll(ctx context.Context) map[Dependency]*CheckResult
func (dc *DependencyChecker) CheckOne(ctx context.Context, dep Dependency) *CheckResult
func (dc *DependencyChecker) Platform() PlatformInfo
func detectOS() (PlatformInfo, error)
// Internal check methods:
func (dc *DependencyChecker) checkGo(ctx context.Context) *CheckResult
func (dc *DependencyChecker) checkUV(ctx context.Context) *CheckResult
func (dc *DependencyChecker) checkHelixDB(ctx context.Context) *CheckResult
func (dc *DependencyChecker) checkDocker(ctx context.Context) *CheckResult
func (dc *DependencyChecker) checkGit(ctx context.Context) *CheckResult

```
#### internal/setup/install.go
```

type Installer struct {
    DryRun    bool
    Verbose   bool
    Platform  PlatformInfo
}
func NewInstaller(dryRun, verbose bool) *Installer
func (inst *Installer) InstallUV(ctx context.Context) error
func (inst *Installer) InstallHelixDB(ctx context.Context) error
func (inst *Installer) CreateVenv(ctx context.Context, toolsDir string) error
func (inst *Installer) BuildGoBinary(ctx context.Context, outputPath string) error
func (inst *Installer) RegisterMCPConfig(ctx context.Context) error
func (inst *Installer) GenerateConfig(ctx context.Context) (*Config, error)

```
#### internal/setup/config.go
```

type Config struct {
    Version string      ` + "`" + `yaml:"version"` + "`" + `
    Paths   PathsConfig ` + "`" + `yaml:"paths"` + "`" + `
    HelixDB HelixConfig ` + "`" + `yaml:"helixdb"` + "`" + `
    MCP     MCPConfig   ` + "`" + `yaml:"mcp"` + "`" + `
}
type PathsConfig struct {
    HelixBinary, ZyroBinary, Venv, MCPTools string
    ConfigDir, SkillsDir, AuditDir          string
}
type HelixConfig struct {
    Host      string ` + "`" + `yaml:"host"` + "`" + `
    Port      int    ` + "`" + `yaml:"port"` + "`" + `
    AutoStart bool   ` + "`" + `yaml:"auto_start"` + "`" + `
}
type MCPConfig struct {
    AutoRegister bool ` + "`" + `yaml:"auto_register"` + "`" + `
    LazyLoader   bool ` + "`" + `yaml:"lazy_loader"` + "`" + `
}
func LoadConfig() (*Config, error)
func DefaultConfig() *Config
func SaveConfig(cfg *Config) error
func ConfigPath() string

```
#### internal/setup/doctor.go
```

type DoctorReport struct {
    Timestamp string        ` + "`" + `json:"timestamp"` + "`" + `
    Status    string        ` + "`" + `json:"status"` + "`" + `
    Checks    []CheckResult ` + "`" + `json:"checks"` + "`" + `
}
type FixStep struct {
    Description string ` + "`" + `json:"description"` + "`" + `
    Command     string ` + "`" + `json:"command,omitempty"` + "`" + `
    AutoFix     bool   ` + "`" + `json:"auto_fix"` + "`" + `
}
type Doctor struct {
    checker   Checker
    installer *Installer
    config    *Config
}
func NewDoctor(cfg *Config) *Doctor
func (d *Doctor) Run(ctx context.Context, fix bool) (*DoctorReport, error)
func (d *Doctor) Fix(ctx context.Context) error
func (d *Doctor) Report(ctx context.Context) (*DoctorReport, error)

```
### 2.3 Diagrama: zyro setup
```

zyro setup
  1. detectOS() → PlatformInfo{OS, Arch}
  2. CheckAll() → go:missing, uv:ok, helix:missing, git:ok
  3. InstallUV()     → curl astral.sh/uv/install.sh | bash (idempotente)
  4. InstallHelixDB()→ download GH release, chmod +x, start helixd
  5. CreateVenv()    → uv venv && uv sync mcp-tools/pyproject.toml
  6. BuildGoBinary() → go build -o ~/.local/bin/zyrocli
  7. GenerateConfig()-> ~/.zyro/config.yaml
  8. RegisterMCP()   → opencode.jsonc + plugins
  9. Doctor.Run(fix) → verifica y repara residuales

```
### 2.4 Schema ~/.zyro/config.yaml
```

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
### 2.5 Manejo de Errores
|Error|Causa|Manejo|
|---|---|---|
|ErrOSNotSupported|OS no Linux/Darwin|Mensaje + enlace issues|
|ErrGoNotInstalled|Go no en PATH|Link oficial, no auto-instala|
|ErrHelixDBNotRunning|No responde /health|Intentar start, warning|
|ErrBuildFailed|go build falla|Mostrar output del build|
|ErrConfigPermission|No escribir ~/.zyro|Sugerir sudo/chmod|
|Timeout instalacion|Red lenta|60s timeout, 1 retry|

### 2.6 Tests
```

func TestDetectOS(t *testing.T)
func TestDependencyChecker_CheckGo(t *testing.T) // mock exec.Command
func TestDependencyChecker_CheckGoMissing(t *testing.T)
func TestDependencyChecker_CheckUV(t *testing.T)
func TestInstaller_InstallUV(t *testing.T) // idempotente
func TestInstaller_InstallHelixDB(t *testing.T)
func TestLoadConfig(t *testing.T)
func TestDefaultConfig(t *testing.T)
func TestSaveConfig(t *testing.T)
func TestDoctor_Run(t *testing.T)
func TestDoctor_Fix(t *testing.T)
// Mocks: execCmdFunc, httpClientFunc, osStatFunc

```

---

## 3. Sprint 1: Agent-as-Validator

### 3.1 Archivos
```

mcp-tools/
├── agent.py             (NEW)  ← PydanticAI Agent con output_type=AgentDecision
├── capabilities.py      (NEW)  ← HelixReadCapability, AgentDependencies
├── approval.py          (NEW)  ← ApprovalGate (console / go_bridge modes)
├── models.py            (NEW)  ← AgentDecision, HelixNodeOutput, ZyroAgentInput
├── runner.py            (MOD)  ← REFACTOR: orquestador que llama al agente
├── pyproject.toml       (MOD)  ← +pydantic-graph, +boundari
├── helix_client.py      (MOD)  ← mantener solo debug interno
├── task_context.py      (MOD)  ← tool read-only del agente
├── search_code.py       (MOD)  ← tool read-only del agente
├── search_skills.py     (MOD)  ← tool read-only del agente
└── boundari_wrapper.py  (NEW)  ← envuelve tools con politica de fase

```
### 3.2 Modelos Pydantic (models.py)
```

from pydantic import BaseModel, Field
from enum import Enum
from typing import Optional, Any

class Action(str, Enum):
    create = "create"
    update = "update"
    search = "search"
    skip = "skip"

class HelixNodeOutput(BaseModel):
    label: str = Field(...)
    properties: dict[str, Any] = Field(default_factory=dict)
    project_id: Optional[int] = Field(default=None, ge=1)
    requires_approval: bool = Field(default=False)

class AgentDecision(BaseModel):
    action: Action
    reasoning: str = Field(..., min_length=10, max_length=5000)
    nodes: list[HelixNodeOutput] = Field(default_factory=list)
    requires_approval: bool = Field(default=False)
    metadata: dict[str, Any] = Field(default_factory=dict)

class ZyroAgentInput(BaseModel):
    protocol: str = "zyro-agent-v2"
    version: str = "2.0.0"
    request_id: str = ""
    phase: str = ""
    task: str = ""
    memory_context: str = ""
    boundari_phase: str = ""
    timeout_seconds: int = 120
    read_cap: dict[str, Any] = Field(default_factory=dict)

class HelixSearchResult(BaseModel):
    id: int; label: str = ""; content: str = ""
    score: float = 0.0; source: str = ""

class HelixReadInput(BaseModel):
    query: str; limit: int = 10
    node_labels: list[str] = Field(default_factory=list)

```
### 3.3 Capacidades (capabilities.py)
```

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
    memory_context: str = ""
    boundari_phase: str = ""
    request_id: str = ""

```
### 3.4 Agente Principal (agent.py)
```

from pydantic_ai import Agent, RunContext

zyro_agent: Agent[AgentDependencies, AgentDecision] = Agent(
    model="openai:gpt-5.2",
    deps_type=AgentDependencies,
    output_type=AgentDecision,
    system_prompt="""Eres un agente de analisis para el pipeline SDD.
NUNCA intentes escribir en HelixDB directamente.
NUNCA solicites tools de escritura.
Si detectas contradiccion con memoria causal, incluvelo en reasoning.""",
)

@zyro_agent.tool
async def search_code(ctx: RunContext[AgentDependencies], input: HelixReadInput) -> list[HelixSearchResult]
@zyro_agent.tool
async def search_skills(ctx: RunContext[AgentDependencies], input: HelixReadInput) -> list[HelixSearchResult]
@zyro_agent.tool
async def task_context(ctx: RunContext[AgentDependencies], task_id: int) -> str

async def run_agent(input_data: ZyroAgentInput, deps: AgentDependencies) -> AgentDecision

```
### 3.5 Approval Gate (approval.py)
```

class ApprovalGate:
    def __init__(self, mode: str = "console"): ...
    async def request_approval(self, decision: AgentDecision, phase: str) -> bool
    async def _console_approver(self, decision: AgentDecision) -> bool
    async def _go_bridge_approver(self, decision: AgentDecision) -> bool

```
### 3.6 Runner Refactorizado (runner.py)
```

async def main():
    input_raw = sys.stdin.read()
    input_data = ZyroAgentInput.model_validate_json(input_raw)
    deps = AgentDependencies(
        read_cap=HelixReadCapability(),
        phase=input_data.phase,
        task_description=input_data.task,
        memory_context=input_data.memory_context,
        boundari_phase=input_data.boundari_phase,
        request_id=input_data.request_id,
    )
    decision = await run_agent(input_data, deps)
    print(decision.model_dump_json(indent=2))

if __name__ == "__main__":
    asyncio.run(main())

```
### 3.7 Diagrama: Agent-as-Validator Flow
```

Go Scheduler                    Python Agent (subprocess)
    │                                   │
    ├── Construye contexto              │
    ├── Serializa ZyroAgentInput        │
    ├── Lanza subprocess Python         │
    │   stdin: JSON ──────────────────►  │
    │                                   ├── ZyroAgentInput.model_validate
    │                                   ├── Crea AgentDependencies
    │                                   ├── Si Boundari: wrap_tool
    │                                   ├── zyro_agent.run(task)
    │                                   │   ├── search_code (read)
    │                                   │   ├── search_skills (read)
    │                                   │   └── task_context (read)
    │                                   ├── Retorna AgentDecision
    │   ◄── stdout: JSON ────────────── │
    ├── Valida AgentDecision            │
    ├── Si requires_approval → Gate     │
    ├── Go escribe a HelixDB (SDK)      │
    ▼                                   ▼

```
### 3.8 Manejo de Errores
|Error|Causa|Manejo|
|---|---|---|
|stdin vacio|Go no envio datos|sys.exit(1)|
|model_validate falla|JSON mal formado|Error con detalle de campo|
|Timeout (120s)|LLM tarda mucho|Go mata proceso, log|
|Output invalido|No conforme AgentDecision|Reintentar 1 vez|
|Tool falla|HelixDB no responde|Tool retorna error, agente decide|
|save_to_helix llamado|Error de programa|raise RuntimeError|

### 3.9 Tests
```

# Unit tests
def test_agent_decision_valid()
def test_agent_decision_invalid_reasoning()
def test_helix_node_output_valid()

# Async tests
@pytest.mark.asyncio
async def test_run_agent_search()
async def test_agent_no_write_tools()
async def test_approval_gate_console_approve()
async def test_approval_gate_go_bridge()

# Mocks: monkeypatch sys.stdin, RunContext mock, LLM mock

```

---

## 4. Sprint 2: HelixDB SDK Go

### 4.1 Archivos
```

internal/db/helix/
├── helix.go              (MOD)  ← wrapper sobre client.Exec()
├── queries.go            (NEW)  ← queries tipadas (FindTask, UpsertCodeNode...)
├── search.go             (NEW)  ← busqueda hibrida vector+BM25+RRF
├── traverse.go           (NEW)  ← Repeat, Union, Choose, Coalesce
├── embedding.go          (NEW)  ← pipeline embeddings (OpenAI/Ollama)
├── indexes.go            (NEW)  ← CreateIndexIfNotExists
├── types.go              (MOD)  ← reemplazar tipos propios
├── errors.go             (MOD)  ← reemplazar por helix.HelixError
└── helix_test.go         (MOD)  ← tests actualizados

```
### 4.2 Client Wrapper (helix.go)
```

type ClientOptions struct {
    BaseURL    string        // default: "http://localhost:6969"
    Timeout    time.Duration // default: 30s
    MaxRetries int           // default: 3
}
type Client struct {
    inner *helix.Client
    opts  ClientOptions
}
func NewClient(opts ClientOptions) (*Client, error)
func (c *Client) Exec(ctx context.Context, q helix.Request, out any) error
func (c *Client) HealthCheck(ctx context.Context) error
func (c *Client) Close() error

```
### 4.3 Queries Tipadas (queries.go)
```

// Row types
type TaskRow struct { ID int64 ` + "`" + `json:"$id"` + "`" + `; Name, Description, Phase, Status string }
type CodeNodeRow struct { ID int64 ` + "`" + `json:"$id"` + "`" + `; Path, Summary, Language, Hash string }
type FactRow struct { ID int64 ` + "`" + `json:"$id"` + "`" + `; Type, Content string; Salience, Confidence float64 }
type ProjectRow struct { ID int64 ` + "`" + `json:"$id"` + "`" + `; Name, Description, Status, CurrentPhase string }
type SkillRow struct { ID int64 ` + "`" + `json:"$id"` + "`" + `; Name, Type, Version string }
type PatternRow struct { ID int64 ` + "`" + `json:"$id"` + "`" + `; Name, Description, Language string }

// Query builders (return helix.Request)
func FindTask(name string, projectID int64) helix.Request
func UpsertCodeNode(projectID int64, path, summary, language, hash string) helix.Request
func FindProject(name string) helix.Request
func ListFactsByPhase(phase, tenantID string, limit int64) helix.Request
func CreateFact(label string, props map[string]any, embedding []float32) helix.Request
func CreateEdge(fromID, toID int64, edgeType string, props map[string]any) helix.Request
func FindSkills(query string, limit int64) helix.Request
func FindPatterns(query string, language string, limit int64) helix.Request

```
### 4.4 Busqueda Hibrida (search.go)
```

type SearchResult struct {
    ID      int64   ` + "`" + `json:"$id"` + "`" + `
    Label   string  ` + "`" + `json:"label"` + "`" + `
    Content string  ` + "`" + `json:"content"` + "`" + `
    Score   float64 ` + "`" + `json:"score"` + "`" + `
    Source  string  ` + "`" + `json:"source"` + "`" + ` // "vector"|"text"|"hybrid"
}

type HybridSearchOptions struct {
    MaxResults int      // default: 10
    RRFFusionK int      // default: 60
    NodeLabels []string
    TenantID   string
    MinScore   float64
}

func HybridSearch(ctx context.Context, client *Client, query string,
    embedding []float32, opts HybridSearchOptions) ([]SearchResult, error)

func vectorSearch(ctx context.Context, client *Client, embedding []float32,
    opts HybridSearchOptions) ([]SearchResult, error)

func textBM25Search(ctx context.Context, client *Client, query string,
    opts HybridSearchOptions) ([]SearchResult, error)

// fuseRRF: Reciprocal Rank Fusion
// RRFScore = sum(1 / (k + rank_i(d)))
func fuseRRF(vector, text []SearchResult, k, maxResults int) []SearchResult

```
### 4.5 Traversals (traverse.go)
```

type LibraryInfo struct { ID int64 ` + "`" + `json:"$id"` + "`" + `; Name, Version string }
type FactWithPath struct { ID int64; Type, Content, Phase string; Confidence float64 }
type ProjectContext struct {
    Project   ProjectRow
    Tasks     []TaskRow
    Patterns  []PatternRow
    Skills    []SkillRow
    Libraries []LibraryInfo
}
type ContradictionPair struct { FactA, FactB FactRow; Similarity float64 }

func DiscoverCrossProjectSkills(ctx, client, skillName string) ([]LibraryInfo, error)
func TraverseCausalChain(ctx, client, factID int64, maxDepth int) ([]FactWithPath, error)
func TraverseProjectContext(ctx, client, projectID int64) (*ProjectContext, error)
func FindContradictions(ctx, client, tenantID string, threshold float64) ([]ContradictionPair, error)

```
### 4.6 Pipeline Embeddings (embedding.go)
```

type EmbeddingProvider string
const (
    ProviderOllama          EmbeddingProvider = "ollama"
    ProviderScaleway        EmbeddingProvider = "scaleway"
    ProviderGitHubModels    EmbeddingProvider = "github_models"
    ProviderCohere          EmbeddingProvider = "cohere"
    ProviderOpenAI          EmbeddingProvider = "openai"
)

type EmbeddingConfig struct {
    Provider   EmbeddingProvider  // "ollama" (default) → "scaleway" → "github_models"
    Model      string             // "mxbai-embed-large" (default) | "qwen3-embedding-8b"
    Dims       int                // 768 (default) | 1024 | 1536
    APIKey     string
    BaseURL    string
    BatchSize  int                // default: 20
    CacheSize  int                // default: 1000 (LRU)
    CacheDir   string             // ~/.zyro/embedding-cache/
    MaxRetries int                // default: 3
}

type EmbeddingService struct {
    config EmbeddingConfig
    client *http.Client
    cache  *lru.Cache
    mu     sync.RWMutex
}

func NewEmbeddingService(config EmbeddingConfig) (*EmbeddingService, error)
func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float32, error)
func (s *EmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
func (s *EmbeddingService) embedOllama(ctx context.Context, texts []string) ([][]float32, error)
func (s *EmbeddingService) embedScaleway(ctx context.Context, texts []string) ([][]float32, error)
func (s *EmbeddingService) embedGitHubModels(ctx context.Context, texts []string) ([][]float32, error)
func (s *EmbeddingService) embedCohere(ctx context.Context, texts []string) ([][]float32, error)
func (s *EmbeddingService) embedOpenAI(ctx context.Context, texts []string) ([][]float32, error)
func (s *EmbeddingService) embeddingAvailable() bool

```
### 4.6.1 Prioridad de Proveedores

El sistema implementa un pipeline de prioridad con degradación graceful:

```
┌─ ¿Ollama + mxbai-embed-large disponible?
│   ✅ → Usar (local, CPU/GPU, 768 dims)
├─ ¿No? → ¿Scaleway API configurado?
│   ✅ → Usar (qwen3-embedding-8b, 1M tokens gratis)
├─ ¿No? → ¿GitHub Models / Cohere configurado?
│   ✅ → Fallback terciario
└─ ¿Nada disponible?
    → BM25 puro (degradación graceful)
    → EmbeddingService.Embed() retorna nil
    → El sistema funciona completamente sin vectores
```

### 4.6.2 Embedding Harness MCP

El embedding harness es un MCP server separado que provee tools al agente:

- **Servidor:** `mcp-tools/embedding_harness.py` (FastMCP)
- **Cache LRU:** en disco en `~/.zyro/embedding-cache/` (SQLite)
- **Pipeline de prioridad:** Ollama → Scaleway → GitHub Models → BM25
- **Tools expuestas:**
  - `embed(text) → list[float]`: genera embedding para un texto
  - `embed_batch(texts) → list[list[float]]`: genera embeddings en batch
  - `status() → dict`: reporta proveedor activo, modelo, tamaño de caché
- **Instalación:** `zyro setup` configura todo interactivamente

```
OpenCode Agent
  │
  ├── llama a zyro-sdd-tasks (normal)
  │
  └── llama a zyro-embedding-harness (MCP tool)
        │
        ├── Si Ollama disponible → mxbai-embed-large (local, CPU)
        ├── Si no, pero hay API key → Scaleway/GitHub/Cohere (free API)
        └── Si nada → BM25 fallback (degradación graceful)
```

### 4.7 Indices (indexes.go)
```

type IndexSpec struct {
    Label, Property string
    IndexType       IndexType  // vector|text|equality|range|unique
    TenantProperty  string
}
type IndexType string
const ( IndexVector IndexType = "vector"; IndexText = "text"
       IndexEquality = "equality"; IndexRange = "range"; IndexUnique = "unique_equality" )

func EnsureIndexes(ctx context.Context, client *Client, specs []IndexSpec) error
func DefaultIndexes() []IndexSpec
func CreateIndexIfNotExists(spec IndexSpec) helix.Request

```
### 4.8 Diagrama: Busqueda Hibrida con RRF
```

HybridSearch(query, embedding, opts)
    │
    ├── goroutine 1: vectorSearch(embedding)
    │   VectorSearchNodes("Fact","embedding",queryVec,k,tenant)
    │
    ├── goroutine 2: textBM25Search(query)
    │   TextSearchNodes("Fact","content",queryStr,k,tenant)
    │
    └── sync.WaitGroup + channel
        │
        ▼
    fuseRRF(vectorHits, textHits, k=60, maxResults=10)
        │ RRFScore = sum(1/(k + rank_i(d)))
        ▼
    sort by RRFScore desc → Return top N

```
### 4.9 Manejo de Errores
|Error|Causa|Manejo|
|---|---|---|
|ErrConnectionFailed|HelixDB down|Retry 3x con backoff (100,200,300ms)|
|ErrConflict|Write conflict|Retry con backoff exponencial|
|ErrNotFound|Nodo no existe|Retornar nil, no error|
|Timeout embedding|API externa lenta|Fallback a Ollama|
|Cache miss|No cacheado|Calcular y almacenar|

### 4.10 Tests
```

// Client
func TestNewClient(t *testing.T)
func TestClientExec(t *testing.T)
func TestClientExecRetry(t *testing.T)

// Queries
func TestFindTaskQuery(t *testing.T)
func TestUpsertCodeNodeQuery(t *testing.T)
func TestCreateFactQuery(t *testing.T)

// Search
func TestHybridSearch(t *testing.T)
func TestFuseRRF(t *testing.T)
func TestFuseRRFEmpty(t *testing.T)

// Traverse
func TestDiscoverCrossProjectSkills(t *testing.T)
func TestTraverseCausalChain(t *testing.T)
func TestTraverseProjectContext(t *testing.T)

// Embeddings
func TestEmbeddingService(t *testing.T)
func TestEmbeddingServiceCache(t *testing.T)
func TestEmbeddingServiceFallback(t *testing.T)

// Indexes
func TestEnsureIndexes(t *testing.T)
func TestDefaultIndexes(t *testing.T)

// Mocks: helix.Client interface, http.Client, lru.Cache

```

---

## 5. Sprint 3: Boundari por Fase

### 5.1 Archivos
```

boundari/
├── phase0-boundari.yaml   (NEW)  ← F0: solo lectura
├── phase1-boundari.yaml   (NEW)  ← F1: lectura + web_fetch approval
├── phase2-boundari.yaml   (NEW)  ← F2: escritura planos con approval
├── phase3-boundari.yaml   (NEW)  ← F3: implementacion intensiva
└── phase4-boundari.yaml   (NEW)  ← F4: solo lectura otra vez

internal/boundari/
├── loader.go              (NEW)  ← LoadPolicy(phase)
├── enforcer.go            (NEW)  ← Enforce(action, policy)
└── types.go               (NEW)  ← Policy, ToolPolicy, Budget

mcp-tools/boundari_wrapper.py  (NEW)  ← wrapper Python para tools del agente

```
### 5.2 Tipos (types.go)
```

type Phase string
const ( PhaseF0 Phase = "F0"; PhaseF1 = "F1"; PhaseF2 = "F2"; PhaseF3 = "F3"; PhaseF4 = "F4" )

type Policy struct {
    Version     string                ` + "`" + `yaml:"version"` + "`" + `
    Phase       string                ` + "`" + `yaml:"phase"` + "`" + `
    Description string                ` + "`" + `yaml:"description"` + "`" + `
    Budget      Budget                ` + "`" + `yaml:"budget"` + "`" + `
    Tools       map[string]ToolPolicy ` + "`" + `yaml:"tools"` + "`" + `
    Data        DataConfig            ` + "`" + `yaml:"data,omitempty"` + "`" + `
    Outputs     OutputsConfig         ` + "`" + `yaml:"outputs,omitempty"` + "`" + `
    Tests       PolicyTests           ` + "`" + `yaml:"policy_tests,omitempty"` + "`" + `
}

type Budget struct {
    MaxToolCalls    int    ` + "`" + `yaml:"max_tool_calls"` + "`" + `
    MaxRuntimeSecs  int    ` + "`" + `yaml:"max_runtime_seconds"` + "`" + `
    MaxCostUSD      string ` + "`" + `yaml:"max_cost_usd"` + "`" + `
    MaxTokens       int    ` + "`" + `yaml:"max_tokens,omitempty"` + "`" + `
}

type ToolPolicy struct {
    Allow    bool            ` + "`" + `yaml:"allow"` + "`" + `
    Deny     bool            ` + "`" + `yaml:"deny,omitempty"` + "`" + `
    Approval *ApprovalPolicy ` + "`" + `yaml:"approval,omitempty"` + "`" + `
    Scopes   []string        ` + "`" + `yaml:"scopes,omitempty"` + "`" + `
    Risk     string          ` + "`" + `yaml:"risk,omitempty"` + "`" + `
}

type ApprovalPolicy struct {
    Required bool   ` + "`" + `yaml:"required,omitempty"` + "`" + `
    When     string ` + "`" + `yaml:"when,omitempty"` + "`" + ` // condicion safe_eval
}

type EnforcementResult struct {
    Allowed         bool   ` + "`" + `json:"allowed"` + "`" + `
    ToolName        string ` + "`" + `json:"tool_name"` + "`" + `
    Reason          string ` + "`" + `json:"reason,omitempty"` + "`" + `
    RequiresApproval bool  ` + "`" + `json:"requires_approval,omitempty"` + "`" + `
}

type AuditEvent struct {
    Timestamp, Phase, ToolName string
    Args                       map[string]any
    Allowed                    bool
    Reason, Duration           string
}

type BudgetUsage struct {
    ToolCalls int; RuntimeSecs float64; CostUSD float64; Tokens int
}

```
### 5.3 Loader (loader.go)
```

func LoadPolicy(phase Phase, searchDirs ...string) (*Policy, error)
func LoadDefaultPolicy(phase Phase) *Policy
func ValidatePolicy(p *Policy) error

```
### 5.4 Enforcer (enforcer.go)
```

type Enforcer struct {
    policy  *Policy
    usage   BudgetUsage
    startAt time.Time
}

func NewEnforcer(policy *Policy) *Enforcer
func (e *Enforcer) CheckTool(toolName string, args map[string]any) EnforcementResult
func (e *Enforcer) LogAudit(event AuditEvent)
func (e *Enforcer) Usage() BudgetUsage
func (e *Enforcer) IsBudgetExceeded() bool
func (e *Enforcer) SaveAuditLog(ctx context.Context, path string) error
func (e *Enforcer) Reset()

```
### 5.5 Wrapper Python (boundari_wrapper.py)
```

class BoundariWrapper:
    def __init__(self, phase: str, policies_dir: str = None, audit_dir: str = None):
        self.phase = phase
        self.policies_dir = policies_dir or "~/.zyro/boundari"
        self.audit_dir = audit_dir or "~/.zyro/audit"
        self.tool_calls = 0
        self.audit_log: list[dict] = []
        self.boundary = None
        if BOUNDARI_AVAILABLE:
            self._load_boundary()

    def _load_boundary(self) -> None: ...
    def wrap_tool(self, tool_name: str, tool_func: Callable,
                  raise_on_denied: bool = True) -> Callable: ...
    def _check_policy_fallback(self, tool_name: str, args: dict) -> Decision: ...
    def save_audit(self, phase: str) -> str: ...

```
### 5.6 YAML Policies (resumen F0-F4)
```

# phase0-boundari.yaml: F0 — solo lectura
budget: {max_tool_calls: 30, max_runtime_seconds: 300, max_cost_usd: "0.10"}
tools:
  read_file:      {allow: true}
  search_code:    {allow: true}
  write_file:     {allow: false, deny: true}
  shell_exec:     {allow: false, deny: true}
  delete_file:    {allow: false, deny: true}
  git_commit:     {allow: false, deny: true}
  git_push:       {allow: false, deny: true}
  network_request:{allow: false, deny: true}
  npm_install:    {allow: false, deny: true}

# phase1-boundari.yaml: F1 — lectura + web_fetch con approval
budget: {max_tool_calls: 40, max_runtime_seconds: 600, max_cost_usd: "0.25"}
tools:
  web_fetch:      {allow: true, approval: {when: "'localhost' not in url"}}
  write_file:     {allow: false, deny: true}
  shell_exec:     {allow: false, deny: true}
  pypi_search:    {allow: true}

# phase2-boundari.yaml: F2 — escritura planos con approval
budget: {max_tool_calls: 50, max_runtime_seconds: 600, max_cost_usd: "0.35"}
tools:
  write_file:     {allow: true, approval: {required: true}}
  shell_exec:     {allow: true, approval: {when: "command not in safe_commands"}}
  delete_file:    {allow: false, deny: true}

# phase3-boundari.yaml: F3 — implementacion intensiva
budget: {max_tool_calls: 150, max_runtime_seconds: 1800, max_cost_usd: "1.00"}
tools:
  write_file:     {allow: true, approval: {when: "'src/' not in path"}}
  shell_exec:     {allow: true, approval: {required: true}}
  npm_install:    {allow: true, approval: {when: "package_count > 3"}}

# phase4-boundari.yaml: F4 — solo lectura + approval para correcciones
budget: {max_tool_calls: 30, max_runtime_seconds: 300, max_cost_usd: "0.10"}
tools:
  write_file:     {allow: true, approval: {required: true}}
  shell_exec:     {allow: false, deny: true}
  git_commit:     {allow: true, approval: {required: true}}
  git_push:       {allow: true, approval: {required: true}}

```
### 5.7 Diagrama: Flujo del Enforcer
```

Enforcer.CheckTool(toolName, args)
  1. IsBudgetExceeded()? → Si: Decision("budget_exceeded")
  2. ToolPolicy existe?  → No: Decision("tool_not_defined")
  3. policy.Deny?        → Si: Decision("denied")
  4. policy.Allow?       → No: Decision("not_allowed")
  5. Approval.Required?  → Si: Decision(allowed=true, requires_approval=true)
  6. Approval.When?      → Evaluar condicion safe_eval
  7. LogAudit(event)     → SaveAuditLog(path)

```
### 5.8 Tests
```

func TestLoadPolicy(t *testing.T)
func TestLoadPolicyNotFound(t *testing.T)
func TestLoadDefaultPolicy(t *testing.T)
func TestValidatePolicy(t *testing.T)

func TestEnforcer_CheckTool_Allow(t *testing.T)
func TestEnforcer_CheckTool_Deny(t *testing.T)
func TestEnforcer_CheckTool_NotFound(t *testing.T)
func TestEnforcer_CheckTool_BudgetExceeded(t *testing.T)
func TestEnforcer_CheckTool_ApprovalRequired(t *testing.T)
func TestEnforcer_SaveAuditLog(t *testing.T)
func TestEnforcer_IsBudgetExceeded(t *testing.T)

func TestAllPoliciesLoad(t *testing.T) // carga los 5 YAML
func TestPhase0NoWriteTools(t *testing.T)
func TestPhase3WriteFileConditional(t *testing.T)

```

---

## 6. Sprint 4: Memoria Causal

### 6.1 Archivos
```

internal/memory/
├── memory.go              (NEW)  ← interfaz EngramStore
├── schema.go              (NEW)  ← Fact, FactType, CausalEdgeType
├── store.go               (NEW)  ← SaveFact, SaveFactsBatch, AddCausalEdge
├── recall.go              (NEW)  ← RecallMemories (hibrida + navegacion causal)
├── contradictions.go      (NEW)  ← DetectContradictions, ResolveContradiction
└── decay.go               (NEW)  ← ApplyDecay (Ebbinghaus), ReinforceSalience

internal/scheduler/
└── memory_hook.go         (NEW)  ← PrePhase, PostPhase hooks

agents/
└── fact_extractor.py      (NEW)  ← extractor de hechos con LLM local

```
### 6.2 Schema (schema.go)
```

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
    ID             int64              ` + "`" + `json:"$id"` + "`" + `
    Type           FactType           ` + "`" + `json:"type"` + "`" + `
    Content        string             ` + "`" + `json:"content"` + "`" + `
    Embedding      []float32          ` + "`" + `json:"embedding,omitempty"` + "`" + `
    Salience       float64            ` + "`" + `json:"salience"` + "`" + `
    Confidence     float64            ` + "`" + `json:"confidence"` + "`" + `
    Source         string             ` + "`" + `json:"source"` + "`" + `
    Phase          string             ` + "`" + `json:"phase"` + "`" + `
    CreatedAt      time.Time          ` + "`" + `json:"created_at"` + "`" + `
    LastAccessedAt time.Time          ` + "`" + `json:"last_accessed_at"` + "`" + `
    AccessCount    int64              ` + "`" + `json:"access_count"` + "`" + `
    DecayRate      float64            ` + "`" + `json:"decay_rate"` + "`" + `
    ExpiresAt      time.Time          ` + "`" + `json:"expires_at"` + "`" + `
    IsActive       bool               ` + "`" + `json:"is_active"` + "`" + `
    IsStale        bool               ` + "`" + `json:"is_stale"` + "`" + `
    ProjectID      string             ` + "`" + `json:"project_id"` + "`" + `
    Metadata       map[string]any     ` + "`" + `json:"metadata,omitempty"` + "`" + `
}

type CausalEdge struct {
    ID, FromID, ToID int64
    Type             CausalEdgeType
    CreatedAt        time.Time
    Properties       map[string]any
}

type ContradictionPair struct { FactA, FactB Fact; Similarity float64 }
type ContradictionStrategy string
const (
    StrategyNewestWins        = "newest_wins"
    StrategyHighestConfidence = "highest_confidence"
    StrategyKeepBoth          = "keep_both"
)

type RecallOpts struct {
    QueryText    string
    QueryVector  []float32
    MaxResults   int       // default: 10
    MinSalience  float64   // default: 0.2
    FactTypes    []FactType
    IncludeStale bool
    Phase        string
    ProjectID    string
}

type MemoryResult struct {
    FactID int64; Type, Content string
    Salience, Confidence float64; Phase string; Score float64
}

type DecayConfig struct {
    BaseDecayRate     float64 // 0.05
    AccessBoost       float64 // 0.3
    SalienceThreshold float64 // 0.15
    MaxSalience       float64 // 1.0
    DefaultExpiryDays int     // 90
}

```
### 6.3 Interfaz EngramStore (memory.go)
```

type EngramStore interface {
    SaveFact(ctx context.Context, fact *Fact) (int64, error)
    SaveFactsBatch(ctx context.Context, facts []*Fact) ([]int64, error)
    AddCausalEdge(ctx context.Context, edge *CausalEdge) error
    RecallMemories(ctx context.Context, opts RecallOpts) ([]*MemoryResult, error)
    DetectContradictions(ctx context.Context, projectID string, threshold float64) ([]ContradictionPair, error)
    ResolveContradiction(ctx context.Context, pair ContradictionPair, strategy ContradictionStrategy) error
    ReinforceSalience(ctx context.Context, factIDs []int64) error
    DecayAndRefresh(ctx context.Context, projectID string) error
    GetFactByID(ctx context.Context, factID int64) (*Fact, error)
    GetCausalChain(ctx context.Context, factID int64, maxDepth int) ([]*Fact, error)
}

```
### 6.4 Store Implementation (store.go)
```

type HelixEngramStore struct {
    client       *db.Client
    embeddingSvc *db.EmbeddingService
    decayConfig  DecayConfig
}

func NewHelixEngramStore(client *db.Client, embeddingSvc *db.EmbeddingService) *HelixEngramStore
func (s *HelixEngramStore) SaveFact(ctx context.Context, fact *Fact) (int64, error)
func (s *HelixEngramStore) SaveFactsBatch(ctx context.Context, facts []*Fact) ([]int64, error)
func (s *HelixEngramStore) AddCausalEdge(ctx context.Context, edge *CausalEdge) error

```
### 6.5 Recall (recall.go)
```

func (s *HelixEngramStore) RecallMemories(ctx context.Context, opts RecallOpts) ([]*MemoryResult, error)
func (s *HelixEngramStore) GetCausalChain(ctx context.Context, factID int64, maxDepth int) ([]*Fact, error)

```
### 6.6 Contradicciones (contradictions.go)
```

func (s *HelixEngramStore) DetectContradictions(ctx context.Context, projectID string,
    threshold float64) ([]ContradictionPair, error)
func (s *HelixEngramStore) ResolveContradiction(ctx context.Context,
    pair ContradictionPair, strategy ContradictionStrategy) error
func (s *HelixEngramStore) deactivateFact(ctx context.Context, factID int64, reason string) error

```
### 6.7 Decaimiento Ebbinghaus (decay.go)
```

// salience(t) = salience_0 * e^(-decay_rate * days_since_access)
func (s *HelixEngramStore) DecayAndRefresh(ctx context.Context, projectID string) error

// salience += accessBoost * (maxSalience - salience) = 0.7*old + 0.3
func (s *HelixEngramStore) ReinforceSalience(ctx context.Context, factIDs []int64) error

```
### 6.8 Memory Hooks (memory_hook.go)
```

type MemoryHooks struct {
    store       memory.EngramStore
    embeddingSvc *db.EmbeddingService
    factExtractorPath string
}

func NewMemoryHooks(store memory.EngramStore, embeddingSvc *db.EmbeddingService) *MemoryHooks

// PrePhase: inyecta memoria causal en el prompt antes de ejecutar fase
func (h *MemoryHooks) PrePhase(ctx context.Context, phase string, taskDesc string) (string, error)

// PostPhase: extrae hechos del log de la fase y los guarda en HelixDB
func (h *MemoryHooks) PostPhase(ctx context.Context, phase string, conversationLog []byte) error

func formatMemoryForPrompt(facts []*memory.MemoryResult) string

```
### 6.9 Extractor Python (fact_extractor.py)
```

#!/usr/bin/env python3
"""Extractor de hechos para memoria causal.
Analiza logs de conversacion y extrae Facts estructurados.
Uso: python fact_extractor.py --input <log.json> --phase F1"""

FACT_PATTERNS = {
    "decision": [r"(?:vamos a usar|usamos|decidimos) (.+?)(?:\.|$)"],
    "error":    [r"(?:error|bug|problema|fallo):?\s*(.+?)(?:\.|$)"],
    "preference":[r"(?:prefiero|preferimos|mejor usar) (.+?)(?:\.|$)"],
    "pattern":  [r"(?:patron|pattern|arquitectura) (.+?)(?:\.|$)"],
    "dependency":[r"(?:dependemos de|necesitamos|requiere) (.+?)(?:\.|$)"],
    "observation":[r"(?:observo|noto|veo que|detecto) (.+?)(?:\.|$)"],
}

def extract_facts_regex(log_text: str, phase: str) -> list[dict]: ...
def extract_facts_llm(log_text: str, phase: str, ollama_model="llama3.2") -> list[dict]: ...
def main(): ...

```
### 6.10 Diagrama: Ciclo de Memoria
```

PRE-FASE:
  Scheduler.RunPhase(F1)
    ├── MemoryHooks.PrePhase(F1, taskDesc)
    │     → EngramStore.RecallMemories(query, phase=F1)
    │       ├── VectorSearch (ANN) + TextSearch (BM25) + RRF
    │       └── Filter: salience > 0.2, is_active
    │     → formatMemoryForPrompt(facts)
    │     → Inyectar en prompt del agente
    └── Ejecutar fase...

POST-FASE:
  MemoryHooks.PostPhase(F0, log)
    ├── Guardar log temporal
    ├── python fact_extractor.py --input log --phase F0
    ├── Parsear output → []Fact
    ├── computeEmbedding para cada fact
    ├── EngramStore.SaveFactsBatch(facts)
    ├── Crear edges causales
    ├── DetectContradictions → ResolveContradiction
    └── ReinforceSalience(accessedFactIDs)

CRON:
  EngramStore.DecayAndRefresh(projectID)
    ├── Load all active Facts
    ├── For each: newSalience = salience * e^(-decay * daysSinceAccess)
    ├── if newSalience < 0.15 → mark stale
    └── if expired → mark inactive

```
### 6.11 Tests
```

func TestFactTypes(t *testing.T)
func TestCausalEdgeTypes(t *testing.T)
func TestHelixEngramStore_SaveFact(t *testing.T)
func TestHelixEngramStore_SaveFactsBatch(t *testing.T)
func TestHelixEngramStore_AddCausalEdge(t *testing.T)
func TestHelixEngramStore_RecallMemories(t *testing.T)
func TestHelixEngramStore_RecallMemories_Empty(t *testing.T)
func TestHelixEngramStore_GetCausalChain(t *testing.T)
func TestDetectContradictions(t *testing.T)
func TestResolveContradiction_NewestWins(t *testing.T)
func TestResolveContradiction_HighestConfidence(t *testing.T)
func TestDecayAndRefresh_Ebbinghaus(t *testing.T) // 0.7*e^(-0.05*30) approx 0.156
func TestReinforceSalience(t *testing.T) // 0.5 -> 0.5+0.3*(1-0.5)=0.65
func TestFormatMemoryForPrompt(t *testing.T)

```

---

## 7. Sprint 5: OpenCode + Boomerang

### 7.1 Archivos
```

internal/boomerang/
├── orchestrator.go       (NEW)  ← ciclo Boomerang completo (6 pasos)
├── memory.go             (NEW)  ← paso 1: consultar memoria causal
├── think.go              (NEW)  ← paso 2: planificar DAG de tareas
├── delegate.go           (NEW)  ← paso 3: repartir a subagentes
├── git.go                (NEW)  ← paso 4: verificar estado repo
├── quality.go            (NEW)  ← paso 5: linters, tests, validaciones
└── save.go               (NEW)  ← paso 6: guardar decisiones en HelixDB

internal/opencode/
├── config.go             (MOD)  ← simplificar: solo perfiles + plugins
├── plugin.go             (NEW)  ← gestion de plugins OpenCode
├── mcptools_embed.go     (MOD)  ← deprecar: marcar obsoleto
└── skills_embed.go       (MOD)  ← deprecar: marcar obsoleto

internal/scheduler/
├── scheduler.go          (MOD)  ← integrar ciclo Boomerang en Run
├── approval.go           (MOD)  ← reemplazar stdin por subagentes
└── phase.go              (MOD)  ← agregar campos Boomerang

```
### 7.2 Orquestador (orchestrator.go)
```

type BoomerangOrchestrator struct {
    memoryStore     memory.EngramStore
    boundariLoader  *boundari.Loader
    delegateSvc     *DelegateService
    gitChecker      *GitChecker
    qualityGate     *QualityGate
    saveService     *SaveService
    maxIterations   int // default: 3
}

func NewBoomerangOrchestrator(store memory.EngramStore, bl *boundari.Loader) *BoomerangOrchestrator

// RunPhase ejecuta el ciclo Boomerang completo: Memory→Think→Delegate→Git→Quality→Save
func (o *BoomerangOrchestrator) RunPhase(ctx context.Context, cfg PhaseConfig) (*PhaseResult, error)

type PhaseConfig struct {
    Phase, TaskDesc, ProjectID string
    MemoryLimit, Iterations    int
    Timeout                    time.Duration
}

type PhaseResult struct {
    Phase                        string ` + "`" + `json:"phase"` + "`" + `
    Success                      bool   ` + "`" + `json:"success"` + "`" + `
    Iterations, MemoryUsed       int    ` + "`" + `json:",omitempty"` + "`" + `
    TasksPlanned, NodesCreated   int    ` + "`" + `json:",omitempty"` + "`" + `
    GitStatus                    string ` + "`" + `json:"git_status"` + "`" + `
    QualityOK                    bool   ` + "`" + `json:"quality_ok"` + "`" + `
    FactsSaved                   int    ` + "`" + `json:"facts_saved"` + "`" + `
    Duration                     time.Duration ` + "`" + `json:"duration_ms"` + "`" + `
    Error                        string ` + "`" + `json:"error,omitempty"` + "`" + `
}

```
### 7.3 Think Step (think.go)
```

type TaskDAG struct {
    Tasks          []TaskSpec   ` + "`" + `json:"tasks"` + "`" + `
    Deps           [][2]int     ` + "`" + `json:"deps"` + "`" + `
    ParallelGroups [][]int      ` + "`" + `json:"parallel_groups"` + "`" + `
}

type TaskSpec struct {
    ID          int      ` + "`" + `json:"id"` + "`" + `
    Name        string   ` + "`" + `json:"name"` + "`" + `
    Description string   ` + "`" + `json:"description"` + "`" + `
    Agent       string   ` + "`" + `json:"agent"` + "`" + ` // subagente OpenCode
    Tags        []string ` + "`" + `json:"tags,omitempty"` + "`" + `
    DependsOn   []int    ` + "`" + `json:"depends_on,omitempty"` + "`" + `
}

func (o *BoomerangOrchestrator) ThinkStep(ctx context.Context, phase string, memoryContext string) (*TaskDAG, error)
func generateDAGForPhase(phase string, memoryContext string) *TaskDAG

```
### 7.4 Delegate Step (delegate.go)
```

type DelegateService struct { opencodeBin string }
func NewDelegateService(opencodeBin string) *DelegateService

func (o *BoomerangOrchestrator) DelegateStep(ctx context.Context, dag *TaskDAG,
    phase string, boundariPolicy *boundari.Policy) (*DelegateResult, error)

type DelegateResult struct {
    NodesCreated int
    TaskResults  map[string]TaskResult
}
type TaskResult struct {
    TaskName string; Success bool; Output string; Nodes []int64
}

```
### 7.5 Git Step (git.go)
```

type GitChecker struct{}
func NewGitChecker() *GitChecker

type GitStatus struct {
    Clean              bool   ` + "`" + `json:"clean"` + "`" + `
    Branch             string ` + "`" + `json:"branch"` + "`" + `
    Changed, Untracked int    ` + "`" + `json:",omitempty"` + "`" + `
    Ahead, Behind      int    ` + "`" + `json:",omitempty"` + "`" + `
    Error              string ` + "`" + `json:"error,omitempty"` + "`" + `
}

func (o *BoomerangOrchestrator) GitStep(ctx context.Context) (*GitStatus, error)
func (g *GitChecker) Status(ctx context.Context) (*GitStatus, error)
func (g *GitChecker) DiffCount(ctx context.Context) (int, error)

```
### 7.6 Quality Step (quality.go)
```

type QualityGate struct{}
type QualityResult struct {
    Passed bool; Issues []QualityIssue
}
type QualityIssue struct {
    Severity, Tool, Message string
}

func (o *BoomerangOrchestrator) QualityStep(ctx context.Context, phase string, dag *TaskDAG) (*QualityResult, error)

```
### 7.7 Save Step (save.go)
```

type SaveService struct {
    memoryStore  memory.EngramStore
    embeddingSvc *db.EmbeddingService
}
type SaveResult struct {
    FactsSaved, EdgesCreated, Contradictions int
}

func (o *BoomerangOrchestrator) SaveStep(ctx context.Context, phase string,
    delegateResult *DelegateResult, logData []byte) (*SaveResult, error)

```
### 7.8 Approval Refactor (approval.go)
```

// Reemplaza PromptApproval() por subagentes OpenCode con question:ask
func ApprovalGate(ctx context.Context, phase Phase, summary string) (bool, error) {
    cmd := exec.CommandContext(ctx, "opencode", "subagent", "zyro-approval-gate",
        "--param", "phase="+string(phase),
        "--param", "summary="+summary,
    )
    output, err := cmd.Output()
    if err != nil { return false, err }
    var result struct { Approved bool ` + "`" + `json:"approved"` + "`" + ` }
    json.Unmarshal(output, &result)
    return result.Approved, nil
}

```
### 7.9 Plugin Management (plugin.go)
```

type PluginConfig struct {
    ClaudeBridge bool
    LazyLoader   bool
    MultiAgent   bool
    CustomPaths  []string
    Sources      []SourceConfig
}
type SourceConfig struct { Dir, Namespace string }

func EnsurePluginsConfig(opencodeDir string, cfg PluginConfig) error
func WriteBridgePlugin(pluginsDir string, cfg PluginConfig) error

```
### 7.10 Diagrama: Ciclo Boomerang Completo
```

BoomerangOrchestrator.RunPhase(ctx, config)
    │
    ├── [1/6] MEMORY ──────────────────────────────────
    │     MemoryStep(ctx, phase, taskDesc)
    │     → "MEMORIA CAUSAL: ..." (string para inyectar)
    │
    ├── [2/6] THINK ───────────────────────────────────
    │     ThinkStep(ctx, phase, memoryContext)
    │     → TaskDAG{Tasks, Deps, ParallelGroups}
    │
    ├── [3/6] DELEGATE ────────────────────────────────
    │     DelegateStep(ctx, dag, phase, boundariPolicy)
    │     │ Para cada grupo paralelo en dag.ParallelGroups:
    │     │   ├── task[0] → subproceso OpenCode (subagente A)
    │     │   ├── task[1] → subproceso OpenCode (subagente B)
    │     │   └── task[2] → subproceso OpenCode (subagente C)
    │     │ Cada subproceso: skill via bridge + MCP lazy + Agent + Boundari
    │     └── DelegateResult{NodesCreated, TaskResults}
    │
    ├── [4/6] GIT ─────────────────────────────────────
    │     GitStep(ctx) → GitStatus{Clean, Branch, Changed}
    │
    ├── [5/6] QUALITY ─────────────────────────────────
    │     QualityStep(ctx, phase, dag)
    │     │ if !passed && iteration < maxIterations:
    │     │   → vuelve a DELEGATE con feedback
    │     └── QualityResult{Passed, Issues}
    │
    └── [6/6] SAVE ────────────────────────────────────
          SaveStep(ctx, phase, delegateResult, logData)
          → SaveResult{FactsSaved, EdgesCreated, Contradictions}
    
    → PhaseResult{Success, Iterations, ...}

```

### 7.11 Tests
```

func TestBoomerangOrchestrator_RunPhase(t *testing.T)
func TestBoomerangOrchestrator_RunPhase_QualityLoop(t *testing.T)
func TestBoomerangOrchestrator_RunPhase_MaxIterations(t *testing.T)
func TestMemoryStep(t *testing.T)
func TestFormatMemoryPrompt(t *testing.T)
func TestThinkStep_PhaseF0(t *testing.T)
func TestThinkStep_PhaseF3(t *testing.T)
func TestGenerateDAGForPhase(t *testing.T)
func TestDelegateStep(t *testing.T)
func TestDelegateStep_ParallelGroups(t *testing.T)
func TestGitStep_Clean(t *testing.T)
func TestGitStep_Dirty(t *testing.T)
func TestQualityStep_Passed(t *testing.T)
func TestQualityStep_Failed(t *testing.T)
func TestSaveStep(t *testing.T)
func TestApprovalGate(t *testing.T)

// Mocks: memory.EngramStore, boundari.Enforcer, exec.Command, git exec, HelixDB

```

---

## 8. Contratos Inter-Sprint

### 8.1 Go ↔ Python (stdin/stdout JSON)
```

PROTOCOLO: zyro-agent-v2
TRANSPORTE: stdin/stdout del subproceso Python
TIMEOUT: configurable (default 120s)
ENCODING: UTF-8 JSON

INPUT (Go → Python):
{
  "protocol": "zyro-agent-v2",
  "version": "2.0.0",
  "request_id": "uuid-v7",
  "phase": "F0",
  "task": "descripcion de tarea",
  "memory_context": "texto...",
  "boundari_phase": "F0",
  "timeout_seconds": 120,
  "read_cap": {"max_results": 10, "allowed_nodes": ["Pattern","Library",...]}
}

OUTPUT (Python → Go):
{
  "protocol": "zyro-agent-v2",
  "version": "2.0.0",
  "request_id": "uuid-v7",
  "action": "search",
  "reasoning": "texto...",
  "nodes": [
    {"label": "Pattern", "properties": {...},
     "project_id": 1005, "requires_approval": false}
  ],
  "requires_approval": false,
  "metadata": {"model": "gpt-5.2", "tokens_used": 4500}
}

ERROR (Python → Go):
{
  "protocol": "zyro-agent-v2",
  "version": "2.0.0",
  "request_id": "uuid-v7",
  "error": {"code": "TIMEOUT", "message": "Agent timed out after 120s"}
}

```
### 8.2 Labels HelixDB y Edges
|Label|Indexes|Edges|
|---|---|---|
|Developer|equality: name|HAS_PROJECT|
|Project|text: name, equality: status|HAS_CODENODE, HAS_PATTERN, USES_LIB, REQUIRES_SKILL|
|CodeNode|text: summary, vector: embedding, equality: path|—|
|Skill|text: name|REQUIRES_SKILL(inv)|
|Task|text: name, equality: phase|DEPENDS_ON|
|Pattern|text: name, text: description|REFERENCES|
|Library|text: name|—|
|Fact|text: content, vector: embedding, equality: type, equality: is_active|CAUSED, PRECEDES, CONTRADICTS, SUPPORTS, REQUIRES, DERIVES_FROM, REFERENCES|

### 8.3 Formatos de Archivo
```

~/.zyro/config.yaml                    ← Configuracion general (Sprint 0)
~/.zyro/audit/phase{N}-{timestamp}.jsonl ← Auditoria Boundari (Sprint 3)
~/.config/opencode/opencode.jsonc      ← Config OpenCode (Sprint 5)
~/.config/opencode/plugins/zyrocli.ts  ← Plugin bridge (Sprint 5)
~/.config/zyro/skills/*/SKILL.md       ← Skills declarativas (Sprint 5)
boundari/phase{N}-boundari.yaml        ← Politicas por fase (Sprint 3)

```

---

## 9. Estrategia de Tests

### 9.1 Piramide de Tests
```

         /\
        /  \          E2E: Pipeline completo (1-2 tests)
       /    \
      /------\       Integracion: SDK + HelixDB real, Python real (~10 tests)
     /        \
    /----------\
   /            \   Unitarios: cada funcion, struct, query builder (~150 tests)
  /--------------\

```
### 9.2 Tests Unitarios Go
|Paquete|Tests|Mocks|
|---|---|---|
|internal/setup/|15|exec.Command, os.Stat, http.Client|
|internal/db/helix/|25|helix.Client interface, http.Client|
|internal/boundari/|15|YAML loader (archivos temporales)|
|internal/memory/|20|db.Client mock, EmbeddingService mock|
|internal/boomerang/|20|memory.EngramStore, exec.Command|
|internal/scheduler/|10|PhaseRunner mock|
|Total Go|~105||

### 9.3 Tests Unitarios Python
|Modulo|Tests|Mocks|
|---|---|---|
|models.py|8|— (puro Pydantic)|
|agent.py|6|RunContext mock, LLM mock|
|approval.py|4|sys.stdin monkeypatch|
|boundari_wrapper.py|6|boundari.Boundary mock|
|fact_extractor.py|8|ollama.Client mock|
|Total Python|~32||

### 9.4 Tests de Integracion
|Test|Descripcion|Dependencia|
|---|---|---|
|TestHelixDBExec|Exec real contra HelixDB test container|Docker: helixdb/helix-db:latest|
|TestHybridSearchE2E|Insertar nodos, buscar, verificar RRF|HelixDB real|
|TestEmbeddingOpenAI|Llamar API real de embeddings|API key OpenAI|
|TestPythonAgent|Lanzar subproceso JSON → decision|Python + uv|
|TestZyroSetup|zyro setup --dry-run|Binario compilado|

### 9.5 CI Pipeline
```

# .github/workflows/test.yml
jobs:
  unit-go:
    runs-on: ubuntu-latest
    steps:
      - setup-go 1.21+
      - go test ./internal/... -short

  unit-python:
    runs-on: ubuntu-latest
    steps:
      - setup-python 3.11+
      - pip install -e mcp-tools/
      - pytest mcp-tools/tests/

  integration:
    runs-on: ubuntu-latest
    services:
      helixdb:
        image: helixdb/helix-db:latest
        ports: [6969:6969]
    steps:
      - setup-go 1.21+
      - setup-python 3.11+
      - go test ./internal/db/helix/... -run Integration
      - python mcp-tools/tests/test_integration.py

  lint:
    runs-on: ubuntu-latest
    steps:
      - golangci-lint run ./...
      - ruff check mcp-tools/

```

---

## 10. Matriz de Riesgos Tecnicos

|#|Riesgo|Severidad|Prob|Impacto|Mitigacion|
|---|---|---|---|---|---|
|1|Boundari v0.1.0 alpha cambia API|CRITICO|Alta|Bloquea S3|Congelar version. Enforcer Go hardcodeado como fallback. Tests CI.|
|2|HelixDB Go SDK cambia API|CRITICO|Media|Bloquea S2+S4|Version exacta en go.mod. Tests integracion CI. Wrapper Client propio.|
|3|OpenCode plugin system inestable|CRITICO|Media|Bloquea S5|Mantener escritura directa opencode.jsonc como fallback.|
|4|Embeddings locales lentos (CPU)|ALTO|Alta|Setup >60s|Cache LRU 1000 entradas. Batch asincrono. Fallback BM25-only.|
|5|Agente Python se cuelga/timeout|ALTO|Media|Pipeline bloqueado|Timeout 120s Go. Kill+restart. Logs de timeout.|
|6|Extractor LLM produce falsos positivos|ALTO|Alta|Ruido en memoria|confidence threshold 0.6. Revision humana opcional.|
|7|Grafo causal crece sin control|MEDIO|Media|Degradacion|Limite 10000 facts/tenant. Decaimiento diario. Archive 90d.|
|8|Contradicciones no detectadas|MEDIO|Media|Memoria inconsistente|Embedding + reglas explicitas. Humano puede marcar manual.|
|9|OpenCode sin paralelizacion nativa|MEDIO|Alta|Fases lentas|Plugin multiagent. Go lanza procesos en paralelo y consolida.|
|10|Subagente no produce output|MEDIO|Baja|Delegate falla|Reintentar 1 vez con prompt mas especifico.|

---

*Fin del documento de diseno*