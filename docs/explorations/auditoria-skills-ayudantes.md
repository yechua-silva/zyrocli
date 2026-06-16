# Auditoría de Seguridad — Skills Ayudantes Externos

> **Fecha:** 2026-06-15  
> **Propósito:** Validar que las skills/MCP servers que usará el agente asistente son seguras antes de instalarlas  
> **Auditor realizado por:** zyro-skills-audit  

---

## Resumen

| # | Skill/MCP | Tipo | Veredicto | Riesgo Principal | Recomendación |
|---|-----------|------|-----------|------------------|---------------|
| 1 | `afshinator/mcp-server-go-quality` | MCP Go | ⚠️ Precaución | Ejecuta 3 subprocesos (golangci-lint, govulncheck, nilaway); govulncheck hace llamada de red única para DB de CVEs | Usar con límite de timeout |
| 2 | `wavilen/golangci-lint-mcp` | MCP Go + npm | ⚠️ Precaución | El npm install script escribe en ~/.agents, ~/.config/opencode, .mcp.json; ejecuta golangci-lint como subproceso | Preferir solo el binario Go (sin npm install) |
| 3 | `github/github-mcp-server` | MCP Go | ⚠️ Precaución | Requiere GitHub PAT con scopes repo; acceso total a repos según token | Usar `--read-only` y toolsets mínimos |
| 4 | `modelcontextprotocol/server-git` | MCP Python | ✅ Segura | Ejecuta git commands como subprocesos; puede modificar el repo | Limitar con `--repository` |
| 5 | `modelcontextprotocol/server-filesystem` | MCP TypeScript | ⚠️ Precaución | Puede leer/escribir archivos en directorios permitidos; write_file sobreescribe | Limitar directorios estrictamente |
| 6 | `obra/superpowers` (skills) | Skills instructivas | ✅ Seguras | Solo instrucciones markdown, no ejecutan código | Sin restricciones |
| 7 | `monsterxx03/gospy` | MCP Go | ❌ No usar | Requiere root/sudo para leer memoria de procesos | Usar pprof/dlv en su lugar |
| 8 | `@wavilen/golangci-lint-guide` (npm) | npm install | ⚠️ Precaución | Escribe en múltiples directorios del sistema y modifica configs | Instalar solo el binario Go, no el npm |

---

## Auditorías Detalladas

---

### 1. mcp-server-go-quality

- **URL:** https://github.com/afshinator/mcp-server-go-quality
- **Versión auditada:** main (51 commits, 0 stars)
- **Código revisado:** Sí — repositorio público, código Go legible, sin ofuscación
- **Veredicto:** ⚠️ Precaución
- **Análisis:**

  **Permisos:**
  - Ejecuta 3 subprocesos como comandos de shell: `golangci-lint`, `govulncheck`, `nilaway`
  - Lee el filesystem del proyecto para analizar archivos Go
  - **No escribe archivos** (solo lectura de código fuente)
  - `govulncheck` descarga DB de vulnerabilidades (llamada de red única al iniciar)
  - Usa `exec.CommandContext` con validación de rutas relativas (no permite `../` ni rutas absolutas)

  **Riesgos:**
  - Ejecución de código: MEDIO — los 3 binarios son herramientas legítimas y conocidas
  - Fuga de datos: BAJO — no envía datos a red (excepto govulncheck que descarga DB)
  - Modificación de archivos: BAJO — no tiene herramientas de escritura
  - Path traversal: BAJO — valida que el path no sea absoluto ni suba de directorio

  **Dependencias:**
  ```
  github.com/mark3labs/mcp-go v0.54.1  (framework MCP estándar)
  gopkg.in/yaml.v3 v3.0.1
  github.com/google/uuid v1.6.0
  ```
  Dependencias limpias, sin paquetes sospechosos.

- **Recomendación de uso:** ✅ Aprobado con precaución. Usar solo en proyectos de desarrollo. Configurar timeout razonable. La primera ejecución de govulncheck descargará la DB de vulnerabilidades (requiere internet).

---

### 2. golangci-lint-mcp

- **URL:** https://github.com/wavilen/golangci-lint-mcp
- **Versión auditada:** main (9 commits, 1 star)
- **Código revisado:** Sí — repositorio público, Go + JavaScript, sin ofuscación
- **Veredicto:** ⚠️ Precaución

