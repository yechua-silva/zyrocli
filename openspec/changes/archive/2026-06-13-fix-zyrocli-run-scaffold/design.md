# Design: fix-zyrocli-run-scaffold

## Technical Approach

Replace the SDD pipeline (scheduler.F1→F4) in `run.go` with the scaffold+opencode pattern already proven in `init.go` (lines 42-91). The "already initialized" path stays untouched. The change is surgically scoped to `cmd/zyrocli/run.go` — no other files are modified.

## Architecture Decisions

### Decision: Reuse init.go's scaffold pattern verbatim

**Choice**: Copy the handoff→Config→scaffold.Run→opencode pattern from init.go lines 42-91
**Alternatives considered**: Extract shared helper function between init.go and run.go
**Rationale**: YAGNI. Two call sites don't justify a helper yet. If a third command needs this pattern, extract then. Keeps the diff minimal and reviewable.

### Decision: Lenient opencode check (warn, don't fail)

**Choice**: If `exec.LookPath("opencode")` fails, warn to stderr and continue. Scaffold already created.
**Alternatives considered**: Fail hard before scaffold — prevent orphaned project dirs
**Rationale**: Matches spec requirement (lines 37-49 of spec.md). Scaffolded project is still useful without opencode — user can install opencode later and re-run.

### Decision: Remove --phase flag entirely

**Choice**: Delete `runPhase` var and the `--phase` flag registration
**Alternatives considered**: Keep flag but make it a no-op with deprecation warning
**Rationale**: Spec REMOVED requirements (lines 95-104) explicitly remove `--phase`. No backward compat needed — this is a breaking change by design.

## Data Flow

```
handoff.yaml
    │
    ▼
handoff.Parse("handoff.yaml")  ──→  Payload
    │
    ▼
handoff.Validate(payload)       ──→  error (if invalid)
    │
    ▼
os.ReadFile("handoff.yaml")     ──→  rawBytes (for RawHandoff field)
    │
    ▼
payload → scaffold.Config       ──→  cfg
    │
    ▼
exec.LookPath("opencode")       ──→  opencodeAvailable bool
    │
    ▼
scaffold.Run(cfg)               ──→  Result{TargetDir, FilesCreated}
    │
    ▼
if opencodeAvailable:
    exec.Command("opencode", result.TargetDir)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/zyrocli/run.go` | Modify | Replace SDD pipeline with scaffold+opencode flow |

No other files are created, modified, or deleted.

## Current Code (antes)

```go
// cmd/zyrocli/run.go — COMPLETE FILE (125 lines)

package main                                                    // L1
                                                                // L2
import (                                                        // L3
    "context"                                                   // L4  ← REMOVE
    "fmt"                                                       // L5
    "os"                                                        // L6
    "os/exec"                                                   // L7
                                                                // L8
    "github.com/secko/zyrocli/internal/handoff"                 // L9
    "github.com/secko/zyrocli/internal/scaffold"                // L10
    "github.com/secko/zyrocli/internal/scheduler"               // L11 ← REMOVE
    "github.com/spf13/cobra"                                    // L12
)                                                               // L13
                                                                // L14
var runPhase string                                             // L15 ← REMOVE
                                                                // L16
var runCmd = &cobra.Command{                                    // L17
    Use:   "run",                                               // L18
    Short: "Execute SDD pipeline (F1→F2→F3→F4)",                // L19 ← MODIFY
    Long: `Execute the 4-phase SDD pipeline sequentially...`,   // L20-25 ← MODIFY
    RunE: func(cmd *cobra.Command, args []string) error {       // L26
        // L27-50: Check initialized → open opencode [KEEP]
        // L52-55: Check handoff.yaml exists [KEEP]
        // L57-61: scheduler.LoadConfig [REPLACE]
        // L63-104: SDD pipeline runners [REPLACE]
        // L106-118: Print summary [REPLACE]
    },                                                          // L119
}                                                               // L120
                                                                // L121
func init() {                                                   // L122
    rootCmd.AddCommand(runCmd)                                  // L123
    runCmd.Flags().StringVarP(&runPhase, "phase", "p", ...)     // L124 ← REMOVE
}                                                               // L125
```

### What gets removed (lines 57-118):
- `scheduler.LoadConfig("handoff.yaml")` call
- `scheduler.NewScheduler()` + `scheduler.PhaseRunner` slice
- `scheduler.RunPhase()` / `scheduler.Run()` calls
- Phase validation loop (lines 76-88)
- Results summary printer (lines 106-118)
- `--phase` flag var and registration

