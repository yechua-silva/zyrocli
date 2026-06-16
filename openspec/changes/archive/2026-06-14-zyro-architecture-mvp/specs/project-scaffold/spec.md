# Delta for Project Scaffold

## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: File Count Updated

`scaffold.Run()` MUST now create 10 files (was 9) due to the addition of `.zyro/conventions.yaml`. The 10 files are: AGENT.md, opencode.json, handoff.yaml, .gitignore, README.md, cmd/{name}/main.go, .zyro/conventions.yaml, scripts/explorer.py, scripts/test-runner.py, scripts/linter.py.
(Previously: 9 files — conventions.yaml was not included)

#### Scenario: All 10 files created
- GIVEN a scaffolded project
- WHEN `result.FilesCreated` is checked
- THEN it equals 10

### Requirement: Build Verification Extended

The scaffold MUST pass `go test ./...` with all packages including the new conventions.yaml verification. The template for conventions.yaml MUST be valid YAML when rendered.
(Previously: only 9 files were checked)

#### Scenario: Conventions template renders valid YAML
- GIVEN the conventions.yaml.tmpl template
- WHEN rendered with a Config
- THEN the output is valid YAML
