# `zyro onboard` — Diseño Técnico

> **Fecha:** 17 Junio 2026
> **Versión:** 1.0
> **Spec:** [spec.md](./spec.md)
> **Tareas:** [tasks.md](./tasks.md)

---

## 1. Arquitectura General

```
┌─────────────────────────────────────────────────────────────────────┐
│                        zyrocli binary                                │
│                                                                      │
│  cmd/zyrocli/main.go                                                 │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  rootCmd ── onboardCmd (onboard.go)                            │  │
│  │    │                                                            │  │
│  │    │ 1. Parse flags / args                                      │  │
│  │    │ 2. Resolve path → absolute                                 │  │
│  │    │ 3. Validate (existe, es dir, no ya onboard)                │  │
│  │    │                                                            │  │
│  │    ├──▶ internal/scanner/scanner.go                             │  │
│  │    │      ProjectScanner.Scan(path) → *ProjectInfo               │  │
│  │    │      • WalkDir con ignore patterns                          │  │
│  │    │      • DetectLanguage(root) → Language                      │  │
│  │    │      • HashFile(path) → string                             │  │
│  │    │                                                            │  │
│  │    ├──▶ internal/db/helix/client.go (HelixDB sync)              │  │
│  │    │      • CreateNode("Project", ...)                          │  │
│  │    │      • Loop: CreateNode("CodeNode", ...) + edge             │  │
│  │    │                                                            │  │
│  │    ├──▶ .zyro/task.yaml (write context)                         │  │
│  │    │                                                            │  │
│  │    └──▶ exec.Command("opencode", projectDir)                    │  │
│  │                                                                  │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  internal/scanner/                                                    │
│  ├── types.go      → ProjectInfo, FileInfo, Language                 │
│  ├── scanner.go    → ProjectScanner, Scan(), DetectLanguage()        │
│  └── scanner_test.go → tests                                         │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. Diagrama de Flujo Detallado

```
onboardCmd.RunE(args)
│
├─ 1. Determinar path (args[0] o ".")
│  └─ filepath.Abs(path)
│
├─ 2. Validar path
│  ├─ os.Stat(path) → error si no existe
│  ├─ es directorio? → error si no
│  └─ ReadState(path) → si existe y no --force → error "already onboarded"
│
├─ 3. Dry-run? → saltar a preview
│
├─ 4. Crear ProjectScanner
│  └─ scanner.NewScanner(ignorePatterns...)
│
├─ 5. Scan(path)
│  └─ info, err := scanner.Scan(path)
│     ├─ WalkDir recursivo
│     ├─ Aplicar ignore patterns
│     ├─ Detectar lenguaje (go.mod, package.json, etc.)
│     ├─ Por cada archivo relevante:
│     │  ├─ Computar path relativo
│     │  ├─ Detectar extensión
│     │  ├─ Computar hash SHA256
│     │  └─ Agregar a FileInfo slice
│     └─ Retornar *ProjectInfo
│
├─ 6. Mostrar resumen al usuario
│  └─ cmd.Printf("✓ Proyecto: %s (%s)\n", name, language)
│  └─ cmd.Printf("  Archivos: %d\n", len(files))
│
├─ 7. Dry-run? → return (no escribe nada)
│
├─ 8. Conectar a HelixDB
│  ├─ NewClient(ctx)
│  ├─ EnsureStarted(ctx) → warning si no disponible
│  └─ defer client.Close()
│
├─ 9. Crear Project node
│  ├─ client.CreateNode("Project", {
│  │    "name":        name,
│  │    "language":    language,
│  │    "path":        absPath,
│  │    "status":      "onboarded",
│  │    "current_phase": "phase0",
│  │    "onboarded_at": time.Now().UTC(),
│  │    "file_count":  len(files),
│  │  })
│  └─ projectID = result.ID
│
├─10. Crear CodeNodes (hasta 500)
│  ├─ for _, file := range files[:min(500, len(files))] {
│  │    nodeID, err = client.UpsertCodeNode(
│  │      projectID, file.Path, file.Name, "", file.Hash, nil,
│  │    )
│  │  }
│  └─ cmd.Printf("  CodeNodes: %d\n", created)
│
├─11. Escribir .zyro/state.json
│  └─ scaffold.WriteState(path, &State{
│       Initialized: true,
│       ProjectName: name,
│       TargetDir:   path,
│       Version:     "onboard",
│     })
│
├─12. Escribir .zyro/task.yaml
│  └─ os.WriteFile(path.Join(path, ".zyro", "task.yaml"), yaml, 0644)
│     content:
│       phase: "F0"
│       agent: "zyro-onboarding"
│       is_onboard: true
│       project_language: "Go"
│       file_count: 142
│
├─13. --no-opencode? → return
│
├─14. Launch OpenCode
│  └─ exec.Command("opencode", path).Run()
│
└─15. Done
```

---

## 3. ProjectScanner: Estructura y Métodos

### 3.1 Types (`internal/scanner/types.go`)

```go
package scanner

