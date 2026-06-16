# Design: fix-python-tools-compliance

## Technical Approach

Minimal changes to 3 Python scripts + 1 Go file + 1 test file.

### test-runner.py changes

- Parse `coverage.out` to extract `lines` and `percent` as separate numeric fields.
- Parse stderr into structured `errors` list by splitting on "--- FAIL" markers.
- Keep existing framework detection and command execution intact.

### linter.py changes

- No-config case: `print(json.dumps(...))` without `sys.exit(1)` (let it fall through to normal exit).
- Parse linter JSON output to extract `severity` field (golangci-lint output is already JSON; for others, parse by pattern).
- Count actual fixed issues from linter output when `--fix` is used.

### scripts.go changes

- Add `ReadScript(name string) ([]byte, error)` that reads from `scriptsFS` and strips the `templates/go-project/scripts/` prefix.

### scaffold_test.go changes

- Add `TestPythonScriptsRun` using `exec.Command("python3", ...)` for `explorer.py`.
- Add `## Tools por Fase` string assertion in `TestScaffoldCreatesAllFiles` (or a new focused test).

## File Changes

| File | Change |
|------|--------|
| `internal/scaffold/templates/go-project/scripts/test-runner.py` | Parse coverage as struct, errors as list |
| `internal/scaffold/templates/go-project/scripts/linter.py` | No-config exit 0, add severity, count fixes |
| `internal/scaffold/scripts.go` | Add `ReadScript()` |
| `internal/scaffold/scaffold_test.go` | Add Python runtime test + Tools por Fase assertion |

## Testing Strategy

1. `go test ./internal/scaffold/...` — existing + new tests
2. `go vet ./internal/scaffold/...` — no regressions
3. `go build ./...` — clean build
