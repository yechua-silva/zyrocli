# `zyro onboard` — Onboarding de proyectos existentes en ZyroCLI

> **Fecha:** 17 Junio 2026
> **Versión:** 1.0
> **Proyecto:** ZyroAgentCLI (`github.com/secko/zyrocli`)

---

## 1. Objetivo

`zyro onboard /path/to/project` permite incorporar proyectos EXISTENTES al ecosistema ZyroCLI sin necesidad de `handoff.yaml`. A diferencia de `zyro init` (que crea proyectos nuevos desde cero), `onboard` analiza la estructura del proyecto, detecta su lenguaje y stack, crea los nodos correspondientes en HelixDB, y abre OpenCode con un pipeline F0 adaptado que analiza el código real en vez de buscar en internet.

**Valor clave:** un desarrollador puede tomar cualquier proyecto existente (Go, JS/TS, Rust, Python, etc.) y empezar a usar ZyroCLI para desarrollo asistido sin configurar nada manualmente.

---

## 2. Flujo de Usuario

```
Usuario                       ZyroCLI                          HelixDB              OpenCode
  │                             │                                │                     │
  │  zyro onboard /mi-proyecto  │                                │                     │
  │────────────────────────────▶│                                │                     │
  │                             │                                │                     │
  │                             │  ┌─ 1. Validar path            │                     │
  │                             │  │    • existe?                │                     │
  │                             │  │    • es directorio?         │                     │
  │                             │  │    • único? (no ya onboard) │                     │
  │                             │  └─────────────────────────────│                     │
  │                             │                                │                     │
  │  Escaneando proyecto...     │  ┌─ 2. ProjectScanner          │                     │
  │◀────────────────────────────│  │    • WalkDir (rápido, <5s)  │                     │
  │                             │  │    • Ignora .git,node_modules│                     │
  │                             │  │    • Detecta lenguaje        │                     │
  │  Lenguaje: Go               │  │    • Cuenta archivos por tipo│                     │
  │  Archivos: 142 (.go: 98)    │  └─────────────────────────────│                     │
  │◀────────────────────────────│                                │                     │
  │                             │                                │                     │
  │                             │  ┌─ 3. Sync HelixDB            │                     │
  │                             │  │    • Crear Project node     │────▶ Project{Go}    │
  │                             │  │    • Crear CodeNodes        │────▶ CodeNode{x142} │
  │                             │  │    • Crear edges HAS_CODENODE│────▶ edges{142}     │
  │  ✓ Proyecto registrado      │  └─────────────────────────────│                     │
  │◀────────────────────────────│                                │                     │
  │                             │                                │                     │
  │                             │  ┌─ 4. Escribir .zyro/task.yaml│                     │
  │                             │  │    • Contexto de onboarding │                     │
  │                             │  │    • Flag is_onboard=true   │                     │
  │                             │  └─────────────────────────────│                     │
  │                             │                                │                     │
  │                             │  ┌─ 5. Launch OpenCode         │                     │
  │  Abriendo OpenCode...       │  │    exec("opencode", path)   │                    │
  │◀────────────────────────────│────────────────────────────────────────────────────│
  │                             │                                │                     │
  │                             │     [F0-onboarding se ejecuta dentro de OpenCode]    │
  │                             │                                │                     │
  │                             │     ┌─ Lee CodeNodes de HelixDB◀──── CodeNodes      │
  │                             │     └─ Analiza código real      │                     │
  │                             │     └─ Detecta patrones         │────▶ Facts{patrón} │
  │                             │     └─ Identifica librerías     │────▶ Facts{librería}│
  │                             │     └→ Detecta skills necesarias│────▶ Facts{skill}  │
```

---

## 3. Especificación Técnica Detallada

### 3.1 Comando CLI

```
zyro onboard [path] [flags]

Arguments:
  path    Ruta al proyecto existente (directorio). Por defecto: "."

Flags:
  --no-opencode    Solo registrar en HelixDB, no abrir OpenCode
  --dry-run        Mostrar qué se haría sin ejecutar cambios
  --force          Re-onboardear aunque ya esté registrado
  --name string    Nombre del proyecto (por defecto: nombre del directorio)
  --desc string    Descripción del proyecto (opcional)
```

### 3.2 Restricciones y validaciones