import "time"

// Language representa un lenguaje de programación detectado.
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
    Path     string   `json:"path"`     // relativo al root del proyecto
    Name     string   `json:"name"`     // filename sin path
    Ext      string   `json:"ext"`      // extensión (incluye punto, ej: ".go")
    Size     int64    `json:"size"`     // bytes
    Hash     string   `json:"hash"`     // SHA256 hex
    Language Language `json:"language"` // lenguaje derivado de la extensión
}

// ProjectInfo contiene el resultado completo del escaneo.
type ProjectInfo struct {
    Root        string     `json:"root"`         // ruta absoluta al proyecto
    Name        string     `json:"name"`         // nombre del directorio
    Language    Language   `json:"language"`     // lenguaje principal detectado
    Files       []FileInfo `json:"files"`        // archivos escaneados
    FileCount   int        `json:"file_count"`   // total de archivos
    TotalSize   int64      `json:"total_size"`   // suma de tamaños en bytes
    ScannedAt   time.Time  `json:"scanned_at"`   // timestamp del escaneo
}
```

### 3.2 Scanner Engine (`internal/scanner/scanner.go`)

```go
package scanner

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
    "time"
)

// DefaultIgnorePatterns son los directorios/archivos a excluir del escaneo.
var DefaultIgnorePatterns = []string{
    ".git", ".hg", ".svn",
    "node_modules", "vendor", "dist", "build", "target",
    ".venv", "__pycache__", ".env", ".env.local",
    ".idea", ".vscode",
    ".zyro", ".helix",
}

// ProjectScanner escanea proyectos existentes.
type ProjectScanner struct {
    ignorePatterns []string
    maxFiles       int // límite de archivos a escanear (0 = sin límite)
}

// NewScanner crea un ProjectScanner con los patrones de ignorado por defecto.
func NewScanner() *ProjectScanner {
    return &ProjectScanner{
        ignorePatterns: DefaultIgnorePatterns,
        maxFiles:       10000,
    }
}

// WithIgnorePatterns reemplaza los patrones de ignorado.
func (s *ProjectScanner) WithIgnorePatterns(patterns []string) *ProjectScanner {
    s.ignorePatterns = patterns
    return s
}

// WithMaxFiles establece el límite de archivos.
func (s *ProjectScanner) WithMaxFiles(n int) *ProjectScanner {
    s.maxFiles = n
    return s
}

