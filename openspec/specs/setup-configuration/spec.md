# Setup & Configuration Specification

## Purpose

Service URL resolution for external services (Ollama, HelixDB) with configurable precedence, and the TUI health-check/testing infrastructure.

## Requirements

### Requirement: Service URL Resolution

The system MUST provide dynamic service URL resolution with the following precedence order:
1. Environment variable (`OLLAMA_HOST` for Ollama, `HELIXDB_URL` for HelixDB)
2. Persistent config file (`~/.zyro/config.yaml` → `services.ollama_url` / `services.helixdb_url`)
3. Hardcoded default (`http://localhost:11434` / `http://localhost:6969`)

<details>
<summary>Scenarios</summary>

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Env var set | `OLLAMA_HOST=http://ollama:11434` is set | `GetOllamaURL()` is called | Returns `http://ollama:11434` |
| Config file value | `~/.zyro/config.yaml` has `services.ollama_url: http://custom:11434` | `GetOllamaURL()` is called (no env var) | Returns `http://custom:11434` |
| Default fallback | No env var and no config file | `GetOllamaURL()` is called | Returns `http://localhost:11434` |
| Env var overrides config | Both `OLLAMA_HOST=env:11434` and config `ollama_url: config:11434` exist | `GetOllamaURL()` is called | Returns `env:11434` |
| HelixDB env var | `HELIXDB_URL=http://helix:6969` is set | `GetHelixDBURL()` is called | Returns `http://helix:6969` |
| HelixDB default fallback | No env var and no config | `GetHelixDBURL()` is called | Returns `http://localhost:6969` |
</details>

### Requirement: ServicesConfig Struct

The `Config` struct in `internal/setup/config.go` MUST include a `Services` field of type `ServicesConfig` with `ollama_url` and `helixdb_url` YAML mappings.

<details>
<summary>Scenarios</summary>

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| YAML marshal | A `ServicesConfig` with populated fields is serialized | `yaml.Marshal(cfg)` | YAML output contains `services:` with `ollama_url` and `helixdb_url` keys |
| YAML unmarshal | YAML has `services.ollama_url` and `services.helixdb_url` | `yaml.Unmarshal(data, &cfg)` | `cfg.Services.OllamaURL` and `cfg.Services.HelixDBURL` are populated |
</details>

### Requirement: TUI Test HTTP Body

`TestEmbedding()` and `TestChat()` in `internal/tui/test_flow.go` MUST send a JSON body with `model` and `prompt` fields via `json.Marshal` + `bytes.NewReader` instead of passing `nil` as the HTTP body.

<details>
<summary>Scenarios</summary>

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| TestEmbedding body | `TestEmbedding()` is called | HTTP POST request is built | Body is `{"model": <name>, "prompt": "ZyroAgentCLI test..."}` JSON |
| TestChat body | `TestChat()` is called | HTTP POST request is built | Body is `{"model": <name>, "prompt": "...", "stream": false}` JSON |
| No nil body | Either test function is called | HTTP request is inspected | Body is NOT `nil` |
</details>

### Requirement: Dynamic URLs in Consumers

All consumers of Ollama and HelixDB URLs MUST use the dynamic URL resolvers (`setup.GetOllamaURL()` / `setup.GetHelixDBURL()`) instead of hardcoded `localhost` URLs.

<details>
<summary>Affected consumers</summary>

| File | Function | Change |
|------|----------|--------|
| `internal/tui/services_flow.go` | `CheckHelixDB()` | `http://localhost:6969` → `setup.GetHelixDBURL()` |
| `internal/tui/services_flow.go` | `CheckOllama()` | `http://localhost:11434` → `setup.GetOllamaURL()` |
| `internal/setup/doctor.go` | `checkHelixHealth()` | `http://localhost:6969` → `GetHelixDBURL()` |
| `internal/scheduler/config.go` | `NewDefaultConfig()` | Hardcoded → `setup.GetHelixDBURL()` |
| `internal/db/helix/client.go` | `NewClient()` | Default → `setup.GetHelixDBURL()` |
| `internal/db/helix/embedding.go` | `embedOllama()` | Fallback → `setup.GetOllamaURL()` |
</details>

### Requirement: No Import Cycles

The `internal/setup` package MUST NOT import any project-internal package. It MAY only import stdlib and `gopkg.in/yaml.v3`. No consumer package MAY form an import cycle with `internal/setup`.