| Condición | Acción |
|-----------|--------|
| Path no existe | Error: `path /foo/bar: no such file or directory` |
| Path no es directorio | Error: `path /foo/bar is not a directory` |
| Path ya tiene `.zyro/state.json` (ya onboardeado) | Error: `project already initialized. Use --force to re-onboard` |
| Path vacío | Error: `path is empty` |
| HelixDB no disponible | Advertencia no bloqueante: proyecto se registra pero sin persistencia |
| opencode no disponible | Advertencia: proyecto registrado pero no se abre editor |

### 3.3 ProjectScanner

**Archivo:** `internal/scanner/scanner.go`

Scanner rápido (<5s en proyectos <10K archivos) que produce un `ProjectInfo`:

```go
type ProjectScanner struct {
    ignorePatterns []string  // .git, node_modules, vendor, dist, .venv, __pycache__, target, build
}

func (s *ProjectScanner) Scan(root string) (*ProjectInfo, error)
```

**Ignore patterns por defecto:**
```
.git, .hg, .svn, node_modules, vendor, dist, build, target,
.venv, __pycache__, *.pyc, *.pyo, .env, .env.local,
.idea, .vscode, *.swp, *.swo, .DS_Store, Thumbs.db,
*.exe, *.dll, *.so, *.dylib, .zyro, .helix
```

**Algoritmo de detección de lenguaje:**

| Archivo presente | Lenguaje detectado |
|-----------------|-------------------|
| `go.mod` | Go |
| `package.json` (+ `tsconfig.json` → TypeScript, si no → JavaScript) |
| `Cargo.toml` | Rust |
| `pyproject.toml` o `requirements.txt` | Python |
| `Gemfile` | Ruby |
| `Cargo.toml` | Rust |
| `composer.json` | PHP |
| `CMakeLists.txt` | C/C++ |
| `*.csproj` o `*.sln` | C# |
| `pom.xml` o `build.gradle` | Java |
| `mix.exs` | Elixir |
| `swift` | Swift |
| `Dockerfile` (sin otro) | Desconocido (infra) |

Si no se detecta ningún lenguaje, se reporta como `LanguageUnknown`.

**Archivos escaneados:** todo archivo regular que no coincida con ignore patterns. Se computa:
- Path relativo
- Extensión
- Tamaño
- Hash SHA256 del contenido (para detectar cambios futuros)
- Language (derivado de extensión para archivos individuales)

### 3.4 HelixDB Sync

**Dónde:** inline en `cmd/zyrocli/onboard.go`

**Nodos a crear:**

1. **Project node**
   - Label: `Project`
   - Properties: `name`, `language`, `path` (absoluto), `status: "onboarded"`, `current_phase: "phase0"`, `onboarded_at: <timestamp>`, `file_count`, `description`

2. **CodeNode por cada archivo escaneado** (hasta un límite configurable, default 500)
   - Label: `CodeNode`
   - Properties: `project_id`, `path`, `name`, `hash`, `language`, `size`
   - Edge: `Project ── HAS_CODENODE ──▶ CodeNode`

3. **Limitaciones de escalabilidad:**
   - Si el proyecto tiene >500 archivos, solo se crean CodeNodes para los primeros 500 más importantes (por extensión reconocida: .go, .ts, .js, .rs, .py, etc.)
   - Si hay >10,000 archivos, se aborta con sugerencia de ejecutar `absorb` para subconjuntos

### 3.5 OpenCode Launch

Se escribe `.zyro/task.yaml` con:

```yaml
phase: "F0"
agent: "zyro-onboarding"   # agente especializado
is_onboard: true
project_language: "Go"
file_count: 142
detected_patterns: []
required_output:
  patterns: true
  libraries: true
  skills: true
```

Luego se ejecuta:
```go
exec.Command("opencode", projectDir)
```

El agente `zyro-onboarding` es un nuevo agente (o modificación de los agentes F0) que en vez de buscar en internet:
1. Lee CodeNodes de HelixDB
2. Analiza archivos fuente reales
3. Detecta patrones de diseño usados (Repository, Factory, DTO, Controller, etc.)
4. Identifica librerías desde `go.mod`/`package.json`/`Cargo.toml`
5. Detecta skills necesarias (Docker, Cloud, Testing, etc.)
6. Guarda Facts en HelixDB

### 3.6 Modo Dry-Run

