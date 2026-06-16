# Delta Spec: Investigation (Macro 1)

## Change Context

- **Change**: zyro-architecture-mvp
- **Domain**: investigation
- **Type**: delta (new package)
- **Base spec**: `openspec/specs/investigation/spec.md`

## What Changed

Introduced `internal/investigation/` package implementing Macro 1 of the SDD
pipeline. The package replaces ad-hoc investigation with a structured engine
that concurrently queries three source types.

### New Requirements

#### R-INV-004: ResearchEngine

The investigation **MUST** provide a `ResearchEngine` that runs the following
sources concurrently via goroutines:
1. Context MCP documentation queries (via injected `DocQueryFn`)
2. Git repository analysis (via native `git` CLI commands)
3. Web URL fetching (via injected `WebFetchFn`)

Each source **MUST** be independent — failures in one **MUST NOT** block others.

#### R-INV-005: Investigation Report

The `ResearchEngine` **MUST** return an `*investigation.Report` containing:
- `DocSources` — per-library query results with optional error
- `GitAnalysis` — code structure, language breakdown, recent commits, patterns
- `WebSources` — per-URL fetched content with optional error
- `GeneratedAt` — timestamp of generation

The report **MUST** render as structured markdown via `Report.Markdown()`.

#### R-INV-006: Advisor

The investigation **MUST** provide an `Advisor` that consolidates research into
recommendations:
- Stack/libraries with pros/cons and confidence scores
- Architectural patterns with rationale and effort estimates
- Skills to install by layer (excluding already-installed skills)
- MVP approach based on handoff boundaries

### Modified Requirements

#### R-INV-003: Investigation Report — Output Path

The report path `docs/contexto_proyecto/investigation.md` is **deprecated**.
Instead, the report **SHOULD** be persisted to Engram under topic key
`zyro/{project}/investigation` and **MAY** be exported on-demand via doc tools.

### Scenarios (Delta)

#### Scenario: Concurrent source execution
- GIVEN a ResearchEngine with 3 doc libraries and 2 URLs
- WHEN `Run(ctx)` is called
- THEN all sources are queried concurrently
- AND the report contains 3 doc sources and 2 web sources

#### Scenario: Partial source failure
- GIVEN a ResearchEngine with a failing doc query
- WHEN `Run(ctx)` is called
- THEN the failing source has a non-empty Error field
- AND other sources still succeed

#### Scenario: No MVP boundaries
- GIVEN an Advisor with empty MVP boundaries
- WHEN `Analyze()` is called
- THEN the report warns about missing MVP scope

#### Scenario: Stack recommendations by language
- GIVEN a Go project
- WHEN `Analyze()` is called
- THEN Cobra is in the stack recommendations

#### Scenario: Skills exclude installed
- GIVEN an Advisor with `AvailableSkills: ["go-testing"]`
- WHEN `Analyze()` is called
- THEN "go-testing" is NOT in the recommended skills
