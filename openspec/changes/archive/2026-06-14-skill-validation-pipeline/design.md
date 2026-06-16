# Design: Skill Validation Pipeline

## Architecture Overview

Un pipeline único de 6 capas que reemplaza los dos caminos actuales (run.go usa solo registry local; scheduler F1 usa solo API skills.sh). El entry point es `DiscoverAndRank(payload, n)` que orquesta: BuildDiscoveryQuery → exec python3 scripts/discover.py + Registry.LoadDefaults() → Merge → ValidateAndScore → sort + top-N.

**Principio arquitectónico**: Python para tools operativas (discovery, linter, test-runner, explorer), Go solo para orquestación y validación determinística.

```
handoff.Payload
       │
       ▼
BuildDiscoveryQuery() ──→ DiscoveryQuery{Language, Framework, ProjectType, Keywords}
       │
       ├──── exec python3 scripts/discover.py ──┐
       │     (JSON on stdout)                     │
       │                                          │
       └──── Registry.LoadDefaults() ─────────────┤
                                                  │
                                        Merge (local wins on dupes)
                                                  │
                                                  ▼
                                        ValidateAndScore()
                                         1. SocketAlerts > 0 → REJECT (hard block)
                                         2. Publisher ∉ whitelist → penalty -50
                                         3. Language mismatch → -10
                                         4. Framework mismatch → -20
                                         5. ProjectType mismatch → -30
                                         6. ScoreSkillWeighted()
                                                  │
                                                  ▼
                                        Sort desc + top-N
                                                  │
                                                  ▼
                                        []ValidatedSkill
```

## Types

### types.go (modificado)

Añadir tres tipos nuevos sin eliminar los existentes (`Skill`, `ScoreResult`, `SkillQuery`):

```go
type DiscoveryQuery struct {
    Language    string
    Framework   string
    ProjectType string
    Keywords    []string
}

type ValidationError struct {
    HardBlocked bool
    Reason      string
}

type ValidatedSkill struct {
    Skill           SkillEntry
    Score           ScoreResult
    Rejected        bool
    RejectReason    string
    ValidationError *ValidationError
}
```

`DiscoveryQuery` se construye desde `handoff.Payload` y enriquece `SkillQuery` con keywords. `ValidatedSkill` envuelve cada skill con su score + metadata de validación. `ValidationError` se usa para hard blocks (SocketAlerts).

## Pipeline Flow

### Happy Path

```
payload → BuildDiscoveryQuery
  → exec python3 scripts/discover.py + Registry.LoadDefaults() (concurrent)
  → merge (local wins on duplicate names)
  → ValidateAndScore: all layers pass, cumulative penalties
  → sort by Score.Normalized() desc
  → return top-N []ValidatedSkill
```

### Discover.py Failure (graceful degradation)

```
payload → BuildDiscoveryQuery
  → exec python3 scripts/discover.py → ERROR (Python not found, HTTP timeout, malformed JSON)
  → log warning: "discover.py unavailable, using local registry"
  → Registry.LoadDefaults() → local skills
  → ValidateAndScore
  → return []ValidatedSkill (from local only)
```

### Hard Block (SocketAlerts > 0)

```
→ ValidateAndScore
  → Layer 1: skill.SocketAlerts > 0
  → ValidationError{HardBlocked: true, Reason: "socket_alerts=N"}
  → Skill excluded from results immediately (no further scoring)
```

### Unknown Publisher

```
→ ValidateAndScore
  → Layer 2: publisher "unknown-corp" ∉ KnownPublishers
  → VerifiedBonus forced to 0 (even if skill.Verified=true)
  → Cumulative penalty: -50
  → Skill still included but ranked lower
```

## Component Details

### query.go (nuevo — `internal/skilladvisor/query.go`)

| Function | Signature | Description |
|----------|-----------|-------------|
| `BuildDiscoveryQuery` | `(payload *handoff.Payload) DiscoveryQuery` | Extrae Language/ProjectType del payload, infiere Framework, extrae Keywords |
| `detectFramework` | `(payload *handoff.Payload) string` | Mapping determinístico por language |
| `extractKeywords` | `(payload *handoff.Payload) []string` | Parsea MVP.Scope + Features + ValidatedIdea.Problem, tokens lowercased, deduplicated, max 10 |