Con `--dry-run`, el comando:
1. Valida el path
2. Ejecuta el scanner (para mostrar stats)
3. Muestra qué nodos se crearían en HelixDB
4. NO escribe nada en disco ni DB
5. NO abre OpenCode

---

## 4. Componentes

| Componente | Archivo | Propósito | LOC estimado |
|-----------|---------|-----------|-------------|
| CLI Command | `cmd/zyrocli/onboard.go` | Punto de entrada Cobra, flags, orquestación | ~130 |
| Scanner Types | `internal/scanner/types.go` | `ProjectInfo`, `FileInfo`, `Language` enum | ~50 |
| Scanner Engine | `internal/scanner/scanner.go` | WalkDir, detección de lenguaje, hashing | ~100 |
| HelixDB Sync | `cmd/zyrocli/onboard.go` (inline) | Crear Project + CodeNodes en DB | ~80 |
| OpenCode Launch | `cmd/zyrocli/onboard.go` (inline) | Escribir .zyro/task.yaml + exec opencode | ~40 |
| AGENT.md update | `AGENT.md` | Documentar comando `onboard` + flujo | ~20 |
| Tests | `internal/scanner/scanner_test.go` | Tests unitarios del scanner | ~80 |

---

## 5. Archivos Afectados

| Archivo | Cambio |
|---------|--------|
| `cmd/zyrocli/onboard.go` | **Nuevo** — comando `zyro onboard` |
| `internal/scanner/types.go` | **Nuevo** — tipos del scanner |
| `internal/scanner/scanner.go` | **Nuevo** — implementación del scanner |
| `internal/scanner/scanner_test.go` | **Nuevo** — tests del scanner |
| `cmd/zyrocli/main.go` | Registrar comando `onboardCmd` en init() |
| `AGENT.md` | Agregar sección de onboarding + comando |

---

## 6. Criterios de Aceptación

### CA1: Onboarding básico
- [ ] `zyro onboard /tmp/test-project` escanea y registra correctamente
- [ ] Muestra "Lenguaje: Go" cuando hay go.mod
- [ ] Crea Project node en HelixDB con status "onboarded"
- [ ] Crea CodeNodes para archivos relevantes

### CA2: Sin handoff.yaml
- [ ] El comando funciona sin que exista handoff.yaml
- [ ] No se crea handoff.yaml automáticamente

### CA3: Detección de lenguaje
- [ ] Detecta correctamente Go, TypeScript, JavaScript, Rust, Python
- [ ] Reporta "unknown" para proyectos sin archivos de configuración reconocibles
- [ ] La detección es determinística (mismo resultado siempre)

### CA4: Ignorar directorios correctamente
- [ ] No escanea .git/
- [ ] No escanea node_modules/
- [ ] No escanea vendor/
- [ ] No escanea .venv/ ni __pycache__/

### CA5: OpenCode launch
- [ ] Abre OpenCode en el directorio del proyecto
- [ ] Escribe .zyro/task.yaml con is_onboard=true
- [ ] Respeta --no-opencode flag

### CA6: HelixDB no disponible
- [ ] El comando no falla si HelixDB no está corriendo
- [ ] Muestra advertencia pero el proyecto queda usable

### CA7: Dry-run
- [ ] `--dry-run` muestra stats sin escribir nada
- [ ] No abre OpenCode en dry-run
- [ ] No crea .zyro/ directory

### CA8: Re-onboarding
- [ ] Si el proyecto ya fue onboardeado, muestra error claro
- [ ] `--force` permite re-onboardear (actualiza nodos)

### CA9: Velocidad
- [ ] Scanner completa <5s para proyectos <10K archivos
- [ ] HelixDB sync completa <2s para 500 CodeNodes

---

## 7. Riesgos

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|-------------|---------|------------|
| Proyecto enorme (>50K archivos) | Baja | Alto | Límite de 500 CodeNodes, abortar si >10K |
| HelixDB caída | Media | Bajo | Advertencia no bloqueante, el proyecto funciona igual |
| Hash collision (SHA256) | Muy baja | Bajo | Aceptable para este caso de uso |
| Directorio con permisos restringidos | Baja | Medio | Skip con warning, no abortar todo |
| opencode no instalado | Media | Bajo | Advertencia, proyecto queda listo para abrir manualmente |
| Proyecto ya en otro Pipeline SDD | Baja | Medio | Detectar .zyro/state.json y pedir --force |
