# Doc Tools Specification

## Purpose

Define the documentation toolchain: conventions.yaml management, index generation from Engram topic keys, structured search protocol, sync cycle with export templates, and graph diff tracking.

## Requirements

### Requirement: Conventions YAML

`Conventions` MUST be loadable from `conventions.yaml` with fields: `type` (code/doc/review), `pattern` (glob), `rule` (description), `severity` (must/should/may). The registry MUST support listing all conventions and filtering by type.

#### Scenario: Load conventions
- GIVEN a valid `conventions.yaml` with 5 conventions across 3 types
- WHEN `LoadConventions("conventions.yaml")` is called
- THEN all 5 conventions are loaded and filterable by type

### Requirement: Doc Index Generation

The system MUST generate a doc index at `.zyro/doc-index.yaml` from `.zyro/conventions.yaml` known topic keys and active change directories. Fields per entry: `topic_key`, `type`, `observation_id`, `last_modified`, `change_name`.

#### Scenario: Generate index from conventions
- GIVEN a project with `.zyro/conventions.yaml` and no active changes
- WHEN `GenerateIndex()` is called
- THEN the returned index contains only project-scoped entries

#### Scenario: Generate index with active changes
- GIVEN a project with active change directories under `openspec/changes/`
- WHEN `GenerateIndex()` is called
- THEN the returned index includes change-scoped entries for each active change

### Requirement: Index Search Protocol

The system SHOULD implement a 4-step search protocol over the doc index:

1. **Fast path**: exact `topic_key` → return entry directly
2. **Slow path**: `query` text search across topic_key, type, and change_name
3. **Fallback**: filter by `type` or `change_name`
4. **Last resort**: return no results (caller asks the human)

#### Scenario: Search by exact topic key
- GIVEN a populated doc index
- WHEN `SearchIndex()` is called with an exact `topic_key`
- THEN the matching entry is returned immediately

#### Scenario: Search by query text
- GIVEN a populated doc index with no exact match
- WHEN `SearchIndex()` is called with a `query` string
- THEN entries matching the query across topic_key, type, or change_name are returned

#### Scenario: Fallback by type
- GIVEN a populated doc index with no exact or query match
- WHEN `SearchIndex()` is called with a `type` filter
- THEN all entries whose type prefix matches are returned

### Requirement: Sync Cycle

The system MUST provide a `Sync()` function that orchestrates:
1. `GenerateIndex()` — build doc index
2. `SaveIndex()` — write `.zyro/doc-index.yaml`
3. `Export()` — render ARCHITECTURE.md + CHANGELOG.md from Go templates
4. `UpdateGraph()` — compare with previous state; persist if significant

#### Scenario: Full sync cycle
- GIVEN a project with valid `.zyro/conventions.yaml`
- WHEN `Sync()` is called
- THEN `.zyro/doc-index.yaml` is created
- AND `ARCHITECTURE.md` is generated at the project root
- AND `CHANGELOG.md` is generated at the project root
- AND graph state is tracked

### Requirement: CLI Sync Command

The system MUST expose a `zyrocli doc sync` CLI command that runs the full sync cycle and reports the result.

#### Scenario: CLI sync
- GIVEN a project root with `.zyro/conventions.yaml`
- WHEN `zyrocli doc sync` is run from the project root
- THEN the sync cycle completes successfully
- AND output confirms the generated files

### Requirement: Export Templates

The system MUST embed two Go templates:
- `ARCHITECTURE.md.tmpl` — renders all index entries grouped by type
- `CHANGELOG.md.tmpl` — renders change-scoped entries with timestamps

#### Scenario: Templates render correctly
- GIVEN valid template data
- WHEN templates are rendered with populated index data
- THEN ARCHITECTURE.md and CHANGELOG.md are produced without errors

### Requirement: Graph Diff

The system SHOULD track doc index changes across sync runs. A change is "significant" when the entry count differs by ≥5 from the previous state.

#### Scenario: Significant graph change
- GIVEN a previous graph state with 2 entries
- WHEN the current index has 10 entries (diff ≥5)
- THEN `UpdateGraph()` persists the new state with 10 entries

#### Scenario: Insignificant graph change
- GIVEN a previous graph state with 2 entries
- WHEN the current index has 3 entries (diff <5)
- THEN `UpdateGraph()` does not update the persisted state
