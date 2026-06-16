# Design: scaffold-opencode-init

## Technical Approach

Add `--scaffold` and `--opencode` flags to the existing `init` command. The scaffold engine lives in `internal/scaffold/` — three files: orchestrator (`scaffold.go`), template renderer (`renderer.go`), filesystem writer (`writer.go`). Templates are embedded via `//go:embed templates/*` in `internal/templates/`. The engine reads a `Config` derived from `handoff.Payload`, renders `.tmpl` files with `text/template`, writes the project tree, and optionally execs `opencode`. Zero external dependencies — stdlib only.

## Package Architecture

```
cmd/zyrocli/init.go          ──→  internal/scaffold/scaffold.go
  (parse payload)                 (orchestrates render → write → launch)
                                       │            │
                                       ▼            ▼
                                 renderer.go    writer.go
                                 (embed.FS +    (MkdirAll +
                                  text/template) WriteFile)
                                       │
                                       ▼
                                 internal/templates/
                                   go-project/*.tmpl
```

## Key Types

```go
// internal/scaffold/scaffold.go
type Config struct {
    ProjectName     string
    Language        string
    Module          string
    Problem         string
    SuccessCriteria string
    ScaffoldDir     string
    LaunchOpenCode  bool
}

type Result struct {
    TargetDir       string
    FilesCreated    int
    OpenCodeLaunched bool
}

func Run(cfg Config) (*Result, error)
```

## Sequence Diagram

```
init.RunE
  │
  ├─ handoff.Parse(path) → payload
  ├─ handoff.Validate(payload)
  │
  ├─ [if --scaffold]
  │    │
  │    ├─ scaffold.Run(Config{...from payload...})
  │    │    │
  │    │    ├─ sanitize(projectName) → safe name
  │    │    ├─ renderer.Render("AGENT.md", cfg) → string
  │    │    ├─ renderer.Render("opencode.json", cfg) → string
  │    │    ├─ renderer.Render("main.go", cfg) → string
  │    │    ├─ renderer.Render("README.md", cfg) → string
  │    │    ├─ renderer.Render(".gitignore", cfg) → string
  │    │    ├─ copy handoff.yaml verbatim
  │    │    ├─ writer.WriteProject(targetDir, files)
  │    │    │    ├─ sort keys (dirs first)
  │    │    │    ├─ os.MkdirAll for each dir prefix
  │    │    │    └─ os.WriteFile for each file
  │    │    │
  │    │    └─ [if LaunchOpenCode]
  │    │         ├─ exec.LookPath("opencode")
  │    │         ├─ exec.Command("opencode", targetDir).Run()
  │    │         └─ or warn if not found
  │    │
  │    └─ fmt.Printf("Project created at %s/\n", result.TargetDir)
  │
  └─ return nil
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/zyrocli/init.go` | Modify | Add `--scaffold`, `--opencode` flags; call `scaffold.Run()` in RunE after validate |
| `internal/scaffold/scaffold.go` | Create | Entry point: `Run(Config) (*Result, error)`. Orchestrates sanitize → render → write → launch. Cleans up on mid-write failure. |
| `internal/scaffold/renderer.go` | Create | `Renderer` struct with `//go:embed templates/*` FS. FuncMap: `lower`, `kebab`, `pascal`. `Render(name, cfg) (string, error)` parses `.tmpl` and executes to buffer. |
| `internal/scaffold/writer.go` | Create | `WriteProject(targetDir, files map[string]string) error`. Sorts keys (dirs before files), MkdirAll, WriteFile. Calls `os.RemoveAll(targetDir)` on any write error. |
| `internal/scaffold/scaffold_test.go` | Create | Table-driven tests: scaffold creates all files, name sanitization, module default, error on existing dir, cleanup on failure. |
| `internal/templates/go-project/AGENT.md.tmpl` | Create | Ultra-condensed agent rules (~350 chars): rules, stack, 4 phases, delegation table, sub-agents. |
| `internal/templates/go-project/opencode.json.tmpl` | Create | Valid JSON: zyro-agent (primary, read+task), zyro-reader (subagent, read+bash+write), zyro-writer (subagent, read+write+edit+bash), graphify (subagent, read). Uses `permission` object format. |
| `internal/templates/go-project/main.go.tmpl` | Create | Go entry point stub with package main and comments. |
| `internal/templates/go-project/.gitignore.tmpl` | Create | Standard Go .gitignore. |
| `internal/templates/go-project/README.md.tmpl` | Create | Project README with name, problem, language. |

## Interfaces / Contracts

The scaffold package has no interfaces — it's a pure functional pipeline. The only contract boundary is `scaffold.Config` which is populated from `handoff.Payload` fields. The `Run` function returns `(*Result, error)` following stdlib error conventions.

The renderer exposes:
```go
type Renderer struct { /* unexported fields */ }
func NewRenderer() *Renderer
func (r *Renderer) Render(name string, cfg Config) (string, error)
```

The writer exposes:
```go
func WriteProject(targetDir string, files map[string]string) error
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit: renderer | Template parsing, FuncMap helpers, render output | `NewRenderer().Render("AGENT.md", testCfg)` → assert non-empty, no error |
| Unit: writer | File creation, dir structure, cleanup on error | `WriteProject(t.TempDir()+"/test", files)` → walk dir, assert all files exist |
| Unit: scaffold | End-to-end Run with mock payload | `Run(cfg)` in `t.TempDir()` → assert all 8 files exist, correct content |
| Integration: init | Flag parsing + scaffold invocation | Cobra test harness or direct `initCmd.RunE` with test args |

## Migration / Rollout

No migration required. The `--scaffold` and `--opencode` flags are additive — existing `init` behavior is unchanged when flags are absent. The `internal/scaffold/` and `internal/templates/` packages are new and don't affect existing code paths.

## Resolved Decisions

- **Existing directory**: Ask the user with prompt "Directory {name}/ already exists. Overwrite? [y/N]". Normalize project name (lowercase, spaces→hyphens, trim specials).
- **handoff.yaml**: Template, not copy. Must be a `.tmpl` rendered from Config so Holdin Admin defines the contract format.
- **docs/ and skills/**: Create as empty directories in scaffold. Content populated in F1-F2 phases (context + GitMCP + skilladvisor). Skills are authored by us, not copied from upstream.