// Scan ejecuta el escaneo completo del proyecto.
func (s *ProjectScanner) Scan(root string) (*ProjectInfo, error) {
    absRoot, err := filepath.Abs(root)
    if err != nil {
        return nil, fmt.Errorf("scanner: resolve path: %w", err)
    }

    info := &ProjectInfo{
        Root:      absRoot,
        Name:      filepath.Base(absRoot),
        ScannedAt: time.Now(),
    }

    ignoreSet := make(map[string]bool, len(s.ignorePatterns))
    for _, p := range s.ignorePatterns {
        ignoreSet[p] = true
    }

    err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
        if err != nil {
            return filepath.SkipDir // permisos, etc.
        }

        relPath, _ := filepath.Rel(absRoot, path)

        // Ignorar directorios completos
        if d.IsDir() {
            if ignoreSet[d.Name()] {
                return filepath.SkipDir
            }
            return nil
        }

        // Ignorar archivos en directorios ignorados (por si WalkDir no saltó)
        parts := strings.Split(relPath, string(filepath.Separator))
        for _, part := range parts {
            if ignoreSet[part] {
                return nil
            }
        }

        // Límite de archivos
        if s.maxFiles > 0 && len(info.Files) >= s.maxFiles {
            return filepath.SkipAll
        }

        // Obtener info del archivo
        fInfo, err := d.Info()
        if err != nil {
            return nil // skip
        }

        // Computar hash SHA256
        hash, err := hashFile(path)
        if err != nil {
            return nil // skip
        }

        ext := filepath.Ext(d.Name())
        fileLang := languageFromExt(ext, info.Language)

        info.Files = append(info.Files, FileInfo{
            Path:     relPath,
            Name:     d.Name(),
            Ext:      ext,
            Size:     fInfo.Size(),
            Hash:     hash,
            Language: fileLang,
        })
        info.TotalSize += fInfo.Size()

        return nil
    })

    if err != nil {
        return nil, fmt.Errorf("scanner: walk: %w", err)
    }

    info.FileCount = len(info.Files)

    // Detectar lenguaje principal (desde archivos de configuración del proyecto)
    info.Language = s.detectProjectLanguage(absRoot, info.Files)

    return info, nil
}

// detectProjectLanguage detecta el lenguaje principal del proyecto.
// Primero revisa archivos de configuración (go.mod, package.json, etc.),
// luego usa el lenguaje más frecuente entre los archivos escaneados.
func (s *ProjectScanner) detectProjectLanguage(root string, files []FileInfo) Language {
    // 1. Revisar archivos de configuración del proyecto
    entries, _ := os.ReadDir(root)
    entryNames := make(map[string]bool)
    for _, e := range entries {
        entryNames[e.Name()] = true
    }

    if entryNames["go.mod"] {
        return LanguageGo
    }
    if entryNames["Cargo.toml"] {
        return LanguageRust
    }
    if entryNames["package.json"] {
        if entryNames["tsconfig.json"] {
            return LanguageTypeScript
        }
        return LanguageJavaScript
    }
    if entryNames["pyproject.toml"] || entryNames["requirements.txt"] {
        return LanguagePython
    }
    if entryNames["Gemfile"] {
        return LanguageRuby
    }
    if entryNames["composer.json"] {
        return LanguagePHP
    }
    if entryNames["CMakeLists.txt"] {
        // Buscar .c o .cpp para diferenciar
        for _, f := range files {
            if f.Ext == ".cpp" || f.Ext == ".cc" {
                return LanguageCpp
            }
            if f.Ext == ".c" {
                return LanguageC
            }
        }
        return LanguageC
    }
    if entryNames["mix.exs"] {
        return LanguageElixir
    }

    // 2. Fallback: estadística de extensiones
    return languageFromExtStats(files)
}

// languageFromExtStats determina el lenguaje por extensión más frecuente.
func languageFromExtStats(files []FileInfo) Language {
    counts := make(map[Language]int)
    for _, f := range files {
        lang := languageFromSingleExt(f.Ext)
        if lang != LanguageUnknown {
            counts[lang]++
        }
    }

    var maxLang Language
    var maxCount int
    for lang, count := range counts {
        if count > maxCount {
            maxCount = count
            maxLang = lang
        }
    }

    if maxCount > 0 {
        return maxLang
    }
    return LanguageUnknown
}

// hashFile computa SHA256 de un archivo.
func hashFile(path string) (string, error) {
    f, err := os.Open(path)
    if err != nil {
        return "", err
    }
    defer f.Close()

    h := sha256.New()
    if _, err := io.Copy(h, f); err != nil {
        return "", err
    }

    return hex.EncodeToString(h.Sum(nil)), nil
}

