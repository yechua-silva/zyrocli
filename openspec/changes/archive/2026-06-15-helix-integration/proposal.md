# Proposal: HelixDB Integration — 3-Layer Hybrid Architecture

## Intent

Replace the incorrect `internal/mcp/` Go-based MCP server with a correct 3-layer hybrid architecture for HelixDB integration. HelixDB is the central knowledge graph backing all MCP tools (task_context, search_code, search_skills). The old approach built a custom MCP server in Go — a wheel that doesn't need reinventing since HelixDB exposes HTTP API + SDKs natively.

## Scope

### In Scope
- Delete `internal/mcp/` (5 files) and `cmd/zyrocli/mcp.go`
- Implement real Go SDK client at `internal/db/helix/` (Connect, CRUD, search, schema)
- Implement real Go SDK at `internal/taskcontext/` (6 traversals: skills, code, docs, patterns, dependents, dependencies)
- Update `cmd/zyrocli/context.go` with deprecation warning + delegate to MCP tool
- Build Python MCP tools: `mcp-tools/task_context.py`, `mcp-tools/search_code.py`, `mcp-tools/search_skills.py` (PydanticAI)
- Configure global skills in `~/.config/opencode/opencode.json` + MCP tool registration

### Out of Scope
- Building a native MCP server for HelixDB (HelixDB already has HTTP API + SDKs)
- Implementing `internal/codeparse/` or `internal/git/` packages
- Migration of existing data from previous helix approaches
- Python MCP tool testing framework (manual verification only in this phase)
- Installation automation for Python venv or HelixDB itself (prerequisites)

## Capabilities

### New Capabilities
- `helix-db-client`: Go SDK client for HelixDB — connect, CRUD nodes/edges, text search, schema management
- `task-context`: Go SDK for task context queries — 6 graph traversals (skills, code, docs, patterns, dependents, dependencies) with JSON/prompt/text formatters
- `mcp-tools-python`: Python PydanticAI MCP tools — task_context, search_code, search_skills registered as OpenCode MCP tools
- `helix-agent-http`: Direct HTTP queries from agent to HelixDB POST /v1/query (Capa 1, no middleware)

### Modified Capabilities
- None (no existing specs to modify)

## Approach

1. **Delete**: Remove `internal/mcp/` directory and `cmd/zyrocli/mcp.go` (the old Go MCP server approach)
2. **Implement Go SDK (Capa 3)**: Flesh out `internal/db/helix/` with real HelixDB HTTP client using Go SDK (`github.com/helixdb/helix-db/sdks/go`). Implement full CRUD for nodes/edges, text search, and schema operations in `internal/taskcontext/`.
3. **Build Python MCP tools (Capa 2)**: Create `mcp-tools/` directory with 3 PydanticAI-based MCP tools that query HelixDB via HTTP and return structured context to OpenCode.
4. **Configure**: Register MCP tools in `~/.config/opencode/opencode.json` and global skills. Update `cmd/zyrocli/context.go` with deprecation warning + delegate-to-MCP message.
5. **Merge to main** after every safe change batch.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/mcp/` | Removed | 5 files deleted (Go MCP server approach) |
| `cmd/zyrocli/mcp.go` | Removed | MCP cobra command deleted |
| `internal/db/helix/helix.go` | Modified | Stub → real HelixDB client (CRUD, search, schema) |
| `internal/taskcontext/taskcontext.go` | Modified | Stub → real 6-traversal queries |
| `cmd/zyrocli/context.go` | Modified | Add deprecation warning + delegate to MCP |
| `mcp-tools/` | New | 3 Python PydanticAI MCP tools |
| `~/.config/opencode/opencode.json` | Modified | MCP tools + global skills registration |
| `openspec/changes/helix-integration/` | New | SDD change artifact folder |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| HelixDB not running locally at `localhost:6969` | Medium | Document as prerequisite; add clear error message in Go SDK |
| Python MCP tool maturity (PydanticAI ecosystem) | Medium | Pin versions in requirements.txt; test manually before registering |
| Go SDK API drift vs HelixDB HTTP API | Low | Use the official Go SDK; abstract behind interface for testability |
| Breaking existing context.go users | Low | Keep deprecation warning + backward-compatible output |

## Rollback Plan

1. Revert Go SDK changes: `git checkout HEAD -- internal/db/helix/ internal/taskcontext/ cmd/zyrocli/context.go`
2. Restore deleted files: `git checkout HEAD -- internal/mcp/ cmd/zyrocli/mcp.go`
3. Unregister MCP tools from `~/.config/opencode/opencode.json`
4. Delete `mcp-tools/` directory
5. Test that `zyrocli context <id>` works as before

## Dependencies

- HelixDB installed and running (prerequisite for all layers)
- Go SDK: `github.com/helixdb/helix-db/sdks/go` (Capa 3 writes)
- Python 3.10+ with `pydantic-ai` and `httpx` (Capa 2 MCP tools)
- OpenCode with MCP tool support (consumer of Capa 2)

## Success Criteria

- [ ] `internal/mcp/` and `cmd/zyrocli/mcp.go` deleted — no MCP server in Go
- [ ] `internal/db/helix/` implements real CRUD + search + schema operations against HelixDB HTTP API
- [ ] `internal/taskcontext/` implements 6 traversals and 3 output formatters
- [ ] `mcp-tools/task_context.py` returns task context from HelixDB via OpenCode MCP
- [ ] `mcp-tools/search_code.py` and `mcp-tools/search_skills.py` return search results
- [ ] MCP tools registered in `~/.config/opencode/opencode.json` and functional
- [ ] `zyrocli context <id>` prints deprecation warning and works as delegate
- [ ] All changes merged to main
