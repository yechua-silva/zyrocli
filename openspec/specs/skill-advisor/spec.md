# Skill Advisor Specification

## Purpose

Define the Skill Advisor engine that scores available agent skills against a query using deterministic weighted tag matching, loads skill manifests from YAML, and surfaces the top-N recommendations for agent orchestration.

<!-- delta:start -->
### Requirement: Pipeline Type Definitions

`DiscoveryQuery` MUST expose `Language`, `Framework`, `ProjectType`, `Keywords []string`. `ValidatedSkill` MUST wrap `SkillEntry` with `Score float64`, `Normalized float64`, `HardBlocked bool`, `RejectReason string`. `ValidationError` MUST contain `HardBlocked bool` and `Reason string`; when true, the caller MUST stop processing.

#### Scenario: ValidatedSkill from pipeline
- GIVEN a skill that passes all validation layers
- WHEN the pipeline produces a ValidatedSkill
- THEN HardBlocked is false and Score > 0

### Requirement: BuildDiscoveryQuery

`BuildDiscoveryQuery(payload handoff.Payload) DiscoveryQuery` MUST extract `Language` and `ProjectType` directly from the payload, infer `Framework` via `detectFramework(payload)`, and collect `Keywords` via `extractKeywords(payload)`.

#### Scenario: Full enrichment
- GIVEN a handoff with Language: "go" and ProjectType: "cli"
- WHEN `BuildDiscoveryQuery` is called
- THEN Language is "go", ProjectType is "cli", Framework is inferred (e.g. "cobra"), Keywords is non-empty

### Requirement: detectFramework

`detectFramework(payload handoff.Payload) string` MUST infer framework from payload language and project characteristics: go → cobra/gin/echo, typescript → next/astro/react, python → django/fastapi. Returns empty string if unknown.

#### Scenario: Go detection
- GIVEN payload Language "go" and a CLI-context project
- WHEN `detectFramework` is called
- THEN it returns "cobra"

#### Scenario: Unknown language
- GIVEN payload Language "rust"
- WHEN `detectFramework` is called
- THEN it returns ""

### Requirement: extractKeywords

`extractKeywords(payload handoff.Payload) []string` MUST extract lowercased, deduplicated tokens from `MVP.Scope`, `Features`, and `ValidatedIdea.Problem`.

#### Scenario: Extracts tokens
- GIVEN MVP.Scope contains "user authentication"
- WHEN `extractKeywords` is called
- THEN result includes "user" and "authentication" (lowercased, deduplicated)

### Requirement: ValidateAndScore with Hard Block

`ValidateAndScore(entries []SkillEntry, query DiscoveryQuery) ([]ValidatedSkill, error)` MUST apply 6 sequential layers:

| Layer | Check | Effect |
|-------|-------|--------|
| 1 | SocketAlerts > 0 | Hard block — `ValidationError{HardBlocked: true}` |
| 2 | Publisher not in whitelist | Penalty -50 |
| 3 | Language mismatch | Penalty -10 |
| 4 | Framework mismatch | Penalty -20 |
| 5 | ProjectType mismatch | Penalty -30 |
| 6 | `ScoreSkillWeighted()` | Final rank score (unchanged) |

Publisher whitelist: NVIDIA, Anthropic, Microsoft, Google, Meta, Amazon, OpenAI, HashiCorp, Docker, Netlify, Vercel, opencode-community. Layer 1 hard block MUST stop processing immediately. Layers 2–5 MUST apply cumulative penalties. Layer 6 MUST use existing `ScoreSkillWeighted`.

#### Scenario: Hard block on socket alerts
- GIVEN a skill with SocketAlerts: 3, publisher "NVIDIA"
- WHEN `ValidateAndScore` runs
- THEN it returns `ValidationError{HardBlocked: true, Reason: "socket_alerts=3"}`

#### Scenario: Unknown publisher penalty
- GIVEN a skill with SocketAlerts: 0, publisher "unknown-corp"
- WHEN `ValidateAndScore` runs
- THEN HardBlocked is false, total score is penalized by -50

#### Scenario: Full valid pipeline
- GIVEN entries from API and local registry, zero alerts, whitelisted publishers
- WHEN `ValidateAndScore` runs
- THEN all entries pass with cumulative layer penalties applied

### Requirement: Merge Local and API Results

`DiscoverAndRank` MUST call `Discover(query)` (API skills.sh) and `Registry.Recommend(query, n)` (local) concurrently. Results MUST merge with local entries winning on duplicate skill names. The merged list MUST be passed to `ValidateAndScore`.

#### Scenario: Duplicate resolution
- GIVEN API and local both return a skill named "go-testing"
- WHEN merging
- THEN the local entry is kept, the API entry is discarded for that name

### Requirement: DiscoverAndRank (Unified Entry Point)

`DiscoverAndRank(payload handoff.Payload, n int) ([]ValidatedSkill, error)` MUST orchestrate: (1) `BuildDiscoveryQuery`, (2) concurrent API + local discovery + merge, (3) `ValidateAndScore`, (4) sort by normalized score descending, return top-N. `RecommendFromHandoff` is **deprecated** — it MUST be a wrapper calling `DiscoverAndRank(payload, 0)`.

#### Scenario: Full pipeline returns ranked skills
- GIVEN a valid handoff.Payload with Go project
- WHEN `DiscoverAndRank` is called with n=5
- THEN returns up to 5 ValidatedSkills, sorted by Normalized desc, HardBlocked is false

