# Tasks: MCP Task Tools Refactor

## T1.1 — Add CreateTask method
**File**: `internal/boomerang/task_manager.go`

## T1.2 — Fix CancelTask activeCount leak
**File**: `internal/boomerang/task_manager.go`

## T1.3 — Rename + refactor tool definitions
**File**: `cmd/zyrocli/mcp_server.go`

## T1.4 — Add task_create handler
**File**: `cmd/zyrocli/mcp_server.go`

## T1.5 — Fix DoneAt zero-time formatting
**File**: `cmd/zyrocli/mcp_server.go`

## T1.6 — Add PRE-F0 to schema enum
**File**: `cmd/zyrocli/mcp_server.go`

## T1.7 — Update boundari permissions
**File**: `internal/boundari/loader.go`

## T1.8 — Update opencode.json
**File**: `.config/opencode/opencode.json`

## T1.9 — Verify
**Command**: `go build ./... && go test ./... -count=1 -timeout 120s`
