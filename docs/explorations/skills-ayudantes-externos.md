# Skills y MCP Servers para Desarrollo Go como Agente IA

> **Fecha:** 2026-06-15
> **Propósito:** Investigación de herramientas externas (MCP servers, skills de OpenCode/skills.sh, paquetes npm) que un agente de IA puede usar para ser más efectivo trabajando con Go: build, test, lint, code review, gestión de proyectos.
> **Fuentes consultadas:** skills.sh, GitHub MCP servers, awesome-mcp-servers, Smithery.ai, npm, obra/superpowers

---

## Resumen Ejecutivo

Tras explorar skills.sh, el ecosistema MCP, npm y repositorios GitHub, la conclusión es:

| Categoría | ¿Hay skills/MCP listos? | Recomendación |
|-----------|------------------------|---------------|
| Build/run Go (go build, go run, go vet, go fmt) | ❌ No hay skills ni MCP específicos | Usar comandos bash directamente |
| Test Go (go test, coverage, benchmarks) | ❌ No hay herramientas Go-específicas | Usar bash + skill TDD de superpowers |
| Linting/SAST (golangci-lint, staticcheck) | ✅ **2 MCP servers excelentes** | Instalar ambos |
| Code Review (diff analysis, PR review) | ✅ **GitHub MCP Server oficial** | Instalar |
| Gestión proyectos (tasks, issues, commits) | ✅ **Git MCP Server + GitHub MCP** | Ambos son indispensables |

---

## 🔍 Hallazgos por Fuente

### 1. skills.sh (OpenCode Skills Registry)

**Problema detectado:** skills.sh **no tiene skills específicas para Go, build, test, ni lint**. El buscador devuelve el menú lateral sin resultados para términos como "golang", "go build", "go test", "golangci-lint".

Sin embargo, hay skills **transversales** muy útiles:

| Skill | Repo | ¿Qué hace? | Valoración |
|-------|------|-----------|------------|
| `test-driven-development` | obra/superpowers | Loop TDD: escribir test fallido → implementar → verificar → refactorizar. Funciona para Go. | ⭐ |
| `requesting-code-review` | obra/superpowers | Pre-review checklist antes de pedir revisión | ⭐ |
| `verification-before-completion` | obra/superpowers | Forzar pase de verificación antes de marcar tarea completa | ⭐ |
| `webapp-testing` | anthropics/skills | Patrones de testing web (unit, integration, e2e) | 👍 |
| `brainstorming` | obra/superpowers | Refinar ideas mediante preguntas socráticas | 👍 |
| `writing-plans` | obra/superpowers | Descomponer trabajo en tareas atómicas | 👍 |
| `executing-plans` | obra/superpowers | Ejecución por lotes con checkpoints humanos | 👍 |
| `subagent-driven-development` | obra/superpowers | Subagentes por tarea con revisión en 2 etapas | 👍 |
| `systematic-debugging` | obra/superpowers | Debug de 4 fases: causa raíz, defensa en profundidad | 👍 |

**Riesgos de seguridad:** Las skills de superpowers solo proporcionan instrucciones al agente; no ejecutan código arbitrario. Riesgo bajo.

---

### 2. MCP Servers para Desarrollo Go

#### 🥇 GOLD: MCP Servers Específicos para Calidad de Código Go

##### 1. golangci-lint-mcp (`wavilen/golangci-lint-mcp`) ⭐ INDISPENSABLE

| Campo | Detalle |
|-------|---------|
| **URL** | https://github.com/wavilen/golangci-lint-mcp |
| **Lenguaje** | Go (mcp-go framework) |
| **Transporte** | stdio |
| **Estrellas** | ~1 (proyecto nuevo, junio 2026) |
| **Licencia** | MIT |

**Qué hace:**
- MCP server que envuelve **golangci-lint** con 629 guías de fix incorporadas
- **5 tools MCP:**
  - `golangci_lint_run(path)` — ejecuta golangci-lint + devuelve guías de fix
  - `golangci_lint_parse(output)` — parsea JSON de golangci-lint existente
  - `golangci_lint_guide(linter, rule)` — lookup por linter/regla
  - `golangci_lint_list()` — lista linters soportados
  - `golangci_lint_summarize()` — resumen estratégico de issues
- Cubre: staticcheck (172 reglas), gocritic (108), revive (101), gosec (61), govet (35), testifylint (20) y más
- Incluye skill para OpenCode + plugin que limpia flags antes de ejecutar

**Instalación:**
```bash
# Go binary
go install github.com/wavilen/golangci-lint-mcp@latest

# OpenCode skill
npx @wavilen/golangci-lint-guide
```

**Configuración OpenCode:**
```json
{
  "mcp": {
    "golangci-lint": {
      "type": "local",
      "command": ["golangci-lint-mcp"]
    }
  }
}
```

**Riesgos de seguridad:**
- ✅ Ejecuta `golangci-lint` como subproceso — **ejecuta código arbitrario** (el linter corre análisis)
- ✅ Accede al **filesystem** del proyecto para leer archivos Go
- ❌ No hace llamadas de red (las guías van embebidas en el binario)
- Permisos: necesita leer/escribir archivos del proyecto

**Alternativas:** `mcp-server-go-quality` (ver abajo), o ejecutar `golangci-lint` directamente via bash

---

##### 2. mcp-server-go-quality (`afshinator/mcp-server-go-quality`) ⭐ INDISPENSABLE

| Campo | Detalle |
|-------|---------|
| **URL** | https://github.com/afshinator/mcp-server-go-quality |
| **Lenguaje** | Go |
| **Transporte** | stdio |
| **Estrellas** | ~0 (recién publicado) |
| **Licencia** | MIT |

**Qué hace:**
- **3 herramientas en 1 MCP:** golangci-lint + govulncheck + nilaway
- Ejecución **paralela** con timeouts independientes
- Output unificado: `Diagnostic[]` con `file:line:column` consistente
- **6 tools MCP:**
  - `run_code_checks` — los 3 checkers en paralelo
  - `run_lint` — solo golangci-lint
  - `run_vuln_check` — solo govulncheck (vulnerabilidades CVE)
  - `run_nil_check` — solo nilaway (nil panics)
  - `install_tools` — auto-instala los 3 binarios
- Auto-descubre `go.work` y `go.mod`
- Soporta configuración vía `.go-quality.yaml`

**Instalación:**
```bash
go install github.com/afshinator/mcp-server-go-quality/cmd/mcp-server-go-quality@latest
```

**Configuración OpenCode:**
```json
{
  "mcpServers": {
    "go-quality": {
      "command": "mcp-server-go-quality",
      "args": []
    }
  }
}
```

**Riesgos de seguridad:**
- ✅ Ejecuta 3 binarios diferentes (golangci-lint, govulncheck, nilaway) como subprocesos — **ejecuta código arbitrario**
- ✅ Accede al **filesystem** del proyecto
- ✅ `govulncheck` **descarga DB de vulnerabilidades** (llamada de red única)
- Permisos: necesita lectura de todo el proyecto, escritura no requerida

**Alternativas:** `golangci-lint-mcp` (más guías de fix), o ejecutar cada herramienta por separado

---

#### 🥈 Fundacionales: MCP Servers Generales Esenciales

##### 3. GitHub MCP Server (`github/github-mcp-server`) ⭐ INDISPENSABLE

| Campo | Detalle |
|-------|---------|
| **URL** | https://github.com/github/github-mcp-server |
| **Lenguaje** | Go |
| **Transporte** | stdio + HTTP remoto |
| **Estrellas** | ~30k |
| **Licencia** | MIT |

**Qué hace:**
- **El MCP server oficial de GitHub** — mantenido por GitHub
- Toolsets: `context`, `repos`, `issues`, `pull_requests`, `actions`, `code_security`, `dependabot`, `discussions`, `gists`, `git`, `users`, `orgs`, `projects`, `stargazers`, `labels`, `notifications`, `secret_protection`, `security_advisories`, `copilot`
- Útil para: **PR review**, **code analysis**, **issue management**, **Actions monitoring**, **Dependabot alerts**
- Se puede ejecutar local (Docker/binary) o remoto (con GitHub Copilot)