**detectFramework mapping:**

| Language | Conditions | Return |
|----------|-----------|--------|
| go | `spf13/cobra` en dependencies/imports | `"cobra"` |
| go | `gin-gonic` o `labstack` en dependencies | `"gin"` o `"echo"` |
| go | Default | `""` |
| typescript/javascript | `astro` en scope | `"astro"` |
| typescript/javascript | `next` en scope | `"next"` |
| typescript/javascript | `react` en scope | `"react"` |
| typescript/javascript | `vue` en scope | `"vue"` |
| python | keywords: django | `"django"` |
| python | keywords: fastapi | `"fastapi"` |
| python | keywords: flask | `"flask"` |
| other | — | `""` |

**extractKeywords:** tokeniza por espacios, filtra stopwords básicos (a, the, de, el, la, etc.), lowercased, deduplicated, max 10.

### verify.go (nuevo — `internal/skilladvisor/verify.go`)

```go
var KnownPublishers = []string{
    "anthropic", "nvidia", "microsoft", "google", "meta",
    "amazon", "openai", "hashicorp", "docker", "netlify",
    "vercel", "opencode-community",
}

func VerifyPublisher(publisher string) bool {
    lower := strings.ToLower(publisher)
    for _, p := range KnownPublishers {
        if lower == p { return true }
    }
    return false
}
```

### score.go (modificado)

Añadir `ValidateAndScore` que reemplaza el scoring plano de `Recommend`:

| Layer | Check | Effect |
|-------|-------|--------|
| 1 | `skill.SocketAlerts > 0` | Hard block: `ValidationError{HardBlocked: true}`, exclude from results |
| 2 | `!VerifyPublisher(skill.Publisher)` | `VerifiedBonus = 0` (override, même si `skill.Verified=true`) |
| 3 | `query.Language != "" && !EqualFold(skill.Language, query.Language)` | Penalty: score -= 10 |
| 4 | `query.Framework != "" && !EqualFold(skill.Framework, query.Framework)` | Penalty: score -= 20 |
| 5 | `query.ProjectType != "" && !EqualFold(skill.ProjectType, query.ProjectType)` | Penalty: score -= 30 |
| 6 | `ScoreSkillWeighted(skill, query)` | Base score |

```go
func ValidateAndScore(skills []SkillEntry, query DiscoveryQuery) []ValidatedSkill {
    var results []ValidatedSkill
    for _, skill := range skills {
        // Layer 1: hard block
        if skill.SocketAlerts > 0 {
            results = append(results, ValidatedSkill{
                Skill: skill,
                ValidationError: &ValidationError{
                    HardBlocked: true,
                    Reason: fmt.Sprintf("socket_alerts=%d", skill.SocketAlerts),
                },
                Rejected:     true,
                RejectReason: fmt.Sprintf("socket_alerts=%d", skill.SocketAlerts),
            })
            continue
        }

        // Layer 2: publisher check
        sq := SkillQuery{
            Language:    query.Language,
            Framework:   query.Framework,
            ProjectType: query.ProjectType,
        }
        sr := ScoreSkillWeighted(skill, sq)

        if !VerifyPublisher(skill.Publisher) {
            sr.VerifiedBonus = 0
            sr.TotalScore -= WeightVerifiedBonus
        }

        // Layers 3-5: cumulative penalties (applied by ScoreSkillWeighted via SkillQuery)
        // Layer 6: base score from ScoreSkillWeighted (already done)

        results = append(results, ValidatedSkill{
            Skill: skill,
            Score: sr,
        })
    }
    return results
}
```

**Nota:** Las capas 3-5 son inherentemente manejadas por `ScoreSkillWeighted` — si el query tiene Language/Framework/ProjectType y el skill no matchea, esos componentes simplemente son 0. El penalty adicional de publisher (capa 2) se aplica como substracción directa al TotalScore.

### discover.go (modificado — deprecado, sin cambios HTTP)

