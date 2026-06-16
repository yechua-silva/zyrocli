# Design: Fase 2 — HelixDB Integration

## Technical Approach

Wrapper sobre el Go SDK oficial de HelixDB con tenant injection automática. El wrapper expone operaciones CRUD tipadas, búsqueda semántica y gestión de edges, inyectando `tenant_id` en cada operación para aislamiento row-level. Schema idempotente via `CreateIndexIfNotExists`.

## Architecture Decisions

### Decision: Wrapper Pattern sobre Go SDK

**Choice**: Wrapper que encapsula `helixdb.Client` con tenant injection
**Alternatives considered**: 
- Usar Go SDK directamente en cada punto de uso
- Crear ORM layer completo
**Rationale**: El wrapper previene bugs de filtrado por tenant, centraliza manejo de errores y expone API más simple. No necesitamos ORM completo porque HelixDB es schemaless.

### Decision: Tenant Injection por Wrapper

**Choice**: Cada método que crea/query nodos inyecta `tenant_id` automáticamente
**Alternatives considered**: 
- Filtrar por tenant en cada query manualmente
- Usar HelixDB namespaces (no existen)
**Rationale**: Row-level tenancy es la única opción en HelixDB. Wrapper centraliza esto y previene olvidos.

### Decision: Schema Idempotente

**Choice**: `CreateIndexIfNotExists` se ejecuta en cada deploy
**Alternatives considered**: 
- Migraciones versionadas
- Validación de schema en app layer
**Rationale**: HelixDB es schemaless. Los índices son idempotentes. No hay migración, solo creación.

### Decision: Error Handling Tipado

**Choice**: Errores custom con `errors.New()` + wrapping
**Alternatives considered**: 
- Códigos de error numéricos
- Panic/recover
**Rationale**: Errores Go idiomáticos. Wrapping permite inspectar causas raíz.

## Data Flow

### Schema Initialization

```
zyrocli db init
    ↓
internal/db/helix.NewClient()
    ↓
client.InitSchema()
    ↓
CreateSchemaIndexes() → helixdb.WriteQuery
    ↓
client.Exec() → HelixDB (localhost:6969)
    ↓
Índices creados (idempotente)
```

### Document Absorption

```
zyrocli absorb
    ↓
Walk ./docs/ directory
    ↓
For each file:
    ├── Detect type (.md, .yaml, .json, .txt)
    ├── Infer doc_type ("handoff", "spec", etc.)
    ├── Read content
    └── UpsertDocument(topicKey, title, content, docType, tenantID)
        ↓
        helixdb.WriteQuery → HelixDB
        ↓
        Document node created/updated
```

### Tenant Injection Flow

```
CreateNode("Project", props)
    ↓
Wrapper adds tenant_id to props
    ↓
helixdb.G().AddN("Project", props+tenant_id)
    ↓
FindNodes("Project", filters)
    ↓
Wrapper adds tenant_id filter
    ↓
helixdb.NWithLabelWhere("Project", tenant_id=...)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/db/helix/client.go` | Create | Client wrapper + lifecycle (New, Close, Ping) |
| `internal/db/helix/schema.go` | Create | CreateSchemaIndexes() idempotente |
| `internal/db/helix/nodes.go` | Create | CRUD operations (CreateNode, GetNode, etc.) |
| `internal/db/helix/edges.go` | Create | Edge/traversal operations |
| `internal/db/helix/search.go` | Create | Vector/text search |
| `internal/db/helix/errors.go` | Create | Error types (ErrNotFound, etc.) |
| `cmd/zyrocli/db.go` | Create | `zyrocli db {init,status,reset}` commands |
| `cmd/zyrocli/absorb.go` | Create | `zyrocli absorb` command |
| `go.mod` | Modified | +`github.com/helixdb/helix-db/sdks/go` |

## Interfaces / Contracts

### Client Interface

```go
type Client struct {
    inner    *helixdb.Client
    tenantID string
}

type Option func(*Options)
type Options struct {
    TenantID string
    BaseURL  string  // default http://localhost:6969
}

// Lifecycle
func NewClient(ctx context.Context, opts ...Option) (*Client, error)
func (c *Client) Close() error
func (c *Client) Ping(ctx context.Context) bool

// Schema
func (c *Client) InitSchema(ctx context.Context) error

// Node CRUD (auto-inyecta tenant_id en props)
func (c *Client) CreateNode(ctx context.Context, label string, props map[string]interface{}) (int64, error)
func (c *Client) GetNode(ctx context.Context, id int64) (*Node, error)
func (c *Client) UpdateNode(ctx context.Context, id int64, props map[string]interface{}) error
func (c *Client) DeleteNode(ctx context.Context, id int64) error
func (c *Client) FindNodes(ctx context.Context, label string, filters map[string]interface{}) ([]*Node, error)

// Traversals
func (c *Client) GetOutgoing(ctx context.Context, nodeID int64, edgeLabel string) ([]*Node, error)
func (c *Client) GetIncoming(ctx context.Context, nodeID int64, edgeLabel string) ([]*Node, error)
func (c *Client) CreateEdge(ctx context.Context, fromID, toID int64, label string, props map[string]interface{}) (int64, error)
func (c *Client) DeleteEdge(ctx context.Context, edgeID int64) error

// Semantic search
func (c *Client) VectorSearch(ctx context.Context, label string, embedding []float32, k int) ([]*Node, error)
func (c *Client) TextSearch(ctx context.Context, label string, query string) ([]*Node, error)
```

### Node Type

```go
type Node struct {
    ID    int64
    Label string
    Props map[string]interface{}
}
```

### Error Types

```go
var (
    ErrNotFound       = errors.New("helix: node not found")
    ErrConnection     = errors.New("helix: connection failed")
    ErrUnauthorized   = errors.New("helix: unauthorized")
    ErrTenantMismatch = errors.New("helix: tenant mismatch")
)
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Tenant injection en CreateNode | Mock helixdb.Client, verificar props |
| Unit | Filtro por tenant en FindNodes | Mock helixdb.Client, verificar query |
| Unit | InitSchema llama CreateIndexIfNotExists | Mock, verificar llamadas |
| Unit | UpsertDocument logic | Mock, verificar create vs update |
| Integration | Absorb con directorio temporal | Crear .docs/ temporal, ejecutar absorb |
| E2E | zyrocli db init + absorb | Requiere HelixDB corriendo |

**No integration tests** en CI (requieren HelixDB). Tests unitarios mockeados son suficientes para validación.

## Migration / Rollout

1. **Instalar dependencia**: `go get github.com/helixdb/helix-db/sdks/go`
2. **Crear package `internal/db/helix/`**: 6 archivos, ~300 líneas total
3. **Crear comandos CLI**: `db.go` + `absorb.go`
4. **Probar localmente**: `zyrocli db init` → verificar índices en HelixDB
5. **Probar absorb**: `zyrocli absorb` → verificar Document nodes

No hay migración de datos existentes. HelixDB arranca vacío.

## Open Questions

- [ ] ¿HelixDB local (localhost:6969) es suficiente para development, o necesitamos cloud para testing?
- [ ] ¿El tenant_id debe ser `Developer.name` o un UUID separado?
- [ ] ¿Absorb debe soportar frontmatter YAML en .md files, o solo contenido raw?
- [ ] ¿Necesitamos `zyrocli db reset` con confirmación interactiva (bubbletea)?
