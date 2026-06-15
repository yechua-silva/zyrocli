# Evaluación: HelixDB como EJE CENTRAL via MCP

**Fecha**: 2026-06-14
**Estado**: Research — NO hay código nuevo
**Contexto**: Evaluar si HelixDB MCP nativo puede reemplazar el middleware Go actual

---

## 1. Hallazgo Principal: NO Existe HelixDB MCP Server

### Evidencia

| Fuente | Resultado |
|--------|-----------|
| GitHub org `HelixDB/` | 14 repos — **ninguno es MCP server** |
| GitHub topic `helixdb-mcp` | **0 repos públicos** |
| Docs `docs.helix-db.com` | Sin mención de MCP en ninguna página |
| Roadmap | Feature requests: SSO, RBAC, PrivateLink — **sin MCP** |
| SDKs oficiales | Go, Rust, TypeScript, Python — **ninguno es MCP** |

### Lo que SÍ existe en HelixDB

| Componente | Tipo | Props |
|------------|------|-------|
| `helix-go` | Go SDK | HTTP REST, `client.Exec()`, dynamic queries |
| `helix-cli` | CLI | `helix start`, `helix query`, `helix chef` |
| `helix-db` | Core (Rust) | Puerto 6969, HTTP API, HelixQL compilado |
| `helix chef` | Bootstrapper | Instala docs MCP (Context7), scaffolds proyecto |
| HelixDB Cloud | Managed | Object storage, ACID, auto-scaling readers |

### Referencia Confusa: `helix chef`

El comando `helix chef` instala "query skills and docs MCP" — pero esto se refiere al **Context7 docs MCP** (el mismo que ya usamos en `internal/context/bridge.go`), NO a un MCP server de HelixDB para consulta de grafos.

```bash
# Lo que dice el README:
helix chef  # "installs the HelixDB query skills and docs MCP"

# Lo que realmente hace:
# 1. Instala Context7 MCP para docs de librerías
# 2. Scaffolds un proyecto HelixDB
# 3. Inicia instancia local
# 4. Crea HELIX_CHEF_PROMPT.md
```

---

## 2. Escenarios Arquitectónicos

### Escenario A — HelixDB MCP Server NATIVO (NO EXISTE)

```
OpenCode/Gentle AI
  │
  ├── MCP → HelixDB MCP Server (¿?¿?)
  │         ├── query graph
  │         ├── search vectors
  │         ├── read nodes
  │         └── traverse edges
  │
  └── CLI → ZyroCLI (scaffold, init, sync)
```

**Veredicto**: Imposible implementar. No hay server que conectar.

### Escenario B — HelixDB como backend con ZyroCLI como middleware (ACTUAL)

```
OpenCode/Gentle AI
  │
  ├── MCP → Context7 (docs de librerías)  ← bridge.go
  │
  └── CLI → ZyroCLI → HelixDB (vía Go SDK)
                      ├── context.go → taskcontext.GetTaskContext()
                      ├── db.go → schema init, status
                      ├── task.go → task create/link
                      └── absorb.go → data ingestion
```

**Estado actual**: Funcional. ZyroCLI es el middleware que traduce comandos CLI a queries HelixDB.

### Escenario C — MCP Server CUSTOM (FACTIBLE)

```
OpenCode/Gentle AI
  │
  ├── MCP → helix-mcp-server (Go, NUESTRO)
  │         ├── tools: query_task_context
  │         ├── tools: search_similar_code
  │         ├── tools: search_decisions
  │         ├── tools: get_project_context
  │         ├── resources: project://graph
  │         └── resources: task://context/{id}
  │
  ├── MCP → Context7 (docs de librerías)  ← bridge.go
  │
  └── CLI → ZyroCLI (scaffold, init, sync, db admin)
```

**Estado**: Requiere construir el MCP server. Go tiene excellent MCP SDK.

### Escenario D — HÍBRIDO (RECOMENDADO)

```
OpenCode/Gentle AI
  │
  ├── MCP → helix-mcp-server (lecturas, consultas en vivo)
  │         ├── tools: task_context, search_code, search_decisions
  │         └── resources: project_context
  │
  ├── MCP → Context7 (docs de librerías)
  │
  └── CLI → ZyroCLI (escrituras, admin, scaffold)
            ├── db init/status/reset (admin)
            ├── task create/link (escrituras controladas)
            ├── absorb (ingesta batch)
            └── project add/status (gestión)
```

---

## 3. Impacto en el Código Existente

### Qué SE MANTIENE (sin cambios)

