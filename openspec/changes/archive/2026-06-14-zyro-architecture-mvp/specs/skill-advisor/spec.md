# Skill Advisor — Delta Spec

**Change**: zyro-architecture-mvp  
**PR**: 1 (foundation-skill-advisor)  
**Status**: active  

## What changed

This delta introduces the Skill Advisor engine as a new capability. The main spec at `openspec/specs/skill-advisor/spec.md` was created as part of this change.

### Key additions

| Component | Description |
|-----------|-------------|
| `Registry.Load()` | YAML directory loading with `slog.Warn` for malformed files |
| `ScoreSkillWeighted()` | Deterministic scoring with 5 weighted components (max 125) |
| `Recommend()` | Top-N recommendation with descending score sort |
| `DiscoverClient.Discover()` | HTTP client with TTL cache for skills.sh API |
| `TagsVector.ScoreSkill()` | Legacy tag overlap scorer (backward compat) |

### Scoring weights

| Component | Weight |
|-----------|--------|
| Language match | 10 |
| Framework match | 20 |
| Project type match | 30 |
| Verified publisher | 50 |
| Socket zero alerts | 15 |

### Implementation references

- `internal/skilladvisor/types.go` — Skill, ScoreResult, SkillQuery, Layer types
- `internal/skilladvisor/registry.go` — Registry with Load()
- `internal/skilladvisor/score.go` — ScoreSkill, ScoreSkillWeighted, Recommend
- `internal/skilladvisor/discover.go` — DiscoverClient, DiscoverCache, Discover()
- `internal/skilladvisor/skilladvisor_test.go` — 20+ test cases covering all components
