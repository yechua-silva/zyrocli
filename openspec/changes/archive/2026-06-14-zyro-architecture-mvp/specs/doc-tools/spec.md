# Delta Spec: doc-tools

## Parent spec

`openspec/specs/doc-tools/spec.md` (created by sdd-init).

## Change context

- **Change**: zyro-architecture-mvp
- **PR**: 5 — delivery-doc-tools

## New requirements

### R-DOC-004: Doc Index Generation

The system MUST generate a doc index at `.zyro/doc-index.yaml` from
`.zyro/conventions.yaml` known topic keys and active change directories.

**Fields** per entry: `topic_key`, `type`, `observation_id`, `last_modified`, `change_name`.

### R-DOC-005: Index Search Protocol

The system SHOULD implement a 4-step search protocol over the doc index:

1. **Fast path**: exact `topic_key` → return entry directly
2. **Slow path**: `query` text search across topic_key, type, and change_name
3. **Fallback**: filter by `type` or `change_name`
4. **Last resort**: return no results (caller asks the human)

### R-DOC-006: Sync Cycle

The system MUST provide a `Sync()` function that orchestrates:
1. `GenerateIndex()` — build doc index
2. `SaveIndex()` — write `.zyro/doc-index.yaml`
3. `Export()` — render ARCHITECTURE.md + CHANGELOG.md from Go templates
4. `UpdateGraph()` — compare with previous state; persist if significant

### R-DOC-007: CLI Sync Command

The system MUST expose a `zyrocli doc sync` CLI command that runs the
full sync cycle and reports the result.

### R-DOC-008: Export Templates

The system MUST embed two Go templates:
- `ARCHITECTURE.md.tmpl` — renders all index entries grouped by type
- `CHANGELOG.md.tmpl` — renders change-scoped entries with timestamps

### R-DOC-009: Graph Diff

The system SHOULD track doc index changes across sync runs. A change is
"significant" when the entry count differs by ≥5 from the previous state.

## Delta scenarios

### Scenario D-DOC-004-1: Generate index from conventions

**Given** a project with `.zyro/conventions.yaml` and no active changes
**When** `GenerateIndex()` is called
**Then** the returned index contains only project-scoped entries
**And** each entry has the required fields

### Scenario D-DOC-004-2: Generate index with active changes

**Given** a project with active change directories under `openspec/changes/`
**When** `GenerateIndex()` is called
**Then** the returned index includes change-scoped entries for each active change

### Scenario D-DOC-005-1: Search by exact topic key

**Given** a populated doc index
**When** `SearchIndex()` is called with an exact `topic_key`
**Then** the matching entry is returned immediately
**And** no fallback path is reached

### Scenario D-DOC-005-2: Search by query text

**Given** a populated doc index with no exact match
**When** `SearchIndex()` is called with a `query` string
**Then** entries matching the query across topic_key, type, or change_name are returned

### Scenario D-DOC-005-3: Fallback by type

**Given** a populated doc index with no exact or query match
**When** `SearchIndex()` is called with a `type` filter
**Then** all entries whose type prefix matches are returned

### Scenario D-DOC-006-1: Full sync cycle

**Given** a project with valid `.zyro/conventions.yaml`
**When** `Sync()` is called
**Then** `.zyro/doc-index.yaml` is created
**And** `ARCHITECTURE.md` is generated at the project root
**And** `CHANGELOG.md` is generated at the project root
**And** `.zyro/graph-state.yaml` is created

### Scenario D-DOC-007-1: CLI sync

**Given** a project root with `.zyro/conventions.yaml`
**When** `zyrocli doc sync` is run from the project root
**Then** the sync cycle completes successfully
**And** output confirms the generated files

### Scenario D-DOC-009-1: Significant graph change

**Given** a previous graph state with 2 entries
**When** the current index has 10 entries (diff ≥5)
**Then` `UpdateGraph()` persists the new state with 10 entries

### Scenario D-DOC-009-2: Insignificant graph change

**Given** a previous graph state with 2 entries
**When** the current index has 3 entries (diff <5)
**Then** `UpdateGraph()` does not update the persisted state
