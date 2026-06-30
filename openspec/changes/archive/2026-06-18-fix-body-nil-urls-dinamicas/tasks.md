# Tasks: fix-body-nil-urls-dinamicas

## HelixDB Task References
- Task 5023 — Design approval
- Task 5024 — Implement ServicesConfig struct
- Task 5025 — Implement GetOllamaURL / GetHelixDBURL
- Task 5026 — Fix body nil in test_flow.go
- Task 5027 — Update TUI consumers (services_flow.go)
- Task 5028 — Update infra consumers (doctor, scheduler, helix)
- Task 5029 — Verify build and vet

## Fase 1: Fundación — ServicesConfig + URL helpers

- [x] 1.1 Agregar `ServicesConfig` struct a `internal/setup/config.go` con campos `OllamaURL` y `HelixDBURL`
- [x] 1.2 Agregar campo `Services ServicesConfig` al struct `Config`
- [x] 1.3 Implementar `GetOllamaURL()` con orden de precedencia: env var > config > default
- [x] 1.4 Implementar `GetHelixDBURL()` con orden de precedencia: env var > config > default

## Fase 2: Consumidores de URLs

- [x] 2.1 `internal/tui/test_flow.go` — Fix body nil (json.Marshal + bytes.NewReader) + usar `setup.GetOllamaURL()`
- [x] 2.2 `internal/tui/services_flow.go` — Usar `setup.GetOllamaURL()` y `setup.GetHelixDBURL()`
- [x] 2.3 `internal/setup/doctor.go` — Usar `GetHelixDBURL()` en `checkHelixHealth()`
- [x] 2.4 `internal/scheduler/config.go` — Usar `setup.GetHelixDBURL()` en `NewDefaultConfig()`
- [x] 2.5 `internal/db/helix/client.go` — Default `Options.BaseURL` usa `setup.GetHelixDBURL()`
- [x] 2.6 `internal/db/helix/embedding.go` — Fallback usa `setup.GetOllamaURL()`

## Fase 3: Verificación

- [x] 3.1 `go build ./...` compila sin errores
- [x] 3.2 `go vet ./...` sin warnings
- [x] 3.3 Sin import cycles

## Resumen

| Total | Completadas | Pendientes |
|-------|-------------|------------|
| 12    | 12          | 0          |
