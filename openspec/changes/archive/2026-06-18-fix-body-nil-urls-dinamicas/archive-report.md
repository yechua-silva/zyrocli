# Archive Report — fix-body-nil-urls-dinamicas

**Change**: FIX: Body nil en test_flow + URLs dinámicas con orden de precedencia
**Archived to**: `openspec/changes/archive/2026-06-18-fix-body-nil-urls-dinamicas/`
**Date**: 2026-06-18

## HelixDB Tasks
Tasks 5023–5029 were completed during implementation. HelixDB was unreachable (HTTP 500 on `/v1/query`) at archive time, so task status marking via HelixDB API could not be performed. The task board (`zyro-task-board`) also did not find these task IDs — they exist as HelixDB graph nodes rather than task board entries.

## Specs Synced
| Domain | Action | Details |
|--------|--------|---------|
| setup-configuration | Created | New domain spec for service URL configuration and TUI testing infrastructure |

## Archive Contents
- state.yaml ✅
- design.md ✅
- specs/setup-configuration/spec.md ✅ (delta spec)
- tasks.md ✅ (12/12 tasks complete)
- verify-report.md ✅ (PASS)

## Verificación
- `go build ./...` ✅ sin errores
- `go vet ./...` ✅ sin errores
- Sin import cycles ✅
- `GetOllamaURL()` respeta precedencia: env > config > default ✅
- `GetHelixDBURL()` respeta precedencia: env > config > default ✅
- `TestEmbedding()` y `TestChat()` ya no pasan body nil ✅

## Source of Truth Updated
The following spec now reflects the new behavior:
- `openspec/specs/setup-configuration/spec.md`

## SDD Cycle Complete
The change has been fully implemented, verified, and archived.

### Summary
- **Bug fix**: Body nil en `internal/tui/test_flow.go` — ahora envía JSON body con `json.Marshal` + `bytes.NewReader`
- **Feature**: URLs dinámicas con orden de precedencia (env var > YAML > default) via `setup.GetOllamaURL()` y `setup.GetHelixDBURL()`
- **Files**: 7 archivos modificados (1 core + 6 consumidores)
- **Build**: ✅ go build + go vet clean
