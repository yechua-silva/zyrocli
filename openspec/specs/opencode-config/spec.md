# OpenCode Config Specification

## Purpose

Package `internal/opencode/` provides curated provider/model data and read/write access to `~/.config/opencode/opencode.json` for the Zyro CLI companion.

## Requirements

### Requirement: Provider and Model Structs

The package MUST define `Provider` (ID, Name, Models) and `Model` (ID, Name) struct types for representing AI providers and their available models.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| New provider instance | `Provider{}` is constructed | Accessing `.Models` | Returns empty `[]Model` slice |
| Model with fields | `Model{ID: "gpt-4", Name: "GPT-4"}` | Accessing fields | `.ID` is "gpt-4", `.Name` is "GPT-4" |

### Requirement: KnownProviders

The package MUST expose a `KnownProviders` variable with a curated list of providers (no HTTP calls). Provider list MUST match the user's `opencode.json` configuration:

- `opencode-go`: deepseek-v4-flash, deepseek-v4-pro, mimo-v2.5, mimo-v2.5-pro, qwen3.7-max, minimax-m3, kimi-k2.6
- `opencode`: deepseek-v4-flash-free, mimo-v2.5-free, nemotron-3-super-free, minimax-m3-free
- `google`: gemini-2.5-flash, gemini-2.5-pro
- `groq`: meta-llama/llama-4-scout-17b-16e-instruct
- `openrouter`: qwen/qwen3-coder:free
- `cerebras`: gpt-oss-120b
- `nvidia`: (dynamic via NVIDIA_API_KEY — unknown models)
- `anthropic`: claude-sonnet-4-6, claude-opus-4-20250514, claude-haiku-3-5

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Nvidia has no static models | `KnownProviders["nvidia"]` | Accessing `.Models` | Empty list — TUI shows "unknown models" |
| Provider count | `KnownProviders` is loaded | Counting entries | At least 8 curated providers |

### Requirement: ReadProviders

`ReadProviders(path string)` MUST parse the `providers` section from `opencode.json` and return merged `[]Provider`. JSON providers have priority over curated ones with the same ID. If the file is missing or has no `providers` section, MUST return the curated list only.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Valid JSON | opencode.json has `providers` with 3 entries | `ReadProviders(path)` | Returns 3+ curated = merged list |
| Missing file | opencode.json does not exist | `ReadProviders(path)` | Returns curated list, no error |
| No providers key | JSON exists but no `providers` key | `ReadProviders(path)` | Returns curated list only |
| Provider overlap | JSON overrides `anthropic.models` | Merged result | JSON models replace curated for anthropic |

### Requirement: ReadAgentConfigs

`ReadAgentConfigs(path string)` MUST parse the `agent` section from `opencode.json` and return `map[string]AgentConfig`. Returns empty map if missing. `AgentConfig` MUST have `Model`, `Mode`, `ReasoningEffort` string fields.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Agent section exists | JSON has `agent.sdd-apply` entry | `ReadAgentConfigs(path)` | Returns map with key "sdd-apply" |
| No agent section | JSON has no `agent` key | `ReadAgentConfigs(path)` | Returns empty map, no error |

### Requirement: WriteAgentConfig

`WriteAgentConfig(path, profileName, configs map[string]AgentConfig)` MUST write to the `agent` section of `opencode.json`. Persists existing JSON keys, overwrites `agent.{profileName}` with the new configs.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Write new config | `opencode.json` exists with `$schema` and `providers` | `WriteAgentConfig(path, "zyro", cfg)` | `agent.sdd-orchestrator-zyro` is written, `$schema` is preserved |
| Overwrite existing | `agent` section already has entries | `WriteAgentConfig(path, "zyro", cfg)` | Only the `zyro` profile is updated, other agents preserved |
| Non-existent file | File does not exist | `WriteAgentConfig(path, "zyro", cfg)` | Returns error — file must exist |

## Removed Requirements

The hardcoded `modelPool` in `cmd/zyrocli/profile_tui.go` is replaced by `KnownProviders` + `ReadProviders()`.
