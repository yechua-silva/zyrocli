# Design: HelixDB Integration — 3-Layer Hybrid Architecture

## Technical Approach

Replace the Go-based MCP server (`internal/mcp/`) with a 3-layer hybrid architecture: (1) direct HTTP from agent, (2) Python MCP tools via PydanticAI, (3) Go SDK wrapping the official HelixDB Go client (`github.com/helixdb/helix-db/sdks/go`). The official SDK uses a dynamic query DSL posting to `/v1/query` — our Go layer wraps this in a higher-level CRUD API. Python tools use `httpx` for HTTP calls and `uv run` for execution.

## Architecture Decisions

| Decision | Choice | Alternatives | Rationale |
|----------|--------|-------------|-----------|
| Go client approach | Wrap official SDK (`helixdb/helix-db/sdks/go`) in higher-level API | Raw `net/http` wrapper | SDK handles serialization, error mapping, conflict retry; avoids reinventing the wheel |
| Python execution | `uv run mcp-tools/runner.py` | pip/venv, Docker | Matches spec R-MCP-006; uv is faster, simpler, no venv activation needed |
| Client lifecycle | Stateless — new `Client` per command invocation | Connection pool | CLI is short-lived; pool adds complexity with no benefit |
| Error mapping | Wrap official `*HelixError` into typed sentinel errors (`ErrNotFound`, `ErrConnectionFailed`, `ErrInvalidRequest`) | Return raw SDK errors | Callers need predictable error types per spec R-HELIX-005 |
| TaskContext traversals | 6 sequential `client.Exec()` calls (skills→code→docs→patterns→dependents→dependencies) | Parallel batch query | Sequential is simpler, debuggable; parallel can be added later if perf matters |
| Scope isolation | `x-project-id` header via `WithProjectID` option | Per-query filter | Single option sets header for all calls; matches Capa 1 spec |

## Data Flow

```
Agent (direct HTTP)
  │  POST /v1/query + x-project-id
  ▼
HelixDB ◄──── Go SDK (internal/db/helix/) ──── zyrocli context/cmd
  ▲                    │
  │                    ▼
  │            internal/taskcontext/
  │                    │
  └──── Python MCP ────┘
         (mcp-tools/)
         httpx → POST /v1/query
         registered in opencode.json
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/mcp/server.go` | Delete | Go MCP server no longer needed |
| `internal/mcp/handlers.go` | Delete | Tool handlers move to Python MCP tools |
| `internal/mcp/types.go` | Delete | MCP types no longer needed |
| `internal/mcp/errors.go` | Delete | MCP error codes no longer needed |
| `internal/mcp/server_test.go` | Delete | Tests for deleted code |
| `cmd/zyrocli/mcp.go` | Delete | MCP cobra command removed |
| `internal/db/helix/helix.go` | Modify | Wrap official SDK: Node/Edge CRUD, TextSearch, VectorSearch, Schema, typed errors |
| `internal/db/helix/errors.go` | Create | Sentinel errors: ErrNotFound, ErrConnectionFailed, ErrInvalidRequest, ErrTaskNotFound |
| `internal/db/helix/types.go` | Create | Node, Edge, SearchResult, IndexSpec structs |
| `internal/taskcontext/taskcontext.go` | Modify | Implement 6 traversals + 3 formatters |
| `cmd/zyrocli/context.go` | Modify | Keep existing deprecation warning + `--format` flag |
| `mcp-tools/runner.py` | Create | PydanticAI MCP server entry point |
| `mcp-tools/helix_client.py` | Create | httpx wrapper for HelixDB HTTP API |
| `mcp-tools/task_context.py` | Create | `task_context(id)` MCP tool |
| `mcp-tools/search_code.py` | Create | `search_code(query, limit)` MCP tool |
| `mcp-tools/search_skills.py` | Create | `search_skills(query, limit)` MCP tool |
| `mcp-tools/pyproject.toml` | Create | uv project config with pydantic-ai, httpx deps |

## Interfaces / Contracts

### Go SDK wrapper (`internal/db/helix/`)