| Archivo/Módulo | Razón |
|----------------|-------|
| `internal/db/helix/client.go` | Go SDK wrapper — lo usa TANTO el CLI como el MCP server futuro |
| `internal/db/helix/schema.go` | Schema init — operación administrativa, no MCP |
| `internal/db/helix/nodes.go` | CRUD de nodos — reutilizable por MCP server |
| `internal/db/helix/edges.go` | CRUD de edges — reutilizable por MCP server |
| `internal/db/helix/search.go` | Vector/Text search — reutilizable por MCP server |
| `cmd/zyrocli/db.go` | CLI admin commands — se mantienen |
| `cmd/zyrocli/task.go` | Task create/link — escrituras controladas |
| `cmd/zyrocli/absorb.go` | Data ingestion — batch operations |
| `internal/scheduler/` | State machine — orquestación, no storage |
| `internal/handoff/` | Handoff parsing — input processing |
| `internal/skilladvisor/` | Skill scoring — lógica de negocio |
| `internal/apply/runner.go` | Task runner — implementación |
| `internal/spec/` | C-I-O DSL — especificación |
| `internal/scaffold/` | Project scaffolding — generación |
| `internal/context/bridge.go` | Context7 MCP — documentación de libs |

### Qué CAMBIA (si adoptamos Escenario D)

| Archivo | Cambio | Esfuerzo |
|---------|--------|----------|
| `cmd/zyrocli/context.go` | **Se depreca** — el agente consulta via MCP, no via CLI | Bajo |
| `internal/taskcontext/queries.go` | **Se depreca como CLI endpoint** — la lógica se mueve al MCP server | Bajo |
| `internal/taskcontext/formatter.go` | **Se depreca** — el MCP server formatea su propia respuesta | Bajo |
| `internal/taskcontext/types.go` | **Se reutiliza** — tipos compartidos entre CLI y MCP | Cero |
| `AGENT.md` línea 13 | Stack: agregar "helix-mcp-server" | Trivial |
| `AGENT.md` línea 120 | Comando context: MCP en vez de CLI | Trivial |

### Qué SE AGREGA

| Archivo/Módulo | Propósito | Esfuerzo |
|----------------|-----------|----------|
| `cmd/helix-mcp/main.go` | Entry point del MCP server | Medio |
| `internal/mcp/tools.go` | Tool definitions para MCP | Medio |
| `internal/mcp/resources.go` | Resource definitions para MCP | Bajo |
| `internal/mcp/handler.go` | Request handler — delega a `internal/db/helix/` | Medio |
| `opencode.json` (proyecto) | Config MCP server | Bajo |

### MCP Server Go — Tools Propuestos

```go
// Tools (lecturas — el agente puede llamar)
tools := []mcp.Tool{
    {
        Name:        "task_context",
        Description: "Get full context for a task: skills, code nodes, docs, patterns",
        InputSchema: map[string]interface{}{
            "task_id": "number (required)",
            "format":  "string (text|json|prompt, default: prompt)",
        },
    },
    {
        Name:        "search_code",
        Description: "Search code entities by vector similarity or text",
        InputSchema: map[string]interface{}{
            "query": "string (required) — natural language or code snippet",
            "k":     "number (default: 10)",
        },
    },
    {
        Name:        "search_decisions",
        Description: "Find technical decisions by semantic similarity",
        InputSchema: map[string]interface{}{
            "query": "string (required)",
            "k":     "number (default: 5)",
        },
    },
    {
        Name:        "project_context",
        Description: "Get project overview: technologies, skills, recent decisions",
        InputSchema: map[string]interface{}{
            "project_name": "string (optional, defaults to current)",
        },
    },
}

// Resources (estado del grafo)
resources := []mcp.Resource{
    {
        URI:  "helix://project/{name}/graph",
        Name: "Project knowledge graph",
    },
    {
        URI:  "helix://task/{id}/context",
        Name: "Task context bundle",
    },
}
```

### Configuración en opencode.json

```json
{
  "mcp": {
    "helix": {
      "type": "local",
      "command": ["zyrocli", "mcp", "serve"],
      "enabled": true,
      "env": {
        "HELIX_URL": "http://localhost:6969",
        "HELIX_PROJECT": "{env:ZYRO_PROJECT_ID}"
      }
    }
  }
}
```

---

## 4. Análisis Comparativo

### Opción 1: Status Quo (Escenario B)

| Aspecto | Valoración |
|---------|-----------|
| Complejidad | Baja — ya funciona |
| Latencia queries | Media — CLI invocation overhead (~100ms+) |
| Flexibilidad agente | Baja — agente depende de CLI para cada query |
| Mantenimiento | Bajo — código existente |
| Capacidad de discovery | Baja — agente no puede navegar el grafo autónomamente |

### Opción 2: MCP Server Custom (Escenario D)

