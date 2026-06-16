# Delivery Macro Specification

## Purpose

Define Macro 4 of the SDD pipeline: delivery. The agent MUST create commits as reviewable work units, generate PRs, and support chained PR splitting for large changes.

## Requirements

### Requirement: Work Unit Commits

The delivery macro MUST follow the `work-unit-commits` skill pattern: each commit MUST represent a single reviewable work unit with tests and docs bundled. Commit messages MUST follow conventional commits format.

#### Scenario: Commit structure
- GIVEN 3 completed task files
- WHEN delivery creates commits
- THEN 3 commits are created, each with conventional commit message

#### Scenario: Tests bundled with code
- GIVEN a Go source file change
- WHEN delivery creates the commit
- THEN the corresponding `_test.go` file is in the same commit

### Requirement: PR Creation

`CreatePR(title, body string) (string, error)` MUST create a GitHub PR via `gh` CLI. The body MUST include a summary, checklist, and reference to the SDD change name. On success, the PR URL MUST be returned.

#### Scenario: PR created successfully
- GIVEN a branch with committed changes
- WHEN `CreatePR("feat: add auth", "body")` is called
- THEN a PR URL is returned and the PR exists on GitHub

#### Scenario: gh not installed
- GIVEN `gh` CLI is not in PATH
- WHEN `CreatePR(...)` is called
- THEN an error wrapping "gh not found" is returned

### Requirement: Chained PR Splitting

If the total diff exceeds 400 lines, the delivery macro MUST invoke the `chained-pr` skill to split into stacked PRs. Each chain must be buildable and reviewable independently.

#### Scenario: Large diff splits
- GIVEN a diff of 600 lines across 3 work units
- WHEN delivery invokes chained PR splitting
- THEN 3 stacked PRs are created, each < 400 lines

#### Scenario: Small diff no split
- GIVEN a diff of 150 lines
- WHEN delivery evaluates splitting
- THEN a single PR is created (no splitting)