```go
// errors.go
var (
    ErrNotFound         = errors.New("helix: not found")
    ErrConnectionFailed = errors.New("helix: connection failed")
    ErrInvalidRequest   = errors.New("helix: invalid request")
)

// types.go
type Node struct {
    ID         int64            `json:"id"`
    Type       string           `json:"type"`
    Properties map[string]any   `json:"properties,omitempty"`
}

type Edge struct {
    ID         int64            `json:"id"`
    SourceID   int64            `json:"source_id"`
    TargetID   int64            `json:"target_id"`
    Relation   string           `json:"relation"`
    Properties map[string]any   `json:"properties,omitempty"`
}

type SearchResult struct {
    Nodes []*Node `json:"nodes"`
    Score float64 `json:"score,omitempty"`
}

// helix.go — wraps official SDK
func (c *Client) CreateNode(ctx context.Context, label string, props map[string]any) (*Node, error)
func (c *Client) GetNode(ctx context.Context, label string, id int64) (*Node, error)
func (c *Client) UpdateNode(ctx context.Context, label string, id int64, props map[string]any) (*Node, error)
func (c *Client) DeleteNode(ctx context.Context, label string, id int64) error
func (c *Client) CreateEdge(ctx context.Context, fromID, toID int64, label string, props map[string]any) (*Edge, error)
func (c *Client) GetOutgoing(ctx context.Context, nodeID int64, edgeLabel string) ([]*Edge, error)
func (c *Client) GetIncoming(ctx context.Context, nodeID int64, edgeLabel string) ([]*Edge, error)
func (c *Client) DeleteEdge(ctx context.Context, edgeID int64) error
func (c *Client) TextSearch(ctx context.Context, label, property, query string, limit int) ([]*Node, error)
func (c *Client) VectorSearch(ctx context.Context, label, property string, vector []float32, k int) ([]*Node, error)
func (c *Client) CreateIndex(ctx context.Context, spec IndexSpec) error
func (c *Client) ListIndexes(ctx context.Context) ([]IndexSpec, error)
```

### TaskContext (`internal/taskcontext/`)

```go
type TaskContext struct {
    TaskID       uint64     `json:"task_id"`
    Skills       []*Node    `json:"skills"`
    Code         []*Node    `json:"code"`
    Docs         []*Node    `json:"docs"`
    Patterns     []*Node    `json:"patterns"`
    Dependents   []*Node    `json:"dependents"`
    Dependencies []*Node    `json:"dependencies"`
    Errors       []error    `json:"errors,omitempty"` // partial failures
}

func GetTaskContext(ctx context.Context, client *helix.Client, taskID uint64) (*TaskContext, error)
func (tc *TaskContext) FormatJSON() (string, error)
func (tc *TaskContext) FormatPrompt() string
func (tc *TaskContext) FormatText() string
```

### Python MCP tools

```python
# runner.py — PydanticAI MCP server
# Registers: task_context, search_code, search_skills

# helix_client.py
class HelixClient:
    def __init__(self, base_url: str = "http://localhost:6969"): ...
    async def query(self, gql: str, params: dict = None) -> dict: ...

# Registration in opencode.json
{
  "mcpTools": {
    "helix-integration": {
      "command": "uv",
      "args": ["run", "mcp-tools/runner.py"]
    }
  }
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Go SDK wrapper methods (error mapping, types) | Table-driven tests with mock HTTP responses |
| Unit | TaskContext formatters (JSON, Prompt, Text) | Golden file tests with fixture data |
| Integration | Go SDK against live HelixDB | Skip if `HELIX_URL` unreachable; use build tag `//go:build integration` |
| E2E | Python MCP tools via `uv run` | Manual verification: invoke tool, assert JSON response shape |

## Migration / Rollout

No data migration required. The rollout is:

1. Delete `internal/mcp/` + `cmd/zyrocli/mcp.go` (safe — no callers outside this project)
2. Implement Go SDK wrapper + TaskContext (backward-compatible with existing `context.go` command)
3. Build Python MCP tools + register in `opencode.json`
4. `zyrocli context <id>` continues to work with deprecation warning

## Open Questions

- [ ] HelixDB query language for traversals: confirm the exact DSL syntax for 6-traversal task context (skills, code, docs, patterns, dependents, dependencies) — needs HelixDB schema knowledge
- [ ] Python MCP tool error format: confirm HelixDB HTTP error shape for structured error responses in R-MCP-005