| Aspecto | Valoración |
|---------|-----------|
| Complejidad | Media — ~300-500 líneas Go nuevas |
| Latencia queries | Baja — MCP stdio directo, sin process spawn |
| Flexibilidad agente | Alta — agente navega grafo, busca semánticamente, descubre contexto |
| Mantenimiento | Medio — MCP server es código nuevo pero reutiliza `internal/db/helix/` |
| Capacidad de discovery | Alta — agente puede autónomamente: "¿qué decisiones tomamos sobre auth?" |

### Opción 3: HelixDB MCP Oficial (Escenario A)

| Aspecto | Valoración |
|---------|-----------|
| Disponibilidad | **NO EXISTE** — ni hoy ni en roadmap cercano |
| Riesgo | Depender de un feature que no está en el roadmap de HelixDB |
| Timeline | Indefinido — meses si algún día sale |

---

## 5. Preguntas Abiertas — Respuestas

### ¿HelixDB MCP server se autentica?
**N/A** — No existe. Pero si construimos uno, la autenticación sería Hereda del Go SDK: `WithAPIKey()` para HelixDB Cloud, o sin auth para local dev.

### ¿Expone operaciones de escritura o solo lectura?
**N/A** — Si construimos uno, la recomendación es: **SOLO LECTURA via MCP**. Las escrituras se mantienen via CLI (controladas, auditadas, con validación humana).

### ¿Conviene tener AMBOS: MCP para consultas + Go SDK para writes?
**SÍ** — Es el Escenario D (híbrido). MCP para lecturas en vivo del agente, Go SDK para escrituras batch/administrativas.

### ¿Funciona con HelixDB Cloud?
**SÍ** — El Go SDK soporta Cloud vía `WithAPIKey()`. Un MCP server construido sobre el SDK hereda esa capacidad automáticamente.

---

## 6. Recomendación

### Veredicto: **Construir MCP Server Custom (Escenario D)**

**NO migrar a un MCP server que no existe. SÍ construir el nuestro.**

#### Por qué Escenario D y no B:

1. **El agente (Gentle AI) se vuelve autónomo** — puede navegar el grafo sin pasar por `zyrocli context` para cada query
2. **Latencia reducida** — MCP stdio es más rápido que spawn de proceso CLI
3. **Discovery semántico** — el agente puede hacer preguntas como "¿qué patrones usamos en proyectos similares?" directamente
4. **El código existente se reutiliza** — `internal/db/helix/` es la capa de acceso, el MCP server es solo una nueva Interface sobre ella
5. **`zyrocli context` se depreca gradualmente** — no es un breaking change

#### Por qué NO Escenario A:

1. No existe HelixDB MCP server oficial
2. No está en el roadmap de HelixDB
3. Depender de un feature no garantizado es arriesgado
4. Construir el nuestro es factible en ~2-3 días

---

## 7. Próximos Pasos (si se aprueba)

### Fase 1: MCP Server Foundation (1-2 días)
1. Crear `cmd/helix-mcp/main.go` — entry point MCP server Go
2. Usar `github.com/mark3labs/mcp-go` o `github.com/modelcontextprotocol/go-sdk` como MCP framework
3. Implementar 2 tools iniciales: `task_context`, `search_code`
4. Registrar en `opencode.json` del proyecto ZyroAgentCLI

### Fase 2: Tools Completos (1 día)
5. Agregar `search_decisions`, `project_context`
6. Agregar resources: `helix://project/{name}/graph`
7. Tests de integración MCP

### Fase 3: Migración (0.5 días)
8. Actualizar `AGENT.md` con la nueva arquitectura
9. Deprecar `zyrocli context` (mantener por backward compat)
10. Documentar en `docs/helixdb-mcp-evaluation.md` (este archivo)

### Dependencias
- `github.com/mark3labs/mcp-go` — MCP server framework para Go (o similar)
- `github.com/helixdb/helix-db/sdks/go v0.1.1` — ya está en go.mod

---

## 8. Riesgos

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|-------------|---------|------------|
| MCP framework Go inmaduro | Media | Bajo | Evaluar `mcp-go` antes de comprometer |
| HelixDB lanza MCP oficial después | Baja | Medio | Nuestro MCP server usa el Go SDK, fácil de reemplazar |
| Over-engineering para 2-3 proyectos | Media | Bajo | Empezar con 2 tools, expandir solo si se necesita |
| Agente abusa de queries MCP (latencia) | Baja | Medio | Rate limiting en el MCP server |

---

## Referencias

- HelixDB Go SDK: `github.com/helixdb/helix-db/sdks/go`
- HelixDB Docs: https://docs.helix-db.com
- OpenCode MCP Config: ver skill `customize-opencode` → sección "MCP servers"
- MCP Protocol: https://modelcontextprotocol.io
- mcp-go (Go SDK): https://github.com/mark3labs/mcp-go
