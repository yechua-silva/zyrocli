# Design — FIX: Body nil en test_flow + URLs dinámicas con orden de precedencia

**HelixDB Design ID:** 5021
**Spec ID:** 5020
**Estado:** Draft
**Proyecto:** ZyroAgentCLI

---

## Resumen

Dos bugs independientes pero en los mismos archivos:

1. **Body nil en `test_flow.go`** — `TestEmbedding()` y `TestChat()` envían body `nil` en
   `client.Post()`. La API de Ollama requiere un JSON body con `model` y `prompt`.

2. **URLs dinámicas con orden de precedencia** — Todas las URLs de Ollama y HelixDB están
   hardcodeadas como `http://localhost:11434` y `http://localhost:6969`. Deben resolverse
   dinámicamente con la precedencia: `env var > ~/.zyro/config.yaml > default`.

---

## Bug 1: Body nil en test_flow.go

**Archivo:** `internal/tui/test_flow.go`

**Problema:** Las funciones `TestEmbedding()` y `TestChat()` pasan `nil` como body en
`client.Post()`. Ollama ignora requests sin body JSON.

**Fix:** Construir JSON body con `{"model": modelo, "prompt": "ZyroAgentCLI test..."}`
usando `json.Marshal` + `bytes.NewReader`.

**Imports a agregar:** `bytes`, `encoding/json`

**Código nuevo (ambas funciones):**
```go
body := map[string]string{
    "model":  model,
    "prompt": "ZyroAgentCLI test - " + model,
}
jsonBody, _ := json.Marshal(body)
// luego en client.Post:
client.Post(url, "application/json", bytes.NewReader(jsonBody))
```

**No cambia:** Lógica de negocio, nombres de funciones exportadas, API pública.

---

## Bug 2: URLs dinámicas con orden de precedencia

### Orden de precedencia
1. **Env var** — `OLLAMA_HOST` para Ollama, `HELIXDB_URL` para HelixDB
2. **Config file** — `~/.zyro/config.yaml` > `services.ollama_url` / `services.helixdb_url`
3. **Default hardcodeado** — `http://localhost:11434` / `http://localhost:6969`

### Archivos a modificar (7 files)

#### 1. `internal/setup/config.go` — Estructura y funciones getter

**Agregar al struct `Config`:**
```go
type ServicesConfig struct {
    OllamaURL  string `yaml:"ollama_url"`
    HelixDBURL string `yaml:"helixdb_url"`
}
```

Y en `Config`:
```go
Services ServicesConfig `yaml:"services"`
```

**Nuevas funciones:**
```go
func GetOllamaURL() string {
    if env := os.Getenv("OLLAMA_HOST"); env != "" {
        return env
    }
    cfg, err := LoadConfig()
    if err == nil && cfg.Services.OllamaURL != "" {
        return cfg.Services.OllamaURL
    }
    return "http://localhost:11434"
}

func GetHelixDBURL() string {
    if env := os.Getenv("HELIXDB_URL"); env != "" {
        return env
    }
    cfg, err := LoadConfig()
    if err == nil && cfg.Services.HelixDBURL != "" {
        return cfg.Services.HelixDBURL
    }
    return "http://localhost:6969"
}
```

**Import a agregar:** `"os"` (ya existe)

#### 2. `internal/tui/test_flow.go` — Fix body nil + URL dinámica

**Cambios:**
- Agregar imports: `bytes`, `encoding/json`, `"github.com/secko/zyrocli/internal/setup"`
- En `TestEmbedding()`: usar `setup.GetOllamaURL() + "/api/embeddings"` y JSON body
- En `TestChat()`: usar `setup.GetOllamaURL() + "/api/generate"` y JSON body

#### 3. `internal/tui/services_flow.go` — URLs dinámicas

**Cambios:**
- Agregar import: `"github.com/secko/zyrocli/internal/setup"`
- `CheckHelixDB()`: cambiar `http://localhost:6969` → `setup.GetHelixDBURL()`
- `CheckOllama()`: cambiar `http://localhost:11434` → `setup.GetOllamaURL()`