- **Análisis:**

  **Componentes:**
  - **Binario Go** (`go install github.com/wavilen/golangci-lint-mcp@latest`): MCP server que ejecuta golangci-lint
  - **npm package** (`@wavilen/golangci-lint-guide`): Script de instalación que configura skills, hooks, plugins y MCP

  **Permisos del binario Go:**
  - Ejecuta `golangci-lint run --fix --output.json.path stdout` como subproceso
  - Las guías de fix (629) van embebidas en el binario — **sin llamadas de red**
  - Valida rutas (no permite rutas absolutas ni `../`)
  - `--gosec-ai` opcional: envía código a API externa (Gemini) si se configura

  **Permisos del npm package (`bin/install.js`):**
  - **ALTO** — escribe en:
    - `~/.agents/skills/golangci-lint-guide/`
    - `~/.config/opencode/plugins/` y `~/.config/opencode/shared/`
    - `.claude/hooks/`, `.cursor/hooks/`
    - Modifica `.claude/settings.json`, `.cursor/hooks.json`
    - Modifica `opencode.json` o `~/.config/opencode/opencode.json`
    - Modifica `.mcp.json`
  - Ejecuta `which golangci-lint-mcp` y `which golangci-lint` para verificar binarios
  - Crea backups de config antes de modificarlas

  **Riesgos:**
  - Ejecución de código: MEDIO (golangci-lint como subproceso)
  - Modificación de configs: ALTO (el install.js modifica múltiples archivos de configuración)
  - Fuga de API keys: MEDIO (gosec AI envía API key a provider externo)
  - Dependencias npm: solo devDependencies (eslint, desloppify)

  **Dependencias Go:**
  ```
  github.com/mark3labs/mcp-go v0.48.0
  github.com/moby/moby/api v1.54.1
  github.com/testcontainers/testcontainers-go v0.42.0
  ```
  Dependencias de test (testcontainers, ginkgo) elevadas pero solo en tests.

- **Recomendación de uso:** ⚠️ Usar con precaución. **No ejecutar el npm package** (`npx @wavilen/golangci-lint-guide`) ya que modifica configuraciones del sistema sin preguntar. Instalar solo el binario Go. No configurar `--gosec-ai` para evitar enviar código a APIs externas.

---

### 3. GitHub MCP Server

- **URL:** https://github.com/github/github-mcp-server
- **Versión auditada:** main (942 commits, 30.7k stars, 4.4k forks)
- **Código revisado:** Sí — repositorio oficial de GitHub, código Go
- **Veredicto:** ⚠️ Precaución

- **Análisis:**

  **Permisos:**
  - **Requiere GitHub Personal Access Token** (scopes: repo, read:org, security_events)
  - Hace llamadas HTTP a la API de GitHub (api.github.com)
  - **No ejecuta código local** — solo llamadas REST a GitHub
  - Soporta Docker como método de instalación (aislamiento adicional)
  - Acceso a repositorios según permisos del token

  **Modos de seguridad disponibles:**
  - `--read-only`: solo herramientas de lectura (ideal para CI/review)
  - `--toolsets`: habilita solo grupos específicos de herramientas (mínimo privilegio)
  - `--tools`: habilita herramientas individuales
  - Ejemplo seguro: `--toolsets repos,issues --read-only`
  - OAuth disponible en modo remoto (sin token hardcodeado)

  **Riesgos:**
  - Fuga de token: ALTO si el token está hardcodeado en config
  - Acceso a datos: MEDIO-ALTO (depende de scopes del token)
  - Data exfiltration: BAJO (solo llamadas a GitHub API)
  - Ejecución remota: BAJO (GitHub Actions trigger podría ejecutar workflows)

  **Dependencias:**
  Go estándar + SDK de GitHub. Sin dependencias sospechosas.

- **Recomendación de uso:** ⚠️ Usar con precaución. **Usar siempre con `--read-only`** a menos que se necesite escritura. Configurar scopes mínimos del PAT. Usar variable de entorno `${GITHUB_PAT}` en lugar de hardcodear. Considerar usar la versión remota con OAuth.

---

### 4. Git MCP Server

- **URL:** https://github.com/modelcontextprotocol/servers/tree/main/src/git
- **Versión auditada:** main (repo servers 87.3k stars — server oficial de MCP)
- **Código revisado:** Sí — Python, parte del repo oficial de MCP
- **Veredicto:** ✅ Segura

- **Análisis:**

  **Permisos:**
  - Ejecuta comandos git como subprocesos (`git status`, `git diff`, `git commit`, etc.)
  - Lee/escribe en el repositorio especificado
  - `git_commit`, `git_add`, `git_reset`, `git_checkout`, `git_create_branch` — modifican el repo
  - `git_status`, `git_diff_*`, `git_log`, `git_show`, `git_branch` — solo lectura

  **Riesgos:**
  - Modificación accidental: MEDIO (puede hacer commits no deseados)
  - Ejecución de código: BAJO (solo ejecuta git)
  - Fuga de datos: BAJO (los repositorios git son locales)
  - Path traversal: BAJO (usa `--repository` para limitar alcance)

  **Instalación segura:**
  - `uvx mcp-server-git --repository /ruta/especifica/al/repo`
  - Docker con montaje bind del repo específico

