# Design: python-tools-scaffold

## Technical Approach

Add 3 Python CLI scripts as static embedded files to the scaffold output. Scripts live at `templates/go-project/scripts/*.py` and are embedded via a separate `embed.FS` (the existing `templateFS` uses non-recursive `*` and cannot reach subdirectories). `scaffold.go` reads scripts as raw bytes after template rendering, adds them to the `files` map, and `WriteProject` handles directory creation. `AGENT.md.tmpl` gets a new enforcement table section.

## Architecture Decisions

### Decision: Separate embed.FS for scripts

**Choice**: New `//go:embed templates/go-project/scripts/*` directive in a dedicated `scripts.go` file.
**Alternatives**: (a) Change existing glob to recursive — Go embed doesn't support `**`. (b) Move scripts to `templates/go-project/` root — pollutes template directory with non-template files.
**Rationale**: Isolation. Templates stay as `.tmpl` files with `Render()`. Scripts stay as raw `.py` with `ReadFile()`. No risk of template engine choking on Python f-strings or curly braces.

### Decision: Raw bytes, no template rendering

**Choice**: Read scripts via `fs.ReadFile(scriptFS, path)` and cast to string. No `Renderer.Render()`.
**Alternatives**: (a) Use `.tmpl` suffix and render — risky with Python `{}` syntax. (b) Template the project name into scripts — unnecessary, scripts use `__file__` for paths.
**Rationale**: Scripts are self-contained. They discover their context at runtime via `os.path`. Template rendering adds complexity for zero benefit and risks breaking Python syntax.

### Decision: Scripts as files, not empty dir markers

**Choice**: Add `scripts/explorer.py`, `scripts/test-runner.py`, `scripts/linter.py` to the `files` map. `WriteProject` creates `scripts/` automatically via `MkdirAll(filepath.Dir(...))`.
**Alternatives**: (a) Add `scripts/` empty dir marker + files — redundant since files create the dir.
**Rationale**: Simpler. `WriteProject` already handles parent directory creation. No need for explicit dir marker.

## Data Flow

```
scaffold.Run(cfg)
  │
  ├─ renderer.Render() × 6 jobs → files["AGENT.md"], files["opencode.json"], ...
  │
  ├─ fs.ReadFile(scriptFS, "templates/go-project/scripts/explorer.py")
  │  fs.ReadFile(scriptFS, "templates/go-project/scripts/test-runner.py")
  │  fs.ReadFile(scriptFS, "templates/go-project/scripts/linter.py")
  │     │
  │     └─ files["scripts/explorer.py"] = string(rawBytes)
  │        files["scripts/test-runner.py"] = string(rawBytes)
  │        files["scripts/linter.py"] = string(rawBytes)
  │
  ├─ files["skills/"] = ""
  ├─ files["docs/contexto_proyecto/"] = ""
  │
  └─ WriteProject(targetDir, files) → 9 files + 3 dirs
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/scaffold/templates/go-project/scripts/explorer.py` | Create | Directory walker: `--path`, `--pattern`, `--depth`. Returns `{files, dirs, total_files, languages}`. |
| `internal/scaffold/templates/go-project/scripts/test-runner.py` | Create | Test runner: `--path`, `--coverage`, `--format`. Detects framework (go/pytest/jest), runs tests, returns `{passed, failed, errors, coverage}`. |
| `internal/scaffold/templates/go-project/scripts/linter.py` | Create | Linter: `--path`, `--fix`. Detects config (golangci/ruff/flake8/eslint), runs linter, returns `{issues, fixed, warnings}`. |
| `internal/scaffold/scripts.go` | Create | `//go:embed templates/go-project/scripts/*` + `var scriptFS embed.FS` + exported `ReadScript(name string) ([]byte, error)` helper. |
| `internal/scaffold/scaffold.go` | Modify | After template loop: read 3 scripts via `ReadScript()`, add to `files` map. Update `FilesCreated` to `len(jobs) + 3`. Add `files["scripts/"] = ""` is NOT needed (files create dir). |
| `internal/scaffold/templates/go-project/AGENT.md.tmpl` | Modify | Append `## Tools por Fase` section with enforcement table (F1→F4, tool→agent mapping). |
| `internal/scaffold/scaffold_test.go` | Modify | Add `TestScaffoldScriptsExist`, `TestScaffoldScriptContent`, `TestScaffoldFilesCreatedCount`, `TestScaffoldAgentMdEnforcement`. Update existing `TestScaffoldCreatesAllFiles` to check 9 files. |

## Interfaces / Contracts

### scripts.go

```go
package scaffold

import "embed"

//go:embed templates/go-project/scripts/*
var scriptFS embed.FS

// ReadScript returns the raw bytes of an embedded script by filename.
func ReadScript(name string) ([]byte, error) {
    return scriptFS.ReadFile("templates/go-project/scripts/" + name)
}
```

### Script CLI contract (all 3 scripts)

```python
# Usage: python3 <script>.py [args]
# Output: JSON to stdout
# Exit: 0 = success, 1 = error
# {
#   "result": { ... },  # tool-specific payload
#   "error": null        # null on success, string on failure
# }
```

### explorer.py args

```
--path PATH    (required)  Directory to explore
--pattern GLOB (default: "*")  File glob filter
--depth N      (default: 3)    Max directory depth
```

### test-runner.py args

```
--path PATH    (default: ".")   Project root
--coverage     (flag)           Include coverage data
--format FMT   (default: "json")  Output format (json only for now)
```

### linter.py args

```
--path PATH    (default: ".")   Project root
--fix          (flag)           Auto-fix issues
```

### scaffold.go change (pseudocode)

```go
// After template loop, before WriteProject:
scriptNames := []struct{ name, outPath string }{
    {"explorer.py", "scripts/explorer.py"},
    {"test-runner.py", "scripts/test-runner.py"},
    {"linter.py", "scripts/linter.py"},
}
for _, s := range scriptNames {
    raw, err := ReadScript(s.name)
    if err != nil {
        return nil, fmt.Errorf("scaffold: read script %s: %w", s.name, err)
    }
    files[s.outPath] = string(raw)
}

// Update result:
FilesCreated: len(jobs) + len(scriptNames),
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit: scripts.go | `ReadScript` returns valid bytes | Call `ReadScript("explorer.py")`, assert non-empty, starts with `#!/usr/bin/env python3` |
| Unit: scaffold | 3 script files in output | `Run(cfg)` in `t.TempDir()` → `os.Stat("scripts/explorer.py")` etc. |
| Unit: scaffold | FilesCreated = 9 | `result.FilesCreated == 9` |
| Unit: scaffold | AGENT.md has enforcement table | Read `AGENT.md` from output, assert contains `## Tools por Fase` |
| Integration | Scripts run standalone | `exec.Command("python3", "scripts/explorer.py", "--path", ".")` in scaffold output dir → JSON parse succeeds |

## Migration / Rollout

No migration required. Scripts are additive — existing scaffold output gains a `scripts/` directory. `FilesCreated` changes from 6 to 9, which is a visible but harmless change. No flags, no config changes, no breaking changes.
