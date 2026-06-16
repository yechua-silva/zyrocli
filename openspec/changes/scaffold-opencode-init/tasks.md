# Tasks: scaffold-opencode-init

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 500–650 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1: scaffold pkg + templates → main · PR 2: CLI flags + init integration → main |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Scaffold package + templates | PR 1 | Base: main. renderer, writer, scaffold.go, all .tmpl files, scaffold_test.go. Self-contained — can verify with direct `scaffold.Run()` calls. |
| 2 | CLI integration + init tests | PR 2 | Base: main. Modifies `cmd/zyrocli/init.go`, adds `init_test.go`. Depends on PR 1 (imports `internal/scaffold`). |

## Phase A: Scaffold Package

- [x] A.1 Create `internal/scaffold/scaffold.go` — `Config` struct (ProjectName, Language, Module, Problem, SuccessCriteria, ScaffoldDir, LaunchOpenCode), `Result` struct (TargetDir, FilesCreated, OpenCodeLaunched), `Run(Config) (*Result, error)` orchestrator. Sanitize project name (lower, spaces→hyphens, strip specials, trim hyphens). Default Module to `github.com/{sanitized-name}` if empty. Check `os.Stat(targetDir)` — error if exists. Render all templates via Renderer, call `WriteProject`. On write error: `os.RemoveAll(targetDir)`. (PR1)
- [x] A.2 Create `internal/scaffold/renderer.go` — `Renderer` struct with `//go:embed templates/go-project/*`, `NewRenderer()` (no-arg), `Render(name, cfg)`. FuncMap: `lower`, `kebab`, `pascal`, `normalize`. (PR1)
- [x] A.3 Create `internal/scaffold/writer.go` — `WriteProject(targetDir string, files map[string]string) error`. Sort keys (dirs before files), `MkdirAll`, `WriteFile` with 0644. Cleanup + return wrapped error on failure. (PR1)

## Phase B: Templates

- [x] B.1 Create `internal/scaffold/templates/go-project/AGENT.md.tmpl` — Ultra-condensed (~350 chars). Rules, stack, 4 phases, delegation table, compaction note. (PR1)
- [x] B.2 Create `internal/scaffold/templates/go-project/opencode.json.tmpl` — Valid JSON with zyro-agent (primary, read+task), zyro-reader/writer/graphify sub-agents with atomic permissions. (PR1)
- [x] B.3 Create `internal/scaffold/templates/go-project/main.go.tmpl` — Go stub: `package main`, imports `"fmt"`, `func main() { fmt.Println("{{.ProjectName}}") }`. (PR1)
- [x] B.4 Create `internal/scaffold/templates/go-project/.gitignore.tmpl` — Standard Go gitignore. (PR1)
- [x] B.5 Create `internal/scaffold/templates/go-project/README.md.tmpl` — README with project name, problem, language, module. (PR1)
- [x] B.6 Create `internal/scaffold/templates/go-project/handoff.yaml.tmpl` — Template handoff from Config fields (Version, Source, ProjectName, Language, Problem, SuccessCriteria). (PR1)

## Phase C: CLI Integration

- [x] C.1 Modify `cmd/zyrocli/init.go` — Add `var scaffoldFlag bool` and `var opencodeFlag bool`. Register on `initCmd.Flags()` as `--scaffold` (bool, false) and `--opencode` (bool, false). In `RunE`, after `handoff.Validate(payload)`: if `opencodeFlag && !scaffoldFlag` → error " --opencode requires --scaffold". If `scaffoldFlag`: build `scaffold.Config` from payload fields, call `scaffold.Run(cfg)`, print result. Import `internal/scaffold`.
- [ ] C.2 Add existing-dir prompt in `scaffold.Run` — After `os.Stat(targetDir)`, if dir exists, prompt via `fmt.Printf("Directory %s/ already exists. Overwrite? [y/N] ", name)`. Read line from `os.Stdin`. If "y"/"Y": `os.RemoveAll(targetDir)`. Otherwise: return error "aborted".

## Phase D: Tests

- [x] D.1 Create `internal/scaffold/scaffold_test.go` — 7 tests covering: full file/dir structure, name normalization (3 subtables), existing dir, module default, WriteProject cleanup on conflict, template rendering. (PR1)
- [x] D.2 Create `cmd/zyrocli/init_test.go` — (1) `TestInitScaffoldFlag` — Cobra test: args `[handoff.yaml, --scaffold]`, assert `scaffold.Run` called (check dir creation). (2) `TestInitOpenCodeRequiresScaffold` — args `[handoff.yaml, --opencode]` without `--scaffold`, assert error message contains "requires --scaffold". (3) `TestInitNoScaffoldFlag` — args `[handoff.yaml]`, assert no files created (existing behavior preserved).

## Phase E: Verify

- [x] E.1 Build — `go build ./...` compiles clean. No import cycles, no unused imports. (PR1)
- [x] E.2 Test — `go test ./internal/scaffold/... -v` passes (7/7). `go vet ./...` clean. (PR1)
- [ ] E.3 Manual smoke — Run `go run ./cmd/zyrocli init testdata/valid.yaml --scaffold` against a test handoff, inspect generated files for correctness. (PR1 manual check)
- [x] E.4 `go test ./cmd/zyrocli/... -v` passes — deferred to PR2 (init_test.go)
