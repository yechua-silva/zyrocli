# `zyro onboard` — Tareas Atómicas

> **Fecha:** 17 Junio 2026
> **Versión:** 1.0
> **Spec:** [spec.md](./spec.md)
> **Diseño:** [design.md](./design.md)

---

## Resumen de Tareas

| ID | Nombre | Dependencias | LOC | Archivos |
|----|--------|-------------|-----|----------|
| **T1** | Scanner types | — | ~50 | `internal/scanner/types.go` |
| **T2** | Scanner engine | T1 | ~130 | `internal/scanner/scanner.go` |
| **T3** | CLI skeleton + flags | — | ~50 | `cmd/zyrocli/onboard.go`, `cmd/zyrocli/main.go` |
| **T4** | HelixDB sync | T2 | ~80 | `cmd/zyrocli/onboard.go` |
| **T5** | OpenCode launch | T3 | ~40 | `cmd/zyrocli/onboard.go` |
| **T6** | AGENT.md update | — | ~20 | `AGENT.md` |
| **T7** | Unit tests | T2 | ~80 | `internal/scanner/scanner_test.go` |

**Total estimado:** ~450 LOC, **7 tareas atómicas**, **sin dependencias externas nuevas**.

---

## T1: Scanner types

### Descripción

Crear `internal/scanner/types.go` con los tipos base del sistema de escaneo de proyectos: `Language` (enum), `FileInfo` (struct), `ProjectInfo` (struct). Estos tipos son compartidos por el engine de scanner y por el CLI que consumirá los resultados.

### Archivos afectados

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `internal/scanner/types.go` | **Nuevo** | ~50 |

### Especificación

```go
package scanner

import "time"

// Language representa un lenguaje de programación.
type Language string

const (
    LanguageGo         Language = "Go"
    LanguageTypeScript Language = "TypeScript"
    LanguageJavaScript Language = "JavaScript"
    LanguageRust       Language = "Rust"
    LanguagePython     Language = "Python"
    LanguageRuby       Language = "Ruby"
    LanguagePHP        Language = "PHP"
    LanguageC          Language = "C"
    LanguageCpp        Language = "C++"
    LanguageCSharp     Language = "C#"
    LanguageJava       Language = "Java"
    LanguageElixir     Language = "Elixir"
    LanguageSwift      Language = "Swift"
    LanguageKotlin     Language = "Kotlin"
    LanguageUnknown    Language = "Unknown"
)

// FileInfo contiene metadatos de un archivo escaneado.
type FileInfo struct {
    Path     string   `json:"path"`
    Name     string   `json:"name"`
    Ext      string   `json:"ext"`
    Size     int64    `json:"size"`
    Hash     string   `json:"hash"`
    Language Language `json:"language"`
}

// ProjectInfo contiene el resultado completo del escaneo.
type ProjectInfo struct {
    Root      string     `json:"root"`
    Name      string     `json:"name"`
    Language  Language   `json:"language"`
    Files     []FileInfo `json:"files"`
    FileCount int        `json:"file_count"`
    TotalSize int64      `json:"total_size"`
    ScannedAt time.Time  `json:"scanned_at"`
}
```

### Criterios de aceptación

- [ ] `Language` es un `string` type (fácil serialización JSON)
- [ ] `FileInfo` incluye todos los campos necesarios para HelixDB sync
- [ ] `ProjectInfo` incluye metadatos completos del proyecto
- [ ] Todos los structs tienen tags JSON
- [ ] No hay dependencias del package scanner hacia otros packages internos

---

## T2: Scanner engine

### Descripción

Implementar `internal/scanner/scanner.go` con `ProjectScanner` struct, el método `Scan(root string) (*ProjectInfo, error)` que realiza el walk del árbol de directorios, aplica ignore patterns, detecta el lenguaje, computa hashes SHA256, y construye el `ProjectInfo`.

### Archivos afectados

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `internal/scanner/scanner.go` | **Nuevo** | ~130 |

### Código a implementar

