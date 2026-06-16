# Exploration: fase4-helixdb-central

## Current State

### Bridge truth (THE KEY QUESTION)

`internal/context/bridge.go` line 19:
```go
bridgeBinary = "context"
```

The Bridge executes the binary `context` with arguments `serve --libs` (line 68):
```go
cmd := exec.CommandContext(ctx, bridgeBinary, "serve", "--libs")
```

**Protocol**: JSON-RPC 2.0 over stdin/stdout. Two methods:
- `query_docs(library_id, query)` — queries library documentation
- `resolve_library_id(package_name)` — resolves a package name to canonical library ID

**THIS IS NOT CONTEXT7**. The binary is `context` — a local GitMCP binary. The previous exploration was WRONG. There is no Context7 dependency anywhere in the codebase:
- `go.mod`: no Context7 package
- Zero references to "context7" in any `.go` files (only in docs as deprecated plans)
- The `internal/context/types.go` comment on line 12 explicitly says: `LibraryID represents a Context MCP library identifier (the local 'context' binary).`

### Investigation truth

`internal/investigation/research.go` does NOT import or use the bridge at all. The `ResearchEngine` accepts a `DocQueryFn` callback (line 138) and a `WebFetchFn` callback (line 141). The bridge is a separate concern. The investigation engine is currently unused by any CLI command — it's a standalone engine for running concurrent doc queries + web fetches + git analysis.

### HelixDB Client — Complete API surface

`internal/db/helix/client.go` wraps `helixsdk.Client` with project-level isolation (133 lines). Methods across all files:

**client.go**: `NewClient`, `Close`, `Ping`, `InjectProject`, `Inner`, `ProjectID`
**nodes.go** (590 lines): `CreateNode`, `GetNode`, `UpdateNode`, `DeleteNode`, `FindNodes`, `FindSharedSkills`, `CreateSkill`, `LinkSkillToProject`, `GetProjectSkills`, `UpsertSkill`, `UpsertCodeNode`, `GetCodeNodesByProject`, `LinkTaskToCodeNodes`
**edges.go** (180 lines): `CreateEdge`, `GetOutgoing`, `GetIncoming`, `DeleteEdge`
**search.go** (217 lines): `VectorSearch`, `TextSearch`, `VectorSearchGlobal`, `TextSearchGlobal`
**schema.go** (86 lines): `InitSchema` (idempotent index creation)
**errors.go**: `ErrNotFound`, `ErrConnection`, `ErrUnauthorized`, `ErrNotImplemented`

### Task context queries

`internal/taskcontext/queries.go` makes HelixDB graph traversals:
1. `GetNode(taskID)` — fetch task node
2. `GetOutgoing(taskID, "REQUIRES")` — skills needed
3. `GetOutgoing(taskID, "REFERENCES")` — CodeNodes referenced
4. `GetIncoming(taskID, "HAS_TASK")` — parent project
5. `GetOutgoing(projectID, "HAS_DOC")` — project documents
6. `GetOutgoing(projectID, "HAS_PATTERN")` — project patterns

3 formats: `FormatText`, `FormatJSON`, `FormatPrompt`

### CLI commands

`cmd/zyrocli/context.go`: `zyrocli context [task-id]` — connects to HelixDB, calls `taskcontext.GetTaskContext`, formats output. This is the ONLY command that provides context to subagents. No MCP server exists yet.

## Affected Areas

- `internal/context/bridge.go` — Bridge launching `context` binary (deprecated per roadmap, to be replaced by Context + GitMCP)
- `internal/investigation/research.go` — DocQueryFn callback, currently unused by CLI
- `cmd/zyrocli/context.go` — The only context provider, to be wrapped by MCP tools
- `internal/taskcontext/` — HelixDB graph queries for task context (types, queries, formatter)
- `internal/db/helix/` — All HelixDB operations (client, nodes, edges, search, schema, errors)
- `cmd/zyrocli/task.go` — Task create/link/list commands

## What Changes for Fase 4

Based on `docs/roadmap.md` and `docs/architecture-v2.md`, Fase 4 has 4 items:

### 1. MCP Server (HIGH priority) — BUILD NEW
Create an MCP server that exposes tools:
- `task_context` — wraps `taskcontext.GetTaskContext`
- `search_code` — wraps `helix.Client.TextSearch`/`VectorSearch` for CodeNodes
- `search_skills` — wraps `helix.Client.VectorSearchGlobal`/`FindSharedSkills`

Architecture: MCP server runs as a separate process, internally calls HelixDB Go SDK.

### 2. Helix Skills install (MEDIUM) — CONFIGURATION
Install `helix-query-*` skills in the dev environment. Pure setup, no code.

### 3. Replace Context bridge with Context + GitMCP (MEDIUM) — MODIFY
The `context` binary bridge is deprecated. Replace with:
- `context` + GitMCP for library docs (same binary, different approach)
- Or remove the bridge entirely if MCP tools handle all context needs

### 4. Deprecate `zyrocli context` (LOW) — MODIFY
Once MCP tools exist, `zyrocli context` becomes redundant. Keep logic in MCP tools, deprecate the CLI command.

## Approaches

### 1. MCP Server as Go package (internal/mcp/)
- Build MCP server as a Go package using an MCP Go SDK
- Tools call HelixDB client directly
- Stdio transport (standard MCP protocol)
- **Pros**: Native Go, same module, direct SDK access
- **Cons**: New dependency (MCP Go SDK), more complex binary
- **Effort**: Medium

### 2. MCP Server as Python subprocess
- Use the existing Python os/exec pattern
- Python MCP SDK is more mature
- **Pros**: More MCP tooling available in Python
- **Cons**: Inconsistent with the Go codebase, adds Python dependency for a core feature
- **Effort**: Medium

### 3. HTTP API wrapper (not MCP)
- Expose HelixDB queries via a local HTTP API
- Agent queries directly (Capa 1 of hybrid model)
- **Pros**: Simpler, no MCP protocol needed
- **Cons**: No MCP tool integration, agent needs custom HTTP calls
- **Effort**: Low

## Recommendation

**Approach 1: MCP Server as Go package**. The project is Go-native, HelixDB Go SDK is already imported, and the hybrid model (HTTP direct + MCP tools + ZyroCLI) is the decided architecture. Build the MCP server in Go with stdio transport.

## Risks

- MCP Go SDK maturity — need to verify which Go MCP SDK is production-ready
- The `context` binary deprecation depends on whether Context + GitMCP actually replaces the functionality
- Task context queries are sequential (6 HelixDB calls per task) — may need parallelization for performance

## Ready for Proposal

Yes. The codebase state is clear:
1. Bridge is `context` binary (GitMCP), NOT Context7
2. Investigation engine is unused by CLI
3. HelixDB client is complete with all CRUD + search + schema
4. Task context queries work but only serve the CLI command
5. MCP server is the missing piece that connects everything
