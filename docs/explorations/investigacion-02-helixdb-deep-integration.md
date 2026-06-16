# Investigación: HelixDB Deep Integration

**Fecha**: 2026-06-15
**Proyecto**: ZyroAgentCLI
**Autor**: Investigación técnica  
**Estado**: Completa
**Versión**: 1.0

---

## Resumen Ejecutivo

HelixDB es una base de datos OLTP **grafo + vector** construida en Rust. Ofrece búsqueda ANN (k-NN aproximado), BM25 full-text, transacciones ACID, y un modelo de datos de grafo de propiedades etiquetadas — todo en un solo sistema, sin necesidad de integrar bases separadas.

ZyroAgentCLI ya tiene integración HelixDB funcional:
- Cliente Go en `internal/db/helix/` (CRUD nodos/edges, text search, vector search, schema)
- Cliente Python en `mcp-tools/helix_client.py` (5 MCP tools: task_context, search_code, search_skills, save_to_helix, link_to_project, find_project)

**Sin embargo, la integración actual solo aprovecha ~30% de las capacidades de HelixDB.** Este documento identifica las brechas y prescribe cómo cerrarlas usando el Go SDK oficial, patrones del `helix-memory-system`, búsqueda híbrida, y un MCP server propio.

---

## Capacidades No Aprovechadas Actualmente

| Capacidad | Estado Actual | Potencial |
|-----------|--------------|-----------|
| **Go SDK oficial** | Cliente propio con raw JSON | Usar `github.com/helixdb/helix-db/sdks/go` con tipos, params, `client.Exec` |
| **Parámetros dinámicos** | Literales hardcodeados | `q.ParamString`, `q.ParamI64`, `q.ParamDateTime` |
| **Traversals complejos** | Solo `Out`/`In` simple | `Repeat`, `Union`, `Choose`, `Coalesce`, `Optional`, multi-hop |
| **Búsqueda vectorial** | No se usa en MCP tools | `VectorSearchNodes` con embeddings reales |
| **Búsqueda híbrida** | No existe | Fusión app-side de vector + BM25 |
| **Índices vector/text** | Solo `CreateIndex` para texto | `CreateIndexIfNotExists` con `IndexSpec` completo |
| **Escrituras batch** | Nodo por nodo | `ForEachParam`, `writeBatch` con múltiples `varAs` |
| **Condicionales** | No existen | `varAsIf` con `BatchCondition` (upsert, fetch-conditional) |
| **Proyecciones** | Solo `id` | `ValueMap`, `Project` con alias, `$distance` |
| **Pipeline de embeddings** | No existe | Embeddings externos (OpenAI text-embedding-3-small) → F32 array |
| **MCP server Go** | Python FastMCP (stdio) | Go MCP server nativo con `mark3labs/mcp-go` |
| **Graph traversal semántico** | No implementado | Caminatas: "qué skills requiere este proyecto → qué otros proyectos la usan" |
| **Multi-tenencia** | `project_id` manual | `tenant_id` como partición nativa en índices vector/text |
| **Lifecycle de memoria** | No implementado | `isLatest`, `validFrom`, `validTo`, `deletedAt`, `expiresAt` |

---

## SDK Go: Cómo Usar Queries, Writes, Traversals

### Arquitectura del SDK Oficial

El módulo Go oficial es `github.com/helixdb/helix-db/sdks/go` (importado como `helix`).

**Patrón general:**

```go
import helix "github.com/helixdb/helix-db/sdks/go"

// 1. Definir tipos de respuesta
type UserRow struct {
    ID   int64  `json:"$id"`
    Name string `json:"name"`
}

type FindUsersResponse struct {
    Users []UserRow `json:"users"`
}

// 2. Escribir función query que devuelve helix.Request
func FindUsers(tenantID string, limit int64) helix.Request {
    q := helix.ReadQuery("find_users")
    tenant := q.ParamString("tenant_id", tenantID)
    maxRows := q.ParamI64("limit", limit)

    return q.
        VarAs("users",
            helix.G().
                NWithLabel("User").
                Where(helix.PredEq("tenantId", tenant)).
                Limit(maxRows).
                ValueMap("$id", "name", "tenantId"),
        ).
        Returning("users")
}

// 3. Ejecutar con client.Exec
client, _ := helix.NewClient("http://localhost:6969")
var out FindUsersResponse
err = client.Exec(ctx, FindUsers("acme", 25), &out)
```

