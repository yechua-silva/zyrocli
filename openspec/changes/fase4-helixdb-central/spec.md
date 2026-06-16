# Spec: Fase 4 — HelixDB Central Axis

- **project**: ZyroAgentCLI
- **change**: fase4-helixdb-central
- **artifact**: spec
- **status**: draft

## Domains

| Domain | Type | Location |
|--------|------|----------|
| mcp-server | New | `internal/mcp/` |
| helix-query-skills | New | Skills directory |
| zyrocli-context | Modified | `cmd/zyrocli/context.go` |
| context-mcp-bridge | Modified | `internal/context/bridge.go` |

---

## Domain: mcp-server

### Requirements

| ID | Requirement | Strength |
|----|-------------|----------|
| R-MCP-001 | Transport MUST be JSON-RPC 2.0 over stdin/stdout; server exits cleanly when stdin closes | MUST |
| R-MCP-002 | `task_context(task_id: uint64, format?: string)` MUST return TaskContext via `GetTaskContext()`. Formats: text/json/prompt (default text) | MUST |
| R-MCP-003 | `search_code(query: string, project_id?: string, limit?: int)` MUST return CodeNodes via `helix.TextSearch()`. Default limit 10, max 50 | MUST |
| R-MCP-004 | `search_skills(query: string, limit?: int)` MUST return Skills via `helix.TextSearchGlobal()` + `FindSharedSkills()`. Default limit 10, max 50 | MUST |
| R-MCP-005 | HelixDB unreachable MUST return JSON-RPC error -32000 "HelixDB connection failed" (no crash) | MUST |
| R-MCP-006 | Invalid params MUST return JSON-RPC error -32602 | MUST |
| R-MCP-007 | Each tool handler MUST enforce 30s timeout; timeout returns -32000 "request timed out" | MUST |
| R-MCP-008 | Server MUST register in opencode.json via `mcpServers.command` mode | MUST |

### Scenarios

**task_context existing task**: GIVEN HelixDB running with task #1; WHEN `task_context(1)`; THEN formatted context with skills, codenodes, documents, patterns.

**task_context not found**: GIVEN no task #999; WHEN `task_context(999)`; THEN error -32000 "task not found".

**task_context missing param**: GIVEN server running; WHEN `task_context()` without params; THEN error -32602.

**search_code all projects**: GIVEN CodeNodes exist; WHEN `search_code("auth")`; THEN max 10 matching results.

**search_code scoped**: GIVEN multi-project; WHEN `search_code("handler","my-project")`; THEN results scoped to project_id.

**search_skills**: GIVEN global Skills; WHEN `search_skills("react")`; THEN Skills with name, type, source_url.

**HelixDB down**: GIVEN HelixDB not running; WHEN any tool; THEN error -32000 (no crash, server continues).

---

## Domain: helix-query-skills

### Requirements

| ID | Requirement | Strength |
|----|-------------|----------|
| R-SKL-001 | Skills `helix-query-code`, `helix-query-skills`, `helix-query-context` MUST be installable via command | MUST |
| R-SKL-002 | Each skill MUST verify HelixDB reachable before executing; unreachable → stderr error + exit 1 | MUST |
| R-SKL-003 | Each skill MUST output results as JSON | MUST |

### Scenarios

**Install**: GIVEN HelixDB running; WHEN install command runs; THEN 3 skill files exist.

**HelixDB unreachable**: GIVEN HelixDB down; WHEN skill runs; THEN stderr "HelixDB not reachable" + exit 1.

---

## Domain: zyrocli-context (MODIFIED)

### Requirements

| ID | Requirement | Strength |
|----|-------------|----------|
| R-CTX-001 | `zyrocli context <id>` MUST print stderr warning "DEPRECATED: use MCP tool task_context via OpenCode" | MUST |
| R-CTX-002 | After warning, MUST attempt MCP delegation; MCP binary not found → fallback to direct HelixDB query | MUST |
| R-CTX-003 | `--format` flag MUST continue working during deprecation | MUST |

### Scenarios

**Warning**: GIVEN `zyrocli context 1`; WHEN runs; THEN stderr shows deprecation warning; stdout shows normal output.

**MCP delegation**: GIVEN MCP binary in PATH; AFTER warning; THEN delegates via JSON-RPC to MCP server.

**Fallback**: GIVEN MCP binary NOT in PATH; AFTER warning; THEN direct HelixDB query (unchanged).

---

## Domain: context-mcp-bridge (MODIFIED)

### Requirements

| ID | Requirement | Strength |
|----|-------------|----------|
| R-BRG-001 | bridge.go MUST remain unchanged (zero lines changed) | MUST |
| R-BRG-002 | MCP server and context bridge are separate processes, no stdio conflict | MUST |

### Scenario

**Coexistence**: GIVEN both bridge.go and MCP server running; WHEN bridge sends JSON-RPC to `context` binary; THEN response from `context` binary, not MCP server.

---

## Out of Scope

- Write operations via MCP (writes stay in `zyrocli` CLI)
- Community detection in HelixDB
- TUI/web for context
- Non-Go language parsing
- Replacing `openspec/`
- Modifying or removing bridge.go
- Replacing Context7 (never existed as code — only as architectural reference)