```go
package scanner

import (
    "crypto/sha256"
    "encoding/hex"
    "io"
    "os"
    "path/filepath"
    "strings"
    "time"
)

var DefaultIgnorePatterns = []string{
    ".git", ".hg", ".svn",
    "node_modules", "vendor", "dist", "build", "target",
    ".venv", "__pycache__",
    ".idea", ".vscode",
    ".zyro", ".helix",
}

type ProjectScanner struct {
    ignorePatterns []string
    maxFiles       int
}

func NewScanner() *ProjectScanner {
    return &ProjectScanner{
        ignorePatterns: DefaultIgnorePatterns,
        maxFiles:       10000,
    }
}

func (s *ProjectScanner) WithIgnorePatterns(p []string) *ProjectScanner {
    s.ignorePatterns = p; return s
}

func (s *ProjectScanner) WithMaxFiles(n int) *ProjectScanner {
    s.maxFiles = n; return s
}

func (s *ProjectScanner) Scan(root string) (*ProjectInfo, error)

// detectProjectLanguage — mira go.mod, package.json, Cargo.toml, etc.
func (s *ProjectScanner) detectProjectLanguage(root string, files []FileInfo) Language

// hashFile — SHA256 del contenido
func hashFile(path string) (string, error)
```

### Detalles de implementación

**Scan() algoritmo:**
1. Resolver path a absoluto con `filepath.Abs`
2. Construir `ignoreSet` (`map[string]bool`) desde `ignorePatterns`
3. `filepath.WalkDir`:
   - Si es directorio y está en ignoreSet → `filepath.SkipDir`
   - Si es archivo:
     - Verificar que ningún segmento del path relativo esté en ignoreSet
     - Si `maxFiles > 0` y ya se alcanzó → `filepath.SkipAll`
     - Obtener `d.Info()` para tamaño
     - `hashFile(path)` para SHA256
     - Detectar lenguaje por extensión
     - Append a `info.Files`
4. Al finalizar, ejecutar `detectProjectLanguage` para determinar lenguaje principal

**detectProjectLanguage() algoritmo:**
1. Leer `os.ReadDir(root)` para obtener nombres de archivos raíz
2. Check en orden: go.mod → Go, Cargo.toml → Rust, package.json+tsconfig → TS, solo package.json → JS, pyproject.toml/requirements.txt → Python, Gemfile → Ruby, composer.json → PHP, CMakeLists.txt → C/C++, mix.exs → Elixir
3. Fallback: estadística de extensiones de archivos escaneados

**languageFromExt() mapa:**
```go
var extLangMap = map[string]Language{
    ".go": LanguageGo, ".ts": LanguageTypeScript, ".tsx": LanguageTypeScript,
    ".js": LanguageJavaScript, ".jsx": LanguageJavaScript, ".mjs": LanguageJavaScript,
    ".rs": LanguageRust, ".py": LanguagePython, ".rb": LanguageRuby,
    ".php": LanguagePHP, ".c": LanguageC, ".h": LanguageC,
    ".cpp": LanguageCpp, ".cc": LanguageCpp, ".cxx": LanguageCpp,
    ".hpp": LanguageCpp, ".cs": LanguageCSharp, ".java": LanguageJava,
    ".kt": LanguageKotlin, ".kts": LanguageKotlin,
    ".ex": LanguageElixir, ".exs": LanguageElixir,
    ".swift": LanguageSwift,
}
```

### Criterios de aceptación

- [ ] WalkDir ignora correctamente `.git`, `node_modules`, `vendor`, etc.
- [ ] WalkDir no ignora archivos con nombres similares (ej: `notes.md` no se ignora)
- [ ] SHA256 hash es correcto y reproducible
- [ ] Detección de lenguaje por archivo de proyecto funciona para Go, TS, JS, Rust, Python
- [ ] Fallback por extensión funciona cuando no hay archivo de configuración
- [ ] Límite de 10,000 archivos se respeta y no causa panic
- [ ] Directorios sin permisos no detienen el scan (skip)

---

## T3: CLI skeleton + flags

### Descripción