### Diferencia Clave vs Cliente Actual

| Aspecto | Cliente Actual (`internal/db/helix/`) | SDK Oficial |
|---------|---------------------------------------|-------------|
| Construcción de query | Raw `map[string]any` JSON | DSL fluido con tipos |
| Parámetros | Valores inline en AST | `ParamString`/`ParamI64` con type hints |
| Response parsing | Manual con `json.RawMessage` | `client.Exec` con structs tipados |
| Tipos de error | `ErrNotFound`, `ErrConnectionFailed` | `*helix.HelixError` con `StatusCode` |
| Conflict handling | No implementado | `helix.IsConflict(err)`, `helix.ErrConflict` |

### Traversals Complejos

El SDK oficial soporta estos pasos de traversal que el cliente actual no usa:

**Repeat (caminatas de longitud variable):**
```go
helix.G().
    N(helix.NodeVar("user")).
    Repeat(helix.Repeat(helix.Sub().Out("FOLLOWS")).WithTimes(3).EmitAll()).
    Dedup().
    ValueMap("$id", "username")
```

**Union (múltiples caminos):**
```go
helix.G().
    N(helix.NodeVar("user")).
    Union(helix.Sub().Out("AUTHORED"), helix.Sub().Out("LIKED")).
    Dedup().
    ValueMap("$id", "title")
```

**Choose (if/else por nodo):**
```go
helix.G().
    NWithLabel("User").
    Choose(
        helix.PredEq("tier", "pro"),
        helix.Sub().Out("PRO_FEED"),
        helix.Sub().Out("DEFAULT_FEED"),
    ).
    ValueMap("$id", "title")
```

**Coalesce (primer branch no vacío):**
```go
helix.G().
    N(helix.NodeVar("user")).
    Coalesce(
        helix.Sub().Out("PRIMARY_FEED"),
        helix.Sub().Out("DEFAULT_FEED"),
    )
```

### Escrituras Batch

```go
func SeedData() helix.Request {
    return helix.WriteQuery("seed_data").
        VarAs("alice",
            helix.G().
                AddN("User", helix.Props{
                    helix.Prop("username", "alice"),
                    helix.Prop("tier", "pro"),
                }).
                Project(helix.ProjectPropAs("$id", "id")),
        ).
        VarAs("bob",
            helix.G().
                AddN("User", helix.Props{
                    helix.Prop("username", "bob"),
                    helix.Prop("tier", "free"),
                }).
                Project(helix.ProjectPropAs("$id", "id")),
        ).
        VarAs("edge",
            helix.G().
                N(helix.NodeVar("alice")).
                AddE("FOLLOWS", helix.NodeVar("bob"), helix.Props{
                    helix.Prop("since", "2026-05-01"),
                }).
                Count(),
        ).
        Returning("alice", "bob", "edge")
}
```

### Upsert Condicional

```go
func UpsertUser(username string, tier string) helix.Request {
    q := helix.WriteQuery("upsert_user")
    usernameParam := q.ParamString("username", username)
    tierParam := q.ParamString("tier", tier)

    return q.
        VarAs("existing",
            helix.G().NWithLabel("User").Where(helix.PredEq("username", usernameParam)),
        ).
        VarAsIf("updated",
            helix.VarNotEmpty("existing"),
            helix.G().N(helix.NodeVar("existing")).SetProperty("tier", tierParam),
        ).
        VarAsIf("created",
            helix.VarEmpty("existing"),
            helix.G().AddN("User", helix.Props{
                helix.Prop("username", usernameParam),
                helix.Prop("tier", tierParam),
            }),
        ).
        Returning("updated", "created")
}
```

### Bulk Insert con ForEachParam

