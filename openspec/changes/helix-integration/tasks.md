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

- [ ] A1. Delete `internal/mcp/server.go` (102 lines)
- [ ] A2. Delete `internal/mcp/handlers.go` (177 lines)
- [ ] A3. Delete `internal/mcp/types.go` (46 lines)
- [ ] A4. Delete `internal/mcp/errors.go` (18 lines)
- [ ] A5. Delete `internal/mcp/server_test.go` (371 lines)
- [ ] A6. Delete `cmd/zyrocli/mcp.go` (41 lines)
- [ ] A7. Verify build: `go build ./...` compiles without errors
- [ ] A8. Commit: `chore: delete internal/mcp/ Go MCP server (superseded by Python MCP)`

## Phase B: Go SDK Implementation

- [ ] B1. Create `internal/db/helix/errors.go` — sentinel errors (ErrNotFound, ErrConnectionFailed, ErrInvalidRequest, ErrTaskNotFound) (~15 lines)
- [ ] B2. Create `internal/db/helix/types.go` — Node, Edge, SearchResult, IndexSpec structs with JSON tags (~55 lines)
- [ ] B3. Implement `internal/db/helix/helix.go` — wrap official SDK (`github.com/helixdb/helix-db/sdks/go`): NewClient with health check, CreateNode, GetNode, UpdateNode, DeleteNode, CreateEdge, GetOutgoing, GetIncoming, DeleteEdge, TextSearch, VectorSearch, CreateIndex, ListIndexes; map SDK errors to sentinels (~180 lines)
- [ ] B4. Implement `internal/taskcontext/taskcontext.go` — TaskContext struct (Skills, Code, Docs, Patterns, Dependents, Dependencies fields), GetTaskContext with 6 sequential `client.Exec()` calls, FormatJSON (indented), FormatPrompt (section headers), FormatText (human-readable) (~120 lines)
- [ ] B5. Create `internal/db/helix/helix_test.go` — table-driven tests for error mapping with mock HTTP responses; build tag `//go:build integration` for live tests (~90 lines)
- [ ] B6. Create `internal/taskcontext/taskcontext_test.go` — golden file tests for FormatJSON, FormatPrompt, FormatText with fixture TaskContext data (~90 lines)
- [ ] B7. Update `cmd/zyrocli/context.go` — verify deprecation warning + `--format` flag still work with real SDK; no logic changes needed (existing code already compatible) (~0 lines, verify only)
- [ ] B8. Run `go mod tidy` to fetch `github.com/helixdb/helix-db/sdks/go`
- [ ] B9. Verify build: `go build ./...` and `go test ./internal/db/helix/ ./internal/taskcontext/`

## Phase C: Python MCP Tools

- [ ] C1. Create `mcp-tools/pyproject.toml` — uv project config with deps: pydantic-ai, httpx (~20 lines)
- [ ] C2. Create `mcp-tools/helix_client.py` — HelixClient class wrapping httpx for HelixDB HTTP API; methods: query(gql, params), health_check() (~80 lines)
- [ ] C3. Create `mcp-tools/runner.py` — PydanticAI MCP server entry point; registers task_context, search_code, search_skills tools (~40 lines)
- [ ] C4. Create `mcp-tools/task_context.py` — @mcp.tool() task_context(id: int) → calls HelixClient, returns 6-section JSON (~50 lines)
- [ ] C5. Create `mcp-tools/search_code.py` — @mcp.tool() search_code(query: str, limit: int = 10) → text search on CodeNode (~40 lines)
- [ ] C6. Create `mcp-tools/search_skills.py` — @mcp.tool() search_skills(query: str, limit: int = 10) → text search + shared skills merge (~40 lines)
- [ ] C7. Create `mcp-tools/README.md` — registration docs for `~/.config/opencode/opencode.json` mcpTools.helix-integration config (~30 lines)
- [ ] C8. Verify: `uv run mcp-tools/runner.py` starts without import errors

## Phase D: Verification & Documentation

- [ ] D1. Run full Go test suite: `go test ./...`
- [ ] D2. Verify `zyrocli context <id>` prints deprecation warning and works
- [ ] D3. Manual E2E: invoke each Python MCP tool via `uv run` against live HelixDB
- [ ] D4. Update `openspec/changes/helix-integration/` artifacts with final status
