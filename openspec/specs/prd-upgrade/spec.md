# PRD: Absorción de skills Pocock en ZyroCLI — Pipeline SDD v2

## Problema
ZyroCLI tiene un pipeline SDD (F0→F4) que arranca investigación sin contexto de dominio. Los agentes de F0 (patterns, libraries, skills-find) buscan sin saber QUÉ están buscando. No hay alineación previa con el usuario. Además, los skills de exploración (sdd-explore) y especificación (sdd-spec) son demasiado thin (28 y 35 líneas) y no incorporan técnicas modernas de entrevista y documentación que Matt Pocock ha demostrado en su ecosistema de skills (135K ⭐).

## Solución
Se absorben 6 skills de Matt Pocock directamente en 3 agentes de ZyroCLI + se agregan 2 skills Go independientes. No se crean skills paralelas. Se modifica el pipeline SDD para incluir una fase PRE-F0 obligatoria antes de F0.

Pipeline resultante:
```
PRE-F0 (zyro-pre-f0) — NUEVO, absorbe grill-me, domain-model, triage, improve-codebase-architecture
   ↓
F0 (zyro-phase-0-patterns, libraries, skills-find) — sin cambios estructurales
   ↓
F1 (zyro-sdd-spec) — absorbe to-prd, ahora produce PRD.md
   ↓
F2 (design + tasks) — sin cambios
   ↓
F3 (apply + verify) — sin cambios
   ↓
F4 (archive) — sin cambios
```

## Usuarios / Actores afectados
1. **Operadores ZyroCLI** — ejecutan el pipeline SDD. Ahora tienen una fase de alineación previa que les da contexto.
2. **Agentes SDD (subagentes)** — zyro-pre-f0 es nuevo, zyro-sdd-explore y zyro-sdd-spec se modifican.
3. **Orquestador (zyro-orchestrator)** — su flujo cambia: ahora incluye PRE-F0 antes de F0.
4. **Desarrolladores Go** — usan golang-documentation y golang-testing skills para escribir código.

## User Stories

1. Como operador, cuando ejecuto el pipeline SDD, quiero que primero se ejecute una fase de alineación PRE-F0 que entreviste al usuario sobre el dominio, para que los agentes de F0 tengan contexto real y no busquen a ciegas.

2. Como operador, cuando ejecuto PRE-F0, quiero que el agente haga grill-me (una pregunta a la vez, sharpening de términos, stress-test con escenarios) para que el dominio quede bien definido antes de investigar.

3. Como operador, después del grill-me, quiero que el agente genere un domain-model (alignment.md + domain-model.md con glosario) para que el vocabulario quede documentado y los agentes de F0 lo usen.

4. Como operador con backlog acumulado, quiero que PRE-F0 incluya triage para ordenar issues/features por prioridad, aplicando una state machine (needs-triage → ready-for-agent / needs-info / wontfix).

5. Como operador post-implementación, quiero que improve-codebase-architecture escanee la fricción arquitectónica y proponga refactors profundos (deep modules, seams, adapters).

6. Como operador, cuando zyro-sdd-explore se ejecute, quiero que primero lea los docs existentes (openspec/, docs/adr/, CONTEXT.md, AGENT.md, .zyro/config.yaml) y luego entreviste al usuario con las reglas de grill-with-docs, para que la exploración sea más precisa.

7. Como operador, cuando zyro-sdd-spec se ejecute, quiero que produzca un PRD.md en el nuevo formato (Problema, Solución, Usuarios, Stories, Acceptance, Decisiones, Testing, Out of Scope, Módulos, Notas) sin re-entrevistarme, sintetizando lo que ya se resolvió en PRE-F0 + F0.

8. Como agente zyro-sdd-apply, quiero tener disponibles las skills golang-documentation y golang-testing instaladas en .skills/ del proyecto para escribir código Go idiomático.

## Criterios de aceptación

