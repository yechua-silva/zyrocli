# Context MCP Bridge — Delta Spec

**Change**: zyro-architecture-mvp  
**PR**: 1 (foundation-skill-advisor)  
**Status**: active  

## What changed

This delta introduces the Context MCP Bridge as a new capability. The main spec at `openspec/specs/context-mcp-bridge/spec.md` was created as part of this change.

### Key additions

| Component | Description |
|-----------|-------------|
| `Bridge.Start()` | Spawns `context serve --libs` via exec.Command |
| `Bridge.Stop()` | SIGTERM → 5s grace → SIGKILL |
| `Bridge.QueryDocs()` | JSON-RPC query_docs over stdin/stdout |
| `Bridge.ResolveLibraryID()` | JSON-RPC resolve_library_id |
| `Bridge.sendRequest()` | Shared JSON-RPC framing (request + response) |

### Configuration

| Parameter | Default |
|-----------|---------|
| Binary | `context` (`npm i -g @neuledge/context`) |
| Protocol | JSON-RPC 2.0 over stdio |
| Query timeout | 30s |
| Stop grace period | 5s |

### Implementation references

- `internal/context/types.go` — QueryResult, LibraryID types
- `internal/context/bridge.go` — Bridge with full lifecycle and JSON-RPC
- `internal/context/bridge_test.go` — Pipe-based mock tests for JSON-RPC
