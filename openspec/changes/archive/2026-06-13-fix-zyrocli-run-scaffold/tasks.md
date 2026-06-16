# Tasks: fix-zyrocli-run-scaffold

## Phase 1: Limpieza de imports y variables obsoletas
- [x] 1.1 Eliminar import `"context"` de `cmd/zyrocli/run.go`
- [x] 1.2 Eliminar import `"github.com/secko/zyrocli/internal/scheduler"` de `cmd/zyrocli/run.go`
- [x] 1.3 Eliminar variable `var runPhase string`

## Phase 2: Eliminar lógica del pipeline SDD
- [x] 2.1 Eliminar líneas 57-61: `scheduler.LoadConfig("handoff.yaml")` y su manejo de error
- [x] 2.2 Eliminar líneas 63-69: construcción de `runners` slice con F1Runner..F4Runner
- [x] 2.3 Eliminar líneas 71-74: `scheduler.NewScheduler`, `context.Background()`, `var results`
- [x] 2.4 Eliminar líneas 76-104: bloque completo if/else de `runPhase` (RunPhase + Run)
- [x] 2.5 Eliminar líneas 106-117: bloque de print summary (results loop)

## Phase 3: Insertar scaffold+opencode flow
- [x] 3.1 Después de la línea 55 (`handoff.yaml not found` error check), insertar: `handoff.Parse("handoff.yaml")` + `handoff.Validate(payload)` con manejo de error
- [x] 3.2 Insertar: `os.ReadFile("handoff.yaml")` para obtener raw bytes (patrón init.go línea 49)
- [x] 3.3 Insertar: mapeo `payload → scaffold.Config` (mismo patrón init.go líneas 56-67, sin `--scaffold` flag check)
- [x] 3.4 Insertar: `exec.LookPath("opencode")` — lenient, warn si falta, setear `cfg.LaunchOpenCode = false`
- [x] 3.5 Insertar: `scaffold.Run(cfg)` con manejo de error
- [x] 3.6 Insertar: print success summary
- [x] 3.7 Insertar: `exec.Command("opencode", result.TargetDir)` — solo si opencode encontrado

## Phase 4: Eliminar flag --phase
- [x] 4.1 Eliminar registro de flag `--phase` en init()
- [x] 4.2 Actualizar `Short` y `Long` del command

## Phase 5: Verificación
- [x] 5.1 `go build ./cmd/zyrocli/...` — ✅ pasó
- [x] 5.2 `go vet ./cmd/zyrocli/...` — ✅ pasó
- [x] 5.3 `go test ./...` — ✅ pasó
- [x] 5.4 Verificar `--phase` ya no aparece en help — ✅ confirmado