`discover.go` **NO se modifica para hacer llamadas HTTP**. Solo se deprecata el client y se mantiene la infraestructura de cache:

| Component | Action | Detail |
|-----------|--------|--------|
| `DiscoverCache` + `cacheEntry` | **Mantener** | Cache sigue siendo útil para deduplicar calls del pipeline |
| `DiscoverClient` struct | **Deprecar** | Añadir comentario `// Deprecated: use execPythonDiscover instead. Kept for cache infra only.` |
| `fetchFromAPI` | **No eliminar** | Se mantiene por si en el futuro se restaura HTTP directo |
| `Discover(query string)` | **Deprecar** | `// Deprecated: use DiscoverAndRank with DiscoveryQuery instead.` |
| `DiscoverWithQuery` | **NO crear** | La discovery ahora la hace Python, no Go HTTP |

```go
// DiscoverClient is DEPRECATED. Discovery is now handled by scripts/discover.py.
// Kept for DiscoverCache infrastructure — the pipeline.go execPythonDiscover function
// caches results using DiscoverCache directly, bypassing this client.
type DiscoverClient struct { /* unchanged */ }
```

### scripts/discover.py (nuevo — `internal/scaffold/templates/go-project/scripts/discover.py`)

Script Python para discovery de skills vía skills.sh API. Sigue el patrón de explorer.py/test-runner.py/linter.py: argparse + JSON stdout.

```python
#!/usr/bin/env python3
"""Discover skills from skills.sh API and return structured JSON."""
import argparse, json, os, sys, urllib.request, urllib.error, urllib.parse

SKILLS_API = os.getenv("SKILLS_API_URL", "https://skills.sh/api/search")


def discover(lang, framework, project_type, keywords):
    """Query the skills.sh API and return a list of skill dicts."""
    query = " ".join(filter(None, [lang, framework, project_type] + keywords))
    url = f"{SKILLS_API}?q={urllib.parse.quote(query)}"

    try:
        with urllib.request.urlopen(url, timeout=30) as resp:
            data = resp.read().decode()
            return json.loads(data)
    except urllib.error.URLError as e:
        print(json.dumps({"error": str(e), "skills": []}), file=sys.stderr)
        return []
    except json.JSONDecodeError as e:
        print(json.dumps({"error": f"invalid JSON from API: {e}", "skills": []}), file=sys.stderr)
        return []


def main():
    p = argparse.ArgumentParser(description="Discover skills from skills.sh API")
    p.add_argument("--lang", default="", help="Language filter (e.g. go, python)")
    p.add_argument("--framework", default="", help="Framework filter (e.g. cobra, react)")
    p.add_argument("--project-type", default="", help="Project type (e.g. cli, backend)")
    p.add_argument("--keywords", default="", help="Comma-separated keywords")
    args = p.parse_args()

    keywords = [k.strip() for k in args.keywords.split(",") if k.strip()]
    skills = discover(args.lang, args.framework, args.project_type, keywords)
    print(json.dumps(skills))


if __name__ == "__main__":
    main()
```

**Protocolo de salida**: stdout = JSON array de skills (misma estructura que `SkillEntry`), stderr = errores. Go parsea stdout y mapea errores de stderr a Go errors.

### scripts.go (modificado — `internal/scaffold/scripts.go`)

Añadir `discover.py` al embed.FS:

```go
//go:embed templates/go-project/scripts/*
var scriptsFS embed.FS

// ReadScript reads a script from the embedded scripts filesystem by name.
// Valid names: "explorer.py", "test-runner.py", "linter.py", "discover.py".
func ReadScript(name string) ([]byte, error) {
    data, err := scriptsFS.ReadFile("templates/go-project/scripts/" + name)
    if err != nil {
        return nil, fmt.Errorf("script %q not found: %w", name, err)
    }
    return data, nil
}
```

El glob `templates/go-project/scripts/*` ya captura cualquier archivo nuevo en ese directorio — no necesita cambio en la directiva `//go:embed`. Solo el archivo nuevo `discover.py` en ese path es suficiente.

### scaffold.go (modificado — `internal/scaffold/scaffold.go`)

