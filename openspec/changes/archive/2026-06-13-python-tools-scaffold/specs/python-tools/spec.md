# Python Tools Specification

## Purpose

Define 3 standalone Python CLI scripts (explorer, test-runner, linter) bundled in the scaffold output. These tools give zyro-reader and zyro-writer the ability to explore, test, and lint the target codebase without external dependencies.

## Requirements

### Requirement: Script Bundle

The scaffold MUST include 3 Python scripts in `scripts/`: `explorer.py`, `test-runner.py`, `linter.py`. Each script MUST follow the single-CLI pattern: `argparse` for args, `json.dumps(run(args))` for output. Each script MUST be zero external deps, Python 3.8+ stdlib only. Each script MUST exit 0 on success and 1 on error, with JSON output to stdout in both cases (error field populated on failure).

<details>
<summary>Scenarios</summary>

- **All scripts present**: GIVEN a scaffolded project; WHEN inspecting `scripts/`; THEN `explorer.py`, `test-runner.py`, `linter.py` exist and are non-empty.
- **Standalone execution**: GIVEN `scripts/explorer.py`; WHEN running `python3 scripts/explorer.py --path .`; THEN valid JSON is printed to stdout with `"error": null`.
- **Missing required args**: GIVEN `scripts/explorer.py`; WHEN running `python3 scripts/explorer.py` (no `--path`); THEN exit code is non-zero and JSON error is printed.
- **Python version guard**: GIVEN Python 2 or < 3.8; WHEN running any script; THEN exit code is non-zero with a clear version error message.
</details>

### Requirement: Explorer Script

`explorer.py` MUST accept `--path` (required), `--pattern` (default `"*"`), `--depth` (default `3`). It MUST walk the directory tree up to `--depth` levels, detect file types by extension, and return JSON: `{files: [{path, type, size}], dirs: N, total_files: N, languages: {".go": 5, ".py": 3}}`. The `type` field MUST be the file extension (e.g. `".go"`, `".py"`). Directories MUST be skipped from the `files` array but counted in `dirs`.

<details>
<summary>Scenarios</summary>

- **Walk with depth**: GIVEN a project with 3 levels of nesting; WHEN running `python3 scripts/explorer.py --path . --depth 2`; THEN only files up to 2 levels deep are returned.
- **Pattern filter**: GIVEN `--pattern "*.go"`; WHEN running explorer; THEN only `.go` files appear in the output.
- **Language counting**: GIVEN 5 `.go` files and 3 `.py` files; WHEN running explorer; THEN `languages` is `{".go": 5, ".py": 3}`.
- **Empty directory**: GIVEN an empty directory; WHEN running explorer; THEN `{files: [], dirs: 0, total_files: 0, languages: {}}`.
- **Non-existent path**: GIVEN `--path /nonexistent`; WHEN running explorer; THEN exit code is 1 and error JSON is returned.
</details>

### Requirement: Test Runner Script

`test-runner.py` MUST accept `--path` (default `"."`), `--coverage` (flag, default false), `--format` (default `"json"`). It MUST detect the test framework by checking for `go.mod` (→ `go test`), `pytest.ini`/`pyproject.toml` (→ `pytest`), `package.json` with jest config (→ `jest`), or `Makefile` with test target (→ `make test`). It MUST run the detected command, capture stdout/stderr, and return JSON: `{passed: N, failed: N, errors: [{file, test, message}], coverage: {lines: N, percent: N}}`. If `--coverage` is set, it MUST attempt to extract coverage data from the test output. If no framework is detected, it MUST return an error.

<details>
<summary>Scenarios</summary>

- **Go project**: GIVEN a `go.mod` in `--path`; WHEN running test-runner; THEN `go test ./...` is executed and results are returned.
- **Python project**: GIVEN a `pyproject.toml` with `[tool.pytest]`; WHEN running test-runner; THEN `pytest` is executed.
- **No framework**: GIVEN an empty directory with no markers; WHEN running test-runner; THEN exit code is 1 and error indicates "no test framework detected".
- **With coverage**: GIVEN `--coverage` flag and a Go project; WHEN running test-runner; THEN coverage data is extracted from `go test -coverprofile` output.
- **Test failures**: GIVEN 2 passing and 1 failing test; WHEN running test-runner; THEN `passed: 2, failed: 1` and the failing test appears in `errors`.
</details>

### Requirement: Linter Script

