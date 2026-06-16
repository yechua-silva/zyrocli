# Context MCP Bridge Specification

## Purpose

Define the Context MCP server bridge that manages an external MCP process lifecycle, sends JSON-RPC queries for library documentation, and returns structured results to the agent investigation pipeline.

## Requirements

### Requirement: Process Lifecycle

`Bridge.Start(ctx) error` MUST launch the MCP server binary (`context serve --libs`) via `os/exec` with stdin/stdout piped. `Bridge.Stop() error` MUST send SIGTERM and wait up to 5s for graceful shutdown, then fall back to SIGKILL if the process does not exit. `Bridge.IsRunning() bool` MUST return the process state.

Attempting to start an already-running bridge MUST return an error wrapping "already running". Stopping an already-stopped bridge MUST return nil (no-op).

#### Scenario: Start and verify running
- GIVEN a valid `context` binary in PATH
- WHEN `Start(ctx)` is called
- THEN the process is running and `IsRunning()` returns `true`

#### Scenario: Graceful shutdown
- GIVEN a running MCP server
- WHEN `Stop()` is called
- THEN SIGTERM is sent and the process exits within 5s

#### Scenario: Force kill on timeout
- GIVEN a running MCP server that ignores SIGTERM
- WHEN `Stop()` is called and 5s elapses
- THEN SIGKILL is sent and the process terminates

#### Scenario: Stop on unstarted bridge
- GIVEN a bridge that has never been started
- WHEN `Stop()` is called
- THEN no error is returned

### Requirement: JSON-RPC Query

`Bridge.QueryDocs(ctx, libraryID, query string) ([]byte, error)` MUST send a properly framed JSON-RPC 2.0 request to the MCP server via stdin, read the newline-delimited response from stdout, and return the result payload. Timeout MUST default to 30s. Invalid JSON-RPC responses MUST return an error wrapping "decode". JSON-RPC errors MUST be surfaced with code and message.

#### Scenario: Successful query
- GIVEN a running MCP server and a valid library ID
- WHEN `QueryDocs(ctx, "/vercel/next.js", "how to use app router")` is called
- THEN documentation content is returned as bytes without error

#### Scenario: Server not running
- GIVEN the MCP server is not started
- WHEN `QueryDocs(ctx, "/vercel/next.js", "app router")` is called
- THEN an error wrapping "not running" is returned

#### Scenario: JSON-RPC error response
- GIVEN the MCP server returns a JSON-RPC error
- WHEN `QueryDocs(ctx, "/vercel/next.js", "query")` is called
- THEN an error wrapping the JSON-RPC error code and message is returned

### Requirement: Library ID Resolution

`Bridge.ResolveLibraryID(ctx, packageName string) (string, error)` MUST send a JSON-RPC `resolve_library_id` request and return the canonical `/org/project` ID string. If the response contains an empty `library_id`, MUST return an error wrapping "empty library ID". On timeout or network error, MUST return a wrapped error.

#### Scenario: Resolve known package
- GIVEN a running MCP server
- WHEN `ResolveLibraryID(ctx, "next.js")` is called
- THEN `/vercel/next.js` is returned with nil error

#### Scenario: Unknown package
- GIVEN a running MCP server that returns empty library_id
- WHEN `ResolveLibraryID(ctx, "nonexistent-pkg")` is called
- THEN an error wrapping "empty library ID" is returned

### LibraryID Type

`LibraryID` is a struct with `Org`, `Project`, and optional `Version` fields. `LibraryID.String()` returns the canonical ID: `/org/project` or `/org/project/version`.