```go
func BulkAddUsers(users []map[string]any) helix.Request {
    q := helix.WriteQuery("bulk_add_users")
    q.ParamArray("users", users, helix.ParamTypeObject())

    return q.
        ForEachParam("users",
            helix.Write().VarAs("created",
                helix.G().AddN("User", helix.Props{
                    helix.Prop("username", helix.ParamInput("username")),
                    helix.Prop("tier", helix.ParamInput("tier")),
                }),
            ),
        ).
        Returning("created")
}
```

---

## MCP Macro: Conversión HelixQL → MCP Tool

### Hallazgo: No Existe "MCP Macro" Como Producto

La URL `https://docs.helix-db.com/mcp-macro` devuelve 404. No hay un producto llamado "MCP Macro" en HelixDB. El concepto que probablemente se refiere es a cómo **exponer consultas HelixQL como tools MCP**, lo cual es exactamente lo que ZyroAgentCLI ya hace con su MCP server Python.

### Estado Actual de la Integración MCP

ZyroAgentCLI tiene un MCP server Python (FastMCP) con 6 tools:

| Tool | Función | Limitación |
|------|---------|------------|
| `task_context` | Obtener contexto completo de una tarea | Solo traversal Out/In básico |
| `search_code` | Buscar CodeNodes por texto | Solo BM25, sin vector search |
| `search_skills` | Buscar Skills por texto | Solo BM25, sin vector search |
| `save_to_helix` | Crear nodo | Sin upsert, sin validación de existencia |
| `link_to_project` | Crear edge | Sin verificación de duplicados |
| `find_project` | Buscar proyecto por nombre | Solo text search limit=1 |

### Patrón para Exponer HelixQL como MCP Tool

Cada tool MCP sigue este patrón de 3 capas:

1. **Tool definition** (MCP schema) — declara nombre, descripción, input schema
2. **Query builder** (Go SDK / Python HelixClient) — construye la request HelixQL
3. **Response formatter** — transforma la respuesta raw en formato útil para el agente

**Ejemplo: tool híbrida search_code (vector + BM25)**

```python
async def search_code_hybrid_tool(query: str, limit: int = 10) -> str:
    """Search code by vector similarity AND BM25, fused with RRF."""
    client = HelixClient()

    # Run vector search and text search in parallel
    vector_results = await client.vector_search("CodeNode", "embedding", query, limit)
    text_results = await client.text_search("CodeNode", "summary", query, limit)

    # Fuse with Reciprocal Rank Fusion
    fused = rrf_fuse(vector_results, text_results, k=60)
    return json.dumps({"query": query, "count": len(fused), "results": fused[:limit]}, indent=2)
```

### Tools Propuestas para MCP Server Go

Basado en el patrón del `helix-memory-system` y las necesidades de ZyroAgentCLI:

