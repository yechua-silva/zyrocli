# Investigación: Schema HelixDB en HQL (HelixDB Query Language)

> **Fecha**: 2026-06-14
> **HelixDB Version**: v3.0.5 (Cloud / Enterprise)
> **SDK Go**: `github.com/helixdb/helix-db/sdks/go`
> **Propósito**: Validar que el schema propuesto en `architecture-v2.md` es viable en HelixDB, con Go SDK real.

---

## Resumen Ejecutivo

| Pregunta | Respuesta |
|----------|-----------|
| ¿HelixDB tiene schema explícito? | **NO. Es schemaless.** No hay CREATE TABLE. Los nodos tienen labels (stored como `$label`) y propiedades tipadas. |
| ¿Cómo se definen tipos de nodo? | Por **label** en `addN("LabelName", props)`. La label es un string. |
| ¿Cómo se modelan edges con tipo? | Por **label** en `addE("EDGE_LABEL", targetNodeRef, props)`. Los edges son **dirigidos** (source → target). |
| ¿Soporta tenancy nativo? | **Row-level tenancy** via `tenant_id` property + equality index + query filtering. No namespaces separados. |
| ¿Hay migración de schema? | **NO.** Es puramente schemaless. Los índices se crean idempotentemente con `CreateIndexIfNotExists`. |
| ¿El schema propuesto es viable? | **SÍ, con ajustes menores.** Ver detalles abajo. |

---

## 1. Cómo Funciona el Data Model de HelixDB

### 1.1 Nodos y Labels

HelixDB es un **labeled property graph**. Los nodos tienen:
- **`$id`**: 64-bit unsigned ID (asignado automáticamente)
- **`$label`**: string guardado como propiedad reservada
- **Propiedades**: tipadas (bool, i64, f64, string, bytes, arrays, objects)

**No existe definición de schema previa.** Simplemente creás el nodo con la label que quieras:

```go
// Crear nodo Developer — no necesita CREATE TABLE previo
helix.G().AddN("Developer", helix.Props{
    helix.Prop("name", "Sebastian"),
    helix.Prop("default_tech_stack", "Go,TypeScript"),
    helix.Prop("culture", "clean-arch"),
})
```

La label es arbitraria. Podés crear un nodo `"Developer"` sin haber declarado esa label antes.

### 1.2 Propiedades Tipadas

Soporta: `bool`, `i64`, `f64`, `string`, `bytes`, `[]f32`, `[]string`, objetos anidados.

```go
// Propiedades simples
helix.Prop("name", "Sebastian")
helix.Prop("status", "active")
helix.Prop("version", 3)

// Propiedades array (para embeddings)
helix.Prop("embedding", []float32{0.12, 0.85, -0.04})

// Objetos anidados
helix.Prop("metadata", helix.ObjectFromEntries(
    helix.Entry("externalID", "crm-42"),
    helix.Entry("tags", helix.Array(helix.String("trial"), helix.I64(7))),
))
```

**Importante**: Los campos indexados DEBEN ser propiedades de nivel superior (no nested). Los objetos anidados son queryeables con dotted paths (`metadata.externalID`) pero NO son indexables en V1.

### 1.3 Edges Dirigidos

Los edges son **siempre dirigidos**: source → target. Cada edge tiene:
- **`$id`**: ID único
- **`$label`**: tipo de relación
- **Propiedades**: tipadas igual que nodos

```go
// Crear edge HAS_PROJECT de Developer → Project
helix.G().
    N(helix.NodeVar("developer_node")).
    AddE("HAS_PROJECT", helix.NodeVar("project_node"), helix.Props{
        helix.Prop("created_at", "2026-06-14"),
    })
```

**Multigraph**: Soporta múltiples edges entre el mismo par de nodos (edge IDs únicos).

### 1.4 ¿Qué NO Soporta HelixDB?

| Feature | Soporte | Workaround |
|---------|---------|------------|
| Schema explícito / validación de tipos | ❌ | Validar en capa Go antes de insertar |
| Unique constraints en properties | ✅ (unique equality index) | `NodeUniqueEqualityIndex` |
| Foreign keys / referential integrity | ❌ | Validar en app layer |
| Migración de schema | ❌ | Es schemaless. Los índices son idempotentes |
| Namespace/nivel de aislamiento por DB | ❌ (solo row-level) | Filtrar por `tenant_id` en cada query |
| Community detection (graph algorithms) | ❌ | Implementar externo o en Go |

