# Design: Refactor zyro-task-board MCP Tools

> Based on spec: `sdd/mcp-task-tools-refactor/spec.md`

## Overview

Refactor the 6 MCP tools exposed by `zyro-task-board` to fix the `executeTask` stub,
rename for consistency, add `task_create`, and fix 3 bugs.

## Architecture

### Current flow (broken)

```
dispatch_task → TaskManager.DispatchTask()
  → executeTask() [stub auto-completes in µs]
  → task.status = TaskDone
  → complete_task() called by orchestrator → returns false (already done)
```

### New flow (fixed)

```
task_create → TaskManager.CreateTask() → task.status = TaskRunning (no auto-execute)
  → orchestrator does real work
  → task_complete() → task.status = TaskDone (works now)
```

## Detailed changes

### 1. cmd/zyrocli/mcp_server.go

#### Tool definitions (handleToolsList)

Replace the 6 old tool definitions with renamed ones.
Remove `dispatch_task`, add `task_create`.
Add `PRE-F0` to all phase enums.
Fix `DoneAt` to output `null` when zero time.

#### Handler routing (handleToolsCall)

Replace cases:
- `dispatch_task` → removed
- `check_task_status` → `task_status`
- `wait_phase` → `task_wait`
- `list_tasks` → `task_list`
- `cancel_task` → `task_cancel`
- `complete_task` → `task_complete`
- (new) `task_create`

### 2. internal/boomerang/task_manager.go

#### Add CreateTask method

New method that sets status to `TaskRunning` without auto-executing.
Increments `activeCount`.

#### Fix CancelTask activeCount leak

Add `tm.activeCount--` when cancelling a `TaskRunning` task.

### 3. internal/boundari/loader.go

Update tool names in permission allowlist.

### 4. .config/opencode/opencode.json

Update tool references.

## Tasks

See `tasks.md`