```go
// Tools de lectura (MCP)
tools := []mcp.Tool{
    {
        Name: "task_context",
        Description: "Get full context for a task: skills, code nodes, docs, patterns, dependencies",
        InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "task_id": {"type": "integer", "description": "HelixDB node ID"}
            },
            "required": ["task_id"]
        }`),
    },
    {
        Name: "search_code",
        Description: "Hybrid search: vector + BM25 fusion on code nodes",
        InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "query": {"type": "string"},
                "k": {"type": "integer", "default": 10}
            },
            "required": ["query"]
        }`),
    },
    {
        Name: "search_skills",
        Description: "Find skills by BM25 text search",
        InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "query": {"type": "string"},
                "k": {"type": "integer", "default": 10}
            },
            "required": ["query"]
        }`),
    },
    {
        Name: "graph_query",
        Description: "Execute a custom graph traversal query",
        InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "start_label": {"type": "string"},
                "relation": {"type": "string"},
                "direction": {"type": "string", "enum": ["out", "in", "both"]}
            },
            "required": ["start_label", "relation"]
        }`),
    },
    {
        Name: "project_context",
        Description: "Get project overview with all related nodes",
        InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "project_id": {"type": "integer"}
            },
            "required": ["project_id"]
        }`),
    },
}
```

### Ejemplo: Task Context con Go SDK + MCP

```go
func buildTaskContextQuery(taskID int64) helix.Request {
    q := helix.ReadQuery("task_context")

    return q.
        VarAs("task",
            helix.G().NWhere(helix.SourceEq("$id", taskID)).
            ValueMap("$id", "name", "description", "status"),
        ).
        VarAsIf("skills",
            helix.VarNotEmpty("task"),
            helix.G().
                N(helix.NodeVar("task")).
                Out("has_skill").
                ValueMap("$id", "name"),
        ).
        VarAsIf("code",
            helix.VarNotEmpty("task"),
            helix.G().
                N(helix.NodeVar("task")).
                Out("has_code").
                ValueMap("$id", "path", "summary", "language"),
        ).
        VarAsIf("docs",
            helix.VarNotEmpty("task"),
            helix.G().
                N(helix.NodeVar("task")).
                Out("has_doc").
                ValueMap("$id", "name"),
        ).
        VarAsIf("dependents",
            helix.VarNotEmpty("task"),
            helix.G().
                N(helix.NodeVar("task")).
                In("depends_on").
                ValueMap("$id", "name"),
        ).
        Returning("task", "skills", "code", "docs", "dependents")
}
```

---

## Búsqueda Híbrida (Vector + BM25): Ejemplos y Configuración

### Cómo Funciona en HelixDB

HelixDB **no tiene búsqueda híbrida nativa** (no hay un solo paso que fusione vector + BM25). La fusión debe hacerse **app-side**. La ventaja de HelixDB es que ambos índices coexisten en los mismos nodos en la misma base de datos — no hay que integrar sistemas separados.

### Arquitectura de Búsqueda Híbrida

```
Query del agente
    │
    ├──→ VectorSearchNodes (ANN sobre embeddings)
    │       Returns: [{node, $distance}, ...]
    │
    ├──→ TextSearchNodes (BM25 sobre texto)
    │       Returns: [{node, $distance}, ...]
    │
    └──→ Fusion App-side (RRF)
            scores = RRF(vector_hits, text_hits, k=60)
            ranked = sort by scores
            Return: top N resultados fusionados
```

### Implementación en Python

```python
async def hybrid_search(label: str, query: str, vector: list[float], k: int = 10) -> list[dict]:
    """Hybrid search: vector + BM25 fused with Reciprocal Rank Fusion."""
    client = HelixClient()
    tasks = []

    if vector:
        tasks.append(client.vector_search(label, "embedding", vector, k))
    if query:
        tasks.append(client.text_search(label, "content", query, k))

    results = await asyncio.gather(*tasks)
    return rrf_fuse(*results, k=60)[:k]


def rrf_fuse(*ranked_lists: list[dict], k: int = 60) -> list[dict]:
    """Reciprocal Rank Fusion: combine multiple ranked result sets."""
    scores: dict[int, float] = {}
    for rank_list in ranked_lists:
        for rank, item in enumerate(rank_list):
            node_id = item.get("id")
            if node_id is not None:
                scores[node_id] = scores.get(node_id, 0.0) + 1.0 / (k + rank + 1)

    sorted_ids = sorted(scores, key=scores.get, reverse=True)
    return [{"id": id, "score": scores[id]} for id in sorted_ids]