**Instalación:**
```bash
# Docker
docker run -i --rm -e GITHUB_PERSONAL_ACCESS_TOKEN ghcr.io/github/github-mcp-server

# Go build
go build ./cmd/github-mcp-server
```

**Configuración OpenCode:**
```json
{
  "mcpServers": {
    "github": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN", "ghcr.io/github/github-mcp-server"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "tu_pat_aqui"
      }
    }
  }
}
```

**Riesgos de seguridad:**
- ✅ **ALTO** — necesita GitHub Personal Access Token con scopes como `repo`
- ✅ Hace **llamadas de red** a la API de GitHub
- ✅ Acceso a **todos tus repositorios** según permisos del token
- ❌ No ejecuta código local
- ⚠️ No compartir el token; usar variables de entorno

**Alternativas:** GitHub CLI (`gh`) + bash, o la versión archivada `modelcontextprotocol/server-github`

---

##### 4. Git MCP Server (`modelcontextprotocol/server-git`) ⭐ INDISPENSABLE

| Campo | Detalle |
|-------|---------|
| **URL** | https://github.com/modelcontextprotocol/servers/tree/main/src/git |
| **Lenguaje** | Python |
| **Transporte** | stdio |
| **Estrellas** | ~87k (repo completo) |
| **Licencia** | MIT |

**Qué hace:**
- **12 tools MCP** para operaciones Git:
  - `git_status`, `git_diff_unstaged`, `git_diff_staged`, `git_diff`
  - `git_commit`, `git_add`, `git_reset`
  - `git_log`, `git_branch`, `git_create_branch`, `git_checkout`
  - `git_show`
- Perfecto para: diff analysis, status checks, commits automatizados, revisión de cambios

**Instalación:**
```bash
pip install mcp-server-git
# O con uv:
uvx mcp-server-git
```

**Configuración OpenCode:**
```json
{
  "mcpServers": {
    "git": {
      "command": "uvx",
      "args": ["mcp-server-git", "--repository", "/ruta/al/repo"]
    }
  }
}
```

**Riesgos de seguridad:**
- ✅ Accede al **filesystem** del repositorio
- ✅ Puede **modificar el repositorio** (commit, add, reset, checkout, create_branch)
- ❌ No ejecuta código arbitrario, solo comandos git
- ⚠️ Configurar solo repositorios que se deban tocar

**Alternativas:** Ejecutar comandos git directamente via bash (lo que este agente ya hace)

---

##### 5. Filesystem MCP Server (`modelcontextprotocol/server-filesystem`) 👍 ÚTIL

| Campo | Detalle |
|-------|---------|
| **URL** | https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem |
| **Lenguaje** | TypeScript |
| **Transporte** | stdio |
| **Estrellas** | ~87k (repo completo) |
| **Licencia** | MIT |

**Qué hace:**
- 15 tools MCP para operaciones de archivos:
  - `read_text_file`, `read_multiple_files`, `read_media_file`
  - `write_file`, `edit_file` (con dry-run, diff output)
  - `create_directory`, `list_directory`, `list_directory_with_sizes`
  - `move_file`, `search_files`, `directory_tree`, `get_file_info`
  - `list_allowed_directories`
- Sandboxing por directorios permitidos vía CLI o Roots
- Read/Write/Hints para que el agente sepa qué operaciones son destructivas

**Instalación:**
```bash
npx -y @modelcontextprotocol/server-filesystem /ruta/permitida
```

**Configuración OpenCode:**
```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/home/secko/Projects"]
    }
  }
}
```

**Riesgos de seguridad:**
- ✅ **ALTO** — puede leer/escribir cualquier archivo en los directorios permitidos
- ✅ `write_file` y `edit_file` son destructivos (sobreescriben archivos)
- ❌ No ejecuta código arbitrario
- ⚠️ Limitar directorios permitidos estrictamente

**Alternativas:** El agente ya tiene herramientas de archivo (Read, Write, Edit, Glob, Grep)

---

#### 🥉 Especializados: MCP Servers para Casos Específicos

##### 6. Go Process Inspector — gospy (`monsterxx03/gospy`) 👍 ÚTIL

| Campo | Detalle |
|-------|---------|
| **URL** | https://github.com/monsterxx03/gospy |
| **Lenguaje** | Go |
| **Transporte** | HTTP + MCP (streamable HTTP) |
| **Estrellas** | ~97 |
| **Licencia** | MIT |

**Qué hace:**
- Inspecciona procesos Go en ejecución sin instrumentación
- **4 tools MCP:**
  - `goroutines` — dump de gorutinas
  - `gomemstats` — estadísticas de memoria
  - `goruntime` — información del runtime
  - `pgrep` — encontrar PID por nombre
- TUI interactiva y API HTTP
- Útil para debugging en producción

**Instalación:**
```bash
go install github.com/monsterxx03/gospy@latest
```

**Riesgos de seguridad:**
- ✅ **ALTO** — requiere **root/sudo** para leer memoria de procesos
- ✅ Accede a memoria de procesos externos
- ❌ No modifica procesos (solo lectura)
- ⚠️ Solo para debugging, no para uso cotidiano

**Alternativas:** `pprof`, `dlv` (Delve debugger), `strace`

---

##### 7. Buildkite MCP 🤷 OPCIONAL

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/servers/buildkite |
| **Lenguaje** | - |
| **Transporte** | HTTP remoto |

**Qué hace:**
- Gestiona pipelines CI/CD en Buildkite
- Trigger builds, inspeccionar status, manage artifacts
- Útil si el proyecto usa Buildkite para CI

**Riesgos de seguridad:**
- ✅ Requiere API key de Buildkite
- ✅ Llamadas de red
- ❌ No ejecuta código local

---

### 3. npm MCP Packages para Go

Búsqueda en npm (`npm search mcp go`) encuentra principalmente:

| Paquete | Descripción | Relevancia |
|---------|------------|------------|
| `@zenfun510/codex-mcp-go` | NPM wrapper para codex-mcp-go | Baja (wrapper, no herramienta Go) |
| `@apet97/clockify-mcp-go` | Clockify MCP server en Go | Nula (gestión de tiempo, no Go) |
| `@j4flmao/go_blender_mcp` | Blender MCP en Go | Nula (Blender, no Go tools) |
| `@vkhanhqui/figma-mcp-go` | Figma MCP en Go | Nula (Figma, no Go) |

**Conclusión:** npm **no tiene paquetes MCP útiles** para desarrollo Go. Los proyectos relevantes se encuentran directamente en GitHub como binarios Go.

---

### 4. OpenCode Skills Existentes en el Proyecto

El proyecto ZyroAgentCLI ya tiene skills instaladas vía `helix.toml`:

- `zyro-orchestrator` — orquestador principal
- `zyro-sdd-*` — SDD phases (explore, propose, spec, design, tasks, apply, verify, archive)
- `zyro-phase-0-*` — Fase 0 (libraries, patterns)
- `zyro-skills-*` — Skills management (find, audit, apply)

**No hay skills de Go build/test/lint** configuradas actualmente.

---

## 📊 Tabla Comparativa Completa

