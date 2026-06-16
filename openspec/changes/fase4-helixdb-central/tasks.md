# Tasks: Fase 4 — HelixDB Central Axis

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~520 (additions only — no deletions except context.go minor) |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (foundation + handlers) → PR 2 (CLI + tests + deprecation) |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | MCP server foundation: types, errors, server, handlers | PR 1 | base: feature/fase4-mcp; includes all `internal/mcp/` files |
| 2 | CLI integration, tests, deprecation, opencode.json | PR 2 | base: PR 1 branch; thin entry point + test coverage + deprecation |

---

## Phase 1: MCP Server Foundation

- [ ] 1.1 Create `internal/mcp/types.go` — JSON-RPC 2.0 wire types (`Request`, `Response`, `RPCError`) + 3 tool param structs (`TaskContextParams`, `SearchCodeParams`, `SearchSkillsParams`). Reference: `internal/context/bridge.go` lines 23-35 for JSON-RPC message shape. ~60 lines.
- [ ] 1.2 Create `internal/mcp/errors.go` — JSON-RPC error code constants (`ErrCodeInvalidParams = -32602`, `ErrCodeMethodNotFound = -32601`, `ErrCodeServerError = -32000`) + `RPCErrorf(code int, msg string) *RPCError` constructor. ~30 lines.
- [ ] 1.3 Create `internal/mcp/server.go` — `Serve(r io.Reader, w io.Writer) error` function. Server reads newline-delimited JSON from `r` via `json.Decoder`, dispatches to handler map, writes response to `w` via `json.Encoder`. Handles `tools/call` method, unknown methods return -32601. Include 30s per-request timeout via `context.WithTimeout`. ~100 lines.
- [ ] 1.4 Create `internal/mcp/handlers.go` — Handler type `func(ctx context.Context, params json.RawMessage) (interface{}, error)`. Implement `handleTaskContext`: unmarshal `TaskContextParams`, create `helix.NewClient(ctx)` with env `HELIX_URL`/`HELIX_PROJECT_ID`, call `taskcontext.GetTaskContext()`, format via `tc.FormatText()`/`FormatJSON()`/`FormatPrompt()` based on `format` param. Implement `handleSearchCode`: unmarshal `SearchCodeParams`, create client, call `client.TextSearch(ctx, "CodeNode", query)`, enforce limit (default 10, max 50). Implement `handleSearchSkills`: unmarshal `SearchSkillsParams`, create client, call `client.TextSearchGlobal(ctx, "Skill", query)` + `client.FindSharedSkills(ctx, "Skill")`, merge+dedup, enforce limit. ~120 lines.

## Phase 2: CLI Integration

- [ ] 2.1 Create `cmd/zyrocli/mcp.go` — Cobra subcommand `mcp serve` that calls `mcp.Serve(os.Stdin, os.Stdout)`. Register with `rootCmd`. ~30 lines.
- [ ] 2.2 Modify `cmd/zyrocli/context.go` — Add deprecation warning to stderr (`fmt.Fprintln(cmd.ErrOrStderr(), "⚠ DEPRECATED: use MCP tool task_context via OpenCode")`) before existing logic. Keep `--format` flag working. ~15 lines added.

## Phase 3: Testing

- [ ] 3.1 Create `internal/mcp/server_test.go` — Test `handleTaskContext` with mock helix.Client (table-driven: valid task, missing param, task not found). Test `handleSearchCode` (valid query, limit enforcement). Test `handleSearchSkills` (valid query, dedup). Test server dispatch (unknown method → -32601). Integration test: full JSON-RPC request→response over `io.Pipe()`. ~150 lines.

## Phase 4: Configuration

- [ ] 4.1 Update `~/.config/opencode/opencode.json` — Add `mcpServers.zyro-helix` entry with `command: "zyrocli-mcp"`, `args: ["mcp", "serve"]`, `env: {"HELIX_URL": "http://localhost:6969"}`. Document in design.md or README.

---

## Dependencies

```
1.1 (types.go) ──┐
1.2 (errors.go) ─┤
                 ├──→ 1.3 (server.go) ──→ 1.4 (handlers.go) ──→ 2.1 (mcp.go) ──→ 3.1 (tests)
                 │
                 └──→ 2.2 (context.go deprecation) [independent]
```

Phase 1 is fully independent. Phase 2 depends on Phase 1. Phase 3 depends on Phases 1-2. Phase 4 is independent but final.