// languageFromExt mapea extensión a lenguaje.
var extLangMap = map[string]Language{
    ".go":   LanguageGo,
    ".ts":   LanguageTypeScript,
    ".tsx":  LanguageTypeScript,
    ".js":   LanguageJavaScript,
    ".jsx":  LanguageJavaScript,
    ".mjs":  LanguageJavaScript,
    ".rs":   LanguageRust,
    ".py":   LanguagePython,
    ".rb":   LanguageRuby,
    ".php":  LanguagePHP,
    ".c":    LanguageC,
    ".h":    LanguageC,
    ".cpp":  LanguageCpp,
    ".cc":   LanguageCpp,
    ".cxx":  LanguageCpp,
    ".hpp":  LanguageCpp,
    ".cs":   LanguageCSharp,
    ".java": LanguageJava,
    ".kt":   LanguageKotlin,
    ".kts":  LanguageKotlin,
    ".ex":   LanguageElixir,
    ".exs":  LanguageElixir,
    ".swift": LanguageSwift,
}

func languageFromSingleExt(ext string) Language {
    if lang, ok := extLangMap[strings.ToLower(ext)]; ok {
        return lang
    }
    return LanguageUnknown
}
```

---

## 4. CodeNode Creator: Integración con HelixDB

### 4.1 Flujo de creación de nodos

El código de sync HelixDB vive inline en `cmd/zyrocli/onboard.go` (similar a `init.go` y `absorb.go`).

```go
// — en onboard.go —

func syncToHelixDB(ctx context.Context, client *helix.Client, info *scanner.ProjectInfo) (int64, error) {
    // 1. Crear Project node
    projectID, err := client.CreateNode(ctx, "Project", map[string]any{
        "name":          info.Name,
        "language":      string(info.Language),
        "path":          info.Root,
        "status":        "onboarded",
        "current_phase": "phase0",
        "onboarded_at":  time.Now().UTC().Format(time.RFC3339),
        "file_count":    info.FileCount,
    })
    if err != nil {
        return 0, fmt.Errorf("helix: create project node: %w", err)
    }

    // 2. Crear CodeNodes (hasta 500)
    maxCodeNodes := 500
    if len(info.Files) < maxCodeNodes {
        maxCodeNodes = len(info.Files)
    }

    created := 0
    for _, f := range info.Files[:maxCodeNodes] {
        _, _, err := client.UpsertCodeNode(
            uint64(projectID), f.Path, f.Name, "", f.Hash, nil,
        )
        if err != nil {
            continue // skip individual errors
        }
        created++
    }

    _ = created // log si se desea
    return projectID, nil
}
```

### 4.2 Edge structure

```
Project (id=1)
  ├── HAS_CODENODE ──▶ CodeNode (path="main.go", hash="abc123", language="Go")
  ├── HAS_CODENODE ──▶ CodeNode (path="internal/handler.go", hash="def456", ...)
  ├── HAS_CODENODE ──▶ CodeNode (path="go.mod", ...)
  └── ... (hasta 500 edges)