| # | Herramienta | Tipo | Build | Test | Lint | Review | Proyectos | Instalación | Seguridad | Valoración |
|---|-------------|------|-------|------|------|--------|-----------|-------------|-----------|------------|
| 1 | **golangci-lint-mcp** | MCP Go | ❌ | ❌ | ✅ | ❌ | ❌ | `go install` | ⚠️ Ejecuta linter | ⭐ |
| 2 | **mcp-server-go-quality** | MCP Go | ❌ | ❌ | ✅ | ❌ | ❌ | `go install` | ⚠️ Ejecuta 3 bins + red (1x) | ⭐ |
| 3 | **GitHub MCP Server** | MCP Go | ❌ | ❌ | ❌ | ✅ | ✅ | Docker/binary | 🔴 PAT token, red | ⭐ |
| 4 | **Git MCP Server** | MCP Python | ❌ | ❌ | ❌ | ✅ | ✅ | `pip install` | ⚠️ Escribe en repo | ⭐ |
| 5 | **Filesystem MCP** | MCP TS | ❌ | ❌ | ❌ | ❌ | ❌ | `npx` | 🔴 Lee/escribe archivos | 👍 |
| 6 | **gospy (Go Inspector)** | MCP Go | ❌ | ❌ | ❌ | ❌ | ❌ | `go install` | 🔴 Root/sudo | 👍 |
| 7 | **test-driven-development** | Skill | ❌ | ✅ | ❌ | ❌ | ❌ | `npx skills` | Bajo (instrucciones) | ⭐ |
| 8 | **requesting-code-review** | Skill | ❌ | ❌ | ❌ | ✅ | ❌ | `npx skills` | Bajo (instrucciones) | ⭐ |
| 9 | **verification-before-completion** | Skill | ❌ | ❌ | ❌ | ❌ | ❌ | `npx skills` | Bajo (instrucciones) | ⭐ |
| 10 | **webapp-testing** | Skill | ❌ | ✅ | ❌ | ❌ | ❌ | `npx skills` | Bajo (instrucciones) | 👍 |
| 11 | **Buildkite MCP** | MCP remoto | ✅ | ✅ | ❌ | ❌ | ❌ | Smithery | ⚠️ API key, red | 🤷 |
| 12 | **systematic-debugging** | Skill | ❌ | ❌ | ❌ | ❌ | ❌ | `npx skills` | Bajo (instrucciones) | 👍 |
| 13 | **writing-plans** | Skill | ❌ | ❌ | ❌ | ❌ | ✅ | `npx skills` | Bajo (instrucciones) | 👍 |
| 14 | **subagent-driven-development** | Skill | ❌ | ❌ | ❌ | ❌ | ✅ | `npx skills` | Bajo (instrucciones) | 👍 |

---

## 🛠️ Stack Recomendado para Agente Go

Basado en el análisis, esta es la configuración óptima:

### Esencial (empezar aquí):

```json
{
  "mcpServers": {
    "go-quality": {
      "command": "mcp-server-go-quality",
      "args": []
    },
    "github": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN", "ghcr.io/github/github-mcp-server"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_PAT}"
      }
    },
    "git": {
      "command": "uvx",
      "args": ["mcp-server-git", "--repository", "/home/secko/Projects/ZyroAgentCLI"]
    }
  }
}
```

### Skills de superpowers (vía skills.sh):

```bash
npx skills add obra/superpowers/test-driven-development
npx skills add obra/superpowers/requesting-code-review
npx skills add obra/superpowers/verification-before-completion
npx skills add obra/superpowers/writing-plans
npx skills add obra/superpowers/executing-plans
npx skills add obra/superpowers/systematic-debugging
```

### Adicional (opcional):

```bash
# Linter con guías detalladas (complementa mcp-server-go-quality)
go install github.com/wavilen/golangci-lint-mcp@latest
npx @wavilen/golangci-lint-guide

# Inspección de procesos Go
go install github.com/monsterxx03/gospy@latest
```

---

## 🚫 Brechas Detectadas

No existen MCP servers ni skills para:

1. **`go build` / `go run` / `go vet` / `go fmt`** — Hay que usar bash directamente
2. **`go test` con coverage y benchmarks** — No hay MCP especializado; usar bash + TDD skill
3. **`staticcheck` standalone** — Viene incluido en golangci-lint, pero no hay server dedicado
4. **Benchmarks de rendimiento** — No hay herramienta MCP para `go test -bench`
5. **Generación de código Go** — No hay server especializado (ej: `stringer`, `mockgen`)

**Solución:** El agente ya puede ejecutar estos comandos via `bash`. La skill de TDD de superpowers + los MCP de calidad de código cubren el 80% de las necesidades.

---

## 📝 Recomendaciones Finales

### Prioridad Alta (instalar ahora):
1. ✅ **mcp-server-go-quality** — Unifica linting, seguridad y nil-check
2. ✅ **GitHub MCP Server** — PR review, issues, Actions
3. ✅ **Git MCP Server** — diff analysis, commits
4. ✅ **test-driven-development** (superpowers) — TDD loop
5. ✅ **requesting-code-review** (superpowers) — Code review checklist