---

## 2. Go SDK — API Completa

### 2.1 Instalación

```bash
go get github.com/helixdb/helix-db/sdks/go
```

```go
import helix "github.com/helixdb/helix-db/sdks/go"
```

### 2.2 Cliente

```go
client, err := helix.NewClient("http://localhost:6969") // local dev
// o
client, err := helix.NewClient("https://helix.example.com", helix.WithAPIKey("hx_secret"))
```

### 2.3 Crear Nodo (WriteQuery)

```go
func CreateProject(name, description, repoPath string) helix.Request {
    q := helix.WriteQuery("create_project")
    
    nameParam := q.ParamString("name", name)
    descParam := q.ParamString("description", description)
    pathParam := q.ParamString("repo_path", repoPath)
    
    return q.
        VarAs("project",
            helix.G().AddN("Project", helix.Props{
                helix.Prop("name", nameParam),
                helix.Prop("description", descParam),
                helix.Prop("repo_path", pathParam),
                helix.Prop("status", "active"),
                helix.Prop("current_phase", "phase0"),
            }).
            Project(helix.ProjectPropAs("$id", "id")),
        ).
        Returning("project")
}

// Ejecutar
var result struct {
    Project []struct {
        ID   int64  `json:"id"`
        Name string `json:"name"`
    } `json:"project"`
}
err := client.Exec(ctx, CreateProject("zyro", "AI agent CLI", "/home/user/zyro"), &result)
```

### 2.4 Crear Edge entre Nodos Existentes

```go
func LinkDeveloperToProject(devID, projectID uint64) helix.Request {
    return helix.WriteQuery("link_dev_project").
        VarAs("edge",
            helix.G().
                N(helix.NodeID(devID)).
                AddE("HAS_PROJECT", helix.NodeID(projectID), helix.Props{
                    helix.Prop("created_at", "2026-06-14T00:00:00Z"),
                }).
                Count(),
        ).
        Returning("edge")
}
```

### 2.5 Query: Skills que usa un Project (ReadQuery)

```go
func GetProjectSkills(projectID uint64) helix.Request {
    return helix.ReadQuery("get_project_skills").
        VarAs("skills",
            helix.G().
                N(helix.NodeID(projectID)).
                Out("REQUIRES_SKILL").        // seguir edge REQUIRES_SKILL
                Dedup().                        // Skills son compartidos
                Project(
                    helix.ProjectPropAs("$id", "skill_id"),
                    helix.ProjectProp("name"),
                    helix.ProjectProp("type"),
                ),
        ).
        Returning("skills")
}
```

### 2.6 Búsqueda Semántica de Patrones Similares

```go
func FindSimilarPatterns(queryVector []float32, tenantID string, k int64) helix.Request {
    q := helix.ReadQuery("find_similar_patterns")
    
    vecParam := q.ParamArray("query_vector", queryVector, helix.ParamTypeF32())
    kParam := q.ParamI64("k", k)
    tenantParam := q.ParamString("tenant_id", tenantID)
    
    return q.
        VarAs("hits",
            helix.G().
                VectorSearchNodesWith(
                    "Pattern",
                    "embedding",
                    vecParam.Input(),
                    kParam.Bound(),
                    &tenantParam,  // tenant-scoped index
                ).
                Project(
                    helix.ProjectPropAs("$id", "pattern_id"),
                    helix.ProjectPropAs("$distance", "similarity"),
                    helix.ProjectProp("name"),
                    helix.ProjectProp("description"),
                ),
        ).
        Returning("hits")
}
```

### 2.7 Índices (idempotentes, en WriteQuery)

