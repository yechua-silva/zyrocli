# Spec: Refactor zyro-task-board MCP Tools

## Context

The `zyro-task-board` MCP server exposes 6 tools: `dispatch_task`, `check_task_status`,
`wait_phase`, `list_tasks`, `cancel_task`, `complete_task`. The core problem is that
`executeTask` in `task_manager.go` has a stub that auto-completes tasks immediately,
making `complete_task` unreachable and the async orchestration pattern non-functional.

## Problem

1. `dispatch_task` creates a task and auto-completes it via stub (microseconds)
2. `complete_task` can never succeed because task is already `TaskDone`
3. Tool names are inconsistent (`dispatch_task`, `check_task_status` vs `list_tasks`)
4. `cancel_task` leaks `activeCount` (doesn't decrement for running tasks)
5. Tool schema enum only lists `F0-F4`, missing `PRE-F0`
6. `DoneAt` formats zero time as year `0001` instead of `null`

## Solution

### Tools (renamed/replaced)

| Old | New | Behavior |
|-----|-----|----------|
| `dispatch_task` | ❌ Remove | — |
| `check_task_status` | `task_status` | Same, rename only |
| `list_tasks` | `task_list` | Same, rename only |
| `wait_phase` | `task_wait` | Same, rename only |
| `cancel_task` | `task_cancel` | Same, fix activeCount leak |
| `complete_task` | `task_complete` | Now works: task stays running until externally completed |
| — | `task_create` | NEW: creates task as TaskRunning, does NOT auto-execute |

### Bug fixes

1. **executeTask stub removed**: `dispatch_task` no longer exists. `task_create` creates
   task as `TaskRunning` without auto-executing.
2. **cancel_task activeCount leak**: Add `tm.activeCount--` when cancelling a running task.
3. **Schema enum**: Add `PRE-F0` to valid phases.
4. **DoneAt zero time**: If `task.DoneAt.IsZero()`, output `null` instead of year 0001.

### Files to change

| File | Changes |
|------|---------|
| `cmd/zyrocli/mcp_server.go` | Rename tools, add `task_create`, remove `dispatch_task`, fix DoneAt, fix enum |
| `internal/boomerang/task_manager.go` | Fix `CancelTask` activeCount leak, fix `executeTask` stub |
| `internal/boundari/loader.go` | Update tool names in permissions |
| `.config/opencode/opencode.json` | Update tool name if referenced |

### Non-goals

- The `zyrocli task` CLI (`create`, `link`, `list`) stays unchanged
- The internal `TaskManager` in `orchestrator.go`/`delegate.go` stays unchanged
- The `apply.Runner` stays unchanged

## Acceptance Criteria

1. `task_create` creates a task as `TaskRunning` and returns its ID
2. `task_complete` marks a running task as done
3. `task_cancel` cancels a task and decrements activeCount
4. `task_status` returns full task info with DoneAt=null if not done
5. `task_list` accepts `PRE-F0` in phase filter
6. `task_wait` accepts `PRE-F0` in phase filter
7. All existing tests pass
8. `go build ./...` compiles