- [ ] Dado el pipeline SDD, cuando se inicia, entonces PRE-F0 se ejecuta siempre antes que F0.
- [ ] Dado PRE-F0, cuando el agente zyro-pre-f0 se ejecuta, entonces hace grill-me (entrevista 1-pregunta-a-la-vez con sharpening).
- [ ] Dado PRE-F0, después del grill-me, entonces produce alignment.md + domain-model.md (glosario).
- [ ] Dado PRE-F0 con backlog, entonces puede hacer triage con state machine (5 estados).
- [ ] Dado PRE-F0, cuando se solicita improve-codebase-architecture, entonces escanea fricción y genera reporte.
- [ ] Dado zyro-sdd-explore, cuando se ejecuta, entonces lee docs en orden (openspec/ → docs/adr/ → CONTEXT.md → AGENT.md → .zyro/config.yaml) antes de explorar.
- [ ] Dado zyro-sdd-explore, cuando entrevista, entonces hace 1 pregunta a la vez y sharpenea términos vagos.
- [ ] Dado zyro-sdd-explore, cuando termina, entonces produce exploration-summary.md + CONTEXT.md + ADRs opcionales.
- [ ] Dado zyro-sdd-spec, cuando se ejecuta, entonces NO re-entrevista al usuario, sintetiza de inputs previos.
- [ ] Dado zyro-sdd-spec, cuando produce output, entonces es PRD.md con las 9 secciones definidas + deep modules.
- [ ] Dado zyrocli install, cuando se ejecuta, entonces instala golang-documentation y golang-testing en .skills/ del proyecto.
- [ ] Dado el binario de zyrocli, entonces contiene embebidas las 2 nuevas skills Go + el agente zyro-pre-f0.

## Decisiones de implementación

1. **No se crean skills paralelas independientes** para grill-with-docs, to-prd, grill-me, domain-model, triage, improve-codebase-architecture. Se absorben DENTRO de los SKILL.md de los agentes existentes o del nuevo agente.

2. **El nuevo agente se llama `zyro-pre-f0`** y vive en `internal/opencode/skills/zyro-pre-f0/SKILL.md`. Absorbe: grill-me, domain-model, triage, improve-codebase-architecture.

3. **`zyro-sdd-explore`** se reescribe completamente. Su SKILL.md actual (28 líneas) se reemplaza con: READ DOCS → INTERVIEW (grill-with-docs) → OUTPUT.

4. **`zyro-sdd-spec`** se reescribe. Su SKILL.md actual (35 líneas) se reemplaza con el formato PRD de to-prd. Crítico: NO re-entrevistar.

5. **`zyro-orchestrator`** se modifica para incluir PRE-F0 como primera fase, antes de F0.

6. **`golang-documentation`** y **`golang-testing`** se agregan como skills independientes (no absorbidas). Se crean en `internal/opencode/skills/golang-documentation/SKILL.md` y `internal/opencode/skills/golang-testing/SKILL.md`.

7. **skills_embed.go** se modifica para agregar `//go:embed skills/zyro-pre-f0/SKILL.md`, `//go:embed skills/golang-documentation/SKILL.md`, `//go:embed skills/golang-testing/SKILL.md`.

8. **AGENT.md** se actualiza: el flujo PRE-F0 → F0 → F1 → F2 → F3 → F4 reemplaza al actual. Las 4 preguntas de Pre-F0 interview se redistribuyen: Q1 (objetivo) va a grill-me, Q2 (librerías) va a F0, Q3 (LOC) va a F2, Q4 (Engram) queda en orquestador.

9. **No se requieren nuevas dependencias Go** — el stack actual (Cobra, Bubble Tea, Bubbles, stdlib) cubre todo.

## Decisiones de testing

- **Unitario**: Cada SKILL.md se prueba por contenido esperado (verificar que contenga las secciones requeridas).
- **Integración**: El pipeline completo se prueba con un flow end-to-end: PRE-F0 → F0 → F1 → F2 → F3 → F4.
- **Fuera de scope**: No se testea la ejecución real de los agentes (eso depende del runtime de OpenCode/Claude). Se testea que los archivos SKILL.md tengan el formato correcto y que el binario los embeble.

## Fuera de scope

- No se implementa el comando `zyrocli align` como subcomando Cobra separado (la fase PRE-F0 se maneja via el agente zyro-pre-f0, no via CLI).
- No se implementa improve-codebase-architecture como comando separado — vive dentro de zyro-pre-f0.
- No se agregan más skills que las listadas.
- No se modifica el task-board MCP (triage se maneja via el prompt del agente, no via código).

## Módulos afectados

