# Plan de Ejecución: Fase 1 — Companion Funcional

> Junio 2026 · Basado en Arquitectura Decisional v2

---

## Premisa

- **OpenCode lidera** el pipeline de desarrollo
- **ZyroCLI configura** el ecosistema (modelos, perfiles, contexto HelixDB)
- ZyroCLI NO orquesta a OpenCode, NO compite con él

---

## Stack

- Go + Cobra + yaml.v3 + bubbletea
- HelixDB (Go SDK, puerto **6969** — no 8500)
- Sin gRPC, sin protobuf, sin Python server persistente
- os/exec para scripts Python (suficiente, no cambiar)

---

## /zyro-model y zyrocli profile tui

### Cómo funciona (igual que gentle-ai)

El TUI tiene **dos pasos**:
1. El usuario elige el **provider** (anthropic, openrouter, nvidia, etc.)
2. El usuario elige el **modelo** dentro de ese provider

Esto se repite para cada fase SDD. Al confirmar, escribe en
`~/.config/opencode/opencode.json` en este formato:

```json
{
  "agents": {
    "sdd-orchestrator-{profile}": {
      "model": "anthropic/claude-opus-4-20250514",
      "mode": "primary"
    },
    "sdd-apply-{profile}": {
      "model": "openrouter/deepseek/deepseek-v4-flash:free",
      "mode": "subagent"
    }
  }
}
```

### Fuente de modelos

1. Leer `~/.config/opencode/opencode.json` → sección `provider`
2. Las keys de `provider` son los IDs de providers configurados
3. Para cada provider, mostrar su lista de modelos conocidos (lista curada en
   `internal/opencode/models.go`, una entrada por provider)
4. NO hacer llamadas HTTP a APIs externas para descubrir modelos
5. NO leer desde ningún otro lugar que no sea `opencode.json`

### Fases Zyro-SDD

El TUI usa estas fases con prefijo `zyro-`:
- zyro-sdd-explorer-stack (Fase 0)
- zyro-sdd-planning (Fase 1)
- zyro-sdd-implement (Fase 2)
- zyro-sdd-verify (Fase 3)
- (y así sucesivamente)

---

## Archivos a crear/modificar

| Archivo | Acción | Contenido |
|---------|--------|-----------|
| `internal/opencode/models.go` | Nuevo | Structs Provider, Model. Lista curada de providers con sus modelos conocidos |
| `internal/opencode/opencode.go` | Nuevo | Reader/Writer de opencode.json |
| `cmd/zyrocli/profile.go` | Modificar | Subcomando tui lanza TUI de dos pasos |
| `cmd/zyrocli/profile_tui.go` | Reemplazar | Nuevo modelo bubbletea: paso 1 = provider, paso 2 = modelo |
| `cmd/zyrocli/profile_tui_test.go` | Reemplazar | Tests del nuevo flujo |

---

## Ruta completa (Fases 1→2→3)

### Fase 1 — Companion Funcional
- [ ] Refactor zyrocli profile tui (2-pasos + opencode.json)
- [ ] Crear internal/opencode/models.go (lista curada providers + modelos)
- [ ] Crear internal/opencode/opencode.go (reader/writer)
- [ ] zyrocli sync (refresca modelos en runtime)
- [ ] zyrocli init mejorado (scaffold + .zyro/config.yaml)

### Fase 2 — HelixDB Integration
- [ ] internal/db/helix/schema.go — CreateSchemaIndexes()
- [ ] internal/db/helix/client.go — Wrapper con tenant injection
- [ ] zyrocli absorb — ingesta .docs/ → Doc nodes
- [ ] zyrocli db init — arranca schema + índices
- [ ] Context bridge mejorado — query HelixDB → summaries

### Fase 3 — Multi-tenant
- [ ] Developer node como tenant aislado
- [ ] Cross-project Skill sharing
- [ ] CodeNode summaries automatizados
- [ ] zyrocli context [task] — inyección de contexto
