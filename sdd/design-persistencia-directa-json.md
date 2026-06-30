# Design & Tasks — Persistencia directa a JSON

## Overview

**Problem:** `/zyro-model` TUI plugin uses `Bun.$ zyrocli profile set ...` to persist model assignments, but this fails inside OpenCode because PATH doesn't include the correct binary or executes an old version.

**Solution:** Replace `Bun.$` with direct JSON file writing using `Bun.write()`.

## Flow

```
assignModel()
  ├── api.client.global.config.update()    → en memoria (EXISTENTE)
  ├── refrescar api.state.config           → UI fresca (EXISTENTE, BUG 1)
  ├── persistAgentModel()                  → escribe JSON (NUEVO)
  │     ├── detectConfigPath()             → project-level > global
  │     ├── Bun.file().text() + JSON.parse → leer
  │     ├── merge updates in agent         → modificar
  │     └── Bun.write() + JSON.stringify   → escribir
  └── showAgentSelector()                  → UI (EXISTENTE)
```

## `detectConfigPath()` logic

```
SI existe .config/opencode/opencode.json en CWD → usar ESE
SINO → usar ~/.config/opencode/opencode.json
```

## `persistAgentModel()` function

```typescript
async function persistAgentModel(agentName: string, modelStr: string): Promise<void>
```

1. Determine configPath via detectConfigPath()
2. Read file: `const raw = await Bun.file(configPath).text()`
3. Parse: `const config = JSON.parse(raw)`
4. Ensure `config.agent` exists
5. If `agentName === "__SET_ALL__"`: loop all AGENTS, set each model
6. Else: set `config.agent[agentName].model = modelStr`
7. Write: `await Bun.write(configPath, JSON.stringify(config, null, 2) + "\n")`
8. NEVER throws — catches internally and logs to console.error

## Files modified

| File | Change |
|---|---|
| `internal/opencode/tui-plugins/zyro-model.tsx` | Add `persistAgentModel()` + `detectConfigPath()`, replace `Bun.$` |

## Tasks

| ID | Descripción |
|---|---|
| T-1 | Add `detectConfigPath()` helper |
| T-2 | Add `persistAgentModel()` function |
| T-3 | Replace `Bun.$` calls in `assignModel()` |

## Verification

1. `cat .config/opencode/opencode.json | grep model` shows updated model
2. Close and reopen OpenCode — model persists
3. `zyrocli profile list` shows the same model
4. Other JSON sections (mcp, skills, command) remain intact
