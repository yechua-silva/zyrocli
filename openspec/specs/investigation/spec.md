# Investigation Macro Specification

## Purpose

Define Macro 1 of the SDD pipeline: agent investigation. The agent MUST combine Context MCP documentation queries, GitMCP repository analysis, and web fetching to research the codebase before planning. The investigation is driven by a `ResearchEngine` that orchestrates sources concurrently and an `Advisor` that consolidates findings into recommendations.

## Requirements

### Requirement: Context Documentation Lookup

The investigation MUST query the Context MCP bridge with library IDs extracted from `handoff.yaml` dependencies. Results MUST be cached for the session duration.

#### Scenario: Lookup project dependencies
- GIVEN a handoff.yaml listing Go + cobra as dependencies
- WHEN the investigation macro runs
- THEN Context MCP is queried for `/golang/go` and `/spf13/cobra`

#### Scenario: Cache hit
- GIVEN a previously queried library ID
- WHEN the same library is queried again
- THEN the cached result is returned (no external query)

### Requirement: GitMCP Repository Analysis

The investigation MUST query the Git repository via GitMCP for structure, commit history, and file content patterns. Results MUST be summarized as markdown.

#### Scenario: Analyze repo
- GIVEN a git repository in the current directory
- WHEN GitMCP analysis runs
- THEN a markdown summary of structure, language breakdown, and recent commits is produced

#### Scenario: No git repo
- GIVEN no `.git` directory exists
- WHEN GitMCP analysis runs
- THEN the investigation skips git analysis with a warning

### Requirement: Web Fetch for External Context

The investigation MAY fetch external URLs listed in the handoff's `source.url` field. Fetched content MUST be appended to the investigation report.

#### Scenario: Fetch external URL
- GIVEN `source.url: "https://example.com/api-docs"` in handoff.yaml
- WHEN web fetch runs
- THEN the fetched markdown content is appended to the report

#### Scenario: Unreachable URL
- GIVEN an unreachable external URL
- WHEN web fetch runs
- THEN the URL is skipped with a warning, investigation continues

### Requirement: ResearchEngine

The investigation MUST provide a `ResearchEngine` that runs the following sources concurrently via goroutines:
1. Context MCP documentation queries (via injected `DocQueryFn`)
2. Git repository analysis (via native `git` CLI commands)
3. Web URL fetching (via injected `WebFetchFn`)

Each source MUST be independent — failures in one MUST NOT block others.

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

### Requirement: Investigation Report

The `ResearchEngine` MUST return an `*investigation.Report` containing:
- `DocSources` — per-library query results with optional error
- `GitAnalysis` — code structure, language breakdown, recent commits, patterns
- `WebSources` — per-URL fetched content with optional error
- `GeneratedAt` — timestamp of generation

The report MUST render as structured markdown via `Report.Markdown()`.

The report path `docs/contexto_proyecto/investigation.md` is **deprecated**. Instead, the report SHOULD be persisted to Engram under topic key `zyro/{project}/investigation` and MAY be exported on-demand via doc tools.

#### Scenario: Full report generated
- GIVEN all sources are available
- WHEN the investigation completes
- THEN the Report has populated DocSources, GitAnalysis, and WebSources sections

#### Scenario: Report renders as markdown
- GIVEN a populated Report
- WHEN `Report.Markdown()` is called
- THEN structured markdown is returned with all sections

### Requirement: Advisor

The investigation MUST provide an `Advisor` that consolidates research into recommendations:
- Stack/libraries with pros/cons and confidence scores
- Architectural patterns with rationale and effort estimates
- Skills to install by layer (excluding already-installed skills)
- MVP approach based on handoff boundaries

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
