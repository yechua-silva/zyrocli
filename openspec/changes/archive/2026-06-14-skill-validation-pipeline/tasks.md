# Tasks: Skill Validation Pipeline

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 350–450 |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Core pipeline: types, query, verify, score, discover.py, Go→Python bridge | PR 1 (~250 loc) | base: main; new files + discover.py |
| 2 | Entry points + callers + scaffold + tests | PR 2 (~180 loc) | base: PR 1; modifies existing callers |

---

## Phase 1: Foundation — Type Definitions

- [x] 1.1 `TASK-001`: **Pipeline Type Definitions** — Files: `internal/skilladvisor/types.go` — Add `DiscoveryQuery` (Language/Framework/ProjectType/Keywords), `ValidationError` (HardBlocked/Reason), `ValidatedSkill` (Skill/Score/Rejected/RejectReason/ValidationError). Keep existing types (`Skill`, `ScoreResult`, `SkillQuery`). **Deps**: None. **Accept**: Compiles; existing tests pass.

## Phase 2: Core Logic — Query, Verify, Score

- [x] 2.1 `TASK-002`: **query.go — BuildDiscoveryQuery + Helpers** — Files: `internal/skilladvisor/query.go` (new) — `BuildDiscoveryQuery(payload)` extracts Language/ProjectType, calls `detectFramework` (go→cobra/gin/echo, ts→next/astro/react/vue, python→django/fastapi/flask) and `extractKeywords` (tokenize Scope+Features+Problem, stopwords filter, lowercase, dedup, max 10). **Deps**: TASK-001. **Accept**: Tests for detectFramework (go+cobra→"cobra", rust→"") and extractKeywords (dedup, max 10) pass.

- [x] 2.2 `TASK-003`: **verify.go — Publisher Whitelist** — Files: `internal/skilladvisor/verify.go` (new) — `KnownPublishers` var (12 entries: anthropic, nvidia, microsoft, google, meta, amazon, openai, hashicorp, docker, netlify, vercel, opencode-community). `VerifyPublisher(publisher)` does case-insensitive match. **Deps**: TASK-001. **Accept**: Tests: "NVIDIA"→true, "x"→false.

- [x] 2.3 `TASK-004`: **score.go — ValidateAndScore 6 Layers** — Files: `internal/skilladvisor/score.go` — `ValidateAndScore(entries, query)` — Layer 1: SocketAlerts>0 → hard block (Rejected+ValidationError). Layer 2: !VerifyPublisher → VerifiedBonus=0, TotalScore-=50. Layers 3-5: mismatches via ScoreSkillWeighted (natural 0). Layer 6: base score. Returns []ValidatedSkill. **Deps**: TASK-001, TASK-003. **Accept**: Tests: hard block, publisher penalty, full valid, mixed batch.

## Phase 3: Python Discovery + Go Bridge

- [x] 3.1 `TASK-005`: **scripts/discover.py — Python Discovery Script** — Files: `internal/scaffold/templates/go-project/scripts/discover.py` (new) — Python script queries skills.sh API and returns JSON skills on stdout. Args: `--lang`, `--framework`, `--project-type`, `--keywords`. Uses urllib (stdlib, no external deps). Timeout 30s. Fallback: returns empty list + error on stderr. URL configurable via `SKILLS_API_URL` env var. `SKILLS_API = os.getenv("SKILLS_API_URL", "https://skills.sh/api/search")`. **Deps**: None. **Accept**: `python3 discover.py --lang go --framework cobra --project-type cli --keywords "cli,testing"` returns valid JSON. No Python installed → graceful error.

- [x] 3.2 `TASK-006`: **pipeline.go + discover.go — Go→Python Bridge** — Files: `internal/skilladvisor/pipeline.go` (new), `internal/skilladvisor/discover.go` (mod) — `execPythonDiscover(query DiscoveryQuery) ([]SkillEntry, error)`: exec.Command("python3", "scripts/discover.py", ...), captures stdout JSON, parses to []SkillEntry. If python3 not in PATH or script fails, returns error for graceful degradation. `DiscoverAndRank(payload, n)`: calls execPythonDiscover + Registry.LoadDefaults concurrently → MergeAndRank. `discover.go`: deprecate `DiscoverClient`, keep `DiscoverCache` for pipeline cache. Add comment "DEPRECATED: use scripts/discover.py via DiscoverAndRank". **Deps**: TASK-001, TASK-002, TASK-003, TASK-004, TASK-005. **Accept**: Compiles. Python available → skills from API + local. Python unavailable → local only (warning log). Both fail → error.

