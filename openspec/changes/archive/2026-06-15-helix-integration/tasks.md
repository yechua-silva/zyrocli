# Tasks: HelixDB Integration — 3-Layer Hybrid Architecture

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 750–900 (additions only); 1,500+ net (incl. 755 deletions) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1: Cleanup → PR 2: Go SDK → PR 3: Python MCP |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Delete old Go MCP server | PR 1 | base: main; pure deletions, ~755 lines removed; safe merge |
| 2 | Go SDK + TaskContext + context.go | PR 2 | base: main (after PR 1 merges); ~450 lines new Go code |
| 3 | Python MCP tools + registration | PR 3 | base: main (after PR 2 merges); ~300 lines new Python code |

## Phase A: Cleanup — Delete Old Go MCP Server

- [x] A1. Delete `internal/mcp/server.go` (102 lines)
- [x] A2. Delete `internal/mcp/handlers.go` (177 lines)
- [x] A3. Delete `internal/mcp/types.go` (46 lines)
- [x] A4. Delete `internal/mcp/errors.go` (18 lines)
- [x] A5. Delete `internal/mcp/server_test.go` (371 lines)
- [x] A6. Delete `cmd/zyrocli/mcp.go` (41 lines)
- [x] A7. Verify build: `go build ./...` compiles without errors
- [x] A8. Commit: `chore: delete internal/mcp/ Go MCP server (superseded by Python MCP)`

## Phase B: Go SDK Implementation

- [x] B1. Create `internal/db/helix/errors.go` — sentinel errors (ErrNotFound, ErrConnectionFailed, ErrInvalidRequest, ErrTaskNotFound) (~15 lines)
- [x] B2. Create `internal/db/helix/types.go` — Node, Edge, SearchResult, IndexSpec structs with JSON tags (~55 lines)
- [x] B3. Implement `internal/db/helix/helix.go` — net/http client with all CRUD, search, and schema methods; mapped to sentinel errors (~425 lines)
- [x] B4. Implement `internal/taskcontext/taskcontext.go` — TaskContext struct + GetTaskContext with 6 traversals + 3 formatters (~181 lines)
- [x] B5. Create `internal/db/helix/helix_test.go` — table-driven error mapping + functional tests with httptest (~320 lines)
- [x] B6. Create `internal/taskcontext/taskcontext_test.go` — golden file tests for all 3 formatters with fixture data (~85 lines)
- [x] B7. Verify `cmd/zyrocli/context.go` — deprecation warning + `--format` flag compile and work with new API
- [x] B8. Run `go mod tidy` — no new dependencies needed (net/http + encoding/json only)
- [x] B9. Verify build: `go build ./...` and `go test ./...` — all passing

## Phase C: Python MCP Tools

- [x] C1. Create `mcp-tools/pyproject.toml` — uv project config with deps: pydantic-ai, httpx (12 lines)
- [x] C2. Create `mcp-tools/helix_client.py` — HelixClient class wrapping httpx for HelixDB HTTP API; methods: query(payload), health(), text_search(), get_node(), get_outgoing(), get_incoming (99 lines)
- [x] C3. Create `mcp-tools/runner.py` — FastMCP server entry point; registers task_context, search_code, search_skills tools via server.add_tool() (26 lines)
- [x] C4. Create `mcp-tools/task_context.py` — async task_context_tool(id: int) → calls HelixClient, returns 6-section JSON with error handling (100 lines)
- [x] C5. Create `mcp-tools/search_code.py` — async search_code_tool(query: str, limit: int = 10) → text search on CodeNode (31 lines)
- [x] C6. Create `mcp-tools/search_skills.py` — async search_skills_tool(query: str, limit: int = 10) → text search on Skill nodes (31 lines)
- [x] C7. Create `mcp-tools/README.md` — registration docs for `~/.config/opencode/opencode.json` mcpTools.helix-integration config (39 lines)
- [x] C8. Verify: `uv run -> python -c` import check passes; `uv run runner.py` starts on stdio cleanly

## Phase D: Verification & Documentation

- [x] D1. Run full Go test suite: `go test ./...`
- [x] D2. Verify `zyrocli context <id>` prints deprecation warning and works
- [x] D3. Manual E2E: invoke each Python MCP tool via `uv run` against live HelixDB
- [x] D4. Update `openspec/changes/helix-integration/` artifacts with final status
