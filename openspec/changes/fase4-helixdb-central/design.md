# Design: Fase 4 — HelixDB Central Axis

## Technical Approach

Build a JSON-RPC 2.0 server over stdin/stdout in `internal/mcp/` that exposes HelixDB as MCP tools for OpenCode. The server reuses `internal/taskcontext`, `internal/db/helix`, and `internal/taskcontext/formatter` directly — no new HelixDB queries. Each tool handler creates its own `helix.Client` per request (stateless), calls existing functions, and returns formatted results.

## Package Architecture

```
internal/mcp/
├── server.go      — JSON-RPC 2.0 server (stdio transport, request loop)
├── handlers.go    — Tool dispatch + 3 handlers (task_context, search_code, search_skills)
├── types.go       — MCP request/response types, tool param structs
└── errors.go      — JSON-RPC error codes + typed errors
```

**Server binary**: `cmd/zyrocli/mcp.go` — thin `main()` that calls `mcp.Serve(os.Stdin, os.Stdout)`. Built via `go build -o zyrocli-mcp ./cmd/zyrocli/` or as subcommand `zyrocli mcp serve`.

## Key Types

```go
// types.go

// JSON-RPC 2.0 wire types
type Request struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      int             `json:"id"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params"`
}

type Response struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      int             `json:"id"`
    Result  json.RawMessage `json:"result,omitempty"`
    Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

// Tool params
type TaskContextParams struct {
    TaskID uint64 `json:"task_id"`
    Format string `json:"format,omitempty"` // text|json|prompt (default: text)
}

type SearchCodeParams struct {
    Query     string `json:"query"`
    ProjectID string `json:"project_id,omitempty"`
    Limit     int    `json:"limit,omitempty"` // default 10, max 50
}

type SearchSkillsParams struct {
    Query string `json:"query"`
    Limit int    `json:"limit,omitempty"` // default 10, max 50
}
```

## Tool Handlers

**Pattern**: Each handler is a function `func(ctx context.Context, params json.RawMessage) (interface{}, error)`. Server creates `helix.Client` per call (stateless), defers `Close()`.

| Handler | Input | Calls | Output |
|---------|-------|-------|--------|
| `task_context` | `TaskContextParams` | `helix.NewClient()` → `taskcontext.GetTaskContext()` → `tc.FormatText/JSON/Prompt()` | Formatted string or JSON |
| `search_code` | `SearchCodeParams` | `helix.NewClient(opts...)` → `helix.TextSearch("CodeNode", query)` | `[]Node` as JSON |
| `search_skills` | `SearchSkillsParams` | `helix.NewClient()` → `helix.TextSearchGlobal("Skill", query)` + `helix.FindSharedSkills("Skill")` | merged+deduped `[]Node` as JSON |

**Error mapping**:
| Condition | JSON-RPC Code |
|-----------|---------------|
| Missing required param | -32602 |
| HelixDB unreachable | -32000 "HelixDB connection failed" |
| Task not found | -32000 "task not found" |
| Timeout (>30s) | -32000 "request timed out" |
| Unknown method | -32601 |

## Sequence Flow

```
OpenCode                     MCP Server                    HelixDB
   │                              │                            │
   │──── stdin JSON-RPC ─────────→│                            │
   │   {"method":"tools/call",    │                            │
   │    "params":{"name":         │                            │
   │     "task_context",...}}     │                            │
   │                              │── helix.NewClient() ──────→│
   │                              │── GetTaskContext(1) ───────→│
   │                              │                            │
   │                              │←── TaskContext ────────────│
   │                              │── tc.FormatText()          │
   │←── stdout JSON-RPC ─────────│                            │
   │   {"result":"Context for     │                            │
   │    Task #1: ..."}            │                            │
```

## OpenCode Registration

Add to `~/.config/opencode/opencode.json`:

```json
{
  "mcpServers": {
    "zyro-helix": {
      "command": "zyrocli-mcp",
      "args": ["mcp", "serve"],
      "env": {
        "HELIX_URL": "http://localhost:6969"
      }
    }
  }
}
```

Server reads `HELIX_URL` from env (fallback `http://localhost:6969`), reads `HELIX_PROJECT_ID` optionally for project scoping.

## Migration / Rollout

**Phase 1**: Build MCP server, register in opencode.json, verify tools work from OpenCode.

**Phase 2**: Add deprecation warning to `cmd/zyrocli/context.go` — emit `stderr` message "DEPRECATED: use MCP tool task_context via OpenCode" before executing. Logic unchanged.

**Phase 3**: After 1 month, remove `zyrocli context` subcommand (optional — depends on usage).

**Rollback**: `git checkout HEAD -- internal/mcp/ cmd/zyrocli/context.go`, revert opencode.json.

## Resolved Decisions

| Decision | Choice | Alternatives | Rationale |
|----------|--------|-------------|-----------|
| Transport | JSON-RPC 2.0 over stdio | gRPC, HTTP | Matches `bridge.go` pattern; OpenCode MCP convention; zero config |
| Bridge.go | Unchanged | Refactor to share types | Bridge talks to `context` binary for docs; MCP talks to HelixDB; different concerns |
| task_context atomicity | Single tool returns all 6 traversals | Decomposed tools (get_skills, get_codenodes, etc.) | Agent needs full context in one call; decomposition adds roundtrips for no benefit |
| Client lifecycle | New per request (stateless) | Singleton with reconnect | Simpler; HelixDB client is cheap; avoids stale connection state |
| No new deps | stdlib only (encoding/json, bufio, io) | MCP SDK, third-party RPC | Keep binary small; JSON-RPC 2.0 is 20 lines of code |

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/mcp/server.go` | Create | JSON-RPC 2.0 server loop over stdio |
| `internal/mcp/handlers.go` | Create | Tool dispatch + 3 handlers |
| `internal/mcp/types.go` | Create | Request/response/param types |
| `internal/mcp/errors.go` | Create | Error codes and typed errors |
| `internal/mcp/server_test.go` | Create | Unit tests for server + handlers |
| `cmd/zyrocli/mcp.go` | Create | CLI entry point: `zyrocli mcp serve` |
| `cmd/zyrocli/context.go` | Modify | Add deprecation stderr warning |
| `opencode.json` | Modify | Register MCP server |

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | Handler param validation, error mapping | Table-driven tests with mock helix.Client |
| Integration | Full JSON-RPC request→response over pipe | `io.Pipe()` + `json.Encoder/Decoder` |
| E2E | MCP server starts, responds to tools/call | Build binary, send JSON-RPC via stdin, assert stdout |

## Open Questions

None — all decisions resolved.
