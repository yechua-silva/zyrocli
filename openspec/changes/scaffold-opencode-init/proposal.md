# Proposal: scaffold-opencode-init

## Intent

`zyrocli init` only parses + validates handoff contracts. Users must manually scaffold projects and configure OpenCode. This change makes `init --scaffold` generate a complete, portable project from the contract, and `--scaffold --opencode` launch OpenCode in the new project — closing the gap between contract and coding environment.

## Scope

### In Scope
- `--scaffold` flag: generates project directory, AGENT.md, opencode.json, handoff.yaml (template), skills/ (empty), docs/ (empty), .gitignore, Go stubs, README.md
- `--opencode` flag: launches `opencode <targetDir>` after scaffold (requires `--scaffold`)
- `internal/scaffold/` package with renderer (text/template + embed.FS) and writer (os.MkdirAll + file write)
- `internal/templates/` — embedded `*.tmpl` files for all scaffold artifacts
- AGENT.md as ultra-condensado (~350 chars): rules + stack + 4-phase flow + delegation table
- `opencode.json` with inline sub-agents (zyro-reader, zyro-writer, graphify) + atomic permissions
- Unit tests for scaffold package + init command tests
- `cmd/zyrocli/init.go` modified: add `--scaffold`, `--opencode` flags

### Out of Scope
- SDD spec generation from handoff (F2 logic) — deferred
- Skill auto-detection beyond directory creation — F1 phase
- Template customization/choice — fixed set
- Non-Go language scaffolds — future
- `project-scaffold` spec update (that spec covers ZyroCLI's own bootstrap, not this feature)

## Capabilities

### New Capabilities
- `scaffold-engine`: Generate portable project structure from handoff contract payload — AGENT.md, opencode.json with zyro-agent architecture, Go stubs, docs, skills/ dir, .gitignore, README.md

### Modified Capabilities
- `handoff-parser`: CLI requirement changes — `init` subcommand gains optional `--scaffold` and `--opencode` flags; existing parse+validate behavior unchanged

## Approach

1. **Template engine**: Go `text/template` with `embed.FS` — zero external deps. FuncMap for kebab-case, lowercase, etc.
2. **Scaffold entry**: `internal/scaffold/scaffold.go` receives `*handoff.Payload` + flags. Returns `ScaffoldResult{TargetDir, OpenCodeLaunched}`.
3. **Renderer**: Reads embedded `.tmpl` files, parses with `template.FuncMap`, executes to `bytes.Buffer`.
4. **Writer**: `fs.WalkDir` over template dirs, `os.MkdirAll` for dirs, `os.WriteFile` for files. All paths under `<project-name>/`.
5. **opencode.json**: Inline sub-agents (zyro-reader, zyro-writer, graphify) with atomic `permission` blocks. AGENT.md injected via `{file:...}`.
6. **OpenCode launch**: `exec.Command("opencode", targetDir)` with inherited stdin/stdout/stderr. Non-blocking on success, error on missing binary.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/zyrocli/init.go` | Modified | Add `--scaffold`, `--opencode` flags; call `scaffold.Run()` |
| `internal/scaffold/scaffold.go` | New | Entry point, orchestrates render+write+launch |
| `internal/scaffold/renderer.go` | New | Template parsing + FuncMap |
| `internal/scaffold/writer.go` | New | fs.WalkDir + MkdirAll + WriteFile |
| `internal/scaffold/scaffold_test.go` | New | Unit tests with mock templates |
| `internal/templates/` | New | Embedded .tmpl files (AGENT, opencode.json, main.go, README, .gitignore, etc.) |
| `cmd/zyrocli/init_test.go` | New | Flag parsing + scaffold invocation |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `opencode` binary not in PATH | Medium | Check `exec.LookPath("opencode")` before launch; print friendly error |
| Template injection (project name in file path) | Low | Sanitize `project.Name`: strip slashes, dots, spaces |
| Handoff contract payload missing `governance.module` | Low | Default to `github.com/<project-name>` if empty |
| Destructive overwrite of existing dir | Low | Check `os.Stat(targetDir)`; error if exists unless `--force` (future) |

## Rollback Plan

- **Scaffold failed mid-write**: Call `os.RemoveAll(targetDir)` to clean up partial output
- **Bad template**: Template parse errors surface immediately at `scaffold.Run()`; no files written
- **OpenCode launch failed**: Print warning; scaffold is still valid
- **Revert code**: `git checkout -- cmd/zyrocli/init.go` + `git rm -r internal/scaffold/ internal/templates/`

## Dependencies

- Go stdlib: `text/template`, `embed`, `os/exec` (stdlib only — zero new deps)
- `opencode` binary in PATH (for `--opencode` flag only)

## Success Criteria

- [ ] `zyrocli init testdata/valid.yaml --scaffold` generates `<project-name>/` with all 8 required files
- [ ] `zyrocli init testdata/valid.yaml --scaffold --opencode` launches OpenCode in target dir
- [ ] `go build ./...` compiles clean with new packages
- [ ] `go test ./internal/scaffold/...` passes with 80%+ coverage
- [ ] Generated `opencode.json` passes JSON validation and references correct AGENT.md