```go
func CreateSchemaIndexes() helix.Request {
    return helix.WriteQuery("create_indexes").
        // Equality indexes — fast path lookups
        VarAs("idx_dev_name",
            helix.G().CreateIndexIfNotExists(
                helix.NodeUniqueEqualityIndex("Developer", "name"),
            ),
        ).
        VarAs("idx_proj_name",
            helix.G().CreateIndexIfNotExists(
                helix.NodeEqualityIndex("Project", "name"),
            ),
        ).
        VarAs("idx_skill_name",
            helix.G().CreateIndexIfNotExists(
                helix.NodeUniqueEqualityIndex("Skill", "name"),
            ),
        ).
        VarAs("idx_doc_topic",
            helix.G().CreateIndexIfNotExists(
                helix.NodeEqualityIndex("Document", "topic_key"),
            ),
        ).
        VarAs("idx_task_status",
            helix.G().CreateIndexIfNotExists(
                helix.NodeEqualityIndex("Task", "status"),
            ),
        ).
        // Tenant-scoped equality for row-level isolation
        VarAs("idx_proj_tenant",
            helix.G().CreateIndexIfNotExists(
                helix.NodeEqualityIndex("Project", "tenant_id"),
            ),
        ).
        // Vector indexes — semantic search (1536-dim cosine)
        VarAs("idx_pattern_emb",
            helix.G().CreateVectorIndexNodes("Pattern", "embedding", "tenant_id"),
        ).
        VarAs("idx_doc_emb",
            helix.G().CreateVectorIndexNodes("Document", "embedding", "tenant_id"),
        ).
        VarAs("idx_skill_emb",
            helix.G().CreateVectorIndexNodes("Skill", "embedding", "tenant_id"),
        ).
        // Text indexes — BM25 full-text search
        VarAs("idx_doc_content",
            helix.G().CreateTextIndexNodes("Document", "content", "tenant_id"),
        ).
        Returning() // void — idempotent schema setup
}
```

---

## 3. Schema Propuesto — Mapeo a HelixDB

### 3.1 Nodos

| Nodo HelixDB | Label | Propiedades Principales | Index |
|-------------|-------|------------------------|-------|
| Developer | `Developer` | name (unique), default_tech_stack, culture, playbook_ref | `NodeUniqueEqualityIndex("Developer", "name")` |
| Project | `Project` | name, description, status, current_phase, repo_path, tenant_id | `NodeEqualityIndex("Project", "name")`, `NodeEqualityIndex("Project", "tenant_id")` |
| Document | `Document` | doc_type(spec\|design\|adr\|handoff\|decision), title, content, topic_key, tenant_id | `NodeEqualityIndex("Document", "topic_key")`, `NodeVectorIndex("Document", "embedding", "tenant_id")`, `NodeTextIndex("Document", "content", "tenant_id")` |
| Pattern | `Pattern` | name, category(architectural\|design\|testing), description, tenant_id | `NodeEqualityIndex("Pattern", "name")`, `NodeVectorIndex("Pattern", "embedding", "tenant_id")` |
| Library | `Library` | name, version, category, description, validated_at, tenant_id | `NodeEqualityIndex("Library", "name")` |
| Skill | `Skill` | name, type(BE\|FE\|DevOps), source_url, validated_at, version, tenant_id | `NodeUniqueEqualityIndex("Skill", "name")`, `NodeVectorIndex("Skill", "embedding", "tenant_id")` |
| CodeNode | `CodeNode` | module_type(function\|type\|file\|module), name, path, language, signature, summary, tenant_id | `NodeEqualityIndex("CodeNode", "path")` |
| Task | `Task` | description, phase, status, created_at, tenant_id | `NodeEqualityIndex("Task", "status")`, `NodeRangeIndex("Task", "created_at")` |

### 3.2 Edges

| Edge | Source → Target | Propiedades |
|------|----------------|-------------|
| `HAS_PROJECT` | Developer → Project | created_at |
| `HAS_DOC` | Project → Document | added_at |
| `HAS_PATTERN` | Project → Pattern | confidence |
| `USES_LIB` | Project → Library | since |
| `REQUIRES_SKILL` | Project → Skill | required_level |
| `HAS_CODENODE` | Project → CodeNode | discovered_at |
| `HAS_TASK` | Project → Task | assigned_at |
| `REFERENCES` | Task → CodeNode | context |
| `REQUIRES` | Task → Skill | skill_level |

### 3.3 Tenancy — Aislamiento Row-Level

**Patrón**: Cada nodo que pertenece a un Developer lleva `tenant_id` como propiedad. Se filtra en TODAS las queries.

```go
// Crear nodo con tenant_id
helix.G().AddN("Project", helix.Props{
    helix.Prop("tenant_id", "sebastian"),  // Developer.name como tenant
    helix.Prop("name", "zyro"),
    // ...
})

// Query filtrada por tenant
helix.G().
    NWithLabelWhere("Project", helix.SourceEq("tenant_id", "sebastian")).
    Out("HAS_DOC").
    Project(helix.ProjectProp("title"))
```