- **Recomendación de uso:** ✅ Aprobado. Limitar siempre con `--repository` al proyecto específico. Útil para diff analysis y commits automatizados bajo supervisión humana.

---

### 5. Filesystem MCP Server

- **URL:** https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem
- **Versión auditada:** main (repo servers 87.3k stars — server oficial de MCP)
- **Código revisado:** Sí — TypeScript, parte del repo oficial de MCP
- **Veredicto:** ⚠️ Precaución

- **Análisis:**

  **Permisos:**
  - 15 tools MCP para operaciones de archivos
  - **Escritura:** `write_file`, `edit_file`, `create_directory`, `move_file`
  - **Lectura:** `read_text_file`, `read_media_file`, `list_directory`, `search_files`, etc.
  - Control de acceso por directorios permitidos (vía CLI args o MCP Roots)
  - Tool annotations que marcan herramientas como destructivas

  **Riesgos:**
  - Destrucción de datos: ALTO (`write_file` sobreescribe sin confirmación)
  - Lectura no autorizada: MEDIO (puede leer archivos sensibles en directorios permitidos)
  - Ejecución de código: BAJO (no ejecuta comandos, solo operaciones de archivo)

  **Controles de seguridad:**
  - Directorios permitidos: se pasan como argumentos al inicio
  - Roots: permite al cliente actualizar directorios dinámicamente
  - Validación de path: no permite salir de directorios permitidos
  - Anotaciones: `readOnlyHint`, `destructiveHint`, `idempotentHint`

- **Recomendación de uso:** ⚠️ Precaución. Si se instala, limitar a un solo directorio de trabajo. Considerar que el agente ya tiene herramientas de archivo nativas. Docker con montaje `ro` si solo lectura. **No exponer a directorios del sistema** (`/etc`, `~/.ssh`, etc.).

---

### 6. Obra/Superpowers Skills

- **URL:** https://github.com/obra/superpowers
- **Versión auditada:** v5.1.0 (229k stars, 20.3k forks)
- **Código revisado:** Sí — repo extremadamente popular
- **Veredicto:** ✅ Seguras

- **Análisis:**

  **Skills incluidas relevantes:**
  - `test-driven-development` — loop RED-GREEN-REFACTOR
  - `requesting-code-review` — pre-review checklist
  - `verification-before-completion` — verificación antes de marcar tarea completa
  - `brainstorming` — refinamiento de ideas
  - `writing-plans` — planes de implementación
  - `executing-plans` — ejecución por lotes
  - `subagent-driven-development` — subagentes con revisión
  - `systematic-debugging` — debug sistemático
  - `using-git-worktrees` — ramas paralelas
  - `finishing-a-development-branch` — merge/PR workflow

  **Permisos:**
  - **Solo instrucciones markdown** — no ejecutan código
  - Son "prompts" que le dicen al agente cómo comportarse
  - El comando `npx skills add` solo descarga archivos markdown
  - No tienen acceso a red, filesystem, ni ejecución de comandos

  **Riesgos:**
  - Mínimos. El único riesgo es que el agente siga instrucciones incorrectas
  - No hay ejecución de código, fuga de datos, ni modificación de archivos

- **Recomendación de uso:** ✅ Aprobadas sin restricciones. Instalar todas las que sean útiles. No representan riesgo de seguridad.

---

### 7. gospy — Go Process Inspector

- **URL:** https://github.com/monsterxx03/gospy
- **Versión auditada:** v0.8.1 (97 stars, 5 forks)
- **Código revisado:** Sí — Go, código abierto
- **Veredicto:** ❌ No usar

- **Análisis:**

  **Permisos:**
  - **Requiere root/sudo** para funcionar
  - Lee `/proc/<pid>/mem` en Linux (memoria de procesos externos)
  - Usa APIs Mach en macOS para inspección de procesos
  - Expone API HTTP en puerto 8974 (potencialmente accesible desde red)
  - MCP endpoint en HTTP (streamableHTTP)

  **Riesgos:**
  - **ALTO** — ejecución como root
  - Acceso a memoria de cualquier proceso en el sistema
  - Potencial lectura de secretos en memoria de otros procesos
  - Sin autenticación en API HTTP
  - El proyecto fue escrito >90% por AI (aider), lo que reduce confiabilidad
  - Solo ~$2 USD de costo de AI para generarlo

