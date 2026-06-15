# HelixDB Integration — Consolidated Specification

**Change**: helix-integration
**Mode**: New capabilities (no existing specs to modify)
**Artifact store**: openspec

---

## 1. helix-db-client — Go SDK Client

**Purpose**: Stateless Go client for HelixDB HTTP API at `internal/db/helix/`.

### Requirements

| ID | Requirement | Strength |
|----|-------------|----------|
| R-HELIX-001 | Client MUST connect to HelixDB via HTTP at configurable base URL (default `http://localhost:6969`) | MUST |
| R-HELIX-002 | Client MUST support CRUD operations for nodes and edges | MUST |
| R-HELIX-003 | Client MUST support text search and vector search on nodes | MUST |
| R-HELIX-004 | Client MUST support schema management (create/list indexes) | MUST |
| R-HELIX-005 | Client MUST return typed errors: `ErrNotFound`, `ErrConnectionFailed`, `ErrInvalidRequest` | MUST |
| R-HELIX-006 | Client MUST implement option-based functional options pattern for configuration | MUST |
| R-HELIX-007 | Client MUST be stateless — each call creates its own HTTP request | MUST |

### Scenarios

#### R-HELIX-001: Connect with health check
- GIVEN HelixDB running at `http://helix:6969`
- WHEN `NewClient(ctx, WithBaseURL("http://helix:6969"))` is called
- THEN client initializes successfully without error

#### R-HELIX-001: Connection failure on unreachable server
- GIVEN no server at `http://localhost:9999`
- WHEN any CRUD operation is called
- THEN error wrapping `ErrConnectionFailed` is returned
- AND error message includes the configured URL

#### R-HELIX-002: Create and retrieve a node
- GIVEN a valid client scoped to project "my-project"
- WHEN `CreateNode(ctx, kind, props)` returns a node with ID
- THEN `GetNode(ctx, kind, id)` returns the same node with all properties

#### R-HELIX-002: GetNode returns ErrNotFound
- GIVEN a node ID that does not exist
- WHEN `GetNode(ctx, kind, nonexistentID)` is called
- THEN error wrapping `ErrNotFound` is returned

#### R-HELIX-003: Text search returns ranked results
- GIVEN nodes with text content matching "error handling"
- WHEN `TextSearch(ctx, "node_type", "error handling", limit=10)` is called
- THEN results are returned ranked by relevance score
- AND result count does not exceed limit

#### R-HELIX-005: Edge creation with properties
- GIVEN two existing nodes (source, target)
- WHEN `CreateEdge(ctx, sourceID, targetID, relation, props)` is called
- THEN a new edge is created linking source to target
- AND edge includes the provided properties and relation label

---

## 2. task-context — Go SDK Task Context

**Purpose**: Query HelixDB for full task context using 6 graph traversals at `internal/taskcontext/`.

### Requirements

| ID | Requirement | Strength |
|----|-------------|----------|
| R-TASK-001 | `GetTaskContext` MUST traverse 6 relations: skills, code, docs, patterns, dependents, dependencies | MUST |
| R-TASK-002 | `FormatJSON()` MUST output indented JSON of all 6 traversal results | MUST |
| R-TASK-003 | `FormatPrompt()` MUST output structured prompt with section headers per traversal | MUST |
| R-TASK-004 | `FormatText()` MUST output human-readable summary with counts per traversal | MUST |
| R-TASK-005 | `GetTaskContext` MUST return `ErrTaskNotFound` when task node does not exist | MUST |
| R-TASK-006 | On partial traversal failure, MUST return partial results plus list of failed traversals | SHOULD |

### Scenarios

#### R-TASK-001: Full 6-traversal success
- GIVEN task 42 exists in HelixDB with skills, code nodes, docs, patterns, dependents, and dependencies
- WHEN `GetTaskContext(ctx, client, 42)` is called
- THEN all 6 traversals complete without error
- AND returned `TaskContext` contains non-empty arrays for each traversal type

#### R-TASK-005: Task not found
- GIVEN no task node with ID 999
- WHEN `GetTaskContext(ctx, client, 999)` is called
- THEN `ErrTaskNotFound` is returned
- AND no partial data is returned

