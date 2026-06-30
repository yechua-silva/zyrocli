# Design — FIX: Body nil en test_flow + URLs dinámicas con orden de precedencia

**Estado:** Implementado ✅

## Resumen

Dos bugs independientes pero en los mismos archivos:

1. **Body nil en `test_flow.go`** — `TestEmbedding()` y `TestChat()` enviaban body `nil` en
   `client.Post()`. La API de Ollama requiere un JSON body con `model` y `prompt`.

2. **URLs dinámicas con orden de precedencia** — Todas las URLs de Ollama y HelixDB están
   hardcodeadas como `http://localhost:11434` y `http://localhost:6969`. Deben resolverse
   dinámicamente con la precedencia: `env var > ~/.zyro/config.yaml > default`.

## Bug 1: Body nil en test_flow.go

**Archivo:** `internal/tui/test_flow.go`

**Problema:** Las funciones `TestEmbedding()` y `TestChat()` pasaban `nil` como body en
`client.Post()`. Ollama ignora requests sin body JSON.

**Fix:** Construir JSON body con `{"model": modelo, "prompt": "ZyroAgentCLI test..."}`
usando `json.Marshal` + `bytes.NewReader`.

**Imports agregados:** `bytes`, `encoding/json`, `setup`

## Bug 2: URLs dinámicas con orden de precedencia

### Orden de precedencia
1. **Env var** — `OLLAMA_HOST` para Ollama, `HELIXDB_URL` para HelixDB
2. **Config file** — `~/.zyro/config.yaml` > `services.ollama_url` / `services.helixdb_url`
3. **Default hardcodeado** — `http://localhost:11434` / `http://localhost:6969`

### Archivos modificados (7 files)

| # | Archivo | Cambio |
|---|---------|--------|
| 1 | `internal/setup/config.go` | +`ServicesConfig` struct, +`GetOllamaURL()`, +`GetHelixDBURL()` |
| 2 | `internal/tui/test_flow.go` | Fix body nil + usar `setup.GetOllamaURL()` |
| 3 | `internal/tui/services_flow.go` | Usar `setup.GetOllamaURL()` y `setup.GetHelixDBURL()` |
| 4 | `internal/setup/doctor.go` | Usar `setup.GetHelixDBURL()` |
| 5 | `internal/scheduler/config.go` | Usar `setup.GetHelixDBURL()` |
| 6 | `internal/db/helix/client.go` | Default Options usa `setup.GetHelixDBURL()` |
| 7 | `internal/db/helix/embedding.go` | Fallback usa `setup.GetOllamaURL()` |

### Diagrama de flujo de datos

```
+------------------+     +-------------------+
|  os.Getenv()     |     | ~/.zyro/config.   |
|  OLLAMA_HOST /   |     | yaml              |
|  HELIXDB_URL     |     | services:         |
|                  |     |   ollama_url      |
|  (mayor prior.)  |     |   helixdb_url     |
+--------+---------+     +--------+----------+
         |                        |
         v                        v
+------------------------------------------+
|  setup.GetOllamaURL()                     |
|  setup.GetHelixDBURL()                    |
|                                           |
|  1. Si env var existe → return env        |
|  2. Si config existe y tiene valor → ret  |
|  3. Default hardcodeado → return          |
+------------------------------------------+
    |                    |
    v                    v
[ test_flow.go ]    [ services_flow.go ]
[ doctor.go    ]    [ scheduler/config.go ]
[ db/helix/     ]   [ db/helix/embedding.go ]
```

### Sin import cycles
- `internal/setup` solo importa stdlib + `yaml.v3`
- Ningún paquete tiene import cycle con `setup`
