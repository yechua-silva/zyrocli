# Tasks: fix-python-tools-compliance

## Estimated changed lines: 80-120

### Task 1: Fix linter.py

- No-config: exit 0 instead of exit 1 (remove `sys.exit(1)` from no-config branch).
- Parse linter JSON output to extract `severity` field.
- Count actual fixed entries from linter output.
- Update no-config return to `{issues: [], fixed: 0, warnings: 1}`.

**Files**: `internal/scaffold/templates/go-project/scripts/linter.py`
**Test**: `TestPythonScriptsRun` covers explorer.py; linter.py tested via code inspection.

### Task 2: Fix test-runner.py

- Parse `coverage.out` into `{lines: N, percent: N}` instead of raw string.
- Parse stderr into structured `errors: [{file, test, message}]`.
- Keep `coverage: null` when no coverage data available.

**Files**: `internal/scaffold/templates/go-project/scripts/test-runner.py`
**Test**: `TestPythonScriptsRun` covers explorer.py; test-runner.py changes verified by code inspection.

### Task 3: Add ReadScript() helper

- Add `ReadScript(name string) ([]byte, error)` to `scripts.go`.
- Prepend `templates/go-project/scripts/` to name before reading from `scriptsFS`.
- Return error for invalid names.

**Files**: `internal/scaffold/scripts.go`
**Test**: Existing `TestScaffoldScriptsExist` passes; add `TestReadScript` in scaffold_test.go.

### Task 4: Add Python runtime integration test

- Add `TestPythonScriptsRun` that runs `exec.Command("python3", "scripts/explorer.py", "--path", ".")` on scaffold output.
- Validate JSON output contains `total_files` and `languages`.
- Test `--path` with a temp directory to verify basic execution.

**Files**: `internal/scaffold/scaffold_test.go`
**Test**: The test itself is the verification.

### Task 5: Add enforcement table assertion

- In `TestScaffoldCreatesAllFiles`, add assertion that `agentContent` contains `"## Tools por Fase"`.

**Files**: `internal/scaffold/scaffold_test.go`
**Test**: Part of `TestScaffoldCreatesAllFiles`.

## Review Workload Forecast

- **Estimated changed lines**: 80-120
- **400-line budget risk**: Low
- **Decision needed before apply**: No