Crear `cmd/zyrocli/onboard.go` con el comando Cobra `zyro onboard [path]`. Incluye validación del path, resolución a absoluto, check de inicialización previa, flags (`--dry-run`, `--force`, `--no-opencode`, `--name`, `--desc`), y registro del comando en `main.go`. Inicialmente hace solo las validaciones y llama al scanner (T2).

### Archivos afectados

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `cmd/zyrocli/onboard.go` | **Nuevo** | ~130 |
| `cmd/zyrocli/main.go` | Agregar `onboardCmd` al root | ~3 |

### Estructura del comando

```go
package main

import "github.com/spf13/cobra"

var onboardCmd = &cobra.Command{
    Use:   "onboard [path]",
    Short: "Register an existing project in ZyroCLI ecosystem",
    Long:  `...`,
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        // Ver timeline completo en design.md §7.1
    },
}

func init() {
    rootCmd.AddCommand(onboardCmd)
    // flags: --dry-run, --force, --no-opencode, --name, --desc
}
```

**En `main.go`:** solo agregar `rootCmd.AddCommand(onboardCmd)` en un init() o en el bloque de init().

### Criterios de aceptación

- [ ] `zyro onboard` funciona con y sin argumento path
- [ ] `zyro onboard /no/existe` muestra error claro
- [ ] `zyro onboard /path/to/file` (no directorio) muestra error claro
- [ ] `zyro onboard --dry-run .` muestra resumen sin escribir nada
- [ ] `zyro onboard --force .` re-ejecuta aunque ya esté inicializado
- [ ] `zyro onboard --help` muestra flags correctamente
- [ ] Comando aparece en `zyrocli --help`

---

## T4: HelixDB sync

### Descripción

Implementar la función `syncToHelixDB()` en `cmd/zyrocli/onboard.go` que crea el Project node y los CodeNodes en HelixDB a partir del `ProjectInfo` del scanner. Maneja graceful degradation si HelixDB no está disponible.

### Archivos afectados

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `cmd/zyrocli/onboard.go` | Agregar función `syncToHelixDB` | ~80 |

### Función a implementar

```go
func syncToHelixDB(ctx context.Context, client *helix.Client, info *scanner.ProjectInfo) (int64, error) {
    // 1. Crear Project node con propiedades
    //    name, language, path, status="onboarded", current_phase="phase0",
    //    onboarded_at, file_count
    //
    // 2. Loop sobre info.Files (máx 500)
    //    - client.UpsertCodeNode(projectID, path, name, "", hash, nil)
    //    - Ignorar errores individuales, continuar con los demás
    //
    // 3. Retornar projectID
}
```

### Criterios de aceptación

- [ ] Crea Project node correctamente con todas las propiedades
- [ ] Crea CodeNodes para los archivos escaneados (hasta 500)
- [ ] Crea edge HAS_CODENODE entre Project y cada CodeNode
- [ ] Si HelixDB falla, retorna error manejable (no panic)
- [ ] Si un CodeNode falla, los demás continúan

---

## T5: OpenCode launch

### Descripción

Implementar la escritura de `.zyro/task.yaml` con contexto de onboarding y el lanzamiento de OpenCode. El archivo `.zyro/task.yaml` permite que el agente de OpenCode sepa que está en modo onboarding (proyecto existente) en vez de init.

### Archivos afectados

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `cmd/zyrocli/onboard.go` | Agregar funciones `writeOnboardTaskFile` y `launchOpenCode` | ~40 |

### Funciones a implementar

```go
func writeOnboardTaskFile(projectDir string, info *scanner.ProjectInfo, projectID int64) error {
    // Escribe .zyro/task.yaml con:
    //   phase: "F0"
    //   agent: "zyro-onboarding"
    //   is_onboard: true
    //   project_id: <id>
    //   project_language: "<lang>"
    //   file_count: <n>
    //   scanned_at: "<timestamp>"
}

func launchOpenCode(projectDir string) error {
    // exec.LookPath("opencode")
    // exec.Command("opencode", projectDir).Run()
}
```

### Criterios de aceptación

- [ ] `.zyro/task.yaml` se escribe correctamente con formato YAML
- [ ] Contiene `is_onboard: true` (clave para el agente)
- [ ] Respeta `--no-opencode` flag
- [ ] Si opencode no está en PATH, muestra advertencia y no lanza error
- [ ] OpenCode se abre en el directorio correcto