`linter.py` MUST accept `--path` (default `"."`), `--fix` (flag, default false). It MUST detect the linter by checking for `.golangci.yml` (→ `golangci-lint run`), `ruff.toml`/`pyproject.toml` with ruff config (→ `ruff check`), `.flake8` (→ `flake8`), or `eslint.config.js`/`.eslintrc.*` (→ `eslint`). If `--fix` is set, it MUST pass `--fix` to the underlying linter. It MUST capture issues and return JSON: `{issues: [{file, line, severity, message}], fixed: N, warnings: N}`. The `severity` field MUST be one of `"error"`, `"warning"`, `"info"`. If no linter config is detected, it MUST return a warning (not error) with empty issues.

<details>
<summary>Scenarios</summary>

- **Go linting**: GIVEN `.golangci.yml` exists; WHEN running linter; THEN `golangci-lint run` is executed and issues are returned.
- **Python linting**: GIVEN `ruff.toml` exists; WHEN running linter; THEN `ruff check` is executed.
- **Auto-fix**: GIVEN `--fix` flag and ruff config; WHEN running linter; THEN `ruff check --fix` is executed and `fixed` count is returned.
- **No linter config**: GIVEN no linter config files; WHEN running linter; THEN exit code is 0, `issues` is empty, and a warning is included.
- **Issues found**: GIVEN 3 errors and 2 warnings; WHEN running linter; THEN `issues` has 5 entries and `warnings: 2`.
</details>

### Requirement: Scaffold Integration

`scaffold.go` MUST embed the scripts via a separate `embed.FS` (the existing `templateFS` uses non-recursive `*` glob and won't match subdirectories). After rendering template jobs, `Run()` MUST read each script from the script FS as raw bytes (no `text/template` rendering) and add them to the `files` map with paths like `scripts/explorer.py`. `FilesCreated` MUST reflect the total count of templates + scripts.

<details>
<summary>Scenarios</summary>

- **Scripts embedded**: GIVEN `//go:embed templates/go-project/scripts/*` in a script FS; WHEN `scaffold.Run()` executes; THEN `scripts/explorer.py`, `scripts/test-runner.py`, `scripts/linter.py` exist in the output directory.
- **FilesCreated count**: GIVEN 6 template jobs + 3 scripts; WHEN scaffold completes; THEN `result.FilesCreated` is `9`.
- **Raw content preserved**: GIVEN a script with `#!/usr/bin/env python3`; WHEN scaffold writes it; THEN the shebang line is intact (not template-rendered).
- **Empty scripts dir marker**: GIVEN scripts are written as files; WHEN scaffold creates the output; THEN `scripts/` directory exists with 3 `.py` files (no empty dir marker needed — files create the directory).
</details>

### Requirement: AGENT.md Enforcement Table

`AGENT.md.tmpl` MUST include a tool enforcement table at the end mapping each tool to its agent and phase. The table MUST have columns: Fase, Tool, Agente, Restriction. Tools MUST be referenced by relative path (`scripts/explorer.py`).

<details>
<summary>Scenarios</summary>

- **Table present**: GIVEN a scaffolded project; WHEN reading `AGENT.md`; THEN a `## Tools por Fase` section exists with a markdown table.
- **Tool-agent mapping**: GIVEN the enforcement table; WHEN inspecting rows; THEN `explorer.py` maps to `zyro-reader`, `test-runner.py` and `linter.py` map to `zyro-writer`.
- **Phase coverage**: GIVEN the table; WHEN counting unique phases; THEN all 4 phases (F1-F4) are represented.
</details>

### Requirement: Test Coverage

The scaffold package MUST have tests verifying: (1) all 3 script files exist in scaffold output, (2) script content is non-empty and starts with the shebang line, (3) `FilesCreated` includes scripts in the count, (4) the AGENT.md output contains the enforcement table.

<details>
<summary>Scenarios</summary>

- **Script file test**: GIVEN `scaffold.Run()` with a temp dir; WHEN checking the output; THEN `scripts/explorer.py`, `scripts/test-runner.py`, `scripts/linter.py` exist.
- **Script content test**: GIVEN the scaffold output; WHEN reading `scripts/explorer.py`; THEN the first line is `#!/usr/bin/env python3`.
- **FilesCreated test**: GIVEN `scaffold.Run()`; WHEN checking `result.FilesCreated`; THEN the value is `9` (6 templates + 3 scripts).
- **Enforcement table test**: GIVEN the scaffold output; WHEN reading `AGENT.md`; THEN the string `## Tools por Fase` is present.
</details>
