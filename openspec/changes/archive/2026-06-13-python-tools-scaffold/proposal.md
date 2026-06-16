# Proposal: python-tools-scaffold

## Intent

Scaffold generates AGENT.md, opencode.json, Go stubs — but zyro sub-agents lack tools to explore, test, and lint the codebase. Without these, zyro-reader and zyro-writer can't effectively navigate or verify the project. This change ships 3 standalone Python CLI tools as part of the scaffold output.

## Scope

### In Scope
- 3 Python scripts in `scripts/`: explorer, test-runner, linter
- `AGENT.md.tmpl` modified: add enforcement table (tool → agent)
- `scaffold.go` modified: embed + copy scripts/ to target dir
- `scaffold_test.go` modified: verify scripts/ and tool files exist

### Out of Scope
- Additional tools (formatter, type-checker, dependency graph, etc.)
- Template rendering for scripts (they are static files, not `.tmpl`)
- `opencode.json.tmpl` changes (bash permissions already cover this)
- `cmd/zyrocli/init.go` changes (no new flags)
- Go-specific tooling — Python only, zero external deps
- Tests for the Python scripts themselves (covered in target project)

## Capabilities

### New Capabilities
- `python-tools`: 3 standalone Python CLI scripts (explorer, test-runner, linter) for codebase exploration, test execution, and linting — bundled in the scaffold for zyro sub-agent invocation via bash.

### Modified Capabilities
None. `project-scaffold` spec covers ZyroCLI's own bootstrap, not scaffold output. `handoff-parser` unchanged.

## Approach

1. **Write 3 scripts** under `internal/scaffold/templates/go-project/scripts/`. Each follows the single-CLI pattern: `argparse` + `json.dumps(run(args))` — zero external deps, Python 3.8+.
2. **Embed as static files** in the existing `embed.FS`. No `.tmpl` suffix — read raw bytes, skip template rendering.
3. **scaffold.go**: add `scripts/*.py` entries to the jobs map with `outPath: "scripts/<name>.py"`. Use raw content read (not Render).
4. **AGENT.md.tmpl**: add enforcement table row linking each tool to its agent.
5. **scaffold_test.go**: verify `scripts/explorer.py`, `test-runner.py`, `linter.py` exist in output.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/scaffold/templates/go-project/scripts/explorer.py` | New | FS exploration: `--path`, `--pattern`, `--depth` |
| `internal/scaffold/templates/go-project/scripts/test-runner.py` | New | Test run + coverage: `--path`, `--coverage`, `--format` |
| `internal/scaffold/templates/go-project/scripts/linter.py` | New | Lint + auto-fix: `--path`, `--fix` |
| `internal/scaffold/templates/go-project/AGENT.md.tmpl` | Modified | Add enforcement table |
| `internal/scaffold/scaffold.go` | Modified | Copy scripts/ dir to target |
| `internal/scaffold/scaffold_test.go` | Modified | Assert scripts/ exists with tools |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Python 3 not in PATH on target | Low | AGENT.md documents python3 requirement; scripts check `sys.version_info` |
| Script path resolution from sub-agents | Low | Scripts use `os.path.dirname(__file__)`; target path is `<root>/scripts/` |
| Adding raw files to embed.FS confuses template loop | Low | Separate embed path — scripts use `templates/go-project/scripts/*.py`, templates stay at `templates/go-project/*.tmpl` |

## Rollback Plan

- **Code revert**: `git checkout -- internal/scaffold/templates/go-project/scripts/ internal/scaffold/templates/go-project/AGENT.md.tmpl internal/scaffold/scaffold.go internal/scaffold/scaffold_test.go`
- **Scaffold output**: scripts/ written alongside other files — `WriteProject` cleanup handles partial writes.

## Dependencies

- Python 3.8+ (target project runtime requirement)
- Go stdlib `embed` (already in use for templates)

## Success Criteria

- [ ] `go build ./...` compiles clean with embedded scripts
- [ ] `go test ./internal/scaffold/...` passes (incl. scripts/ existence check)
- [ ] Scaffold output contains `scripts/explorer.py`, `scripts/test-runner.py`, `scripts/linter.py`
- [ ] Each script runs standalone: `python3 scripts/explorer.py --path .` returns valid JSON
- [ ] AGENT.md in scaffold output includes tool enforcement table