## Phase 4: Embed + Scaffold + Callers

- [x] 4.1 `TASK-007`: **Embed discover.py + scaffold** — Files: `internal/scaffold/scripts.go`, `internal/scaffold/scaffold.go` — `scripts.go`: verify that `templates/go-project/scripts/*` glob already captures `discover.py` (it does — no embed directive change needed). Update `ReadScript` doc comment to list "discover.py". `scaffold.go`: add `{"templates/go-project/scripts/discover.py", "scripts/discover.py"}` to `scriptEntries`. Change `RecommendedSkills []skilladvisor.ScoreResult` → `[]skilladvisor.ValidatedSkill` (line 24). **Deps**: TASK-005. **Accept**: scaffold produces `scripts/discover.py`. Compiles.

- [x] 4.2 `TASK-008`: **scheduler — Result.Skills + F1AgentFunc** — Files: `internal/scheduler/phase.go`, `internal/scheduler/macro_runner.go` — Add `Skills []skilladvisor.ValidatedSkill` to `Result` struct. F1AgentFunc: replace `Discover(name)` with `DiscoverAndRank(payload, 0)`, set `Result.Skills = validated`. F1 now calls Python internally via pipeline. **Deps**: TASK-006, TASK-007. **Accept**: Compiles. F1AgentFunc populates Result.Skills.

- [x] 4.3 `TASK-009`: **run.go — Migrate to DiscoverAndRank** — Files: `cmd/zyrocli/run.go` — Replace `RecommendFromHandoff(lang, type, 8)` → `DiscoverAndRank(payload, 8)`. Change type `[]ScoreResult` → `[]ValidatedSkill`. Update display: Rejected → `✗ name — REJECTED: reason`, else → `• name — desc (score: N.NN)`. **Deps**: TASK-006. **Accept**: Compiles. Output shows validated/rejected per skill.

## Phase 5: Testing + Verification

- [x] 5.1 `TASK-010`: **Integration Tests** — Files: `internal/skilladvisor/skilladvisor_test.go`, `internal/scheduler/scheduler_test.go` — Updated tests: `TestExecPythonDiscover_Success` (mock exec.Command or httptest+httpretty for Python), `TestExecPythonDiscover_PythonMissing` (exec.Command fails → graceful degradation), `TestDiscoverAndRank_PythonFallback` (Python unavailable → local only), `TestDiscoverAndRank_EndToEnd` (Python available + mock API → full pipeline). Maintain existing tests: ValidateAndScore, MergeAndRank, BuildDiscoveryQuery, detectFramework, extractKeywords, VerifyPublisher. **Deps**: TASK-006, TASK-007, TASK-008, TASK-009. **Accept**: `go test ./...` passes.

- [x] 5.2 `TASK-011`: **Final Verification** — Files: N/A — Run `go test ./...`, `go build ./...`, `go vet ./...`. Fix failures. Verify templates render correctly with ValidatedSkill type. Check `scaffold.Config` callers (run.go, init.go) compile with new type. **Deps**: TASK-010. **Accept**: Zero failures; zero compilation errors; go vet clean.

---

## Technical Debt

| Item | Severity | Description | Mitigation |
|------|----------|-------------|------------|
| TD-001 | Medium | `discover.go` HTTP client (`DiscoverClient`, `fetchFromAPI`) deprecated but not removed. Legacy code no longer used. | Add DEPRECATED comment. Remove in next major iteration. |
| TD-002 | Low | Python 3 runtime dependency. If not installed, degrades gracefully to local registry only (fewer skills). | Documented in design. Graceful fallback implemented. |
| TD-003 | Low | Subprocess overhead (~50-100ms per exec) on every discovery call. | Cache in Go (`DiscoverCache`) mitigates repeated calls. |
| TD-004 | Low | Dual schema: Python dict ↔ Go SkillEntry. If skills.sh schema changes, both sides require update. | Contract documented in discover.py comments. |
| TD-005 | Medium | `scaffold.Config.RecommendedSkills` changes from `[]ScoreResult` to `[]ValidatedSkill`. Templates referencing `.TotalScore` directly break. | Verify templates in `internal/scaffold/templates/` referencing `.TotalScore` and update to `.Score.TotalScore`. |
| TD-006 | Low | Embed glob `*` in scripts.go captures ALL `.py` — if someone adds an unwanted script, it's included automatically. | Documented. If fine-grained control needed, switch to explicit list. |
| TD-007 | Low | No Python tests (no pytest configured in Go project). discover.py only tested indirectly via Go exec mock. | Open issue to consider Python tests if script grows complex. |
