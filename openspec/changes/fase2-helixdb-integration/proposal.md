# Proposal: Fase 2 — HelixDB Integration

## Intent

ZyroCLI necesita un almacén de conocimiento vivo. Fase 2 integra HelixDB como grafo con 8 labels (Developer → Project → Doc/Pattern/Library/Skill/CodeNode/Task) para inyectar contexto preciso en delegaciones, eliminando el grafo estático.

## Scope

### In Scope
- `internal/db/helix/schema.go` — CreateSchemaIndexes() idempotente (~80 líneas)
- `internal/db/helix/client.go` — Wrapper con tenant injection, helpers, lifecycle
- `zyrocli db init` — subcomando InitSchema
- `zyrocli absorb` — lee `.docs/`, parsea, crea Doc nodes

### Out of Scope
- CodeNode summaries (F3), Skill sharing (F3), `zyrocli context` (F3)
- Community detection, reemplazo openspec/, context bridge (F2.5)

## Capabilities

### New Capabilities
- `helixdb-core`: Schema índices, client wrapper con tenancy, lifecycle
- `zyrocli-db`: Subcomandos `db init` (schema) y `absorb` (ingesta `.docs/`)

### Modified Capabilities
- `zyrocli-run`: Añade subcomandos `db init`, `absorb`
- `doc-tools`: Absorb extiende ingesta documental a HelixDB

## Approach

1. `schema.go` — `CreateSchemaIndexes()` con índices equality, vector 1536d, text BM25, unique; todo idempotente
2. `client.go` — Wrapper inyecta `tenant_id`, expone helpers, maneja connect/disconnect
3. `cmd/zyrocli/db.go` — `zyrocli db init` → `InitSchema()`
4. `cmd/zyrocli/absorb.go` — Glob `.docs/`, parse frontmatter + body, crea Document nodes
5. Tenancy: `tenant_id` en cada nodo, filtro en queries, inyectado por wrapper

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/db/helix/schema.go` | New | CreateSchemaIndexes |
| `internal/db/helix/client.go` | New | Client + tenant injection |
| `cmd/zyrocli/db.go` | New | `zyrocli db init` |
| `cmd/zyrocli/absorb.go` | New | `zyrocli absorb` |
| `go.mod` / `go.sum` | Modified | +`helix-db/sdks/go` |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Sin schema validation | Medium | Structs Go antes de insertar |
| Sin RI en edges | Low | App layer validation |
| Tenant bug expone datos | Low | Wrapper central + test |
| SDK dynamic-first | Medium | Helpers tipados |
| HelixDB proceso externo | Low | Script `helix start dev --disk` |

## Rollback Plan

1. `git checkout HEAD -- internal/db/ cmd/zyrocli/db.go cmd/zyrocli/absorb.go go.mod go.sum`
2. `go build ./... && go test ./...`
3. `rm -rf ~/.helixdb/`

## Dependencies

- `github.com/helixdb/helix-db/sdks/go` — Go SDK
- HelixDB CLI dev local, puerto 6969

## Success Criteria

- [ ] `internal/db/helix/` compila con `go build ./...`
- [ ] `go test ./internal/db/helix/...` pasa con HelixDB corriendo
- [ ] `zyrocli db init` ejecuta CreateSchemaIndexes sin error
- [ ] `zyrocli absorb` lee `.docs/` y crea Doc nodes verificables
- [ ] Cada nodo lleva `tenant_id` correcto
- [ ] `go test ./...` sin regresiones en Fase 1
