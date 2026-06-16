# Exploration: ZyroAgentCLI Fase 1 — Arquitectura Decisional v2

> SDD Explore Phase · Engram topic: `explore/zyroagentcli-fase1`

## Summary

Refactor TUI from flat list (7 phases × 5 models) to 2-step flow (provider→model) with 4 renamed phases, writing to `~/.config/opencode/opencode.json` instead of per-project profile files.

## Key Findings

1. **TUI is a flat list** — no provider abstraction, 5 hardcoded models, 7 SDD phases
2. **No new deps needed** — cobra, bubbletea, lipgloss, yaml.v3 already in go.mod
3. **internal/opencode/ is new** — 11 existing internal packages, this one doesn't exist yet
4. **No existing specs** for profile/opencode/model-selection in openspec/specs/
5. **Plan exists** in docs/plan-fase1.md (102 lines) with exact output format

## Recommendation

Approach 1: Refactor in place, extract data to `internal/opencode/`. TUI stays in `cmd/zyrocli/`, data layer moves to `internal/opencode/`. ~350 lines new, ~900 lines replaced across 5 files.

## Status: Ready for Proposal