#### Scenario: API failure graceful degradation
- GIVEN skills.sh API is unreachable
- WHEN `DiscoverAndRank` is called
- THEN local registry results are returned (no error)
<!-- delta:end -->

## Requirements

### Requirement: YAML Registry Loading

`Registry.Load(path string) error` MUST read a directory of `.yaml` / `.yml` skill manifests, parse each into a `SkillEntry`, and populate the `Skills` map. Duplicate skill names MUST overwrite (last wins). Malformed YAML files MUST be skipped with a warning logged via `slog.Warn`.

The `SkillEntry` struct MUST include the following fields:
- `name` (string, required) — unique skill identifier
- `description` (string) — human-readable summary
- `language` (string) — primary programming language
- `framework` (string) — primary framework
- `project_type` (string) — e.g., "cli", "web", "backend", "mobile"
- `triggers` ([]string) — keywords that trigger this skill
- `location` (string) — filesystem path to the SKILL.md
- `publisher` (string) — author or maintainer
- `verified` (bool) — whether the publisher is verified
- `socket_alerts` (int) — number of Socket.dev security alerts

#### Scenario: Load valid directory
- GIVEN a directory with 3 valid `.yaml` skill files
- WHEN `Registry.Load(dir)` is called
- THEN `Registry.Skills` contains 3 entries
- AND no error is returned

#### Scenario: Skip malformed YAML
- GIVEN a directory with 1 valid and 1 malformed `.yaml` file
- WHEN `Registry.Load(dir)` is called
- THEN the valid entry is loaded, the malformed entry is skipped with a warning
- AND the function does NOT return an error

#### Scenario: Duplicate names overwrite
- GIVEN a directory with 2 `.yaml` files sharing the same `name` field
- WHEN `Registry.Load(dir)` is called
- THEN only 1 entry exists in `Skills` with the last-parsed value

### Requirement: Deterministic Weighted Scoring

`ScoreSkillWeighted(skill SkillEntry, query SkillQuery) ScoreResult` MUST compute relevance using the following deterministic weights:

| Component | Weight | Condition |
|-----------|--------|-----------|
| Language match | 10 | `skill.Language == query.Language` (case-insensitive) |
| Framework match | 20 | `skill.Framework == query.Framework` (case-insensitive) |
| Project type match | 30 | `skill.ProjectType == query.ProjectType` (case-insensitive) |
| Verified publisher | 50 | `skill.Verified == true` |
| Socket zero alerts | 15 | Full if `socket_alerts == 0`, diminishing otherwise (`15 / (1 + alerts)`) |

Maximum possible score: 125. `ScoreResult.TotalScore` is the sum of all components.

`ScoreResult.Normalized() float64` returns `TotalScore / MaxPossibleScore` in `[0.0, 1.0]`.

#### Scenario: Full match
- GIVEN a skill with `Language: "go"`, `Framework: "gin"`, `ProjectType: "cli"`, `Verified: true`, `SocketAlerts: 0`
- AND a query with `Language: "Go"`, `Framework: "Gin"`, `ProjectType: "CLI"`
- WHEN `ScoreSkillWeighted(skill, query)` is called
- THEN `TotalScore` is 125
- AND `Normalized()` is 1.0

#### Scenario: No match
- GIVEN a skill with `Language: "python"` and a query with `Language: "go"`
- WHEN `ScoreSkillWeighted(skill, query)` is called
- THEN `TotalScore` is 0
- AND `Normalized()` is 0.0

### Requirement: Top-N Recommendations

`Registry.Recommend(query SkillQuery, n int) ([]ScoreResult, error)` MUST score all registered skills using `ScoreSkillWeighted`, sort descending by `TotalScore`, and return the top-N with scores > 0. If no skills score above 0, MUST return an empty slice with nil error. If `n <= 0`, MUST return all scored skills.

#### Scenario: Top-N returned
- GIVEN 3 registered skills with varying scores
- WHEN `Recommend(query, 2)` is called
- THEN 2 skills are returned, sorted by score descending

#### Scenario: No matches
- GIVEN a query with no overlapping language/framework/project type
- WHEN `Recommend(query, 3)` is called
- THEN an empty slice is returned with nil error

### Requirement: Skill Discovery from Skills.sh API

`DiscoverClient.Discover(query string) (*DiscoverResult, error)` MUST perform an HTTP GET to the skills.sh API, cache results with a configurable TTL (default 1h), and return `DiscoverResult` with `Skills []SkillEntry` and `Cached bool`.

On network failure, MUST fall back to the cached result if available (even if stale). If no cache and no network, MUST return an empty result with a wrapped error.

A package-level convenience function `Discover(query string) (*DiscoverResult, error)` MUST be available using a default client.

#### Scenario: API success
- GIVEN network is available
- WHEN `Discover("go testing")` is called
- THEN results are returned and `Cached` is `false`

#### Scenario: Network failure with cache
- GIVEN previous results are cached and network is down
- WHEN `Discover("go testing")` is called
- THEN cached results are returned and `Cached` is `true`

#### Scenario: Network failure without cache
- GIVEN no cache and network is down
- WHEN `Discover("go testing")` is called
- THEN an empty result is returned with an error wrapping the network failure

### TagsVector (Legacy)

`ScoreSkill(query, skill TagsVector) float64` MUST compute relevance as `hits / len(query)` where `hits` = intersection count. If the query is empty, score MUST be 0. If a skill has no tags, score MUST be 0. The result MUST be in `[0.0, 1.0]`. This is a legacy function kept for backward compatibility.