### What stays untouched (lines 26-55):
- Project directory detection from `handoff.yaml` (lines 28-34)
- `scaffold.ReadState(projectDir)` check (lines 36-39)
- Initialized → open opencode path (lines 41-50)
- `handoff.yaml` existence check (lines 52-55)

## New Code (después)

```go
package main

import (
    "fmt"
    "os"
    "os/exec"

    "github.com/secko/zyrocli/internal/handoff"
    "github.com/secko/zyrocli/internal/scaffold"
    "github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
    Use:   "run",
    Short: "Initialize project from handoff contract and launch OpenCode",
    Long: `Parse handoff.yaml, scaffold the project structure, and launch OpenCode.

If the project is already initialized (.zyro/state.json exists),
opens OpenCode directly without re-scaffolding.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // ── Phase 1: Detect project directory ──────────────────
        // Unchanged from current code. Reads handoff.yaml to
        // extract project.name → used as projectDir for state check.
        projectDir := "."
        if _, err := os.Stat("handoff.yaml"); err == nil {
            payload, err := handoff.Parse("handoff.yaml")
            if err == nil && payload.Project.Name != "" {
                projectDir = payload.Project.Name
            }
        }

        // ── Phase 2: Check if already initialized ─────────────
        // Unchanged. If .zyro/state.json exists with Initialized=true,
        // open opencode and return immediately.
        state, err := scaffold.ReadState(projectDir)
        if err != nil {
            return fmt.Errorf("run: checking project state: %w", err)
        }

        if state != nil && state.Initialized {
            cmd.Printf("✓ Proyecto ya inicializado: %s\n", state.ProjectName)
            cmd.Println("  Abriendo OpenCode...")

            openCmd := exec.Command("opencode", projectDir)
            openCmd.Stdin = os.Stdin
            openCmd.Stdout = os.Stdout
            openCmd.Stderr = os.Stderr
            return openCmd.Run()
        }

        // ── Phase 3: Scaffold from handoff.yaml ───────────────
        // NEW: replaces the SDD pipeline (old lines 57-118).

        // 3a. Verify handoff.yaml exists (same guard as before)
        if _, err := os.Stat("handoff.yaml"); os.IsNotExist(err) {
            return fmt.Errorf("run: handoff.yaml not found in current directory\nRun 'zyrocli init <file>' first")
        }

        // 3b. Parse and validate the handoff contract
        payload, err := handoff.Parse("handoff.yaml")
        if err != nil {
            return fmt.Errorf("run: %w", err)
        }

        if err := handoff.Validate(payload); err != nil {
            return fmt.Errorf("run: validation failed:\n%v", err)
        }

        // 3c. Read raw YAML bytes for template reference
        rawBytes, err := os.ReadFile("handoff.yaml")
        if err != nil {
            return fmt.Errorf("run: reading handoff: %w", err)
        }

        // 3d. Map handoff payload → scaffold.Config
        //     Mirrors init.go lines 56-67 exactly.
        cfg := scaffold.Config{
            ProjectName:     payload.Project.Name,
            Language:        payload.Project.Language,
            Module:          payload.Project.Repository,
            Problem:         payload.ValidatedIdea.Problem,
            SuccessCriteria: payload.UserStory.Acceptance,
            ScaffoldDir:     payload.Project.Name,
            LaunchOpenCode:  true,
            RawHandoff:      string(rawBytes),
            Version:         payload.Version,
            Source:          payload.Source.System,
        }

        // 3e. Check opencode availability (lenient — warn, don't fail)
        //     Spec: "warn to stderr and continue — scaffold is already done"
        opencodeAvailable := true
        if _, err := exec.LookPath("opencode"); err != nil {
            cmd.PrintErrln("⚠ opencode not found in PATH. Install it to launch after scaffold.")
            opencodeAvailable = false
            cfg.LaunchOpenCode = false
        }

        // 3f. Execute scaffold
        result, err := scaffold.Run(cfg)
        if err != nil {
            return fmt.Errorf("run: %w", err)
        }

        cmd.Printf("✓ Project scaffolded at %s/\n", result.TargetDir)
        cmd.Printf("  Files created: %d\n", result.FilesCreated)

        // ── Phase 4: Launch OpenCode ──────────────────────────
        if opencodeAvailable {
            openCmd := exec.Command("opencode", result.TargetDir)
            openCmd.Stdin = os.Stdin
            openCmd.Stdout = os.Stdout
            openCmd.Stderr = os.Stderr
            if err := openCmd.Run(); err != nil {
                return fmt.Errorf("run: opencode: %w", err)
            }
            cmd.Println("OpenCode session ended. Happy coding!")
        }

        return nil
    },
}