Añadir `discover.py` a `scriptEntries`:

```go
scriptEntries := []struct{ embedPath, outPath string }{
    {"templates/go-project/scripts/explorer.py", "scripts/explorer.py"},
    {"templates/go-project/scripts/test-runner.py", "scripts/test-runner.py"},
    {"templates/go-project/scripts/linter.py", "scripts/linter.py"},
    {"templates/go-project/scripts/discover.py", "scripts/discover.py"},  // NUEVO
}
```

### registry.go (modificado)

`MergeAndRank` reemplaza `Recommend` como la función principal de merge+scoring:

```go
func MergeAndRank(apiSkills []SkillEntry, localSkills []SkillEntry, query DiscoveryQuery, n int) []ValidatedSkill {
    // Merge: local wins on duplicate names
    merged := make(map[string]SkillEntry)
    for _, s := range apiSkills {
        merged[s.Name] = s
    }
    for _, s := range localSkills {
        merged[s.Name] = s // sobreescribe API
    }

    var all []SkillEntry
    for _, s := range merged {
        all = append(all, s)
    }

    // Validate + Score
    validated := ValidateAndScore(all, query)

    // Sort by Normalized descending, exclude hard-blocked
    sort.SliceStable(validated, func(i, j int) bool {
        if validated[i].Rejected != validated[j].Rejected {
            return !validated[i].Rejected // non-rejected first
        }
        return validated[i].Score.Normalized() > validated[j].Score.Normalized()
    })

    if n > 0 && len(validated) > n {
        validated = validated[:n]
    }

    return validated
}

// RecommendFromHandoff se deprecata — wrapper para backward compat
func RecommendFromHandoff(language, projectType string, n int) ([]ScoreResult, error) {
    var r Registry
    if err := r.LoadDefaults(); err != nil {
        return nil, fmt.Errorf("skilladvisor: load defaults: %w", err)
    }
    query := SkillQuery{Language: language, ProjectType: projectType}
    return r.Recommend(query, n) // mantiene firma original
}
```

### pipeline.go (nuevo — `internal/skilladvisor/pipeline.go`)

Entry point unificado. Ejecuta `discover.py` como subprocess Python en lugar de HTTP:

```go
package skilladvisor

import (
    "encoding/json"
    "fmt"
    "log/slog"
    "os/exec"
    "path/filepath"
    "sort"
    "strings"

    "github.com/secko/zyrocli/internal/handoff"
)

// discoverScript is the embedded path to the Python discovery script.
const discoverScript = "scripts/discover.py"

// execPythonDiscover runs scripts/discover.py with the query parameters
// and parses the JSON output into SkillEntry slice.
func execPythonDiscover(query DiscoveryQuery) ([]SkillEntry, error) {
    args := []string{discoverScript}
    if query.Language != "" {
        args = append(args, "--lang", query.Language)
    }
    if query.Framework != "" {
        args = append(args, "--framework", query.Framework)
    }
    if query.ProjectType != "" {
        args = append(args, "--project-type", query.ProjectType)
    }
    if len(query.Keywords) > 0 {
        args = append(args, "--keywords", strings.Join(query.Keywords, ","))
    }

    cmd := exec.Command("python3", args...)
    output, err := cmd.Output()
    if err != nil {
        // Check if Python itself is not found
        if execErr, ok := err.(*exec.Error); ok && execErr.Err.Error() == "exec: \"python3\": executable file not found in $PATH" {
            return nil, fmt.Errorf("python3 not found in PATH: %w", err)
        }
        return nil, fmt.Errorf("discover.py failed: %w", err)
    }

    var skills []SkillEntry
    if err := json.Unmarshal(output, &skills); err != nil {
        return nil, fmt.Errorf("discover.py invalid JSON: %w", err)
    }
    return skills, nil
}

// DiscoverAndRank is the unified entry point for skill discovery + validation + ranking.
// It runs discover.py (Python) + local registry concurrently, merges results,
// validates through 6 layers, and returns top-N.
func DiscoverAndRank(payload *handoff.Payload, n int) ([]ValidatedSkill, error) {
    query := BuildDiscoveryQuery(payload)

    type discoverResult struct {
        skills []SkillEntry
        err    error
    }

    // Concurrent: Python discovery + local registry
    apiCh := make(chan discoverResult, 1)
    localCh := make(chan discoverResult, 1)

    go func() {
        skills, err := execPythonDiscover(query)
        if err != nil {
            slog.Warn("discover.py unavailable, using local registry", "error", err)
            apiCh <- discoverResult{nil, err}
            return
        }
        apiCh <- discoverResult{skills, nil}
    }()

    go func() {
        var r Registry
        if err := r.LoadDefaults(); err != nil {
            localCh <- discoverResult{nil, err}
            return
        }
        var skills []SkillEntry
        for _, s := range r.Skills {
            skills = append(skills, s)
        }
        localCh <- discoverResult{skills, nil}
    }()

    apiRes := <-apiCh
    localRes := <-localCh

    // Graceful degradation: si discover.py falla, usa solo local
    var apiSkills []SkillEntry
    if apiRes.err == nil {
        apiSkills = apiRes.skills
    }

    var localSkills []SkillEntry
    if localRes.err == nil {
        localSkills = localRes.skills
    }

    if len(apiSkills) == 0 && len(localSkills) == 0 {
        return nil, fmt.Errorf("skilladvisor: no skills available (discover.py failed, local empty)")
    }

    return MergeAndRank(apiSkills, localSkills, query, n), nil
}
```

