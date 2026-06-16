# Proposal: Fase 4 — HelixDB Central Axis

## Intent

ZyroCLI tiene HelixDB como almacén de conocimiento (Fase 2-3), pero el agente solo accede via CLI o código Go. Fase 4 expone HelixDB como MCP server propio para que OpenCode consuma contexto de forma nativa: consultar tasks, buscar código, encontrar skills.

**Nota sobre Context7**: Context7 fue el plan arquitectónico original desde el diseño inicial, pero se reemplazó por Context + GitMCP (binario local `context`) antes de escribir código. El bridge `internal/context/bridge.go` siempre implementó el binario `context` — los comentarios decían "Context7" por la referencia arquitectónica original. No hay nada que deprecar, el bridge se mantiene como está.

## Scope

### In Scope
- **MCP server** con tools: `task_context`, `search_code`, `search_skills` (JSON-RPC 2.0 sobre stdio, mismo patrón que bridge.go)
- **Helix Skills** instalación automatizada (`helix-query-*`)
- **Deprecar `zyrocli context` CLI** — lógica migra a MCP tools, comando queda como alias con warning

### Out of Scope
- Reemplazar `openspec/` — HelixDB lo complementa, no lo sustituye
- Community detection en HelixDB (no existe como feature)
- TUI/web para contexto
- Parseo de lenguajes no-Go (TypeScript, Python)
- Write tools en MCP (solo reads por ahora — writes via `zyrocli` CLI)

## Capabilities

### New Capabilities
- `mcp-server`: Server MCP con stdio transport, tools `task_context`, `search_code`, `search_skills`; autenticación delegada a HelixDB
- `helix-query-skills`: Instalación y configuración de skills `helix-query-*` para consultas ad-hoc

### Modified Capabilities
- `zyrocli-context`: Marcar como deprecado, migrar lógica de queries a MCP tools, mantener compatibilidad temporal con warning
- `context-mcp-bridge`: El bridge existente (`internal/context/bridge.go`) NO se reemplaza — se mantiene como conector a `context` binary para documentation queries. El nuevo MCP server es complementario, no sustituto.

## Approach

1. **MCP server** (`internal/mcp/`): Nuevo paquete que implementa un server JSON-RPC 2.0 sobre stdin/stdout (mismo patrón que `internal/context/bridge.go` pero en dirección inversa — este es el server, no el cliente). Expone tres tools:
   - `task_context(task_id)` → llama a `internal/taskcontext.GetTaskContext()` y retorna contexto formateado
   - `search_code(query, project_id)` → llama a `internal/db/helix.Search()` sobre CodeNodes
   - `search_skills(query)` → llama a `internal/db/helix.FindSharedSkills()` + `VectorSearchGlobal()`

2. **Registro en OpenCode**: El MCP server se registra en `~/.config/opencode/opencode.json` como herramienta MCP via `command` mode (stdin/stdout).

3. **Helix Skills**: Script/instrucciones para instalar `helix-query-*` desde el registry de Helix.

4. **Deprecación**: `cmd/zyrocli/context.go` emite `stderr` warning "DEPRECATED: use MCP tool task_context" y delega al nuevo server via exec.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/mcp/` | New | MCP server package (handler, tools, transport) |
| `internal/mcp/server.go` | New | JSON-RPC server sobre stdio |
| `internal/mcp/handlers.go` | New | Tool handlers: task_context, search_code, search_skills |
| `cmd/zyrocli/context.go` | Modified | Añadir warning deprecation + delegar a MCP |
| `go.mod` / `go.sum` | Modified | Posible nueva dependencia si se requiere |
| `~/.config/opencode/opencode.json` | Modified | Registro del MCP server |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| MCP server compite con bridge `context` por stdio | Low | Puertos/transport distintos — bridge habla con binary externo, MCP server es el binary mismo |
| HelixDB no responde y MCP tools fallan | Medium | Error messaging claro en cada tool, no crash |
| OpenCode no soporta MCP tools via command mode | Low | Verificar antes de implementar — fallback a HTTP proxy |
| Deprecación abrupta rompe scripts existentes | Low | Warning + compatibilidad 1 mes |

## Rollback Plan

1. `git checkout HEAD -- internal/mcp/ cmd/zyrocli/context.go`
2. Revertir `opencode.json` cambios
3. `go build ./... && go test ./...`
4. Desinstalar Helix Skills si se instalaron

## Dependencies

- HelixDB corriendo en `localhost:6969` (existente)
- OpenCode ≥ versión con soporte MCP tools vía command mode
- `github.com/helixdb/helix-db/sdks/go` v0.1.1 (existente)
- `internal/taskcontext/`, `internal/db/helix/` (existente)
- `internal/context/bridge.go` — se mantiene, no se reemplaza

## Success Criteria

- [ ] `go build ./...` compila sin errores
- [ ] `go test ./internal/mcp/...` pasa
- [ ] MCP server inicia y responde a `task_context(1)` con contexto válido
- [ ] `search_code()` retorna CodeNodes por query
- [ ] `search_skills()` retorna skills globales
- [ ] `zyrocli context 1` emite warning deprecation
- [ ] OpenCode puede invocar las tres tools
- [ ] `go test ./...` sin regresiones en Fase 1-3