func init() {
    rootCmd.AddCommand(runCmd)
}
```

## Imports

| Import | Status | Reason |
|--------|--------|--------|
| `context` | **REMOVE** | No more scheduler; context only used by scheduler.Run |
| `fmt` | KEEP | Error formatting |
| `os` | KEEP | os.Stat, os.ReadFile |
| `os/exec` | KEEP | exec.Command, exec.LookPath |
| `handoff` | KEEP | handoff.Parse, handoff.Validate |
| `scaffold` | KEEP | scaffold.ReadState, scaffold.Run |
| `scheduler` | **REMOVE** | Entire SDD pipeline removed |
| `cobra` | KEEP | Cobra command |

## Dependencies

| Package | Usage |
|---------|-------|
| `internal/handoff` | `Parse()` → Payload, `Validate()` → error |
| `internal/scaffold` | `ReadState()` → State, `Run()` → Result |
| `os/exec` | `LookPath()` for opencode check, `Command()` for launch |

Removed dependency: `internal/scheduler` (entire package no longer referenced).

## Error Handling

| Error | Source | Behavior |
|-------|--------|----------|
| handoff.yaml not found | `os.Stat` | Return error with hint to run `zyrocli init` |
| handoff.yaml parse error | `handoff.Parse` | Return wrapped error: `run: <parse error>` |
| handoff.yaml validation failed | `handoff.Validate` | Return error with validation details |
| Raw handoff read failure | `os.ReadFile` | Return wrapped error |
| State file corrupt | `scaffold.ReadState` | Return wrapped error (existing behavior) |
| Scaffold target exists | `scaffold.Run` | Return error "directory already exists" (scaffold handles) |
| Scaffold filesystem error | `scaffold.Run` | Return wrapped error |
| opencode not in PATH | `exec.LookPath` | **Warn to stderr**, set `opencodeAvailable = false`, continue |
| opencode fails to launch | `exec.Command.Run` | Return wrapped error after scaffold is complete |

## Integration Points

- **scaffold.ReadState(projectDir)**: Called before scaffold to check if project already initialized. Returns `nil` if no state file exists (not initialized).
- **handoff.Parse("handoff.yaml")**: Called twice — once at Phase 1 (light, for projectDir extraction) and once at Phase 3b (full, for scaffold config). This mirrors init.go's pattern and avoids restructuring the early detection block.
- **handoff.Validate(payload)**: Called once after full parse. Returns `errors.Join` of all validation failures.
- **scaffold.Run(cfg)**: Called with Config mapped from handoff payload. Returns `*Result` with `TargetDir` and `FilesCreated`.
- **exec.Command("opencode", dir)**: Launched only if `opencodeAvailable` is true. Stdin/stdout/stderr piped to os.

## Edge Cases

### Double handoff.Parse
`handoff.Parse("handoff.yaml")` is called twice: once at Phase 1 (lines 29-34) for projectDir extraction, and once at Phase 3b for full payload. This is intentional — the Phase 1 call is guarded by `os.Stat` and errors are silently ignored (best-effort). Phase 3b is the authoritative parse with error handling. Refactoring to pass the payload from Phase 1 would require restructuring the early-return initialized path, adding complexity for no benefit.

### scaffold.Run directory collision
If `projectDir` matches an existing directory, `scaffold.Run` returns an error. The user sees the error and can decide to remove the directory or use a different project name in handoff.yaml.

### opencode launch after scaffold
If opencode is found in PATH but fails to launch (e.g., binary corrupted), the error is returned after the scaffold is complete. The project directory exists and is usable — the user can install a working opencode and re-run `zyrocli run`.

### No approval gates
The old SDD pipeline had human-validation approval gates between phases. These are completely removed. The new flow is fully automatic: parse → validate → scaffold → launch.

### --phase flag removed
The `runPhase` var and `--phase` flag registration are deleted. The `init()` function only calls `rootCmd.AddCommand(runCmd)`. No other flags are added.

## Migration / Rollout

No migration required. This is a behavioral change to an existing command. Users who relied on `zyrocli run` to execute SDD phases will need to use a different workflow. The `scheduler` package remains in the codebase (other commands may reference it) but is no longer imported by `run.go`.

## Open Questions

- None. All decisions align with the formal spec (spec.md) and the user's explicit requirements.
