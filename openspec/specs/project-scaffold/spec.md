# Project Scaffold Specification

## Purpose

Define the skeleton of `github.com/secko/zyrocli` — a compilable Go module with CLI entry point, internal stubs, and config examples. This spec covers what MUST exist for the project to build; it does NOT prescribe business logic.

## Requirements

### Requirement: Module Initialization

The module MUST be `github.com/secko/zyrocli` with `go 1.22+` in `go.mod` (toolchain version detected at init time). Dependencies MUST include `github.com/spf13/cobra` and `gopkg.in/yaml.v3`. A `go.sum` with verified checksums MUST be present.

#### Scenario: Dependencies resolved
- GIVEN Go 1.22+ and a fresh directory
- WHEN `go mod init github.com/secko/zyrocli && go mod tidy`
- THEN `go.mod` declares cobra + yaml.v3 and `go.sum` has checksums

#### Scenario: Missing dependency fails build
- GIVEN `go.mod` missing a required dep
- WHEN `go build ./...`
- THEN build fails with "missing dependency"

### Requirement: File Count

`scaffold.Run()` MUST now create 10 files (was 9) due to the addition of `.zyro/conventions.yaml`. The 10 files are: AGENT.md, opencode.json, handoff.yaml, .gitignore, README.md, cmd/{name}/main.go, .zyro/conventions.yaml, scripts/explorer.py, scripts/test-runner.py, scripts/linter.py.

#### Scenario: All 10 files created
- GIVEN a scaffolded project
- WHEN `result.FilesCreated` is checked
- THEN it equals 10

### Requirement: Cobra CLI Entry Point

`cmd/zyrocli/main.go` MUST initialize a Cobra root command with `Use: "zyrocli"`. It MUST accept `--help` and SHOULD expose a hook for subcommand registration.

#### Scenario: Help displays usage
- GIVEN the binary is built
- WHEN running `zyrocli --help`
- THEN output shows root command usage

#### Scenario: Subcommand registration hook
- GIVEN `cmd/zyrocli/main.go` exists
- WHEN a new file adds a subcommand via `rootCmd.AddCommand()`
- THEN the subcommand appears in `--help` output

### Requirement: Internal Package Stubs

Seven `internal/` packages MUST provide compilable stubs, each exporting at least one type or function: `scheduler`, `handoff`, `skilladvisor` (3 files), `spec` (2 files), `context`, `apply`, `test` (2 files). All MUST compile with `go build ./...`.

#### Scenario: All packages compile
- GIVEN stub directories with `.go` files exist
- WHEN `go build ./internal/...`
- THEN no compilation errors

#### Scenario: Missing package breaks build
- GIVEN one stub directory is removed
- WHEN `go build ./...`
- THEN build fails — "no Go files" in that directory

### Requirement: .zyro/conventions.yaml Generation

`scaffold.Run()` MUST create a `.zyro/conventions.yaml` file in the scaffolded project with topic keys, entry format, search protocol, graphify configuration, and code conventions. This file serves as the project's single source of truth for Engram topic keys and SDD conventions.

#### Scenario: Conventions file present
- GIVEN a scaffolded project
- WHEN inspecting the `.zyro/` directory
- THEN `.zyro/conventions.yaml` exists with valid YAML

#### Scenario: Conventions file has topic keys
- GIVEN a scaffolded project with `.zyro/conventions.yaml`
- WHEN parsed
- THEN it contains `topic_keys.project`, `topic_keys.change`, and `topic_keys.graph` sections

### Requirement: .zyro/ Directory Created

`scaffold.Run()` MUST create the `.zyro/` directory in the project root for storing Zyro metadata (state.json, conventions.yaml, doc-index.yaml).

#### Scenario: .zyro directory exists
- GIVEN a scaffolded project
- THEN `.zyro/` directory exists in the project root

### Requirement: AGENT.md Reflects Macro Fases 1-4

The AGENT.md.tmpl template MUST document the 4 macro SDD phases (F1-F4) and all 8 `zyro-sdd-*` wrapper skills (explore, propose, spec, design, tasks, implement, verify, archive) in the skills table.

#### Scenario: AGENT.md lists all 8 SDD skills
- GIVEN a scaffolded project
- WHEN reading AGENT.md
- THEN it lists all 8 zyro-sdd-* skills (explore, propose, spec, design, tasks, implement, verify, archive)

#### Scenario: AGENT.md describes macro fases
- GIVEN a scaffolded project
- WHEN reading AGENT.md
- THEN it describes 4 macro fases (F1-F4) with their corresponding SDD skills

### Requirement: Build Verification Extended

The project MUST pass `go vet ./...` and `go build ./...` without errors or warnings. The scaffold MUST pass `go test ./...` with all packages including the new conventions.yaml verification. The template for conventions.yaml MUST be valid YAML when rendered.

#### Scenario: vet passes cleanly
- GIVEN all Go source files are present
- WHEN `go vet ./...`
- THEN zero diagnostics emitted

#### Scenario: Conventions template renders valid YAML
- GIVEN the conventions.yaml.tmpl template
- WHEN rendered with a Config
- THEN the output is valid YAML

### Requirement: Example Configs and .gitignore

`handoff.yaml` and `zyro-skill-overrides.yaml` MUST contain valid YAML. A `.gitignore` MUST exclude Go binaries (`zyrocli`, `/zyrocli`), vendor, IDE files, and OS artifacts.

#### Scenario: Config files are valid YAML
- GIVEN both `.yaml` files exist
- WHEN parsed with `yaml.Unmarshal`
- THEN no parse errors occur

#### Scenario: Binary excluded from git
- GIVEN `.gitignore` with `/zyrocli` rule
- WHEN `git check-ignore ./zyrocli`
- THEN exit code is 0 (ignored)