- **Recomendación de uso:** ❌ No usar como MCP server del agente. Requerir root para una herramienta de debugging es un riesgo innecesario. Alternativas: `pprof`, `dlv` (Delve debugger), `go tool pprof`, `strace`.

---

### 8. Buildkite MCP

- **URL:** https://smithery.ai/servers/buildkite
- **Versión auditada:** No disponible (servicio Smithery)
- **Código revisado:** No (solo disponible via Smithery)
- **Veredicto:** ❌ No usar (innecesario)

- **Análisis:**
  - MCP remoto alojado en Smithery
  - Requiere API key de Buildkite
  - Llamadas de red a API de Buildkite
  - **No relevante** para el proyecto ZyroAgentCLI (no usa Buildkite para CI)

- **Recomendación de uso:** ❌ No usar. Solo relevante si el proyecto migra a Buildkite.

---

## Skills NO Recomendadas

| Skill | Motivo |
|-------|--------|
| `monsterxx03/gospy` | ❌ Requiere root/sudo para acceder a memoria de procesos |
| `buildkite` (Smithery) | ❌ No relevante (no usamos Buildkite); código no revisable |
| `@wavilen/golangci-lint-guide` (npm install) | ⚠️ El script de instalación modifica configs sin preguntar. Usar solo el binario Go |

---

## Tabla de Riesgos Comparativa

| Skill | ¿Ejecuta código? | ¿Acceso red? | ¿Filesystem? | ¿API keys? | Riesgo acumulado |
|-------|------------------|--------------|--------------|------------|------------------|
| mcp-server-go-quality | Sí (golangci-lint, govulncheck, nilaway) | Solo govulncheck (DB CVEs) | Lectura | No | ⚠️ Medio |
| golangci-lint-mcp | Sí (golangci-lint) + install.js | Opcional (gosec AI) | Lectura/Escritura (install.js) | Gosec AI opcional | ⚠️ Medio-Alto |
| GitHub MCP Server | No | Sí (GitHub API) | No | GitHub PAT | ⚠️ Medio (token) |
| Git MCP Server | Sí (git commands) | No | Sí (repo) | No | ✅ Bajo |
| Filesystem MCP | No | No | Sí (lectura/escritura) | No | ⚠️ Medio |
| Superpowers skills | No | No | No | No | ✅ Mínimo |
| gospy | No (solo lectura) | Sí (API HTTP) | Sí (/proc/PID/mem) | No | ❌ Alto (root) |

---

## Configuración Segura Recomendada

### Stack esencial mínimo

```json
{
  "mcpServers": {
    "go-quality": {
      "command": "mcp-server-go-quality",
      "args": []
    },
    "github": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-e", "GITHUB_PERSONAL_ACCESS_TOKEN",
        "ghcr.io/github/github-mcp-server",
        "--read-only",
        "--toolsets", "repos,issues,pull_requests"
      ],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_PAT}"
      }
    },
    "git": {
      "command": "uvx",
      "args": [
        "mcp-server-git",
        "--repository", "/home/secko/Projects/ZyroAgentCLI"
      ]
    }
  }
}
```

### Reglas de seguridad

1. **Nunca hardcodear tokens** — usar variables de entorno o input prompts
2. **Filesystem MCP** — NO instalar. El agente ya tiene herramientas nativas de archivo
3. **`golangci-lint-mcp`** — instalar solo el binario Go, NO ejecutar `npx @wavilen/golangci-lint-guide`
4. **GitHub MCP** — usar `--read-only` por defecto; habilitar escritura solo cuando se necesite explícitamente
5. **gospy** — NO instalar, usar `pprof`/`dlv` en su lugar
6. **Skills de superpowers** — instalar sin restricciones vía `npx skills add`

### Skills a instalar (seguras)

```bash
npx skills add obra/superpowers/test-driven-development
npx skills add obra/superpowers/requesting-code-review
npx skills add obra/superpowers/verification-before-completion
npx skills add obra/superpowers/writing-plans
npx skills add obra/superpowers/executing-plans
npx skills add obra/superpowers/systematic-debugging
```

---

## Conclusión

De las 8 herramientas auditadas:

- **2 son completamente seguras** ✅ (Git MCP Server, Superpowers skills)
- **4 requieren precaución** ⚠️ (mcp-server-go-quality, golangci-lint-mcp, GitHub MCP, Filesystem MCP)
- **2 no se recomiendan** ❌ (gospy, Buildkite MCP)

El stack recomendado es: **mcp-server-go-quality** + **GitHub MCP Server** (read-only) + **Git MCP Server** + **Superpowers skills** (TDD, code review).
