# Python Tools Specification — Delta

This document defines delta requirements for the fix-python-tools-compliance change.
All requirements from the main python-tools spec remain in effect unless superseded below.

## Modified Requirements

### Requirement: Script Bundle (exit behavior)

REPLACE "exit 0 on success and 1 on error" with:

> Each script MUST exit 0 on success and 1 on error, with JSON output to stdout in both cases (error field populated on failure). The "no linter config detected" case is NOT an error — linter.py MUST exit 0 with a warning when no config is found.

### Requirement: Test Runner Script (output shape)

REPLACE the output shape definition with:

> `{passed: N, failed: N, errors: [{file, test, message}], coverage: {lines: N, percent: N}}`
>
> - `errors` is a list of structured error objects. If the test runner returns unstructured output, parse stderr into structured entries.
> - `coverage` is an object with `lines` (total lines covered) and `percent` (coverage percentage as float), NOT a raw string. When `--coverage` is set and coverage data is available, parse the coverage output into this structure. When coverage is not requested or unavailable, `coverage` is `null`.

<details>
<summary>Scenarios</summary>

- **With coverage — structured output**: GIVEN `--coverage` flag and a Go project with coverage data; WHEN running test-runner; THEN `coverage` is `{lines: N, percent: N}` with numeric values.
- **Errors as list**: GIVEN test failures with file:line format; WHEN running test-runner; THEN `errors` is an array of `{file, test, message}` objects.
</details>

### Requirement: Linter Script (output shape + no-config behavior)

REPLACE the output shape definition with:

> `{issues: [{file, line, severity, message}], fixed: N, warnings: N}`
>
> - `severity` MUST be one of `"error"`, `"warning"`, `"info"`. Parse the linter output to extract severity.
> - `fixed` MUST be the actual count of issues fixed when `--fix` is used. Count by detecting "fixed" entries in the linter output. When `--fix` is not used, `fixed` is 0.
> - When no linter config is detected, exit 0 and return `{issues: [], fixed: 0, warnings: 1}`.

<details>
<summary>Scenarios</summary>

- **No config — exit 0**: GIVEN no linter config files; WHEN running linter; THEN exit code is 0 (not 1), `issues` is empty, `warnings` is 1.
- **Issues with severity**: GIVEN 2 errors and 1 warning from the linter; WHEN running linter; THEN `issues` has 3 entries each with a `severity` field.
- **Fix count**: GIVEN `--fix` flag and a linter that fixes 3 issues; WHEN running linter; THEN `fixed` is 3.
</details>

## New Requirements

### Requirement: ReadScript Helper

`scripts.go` MUST expose an exported `ReadScript(name string) ([]byte, error)` function that reads a script from the embedded filesystem by name (without the path prefix). Valid names are `"explorer.py"`, `"test-runner.py"`, `"linter.py"`.

<details>
<summary>Scenarios</summary>

- **Valid name**: GIVEN `ReadScript("explorer.py")`; WHEN called; THEN the full content of `templates/go-project/scripts/explorer.py` is returned without error.
- **Invalid name**: GIVEN `ReadScript("nonexistent.py")`; WHEN called; THEN an error is returned.
</details>

### Requirement: Python Runtime Integration Test

The Go test suite MUST include at least one test that runs a Python script via `exec.Command("python3", ...)` and validates its JSON output.

<details>
<summary>Scenarios</summary>

- **explorer.py runs**: GIVEN `python3 scripts/explorer.py --path .`; WHEN executed; THEN valid JSON is printed to stdout with `"error": null` or no error field.
- **explorer.py missing path**: GIVEN `python3 scripts/explorer.py` (no `--path`); WHEN executed; THEN exit code is 2 (argparse error).
</details>

### Requirement: AGENT.md Enforcement Table Test

The test for AGENT.md content MUST explicitly assert the presence of the string `"## Tools por Fase"`.

<details>
<summary>Scenarios</summary>

- **Enforcement table present**: GIVEN scaffold output; WHEN reading AGENT.md; THEN the string `## Tools por Fase` is found in the content.
</details>
