# Proposal: fix-python-tools-compliance

## Intent

Fix spec compliance gaps in the 3 Python CLI scripts (explorer.py, test-runner.py, linter.py) identified during the SDD verification phase of the python-tools-scaffold cycle. Also add the `ReadScript()` helper and improve test coverage.

## Scope

### In Scope

1. **test-runner.py** — `coverage` output format: change from string `"13.7%"` to `{lines: N, percent: N}`. Change `errors` from raw stderr to `[{file, test, message}]`.
2. **linter.py** — No-config case: exit 0 with warning instead of exit 1. `fixed`: return actual count, not 1/0 flag. `issues`: add `severity` field. `issues` as list of structs with `{file, line, severity, message}`.
3. **scripts.go** — Add `ReadScript(name string) ([]byte, error)` helper to encapsulate the embed path prefix.
4. **scaffold_test.go** — Add Python runtime integration test (`exec.Command("python3", ...)`). Add explicit `## Tools por Fase` assertion in AGENT.md test.
5. **AGENT.md.tmpl** — No changes expected (already correct).
6. **spec update** — Update python-tools spec with delta requirements for output shapes and exit behavior.

### Out of Scope

- New scripts or features
- Changes to explorer.py (already compliant after post-verify fixes)
- Changes to scaffold.go writer or renderer

## Approach

1. Update `test-runner.py` output parsing to extract structured errors and coverage data.
2. Update `linter.py` no-config path (exit 0), add `severity` parsing, count actual fixes from linter output.
3. Add `ReadScript()` to `scripts.go`.
4. Add integration test using `exec.Command("python3", "scripts/explorer.py", ...)`.
5. Add explicit `## Tools por Fase` string assertion in `TestScaffoldCreatesAllFiles`.
6. Run `go test ./...` and `go vet ./...` to verify no regressions.