**Resolución de path al script**: `execPythonDiscover` usa `scripts/discover.py` relativo al cwd. Si ZyroCLI se ejecuta desde la raíz del proyecto scaffolded, el path funciona porque el script se despliega con scaffold. Si se ejecuta desde otro directorio, se puede usar `filepath.Join(exeDir, discoverScript)` o `os.Executable()` para resolver la ubicación real.

**Fallback a Python 3 ausente**: Si `python3` no está en PATH, `exec.Command` retorna error con `"executable file not found"`. El pipeline lo captura y degrada a solo local registry con warning.

### scheduler/phase.go (modificado)

Añadir campo `Skills` al struct `Result`:

```go
type Result struct {
    Phase   Phase
    Status  Status
    Summary string
    Error   error
    Skills  []skilladvisor.ValidatedSkill  // propagate from F1
}
```

Requiere import de `skilladvisor`. El import path es `github.com/secko/zyrocli/internal/skilladvisor`.

### scheduler/macro_runner.go (modificado)

`F1AgentFunc` cambia de `Discover()` plano a `DiscoverAndRank()`:

```go
func F1AgentFunc(ctx context.Context, cfg *Config) (*Result, error) {
    payload, err := handoff.Parse("handoff.yaml")
    if err != nil {
        return &Result{Phase: PhaseF1, Status: StatusFail, Summary: fmt.Sprintf("parse failed: %v", err)}, nil
    }

    validated, err := skilladvisor.DiscoverAndRank(payload, 0) // 0 = all
    if err != nil {
        return &Result{Phase: PhaseF1, Status: StatusFail, Summary: fmt.Sprintf("discover failed: %v", err)}, nil
    }

    summary := fmt.Sprintf("Project: %s | Language: %s | Validated skills: %d",
        payload.Project.Name, payload.Project.Language, len(validated))

    return &Result{Phase: PhaseF1, Status: StatusSuccess, Summary: summary, Skills: validated}, nil
}
```

### cmd/zyrocli/run.go (modificado)

Migrar de `RecommendFromHandoff` a `DiscoverAndRank`:

```go
// Reemplazar:
//   skills, err := skilladvisor.RecommendFromHandoff(payload.Project.Language, projectType, 8)
// Por:
validated, err := skilladvisor.DiscoverAndRank(payload, 8)
if err != nil {
    cmd.PrintErrf("⚠ skill advisor warning (non-fatal): %v\n", err)
} else {
    recommendedSkills = validated // tipo cambia de []ScoreResult a []ValidatedSkill
    cmd.Printf("  Skills recomendados para este proyecto: %d\n", len(validated))
    for _, s := range validated {
        if s.Rejected {
            cmd.Printf("    ✗ %s — REJECTED: %s\n", s.Skill.Name, s.RejectReason)
        } else {
            cmd.Printf("    • %s — %s (score: %.2f)\n", s.Skill.Name, s.Skill.Description, s.Score.Normalized())
        }
    }
}
```