### deep module: zyro-pre-f0 (NUEVO)
Ubicación: `internal/opencode/skills/zyro-pre-f0/SKILL.md`
Complejidad interna: 4 skills de Pocock integradas (grill-me, domain-model, triage, improve-codebase-architecture)
Interfaz: Un solo SKILL.md con 4 sub-comandos internos
Testeable por: Verificar que el SKILL.md contenga las 4 secciones

### deep module: zyro-sdd-explore (REESCRITO)
Ubicación: `internal/opencode/skills/zyro-sdd-explore/SKILL.md`
Complejidad interna: READ DOCS + INTERVIEW (grill-with-docs) + OUTPUT
Interfaz: Un SKILL.md con 3 pasos claros
Testeable por: Verificar contenido del archivo

### deep module: zyro-sdd-spec (REESCRITO)
Ubicación: `internal/opencode/skills/zyro-sdd-spec/SKILL.md`
Complejidad interna: Formato PRD de 9 secciones + deep modules + regla NO re-entrevistar
Interfaz: Un SKILL.md que toma inputs (exploration-summary.md + CONTEXT.md + ADRs) y produce PRD.md
Testeable por: Verificar contenido del archivo

### Módulo: skills_embed.go (MODIFICADO)
Agregar 3 //go:embed directives
Testeable por: go build + verificar que los archivos se embeban

### Módulo: golang-documentation (NUEVO)
Ubicación: `internal/opencode/skills/golang-documentation/SKILL.md`
Skill independiente de samber/cc-skills-golang

### Módulo: golang-testing (NUEVO)
Ubicación: `internal/opencode/skills/golang-testing/SKILL.md`
Skill independiente de samber/cc-skills-golang


### Bugs existentes a corregir (identificados en sesión de prueba)

#### Bug 1: buildInstallConfig() no registra nuevos agentes

**Archivo**: `cmd/zyrocli/install.go` — función `buildInstallConfig()` (línea 175)

**Problema**: El mapa de agentes en `buildInstallConfig()` es hardcodeado. No incluye `zyro-pre-f0` ni `to-issues`. Tampoco agrega `zyro-pre-f0` a la lista de subagentes permitidos del orquestador (líneas 193-208). Cuando el usuario ejecuta `zyrocli install`, escribe un config en `~/.config/opencode/opencode.jsonc` que no reconoce los nuevos agentes.

**Fix**: 
1. Agregar entrada `zyro-pre-f0` al mapa de agentes en `buildInstallConfig()`:
   ```go
   "zyro-pre-f0": {
       Mode: "subagent", Description: "PRE-F0: Alineación de dominio — grill-me, domain-model, triage, improve-arch",
       Prompt: "{skill:zyro-pre-f0}", Hidden: true,
       Permission: map[string]any{
           "read": "allow", "bash": "allow", "webfetch": "allow", "question": "allow",
           "write": "deny", "edit": "deny",
           "task": map[string]any{"*": "allow"},
       },
   },
   ```
2. Agregar `to-issues` como agente user-invoked:
   ```go
   "to-issues": {
       Mode: "subagent", Description: "Genera GitHub Issues desde PRDs",
       Prompt: "{skill:to-issues}", Hidden: true,
       Permission: map[string]any{
           "read": "allow", "bash": "allow", "webfetch": "allow",
           "write": "deny", "edit": "deny",
       },
   },
   ```
3. Agregar `"zyro-pre-f0": "allow"` a la lista de subagentes del orquestador (línea ~208)

#### Bug 2: WriteGlobalConfig() pisa config existente

**Archivo**: `internal/opencode/config.go` — función `WriteGlobalConfig()` (línea 71)

**Problema**: Usa `os.WriteFile` sin leer el archivo existente. Si el usuario tenía MCP servers configurados manualmente (gitMCP local, Context con settings, etc.), se pierden al ejecutar `zyrocli install`.

**Fix**: Cambiar `WriteGlobalConfig()` para:
1. Leer el archivo existente si existe
2. Parsear el JSON existente
3. **Fusionar** los MCP servers nuevos con los existentes (no reemplazar)
4. Si un MCP server ya existe con el mismo nombre, NO pisarlo (el usuario puede tener config personalizada)
5. Solo AGREGAR MCP servers nuevos que no existían antes

