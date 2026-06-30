# Activate Boundary Enforcement — Technical Design

> **Documento**: Technical Design  
> **Estado**: Draft  
> **Fase**: F2 (Design)  
> **Basado en**: `spec-activate-boundary-enforcement.md` + `spec-acceptance-criteria-tracking.md`  
> **Propuesta origen**: `openspec/proposals/activate-boundary-enforcement.md`  

---

## Table of Contents

1. [Current State Analysis](#1-current-state-analysis)
2. [Architecture Overview](#2-architecture-overview)
3. [Detailed Design Decisions](#3-detailed-design-decisions)
4. [Component Design: Enforcer Lifecycle](#4-component-design-enforcer-lifecycle)
5. [Component Design: PRE-F0 Policy](#5-component-design-pre-f0-policy)
6. [Component Design: F0–F4 Policy Updates](#6-component-design-f0f4-policy-updates)
7. [Component Design: DelegateStep Enforcement](#7-component-design-delegatestep-enforcement)
8. [Component Design: SaveStep Enforcement](#8-component-design-savestep-enforcement)
9. [Component Design: Audit Logging](#9-component-design-audit-logging)
10. [Component Design: QualityStep + Acceptance Criteria](#10-component-design-qualitystep--acceptance-criteria)
11. [Component Design: ApprovalGate + Pipeline Gate](#11-component-design-approvalgate--pipeline-gate)
12. [Connection with spec-acceptance-criteria-tracking.md](#12-connection-with-spec-acceptance-criteria-trackingmd)
13. [Files to Modify](#13-files-to-modify)
14. [Testing Strategy](#14-testing-strategy)
15. [Risks and Mitigations](#15-risks-and-mitigations)
16. [Implementation Order](#16-implementation-order)

---

## 1. Current State Analysis

### 1.1 The Gap

```
BoomerangOrchestrator {
  boundariLoader: func(string) (*Policy, error)  ← SE ALMACENA  (orchestrator.go:106)
  taskManager:    *TaskManager
  memoryStore:    EngramStore
}

runPhaseV2() {                                    ← (orchestrator.go:121)
  // NO usa boundariLoader                         ← EL GAP
  // NO crea Enforcer
  // NO llama CheckTool()
  // NO loggea audit events
  // NO persiste audit log
  for step in steps {
    switch step {
      case StepDelegate: DelegateStep()            ← sin enforcement
      case StepSave:     SaveStep()                ← sin enforcement
    }
  }
}
```

### 1.2 Evidence from the Codebase

| Location | What exists | What's missing |
|----------|------------|----------------|
| `orchestrator.go:91` | `boundariLoader` field | Never called in `runPhaseV2()` |
| `orchestrator.go:100-111` | `NewBoomerangOrchestrator()` stores `boundariLoader` | No wiring |
| `delegate.go:11` | `DelegateStep(ctx, dag, phase)` | No `*Enforcer` param, no `CheckTool()` |
| `save.go:13` | `SaveStep(ctx, phase, delegateResult, logData)` | No `*Enforcer` param, no `CheckTool()` |
| `orchestrator.go:145-213` | Step loop in `runPhaseV2()` | No budget check, no audit |
| `loader.go:34-68` | `LoadDefaultPolicy()` | Missing `case "PRE-F0"` |
| `loader.go:40-66` | Cases `F0`, `F3`, default | Missing `dispatch_task`, `save_to_helix` rules |
| `phase{0,2,4}-boundari.yaml` | YAML policies | Missing `dispatch_task` rule |
| `phase{1,2,3}-boundari.yaml` | YAML policies | Missing `save_to_helix` rule |
| `cmd/zyrocli/run.go` | Pipeline messages | Don't mention PRE-F0 |
| `quality.go:13-29` | `QualityStep` | Only checks `go build` + task success, no criteria evaluation |
| `approval.go:97-117` | `ApprovalGate` | No criteria status display or blocking |
| `scheduler.go:119` | Pipeline gate | No criteria verification before advancing |

### 1.3 Existing Boundari Module Assets

The `internal/boundari` package already provides all primitives needed:

- **`Enforcer`** (`enforcer.go:13`): Evaluates policies, tracks budget
- **`CheckTool()`** (`enforcer.go:25`): Verifies tool per policy rules
- **`IsBudgetExceeded()`** (`enforcer.go:78`): Budget boundary check
- **`LogAudit()`** (`enforcer.go:107`): Registers audit events
- **`SaveAuditLog()`** (`enforcer.go:113`): Persists JSONL to disk
- **`ClearAuditLog()`** (`enforcer.go:130`): Resets between phases
- **`LoadPolicy()`** (`loader.go:14`): Loads YAML policy per phase
- **`LoadDefaultPolicy()`** (`loader.go:34`): Hardcoded fallback
- **`ValidatePolicy()`** (`loader.go:71`): Schema validation for YAMLs

---

## 2. Architecture Overview

### 2.1 Enforcer Lifecycle Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         runPhaseV2(ctx, config)                             │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  PHASE INIT:                                                       │   │
│  │  1. policy, err := o.boundariLoader(config.Phase)                  │   │
│  │     if err → policy = boundari.LoadDefaultPolicy(config.Phase)     │   │
│  │  2. enforcer := boundari.NewEnforcer(policy)                       │   │
│  │  3. boundari.ClearAuditLog()                                       │   │
│  └──────────────────────────┬──────────────────────────────────────────┘   │
│                             │                                              │
│                             ▼                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  STEP LOOP (for each step in steps):                                │   │
│  │                                                                     │   │
│  │  ┌────────────────────────────────────────────────────────────────┐ │   │
│  │  │  BUDGET CHECK (antes de cada step):                           │ │   │
│  │  │  if enforcer.IsBudgetExceeded() → abort con error controlado  │ │   │
│  │  └────────────────────────────────────────────────────────────────┘ │   │
│  │                                                                     │   │
│  │  ┌──────────┐  ┌──────────┐  ┌────────────┐  ┌──────────┐         │   │
│  │  │ Memory   │  │ Think    │  │ Delegate   │  │ Save     │         │   │
│  │  │ Step     │  │ Step     │  │ Step       │  │ Step     │         │   │
│  │  │(no enf)  │  │(no enf)  │  │(CheckTool  │  │(CheckTool│         │   │
│  │  │          │  │          │  │ dispatch)  │  │ save)    │         │   │
│  │  └──────────┘  └──────────┘  └──────┬─────┘  └────┬─────┘         │   │
│  │                                     │             │               │   │
│  │                                     ▼             ▼               │   │
│  │  ┌─────────────────────────────────────────────────────────────┐  │   │
│  │  │                    boundari.Enforcer                         │  │   │
│  │  │  ┌────────────────┐  ┌──────────────┐  ┌────────────────┐  │  │   │
│  │  │  │ CheckTool()    │  │ IsBudgetEx-  │  │ usage.ToolCalls│  │  │   │
│  │  │  │ → allow/deny   │  │ ceeded()     │  │ + StartedAt    │  │  │   │
│  │  │  └────────────────┘  └──────────────┘  └────────────────┘  │  │   │
│  │  └─────────────────────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────┐  │   │
│  │  │              boundari.Policy (cargada)                       │  │   │
│  │  │  Budget{MaxToolCalls, MaxRuntimeSecs} + []ToolRule           │  │   │
│  │  │  Fuente: YAML (phase{N}-boundari.yaml) o LoadDefaultPolicy  │  │   │
│  │  └─────────────────────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────┐  │   │
│  │  │              auditLogger (singleton)                          │  │   │
│  │  │  LogAudit() → events[] → SaveAuditLog(path) → JSONL          │  │   │
│  │  └─────────────────────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  PHASE TEARDOWN:                                                   │   │
│  │  1. Persist audit log: audit/boomerang-{phase}-{ts}.jsonl          │   │
│  │  2. (warning no bloqueante si falla)                               │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  QUALITY + APPROVAL (cross-cutting con acceptance-criteria spec):  │   │
│  │  - QualityStep.evaluateCriteria() chequea acceptance criteria       │   │
│  │  - ApprovalGate muestra criteria summary                           │   │
│  │  - Pipeline gate verifica criteria antes de avanzar fase           │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Data Flow: Policy → Enforcer → Step

```
YAML file                    LoadPolicy() / LoadDefaultPolicy()
    │                                   │
    ▼                                   ▼
┌──────────────┐             ┌──────────────────┐
│ Policy       │────────────▶│ Enforcer          │
│ - Budget     │             │ - policy *Policy  │
│ - []ToolRule │             │ - usage BudgetUsage│
└──────────────┘             │ - started bool    │
                             └──────────────────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    │                 │                 │
                    ▼                 ▼                 ▼
              CheckTool()      IsBudgetExceeded()   LogAudit()
                    │                 │                 │
                    ▼                 ▼                 ▼
              dispatch_task      budget check     audit/phase.jsonl
              save_to_helix
```

---

## 3. Detailed Design Decisions

### 3.1 Where to Create the Enforcer? Each Phase or Shared?

**Decision: One `Enforcer` per `runPhaseV2()` call (one per phase execution).**

- The `Enforcer` is created at the beginning of each `runPhaseV2()` invocation (line 121 of `orchestrator.go`).
- It lives only for the duration of that phase execution.
- Rationale:
  - Each phase has its own policy (different YAML per phase).
  - `ClearAuditLog()` at phase start prevents cross-phase contamination.
  - Budget is per-phase, so a new `Enforcer` with fresh `BudgetUsage` makes sense.
  - The `Enforcer` is cheap to create (no I/O beyond the policy load, which happens once).
  - Avoids concurrency issues with a shared mutable `Enforcer` across phases.

**Alternative considered:** Create one `Enforcer` at `BoomerangOrchestrator` construction and reuse. Rejected because phases have different policies and budgets; also the audit logger is a singleton so sharing would require complex reset logic.

### 3.2 How to Pass the Enforcer to DelegateStep and SaveStep?

**Decision: Optional `*boundari.Enforcer` parameter as the last argument.**

```go
// DelegateStep — new signature
func (o *BoomerangOrchestrator) DelegateStep(
    ctx context.Context,
    dag *TaskDAG,
    phase string,
    enforcer *boundari.Enforcer,  // ← new, may be nil
) (*DelegateResult, error)

// SaveStep — new signature
func (o *BoomerangOrchestrator) SaveStep(
    ctx context.Context,
    phase string,
    delegateResult *DelegateResult,
    logData []byte,
    enforcer *boundari.Enforcer,  // ← new, may be nil
) (*SaveResult, error)
```

- **If `enforcer` is `nil`**: skip all enforcement (backward compatibility for tests that call these methods directly).
- **If `enforcer` is non-nil**: run `CheckTool()` before each tool call.
- This is a backward-compatible additive change: existing callers in tests pass `nil` and get the old behavior.

**Rationale:**
- Go does not have optional parameters. A pointer that can be `nil` is the idiomatic pattern.
- Tests like `TestDelegateStep` (boomerang_test.go:118) and `TestSaveStep` (boomerang_test.go:158) call these directly and will need updating, but they can pass `nil` and continue working unchanged.
- No interface changes needed — the methods are already on `BoomerangOrchestrator` and are not part of a public interface.

### 3.3 How to Handle Budget Exceeded? Soft Abort vs Panic

**Decision: Soft abort with error return (no panic).**

```go
if enforcer.IsBudgetExceeded() {
    return nil, fmt.Errorf("boundari: budget exceeded for phase %s", config.Phase)
}
```

- The error propagates up through `runPhaseV2()` → `RunPhase()` → `scheduler.Run()`.
- The scheduler marks the phase as `StatusFail` with the error message.
- No panic, no crash. The pipeline can continue (with approval) or the user can retry.
- `IsBudgetExceeded()` is checked at the *start* of every step, not inside a step. This ensures:
  - A step that is mid-execution completes (its budget was already checked at entry).
  - The check is lightweight (two integer comparisons) so there's no perf concern.
  - The granularity is "at most N tool calls per phase", not "at most N-1".

**Alternative considered:** Check inside the tool loop of `DelegateStep` before each `DispatchTask`. This would give finer granularity but adds a check inside every iteration and is redundant because `CheckTool()` already checks budget internally (`enforcer.go:32-36`). The explicit `IsBudgetExceeded()` at step start is a fast-path guard.

### 3.4 Where to Persist the Audit Log?

**Decision: `audit/boomerang-{phase}-{unix_timestamp}.jsonl` at the end of each phase.**

```go
auditDir := "audit"
auditFile := fmt.Sprintf("boomerang-%s-%d.jsonl", config.Phase, time.Now().Unix())
auditPath := filepath.Join(auditDir, auditFile)
if err := boundari.SaveAuditLog(auditPath); err != nil {
    fmt.Fprintf(os.Stderr, "⚠ boundari: error saving audit log: %v\n", err)
}
```

- Path is relative to the working directory (project root).
- `audit/` directory is created automatically by `SaveAuditLog()` via `os.MkdirAll`.
- File format: JSONL (one JSON object per line), which is append-friendly and grep-friendly.
- Naming convention: `boomerang-F0-1718000000.jsonl`
- Saving is **non-blocking warning** — if it fails, the phase result is unaffected.
- `ClearAuditLog()` is called at phase *start* to prevent accumulation across phases.

**Rationale:**
- The `audit/` directory is outside `internal/` so it's accessible from CI or inspection.
- Timestamp in filename prevents overwrites.
- JSONL is machine-parseable and can be loaded into analysis tools.

### 3.5 PRE-F0 Policy: What Tools to Allow?

**Decision: PRE-F0 is a "read + documentation write" phase.**

| Tool | Action | Rationale |
|------|--------|-----------|
| `read_file` | allow | Read existing code and docs |
| `search_code` | allow | Search codebase |
| `search_skills` | allow | Search available skills |
| `task_context` | allow | Get task context |
| `web_search` | allow | Research |
| `web_fetch` | allow | Fetch URLs |
| `glob` | allow | File pattern matching |
| `grep` | allow | Content search |
| `write_file` | allow | Write .md documentation |
| `save_to_helix` | allow | Save findings to HelixDB |
| `dispatch_task` | allow | Delegate to subagents |
| `edit_file` | deny | No code modification |
| `execute_command` | deny | No shell execution |

**Budget:** `MaxToolCalls: 30`, `MaxRuntimeSecs: 300` (conservative — PRE-F0 is a short alignment phase).

**Rationale for `write_file: allow` without restrictions (Option A from proposal):**
- PRE-F0 only produces `.md` documentation files in `openspec/` and `CONTEXT.md`.
- No implementation code is written during this phase.
- The risk of accidental overwrite is low compared to the friction of whitelisting extensions.

**Rationale for `dispatch_task: allow`:**
- The skip matrix for PRE-F0 includes `StepDelegate`, so subagents need to be dispatched.

---

## 4. Component Design: Enforcer Lifecycle

### 4.1 Scope: One Phase

```
┌─ runPhaseV2() ──────────────────────────────────────┐
│                                                      │
│  Enforcer created at top of function                 │
│  ↓                                                   │
│  ClearAuditLog() ← fresh slate for this phase        │
│  ↓                                                   │
│  Step loop with IsBudgetExceeded() checks            │
│  ↓                                                   │
│  SaveAuditLog() at end ← persist events              │
│  ↓                                                   │
│  Enforcer goes out of scope (GC'd)                   │
│  ↓                                                   │
│  PhaseResult returned (success or budget exceeded)   │
│                                                      │
└──────────────────────────────────────────────────────┘
```

### 4.2 Code Insertion Points in `runPhaseV2()`

```go
func (o *BoomerangOrchestrator) runPhaseV2(ctx context.Context, config PhaseConfigV2) (*PhaseResult, error) {
    start := time.Now()
    result := &PhaseResult{Phase: config.Phase}

    // >>> 1. LOAD POLICY + CREATE ENFORCER <<<
    // (After line 123, before line 125)
    policy, err := o.boundariLoader(config.Phase)
    if err != nil {
        policy = boundari.LoadDefaultPolicy(config.Phase)
    }
    enforcer := boundari.NewEnforcer(policy)
    boundari.ClearAuditLog()

    // ... existing matrix and steps setup ...

    for _, step := range steps {
        // >>> 2. BUDGET CHECK BEFORE EVERY STEP <<<
        if enforcer.IsBudgetExceeded() {
            return nil, fmt.Errorf("boundari: budget exceeded for phase %s", config.Phase)
        }

        switch step {
        case StepMemory:   // no enforcement needed
        case StepThink:    // no enforcement needed
        case StepDelegate: // passes enforcer
            o.DelegateStep(ctx, dag, config.Phase, enforcer)
        case StepGit:      // no enforcement needed
        case StepQuality:  // passes enforcer to DelegateStep retry
            // inside retry loop: o.DelegateStep(ctx, dag, config.Phase, enforcer)
        case StepSave:     // passes enforcer
            o.SaveStep(ctx, config.Phase, delegateResult, nil, enforcer)
        }
    }

    // >>> 3. PERSIST AUDIT LOG <<<
    // (Before token estimation, after step loop)
    auditDir := "audit"
    auditFile := fmt.Sprintf("boomerang-%s-%d.jsonl", config.Phase, time.Now().Unix())
    auditPath := filepath.Join(auditDir, auditFile)
    if err := boundari.SaveAuditLog(auditPath); err != nil {
        fmt.Fprintf(os.Stderr, "⚠ boundari: error saving audit log: %v\n", err)
    }

    // ... rest of function (token estimation, measurement callback, success calc) ...
}
```

### 4.3 Imports to Add to `orchestrator.go`

```go
import (
    "context"
    "fmt"        // ← likely already present (via Errorf), verify
    "os"         // ← NUEVO (for Stderr)
    "path/filepath" // ← NUEVO (for Join)
    "time"

    "github.com/secko/zyrocli/internal/boundari"
    "github.com/secko/zyrocli/internal/memory"
    "github.com/secko/zyrocli/internal/tokens"
)
```

**Note:** `fmt` may already be imported transitively. Verify in actual file. If not used elsewhere, it's needed for the error message.

---

## 5. Component Design: PRE-F0 Policy

### 5.1 New YAML File: `internal/boundari/phasePRE-F0-boundari.yaml`

```yaml
version: "1.0"
phase: "PRE-F0"
description: "Alineación de dominio — solo lectura + escritura de documentos .md. No ejecutar comandos."
budget:
  max_tool_calls: 30
  max_runtime_seconds: 300
tools:
  - name: "read_file"
    action: allow
  - name: "search_code"
    action: allow
  - name: "search_skills"
    action: allow
  - name: "task_context"
    action: allow
  - name: "web_search"
    action: allow
  - name: "web_fetch"
    action: allow
  - name: "glob"
    action: allow
  - name: "grep"
    action: allow
  - name: "write_file"
    action: allow
  - name: "save_to_helix"
    action: allow
  - name: "dispatch_task"
    action: allow
  - name: "edit_file"
    action: deny
  - name: "execute_command"
    action: deny
```

### 5.2 Filename Convention

The `LoadPolicy()` function (`loader.go:16`) constructs the filename as:

```go
phaseNum := strings.TrimPrefix(phase, "F")
filename := fmt.Sprintf("phase%s-boundari.yaml", phaseNum)
```

For `"PRE-F0"`:
- `strings.TrimPrefix("PRE-F0", "F")` → `"PRE-F0"` (the string doesn't start with "F")
- `fmt.Sprintf("phase%s-boundari.yaml", "PRE-F0")` → `"phasePRE-F0-boundari.yaml"`

This is correct and produces the expected filename.

### 5.3 Default Policy Fallback: `LoadDefaultPolicy()`

Add `case "PRE-F0"` to the switch in `loader.go:40`:

```go
case "PRE-F0":
    p.Description = "Alineación de dominio — lectura + .md (fallback)"
    p.Budget = Budget{MaxToolCalls: 30, MaxRuntimeSecs: 300}
    p.Tools = []ToolRule{
        {Name: "read_file", Action: ActionAllow},
        {Name: "search_code", Action: ActionAllow},
        {Name: "search_skills", Action: ActionAllow},
        {Name: "task_context", Action: ActionAllow},
        {Name: "web_search", Action: ActionAllow},
        {Name: "web_fetch", Action: ActionAllow},
        {Name: "glob", Action: ActionAllow},
        {Name: "grep", Action: ActionAllow},
        {Name: "write_file", Action: ActionAllow},
        {Name: "save_to_helix", Action: ActionAllow},
        {Name: "dispatch_task", Action: ActionAllow},
        {Name: "edit_file", Action: ActionDeny},
        {Name: "execute_command", Action: ActionDeny},
    }
```

---

## 6. Component Design: F0–F4 Policy Updates

### 6.1 Problem

The existing YAML policies (`phase0-boundari.yaml` through `phase4-boundari.yaml`) were written before the orquestador's `CheckTool()` was wired. They don't include `dispatch_task` or `save_to_helix` as named tools. Since `CheckTool()` defaults to **deny** for tools not in the policy list (`enforcer.go:70-74`), calling `CheckTool("dispatch_task")` on any existing phase would deny all task delegation, breaking the pipeline.

### 6.2 Required Additions

| YAML File | Tool Rules to ADD (no existing rules modified) |
|-----------|------------------------------------------------|
| `phase0-boundari.yaml` (F0) | `{name: "dispatch_task", action: allow}` |
| `phase1-boundari.yaml` (F1) | `{name: "dispatch_task", action: allow}`, `{name: "save_to_helix", action: allow}` |
| `phase2-boundari.yaml` (F2) | `{name: "dispatch_task", action: allow}` (save_to_helix already deny — keep deny) |
| `phase3-boundari.yaml` (F3) | `{name: "dispatch_task", action: allow}` (save_to_helix already allow) |
| `phase4-boundari.yaml` (F4) | `{name: "dispatch_task", action: allow}` (save_to_helix already allow) |

### 6.3 Rationale Per Phase

| Phase | `dispatch_task` | `save_to_helix` | Reasoning |
|-------|----------------|-----------------|-----------|
| F0 | allow | deny (unchanged) | F0 dispatches research tasks; F0 does NOT persist facts (designed as read-only) |
| F1 | allow | allow | F1 writes specs and needs to persist criteria |
| F2 | allow | deny (unchanged) | F2 writes designs; F2 currently denies helix saves (design decision to respect) |
| F3 | allow | allow (unchanged) | F3 implements code and needs to persist results to HelixDB |
| F4 | allow | allow (unchanged) | F4 archives and persists final status |

### 6.4 Also Update `LoadDefaultPolicy()` Cases

In addition to the YAML files, the default policy cases in `loader.go` need `dispatch_task` and `save_to_helix` entries for consistency:

```go
case "F0":  // Add:
    // {Name: "dispatch_task", Action: ActionAllow},
    // {Name: "save_to_helix", Action: ActionDeny},

case "F3":  // Add:
    // {Name: "dispatch_task", Action: ActionAllow},
    // {Name: "save_to_helix", Action: ActionAllow},

default:    // Add:
    // {Name: "dispatch_task", Action: ActionAllow},
    // {Name: "save_to_helix", Action: ActionAllow},
```

---

## 7. Component Design: DelegateStep Enforcement

### 7.1 Signature Change

```go
// Before:
func (o *BoomerangOrchestrator) DelegateStep(ctx context.Context, dag *TaskDAG, phase string) (*DelegateResult, error)

// After:
func (o *BoomerangOrchestrator) DelegateStep(ctx context.Context, dag *TaskDAG, phase string, enforcer *boundari.Enforcer) (*DelegateResult, error)
```

### 7.2 Enforcement Logic

```go
func (o *BoomerangOrchestrator) DelegateStep(ctx context.Context, dag *TaskDAG, phase string, enforcer *boundari.Enforcer) (*DelegateResult, error) {
    result := &DelegateResult{
        TaskResults: make(map[string]TaskResult),
    }

    for _, task := range dag.Tasks {
        // >>> CheckTool before each DispatchTask <<<
        if enforcer != nil {
            checkResult := enforcer.CheckTool("dispatch_task", map[string]any{
                "task_name": task.Name,
                "agent":     task.Agent,
                "phase":     phase,
            })
            boundari.LogAudit(boundari.AuditEvent{
                Phase:   phase,
                Tool:    "dispatch_task",
                Allowed: checkResult.Allowed,
                Reason:  checkResult.Reason,
            })
            if !checkResult.Allowed {
                tr := TaskResult{
                    TaskName: task.Name,
                    Success:  false,
                    Output:   fmt.Sprintf("denied by boundari: %s", checkResult.Reason),
                }
                result.TaskResults[task.Name] = tr
                result.NodesCreated++
                continue  // skip this task, don't dispatch
            }
        }

        // Existing dispatch logic
        taskID := o.taskManager.DispatchTask(ctx, task.Name, task.Agent, phase, nil)
        waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
        completedTask, err := o.taskManager.WaitTask(waitCtx, taskID)
        cancel()
        // ... rest of existing logic ...
    }
    return result, nil
}
```

### 7.3 Behavior Matrix

| Scenario | enforcer? | CheckTool | Action |
|----------|-----------|-----------|--------|
| Test calling directly | nil | — | Legacy behavior, no enforcement |
| Phase with policy | non-nil | allowed | Dispatch normally |
| Phase with policy | non-nil | denied | Skip task, log audit, add failed TaskResult |
| Phase with policy | non-nil | not in policy | Denied by default, audit logged |

### 7.4 Call Sites to Update

In `orchestrator.go`:

1. Line 167 (StepDelegate):
   ```go
   dr, err := o.DelegateStep(ctx, dag, config.Phase, enforcer)
   ```

2. Line 198 (QualityStep retry):
   ```go
   delegateResult, _ = o.DelegateStep(ctx, dag, config.Phase, enforcer)
   ```

---

## 8. Component Design: SaveStep Enforcement

### 8.1 Signature Change

```go
// Before:
func (o *BoomerangOrchestrator) SaveStep(ctx context.Context, phase string, delegateResult *DelegateResult, logData []byte) (*SaveResult, error)

// After:
func (o *BoomerangOrchestrator) SaveStep(ctx context.Context, phase string, delegateResult *DelegateResult, logData []byte, enforcer *boundari.Enforcer) (*SaveResult, error)
```

### 8.2 Enforcement Logic

```go
func (o *BoomerangOrchestrator) SaveStep(ctx context.Context, phase string, delegateResult *DelegateResult, logData []byte, enforcer *boundari.Enforcer) (*SaveResult, error) {
    result := &SaveResult{}

    if o.memoryStore == nil {
        return result, nil
    }

    for _, tr := range delegateResult.TaskResults {
        // >>> CheckTool before each SaveFact <<<
        if enforcer != nil {
            checkResult := enforcer.CheckTool("save_to_helix", map[string]any{
                "task_name": tr.TaskName,
                "phase":     phase,
            })
            boundari.LogAudit(boundari.AuditEvent{
                Phase:   phase,
                Tool:    "save_to_helix",
                Allowed: checkResult.Allowed,
                Reason:  checkResult.Reason,
            })
            if !checkResult.Allowed {
                continue  // skip this fact
            }
        }

        // Existing save logic
        if tr.Output == "" {
            continue
        }
        fact := &memory.Fact{ /* ... */ }
        id, err := o.memoryStore.SaveFact(ctx, fact)
        if err == nil && id > 0 {
            result.FactsSaved++
        }
    }
    return result, nil
}
```

### 8.3 Call Site to Update

In `orchestrator.go`:

1. Line 207 (StepSave):
   ```go
   sr, err := o.SaveStep(ctx, config.Phase, delegateResult, nil, enforcer)
   ```

---

## 9. Component Design: Audit Logging

### 9.1 Events Logged

| Event | Phase | Tool | When |
|-------|-------|------|------|
| Dispatch check | Current | `dispatch_task` | Before each `DispatchTask` |
| Save check | Current | `save_to_helix` | Before each `SaveFact` |

### 9.2 Audit Event Structure

```json
{"timestamp":"2026-06-20T12:00:00Z","phase":"F0","tool":"dispatch_task","allowed":true,"reason":"allowed by policy","duration_ms":0}
{"timestamp":"2026-06-20T12:00:01Z","phase":"F0","tool":"save_to_helix","allowed":false,"reason":"denied by policy: save_to_helix","duration_ms":0}
```

### 9.3 File Lifecycle

```
Phase START → ClearAuditLog()
                ↓
          DispatchTask #1 → LogAudit(allowed)
          DispatchTask #2 → LogAudit(allowed)
          SaveFact #1     → LogAudit(denied)
                ↓
Phase END   → SaveAuditLog("audit/boomerang-F0-1718000000.jsonl")
                ↓
          File written to disk (warning on error, non-blocking)
```

### 9.4 Concurrency Note

The `defaultAuditLogger` is a package-level singleton (`enforcer.go:104`). Currently, phases run sequentially so there is no race condition. If concurrency is added in the future (e.g., parallel phases), the logger must be made per-instance. This is tracked as a future improvement.

---

## 10. Component Design: QualityStep + Acceptance Criteria

### 10.1 Current QualityStep

```go
func (o *BoomerangOrchestrator) QualityStep(ctx context.Context, phase string, dag *TaskDAG, delegateResult *DelegateResult) (bool, error) {
    // 1. go build (solo F3)
    // 2. Verificar task success
    return true, nil
}
```

### 10.2 Proposed Enhancement

```go
func (o *BoomerangOrchestrator) QualityStep(ctx context.Context, phase string, dag *TaskDAG, delegateResult *DelegateResult) (bool, error) {
    // 1. Check compilation (F3 only)
    if phase == "F3" {
        if err := exec.CommandContext(ctx, "go", "build", "./...").Run(); err != nil {
            return false, err
        }
    }

    // 2. Check task success
    for _, tr := range delegateResult.TaskResults {
        if !tr.Success {
            return false, nil
        }
    }

    // 3. (NEW) Evaluate acceptance criteria
    criteriaOK := o.evaluateCriteria(ctx, dag, delegateResult)
    if !criteriaOK {
        return false, nil
    }

    return true, nil
}

// evaluateCriteria checks all acceptance criteria in the DAG.
// Returns true if all criteria are satisfied or if there are no criteria.
func (o *BoomerangOrchestrator) evaluateCriteria(ctx context.Context, dag *TaskDAG, delegateResult *DelegateResult) bool {
    if dag == nil || delegateResult == nil {
        return true  // nothing to evaluate
    }

    allVerified := true
    for _, task := range dag.Tasks {
        for i := range task.AcceptanceCriteria {
            c := &task.AcceptanceCriteria[i]
            if c.Status == CriteriaPending {
                tr, exists := delegateResult.TaskResults[task.Name]
                if !exists || !tr.Success || tr.Output == "" {
                    c.Status = CriteriaFailed
                    allVerified = false
                } else {
                    c.Status = CriteriaVerified
                }
            } else if c.Status == CriteriaFailed {
                allVerified = false
            }
            // CriteriaVerified — no re-evaluation
        }
    }
    return allVerified
}
```

### 10.3 AcceptanceCriteria Type

New file `internal/boomerang/criteria.go` (as specified in `spec-acceptance-criteria-tracking.md`):

```go
package boomerang

type CriteriaStatus string

const (
    CriteriaPending  CriteriaStatus = "pending"
    CriteriaVerified CriteriaStatus = "verified"
    CriteriaFailed   CriteriaStatus = "failed"
)

type AcceptanceCriteria struct {
    ID          string         `json:"id"`
    Description string         `json:"description"`
    Phase       string         `json:"phase"`
    Status      CriteriaStatus `json:"status"`
    Source      string         `json:"source"`
    TaskID      string         `json:"task_id,omitempty"`
}
```

### 10.4 TaskSpec Update

Add to `TaskSpec` in `orchestrator.go`:

```go
type TaskSpec struct {
    ID                 int                  `json:"id"`
    Name               string               `json:"name"`
    Description        string               `json:"description"`
    Agent              string               `json:"agent"`
    Tags               []string             `json:"tags,omitempty"`
    DependsOn          []int                `json:"depends_on,omitempty"`
    AcceptanceCriteria []AcceptanceCriteria `json:"acceptance_criteria,omitempty"` // ← NEW
}
```

### 10.5 QualityStep → Enforcer Interaction

The `QualityStep` itself does NOT receive a direct `*Enforcer` — it's a read-only evaluator. However, the `DelegateStep` retry inside `QualityStep` (line 198 of `orchestrator.go`) MUST pass the enforcer:

```go
case StepQuality:
    qualityRan = true
    for i := 0; i < o.maxIterations; i++ {
        qok, err := o.QualityStep(ctx, config.Phase, dag, delegateResult)
        if err == nil && qok {
            qualityOK = true
            break
        }
        if i < o.maxIterations-1 {
            if delegateResult != nil && dag != nil {
                // >>> Must pass enforcer here too <<<
                delegateResult, _ = o.DelegateStep(ctx, dag, config.Phase, enforcer)
            }
        }
    }
```

---

## 11. Component Design: ApprovalGate + Pipeline Gate

### 11.1 ApprovalGate Enhancement

The `ApprovalGate` function (`approval.go:97`) currently takes `(phase, summary)`. It should also accept an optional `CriteriaSummary`:

```go
type CriteriaSummary struct {
    Total    int `json:"total"`
    Pending  int `json:"pending"`
    Verified int `json:"verified"`
    Failed   int `json:"failed"`
}

func ApprovalGate(phase Phase, summary string, criteria *CriteriaSummary) (bool, error) {
    // Build criteria status block
    criteriaBlock := ""
    if criteria != nil && criteria.Total > 0 {
        criteriaBlock = fmt.Sprintf(
            "\n### Acceptance Criteria\nTotal: %d | ✅ Verified: %d | ⏳ Pending: %d | ❌ Failed: %d\n",
            criteria.Total, criteria.Verified, criteria.Pending, criteria.Failed,
        )
        if criteria.Failed > 0 {
            fmt.Printf("❌ No se puede aprobar: %d acceptance criteria fallaron.\n", criteria.Failed)
            return false, nil
        }
    }

    // ... existing approval dialog, injecting criteriaBlock ...
}
```

### 11.2 Pipeline Gate Enhancement (Scheduler)

In `scheduler.go:Run()` (line 119), before calling `ApprovalGate`, compute criteria summary from the finished phase:

```go
// After phase execution and before approval gate:

var criteriaSummary *CriteriaSummary
if boomerangResult != nil && dag != nil {
    criteriaSummary = computeCriteriaSummary(dag)
    if criteriaSummary.Failed > 0 {
        fmt.Printf("\n❌ %d acceptance criteria fallaron en fase %s. No se puede avanzar.\n",
            criteriaSummary.Failed, phaseName)
        results = append(results, &Result{
            Phase: phaseName,
            Status: StatusFail,
            Summary: fmt.Sprintf("%d acceptance criteria failed", criteriaSummary.Failed),
        })
        return results, fmt.Errorf("criteria check: %d failed in phase %s",
            criteriaSummary.Failed, phaseName)
    }
}

approved, err := ApprovalGate(result.Phase, result.Summary, criteriaSummary)
// ... rest of existing logic ...
```

This connects the acceptance criteria tracking to the pipeline gate, ensuring that:
- QualityStep evaluates criteria during the phase.
- The scheduler computes criteria summary from the DAG after the phase completes.
- If any criteria failed, the pipeline gate stops advancement.
- If no criteria exist (backward compat), the gate passes through.

---

## 12. Connection with spec-acceptance-criteria-tracking.md

### 12.1 Direct Dependencies

| This Design | Acceptance Criteria Spec | Mechanism |
|-------------|------------------------|-----------|
| QualityStep evaluates criteria | §3.6 | `evaluateCriteria()` function |
| ApprovalGate shows criteria | §3.7 | `CriteriaSummary` in approval dialog |
| Pipeline gate checks criteria | §3.12 | Scheduler computes summary before gate |
| TaskSpec gets AcceptanceCriteria | §3.2 | New field on existing type |
| New criteria.go type | §3.1 | `AcceptanceCriteria` struct |
| ThinkStep injects criteria | §3.3 | Criteria in `PhaseConfigV2` (future) |

### 12.2 What This Design Defers

The following items from `spec-acceptance-criteria-tracking.md` are **out of scope** for this design (they belong in separate design/implementation cycles):

| Item | Acceptance Criteria Spec Section | Reason Deferred |
|------|-------------------------------|-----------------|
| HelixDB `TaskRow` field | §3.4 | Requires DB schema migration |
| MCP server persistence | §3.5 | Requires HelixDB integration |
| Handoff criteria summary | §3.8 | Increases scope; handoff already has approval gate |
| Handoff payload fields | §3.9 | Schema change in separate contract |
| SDD-Verify skill update | §3.11 | Different skill, separate change cycle |

### 12.3 What This Design Adds Beyond the Spec

| Item | Rationale |
|------|-----------|
| `evaluateCriteria()` in QualityStep | Essential for connecting criteria eval to the phase lifecycle |
| CriteriaSummary computation in scheduler | Bridges DAG criteria status to ApprovalGate |
| QualityStep retry passes enforcer | Fixes a gap: the spec activates boundari in DelegateStep/SaveStep but QualityStep's retry path also delegates |
| Enforcer nil-safety throughout | Backward compat for direct test calls |

---

## 13. Files to Modify

### 13.1 Summary Table

| # | File | Type of Change | Lines | Risk |
|---|------|---------------|-------|------|
| 1 | `internal/boundari/phasePRE-F0-boundari.yaml` | **NEW** | ~28 | None (new file, no regression) |
| 2 | `internal/boundari/loader.go` | Modify `LoadDefaultPolicy()` | +18 | Low (adds case + entries) |
| 3 | `internal/boomerang/orchestrator.go` | Modify `runPhaseV2()` | +30 | Medium (core logic change) |
| 4 | `internal/boomerang/delegate.go` | Modify signature + loop | +22 | Medium (signature change) |
| 5 | `internal/boomerang/save.go` | Modify signature + loop | +20 | Medium (signature change) |
| 6 | `internal/boomerang/quality.go` | Add `evaluateCriteria()` | +35 | Low (additive, no existing logic changed) |
| 7 | `internal/boomerang/criteria.go` | **NEW** (AcceptanceCriteria type) | ~40 | None (new file) |
| 8 | `internal/boomerang/orchestrator.go` | Add `AcceptanceCriteria` field to `TaskSpec` | +1 | None (omitempty) |
| 9 | `cmd/zyrocli/run.go` | Update 4 string messages | 4 changes | None |
| 10 | `internal/boundari/phase0-boundari.yaml` | Add `dispatch_task: allow` | +1 | Low (additive) |
| 11 | `internal/boundari/phase1-boundari.yaml` | Add `dispatch_task: allow`, `save_to_helix: allow` | +2 | Low (additive) |
| 12 | `internal/boundari/phase2-boundari.yaml` | Add `dispatch_task: allow` | +1 | Low (additive) |
| 13 | `internal/boundari/phase3-boundari.yaml` | Add `dispatch_task: allow` | +1 | Low (additive) |
| 14 | `internal/boundari/phase4-boundari.yaml` | Add `dispatch_task: allow` | +1 | Low (additive) |
| 15 | `internal/scheduler/approval.go` | Add `CriteriaSummary` + new param | +20 | Medium (signature change) |
| 16 | `internal/scheduler/scheduler.go` | Add criteria computation before gate | +15 | Medium (gate behavior change) |
| 17 | `internal/boomerang/boomerang_test.go` | Update call sites for new signatures | +6 | Low (pass `nil`) |
| 18 | `internal/boundari/boundari_test.go` | Add PRE-F0 tests | +15 | None |

### 13.2 Detailed Changes Per File

#### File 1: `internal/boundari/phasePRE-F0-boundari.yaml` (NEW)
- Create YAML with 13 tool rules, budget 30/300, phase "PRE-F0"

#### File 2: `internal/boundari/loader.go`
- Add `case "PRE-F0"` to `LoadDefaultPolicy()` with same rules as YAML
- Add `dispatch_task` (allow) to cases `F0`, `F3`, `default`
- Add `save_to_helix` (allow) to cases `F3`, `default`

#### File 3: `internal/boomerang/orchestrator.go`
- Add imports: `"os"`, `"path/filepath"`, `"fmt"` (if not present)
- In `runPhaseV2()` after `result := &PhaseResult{Phase: config.Phase}`:
  - Load policy, create enforcer, clear audit log
- Before each step in loop: budget check
- Pass `enforcer` to `DelegateStep()` calls (line 167, 198)
- Pass `enforcer` to `SaveStep()` call (line 207)
- After step loop, before token estimation: save audit log
- Add `AcceptanceCriteria` field to `TaskSpec`

#### File 4: `internal/boomerang/delegate.go`
- Change signature: add `enforcer *boundari.Enforcer` as last param
- Before each `DispatchTask`: if enforcer != nil, `CheckTool`, `LogAudit`, skip if denied

#### File 5: `internal/boomerang/save.go`
- Change signature: add `enforcer *boundari.Enforcer` as last param
- Before each `SaveFact`: if enforcer != nil, `CheckTool`, `LogAudit`, skip if denied

#### File 6: `internal/boomerang/quality.go`
- Add `evaluateCriteria()` method
- Call it at the end of `QualityStep()`
- No signature change needed (enforcer not needed for criteria evaluation)

#### File 7: `internal/boomerang/criteria.go` (NEW)
- Define `CriteriaStatus`, `AcceptanceCriteria` types
- Package: `boomerang`

#### File 8: `internal/boomerang/orchestrator.go` (TaskSpec)
- Add `AcceptanceCriteria []AcceptanceCriteria \`json:"acceptance_criteria,omitempty"\``

#### File 9: `cmd/zyrocli/run.go`
- Line 21: `Short` → "Execute SDD pipeline (PRE-F0→F0→F1→F2→F3→F4)"
- Lines 22-27: `Long` → "Execute the 6-phase SDD pipeline..."
- Line 116: pipeline message → "PRE-F0 → F0 → F1 → F2 → F3 → F4"
- Line 143: flag help → "PRE-F0, F0, F1, F2, F3, F4"

#### File 10-14: `phase*-boundari.yaml`
- Add tool rules as specified in §6.2

#### File 15: `internal/scheduler/approval.go`
- Add `CriteriaSummary` struct
- Modify `ApprovalGate` signature to accept `*CriteriaSummary`
- Display criteria block if non-nil
- Return false if `Failed > 0`

#### File 16: `internal/scheduler/scheduler.go`
- Extract criteria summary from boomerang result after phase execution
- Compute `CriteriaSummary` from DAG criteria
- Block advancement if `Failed > 0`
- Pass `CriteriaSummary` to `ApprovalGate`

#### File 17: `internal/boomerang/boomerang_test.go`
- `TestDelegateStep`: add `nil` as last arg
- `TestSaveStep`: add `nil` as last arg
- `TestRunPhaseV2WithCustomSteps`: no change (calls `runPhaseV2` internally)
- `TestQualityStep`: no change (QualityStep signature unchanged)

#### File 18: `internal/boundari/boundari_test.go`
- `TestLoadDefaultPolicy`: add subtest for `"PRE-F0"` (verify `write_file=allow`, budget 30/300)
- `TestAllPoliciesLoad`: add `"PRE-F0"` to phases slice

---

## 14. Testing Strategy

### 14.1 Unit Tests (Modified)

| Test | File | Change |
|------|------|--------|
| `TestDelegateStep` | `boomerang_test.go:118` | Pass `nil` as enforcer. Verify no behavior change. |
| `TestSaveStep` | `boomerang_test.go:158` | Pass `nil` as enforcer. Verify FactsSaved=2. |
| `TestLoadDefaultPolicy` | `boundari_test.go:26` | Add subtest for `"PRE-F0"`: check `write_file=allow`, `dispatch_task=allow`, budget 30/300. |
| `TestAllPoliciesLoad` | `boundari_test.go:145` | Add `"PRE-F0"` to slice. Verify YAML loads. |

### 14.2 Unit Tests (New)

| Test | File | Description |
|------|------|-------------|
| `TestEnforcerCreatedInRunPhaseV2` | `*_test.go` | Mock `boundariLoader` → verify enforcer is created and audit log is saved |
| `TestDelegateStepWithEnforcer` | `boomerang_test.go` | Pass enforcer that denies `dispatch_task` → verify task is skipped and audit logged |
| `TestDelegateStepWithEnforcerAllows` | `boomerang_test.go` | Pass enforcer that allows → verify task dispatches normally |
| `TestSaveStepWithEnforcerDenies` | `boomerang_test.go` | Pass enforcer that denies `save_to_helix` → verify FactsSaved=0 |
| `TestBudgetExceeded` | `boomerang_test.go` | Create enforcer with budget=0, pass to DelegateStep → verify IsBudgetExceeded aborts |
| `TestEnforcerEnforcedInQualityStepRetry` | `boomerang_test.go` | Verify that DelegateStep within QualityStep retry passes the enforcer |
| `TestEvaluateCriteriaAllPass` | `quality_test.go` | All criteria pending, all tasks successful → verified |
| `TestEvaluateCriteriaOneFail` | `quality_test.go` | One criterion, task fails → failed |
| `TestEvaluateCriteriaNoCriteria` | `quality_test.go` | Empty acceptance criteria → true |
| `TestEvaluateCriteriaNilDAG` | `quality_test.go` | nil DAG → true |
| `TestApprovalGateWithFailedCriteria` | `approval_test.go` | CriteriaSummary.Failed > 0 → false |
| `TestApprovalGateNoCriteria` | `approval_test.go` | nil CriteriaSummary → normal behavior |

### 14.3 Integration Tests

| Test | Description |
|------|-------------|
| Phase runs with boundari | `RunPhase("F0")` → creates enforcer, dispatches tasks, audit log file exists |
| Phase with deny policy | Modify mock policy to deny dispatch → tasks skipped, audit logged |
| Pipeline with acceptance criteria | F1→F2 with criteria defined → QualityStep evaluates, ApprovalGate shows status |
| Pipeline without criteria (backward compat) | F0→F1→F2→F3→F4 without acceptance criteria → works without changes |
| CLI phase flag | `--phase PRE-F0` validates and runs |

### 14.4 Verification Checklist

```bash
# Compilation
go build ./...

# Boundari tests
go test ./internal/boundari/...

# Boomerang tests
go test ./internal/boomerang/...

# Scheduler tests
go test ./internal/scheduler/...

# All tests
go test ./...
```

### 14.5 Manual Smoke Test

```bash
# Create a test project with handoff.yaml
cd /tmp/test-project
zyrocli run --phase PRE-F0
# Expected: phase runs, tasks dispatched/failed according to PRE-F0 policy
# Expected: audit/boomerang-PRE-F0-*.jsonl created
# Expected: messages show "PRE-F0 → F0 → F1 → F2 → F3 → F4"
```

---

## 15. Risks and Mitigations

### 15.1 Risk Register

| ID | Risk | Impact | Probability | Mitigation |
|----|------|--------|-------------|------------|
| R1 | **YAMLs F0–F4 sin `dispatch_task`** → todas las tareas denegadas en fases existentes | **Pipeline roto** | **Alta** | Agregar `dispatch_task: allow` a todos los YAMLs existentes (Solución A del spec). Verificar con `TestAllPoliciesLoad` actualizado. |
| R2 | **PRE-F0 sin Boomerang en scheduler** → `PREF0Runner.Run()` abre OpenCode directamente, no pasa por `runPhaseV2()` → enforcement nunca se ejecuta | **No enforcement** | **Alta** | Verificar `scheduler.go:61-78`: cuando `cfg.Boomerang != nil`, se usa `Boomerang.RunPhase()` que SÍ pasa por `runPhaseV2()`. Si el scheduler está configurado con Boomerang, PRE-F0 tiene enforcement. Documentar en setup. |
| R3 | **Tests que llaman `DelegateStep`/`SaveStep` directamente sin enforcer** rompen con la nueva firma | **Compilación rota** | **Alta** | Las firmas cambian pero el nuevo parámetro es opcional (`nil` = legacy). Tests existentes se actualizan pasando `nil` — cambio mecánico. |
| R4 | **Audit logger singleton** no es thread-safe | **Race condition** | Baja (fases secuenciales) | `ClearAuditLog()` al inicio + `SaveAuditLog()` al final protegen el ciclo de vida. Si se implementa paralelismo, migrar a logger por instancia de Enforcer. |
| R5 | **Budget excedido aborta la fase abruptamente** | **Fase incompleta** | Media | Error controlado (no panic). Mensaje claro. Usuario puede re-ejecutar con budget mayor ajustando el YAML. |
| R6 | **PhaseConfig no tiene AcceptanceCriteria** → criteria no viajan de F1→F2 | **Criteria no se evalúan** | Media | `PhaseConfigV2` necesita campo `AcceptanceCriteria []AcceptanceCriteria` (futuro). Mientras tanto, criteria se definen inline en el DAG de `ThinkStep`. |
| R7 | **QualityStep con enforcer nil en llamadas directas de test** | **No evaluation** | Baja | Tests existentes que llaman `QualityStep` directamente (ej: `TestQualityStep`) no cambian firma. `evaluateCriteria()` se llama internamente, no requiere enforcer. |
| R8 | **ApprovalGate con nueva firma rompe llamadas existentes** | **Compilación rota** | Alta | `ApprovalGate` cambia de `(Phase, string)` a `(Phase, string, *CriteriaSummary)`. Todas las llamadas existentes en `scheduler.go` y tests deben actualizarse. El tercer parámetro puede ser `nil`. |
| R9 | **Fase PRE-F0 aparece en pipeline pero no tiene YAML ni default policy** | **Policy not found error** | **Alta** | Se crea YAML y se agrega case a `LoadDefaultPolicy()`. Resuelto en el diseño. |

### 15.2 Backward Compatibility Guarantees

1. **DelegateStep/SaveStep**: nuevo parámetro opcional `nil` = legacy behavior.
2. **TaskSpec.AcceptanceCriteria**: `omitempty` → fases sin criteria funcionan igual.
3. **QualityStep**: firma no cambia. `evaluateCriteria()` es interna y no afecta caller.
4. **PhaseMatrix**: no se modifica. PRE-F0 no se agrega a la default matrix (el scheduler la maneja con runners separados).
5. **ApprovalGate**: nuevo parámetro `nil` = no criteria to display.

### 15.3 Rollback Plan

If enforcement causes issues in production:
1. **Quick rollback (Phase 1)**: In `runPhaseV2()`, skip enforcer creation and pass `nil` to steps. Audit log won't be saved but everything else works.
2. **Full rollback (Phase 2)**: Revert the 7 Go files (not YAMLs — those are additive and safe to keep).
3. **YAML rollback**: Only need to revert if `dispatch_task: allow` in F0-F4 causes unintended behavior (unlikely — it's additive).

---

## 16. Implementation Order

### Phase 1: Foundation (Zero Regression Risk)

| Step | Files | Verification |
|------|-------|-------------|
| 1 | `internal/boundari/phasePRE-F0-boundari.yaml` (NEW) | `go test ./internal/boundari/...` |
| 2 | `internal/boundari/loader.go` (+ case PRE-F0) | `go test ./internal/boundari/...` |
| 3 | `internal/boundari/phase*-boundari.yaml` (add tools) | `go test ./internal/boundari/...` |
| 4 | `internal/boomerang/criteria.go` (NEW) | `go build ./internal/boomerang/...` |

### Phase 2: Core Enforcement (Medium Risk)

| Step | Files | Verification |
|------|-------|-------------|
| 5 | `internal/boomerang/delegate.go` (signature + CheckTool) | `go build ./internal/boomerang/...` |
| 6 | `internal/boomerang/save.go` (signature + CheckTool) | `go build ./internal/boomerang/...` |
| 7 | `internal/boomerang/orchestrator.go` (enforcer lifecycle + budget + audit) | `go build ./...` |
| 8 | `internal/boomerang/quality.go` (evaluateCriteria) | `go test ./internal/boomerang/...` |
| 9 | `internal/scheduler/approval.go` (CriteriaSummary) | `go build ./internal/scheduler/...` |
| 10 | `internal/scheduler/scheduler.go` (gate check) | `go build ./...` |

### Phase 3: Tests and Messages (Low Risk)

| Step | Files | Verification |
|------|-------|-------------|
| 11 | `cmd/zyrocli/run.go` (messages) | Manual |
| 12 | `internal/boomerang/boomerang_test.go` (update call sites) | `go test ./internal/boomerang/...` |
| 13 | `internal/boundari/boundari_test.go` (PRE-F0 tests) | `go test ./internal/boundari/...` |
| 14 | New tests (budget, enforcement, criteria) | `go test ./...` |

### Phase 4: Integration Verification

| Step | Verification |
|------|-------------|
| 15 | `go build ./...` |
| 16 | `go test ./...` |
| 17 | Manual: `zyrocli run --phase PRE-F0` in test project |
| 18 | Manual: `zyrocli run` pipeline display |

---

## Appendix A: File Change Diffs (Conceptual)

### A.1 `orchestrator.go` — `runPhaseV2()` with Enforcer

```go
// --- AFTER line 123 (result := &PhaseResult{Phase: config.Phase}) ---
// >>> 1. Crear Enforcer
policy, err := o.boundariLoader(config.Phase)
if err != nil {
    policy = boundari.LoadDefaultPolicy(config.Phase)
}
enforcer := boundari.NewEnforcer(policy)
boundari.ClearAuditLog()

// --- BEFORE each step in the loop (line 145) ---
for _, step := range steps {
    // >>> 2. Budget check
    if enforcer.IsBudgetExceeded() {
        return nil, fmt.Errorf("boundari: budget exceeded for phase %s", config.Phase)
    }
    // ... existing switch ...
```

### A.2 `delegate.go` — Signature and CheckTool Loop

```go
// OLD signature:
// func (o *BoomerangOrchestrator) DelegateStep(ctx context.Context, dag *TaskDAG, phase string) (*DelegateResult, error) {

// NEW signature:
func (o *BoomerangOrchestrator) DelegateStep(ctx context.Context, dag *TaskDAG, phase string, enforcer *boundari.Enforcer) (*DelegateResult, error) {

    // Inside the for _, task := range dag.Tasks loop, before DispatchTask:
    if enforcer != nil {
        checkResult := enforcer.CheckTool("dispatch_task", map[string]any{
            "task_name": task.Name,
            "agent":     task.Agent,
            "phase":     phase,
        })
        boundari.LogAudit(boundari.AuditEvent{
            Phase:   phase,
            Tool:    "dispatch_task",
            Allowed: checkResult.Allowed,
            Reason:  checkResult.Reason,
        })
        if !checkResult.Allowed {
            // skip task, add failed result
            continue
        }
    }
```

### A.3 `save.go` — Signature and CheckTool Loop

```go
// OLD signature:
// func (o *BoomerangOrchestrator) SaveStep(ctx context.Context, phase string, delegateResult *DelegateResult, logData []byte) (*SaveResult, error) {

// NEW signature:
func (o *BoomerangOrchestrator) SaveStep(ctx context.Context, phase string, delegateResult *DelegateResult, logData []byte, enforcer *boundari.Enforcer) (*SaveResult, error) {

    // Inside the for _, tr := range delegateResult.TaskResults loop, before SaveFact:
    if enforcer != nil {
        checkResult := enforcer.CheckTool("save_to_helix", map[string]any{
            "task_name": tr.TaskName,
            "phase":     phase,
        })
        boundari.LogAudit(boundari.AuditEvent{
            Phase:   phase,
            Tool:    "save_to_helix",
            Allowed: checkResult.Allowed,
            Reason:  checkResult.Reason,
        })
        if !checkResult.Allowed {
            continue
        }
    }
```

### A.4 `approval.go` — New Signature

```go
// OLD:
// func ApprovalGate(phase Phase, summary string) (bool, error) {

// NEW:
type CriteriaSummary struct {
    Total    int
    Pending  int
    Verified int
    Failed   int
}

func ApprovalGate(phase Phase, summary string, criteria *CriteriaSummary) (bool, error) {
```

---

## Appendix B: Design Decision Log

| Decision | Option Chosen | Alternatives Considered | Rationale |
|----------|--------------|------------------------|-----------|
| Enforcer lifecycle | Per `runPhaseV2()` call | Shared in struct | Different policy per phase, fresh budget |
| Enforcer param style | Optional `*Enforcer` (nil = skip) | Separate interface | Simpler, backward compatible, no interface pollution |
| Budget exceeded handling | Soft abort (error return) | Panic | Error is recoverable; panic breaks approval flow |
| Audit log path | `audit/boomerang-{phase}-{ts}.jsonl` | Single file append | One file per phase = clear separation |
| Audit log at phase end | Yes, as non-blocking warning | Yes, as blocking error | Phase result is more important than log |
| PRE-F0 policy source | YAML + default fallback | Only YAML | Resilience: if YAML not found, default policy works |
| Dispatch_task in F0-F4 YAMLs | Add to each | Change enforcer to "allow by default" | "Deny by default" is more secure; explicit allow is safer |
| QualityStep + criteria | `evaluateCriteria()` internal method | New step type | No DAG/step changes needed; backward compatible |
| ApprovalGate criteria display | Optional param | Always required | nil = no criteria to display = legacy behavior |

---

*End of Design Document*