```

### Implementación en Go (SDK oficial)

```go
func HybridSearch(query, queryVector string, k int64) helix.Request {
    q := helix.ReadQuery("hybrid_search")
    textParam := q.ParamString("query", query)
    kParam := q.ParamI64("k", k)
    // Note: vector must be passed as []float32 via ParamArray

    return q.
        VarAs("vector_hits",
            helix.G().
                VectorSearchNodesWith(
                    "CodeNode", "embedding",
                    // vector param: ParamArray
                    helix.PropertyInputParam("query_vector"),
                    kParam.Bound(), nil,
                ).
                Project(
                    helix.ProjectPropAs("$id", "id"),
                    helix.ProjectPropAs("$distance", "vector_score"),
                ),
        ).
        VarAs("text_hits",
            helix.G().
                TextSearchNodesWith(
                    "CodeNode", "content",
                    helix.PropertyInputParam("query"),
                    kParam.Bound(), nil,
                ).
                Project(
                    helix.ProjectPropAs("$id", "id"),
                    helix.ProjectPropAs("$distance", "text_score"),
                ),
        ).
        Returning("vector_hits", "text_hits")
}
```

### Configuración de Índices

```go
func CreateHybridIndexes() helix.Request {
    return helix.WriteQuery("create_hybrid_indexes").
        VarAs("vec_idx",
            helix.G().CreateIndexIfNotExists(
                helix.NodeVectorIndex("CodeNode", "embedding"),
            ),
        ).
        VarAs("txt_idx",
            helix.G().CreateIndexIfNotExists(
                helix.NodeTextIndex("CodeNode", "content"),
            ),
        ).
        VarAs("eq_idx",
            helix.G().CreateIndexIfNotExists(
                helix.NodeUniqueEqualityIndex("CodeNode", "path"),
            ),
        ).
        Returning()
}
```

### Tipos de Índice Soportados

| Función Go | Tipo de Índice |
|------------|---------------|
| `helix.NodeEqualityIndex(label, prop)` | Secondary equality |
| `helix.NodeUniqueEqualityIndex(label, prop)` | Equality + unique |
| `helix.NodeRangeIndex(label, prop)` | Range / ordering |
| `helix.NodeVectorIndex(label, prop, tenant?)` | ANN vector |
| `helix.NodeTextIndex(label, prop, tenant?)` | BM25 text |
| `helix.EdgeEqualityIndex(label, prop)` | Edge equality |
| `helix.EdgeVectorIndex(label, prop, tenant?)` | Edge vector |
| `helix.EdgeTextIndex(label, prop, tenant?)` | Edge text |

---

## Esquema de Grafo Recomendado para Memoria Causal

Basado en el skill `helix-memory-system` de HelixDB y adaptado al modelo de ZyroAgentCLI.

### Labels Canónicos

```
Developer (pool global, SIN tenant_id)
  properties: name, default_tech_stack, culture, playbook_ref

Project (scope por developer)
  properties: project_id, name, description, status, current_phase, repo_path, tenant_id

CodeNode (summary de módulo)
  properties: path, summary, hash, language, tenant_id
  indexes: [NodeText("CodeNode", "summary"), NodeEquality("CodeNode", "path")]

Skill (pool global — COMPARTIDO)
  properties: name, type, source_url, validated_at, version
  indexes: [NodeText("Skill", "name")]

Task (trabajo actual/pasado)
  properties: task_id, name, description, phase, status, created_at, tenant_id

Decision (ADR)
  properties: title, context, decision, status, date, tenant_id

Doc (documentación)
  properties: name, path, doc_type, content, hash, tenant_id

Pattern (patrón de diseño)
  properties: name, description, language, confidence, tenant_id

Library (librería validada)
  properties: name, version, category, description, tenant_id

Memory (memoria del agente — para memoria causal)
  properties: memory_id, tenant_id, content, embedding, kind, salience,
             isLatest, validFrom, validTo, expiresAt, deletedAt, lastAccessedAt
  indexes: [NodeVector("Memory", "embedding", "tenant_id"),
            NodeText("Memory", "content", "tenant_id")]