```go
func WriteGlobalConfig(cfg *Config) (string, error) {
    path := expandHome(OpenCodeConfigPath)
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return "", fmt.Errorf("opencode: create config dir %s: %w", dir, err)
    }

    // Leer config existente si existe
    existing := &Config{}
    if data, err := os.ReadFile(path); err == nil {
        if err := json.Unmarshal(data, existing); err == nil {
            // Fusionar MCP servers: los del usuario tienen prioridad
            if existing.MCP == nil {
                existing.MCP = make(map[string]MCPEntry)
            }
            for k, v := range cfg.MCP {
                if _, exists := existing.MCP[k]; !exists {
                    existing.MCP[k] = v // Solo agregar si no existe
                }
            }
            cfg.MCP = existing.MCP

            // Fusionar agentes: agregar nuevos sin borrar existentes
            if existing.Agent != nil {
                for k, v := range existing.Agent {
                    if _, exists := cfg.Agent[k]; !exists {
                        cfg.Agent[k] = v
                    }
                }
            }
        }
    }

    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return "", fmt.Errorf("opencode: marshal config: %w", err)
    }
    if err := os.WriteFile(path, data, 0644); err != nil {
        return "", fmt.Errorf("opencode: write %s: %w", path, err)
    }
    return path, nil
}
```

#### Bug 3: gitMCP configurado como remote URL inservible

**Archivo**: `cmd/zyrocli/install.go` — línea 340-343

**Problema**: gitMCP está configurado como:
```go
"gitmcp": { Type: "remote", URL: "https://gitmcp.io/docs" },
```
Esto no es un servidor MCP funcional. Es solo una URL de documentación. No permite hacer git operations.

**Fix**: Reemplazar con un servidor git MCP funcional:
```go
"gitmcp": {
    Type: "local",
    Command: []string{"uvx", "mcp-server-git", "--repository", "."},
},
```
O detectar si el usuario tiene `mcp-server-git` instalado y usarlo, o si no, instalar `uvx mcp-server-git` automáticamente durante el install.

#### Bug 3b: Context MCP — comando no verificado y config perdida

**Archivo**: `cmd/zyrocli/install.go` — línea 336-339

**Problema**: Context está registrado como:
```go
"context": { Type: "local", Command: []string{"context", "serve"} },
```

Dos problemas:
1. **Comando no verificado**: No se ha probado que `context serve` sea el comando MCP correcto para `@neuledge/context`. Si el binario expone otro subcomando o necesita flags (ej: `context mcp`, `context start --port X`), este entry point no funciona.
2. **Config perdida**: Si el usuario tenía config personalizada de Context (paths de documentación, settings, etc.), el overwrite de `WriteGlobalConfig()` la borró. Aunque el paquete npm se reinstale (línea 126-132), la configuración MCP específica se pierde.

**Fix**:
1. Verificar el comando MCP real de `@neuledge/context`:
   ```bash
   context --help
   context serve --help  # o el comando que exponga MCP
   ```
   Si el comando es diferente, actualizarlo en `buildInstallConfig()`.

2. Misma solución que Bug 2 — `WriteGlobalConfig()` debe mergear con la config existente para preservar settings personalizados.

3. Opcionalmente, agregar validación en `zyrocli doctor` que verifique que el MCP de Context responde correctamente:
   ```go
   // internal/setup/doctor.go
   func checkContextMCP() error {
       cmd := exec.Command("context", "serve", "--check") // o el flag apropiado
       return cmd.Run()
   }
   ```

#### Bug 4: ASCII art ZYRO 3D se corrompe

**Archivo**: `internal/tui/brand.go` + `internal/tui/assets/brand.txt`

**Problema**: El arte 3D ZYRO usa caracteres Unicode y espacios significativos. `sanitizeArt()` hace `TrimRight` a los espacios, lo que rompe el arte cuando se centra. Además `BrandLines()` trimea el `\n` final.

**Fix**: 
1. NO hacer `TrimRight` en las líneas del brand (los espacios son parte del diseño 3D)
2. En `centeredBlock()`, calcular el ancho visual con `displaywidth` en vez de `len()` (los caracteres Unicode pueden tener ancho variable)
3. En `BrandLines()`, no trimear el string — preservar la última línea

