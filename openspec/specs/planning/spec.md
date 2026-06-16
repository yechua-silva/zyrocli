# Planning Macro Specification

## Purpose

Define Macro 2 of the SDD pipeline: planning. The agent MUST decompose user stories into atomic features via the Decomposer, then produce a dependency-respecting execution schedule via the Scheduler using Kahn's algorithm for topological sort.

## Requirements

### Requirement: Feature Decomposition

The Decomposer MUST accept a user story string and produce a list of atomic `Feature` values. Each feature MUST have:
- A unique ID (F1, F2, ...)
- A kebab-case name derived from the story text
- A description
- Acceptance criteria (preserved from input)
- A complexity estimate (small/medium/large)
- Optional dependency references to other feature IDs

The Decomposer MUST handle these story formats:
- Single sentence
- Bullet lists (lines starting with `-`)
- Numbered lists (lines starting with `N.` or `N)`)
- "and"-conjoined compound stories

#### Scenario: Empty story rejected
- GIVEN an empty user story
- WHEN Decompose is called
- THEN an error is returned: "cannot plan without a user story"

#### Scenario: Bullet list decomposition
- GIVEN a story with 3 bullet items
- WHEN Decompose is called
- THEN 3 features are produced

#### Scenario: Single sentence
- GIVEN a single-sentence user story
- WHEN Decompose is called
- THEN 1 feature is produced

### Requirement: Schedule via Topological Sort

The Scheduler MUST accept a list of features and produce an ordered `Schedule` using Kahn's algorithm for topological sort.

The schedule MUST group features into phases:
- Phase 1: features with no dependencies
- Phase 2: features whose dependencies are all in phase 1
- Phase N: features whose dependencies are all in phases 1..N-1

Each phase entry MUST have a within-phase priority (1-based, ascending).

#### Scenario: Linear dependency chain
- GIVEN features: F1 (no deps), F2 (depends on F1), F3 (depends on F2)
- WHEN Schedule is called
- THEN F1 is phase 1, F2 is phase 2, F3 is phase 3

#### Scenario: Parallel independent features
- GIVEN features F1, F2, F3 with no dependencies
- WHEN Schedule is called
- THEN all three are in phase 1

### Requirement: Circular Dependency Detection

The scheduler MUST detect and report circular dependencies during scheduling. Unknown dependency references MUST also be reported.

#### Scenario: Circular dependency
- GIVEN F1 → F2 and F2 → F1
- WHEN ValidateNoCircularDeps is called
- THEN a circular dependency error is returned

#### Scenario: Unknown dependency
- GIVEN feature F1 depending on F99
- WHEN Schedule is called
- THEN an error indicates F99 is unknown

### Requirement: Schedule Markdown Rendering

The schedule MUST render as structured markdown via `Schedule.Markdown()`, showing phases as tables with priority, feature name, complexity, and blocked-by information.

#### Scenario: Schedule renders as markdown
- GIVEN a populated Schedule with 3 phases
- WHEN `Schedule.Markdown()` is called
- THEN structured markdown is returned with phase tables