```

### Edges Canónicos

```
Developer ──HAS_PROJECT──→ Project
Project    ──HAS_CODENODE→ CodeNode
Project    ──USES_LIB────→ Library
Project    ──HAS_PATTERN─→ Pattern
Project    ──HAS_DOC─────→ Doc
Project    ──HAS_TASK────→ Task
Project    ──REQUIRES_SKILL→ Skill
Task       ──REFERENCES──→ CodeNode
Task       ──REQUIRES────→ Skill
Task       ──DEPENDS_ON──→ Task
Memory     ──MENTIONS────→ CodeNode / Skill / Pattern
Memory     ──UPDATES─────→ Memory (versión anterior)
Memory     ──EXTENDS─────→ Memory (enriquecimiento)
Memory     ──DERIVES─────→ Memory (inferencia)
Decision   ──REFERENCES──→ CodeNode / Doc / Pattern
```

### Ejemplo: Query de Memoria Causal

```go
// Recuperar memorias relevantes para un contexto de agente
func CausalMemoryRecall(tenantID string, queryVector []float32, queryText string, k int64) helix.Request {
    q := helix.ReadQuery("causal_recall")

    tenant := q.ParamString("tenant_id", tenantID)
    vectorParam := q.ParamArray("query_vector", queryVector, helix.ParamTypeF32())
    textParam := q.ParamString("query", queryText)
    kParam := q.ParamI64("k", k)

    return q.
        VarAs("vector_memories",
            helix.G().
                VectorSearchNodes("Memory", "embedding", vectorParam, kParam, tenant).
                Where(helix.PredEq("isLatest", true)).
                Where(helix.PredIsNull("deletedAt")).
                Where(helix.PredIsNull("expiresAt")). // or expiresAt > now
                Project(
                    helix.ProjectPropAs("$id", "memory_id"),
                    helix.ProjectPropAs("$distance", "score"),
                    helix.ProjectProp("content"),
                    helix.ProjectProp("kind"),
                    helix.ProjectProp("salience"),
                    helix.ProjectProp("lastAccessedAt"),
                ),
        ).
        VarAs("text_memories",
            helix.G().
                TextSearchNodes("Memory", "content", textParam, kParam, tenant).
                Where(helix.PredEq("isLatest", true)).
                Where(helix.PredIsNull("deletedAt")).
                Project(
                    helix.ProjectPropAs("$id", "memory_id"),
                    helix.ProjectPropAs("$distance", "score"),
                    helix.ProjectProp("content"),
                    helix.ProjectProp("kind"),
                ),
        ).
        Returning("vector_memories", "text_memories")
}
```

---

## Embeddings Automáticos: Cómo Funciona

### Hallazgo Crítico: No Hay `Embed` en HelixDB v3

La documentación de HelixDB deja claro:

> Helix does **not** embed text on the dynamic-query path; there is no `Embed()`/`SearchV` in the current DSL.
> — `helix-memory-system` SKILL.md

El comando `Embed()` existía en la antigua sintaxis HQL (`.hx`), pero está **deprecado en v3**. No hay un pipeline de embedding automático en HelixDB.

### Pipeline de Embeddings (App-Side)

```
Texto del agente / código / decisión
    │
    ├──→ Application Worker (Go/Python)
    │       Llama a API de embeddings (OpenAI, Ollama, etc.)
    │       Ejemplo: openai/text-embedding-3-small → 1536-dim F32 array
    │
    ├──→ Pasa float32[] a la query HelixDB
    │       Como parámetro: q.ParamArray("embedding", vec, helix.ParamTypeF32())
    │       O literal: []float32{0.12, 0.85, -0.04, ...}
    │
    └──→ Se almacena en propiedad del nodo
            Al crear: helix.Prop("embedding", embeddingArray)
            Para search: VectorSearchNodes("Label", "embedding", queryVec, k, tenant)
```

### Embeddings Recomendados para ZyroAgentCLI

| Tipo de Contenido | Modelo Recomendado | Dimensiones | Propiedad |
|-------------------|-------------------|-------------|-----------|
| Code summaries | `openai/text-embedding-3-small` | 1536 | `CodeNode.embedding` |
| Decision context | `openai/text-embedding-3-small` | 1536 | `Decision.embedding` |
| Task descriptions | `openai/text-embedding-3-small` | 1536 | `Task.embedding` |
| Doc content (chunks) | `openai/text-embedding-3-small` | 1536 | `Doc.embedding` |
| Agent memories | `openai/text-embedding-3-small` | 1536 | `Memory.embedding` |

### Cálculo de Embeddings en Go

```go
// Ejemplo conceptual — la API real depende del provider
func computeEmbedding(text string) ([]float32, error) {
    // OpenAI API call or local Ollama instance
    resp, err := openai.CreateEmbedding(ctx, openai.EmbeddingRequest{
        Model: "text-embedding-3-small",
        Input: []string{text},
    })
    if err != nil {
        return nil, err
    }
    return resp.Data[0].Embedding, nil // []float32, 1536-dim
}

// Luego en la query:
embedding, _ := computeEmbedding("User authentication with JWT")
q := helix.WriteQuery("create_codenode")
embeddingParam := q.ParamArray("embedding", embedding, helix.ParamTypeF32())