```go
func sanitizeArt(art string) string {
    // NO trimear espacios — son parte del diseño 3D
    // Solo normalizar \r\n → \n
    return strings.ReplaceAll(art, "\r\n", "\n")
}

func BrandLines() []string {
    // Preservar estructura exacta
    return strings.Split(brandArt, "\n")
}
```

#### Bug 5: Scroll acumulativo en instalador TUI

**Archivo**: `internal/tui/install.go`

**Problema**: 
1. `tea.Printf` (línea 175) escribe al log de la terminal, acumulándose fuera del modelo
2. `View()` (línea 207) nunca usa `m.height` para limitar la salida
3. No hay límite de altura en la vista

**Fix**:
1. Eliminar `tea.Printf` — reemplazar por actualización del modelo interno
2. En `View()`, si el contenido excede `m.height`, truncar con scroll
3. Agregar límite de líneas visibles

```go
// En Update(), reemplazar tea.Printf por silencio:
case InstalStepMsg:
    // ... manejar estado ...
    m.currentIdx++
    if m.currentIdx >= len(m.steps) {
        m.done = true
        return m, tea.Quit // Sin tea.Printf
    }

// En View(), limitar altura:
func (m InstallModel) View() string {
    var b strings.Builder
    // ... render normal ...
    content := b.String()
    
    // Limitar altura si es necesario
    if m.height > 0 {
        lines := strings.Split(content, "\n")
        maxLines := m.height - 2 // margen
        if len(lines) > maxLines {
            lines = lines[:maxLines]
            lines = append(lines, helpStyle.Render("  [... scroll oculto ...]"))
        }
        content = strings.Join(lines, "\n")
    }
    return content
}
```

#### Bug 6: Logo OpenCode (logo.txt) puede tener trailing spaces

**Archivo**: `internal/tui/assets/logo.txt`

**Problema**: Similar al brand, el logo puede tener espacios que se pierden con TrimRight.

**Fix**: Misma solución que Bug 4 — no trimear espacios.

### Resumen de archivos a modificar

| Archivo | Bug | Cambio |
|---------|-----|--------|
| `cmd/zyrocli/install.go` | Bug 1, 3 | Agregar zyro-pre-f0 + to-issues al config + gitMCP local |
| `internal/opencode/config.go` | Bug 2 | WriteGlobalConfig mergea en vez de pisar |
| `internal/tui/brand.go` | Bug 4 | No trimear espacios en ASCII art |
| `internal/tui/assets/brand.txt` | Bug 4 | Preservar estructura exacta |
| `internal/tui/install.go` | Bug 5 | Eliminar tea.Printf, limitar altura en View |
| `internal/tui/assets/logo.txt` | Bug 6 | Preservar trailing spaces |


### Handoff — Trazabilidad entre fases (absorbido en zyro-orchestrator)

**Problema**: Engram (memoria causal) debía manejar la trazabilidad entre sesiones vía StepSave, pero DecayAndRefresh() es stub, ReinforceSalience() tiene update pendiente, PostPhase solo hace keyword fallback. El humano tiene que recordar contexto manualmente.

**Skill de referencia**: `/handoff` de Matt Pocock. Genera documentos de handoff al final de cada fase.

**Solución**: El orquestador (`zyro-orchestrator`) absorbe `/handoff`. Al finalizar cada fase (PRE-F0, F0, F1, F2, F3, F4), el orquestador genera un archivo de handoff:

```
.zyro/handoffs/<phase>-handoff.md
```

Contenido del handoff:
```markdown
# Handoff: <FASE>
Fecha: <timestamp>

## Contexto actual
Resumen de lo que se hizo, decisiones tomadas, estado de artefactos.

## Artefactos generados
- openspec/exploration-summary.md
- CONTEXT.md
- docs/adr/*.md
- etc.

## Decisiones pendientes
- [ ] Decisión X sin resolver
- [ ] Trade-off Y por revisar

## Próximos pasos
Qué debe hacer la siguiente fase.

## Estado de HelixDB
IDs de nodos creados/modificados en esta fase.
```

**Ubicación en el flujo del orquestador**:
```
Inicio de fase → ejecutar agente de fase → [handoff] → siguiente fase
```

