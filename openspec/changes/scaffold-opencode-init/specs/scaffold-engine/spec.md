# Scaffold Engine Specification

## Purpose

Generate a complete, portable project directory from a handoff contract payload — AGENT.md, opencode.json, Go stubs, docs, skills/, and README — so users go from contract to editable project in one command.

## Requirements

### Requirement: Scaffold Flag

The `init` subcommand MUST accept a `--scaffold` bool flag. When true, `init` MUST call the scaffold engine after parse+validate succeeds. The positional handoff path argument MUST still be required.

<details>
<summary>Scenarios</summary>

- **Flag present, scaffold succeeds**: GIVEN a valid handoff.yaml; WHEN `zyrocli init handoff.yaml --scaffold` runs; THEN scaffold output is created and exit code is 0.
- **No flag**: GIVEN `zyrocli init handoff.yaml` without `--scaffold`; THEN parse+validate runs but no files are created — existing behavior unchanged.
</details>

### Requirement: Project Directory Generation

The scaffold engine MUST create `{project.Name}/` as root. All files are generated from embedded `text/template` templates via `embed.FS`. The project name MUST be normalized (lowercased, spaces→hyphens, stripped of special chars) before use in paths or Go module names. Normalization MUST NOT be destructive — adjacent special chars become a single hyphen, leading/trailing hyphens are trimmed.

<details>
<summary>Scenarios</summary>

- **Full structure**: GIVEN a handoff with `project.name: "my-app"`; WHEN scaffold runs; THEN `my-app/` contains: `AGENT.md`, `opencode.json`, `handoff.yaml` (template), `skills/`, `docs/contexto_proyecto/`, `docs/recursos/`, `.gitignore`, `cmd/my-app/main.go`, `internal/`, `README.md`.
- **Name normalization**: GIVEN `project.name: "My Cool App_v1"`; WHEN scaffold runs; THEN the target directory is `my-cool-app-v1/` and Go module is `github.com/my-cool-app-v1`.
- **Existing directory**: GIVEN `my-app/` already exists; WHEN scaffold runs; THEN the engine MUST ask the user "Directory my-app/ already exists. Overwrite? [y/N]" and proceed accordingly — error on "N", overwrite on "y".
</details>

### Requirement: AGENT.md Generation

The generated `AGENT.md` MUST be ultra-condensado (~350 chars). It MUST contain: absolute rules (only ask, delegate, don't write), the project stack, the 4 macro-phase flow, a delegation table (inline vs delegate), and sub-agent references (zyro-reader, zyro-writer, graphify).

<details>
<summary>Scenarios</summary>

- **Content validation**: GIVEN a handoff with populated `project` and `stack`; WHEN `AGENT.md` is generated; THEN it contains the project stack, all 4 phases, and sub-agent names from opencode.json.
</details>

### Requirement: opencode.json Generation

The generated `opencode.json` MUST define `zyro-agent` as `mode:primary` with permission scope `read` + `task`. It MUST define `zyro-reader`, `zyro-writer`, and `graphify` as `mode:subagent` with atomic permissions as specified in the architecture. AGENT.md MUST be referenced via `{file:AGENT.md}`.

<details>
<summary>Scenarios</summary>

- **Atomic permissions**: GIVEN a generated `opencode.json`; WHEN parsed; THEN `zyro-agent` has no `bash`/`write`/`edit` permissions, `zyro-reader` has `read`+`bash`+`write`, `zyro-writer` has `read`+`write`+`edit`+`bash`, and `graphify` has `read` only.
- **Valid JSON**: GIVEN a generated `opencode.json`; WHEN `json.Unmarshal` runs; THEN it parses without error and `{file:AGENT.md}` reference resolves.
</details>

### Requirement: OpenCode Launch

The `init` subcommand MUST accept a `--opencode` bool flag that requires `--scaffold`. Before launching, it MUST check `exec.LookPath("opencode")`. If found, it MUST exec `opencode <targetDir>` with inherited stdin/stdout/stderr. If not found, it MUST print a friendly error without failing the scaffold.

<details>
<summary>Scenarios</summary>

- **OpenCode in PATH**: GIVEN `opencode` is in PATH; WHEN `--scaffold --opencode` runs; THEN `opencode <targetDir>` is exec'd with stdin/stdout/stderr passthrough.
- **OpenCode missing**: GIVEN `opencode` is NOT in PATH; WHEN `--scaffold --opencode` runs; THEN scaffold completes and a warning "opencode not found" is printed; exit code is 0.
</details>

### Requirement: Error Handling

Template parse errors MUST surface immediately at `scaffold.Run()` — no files written on parse failure. If the scaffold fails mid-write, the engine MUST call `os.RemoveAll(targetDir)` to clean up partial output. If the target directory already exists, the engine MUST fail with an error.

<details>
<summary>Scenarios</summary>

- **Bad template**: GIVEN a template with a syntax error; WHEN scaffold runs; THEN an error wrapping the parse failure is returned; no files exist on disk.
- **Partial write + cleanup**: GIVEN a file write fails after some files were written; WHEN scaffold cleans up; THEN the target directory is removed entirely.
- **Existing directory**: GIVEN `my-app/` already exists; WHEN scaffold runs with `--scaffold`; THEN an error stating the directory exists is returned.
</details>

### Requirement: Testing

The scaffold package MUST have table-driven tests covering output structure. The init command MUST have tests for flag combinations. Generated files MUST be verified by file existence checks.

<details>
<summary>Scenarios</summary>

- **Scaffold structure table test**: GIVEN a valid handoff payload; WHEN `scaffold.Run()` is called with a temp dir; THEN the test asserts all expected relative paths exist.
- **Flag combination test**: GIVEN `--scaffold --opencode`; WHEN `init` runs; THEN both flags are parsed successfully and scaffold+launch sequence fires.
</details>