return q.
    VarAs("node",
        helix.G().AddN("CodeNode", helix.Props{
            helix.Prop("path", "internal/auth/jwt.go"),
            helix.Prop("summary", "User authentication with JWT"),
            helix.Prop("embedding", embeddingParam),
            helix.Prop("language", "go"),
            helix.Prop("tenant_id", projectID),
        }),
    ).
    Returning("node")
```

---

## Recomendaciones Concretas para Mejorar la Integración

### Prioridad Alta (Impacto Inmediato)

#### 1. Migrar a Go SDK Oficial

**Qué**: Reemplazar el cliente raw JSON en `internal/db/helix/` por el SDK oficial `github.com/helixdb/helix-db/sdks/go`.

**Por qué**: Tipado fuerte, parámetros dinámicos, ejecución con `client.Exec`, manejo de conflictos, y acceso a todo el DSL de traversals.

**Cómo**:
```go
// ANTES: raw map[string]any → raw json.Marshal → POST
payload := buildV3Envelope([]v3Query{{Name: "n", Steps: steps}}, "write")
result, _ := c.doQuery(ctx, payload)
return parseSingleNode(result, "n", label)

// DESPUÉS: SDK oficial con tipos
q := helix.WriteQuery("create_node")
return q.VarAs("n", helix.G().AddN("Skill", helix.Props{
    helix.Prop("name", name),
})).Returning("n")

var out CreateNodeResponse
client.Exec(ctx, q, &out)
```

#### 2. Implementar Búsqueda Híbrida en MCP Tools

**Qué**: Agregar vector search y fusión RRF a `search_code`, `search_skills`, y `task_context`.

**Por qué**: Las MCP tools actuales solo usan BM25. El agente pierde resultados semánticamente similares que no comparten tokens exactos.

**Cómo**: Agregar columna `embedding` (F32 array) a CodeNode y Document. Calcular embeddings al hacer `absorb`. En las MCP tools, ejecutar vector + BM25 en paralelo y fusionar con RRF.

#### 3. Pipeline de Embeddings Automáticos

**Qué**: Agregar worker de embeddings en Go que calcule `text-embedding-3-small` para cada CodeNode, Doc, Decision, y Task al crearse.

**Por qué**: Sin embeddings, la búsqueda vectorial y la búsqueda híbrida no funcionan. Es el prerrequisito para #2.

**Cómo**:
- En `internal/embed/` — worker que llama a API de embeddings
- Hook en `internal/codeparse/` para generar embedding del summary
- Hook en `zyrocli absorb` para embedear docs
- Almacenar como `[]float32` en propiedad `embedding`

### Prioridad Media (Fase 4)

#### 4. MCP Server en Go

**Qué**: Migrar de Python FastMCP a un MCP server Go nativo usando `github.com/mark3labs/mcp-go`.

**Por qué**: El MCP server Go puede reutilizar directamente el Go SDK oficial, compartir tipos con `internal/db/helix/`, y ejecutarse como subcomando de `zyrocli` sin dependencia de Python/uv.

**Cómo**:
```
cmd/zyrocli/mcp.go        → subcomando "zyrocli mcp serve"
internal/mcp/server.go    → MCP server + tool handlers
internal/mcp/tools.go     → Definiciones de tools
internal/mcp/fusion.go    → RRF y lógica híbrida
```

#### 5. Agregar Traversals Complejos

**Qué**: Implementar en el Go SDK (o en queries ad-hoc) `Repeat`, `Union`, `Choose`, `Coalesce`, `Optional`.

**Por qué**: Permiten navegación semántica del grafo: "encuentra todos los proyectos que usan un skill → qué otros skills usan esos proyectos".

**Ejemplos de queries útiles**:
```go
// Cross-project skill discovery
helix.G().
    NWithLabel("Skill").
    Where(helix.PredEq("name", "typescript")).
    In("REQUIRES_SKILL").  // Projects that require TS
    Out("USES_LIB").       // Libraries those projects use
    Dedup().
    ValueMap("$id", "name", "version")