**Skill compartido**: Los Skills usan `tenant_id` del Developer que los creó. Cuando un Project los referencia, el edge `REQUIRES_SKILL` no necesita `tenant_id` — se resuelve por traversal.

### 3.4 Flujo de Inicialización

```go
// Enzyrocli db init — crea schema completo
func InitSchema(ctx context.Context, client *helix.Client) error {
    return client.Exec(ctx, CreateSchemaIndexes(), nil,
        helix.WriterOnly(),
        helix.AwaitDurability(true),
    )
}
```

---

## 4. Ejemplos Completos en Go SDK

### 4.1 Crear Developer + Project + Edge en una Transacción

```go
func InitDeveloperWithProject(
    ctx context.Context,
    client *helix.Client,
    devName, projectName, description, repoPath string,
) error {
    q := helix.WriteQuery("init_dev_project")
    
    devNameParam := q.ParamString("dev_name", devName)
    projNameParam := q.ParamString("project_name", projectName)
    descParam := q.ParamString("description", description)
    pathParam := q.ParamString("repo_path", repoPath)
    
    return client.Exec(ctx,
        q.VarAs("developer",
            helix.G().AddN("Developer", helix.Props{
                helix.Prop("name", devNameParam),
            }).
            Project(helix.ProjectPropAs("$id", "dev_id")),
        ).
        VarAs("project",
            helix.G().AddN("Project", helix.Props{
                helix.Prop("tenant_id", devNameParam),
                helix.Prop("name", projNameParam),
                helix.Prop("description", descParam),
                helix.Prop("repo_path", pathParam),
                helix.Prop("status", "active"),
                helix.Prop("current_phase", "phase0"),
            }).
            Project(helix.ProjectPropAs("$id", "proj_id")),
        ).
        VarAs("link",
            helix.G().
                N(helix.NodeVar("developer")).
                Out().  // get developer node
                AddE("HAS_PROJECT", helix.NodeVar("project"), helix.Props{}).
                Count(),
        ).
        Returning("developer", "project", "link"),
        nil,
    )
}
```

### 4.2 Context Injection — Query para Subagente

```go
// "Dame todo el contexto relevante para trabajar en este task"
func GetTaskContext(projectID uint64, taskDesc string) helix.Request {
    q := helix.ReadQuery("task_context")
    descParam := q.ParamString("task_desc", taskDesc)
    
    return q.
        // 1. Skills que el proyecto necesita
        VarAs("skills",
            helix.G().
                N(helix.NodeID(projectID)).
                Out("REQUIRES_SKILL").
                Dedup().
                Project(
                    helix.ProjectPropAs("$id", "skill_id"),
                    helix.ProjectProp("name"),
                    helix.ProjectProp("type"),
                ),
        ).
        // 2. Patrones del proyecto
        VarAs("patterns",
            helix.G().
                N(helix.NodeID(projectID)).
                Out("HAS_PATTERN").
                Project(
                    helix.ProjectPropAs("$id", "pattern_id"),
                    helix.ProjectProp("name"),
                    helix.ProjectProp("description"),
                ),
        ).
        // 3. CodeNodes del proyecto (summaries de módulos)
        VarAs("code_nodes",
            helix.G().
                N(helix.NodeID(projectID)).
                Out("HAS_CODENODE").
                Project(
                    helix.ProjectPropAs("$id", "codenode_id"),
                    helix.ProjectProp("name"),
                    helix.ProjectProp("path"),
                    helix.ProjectProp("summary"),
                ),
        ).
        // 4. Documentos relevantes (specs, decisiones)
        VarAs("documents",
            helix.G().
                N(helix.NodeID(projectID)).
                Out("HAS_DOC").
                Project(
                    helix.ProjectPropAs("$id", "doc_id"),
                    helix.ProjectProp("doc_type"),
                    helix.ProjectProp("title"),
                    helix.ProjectProp("content"),
                ),
        ).
        Returning("skills", "patterns", "code_nodes", "documents")
}
```

### 4.3 Upsert con varAsIf