**Nota:** El tipo de `recommendedSkills` cambia de `[]skilladvisor.ScoreResult` a `[]skilladvisor.ValidatedSkill`. Esto afecta `scaffold.Config.RecommendedSkills` — hay que verificar si `scaffold.Config` referencia `ScoreResult` y adaptarlo.

## Data Flow

```
                    ┌──────────────────┐
                    │  handoff.yaml    │
                    └────────┬─────────┘
                             │
                    ┌────────▼─────────┐
                    │ BuildDiscovery   │
                    │ Query            │
                    └────────┬─────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
     ┌────────▼─────────┐        ┌──────────▼──────────┐
     │ exec python3     │        │ Registry.Load        │
     │ discover.py      │        │ Defaults (local)     │
     │ (JSON stdout)    │        │                      │
     └────────┬─────────┘        └──────────┬──────────┘
              │                             │
              └──────────┬──────────────────┘
                         │
                ┌────────▼─────────┐
                │ MergeAndRank     │
                │ (local wins)     │
                └────────┬─────────┘
                         │
                ┌────────▼─────────┐
                │ ValidateAndScore │
                │ 6 layers         │
                └────────┬─────────┘
                         │
                ┌────────▼─────────┐
                │ Sort + top-N     │
                └────────┬─────────┘
                         │
              ┌──────────┴──────────┐
              │                     │
     ┌────────▼─────────┐  ┌───────▼──────────┐
     │ run.go           │  │ Scheduler F1      │
     │ display loop     │  │ Result.Skills     │
     └──────────────────┘  └──────────────────┘
```

## Error Handling

| Scenario | Behavior |
|----------|----------|
| discover.py unreachable (Python not in PATH) | `log.Warn`, usa solo local registry, no error returned |
| discover.py HTTP timeout / API down | Python prints error to stderr, returns empty array; Go logs warning, uses local |
| discover.py malformed JSON output | Go returns parse error, falls back to local only |
| Registry local empty + discover.py falla | Retorna error `"no skills available"` |
| SocketAlerts > 0 | `ValidatedSkill.Rejected=true`, `ValidationError.HardBlocked=true`, skill excluido de top-N |
| Publisher desconocido | `VerifiedBonus=0`, penalización -50, skill incluido pero rankeado bajo |
| handoff.yaml malformado | Error propagado al caller (run.go o scheduler) |
| Cache key collision | Imposible: key serializa todos los campos de DiscoveryQuery |

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `BuildDiscoveryQuery`, `detectFramework`, `extractKeywords` | Tabla de tests con payloads mock |
| Unit | `VerifyPublisher` | Known publishers + unknown + case-insensitive |
| Unit | `ValidateAndScore` | Hard block, publisher penalty, full valid, mixed |
| Unit | `MergeAndRank` | Duplicate resolution (local wins), merge vacío |
| Unit | `execPythonDiscover` | Mock `exec.Command` via `CommandController` o build tag |
| Unit | `discover.py` | Mock HTTP server en Python (httpretty) o test con urllib.mock |
| Integration | `DiscoverAndRank` end-to-end | Mock HTTP server para API, registry con defaults |
| Integration | F1AgentFunc propagation | Mock handoff.yaml, verify `Result.Skills` populated |
| Integration | run.go display loop | Verify output format con `ValidatedSkill` |

**Tests nuevos en `skilladvisor_test.go`:**
- `TestValidateAndScore_HardBlock`: skill con SocketAlerts > 0 → Rejected=true
- `TestValidateAndScore_UnknownPublisher`: publisher "x" → VerifiedBonus=0
- `TestValidateAndScore_FullValid`: todo pass → score > 0
- `TestMergeAndRank_DuplicateLocalWins`: API + local con mismo name → local conservado
- `TestMergeAndRank_APIFailure`: API retorna error → solo local
- `TestDiscoverAndRank_GracefulDegradation`: mock API down → returned local results
- `TestExecPythonDiscover_PythonNotFound`: python3 not in PATH → error with suggestion
- `TestExecPythonDiscover_MalformedJSON`: discover.py returns garbage → parse error