#### R-TASK-003: Prompt output format
- GIVEN a populated TaskContext with skills ["golang", "testing"] and code nodes [2 entries]
- WHEN `FormatPrompt()` is called
- THEN output contains `## Skills`, `## Code`, `## Docs`, `## Patterns`, `## Dependents`, `## Dependencies` sections
- AND each section has descriptions without raw HelixDB IDs

---

## 3. mcp-tools-python — Python MCP Tools

**Purpose**: PydanticAI-based MCP tools at `mcp-tools/` that query HelixDB from OpenCode.

### Requirements

| ID | Requirement | Strength |
|----|-------------|----------|
| R-MCP-001 | `task_context(id: int)` MUST return full task context from HelixDB | MUST |
| R-MCP-002 | `search_code(query: str, limit: int)` MUST return matching code nodes | MUST |
| R-MCP-003 | `search_skills(query: str, limit: int)` MUST return matching skill nodes | MUST |
| R-MCP-004 | Each tool MUST use PydanticAI `@mcp.tool()` decorator | MUST |
| R-MCP-005 | Tools MUST connect to HelixDB via `httpx` HTTP client | MUST |
| R-MCP-006 | Tools MUST be registered under `mcpTools.helix-integration` in `opencode.json` | MUST |
| R-MCP-007 | SDK at `mcp-tools/helix_client.py` MUST encapsulate HelixDB HTTP calls | SHOULD |

### Scenarios

#### R-MCP-001: task_context returns context
- GIVEN HelixDB has data for task 42
- WHEN agent invokes `task_context` with `id=42`
- THEN tool returns JSON with 6 sections (skills, code, docs, patterns, dependents, dependencies)

#### R-MCP-006: Tool registration in opencode.json
- GIVEN `mcp-tools/runner.py` exists
- WHEN OpenCode starts
- THEN tools `task_context`, `search_code`, `search_skills` are registered as MCP tools
- AND command points to `uv run mcp-tools/runner.py`

#### R-MCP-005: HelixDB unreachable
- GIVEN HelixDB is not running
- WHEN any tool is invoked
- THEN tool returns structured error: `{"error": "HelixDB connection failed", "helix_url": "..."}`

---

## 4. helix-agent-http — Direct HTTP Access

**Purpose**: Documented pattern for agents to query HelixDB directly via HTTP (no middleware layer).

### Requirements

| ID | Requirement | Strength |
|----|-------------|----------|
| R-HTTP-001 | Agent MUST query HelixDB via `POST /v1/query` with JSON body | MUST |
| R-HTTP-002 | Agent MUST include `x-project-id` header for project isolation | MUST |
| R-HTTP-003 | Agent SHOULD use this pattern only for read-only, exploratory queries | SHOULD |
| R-HTTP-004 | Agent MUST handle 4xx (bad request) and 5xx (server error) gracefully | MUST |
| R-HTTP-005 | Agent MUST NOT use this pattern for mutations (use Go SDK or MCP tools instead) | MUST NOT |

### Scenarios

#### R-HTTP-001: Direct query returns results
- GIVEN HelixDB running at `localhost:6969`
- WHEN agent sends `POST /v1/query` with `{"query": "MATCH (n:skill) RETURN n LIMIT 5"}` and header `x-project-id: my-project`
- THEN agent receives 200 with JSON body containing matched nodes
- AND results are scoped to "my-project"

#### R-HTTP-004: Server error handling
- GIVEN HelixDB returns 500
- WHEN agent queries
- THEN agent returns readable error: "HelixDB query failed (HTTP 500)"
- AND agent does not crash — continues with degraded behavior

---

## Summary

| Domain | Type | Requirements | Scenarios |
|--------|------|-------------|-----------|
| helix-db-client | New | 7 (R-HELIX-001–007) | 6 |
| task-context | New | 6 (R-TASK-001–006) | 3 |
| mcp-tools-python | New | 7 (R-MCP-001–007) | 3 |
| helix-agent-http | New | 5 (R-HTTP-001–005) | 2 |

**Coverage**: Happy paths ✓ | Edge cases (ErrNotFound, connection failure) ✓ | Error states (4xx/5xx, partial failure) ✓