### Prioridad Media (instalar después):
6. ⏳ **golangci-lint-mcp** — Guías de fix detalladas (complementa al #1)
7. ⏳ **verification-before-completion** (superpowers)
8. ⏳ **writing-plans** (superpowers) + **executing-plans**
9. ⏳ **systematic-debugging** (superpowers)

### Prioridad Baja (solo si se necesita):
10. 📌 **gospy** — Debugging de procesos en producción
11. 📌 **Buildkite MCP** — Si usas Buildkite para CI
12. 📌 **webapp-testing** — Testing web general (no Go-específico)

---

## 🔗 Referencias

- skills.sh: https://skills.sh
- obra/superpowers: https://github.com/obra/superpowers
- MCP Servers oficiales: https://github.com/modelcontextprotocol/servers
- awesome-mcp-servers: https://github.com/punkpeye/awesome-mcp-servers
- Smithery.ai: https://smithery.ai
- golangci-lint-mcp: https://github.com/wavilen/golangci-lint-mcp
- mcp-server-go-quality: https://github.com/afshinator/mcp-server-go-quality
- GitHub MCP Server: https://github.com/github/github-mcp-server
- gospy: https://github.com/monsterxx03/gospy

---

## Hallazgos Adicionales - Testing y Code Review

> **Fecha:** 2026-06-15
> **Propósito:** Segunda ronda de investigación enfocada en MCP servers externos para Testing en Go, Code Review automatizado, SAST, Coverage Reports y Benchmarking.
> **Fuentes consultadas:** awesome-mcp-servers (punkpeye), Smithery.ai, GitHub MCP Server oficial, modelcontextprotocol/servers, búsquedas adicionales en GitHub.

---

### Resumen

| Categoría | MCP Servers Encontrados | Recomendación |
|-----------|------------------------|---------------|
| Testing (unit, integration, e2e) | bugAgent (109 tools), Postman, Test My Vibes, Mimiq | bugAgent es el más completo |
| Code Review automatizado | Code Sentinel, SpecLock, GitHub MCP (PR review) | Code Sentinel + GitHub MCP |
| SAST (Static Analysis Security) | Semgrep MCP, Trust, Security Auditor, CodeVulnerability, Compuute Scanner | Semgrep MCP (el estándar) |
| Coverage Reports | ❌ No hay MCP específico para code coverage | Usar `go test -cover` via bash |
| Benchmarking | ❌ No hay MCP específico para benchmarks | Usar `go test -bench` via bash |
| Seguridad Supply Chain | safedep/vet, OSV MCP, Agent Bom, Snyk MCP | safedep/vet (Go native) |
| Code Intelligence | agent-lsp (65 tools, 30 langs), gospy (Go inspector) | agent-lsp (revolucionario para Go) |

---

### 1. bugAgent — QA Platform Completa ⭐ RECOMENDADO

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/bugagent/bugagent-mcp |
| **Repo** | https://github.com/bugagent/bugagent-mcp-quickstart |
| **Web** | https://bugagent.com |
| **Tipo** | MCP Server remoto (Smithery managed, verificado) |
| **Uptime** | 100% |
| **Latencia** | 244ms p50 |
| **Tools expuestas** | 109 |

**Descripción:**
Plataforma de QA para equipos AI-native. Proporciona bug reports, test management, automatización con Playwright, security scanning y performance testing — todo desde 109 tools MCP.

**Tools principales:**
- `get_bug_report` — obtener reportes de bugs
- Playwright automation tools — tests e2e automatizados
- Security scanning tools — análisis de seguridad
- Performance testing tools — benchmarks de rendimiento
- Test management tools — gestión de casos de prueba

**Instalación:**
```bash
npx smithery mcp add bugagent/bugagent-mcp
# Requiere API key de app.bugagent.com
```

**Análisis de seguridad:**
- ✅ Requiere API key (`ba_live_...`)
- ✅ Servicio remoto — llamadas de red a API de bugAgent
- ❌ No ejecuta código local
- ✅ Verificado por Smithery
- ⚠️ Datos de código se envían a servidores externos para análisis

**Veredicto:** ⚠️ Precaución — Muy potente pero requiere enviar código a servidores externos. No ideal para proyectos sensibles. Excelente para QA en general.

---

### 2. Code Sentinel — Code Review Automatizado ⭐ RECOMENDADO

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/salrad/code-sentinel |
| **NPM** | https://www.npmjs.com/package/code-sentinel-mcp |
| **Tipo** | MCP Server remoto |
| **Uptime** | 99.86% |
| **Latencia** | 985ms p50 |
| **Tools expuestas** | 7 |

**Descripción:**
Expone vulnerabilidades de seguridad, constructos engañosos y código incompleto antes de que llegue a producción. Revela patrones arquitectónicos y de diseño con guías claras para mejorar consistencia y mantenibilidad. Genera reportes visuales concisos.

**Tools:**
- `analyze_code` — análisis general de calidad (216 llamadas)
- `check_security` — escaneo de vulnerabilidades (126)
- `check_placeholders` — detecta TODOs, FIXMEs, placeholders (135)
- `check_deceptive_patterns` — detecta patrones engañosos (122)
- `analyze_patterns` — análisis de patrones (69)
- `analyze_design_patterns` — patrones de diseño (57)
- `generate_report` — genera reporte visual (56)

**Instalación:**
```bash
npx smithery mcp add salrad/code-sentinel
# No requiere API key (según datos disponibles)
```

**Análisis de seguridad:**
- ✅ Servicio remoto — código se envía para análisis
- ✅ No requiere permisos de filesystem local
- ❌ Moderado — el código analizado viaja a servidores externos
- ✅ Clientes usados: Claude Code, Claude.ai, VS Code

**Veredicto:** ⚠️ Precaución — Bueno para code review pero envía código a externos.

---

### 3. Semgrep MCP — SAST Scanning (El Estándar) ⭐ INDISPENSABLE

| Campo | Detalle |
|-------|---------|
| **URL** | https://github.com/semgrep/mcp |
| **Repo principal** | https://github.com/semgrep/semgrep (migrado) |
| **Lenguaje** | Python |
| **Transporte** | stdio, streamable-http, SSE |
| **Estrellas** | 673 (repo antiguo), 11k+ (semgrep) |
| **Licencia** | MIT |

**Descripción:**
El MCP server oficial de Semgrep para escanear código en busca de vulnerabilidades de seguridad. Semgrep es una herramienta de análisis estático rápida y determinística que entiende semánticamente muchos lenguajes y viene con más de 5,000 reglas.

**Tools:**
1. `security_check` — Escanea código por vulnerabilidades
2. `semgrep_scan` — Escanea archivos con config string
3. `semgrep_scan_with_custom_rule` — Escanea con reglas personalizadas
4. `get_abstract_syntax_tree` — Output del AST del código
5. `semgrep_findings` — Obtiene findings desde Semgrep AppSec Platform (requiere token)
6. `supported_languages` — Lista lenguajes soportados
7. `semgrep_rule_schema` — Schema JSON de reglas Semgrep

**Prompts:**
- `write_custom_semgrep_rule` — Ayuda a escribir reglas Semgrep

**Recursos:**
- `semgrep://rule/schema` — Schema de sintaxis YAML de reglas
- `semgrep://rule/{rule_id}/yaml` — Regla completa en YAML

**Instalación:**
```bash
# Con uv (recomendado)
uvx semgrep-mcp

# Con pipx
pipx install semgrep-mcp

# Con Docker
docker run -i --rm ghcr.io/semgrep/mcp -t stdio

# Opcional: token para Semgrep AppSec Platform
export SEMGREP_APP_TOKEN=<token>
```

**Configuración OpenCode:**
```json
{
  "mcpServers": {
    "semgrep": {
      "command": "uvx",
      "args": ["semgrep-mcp"]
    }
  }
}
```

**Análisis de seguridad:**
- ✅ Ejecuta Semgrep localmente — **NO envía código a externos** (modo local)
- ✅ Acceso al filesystem para leer archivos
- ✅ Opcional: conexión a Semgrep Cloud (solo si se configura token)
- ❌ `semgrep_scan_with_custom_rule` puede ejecutar reglas arbitrarias
- ✅ Hosted server en mcp.semgrep.ai (experimental) — ese SÍ envía código

**Veredicto:** ✅ Seguro (modo local) — El estándar de la industria para SAST. Corre localmente, no envía código.

---

### 4. Trust — DAST + SAST Scanning

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/trust-security/scanner |
| **Web** | https://www.trust-scan.me |
| **Tipo** | MCP Server remoto |
| **Uptime** | 99.58% |
| **Latencia** | 1.8s p50 |
| **Tools expuestas** | 9 |

**Descripción:**
Detecta vulnerabilidades en sitios web activos y repositorios GitHub usando escaneo DAST y SAST automatizado. Identifica secretos expuestos, dependencias inseguras y patrones de código explotables.

**Tools:**
- `analyze_code_security` — análisis de seguridad de código
- `check_secrets` — detecta secretos expuestos
- `scan_repo_and_wait` — escanea repo y espera resultados
- `scan_and_wait` — escanea URL y espera
- `get_scan_result` — obtiene resultados de escaneo
- `get_fix_plan` — plan de remediación
- `scan_url` — escanea URL individual

**Instalación:**
```bash
npx smithery mcp add trust-security/scanner
```

**Análisis de seguridad:**
- ✅ Servicio remoto — envía URLs/repos a escanear
- ✅ No requiere acceso local al filesystem
- ⚠️ Cliente principal: OpenCode (159 llamadas)
- ❌ El código/repos se analizan en servidores externos

**Veredicto:** ⚠️ Precaución — Útil para audit externas, pero confía el código a terceros.

---

### 5. Security Auditor — GitHub Security Scanner ⭐

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/eren-solutions/mcp-security-audit |
| **Repo** | https://github.com/eren-solutions/mcp-security-audit |
| **Tipo** | MCP Server remoto |
| **Uptime** | 99.72% |
| **Latencia** | 491ms p50 |
| **Puntuación** | 100/100 |
| **Tools expuestas** | 4 |

**Descripción:**
Auditor de seguridad de código impulsado por IA. Escanea repositorios GitHub en busca de vulnerabilidades con referencias OWASP/CWE y guías de remediación.

**Tools:**
- `security_scan` — escanea un repo en busca de vulnerabilidades
- `audit_list` — lista auditorías realizadas
- `audit_stats` — estadísticas de auditorías
- `audit_status` — estado de una auditoría

**Instalación:**
```bash
npx smithery mcp add eren-solutions/mcp-security-audit
```

**Análisis de seguridad:**
- ✅ Servicio remoto — escanea repos públicos/privados vía GitHub
- ✅ No requiere credenciales locales
- ⚠️ 100/100 score (perfecto, pero es nuevo con pocas llamadas)
- ❌ El repo se analiza en servidores externos

**Veredicto:** ⚠️ Precaución — Buen complemento para Semgrep pero 100% remoto.

---

### 6. safedep/vet — Malicious Package Detection (Go Native) ⭐ INDISPENSABLE

| Campo | Detalle |
|-------|---------|
| **URL** | https://github.com/safedep/vet |
| **Lenguaje** | Go (95.7%) |
| **Estrellas** | 1,100+ |
| **Licencia** | Apache 2.0 |
| **MCP Server** | built-in (docs/mcp.md) |

**Descripción:**
Protección contra paquetes open source maliciosos. Detecta malware zero-day mediante análisis estático y dinámico. Analiza uso real de dependencias para priorizar riesgos reales. Políticas de seguridad como código con expresiones CEL.

**Soporta ecosistemas:** npm, PyPI, Maven, **Go**, Ruby, Rust, PHP, Docker, OCI, SBOM (CycloneDX, SPDX)

**Características clave:**
- Detección de paquetes maliciosos en tiempo real
- Análisis de vulnerabilidades con evidencia de uso real
- Políticas como código (CEL expressions)
- Escaneo de malware en GitHub Actions, VS Code extensions, containers
- **Modo agente AI** integrado
- **MCP server** integrado (docs/mcp.md)
- Escaneo de skills de agente: `vet scan --agent-skill <owner/repo>`

**Instalación:**
```bash
# Homebrew (recomendado)
brew install safedep/tap/vet

# npm
npm install -g @safedep/vet

# Go
go install github.com/safedep/vet@latest

# Docker
docker run --rm ghcr.io/safedep/vet:latest version
```

**Uso como MCP:**
```bash
vet mcp
# Expone tools para análisis de dependencias, vulnerabilidades y malware
```

**Análisis de seguridad:**
- ✅ Corre **localmente** — no envía código a externos (telemetría anónima opcional)
- ✅ Acceso al filesystem para leer go.mod, go.sum, etc.
- ✅ Descarga DB de vulnerabilidades (llamada de red única)
- ⚠️ Opcional: SafeDep Cloud para detección avanzada (requiere API key)
- ✅ SLSA 3, OpenSSF Scorecard

**Veredicto:** ✅ Seguro — Corre local, es Go nativo, excelente para seguridad de supply chain.

---

### 7. OSV MCP — Open Source Vulnerability Database ⭐

| Campo | Detalle |
|-------|---------|
| **URL** | https://github.com/StacklokLabs/osv-mcp |
| **Lenguaje** | Go (100%) |
| **Estrellas** | 34 |
| **Licencia** | Apache 2.0 |
| **Transporte** | SSE, streamable-http |

**Descripción:**
Servidor MCP que proporciona acceso a la base de datos OSV (Open Source Vulnerabilities). Permite consultar vulnerabilidades por paquete, versión o commit. Ideal para integrar chequeos de seguridad en el pipeline del agente.

**Tools:**
1. `query_vulnerability` — consulta vulnerabilidades para un paquete/versión/commit
2. `query_vulnerabilities_batch` — consulta batch para múltiples paquetes
3. `get_vulnerability` — detalles de una vulnerabilidad específica por ID

**Soporta ecosistemas:** Go, PyPI, npm, Maven, y todos los soportados por OSV

**Instalación:**
```bash
# Build desde fuente
git clone https://github.com/StacklokLabs/osv-mcp.git
cd osv-mcp && task build

# Con ToolHive (recomendado)
thv run osv
```

**Configuración:**
```json
{
  "mcpServers": {
    "osv": {
      "command": "/path/to/osv-mcp-server",
      "args": []
    }
  }
}
```

**Análisis de seguridad:**
- ✅ Servidor local — no envía datos a externos
- ✅ Consultas a API pública de OSV (osv.dev) — solo envía nombres de paquetes
- ✅ No ejecuta código arbitrario
- ✅ No requiere permisos especiales de filesystem
- ❌ Dependencia de API externa (osv.dev)

**Veredicto:** ✅ Seguro — Consulta vulnerabilidades de paquetes Go sin riesgo. Excelente complemento para safedep/vet.

---

### 8. Agent Bom — Supply Chain Security Scanner

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/agent-bom/agent-bom |
| **Repo** | https://github.com/msaad00/agent-bom |
| **Tipo** | MCP Server remoto |
| **Uptime** | 99.58% |
| **Latencia** | 147ms p50 |
| **Puntuación** | 94/100 |

**Descripción:**
Escáner de seguridad de supply chain para AI. 7 tools para escaneo CVE, mapeo de blast radius, generación de SBOM, postura de cumplimiento (OWASP/ATLAS/NIST), enforce de políticas y planificación de remediación.

**Tools:**
- `check` — verifica vulnerabilidades (6 llamadas)
- `scan` — escanea proyectos
- `inventory` — inventario de dependencias (5)
- `registry_lookup` — busca en registros (5)
- `where` — localiza dependencias

**Instalación:**
```bash
npx smithery mcp add agent-bom/agent-bom
```

**Análisis de seguridad:**
- ✅ Read-only — "no credentials accessed" según documentación
- ✅ Servicio remoto pero solo consulta
- ✅ Baja latencia (147ms)
- ❌ Puede enviar nombres de dependencias a externos

**Veredicto:** ⚠️ Precaución — Útil pero remoto. Preferir safedep/vet (local) para proyectos sensibles.

---

### 9. agent-lsp — Code Intelligence Infrastructure (65 tools, 30 languages) ⭐ REVOLUCIONARIO

| Campo | Detalle |
|-------|---------|
| **URL** | https://github.com/blackwell-systems/agent-lsp |
| **Lenguaje** | Go (97.8%) |
| **Estrellas** | 58 |
| **Licencia** | MIT |
| **Transporte** | stdio, HTTP+SSE |
| **Tools** | 65 |
| **Skills** | 24 workflows |

**Descripción:**
**La infraestructura de inteligencia de código definitiva para agentes AI.** Orquesta servidores LSP reales (gopls, rust-analyzer, tsserver, pyright, etc.) y expone 65 tools MCP para navegación, análisis, refactorización y edición especulativa de código.

**Soporta 30 lenguajes (verificados en CI):** Go, Python, TypeScript, Rust, Java, C, C++, C#, Ruby, PHP, Kotlin, Swift, Scala, Zig, Lua, Elixir, Gleam, Clojure, Dart, Terraform, Nix, Prisma, SQL, MongoDB y más.

**Tools destacadas para Go:**
- `blast_radius` — encuentra todo lo que afecta un cambio (callers, exports, test vs non-test)
- `find_callers` / `find_references` — referencias semánticas (no grep)
- `go_to_implementation` — implementaciones de interfaces (type-checked)
- `simulate_edit` — edición especulativa en memoria antes de tocar disco
- `preview_edit` — preview del impacto diagnóstico de cualquier edit
- `safe_apply_edit` — preview + apply combinado, solo si net_delta == 0
- `get_diagnostics` — diagnóstico LSP en vivo
- `get_abstract_syntax_tree` — AST completo
- `get_symbols` — símbolos del workspace
- `type_hierarchy` — jerarquía de tipos
- `call_hierarchy` — jerarquía de llamadas

**Skills (24 workflows):**
- `/lsp-refactor` — impacto → preview → apply → verify → test
- `/lsp-safe-edit` — preview → diagnostic diff → apply if safe
- `/lsp-verify` — diagnostics → build → tests
- `/lsp-impact` — blast-radius analysis
- `/lsp-dead-code` — detecta exports sin referencias
- `/lsp-test-correlation` — encuentra y ejecuta tests que cubren un archivo editado
- `/lsp-inspect` — auditoría completa: dead symbols, test coverage, error handling, doc drift, concurrency safety
- `/lsp-concurrency-audit` — auditoría de concurrencia campo por campo
- `/lsp-fix-all` — aplica code actions para todos los diagnósticos

**Instalación:**
```bash
# Script oficial
curl -fsSL https://raw.githubusercontent.com/blackwell-systems/agent-lsp/main/install.sh | sh

# Homebrew
brew install blackwell-systems/tap/agent-lsp

# Go
go install github.com/blackwell-systems/agent-lsp/cmd/agent-lsp@latest

# Pip / npm
pip install agent-lsp
npm install -g @blackwell-systems/agent-lsp
```

**Configuración:**
```json
{
  "mcpServers": {
    "lsp": {
      "command": "agent-lsp",
      "args": ["go:gopls"]
    }
  }
}
```

**Análisis de seguridad:**
- ✅ Corre **100% local** — no envía código a ningún externo
- ✅ Ejecuta gopls como subproceso para inteligencia de código
- ✅ `simulate_edit` y `preview_edit` **no tocan disco** — operaciones en memoria
- ✅ `safe_apply_edit` solo escribe si no introduce nuevos errores
- ⚠️ `apply_edit`, `write_file` tools pueden modificar archivos
- ✅ Phase enforcement — bloquea tool calls fuera de orden
- ✅ Output en GCF (Graph Compact Format) — 30-84% menos tokens que JSON

**Veredicto:** ✅ Seguro — La herramienta más revolucionaria para desarrollo de código con agentes AI. Corre 100% local. **Altamente recomendado para cualquier proyecto Go.**

---

### 10. SpecLock — AI Constraint Engine

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/sgroy10/speclock |
| **Tipo** | MCP Server |
| **Tools** | 42 |

**Descripción:**
Motor de restricciones para AI con AI Patch Firewall. Proporciona Patch Gateway con veredictos ALLOW/WARN/BLOCK, diff-native review con 10 señales ponderadas, Spec Compiler, Code Graph, constraints tipados. Works with Claude Code, Cursor, Windsurf, Cline.

**Tools destacadas:**
- Patch Gateway (ALLOW/WARN/BLOCK verdicts)
- Diff-native review (10 scored signals, hard escalation rules)
- Spec Compiler — compila especificaciones a constraints
- Code Graph — grafo de código para análisis

**Instalación:**
```bash
npx smithery mcp add sgroy10/speclock
```

**Análisis de seguridad:**
- ✅ 1073 tests (alta calidad)
- ✅ Open source
- ⚠️ Puede bloquear cambios del agente (WARN/BLOCK)
- ❌ Requiere evaluación para entender permisos exactos

**Veredicto:** ⚠️ Precaución — Interesante para equipos que quieren enforce de políticas, pero requiere configuración cuidadosa.

---

### 11. Postman MCP — API Testing ⭐

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/postman |
| **Tipo** | MCP Server remoto (verificado) |
| **Uptime** | 100% |
| **Latencia** | 155ms p50 |
| **Tools expuestas** | 100+ |

**Descripción:**
Testea y debuggea APIs con collections, environments y monitors. Ejecuta requests, inspecciona respuestas y colabora en documentación de API.

**Tools principales:**
- `getCollection`, `createCollectionRequest`, `updateCollectionRequest`
- `getWorkspaces`, `getEnvironments`
- `runCollection` — ejecuta tests de collection
- `getMonitors`, `createMonitor`
- `getApis`, `createApi`

**Instalación:**
```bash
npx smithery mcp add postman
# Requiere cuenta de Postman
```

**Análisis de seguridad:**
- ✅ Servicio remoto oficial de Postman
- ✅ Requiere autenticación Postman
- ⚠️ Las APIs bajo test se acceden desde los servidores de Postman
- ❌ Muchas tools (100+) pueden ser abrumadoras

**Veredicto:** ⚠️ Precaución — Excelente para testing de APIs REST, pero es remoto y requiere cuenta Postman.

---

### 12. Buildkite MCP — CI/CD Pipeline Management

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/buildkite |
| **Tipo** | MCP Server remoto (verificado) |
| **Uptime** | 99.17% |
| **Latencia** | 278ms p50 |

**Descripción:**
Plataforma CI/CD para pipelines de build escalables. Trigger builds, inspecciona status de pipelines y gestiona artifacts de build.

**Instalación:**
```bash
npx smithery mcp add buildkite
# Requiere API key de Buildkite
```

**Análisis de seguridad:**
- ✅ Requiere API key de Buildkite
- ✅ Llamadas de red a API de Buildkite
- ❌ No ejecuta código local

**Veredicto:** ⚠️ Precaución — Útil solo si usas Buildkite como CI. Para GitHub Actions, el GitHub MCP Server es mejor.

---

### 13. Test My Vibes — Agent Testing Infrastructure

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/tony-jqzi/testmyvibes |
| **Web** | https://testmyvibes.com |
| **Tipo** | MCP Server remoto |

**Descripción:**
Infraestructura de testing MCP-native para coding agents. Submit URL + goal, recibe action trail + bugs + screenshots + WebM video. 12 personalidades de evaluación AI (Marketing Strategist, Legal Eagle, MCP Auditor, Skeptical CTO, etc.).

**Instalación:**
```bash
npx smithery mcp add tony-jqzi/testmyvibes
```

**Análisis de seguridad:**
- ✅ Testing de UI/UX — útil para validar interfaces
- ❌ Envía URLs y contenido a servidores externos
- ⚠️ Evaluación por IA externa

**Veredicto:** ⚠️ Precaución — Interesante para testing de agentes pero completamente remoto.

---

### 14. Compuute MCP Security Scanner — MCP Server Scanner

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/daniel-abbay/compuute-scan-api |
| **Web** | https://scan.compuute.se |
| **Tipo** | MCP Server remoto |

**Descripción:**
Escáner de seguridad estático para MCP servers. POST un GitHub URL público, obtiene severity counts, score y findings con file+line. 37 reglas para TypeScript, JavaScript, Python, **Go**, Rust, C#, Java y Kotlin.

**Detecciones:**
- Argument injection para npx/uvx/pipx/pnpx (CWE-88)
- CVEs conocidos en 40+ paquetes populares
- L0 discovery (transport, tool inventory, dependency pinning)

**Instalación:**
```bash
npx smithery mcp add daniel-abbay/compuute-scan-api
# Gratis sin API key; $0.10 USDC por scan vía x402 para premium
```

**Análisis de seguridad:**
- ✅ Escáner especializado para MCP servers
- ✅ Soporta Go (entre otros lenguajes)
- ⚠️ 90% raw false-positive rate según documentación
- ❌ Remoto — envía URL pública de GitHub
- ✅ Free tier sin API key

**Veredicto:** ⚠️ Precaución — Útil para audit de MCP servers pero alto ratio de falsos positivos y remoto.

---

### 15. SkillAudit — Security Scanner for AI Skills

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/megamind-0x/skillaudit |
| **Web** | https://skillaudit.vercel.app |
| **Tipo** | MCP Server remoto |
| **Tools** | Múltiples para auditoría de skills |

**Descripción:**
Escáner de seguridad para AI agent skills y MCP servers. Detecta credential theft, data exfiltration, prompt injection, obfuscated code y 80+ patrones de amenaza. API gratuita + CLI (`npx skillaudit`).

**Instalación:**
```bash
npx smithery mcp add megamind-0x/skillaudit
# O CLI directo:
npx skillaudit
```

**Análisis de seguridad:**
- ✅ Detecta 80+ patrones de amenaza en skills
- ✅ CLI gratuita
- ⚠️ Servicio remoto
- ❌ Los skills auditados se analizan externamente

**Veredicto:** ⚠️ Precaución — Muy útil para auditar skills antes de instalarlas, pero es remoto.

---

### 16. Shrike Security — Runtime Security para AI Agents

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/shrike-security/shrike-mcp |
| **Tipo** | MCP Server remoto |
| **Tools** | 12 |

**Descripción:**
Seguridad en tiempo de ejecución para AI agents. 12 tools MCP detectan y bloquean prompt injection, data exfiltration, privilege escalation y multi-turn attacks. Defensa en capas desde pattern matching hasta análisis LLM.

**Instalación:**
```bash
npx smithery mcp add shrike-security/shrike-mcp
```

**Análisis de seguridad:**
- ✅ Protección activa contra ataques a agents
- ✅ Free tier incluido
- ⚠️ Servicio remoto
- ❌ Intercepta prompts/respuestas — posible latencia

**Veredicto:** ⚠️ Precaución — Útil para hardening de agents pero introduce dependencia externa.

---

### 17. CodeVulnerability — OWASP Auditor (M2MCent)

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/evozim-hv/codevulnerability |
| **Web** | https://codevulnerability-mcp.vercel.app |
| **Tipo** | MCP Server premium (x402 micropagos) |
| **Costo** | $0.12 USDC por tool execution |

**Descripción:**
Auditor DevSecOps OWASP en tiempo real. Escanea código en busca de vulnerabilidades OWASP Top 10. Servicio premium con pago por uso vía x402 (USDC en Base L2).

**Instalación:**
```bash
npx smithery mcp add evozim-hv/codevulnerability
# Pago por uso: $0.12 USDC/tool execution
```

**Análisis de seguridad:**
- ⚠️ Servicio premium de pago
- ⚠️ Micropagos en crypto (x402)
- ❌ Código se envía a servidores externos
- ❌ Sin información de qué pasa con el código después del análisis

**Veredicto:** ❌ Riesgoso — Caro, remoto, crypto payments. Preferir Semgrep MCP (gratis + local).

---

### 18. CleanCode AI — Technical Debt Refactoring

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/evozim-hv/cleancode-ai |
| **Web** | https://cleancode-ai-mcp.vercel.app |
| **Tipo** | MCP Server premium (x402 micropagos) |
| **Costo** | $0.10 USDC por tool execution |

**Descripción:**
Refactorizador de deuda técnica y tipado. Analiza código y sugiere mejoras de calidad. Servicio premium con pago por uso.

**Instalación:**
```bash
npx smithery mcp add evozim-hv/cleancode-ai
```

**Análisis de seguridad:**
- ⚠️ Servicio premium de pago
- ❌ Código se envía a servidores externos
- ❌ Micropagos crypto

**Veredicto:** ❌ Riesgoso — Mismos problemas que CodeVulnerability. Preferir code-sentinel o agent-lsp.

---

### 19. Quality Coach — Code Quality Improvement

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/amcharrett-6als/qualitycoach |
| **Web** | https://qualitycoach.io |
| **Tipo** | MCP Server remoto |
| **Uptime** | 99.31% |
| **Latencia** | 2.2s p50 |

**Descripción:**
Quality Coach MCP server que proporciona consejo táctico sobre cómo mejorar un repositorio de código para iniciativas de mejora de calidad. Diseñado para medir la mejora apropiadamente.

**Instalación:**
```bash
npx smithery mcp add amcharrett-6als/qualitycoach
# Requiere API key (x-api-key header)
```

**Análisis de seguridad:**
- ✅ Enfocado en calidad de código
- ⚠️ Requiere API key
- ❌ Remoto — código se envía a qualitycoach.io
- ❌ 0 tool calls registradas (poco usado)

**Veredicto:** ⚠️ Precaución — Concepto interesante pero parece poco adoptado y es remoto.

---

### 20. Mimiq — UX Testing Agent

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/mimiqai/mimiq |
| **Tipo** | MCP Server remoto |
| **Tools** | Múltiples |
| **Usos** | 274 |

**Descripción:**
Tu AI coding agent tests pages, copy y flows en simulated users mientras builds. Detecta bad copy, confusing UX y conversion killers antes de que usuarios reales lo hagan.

**Instalación:**
```bash
npx smithery mcp add mimiqai/mimiq
```

**Análisis de seguridad:**
- ✅ Testing de UX automatizado
- ❌ Remoto — envía URLs y contenido
- ⚠️ Simula usuarios para testing

**Veredicto:** ⚠️ Precaución — Útil para testing de UX pero completamente remoto.

---

### 21. GitHub MCP Server — Tools de Code Scanning y PR Review (Detalle Adicional)

El GitHub MCP Server oficial (ya documentado arriba) expone **toolsets específicos para testing y code review** que merecen atención adicional:

**Toolset `code_security`:**
- `get_code_scanning_alert` — obtiene alerta de code scanning
- `list_code_scanning_alerts` — lista alertas con filtros por severidad, estado, tool_name
- Filtrable por: `severity` (critical, high, medium, low), `state` (open, closed)

**Toolset `pull_requests`:**
- Tools para revisar, mergear y gestionar PRs
- Comentar en PRs, ver diffs, aprobar cambios

**Toolset `dependabot`:**
- `get_dependabot_alert` — obtiene alerta de Dependabot
- `list_dependabot_alerts` — lista alertas con filtros

**Toolset `actions`:**
- `get_job_logs` — obtiene logs de jobs de Actions (con `failed_only` y `tail_lines`)
- `actions_list` — lista workflows, runs, jobs, artifacts
- `actions_get` — detalles de recursos de Actions
- `actions_run_trigger` — trigger de workflows

**Toolset `copilot` (Insiders):**
- `request_copilot_review` — solicita review de Copilot en un PR
- `assign_copilot_to_issue` — asigna Copilot a un issue

**Veredicto:** ✅ Seguro (con token limitado) — El GitHub MCP Server es esencial para code review y code scanning si usas GitHub.

---

### 22. modelcontextprotocol/server-git — Git Operations (Referencia)

| Campo | Detalle |
|-------|---------|
| **URL** | https://github.com/modelcontextprotocol/servers/tree/main/src/git |
| **Lenguaje** | Python |
| **Instalación** | `pip install mcp-server-git` o `uvx mcp-server-git` |
| **Tools** | 12 |

**Tools:**
- `git_status`, `git_diff_unstaged`, `git_diff_staged`, `git_diff`
- `git_commit`, `git_add`, `git_reset`
- `git_log`, `git_branch`, `git_create_branch`, `git_checkout`
- `git_show`

**Veredicto:** ✅ Seguro — Ya documentado previamente. Esencial para operaciones Git.

---

### 23. modelcontextprotocol/server-filesystem — Filesystem Operations (Referencia)

| Campo | Detalle |
|-------|---------|
| **URL** | https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem |
| **Lenguaje** | TypeScript |
| **Instalación** | `npx -y @modelcontextprotocol/server-filesystem /ruta/permitida` |
| **Tools** | 15 |

**Veredicto:** ⚠️ Precaución — Ya documentado previamente. Útil pero redundante con herramientas nativas del agente.

---

### 24. modelcontextprotocol/server-memory — Knowledge Graph Memory

| Campo | Detalle |
|-------|---------|
| **URL** | https://github.com/modelcontextprotocol/servers/tree/main/src/memory |
| **Lenguaje** | TypeScript |
| **Instalación** | `npx -y @modelcontextprotocol/server-memory` |
| **Uso** | Base de conocimiento persistente basada en grafo |

**Descripción:**
Sistema de memoria persistente basado en grafo de conocimiento. Útil para mantener contexto entre sesiones del agente.

**Veredicto:** ✅ Seguro — Almacena datos localmente en archivo JSON. Útil para persistencia de conocimiento.

---

### 25. ia-qa.com/mcp — LLM y RAG Testing

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/ia-qa/api |
| **Tipo** | MCP Server remoto |
| **Usos** | 4.18k |

**Descripción:**
Tools de testing para LLM agents, RAG, AI tools. Testing de sistemas de IA directamente desde Cursor, Claude Desktop, Windsurf. Incluye herramientas de testing clásicas también. No requiere API key, no requiere signup, gratuito.

**Instalación:**
```bash
npx smithery mcp add ia-qa/api
```

**Veredicto:** ⚠️ Precaución — Interesante para testing de LLMs pero remoto.

---

### 26. Postman MCP (detalle adicional)

Ya documentado arriba. Testing de APIs REST con 100+ tools.

---

### 27. SeedBase — Test Data Generation

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/marcelgl/seedbase |
| **Tipo** | MCP Server remoto |

**Descripción:**
Genera datos de prueba realistas con foreign-key consistency para bases de datos. Lista proyectos, obtiene schema DDL, genera datasets SQL. Free tier sin credit card.

**Veredicto:** ⚠️ Precaución — Útil para generar datos de prueba pero remoto.

---

### 28. Webhook Tester — HTTP Endpoint Testing

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/axel-belfort/webhook-tester |
| **Tipo** | MCP Server remoto (x402) |
| **Costo** | $0.002/call |

**Descripción:**
Testing de webhooks para AI agents. Send POST/PUT/PATCH/DELETE a cualquier endpoint con custom headers y JSON payloads. Mide latency, TLS handshake time y status codes.

**Tool:** `apitestwebhook`

**Veredicto:** ⚠️ Precaución — Útil para testing de APIs pero con micropagos crypto.

---

### 29. Regex Tester — Pattern Testing

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/axel-belfort/regex-generator |
| **Tipo** | MCP Server remoto (x402) |
| **Costo** | $0.001/call |

**Descripción:** Testing de regex patterns. Match, capture groups y explicaciones en lenguaje natural.

**Veredicto:** ⚠️ Precaución — Útil pero mejor usar herramientas locales.

---

### 30. Code Sandbox — Safe Code Execution

| Campo | Detalle |
|-------|---------|
| **URL** | https://smithery.ai/server/axel-belfort/code-sandbox |
| **Tipo** | MCP Server remoto (x402) |
| **Costo** | $0.01/call |

**Descripción:** Ejecución sandboxeada de Python, JavaScript y SQL en entorno aislado. Timeout de 10s.

**Tool:** `codeexecutesandbox`

**Veredicto:** ⚠️ Precaución — Útil para probar snippets pero remoto y de pago.

---

### 31. Go Process Inspector — gospy (Detalle Adicional)

Ya documentado previamente. Herramienta específica para Go que permite inspeccionar procesos Go en ejecución (gorutinas, memoria, runtime).

---

### Tabla Comparativa — Hallazgos Adicionales

| # | Herramienta | Tipo | Testing | Code Review | SAST | Coverage | Benchmark | Supply Chain | Instalación | Seguridad | Valoración |
|---|-------------|------|---------|-------------|------|----------|-----------|--------------|-------------|-----------|------------|
| 1 | **bugAgent** | MCP remoto | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ | Smithery | ⚠️ Remoto | ⭐ |
| 2 | **Code Sentinel** | MCP remoto | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | Smithery | ⚠️ Remoto | ⭐ |
| 3 | **Semgrep MCP** | MCP local | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | `uvx` | ✅ Local | ⭐ INDISPENSABLE |
| 4 | **Trust** | MCP remoto | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | Smithery | ⚠️ Remoto | 👍 |
| 5 | **Security Auditor** | MCP remoto | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | Smithery | ⚠️ Remoto | 👍 |
| 6 | **safedep/vet** | CLI + MCP | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | `brew`/`go install` | ✅ Local | ⭐ INDISPENSABLE |
| 7 | **OSV MCP** | MCP local | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | `go build` | ✅ Local | ⭐ |
| 8 | **Agent Bom** | MCP remoto | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | Smithery | ⚠️ Remoto | 👍 |
| 9 | **agent-lsp** | MCP local | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | `curl \| sh` | ✅ Local | ⭐ REVOLUCIÓN |
| 10 | **SpecLock** | MCP remoto | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | Smithery | ⚠️ Remoto | 👍 |
| 11 | **Postman MCP** | MCP remoto | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | Smithery | ⚠️ Remoto | ⭐ |
| 12 | **Buildkite** | MCP remoto | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | Smithery | ⚠️ Remoto | 🤷 |
| 13 | **Test My Vibes** | MCP remoto | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | Smithery | ⚠️ Remoto | 👍 |
| 14 | **Compuute Scanner** | MCP remoto | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | Smithery | ⚠️ Remoto | 👍 |
| 15 | **SkillAudit** | MCP remoto | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | `npx` | ⚠️ Remoto | 👍 |
| 16 | **Shrike Security** | MCP remoto | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | Smithery | ⚠️ Remoto | 👍 |
| 17 | **GitHub MCP (code_sec)** | MCP remoto | ❌ | ✅ | ✅ | ❌ | ❌ | ✅ | Docker | 🔴 PAT token | ⭐ |
| 18 | **ia-qa** | MCP remoto | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | Smithery | ⚠️ Remoto | 👍 |
| 19 | **SeedBase** | MCP remoto | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | Smithery | ⚠️ Remoto | 🤷 |
| 20 | **Mimiq** | MCP remoto | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | Smithery | ⚠️ Remoto | 👍 |

---

### Brechas Persistentes

1. **❌ Coverage Reports:** No existe ningún MCP server específico para code coverage. `go test -cover` sigue siendo la única opción via bash.
2. **❌ Benchmarking:** No hay MCP para `go test -bench`. Hay que usar bash directamente.
3. **❌ Go test runner:** No hay MCP que ejecute tests y devuelva resultados estructurados. `go test -json` via bash es la solución.
4. **❌ Go code generation:** No hay MCP para `stringer`, `mockgen`, `counterfeiter`, etc.

**Solución propuesta:** Para estas 4 brechas, la opción más pragmática es crear un skill/wrapper bash que:
- Ejecute `go test -cover -json ./...` y estructure el output
- Ejecute `go test -bench=. -benchmem ./...`
- Ejecute `go tool cover -html` para reports visuales
- Integre `go generate ./...` para code generation

---

### Stack Recomendado Ampliado

Basado en los hallazgos adicionales, el stack óptimo para un agente Go ahora incluye:

```json
{
  "mcpServers": {
    "go-quality": {
      "command": "mcp-server-go-quality",
      "args": []
    },
    "semgrep": {
      "command": "uvx",
      "args": ["semgrep-mcp"]
    },
    "github": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN", "ghcr.io/github/github-mcp-server"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_PAT}"
      }
    },
    "lsp": {
      "command": "agent-lsp",
      "args": ["go:gopls"]
    },
    "git": {
      "command": "uvx",
      "args": ["mcp-server-git", "--repository", "/home/secko/Projects/ZyroAgentCLI"]
    },
    "osv": {
      "command": "/path/to/osv-mcp-server",
      "args": []
    }
  }
}
```

**Herramientas CLI complementarias:**
```bash
# Seguridad supply chain (Go native)
brew install safedep/tap/vet

# Code review + security (remoto, vía Smithery)
npx smithery mcp add salrad/code-sentinel

# QA platform (remoto, vía Smithery) — opcional si se necesita testing e2e
npx smithery mcp add bugagent/bugagent-mcp
```

**Skills de superpowers adicionales:**
```bash
npx skills add obra/superpowers/test-driven-development
npx skills add obra/superpowers/requesting-code-review
npx skills add obra/superpowers/verification-before-completion
```

---

### Resumen de Veredictos

| Veredicto | Herramientas |
|-----------|-------------|
| ✅ **Seguro (local)** | Semgrep MCP, safedep/vet, OSV MCP, agent-lsp, gospy, git-mcp |
| ⚠️ **Precaución (remoto)** | Code Sentinel, bugAgent, Trust, Security Auditor, Agent Bom, Postman, Buildkite, SpecLock, Quality Coach, Mimiq, Test My Vibes, SkillAudit, Shrike, ia-qa |
| ❌ **Riesgoso** | CodeVulnerability (pago+remoto), CleanCode AI (pago+remoto) |

---

### Referencias

- awesome-mcp-servers: https://github.com/punkpeye/awesome-mcp-servers
- Smithery.ai: https://smithery.ai
- Semgrep MCP: https://github.com/semgrep/mcp
- safedep/vet: https://github.com/safedep/vet
- OSV MCP: https://github.com/StacklokLabs/osv-mcp
- agent-lsp: https://github.com/blackwell-systems/agent-lsp
- bugAgent: https://bugagent.com
- Code Sentinel: https://www.npmjs.com/package/code-sentinel-mcp
- GitHub MCP Server: https://github.com/github/github-mcp-server
- modelcontextprotocol/servers: https://github.com/modelcontextprotocol/servers
- SpecLock: https://smithery.ai/server/sgroy10/speclock
- Trust Scanner: https://www.trust-scan.me
- Compuute Scanner: https://scan.compuute.se
- SkillAudit: https://skillaudit.vercel.app
- Shrike Security: https://smithery.ai/server/shrike-security/shrike-mcp