```

#### 6. Índices Vectoriales y de Texto con CreateIndexIfNotExists

**Qué**: Agregar `CreateIndexIfNotExists` al flujo `db init` y manejar tenant-partitioned indexes.

**Por qué**: Los índices vectoriales y BM25 son necesarios para búsqueda. `if_not_exists` es idempotente y seguro en cada deploy.

#### 7. Operaciones Condicionales (varAsIf)

**Qué**: Usar `varAsIf` con `BatchCondition` para upserts y queries condicionales.

**Por qué**: Evita writes duplicados, permite "crear si no existe" en una sola transacción, y reduce latencia vs read-then-write en dos viajes.

### Prioridad Baja (Mejora Continua)

#### 8. Memoria Causal del Agente

**Qué**: Implementar el modelo de memoria del `helix-memory-system` skill con labels `Memory`, `Session`, lifecycle flags (`isLatest`, `deletedAt`, `expiresAt`), y edges semánticos (`MENTIONS`, `UPDATES`, `EXTENDS`, `DERIVES`).

**Por qué**: Permite que el agente recuerde decisiones pasadas, evite contradecirse, y mantenga contexto persistente entre sesiones.

#### 9. Multi-Tenencia Nativa

**Qué**: Usar `tenant_id` como propiedad de partición en índices vectoriales y de texto (el tercer argumento de `NodeVectorIndex`/`NodeTextIndex`).

**Por qué**: Aísla datos entre proyectos/developers a nivel de índice, no solo a nivel de query filter. Mejor performance y seguridad.

#### 10. Observabilidad de Queries

**Qué**: Agregar tracing (OpenTelemetry) a las queries HelixDB, midiendo latencia, resultados retornados, y errores.

**Por qué**: Sin métricas es imposible optimizar. El Go SDK oficial no tiene built-in tracing pero es fácil de agregar wrappers.

---

## Referencias

- **HelixDB GitHub**: https://github.com/HelixDB/helix-db
- **Documentación HelixDB**: https://docs.helix-db.com
- **LLMs.txt (índice completo)**: https://docs.helix-db.com/llms.txt
- **Querying Guide Overview**: https://docs.helix-db.com/database/querying-guide/overview
- **Go SDK Setup**: https://docs.helix-db.com/database/go-project-setup
- **Vector and Text Search**: https://docs.helix-db.com/database/querying-guide/search
- **Data Model**: https://docs.helix-db.com/database/data-model
- **Vector Indexes**: https://docs.helix-db.com/database/indexing/vector
- **Text Indexes**: https://docs.helix-db.com/database/indexing/text
- **Helix Skills Repo**: https://github.com/HelixDB/skills
- **helix-memory-system skill**: https://github.com/HelixDB/skills/tree/main/skills/helix-memory-system
- **helix-query-go skill**: https://github.com/HelixDB/skills/tree/main/skills/helix-query-go
- **Código actual**: `internal/db/helix/` y `mcp-tools/helix_client.py`
- **Arquitectura v2**: `docs/architecture-v2.md`
- **Evaluación MCP previa**: `docs/helixdb-mcp-evaluation.md`

---

## Apéndice: Mapa de Migración

| Archivo Actual | Acción | Archivo Destino |
|---------------|--------|-----------------|
| `internal/db/helix/helix.go` | Migrar a SDK oficial | Usar `github.com/helixdb/helix-db/sdks/go` |
| `internal/db/helix/types.go` | Reemplazar por tipos del SDK | Eliminar o mantener como wrappers delgados |
| `internal/db/helix/errors.go` | Reemplazar por `helix.ErrConflict` etc | Eliminar |
| `mcp-tools/helix_client.py` | Mantener para compat | Migrar a Go MCP server en Fase 4 |
| `mcp-tools/runner.py` | Reemplazar por `zyrocli mcp serve` | `cmd/zyrocli/mcp.go` |
| `mcp-tools/search_code.py` | Migrar a Go + búsqueda híbrida | `internal/mcp/tools/search_code.go` |
| `mcp-tools/search_skills.py` | Migrar a Go + búsqueda híbrida | `internal/mcp/tools/search_skills.go` |
| `mcp-tools/task_context.py` | Migrar a Go + traversals complejos | `internal/mcp/tools/task_context.go` |
| `mcp-tools/helix_write.py` | Migrar a Go + upsert | `internal/mcp/tools/helix_write.go` |
| `internal/opencode/mcptools_embed.go` | Mantener o deprecar según migración | — |
