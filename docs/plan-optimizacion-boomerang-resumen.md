# Plan de Optimización Boomerang — Resumen Ejecutivo

> 17 Junio 2026 | Versión 1.0

---

## ⚡ El Problema en 3 Líneas

1. **RunPhase() ejecuta 6 pasos en TODAS las fases** — F0 no necesita Git ni Quality (~40% overhead)
2. **DelegateStep es bloqueante** — `cmd.Output()` + `wg.Wait()` congelan el orquestador, impidiendo recibir input del usuario
3. **Sin event loop** — no hay canales para progreso, cancelación, ni interleaving de input

---

## 🏗️ Solución: BoomerangAsync

Arquitectura basada en 3 pilares con **0 nuevas dependencias externas** (solo errgroup de golang.org/x/sync):

### 1. Phase Skip Matrix
```
F0: Memory✅ Think✅ Delegate✅ Git❌ Quality❌ Save✅   (~40% ahorro)
F1: Memory✅ Think✅ Delegate✅ Git❌ Quality❌ Save✅   (~40% ahorro)
F2: Memory✅ Think✅ Delegate✅ Git❌ Quality❌ Save✅   (~40% ahorro)
F3: Memory✅ Think✅ Delegate✅ Git✅ Quality✅ Save✅   (0% ahorro)
F4: Memory✅ Think❌ Delegate✅ Git✅ Quality❌ Save✅   (~35% ahorro)
```

### 2. Event Loop Asíncrono
```go
select {
case event  := <-o.bus:         // eventos de steps/subagentes
case input  := <-o.userInput:   // input del usuario
case <-ctx.Done():              // cancelación
case <-time.After(...):         // heartbeat
}
```

### 3. Delegate No Bloqueante
- `cmd.Start()` + `cmd.StdoutPipe()` reemplazan a `cmd.Output()`
- `errgroup.Group` reemplaza a `sync.WaitGroup` (con cancelación automática)
- Streaming de output línea por línea

---

## 📁 Archivos Afectados

### Nuevos (5 archivos)
| Archivo | Propósito |
|---------|-----------|
| `internal/boomerang/skip.go` | `PhaseStepMatrix`, `Step`, `ShouldRun()` |
| `internal/boomerang/events.go` | `EventType`, `StepEvent`, `EventBus` |
| `internal/boomerang/phase_config.go` | `PhaseConfigV2` con steps/async |
| `internal/boomerang/async_orchestrator.go` | `BoomerangAsyncOrchestrator`, `RunPhaseAsync()` |
| `internal/boomerang/state.go` | `PhaseState` mutable para cada fase |

### Modificados (6 archivos)
| Archivo | Cambio |
|---------|--------|
| `internal/boomerang/orchestrator.go` | `RunPhase()` usa skip matrix + agrega `runPhaseV2()` |
| `internal/boomerang/delegate.go` | Agrega `DelegateStepAsync()` (mantiene legacy) |
| `internal/boomerang/quality.go` | Agrega validación específica por fase |
| `internal/scheduler/scheduler.go` | Agrega `RunAsync()` |
| `internal/scheduler/approval.go` | Agrega `ApprovalGateAsync()` |
| `internal/boomerang/boomerang_test.go` | Tests para matriz, event bus, delegate async |

---

## 🗓️ Plan de Implementación (7 fases)

| # | Fase | Prioridad | Archivos | Esfuerzo |
|---|------|-----------|----------|----------|
| 1 | **Phase Skip Matrix** | P0 | `skip.go` (nuevo) + `orchestrator.go` (mod) | 1 día |
| 2 | **PhaseConfigV2** | P0 | `phase_config.go` (nuevo) | 0.5 día |
| 3 | **DelegateStep Async** | P1 | `delegate.go` (mod) | 2 días |
| 4 | **Event Bus + Event Loop** | P1 | `events.go`, `async_orchestrator.go`, `state.go` (nuevos) | 3 días |
| 5 | **Approval Gate Async** | P2 | `approval.go` (mod) | 1 día |
| 6 | **Scheduler Async** | P2 | `scheduler.go` (mod) | 2 días |
| 7 | **Legacy Cleanup** | P3 | Varios | 0.5 día |
| | **Total** | | ~9 archivos, ~930 LOC | ~10 días |

---

## ✅ Backward Compatibility

| API actual | Status | Alternativa nueva |
|------------|--------|-------------------|
| `RunPhase(ctx, PhaseConfig)` | ✅ Mantiene | `RunPhaseAsync(ctx, PhaseConfigV2)` |
| `DelegateStep(ctx, dag, phase)` | ✅ Mantiene | `DelegateStepAsync(ctx, dag, phase, events)` |
| `Scheduler.Run(ctx)` | ✅ Mantiene | `Scheduler.RunAsync(ctx)` |
| `ApprovalGate(phase, summary)` | ✅ Mantiene | `ApprovalGateAsync(phase, summary)` |

Todas las adiciones son **opt-in**. Tests existentes deben seguir pasando sin cambios.

---

## 🚨 Riesgos Principales

| Riesgo | Mitigación |
|--------|------------|
| Goroutine leak en delegateStepAsync | `errgroup.WithContext` auto-cancela + `defer` cleanup |
| Procesos huérfanos | `cmd.Cancel` (SIGINT) + `cmd.WaitDelay` (30s) |
| Race condition en estado compartido | `sync.Mutex` + `PhaseState` protegido |
| Event bus bloqueante | `select default` en Publish drop de eventos no críticos |
| **Correcciones aplicadas** | |
| Phase Skip Matrix F4 incorrecta | Corregido: Memory✅ Think❌ (F4 necesita hechos de F3 para archivar) |
| Goroutine leak en scanner loop | Corregido: scanner ahora escucha `ctx.Done()` vía goroutine + select |
| errgroup sin tolerancia a fallos parciales | Corregido: `FailurePolicyContinueOnError` para fases de investigación (F0, F1, F2) |
| sync.Mutex en PhaseState | Corregido: cambiado a `sync.RWMutex` — lecturas concurrentes no se bloquean |

---

## 📊 Beneficios Esperados

| Métrica | Antes | Después |
|---------|-------|---------|
| Tiempo F0 (investigación) | 100% (6 pasos) | ~60% (4 pasos) |
| Tiempo F4 (cierre) | 100% (6 pasos) | ~65% (4 pasos) |
| Responsividad durante delegate | ❌ Bloqueado | ✅ Input del usuario |
| Progreso en tiempo real | ❌ No | ✅ Streaming de output |
| Cancelación granular | ❌ Solo global | ✅ Por subagente |
| Validación por fase | ❌ Genérica | ✅ Específica |
| Dependencias nuevas | 0 | 0 (solo errgroup, stdlib) |

---

## 🔗 Referencias

- Plan detallado: `docs/plan-optimizacion-boomerang.md` (1,608 líneas)
- Investigación patrones: `docs/research/async-orchestration-patterns.md`
- Investigación librerías: `docs/research/async-go-libraries.md`
- Código actual: `internal/boomerang/orchestrator.go` (194 líneas)
- Código actual: `internal/boomerang/delegate.go` (73 líneas)

