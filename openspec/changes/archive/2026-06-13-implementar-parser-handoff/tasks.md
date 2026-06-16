# Tasks: Implementar parser de handoff.yaml + comando init

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~370 (additions + deletions across 6 files) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Complete handoff parser + CLI init + tests | PR 1 | All 6 files in one PR, under 400-line budget |

## Phase 1: Foundation — Structs v2.0

- [x] 1.1 Reemplazar structs en `internal/handoff/payload.go` con el contrato v2.0 completo: `Source`, `Project`, `ValidatedIdea`, `UserStory`, `MVP`, `Governance`, `ApprovalPoint`, `Testing`, `Limits`, `Payload`
- [x] 1.2 Eliminar struct `Phase` y campo `Phases` que no existen en v2.0
- [x] 1.3 Agregar tags `yaml:"..."` y `omitempty` donde aplique (campos opcionales)

## Phase 2: Parser

- [x] 2.1 Crear `internal/handoff/parser.go` con función `Parse(path string) (*Payload, error)`
- [x] 2.2 Implementar soporte archivo: `os.ReadFile(path)` + `yaml.Unmarshal`
- [x] 2.3 Implementar soporte stdin: path `"-"` → `io.ReadAll(os.Stdin)` + `yaml.Unmarshal`
- [x] 2.4 Wrappear errores con `fmt.Errorf("parse: %w", err)` para trazabilidad

## Phase 3: Validación

- [x] 3.1 Crear `internal/handoff/validate.go` con `Validate(p *Payload) error`
- [x] 3.2 Regla: `version` debe ser exactamente `"2.0"`
- [x] 3.3 Regla: `source.system` requerido (non-empty)
- [x] 3.4 Reglas: `project.name` y `project.language` requeridos
- [x] 3.5 Reglas: `governance.mode` y `testing.strategy` requeridos
- [x] 3.6 Usar `errors.Join()` para recolectar TODOS los errores, no solo el primero

## Phase 4: CLI

- [x] 4.1 Crear `cmd/zyrocli/init.go` con subcomando `init` de Cobra
- [x] 4.2 Init acepta un argumento posicional (path o `"-"`), valida `len(args) == 1`
- [x] 4.3 Init llama `Parse` + `Validate`, imprime resumen JSON del Payload o error
- [x] 4.4 Registrar `initCmd` en `rootCmd` via `init()` en el mismo archivo

## Phase 5: Configs

- [x] 5.1 Actualizar `handoff.yaml` raiz a v2.0: reemplazar `source.type` → `source.system`, eliminar `phases`, agregar campos faltantes (`validated_idea`, `user_story`, `mvp`, `governance.mode`, `testing.strategy`)

## Phase 6: Tests

- [x] 6.1 Crear `internal/handoff/payload_test.go` con tests table-driven usando `t.Run`
- [x] 6.2 Test parser: archivo válido con `t.TempDir()`, stdin vía pipe, yaml malformado, archivo no existe
- [x] 6.3 Test validate: payload completo pasa, cada campo requerido faltante individualmente, version `"1.0"` rechazada
- [x] 6.4 Test integración: Parse + Validate con YAML completo v2.0 → cero errores
- [x] 6.5 Test multi-error: Validate con 3 campos faltantes → error contiene las 3 violaciones

## Phase 7: Verification

- [x] 7.1 `go vet ./...` — zero diagnostics
- [x] 7.2 `go build ./...` — compila sin errores
- [x] 7.3 `go test ./internal/handoff/...` — todos los tests pasan
- [x] 7.4 `go test -cover ./internal/handoff/...` — cobertura ≥ 80%
- [x] 7.5 `go run ./cmd/zyrocli init handoff.yaml` — parsea y muestra resumen
- [x] 7.6 `echo 'version: "2.0"' | go run ./cmd/zyrocli init -` — stdin funciona