```

### 4.3 Manejo de errores

| Error | Comportamiento |
|-------|---------------|
| HelixDB no disponible | Warning, seguir sin sync |
| Un CodeNode falla | Skip, continuar con los demás |
| Project creado pero CodeNodes fallan | Warning parcial, project existe |

---

## 5. OpenCode Launcher

### 5.1 Context file: `.zyro/task.yaml`

```go
func writeOnboardContext(projectDir string, info *scanner.ProjectInfo, projectID int64) error {
    taskDir := filepath.Join(projectDir, ".zyro")
    if err := os.MkdirAll(taskDir, 0755); err != nil {
        return err
    }

    content := fmt.Sprintf(`phase: "F0"
agent: "zyro-onboarding"
is_onboard: true
project_id: %d
project_language: "%s"
file_count: %d
total_size: %d
scanned_at: "%s"
`, projectID, info.Language, info.FileCount, info.TotalSize, info.ScannedAt.Format(time.RFC3339))

    return os.WriteFile(filepath.Join(taskDir, "task.yaml"), []byte(content), 0644)
}
```

### 5.2 Launch

```go
func launchOpenCode(projectDir string) error {
    if _, err := exec.LookPath("opencode"); err != nil {
        return fmt.Errorf("opencode not found in PATH")
    }
    cmd := exec.Command("opencode", projectDir)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
```

---

## 6. F0-onboarding Agent (cambios vs F0 estándar)

### 6.1 Diferencias clave

| Aspecto | F0 estándar (init) | F0-onboarding (onboard) |
|---------|-------------------|------------------------|
| Origen de datos | Internet (Context, GitMCP) | Código local (CodeNodes en HelixDB) |
| Tareas | patterns, libraries, skills (búsqueda externa) | code-analysis, pattern-detection, lib-extraction, skill-detection |
| Input | handoff.yaml + idea validada | CodeNodes + archivos fuente |
| Output esperado | Patterns, Libraries, Skills desde web | Patterns detectados, librerías extraídas, skills inferidas |
| Tiempo estimado | Depende de búsquedas | ~30s (análisis local) |

### 6.2 Nuevos subagentes necesarios (o modificaciones)

Los agentes F0 actuales (`zyro-phase-0-patterns`, `zyro-phase-0-libraries`) necesitan una variante que:

1. **Code Analysis**: lee CodeNodes de HelixDB, abre archivos fuente, extrae imports, identifica patrones
2. **Library Extraction**: desde go.mod / package.json / Cargo.toml, extrae dependencias
3. **Skill Detection**: infiere skills necesarias (Docker si hay Dockerfile, Testing si hay _test.go, etc.)

Estos cambios se documentan en AGENT.md y se implementarán en una fase posterior (no en esta iteración).

---

## 7. Código de Referencia

### 7.1 `cmd/zyrocli/onboard.go` (estructura completa)

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "time"

    "github.com/secko/zyrocli/internal/db/helix"
    "github.com/secko/zyrocli/internal/scaffold"
    "github.com/secko/zyrocli/internal/scanner"
    "github.com/spf13/cobra"
)

var (
    onboardNoOpenCode bool
    onboardDryRun     bool
    onboardForce      bool
    onboardName       string
    onboardDesc       string
)

var onboardCmd = &cobra.Command{
    Use:   "onboard [path]",
    Short: "Register an existing project in ZyroCLI ecosystem",
    Long: `Scan an existing project directory, detect its language and structure,
create Project and CodeNode entries in HelixDB, and launch OpenCode
with onboarding context.

Unlike 'zyro init', this does not require a handoff.yaml — it works
with any existing codebase.

Examples:
  zyro onboard /path/to/my-project
  zyro onboard .              # current directory
  zyro onboard --dry-run .    # preview without writing anything`,
    Args: cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        // 1. Resolver path
        path := "."
        if len(args) > 0 {
            path = args[0]
        }

        absPath, err := filepath.Abs(path)
        if err != nil {
            return fmt.Errorf("onboard: resolve path: %w", err)
        }

        // 2. Validar
        info, err := os.Stat(absPath)
        if err != nil {
            return fmt.Errorf("onboard: path %q: %w", absPath, err)
        }
        if !info.IsDir() {
            return fmt.Errorf("onboard: %q is not a directory", absPath)
        }

        // 3. Verificar si ya fue inicializado
        state, _ := scaffold.ReadState(absPath)
        if state != nil && state.Initialized && !onboardForce {
            return fmt.Errorf("onboard: project %q already initialized (use --force to re-onboard)", absPath)
        }

        // 4. Escanear proyecto
        cmd.Print("  Escaneando proyecto...")
        s := scanner.NewScanner()
        projectInfo, err := s.Scan(absPath)
        if err != nil {
            return fmt.Errorf("onboard: scan: %w", err)
        }
        cmd.Printf(" %s\n", projectInfo.Language)
        cmd.Printf("  Archivos: %d\n", projectInfo.FileCount)

        // 5. Dry-run: mostrar y salir
        if onboardDryRun {
            cmd.Println("\n  📋 Dry-run resumen:")
            cmd.Printf("    Path:     %s\n", absPath)
            cmd.Printf("    Nombre:   %s\n", projectInfo.Name)
            cmd.Printf("    Lenguaje: %s\n", projectInfo.Language)
            cmd.Printf("    Archivos: %d\n", projectInfo.FileCount)
            cmd.Printf("    Tamaño:   %d bytes\n", projectInfo.TotalSize)
            cmd.Println("\n  (dry-run, no se escribió nada)")
            return nil
        }

        // 6. Conectar HelixDB
        helixClient, helixErr := helix.NewClient(context.Background())
        var projectID int64 = 0
        if helixErr == nil {
            if err := helixClient.EnsureStarted(context.Background()); err != nil {
                cmd.PrintErrln("  ⚠ HelixDB no disponible — proyecto se registra sin BD")
                cmd.PrintErrln("    Iniciá HelixDB con: helix start dev --disk")
            } else {
                pid, dbErr := syncToHelixDB(context.Background(), helixClient, projectInfo)
                if dbErr != nil {
                    cmd.PrintErrln("  ⚠ Error sync HelixDB:", dbErr)
                } else {
                    projectID = pid
                    cmd.Printf("  ✓ Proyecto registrado en HelixDB (ID: %d)\n", projectID)
                }
            }
            _ = helixClient.Close()
        } else {
            cmd.PrintErrln("  ⚠ HelixDB no disponible — proyecto se registra sin BD")
        }

        // 7. Escribir .zyro/state.json
        projectName := onboardName
        if projectName == "" {
            projectName = projectInfo.Name
        }
        scaffold.WriteState(absPath, &scaffold.State{
            Initialized: true,
            ProjectName: projectName,
            TargetDir:   absPath,
            Version:     "onboard",
        })

        // 8. Escribir .zyro/task.yaml (contexto para OpenCode)
        if err := writeOnboardTaskFile(absPath, projectInfo, projectID); err != nil {
            cmd.PrintErrln("  ⚠ Advertencia: no se pudo escribir task.yaml:", err)
        }

        // 9. No abrir OpenCode?
        if onboardNoOpenCode {
            cmd.Println("  Proyecto listo (--no-opencode)")
            return nil
        }

        // 10. Launch OpenCode
        if _, err := exec.LookPath("opencode"); err != nil {
            cmd.PrintErrln("  ⚠ opencode no encontrado en PATH")
            cmd.Printf("  Abrí manualmente: opencode %s\n", absPath)
            return nil
        }

        cmd.Printf("  Abriendo OpenCode en %s...\n", absPath)
        openCmd := exec.Command("opencode", absPath)
        openCmd.Stdin = os.Stdin
        openCmd.Stdout = os.Stdout
        openCmd.Stderr = os.Stderr
        _ = openCmd.Run()

        return nil
    },
}