---

## T6: AGENT.md update

### Descripción

Actualizar `/home/secko/Projects/ZyroAgentCLI/AGENT.md` para incluir el nuevo comando `zyro onboard`, el flujo de onboarding, y la diferencia con `zyro init`.

### Archivos afectados

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `AGENT.md` | Agregar sección de onboarding | ~20 |

### Cambios

1. En la sección "Estructura del proyecto", agregar `├── onboard.go` a `cmd/zyrocli/`
2. En la sección "Comandos CLI", agregar `- zyro onboard [path] — registra proyecto existente en el ecosistema`
3. Agregar sub-sección "Onboarding vs Init" que explique las diferencias:
   - Init: requiere handoff.yaml, crea proyecto nuevo
   - Onboard: no requiere handoff, escanea proyecto existente
   - Ambos crean Project + CodeNodes en HelixDB
   - Ambos lanzan OpenCode
   - F0 difiere: init busca en internet, onboard analiza código local

### Criterios de aceptación

- [ ] AGENT.md menciona `zyro onboard` como comando CLI
- [ ] Explica cuándo usar onboard vs init
- [ ] Menciona el flujo F0-onboarding

---

## T7: Unit tests

### Descripción

Tests unitarios para el scanner. Incluye tests de detección de lenguaje, ignore patterns, hashing, y construcción de `ProjectInfo`. Usa `testing` standard library + `testify` (ya en go.mod).

### Archivos afectados

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `internal/scanner/scanner_test.go` | **Nuevo** | ~80 |

### Test cases

```go
func TestDetectLanguage_Go(t *testing.T) {
    // Crear temp dir con go.mod → debe detectar Go
}

func TestDetectLanguage_TypeScript(t *testing.T) {
    // Crear temp dir con package.json + tsconfig.json → TypeScript
}

func TestDetectLanguage_JavaScript(t *testing.T) {
    // Crear temp dir con solo package.json → JavaScript
}

func TestDetectLanguage_Rust(t *testing.T) {
    // Crear temp dir con Cargo.toml → Rust
}

func TestDetectLanguage_Python(t *testing.T) {
    // Crear temp dir con pyproject.toml → Python
}

func TestDetectLanguage_Unknown(t *testing.T) {
    // Crear temp dir vacío → Unknown
}

func TestIgnorePatterns(t *testing.T) {
    // Crear temp dir con .git/, node_modules/, .venv/
    // Verificar que no aparecen en Files
}

func TestHashFile(t *testing.T) {
    // Crear archivo temporal, escribir contenido conocido
    // Verificar hash SHA256
}

func TestScanCounter(t *testing.T) {
    // Crear temp dir con archivos
    // Verificar FileCount correcto
}
```

### Criterios de aceptación

- [ ] `go test ./internal/scanner/ -v` pasa todos los tests
- [ ] Tests crean y limpian temp directories
- [ ] Tests de detección de lenguaje cubren al menos: Go, TS, JS, Rust, Python, Unknown
- [ ] Tests de ignore patterns verifican que .git y node_modules no aparecen
- [ ] Test de hash verifica que es reproducible (mismo contenido → mismo hash)
- [ ] Test de límite de archivos (con maxFiles pequeño)

---

## Dependencias entre tareas

```
T1 ──▶ T2 ──▶ T4 ──▶ T5
                  │
T3 ───────────────┘
                  │
T6 ───────────────┘ (puede empezar en paralelo con T1-T5)

T7 ─── depende de T2 (usa el scanner)
```

### Orden de implementación recomendado

1. **T1** — tipos base (sin dependencias)
2. **T3** — CLI skeleton (sin dependencias, al principio para poder testear el comando)
3. **T2** — scanner engine (depende de T1)
4. **T4** — HelixDB sync (depende de T2 + T3)
5. **T5** — OpenCode launch (depende de T3)
6. **T6** — AGENT.md (paralelo con T4-T5)
7. **T7** — tests (depende de T2)