El handoff NO bloquea la ejecución — se genera automáticamente al terminar cada fase. Cuando Engram esté completo (DecayAndRefresh, ReinforceSalience, PostPhase funcionando), el handoff puede deprecarse.

### to-issues — Visibilidad externa (skill independiente)

**Skill de referencia**: `/to-issues` de Matt Pocock. Genera GitHub Issues desde el PRD/tareas.

**Relación con zyro-sdd-tasks**: zyro-sdd-tasks descompone en el Task Board MCP interno. `/to-issues` genera GitHub Issues en formato vertical-slice. Son complementarios, no reemplazos.

**Implementación**: Se agrega como skill independiente (NO absorbida en ningún agente):
- Ubicación: `internal/opencode/skills/to-issues/SKILL.md`
- Se embebe via `//go:embed` en `skills_embed.go`
- Es user-invoked (`disable-model-invocation: true`), para que el humano lo invoque como `/to-issues` en OpenCode
- No va en el pipeline SDD base — se usa cuando se necesita visibilidad externa en GitHub

**SKILL.md** (contenido):
```markdown
---
name: to-issues
description: Generate GitHub Issues from PRD specs and tasks for external visibility
disable-model-invocation: true
---

# To Issues

Generate well-structured GitHub Issues from PRD specs and task definitions.

## When to use
- When you need external visibility into the development process
- When stakeholders need to track progress on GitHub
- When you want vertical-slice issues that cross-cut technical boundaries

## Process
1. Read the PRD from openspec/specs/
2. Read the task definitions from the Task Board
3. For each task/user story, create a GitHub Issue with:
   - Title: Feature/component name
   - Description: User story in Given/When/Then format
   - Labels: phase, component, priority
   - Assignee: (optional)
   - Milestone: (optional)
4. Link related issues as dependencies
5. Apply label `ready-for-dev` when the issue is actionable

## Output
GitHub Issues created via GitHub API or CLI.
```

### Skills Go independientes (embebidas en binario)

Además de las skills absorbidas de Pocock, se agregan dos skills independientes al binario:

1. **golang-documentation** (`samber/cc-skills-golang@golang-documentation`): Documentación Go: doc comments, README, CONTRIBUTING, CHANGELOG, llms.txt. Modo Write y Review.
2. **golang-testing** (`samber/cc-skills-golang@golang-testing`): Testing en Go con mejores prácticas: table-driven tests, subtests, mocking, golden files.

Estas skills:
- Viven en `internal/opencode/skills/golang-documentation/SKILL.md` y `internal/opencode/skills/golang-testing/SKILL.md`
- Se embeben via `//go:embed` en `skills_embed.go`
- Se instalan en `.skills/` del proyecto al ejecutar `zyrocli install`
- Son de ámbito proyecto (no globales)

## Notas técnicas (para zyro-sdd-apply)
### Skills referenciados en HelixDB:
- Patrón: `mattpocock/skills@grill-with-docs` (276K installs, 135K ⭐)
- Patrón: `mattpocock/skills@to-prd` (245.7K installs, 135K ⭐)
- Patrón: `mattpocock/skills@improve-codebase-architecture` (280.2K installs, 135K ⭐)
- Library: `samber/cc-skills-golang@golang-documentation` (17.4K installs)
- Library: `samber/cc-skills-golang@golang-testing` (17.8K installs)

### Librerías validadas en F0:
- Stack actual cubre todo. Sin nuevas dependencias.
- Bubbles (ya transitiva) puede promoverse a directa si se usa textinput.

### Fuentes primarias leídas:
- https://github.com/mattpocock/skills/blob/main/skills/engineering/grill-with-docs/SKILL.md
- https://github.com/mattpocock/skills/blob/main/skills/engineering/to-prd/SKILL.md
- https://github.com/mattpocock/skills/blob/main/skills/engineering/setup-matt-pocock-skills/domain.md
- https://github.com/mattpocock/skills/blob/main/skills/engineering/codebase-design/SKILL.md
- https://github.com/mattpocock/skills/blob/main/CHANGELOG.md
- https://github.com/mattpocock/skills/blob/main/CONTEXT.md
- ADR-0001: Hard vs soft dependencies en skills
- README.md del repo mattpocock/skills
- docs/invocation.md (model-invoked vs user-invoked)