func init() {
    rootCmd.AddCommand(onboardCmd)
    onboardCmd.Flags().BoolVarP(&onboardNoOpenCode, "no-opencode", "", false, "register project but do not open OpenCode")
    onboardCmd.Flags().BoolVarP(&onboardDryRun, "dry-run", "n", false, "preview scan results without writing anything")
    onboardCmd.Flags().BoolVarP(&onboardForce, "force", "f", false, "re-onboard an already initialized project")
    onboardCmd.Flags().StringVarP(&onboardName, "name", "", "", "custom project name (default: directory name)")
    onboardCmd.Flags().StringVarP(&onboardDesc, "desc", "d", "", "project description (optional)")
}
```

### 7.2 Conexión de todas las piezas

El flujo completo conecta:

1. **CLI layer** (`cmd/zyrocli/onboard.go`) — Cobra command, flags, orquestación
2. **Scanner** (`internal/scanner/`) — Escaneo y detección de lenguaje
3. **HelixDB** (`internal/db/helix/`) — Persistencia de Project + CodeNodes
4. **State** (`internal/scaffold/state.go`) — `.zyro/state.json` para idempotencia
5. **OpenCode** — Launch con contexto de onboarding

No se necesita infraestructura nueva: todos los componentes existen en el codebase actual.