```go
func UpsertSkill(name, skillType, sourceURL, tenantID string) helix.Request {
    q := helix.WriteQuery("upsert_skill")
    nameParam := q.ParamString("name", name)
    typeParam := q.ParamString("skill_type", skillType)
    urlParam := q.ParamString("source_url", sourceURL)
    tenantParam := q.ParamString("tenant_id", tenantID)
    
    return q.
        // Buscar si ya existe
        VarAs("existing",
            helix.G().
                NWithLabelWhere("Skill", helix.SourceEq("name", nameParam)).
                Where(helix.PredEq("tenant_id", tenantParam)),
        ).
        // Si existe → actualizar version
        VarAsIf("updated",
            helix.VarNotEmpty("existing"),
            helix.G().
                N(helix.NodeVar("existing")).
                SetProperty("version", helix.ExprAdd(
                    helix.ExprProp("version"),
                    helix.ExprVal(int64(1)),
                )),
        ).
        // Si NO existe → crear
        VarAsIf("created",
            helix.VarEmpty("existing"),
            helix.G().AddN("Skill", helix.Props{
                helix.Prop("name", nameParam),
                helix.Prop("type", typeParam),
                helix.Prop("source_url", urlParam),
                helix.Prop("tenant_id", tenantParam),
                helix.Prop("version", int64(1)),
                helix.Prop("validated_at", helix.ExprDateTime()),
            }),
        ).
        Returning("updated", "created")
}
```

---

## 5. Riesgos y Limitaciones

### 5.1 Limitaciones Técnicas

| Riesgo | Impacto | Mitigación |
|--------|---------|------------|
| **Sin schema validation** | Se pueden crear nodos con propiedades inconsistentes | Validar en capa Go con structs/Pydantic antes de insertar |
| **Sin referential integrity** | Un edge puede apuntar a un nodo borrado | Validar existencia en app layer; cascade delete manual |
| **Índices solo nivel superior** | Nested objects no son indexables | Mantener propiedades indexadas como top-level |
| **Row-level tenancy only** | No aislamiento real por namespace | Usar `tenant_id` en TODA query + vector/text indexes particionados |
| **No community detection** | Grafo sin algoritmos de graph analytics | Implementar en Go si se necesita (BFS, PageRank, etc.) |
| **Deployment sidecar** | HelixDB corre como Docker container separado | No embebido en Go binary. Latencia HTTP localhost ~1ms |

### 5.2 Riesgos de Diseño

| Riesgo | Detalle |
|--------|---------|
| **Skill sharing complexity** | Skills son compartidos entre proyectos pero con diferentes `required_level`. El edge `REQUIRES_SKILL` maneja esto via propiedades del edge |
| **Tenant ID propagation** | Cada query DEBE filtrar por `tenant_id`. Un bug aquí filtra datos de otros devs. **Mitigación**: wrapper Go que inyecta `tenant_id` automáticamente |
| **Embedding dimension** | Si se mezclan modelos de embedding (1536 vs 768), los vectores no son comparables. **Mitigación**: estandarizar un modelo por label |

---

## 6. Recomendación: ¿El Schema Propuesto es Viable?

### Veredicto: **SÍ, con ajustes menores**

**Lo que funciona tal cual:**
- Developer como tenant root ✅
- Project → Skill sharing via edges ✅
- Documentos con topic_key + embeddings ✅
- CodeNode summaries (no código completo) ✅
- Task → CodeNode references ✅

**Ajustes necesarios:**

1. **Agregar `tenant_id` a TODOS los nodos por proyecto** — no solo a Project. Cada Document, Pattern, Library, CodeNode, Task necesita `tenant_id` para row-level isolation.

2. **Skill pool: usar `NodeUniqueEqualityIndex` en `Skill.name`** — ya está en el schema. Skills compartidos se resuelven por nombre único, no por ID duplicado.

3. **No intentar community detection en HelixDB** — si se necesita, hacerlo en Go post-query.

4. **Wrapper Go para inyección automática de tenant** — cada query debe pasar por un wrapper que agrega `tenant_id` filter. Esto previene bugs de filtrado.

5. **Schema initialization idempotente** — `CreateIndexIfNotExists` se ejecuta en cada deploy. No hay migración, solo creación de índices.

### Próximo Paso

Crear `internal/db/helix/schema.go` con el `CreateSchemaIndexes()` completo y `internal/db/helix/client.go` con el wrapper de tenancy. Estos ~200-300 líneas de Go son el core de la integración.
