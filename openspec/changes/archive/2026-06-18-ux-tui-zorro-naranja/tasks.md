# Tasks: ux-tui-zorro-naranja (Spec 5031)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 250–350 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Delivery strategy | single-pr |

## Phase A: New Orange Logo (REQ-1)

- [x] **D5032-1A** Create `internal/tui/assets/logo-new.txt` — Copia de `ascii-art-logo.txt` (raíz del proyecto) con el nuevo zorro estilo Z en 38 líneas
- [x] **D5032-1B** Modify `internal/tui/brand.go` — Add `//go:embed assets/logo-new.txt`, `logoNewArtRaw`, `init()` sanitization, `logoNewStyle` (naranja + bold), `RenderNewLogo()` function
- [x] **D5032-1C** Modify `internal/tui/brand_test.go` — Add `TestRenderNewLogo()` verifying non-empty output and ASCII character content
- [x] **D5032-1D** Modify `internal/opencode/tui-plugins/zorro-logo.tsx` — Replace old art (31 lines, green) with new art (38 lines, uses `theme.accent` = naranja). Update compact art and terminal threshold to 68 cols

## Phase B: Fix Bug #1 — Second Logo Corrupt (REQ-2)

- [x] **D5032-2A** `runSetupFlow()` (main.go:113) — Add `fmt.Print("\033[2J\033[H")` before `RenderBrand()`
- [x] **D5032-2B** `runAutostartFlow()` (main.go:198) — Add `fmt.Print("\033[2J\033[H")` before `RenderBrand()`

## Phase C: Navigation Modal — Clear Screens (REQ-3)

- [x] **D5032-3A** Add `fmt.Print("\033[2J\033[H")` in 8 points across `cmd/zyrocli/main.go`:
  - handleMenu() loop start
  - runInstallFlow() start
  - runInstallFlow() before RunConfirm
  - runSetupFlow() start
  - runModelsFlow() start
  - runModelsFlow() before RunConfirm
  - runModelsFlow() before tests
  - runAutostartFlow() start

## Phase D: About Menu (REQ-4)

- [x] **D5032-4A** Modify `internal/tui/menu.go` — Add `MenuItem{Key: "about", Label: "📖 Acerca de ZyroCLI", Description: "..."}`
- [x] **D5032-4B** Modify `cmd/zyrocli/main.go` — Add `case "about": runAboutFlow()` and new `runAboutFlow()` function with full descriptive text

## Phase E: Verification

- [x] E.1 Build — `go build ./...` compiles clean. No import cycles, no unused imports
- [x] E.2 Vet — `go vet ./...` passes clean
- [x] E.3 Test — `go test ./...` 17/18 pass (1 pre-existing failure: missing TTY in CI)
