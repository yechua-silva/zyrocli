# Spec Delta — Service Configuration & TUI Test Fix

## ADDED Requirements

### Requirement: Service URL Configuration

The system MUST provide dynamic service URL resolution with the following precedence order:
1. Environment variable (`OLLAMA_HOST` for Ollama, `HELIXDB_URL` for HelixDB)
2. Persistent config file (`~/.zyro/config.yaml` → `services.ollama_url` / `services.helixdb_url`)
3. Hardcoded default (`http://localhost:11434` / `http://localhost:6969`)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Env var set | `OLLAMA_HOST=http://ollama:11434` is set | `GetOllamaURL()` is called | Returns `http://ollama:11434` |
| Config file value | `~/.zyro/config.yaml` has `services.ollama_url: http://custom:11434` | `GetOllamaURL()` is called (no env var) | Returns `http://custom:11434` |
| Default fallback | No env var and no config file | `GetOllamaURL()` is called | Returns `http://localhost:11434` |
| Env var overrides config | Both `OLLAMA_HOST=env:11434` and config `ollama_url: config:11434` exist | `GetOllamaURL()` is called | Returns `env:11434` |
| HelixDB env var | `HELIXDB_URL=http://helix:6969` is set | `GetHelixDBURL()` is called | Returns `http://helix:6969` |
| HelixDB default fallback | No env var and no config | `GetHelixDBURL()` is called | Returns `http://localhost:6969` |

### Requirement: Config Struct for Services

The `Config` struct in `internal/setup/config.go` MUST include a `Services` field of type `ServicesConfig` with `ollama_url` and `helixdb_url` YAML mappings.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| YAML marshal | A `ServicesConfig` with `OllamaURL` and `HelixDBURL` is serialized | `yaml.Marshal(cfg)` | YAML output contains `services:` with `ollama_url` and `helixdb_url` keys |
| YAML unmarshal | YAML has `services.ollama_url` and `services.helixdb_url` | `yaml.Unmarshal(data, &cfg)` | `cfg.Services.OllamaURL` and `cfg.Services.HelixDBURL` are populated |

### Requirement: Test HTTP Body Fix

`TestEmbedding()` and `TestChat()` in `internal/tui/test_flow.go` MUST send a JSON body with `model` and `prompt` fields via `json.Marshal` + `bytes.NewReader` instead of passing `nil` as the HTTP body.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| TestEmbedding body | `TestEmbedding()` is called | HTTP POST request is built | Body is `{"model": <name>, "prompt": "ZyroAgentCLI test..."}` JSON |
| TestChat body | `TestChat()` is called | HTTP POST request is built | Body is `{"model": <name>, "prompt": "...", "stream": false}` JSON |
| No nil body | Either test function is called | HTTP request is inspected | Body is NOT `nil` |

### Requirement: Dynamic URLs in TUI consumers

`CheckHelixDB()` and `CheckOllama()` in `internal/tui/services_flow.go` MUST use `setup.GetHelixDBURL()` and `setup.GetOllamaURL()` respectively instead of hardcoded `http://localhost:6969` / `http://localhost:11434`.

### Requirement: Dynamic URL in doctor

`checkHelixHealth()` in `internal/setup/doctor.go` MUST use `GetHelixDBURL()` from the `setup` package instead of hardcoded `http://localhost:6969`.

### Requirement: Dynamic URL in scheduler

`NewDefaultConfig()` in `internal/scheduler/config.go` MUST use `setup.GetHelixDBURL()` as the default BaseURL for `helix.NewClient()` instead of hardcoded `http://localhost:6969`.

### Requirement: Dynamic URL in helix client

`NewClient()` in `internal/db/helix/client.go` MUST use `setup.GetHelixDBURL()` as the default `Options.BaseURL` instead of hardcoded `http://localhost:6969`.

### Requirement: Dynamic URL in embedding fallback

`embedOllama()` in `internal/db/helix/embedding.go` MUST use `setup.GetOllamaURL()` as the fallback BaseURL instead of hardcoded `http://localhost:11434`.

## No REMOVED or MODIFIED Requirements

This change adds new requirements. No existing requirements were removed or modified.