## Migration / Rollout

No migration de datos requerida. Cambio es 100% código.

**Backward compatibility:**
- `RecommendFromHandoff` se mantiene como wrapper → no breaking change para callers existentes
- `Discover(query string)` se mantiene como convenience function (deprecated)
- `ScoreResult` y `SkillQuery` se mantienen sin cambios
- `DiscoverClient` se mantiene pero se deprecata — no se elimina para no romper imports

**Scaffold config:** `scaffold.Config.RecommendedSkills` es `[]skilladvisor.ScoreResult` (ver scaffold.go línea 24). Necesita cambio a `[]skilladvisor.ValidatedSkill` o adapter function. Propongo adapter:

```go
// En scaffold.go, cambiar:
RecommendedSkills []skilladvisor.ScoreResult
// A:
RecommendedSkills []skilladvisor.ValidatedSkill
```

Esto es breaking interno pero no hay callers externos — solo run.go y init.go usan este campo.

## Technical Debt

1. **Wasted discover.go code**: `DiscoverClient` struct, `fetchFromAPI`, HTTP client — todo legacy. `DiscoverCache` y `cacheEntry` se mantienen como infraestructura reutilizable pero `DiscoverClient` pasa a deprecado. `fetchFromAPI` queda como dead code que podría eliminarse en futura limpieza.

2. **Python 3 runtime dependency**: ZyroCLI ahora requiere `python3` en PATH para discovery. Si no está, degrada gracefulmente a solo registry local. Futuras operaciones (explorer, test-runner, linter) ya dependen de Python 3 — discover.py no agrega dependencia nueva, solo consolida la existente.

3. **Cross-language error boundary**: Errores de Python (exceptions, malformed JSON, timeout) deben mapearse a Go errors. Protocolo: stdout = JSON con skills, stderr = errores. Los errores de stderr se loguean como warnings. Go distingue: (a) Python not found, (b) Python crashed, (c) bad JSON output.

4. **Script location**: `scripts/discover.py` debe ejecutarse desde el directorio del proyecto scaffolded. Si ZyroCLI se ejecuta desde otro cwd, falla. Mitigado por `os.Executable()` + `filepath.Dir()` para resolver la raíz del binario y construir path absoluto al script.

5. **Subprocess overhead**: Cada `python3 discover.py` es un subprocess (~50-100ms startup). Mitigado por cache en `DiscoverCache`. Para uso frecuente, considerar warm start o keep-alive del proceso Python.

6. **Testing complexity**: Tests de `discover.py` requieren mock HTTP en Python (httpretty) o mock del subprocess en Go (httptest.Server + exec.Command mock). Doble cobertura necesaria.

7. **Doble parsing**: JSON sale de Python, se parsea en Go — dos schemas que mantener sincronizados. Mitigado por tipos compartidos: `SkillEntry` struct en Go, dict en Python con las mismas keys. Sin schema validation explícita entre lenguajes.

8. **Embedding**: `scripts/discover.py` se agrega automáticamente a `scriptsFS embed.FS` por el glob `templates/go-project/scripts/*`. Se agrega a `scriptEntries` en `scaffold.go` para desplegarlo al scaffoldear.

## Open Questions

- [ ] **scaffold.Config.RecommendedSkills type**: Es `[]skilladvisor.ScoreResult` (verificado en scaffold.go línea 24). Necesita cambio a `[]skilladvisor.ValidatedRisk` o adapter. Propongo cambio directo ya que solo run.go y init.go son callers.
- [ ] **detectFramework — repository introspection**: El mapping dice "spf13/cobra si en repo" — ¿parsear go.mod? O simplemente keywords en scope/features? Propongo keywords en MVP.Scope/Features por simplicidad.
- [ ] **Script resolution en runtime**: ¿Usar `os.Executable()` para path absoluto, o asumir cwd? Propongo executable-based para robustez.
