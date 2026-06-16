# MCP Server Specification

## Purpose

Define the MCP server that exposes HelixDB queries as tools for OpenCode consumption, using JSON-RPC 2.0 over stdin/stdout.

## Requirements

### Requirement: Transport Protocol

The server MUST implement JSON-RPC 2.0 over stdin/stdout. Each request MUST be a single JSON line; each response MUST be a single JSON line. The server MUST exit cleanly when stdin closes.

#### Scenario: Start and respond to request
- GIVEN the MCP server binary is started
- WHEN a valid JSON-RPC request is written to stdin
- THEN a JSON-RPC response is written to stdout

#### Scenario: Graceful exit on stdin close
- GIVEN the MCP server is running
- WHEN stdin is closed
- THEN the server exits with code 0

### Requirement: Tool task_context

`task_context(task_id: uint64)` MUST retrieve task context from HelixDB via `internal/taskcontext.GetTaskContext()` and return it formatted as text, JSON, or prompt format (default: text). The `task_id` parameter is REQUIRED. The `format` parameter is OPTIONAL ("text", "json", "prompt", default: "text").

#### Scenario: Happy path — existing task
- GIVEN HelixDB running with task #1 linked to skills and codenodes
- WHEN `task_context(1)` is called
- THEN a formatted context is returned with skills, codenodes, documents, patterns

#### Scenario: Task not found
- GIVEN HelixDB running without task #999
- WHEN `task_context(999)` is called
- THEN JSON-RPC error -32000 with message "task not found" is returned

#### Scenario: Missing required param
- GIVEN MCP server running
- WHEN `task_context()` is called without params
- THEN JSON-RPC error -32602 with message "missing required parameter: task_id" is returned

### Requirement: Tool search_code

`search_code(query: string, project_id?: string, limit?: int)` MUST search CodeNodes by text content via `helix.TextSearch()`. When `project_id` is provided, results MUST be scoped to that project. `limit` MUST cap results (default 10, max 50).

#### Scenario: Search all projects
- GIVEN CodeNodes exist in HelixDB with "auth" in content
- WHEN `search_code("auth")` is called
- THEN matching CodeNodes are returned, max 10

#### Scenario: Search scoped to project
- GIVEN CodeNodes across multiple projects
- WHEN `search_code("handler", "my-project")` is called
- THEN results include only CodeNodes where project_id="my-project"

#### Scenario: Custom limit
- GIVEN CodeNodes exist matching query
- WHEN `search_code("auth", "", 5)` is called
- THEN at most 5 results are returned

### Requirement: Tool search_skills

`search_skills(query: string, limit?: int)` MUST search global Skills by text content via `helix.TextSearchGlobal()` and `helix.FindSharedSkills()`. Skills are global — no project scope applies. `limit` MUST cap results (default 10, max 50).

#### Scenario: Search available skills
- GIVEN Skills exist globally in HelixDB
- WHEN `search_skills("react")` is called
- THEN matching Skills with name, type, source_url are returned

### Requirement: HelixDB Error Handling

If HelixDB is unreachable, every tool MUST return JSON-RPC error -32000 with message "HelixDB connection failed". The server MUST NOT crash, MUST continue accepting subsequent requests.

#### Scenario: HelixDB down
- GIVEN HelixDB is not running
- WHEN `task_context(1)` is called
- THEN error -32000 "HelixDB connection failed" is returned
- AND the server continues running

### Requirement: Timeout

Each tool handler MUST enforce a 30s timeout for HelixDB queries. Timeout MUST return JSON-RPC error -32000 with message "request timed out".

### Requirement: OpenCode Registration

The MCP server binary MUST be registerable in `opencode.json` via the `mcpServers` section using `command` mode. The registration config MUST specify the binary path and `args: []`.

#### Scenario: Register and invoke
- GIVEN the MCP server binary is built
- WHEN it is registered in opencode.json as an MCP tool
- THEN OpenCode can invoke `task_context`, `search_code`, `search_skills`