#### 4. `internal/setup/doctor.go` — HelixDB URL dinámica

**Cambios:**
- En `checkHelixHealth()`: cambiar `http://localhost:6969` → `GetHelixDBURL()`

#### 5. `internal/scheduler/config.go` — HelixDB URL dinámica

**Cambios:**
- Agregar import: `"github.com/secko/zyrocli/internal/setup"`
- En `NewDefaultConfig()`: cambiar `helix.WithBaseURL("http://localhost:6969")` →
  `helix.WithBaseURL(setup.GetHelixDBURL())`

#### 6. `internal/db/helix/client.go` — Default Options

**Cambios:**
- Agregar import: `"github.com/secko/zyrocli/internal/setup"`
- En `NewClient()`, default `Options{BaseURL: "http://localhost:6969"}` →
  `Options{BaseURL: setup.GetHelixDBURL()}`

#### 7. `internal/db/helix/embedding.go` — Fallback Ollama URL

**Cambios:**
- Agregar import: `"github.com/secko/zyrocli/internal/setup"`
- En `embedOllama()`, fallback `baseURL = "http://localhost:11434"` →
  `baseURL = setup.GetOllamaURL()`

---

## Diagrama de flujo de datos

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

---

## Archivos afectados

| # | Archivo | Cambio |
|---|---------|--------|
| 1 | `internal/setup/config.go` | +`ServicesConfig` struct, +`GetOllamaURL()`, +`GetHelixDBURL()` |
| 2 | `internal/tui/test_flow.go` | Fix body nil + usar `setup.GetOllamaURL()` |
| 3 | `internal/tui/services_flow.go` | Usar `setup.GetOllamaURL()` y `setup.GetHelixDBURL()` |
| 4 | `internal/setup/doctor.go` | Usar `setup.GetHelixDBURL()` |
| 5 | `internal/scheduler/config.go` | Usar `setup.GetHelixDBURL()` |
| 6 | `internal/db/helix/client.go` | Default Options usa `setup.GetHelixDBURL()` |
| 7 | `internal/db/helix/embedding.go` | Fallback usa `setup.GetOllamaURL()` |

### Fuera de alcance (NO TOCAR)
- `scripts/install_tui.py`
- `mcp-tools/embedding_harness.py`

### Lo que NO cambia
- Lógica de negocio de cada función
- Nombres de funciones exportadas
- API pública de los paquetes
- Tests existentes

---

## Orden de cambios sugerido

### Fase 1: Fundación (sin dependencias)
1. **`internal/setup/config.go`** — Agregar `ServicesConfig`, `GetOllamaURL()`, `GetHelixDBURL()`
   - Base para todos los demás cambios
   - Se puede testear unitariamente

### Fase 2: Consumidores de URLs (dependen de Fase 1)
2. **`internal/db/helix/client.go`** — Default usa `setup.GetHelixDBURL()`
3. **`internal/db/helix/embedding.go`** — Fallback usa `setup.GetOllamaURL()`
4. **`internal/setup/doctor.go`** — Usa `setup.GetHelixDBURL()`
5. **`internal/scheduler/config.go`** — Usa `setup.GetHelixDBURL()`

### Fase 3: TUI consumers (dependen de Fase 1 + Fase 2)
6. **`internal/tui/services_flow.go`** — URLs dinámicas
7. **`internal/tui/test_flow.go`** — Body nil + URL dinámica

---

## Verificación

- `go build ./...` debe compilar sin errores
- `go test ./...` debe pasar (tests existentes no se modifican)
- `go vet ./...` sin warnings
- Verificar que `GetOllamaURL()` respeta precedencia: env > config > default
- Verificar que `GetHelixDBURL()` respeta precedencia: env > config > default
- Verificar que `TestEmbedding()` y `TestChat()` ya no pasan body nil
