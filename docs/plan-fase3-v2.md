# Plan de Ejecución: Fase 3 — Multi-Project (Un Solo Developer)

> Junio 2026 · Re-planificación con interpretación CORREGIDA

---

## 1. Corrección de Interpretación

### ❌ Versión incorrecta (plan anterior)
HelixDB era "multi-tenant" — múltiples developers aislados entre sí, con `visibility: public/private` en Skills, `isolation_level`, y autenticación entre tenants.

### ✅ Versión correcta
HelixDB es **una base de conocimiento de UN solo developer** con **muchos proyectos**. No hay "Developer A vs Developer B". Hay "secko tiene ZyroAgentCLI y OtroProyecto".

```
HelixDB
└── secko (único usuario, NO es un nodo tenant)
    ├── Proyecto: ZyroAgentCLI
    │   ├── Docs, Patterns, CodeNodes, Tasks (aislados por project_id)
    │   └── Skills que usa (REQUIRES_SKILL edge)
    ├── Proyecto: OtroProyecto
    │   ├── Docs, Patterns, CodeNodes, Tasks (aislados por project_id)
    │   └── Skills que usa (REQUIRES_SKILL edge)
    └── Skills compartidos entre proyectos
        └── Un solo nodo Skill, referenciado por múltiples Projects
```

**Implicación práctica**: `tenant_id` en el código actual **significa project_id**, no developer. El "developer" no es un concepto de aislamiento — es el dueño de toda la base.

---

## 2. Estado Actual (Fase 2 Completada)

### Archivos en `internal/db/helix/`

| Archivo | Líneas | Qué hace | Problema con la interpretación correcta |
|---------|--------|----------|----------------------------------------|
| `client.go` | 133 | `Client` wrapper con `tenantID string`, `InjectTenant()`, `Ping()` | `tenantID` debería ser `projectID`. El concepto "tenant" no aplica — hay un solo usuario. |
| `nodes.go` | 195 | CRUD: `CreateNode` (inyecta tenant), `GetNode` (verifica tenant), `FindNodes` (filtra por tenant) | `GetNode` verifica `ErrTenantMismatch` — esto desaparece. `FindNodes` filtra por `tenant_id` — debe filtrar por `project_id`. |
| `search.go` | 120 | `VectorSearch` y `TextSearch` particionados por `tenant_id` | Los índices vectoriales usan `tenant_id` como partition key — debe ser `project_id`. |
| `edges.go` | 126 | `CreateEdge`, `GetOutgoing`, `GetIncoming`, `DeleteEdge` | Sin cambios necesarios — los edges no tienen tenant. |
| `schema.go` | 86 | 11 índices (equality, vector, text, range) | Índices vectoriales particionados por `tenant_id` — cambiar a `project_id`. Índices de Skill quitan partición (cross-project). |
| `errors.go` | 11 | `ErrTenantMismatch` y otros | `ErrTenantMismatch` se elimina — no aplica. |

### Modelo actual de datos (erróneo)

```
Developer (name:"sebastian")
  └── HAS_PROJECT → Project (name:"zyro", tenant_id:"sebastian")
        ├── HAS_DOC → Document (tenant_id:"sebastian")
        ├── HAS_PATTERN → Pattern (tenant_id:"sebastian")
        ├── REQUIRES_SKILL → Skill (tenant_id:"sebastian")  ← DUPLICADO por proyecto
        ├── HAS_CODENODE → CodeNode (no existe aún)
        └── HAS_TASK → Task (no existe aún)
```

**Problema**: Cada proyecto crea su propio nodo Skill "TypeScript" con `tenant_id` distinto. Si hay 3 proyectos, hay 3 nodos TypeScript duplicados.

---

## 3. Los 5 Componentes de la Fase 3

### 3.1 Refactor: `tenant_id` → `project_id`

**Qué es**: Renombrar y reorientar el mecanismo de aislamiento. Hoy `tenant_id` significa "developer". Debe significar "project". El developer NO es un concepto de aislamiento en la DB.

**Qué cambia en el código**:

```go
// ANTES (client.go)
type Client struct {
    inner    *helixsdk.Client
    tenantID string  // "sebastian" — el developer
}

// DESPUÉS (client.go)
type Client struct {
    inner     *helixsdk.Client
    projectID string  // UUID del proyecto actual, e.g. "proj-zyroagentcli-001"
}
```

**Cambios específicos**:

| Archivo | Cambio |
|---------|--------|
| `client.go` | `tenantID` → `projectID`. `WithTenantID` → `WithProjectID`. `InjectTenant` → `InjectProject`. `TenantID()` → `ProjectID()`. |
| `nodes.go` | `CreateNode`: `InjectTenant` → `InjectProject`. `GetNode`: eliminar verificación `ErrTenantMismatch` (ya no aplica — el dueño es uno). `FindNodes`: `SourceEq("tenant_id", ...)` → `SourceEq("project_id", ...)`. |
| `search.go` | `VectorSearch`: partition key `tenant_id` → `project_id`. `TextSearch`: idem. |
| `schema.go` | Índices vectoriales: `"tenant_id"` → `"project_id"` como partition key. Índice `idx_proj_tenant` → `idx_proj_project_id` (o eliminar si Project no necesita filtrarse por project_id — él ES el project). |
| `errors.go` | Eliminar `ErrTenantMismatch`. Renombrar `ErrConnection` si se desea (opcional). |

**Lo que NO se toca**: `edges.go` (los edges no tienen project_id — se resuelven por traversal desde el Project node).

**Esfuerzo**: 🟢 Bajo — Es un rename + eliminación de una verificación. ~30 líneas cambiadas.

---

### 3.2 Cross-Project Skill Sharing

**Qué es**: Skills se comparten entre proyectos del mismo developer. Un nodo Skill "TypeScript" existe UNA sola vez y es referenciado por múltiples Projects via `REQUIRES_SKILL`.

**Modelo actual (erróneo)**:
```
Skill "TypeScript" (tenant_id: "zyro")     ← duplicado
Skill "TypeScript" (tenant_id: "otro-proyecto")  ← duplicado
```

**Modelo correcto**:
```
Skill "TypeScript"
  properties:
    name: "TypeScript"
    type: "FE"
    source_url: "https://typescriptlang.org"
    version: "5.4"
    validated_at: "2026-06-14"
    embedding: [0.12, ...]
    // SIN project_id — es global para todo developer

  ← REQUIRES_SKILL ← Project "ZyroAgentCLI" (required_level: "advanced")
  ← REQUIRES_SKILL ← Project "OtroProyecto" (required_level: "intermediate")
```

**Cambios en el código**:

| Archivo | Cambio |
|---------|--------|
| `schema.go` | Índice `idx_skill_emb`: quitar partición por `tenant_id`. Skills son globales. El `NodeUniqueEqualityIndex("Skill", "name")` se mantiene — un skill por nombre. |
| `nodes.go` | `FindNodes` para Skills: NO agregar filtro `project_id`. Skills son del developer, no del proyecto. Necesita un método `FindSharedSkills` o modificar `FindNodes` para aceptar un flag `global bool`. |
| `search.go` | `VectorSearch` para Skills: buscar SIN partición de project_id. Necesita una variante `VectorSearchGlobal`. |

**Propiedad del edge `REQUIRES_SKILL`**:
```
REQUIRES_SKILL
  properties:
    required_level: "advanced" | "intermediate" | "basic"
```
Cada proyecto puede requerir el mismo skill con nivel distinto.

**Esfuerzo**: 🟡 Medio — El modelo de edges ya soporta esto. El cambio principal es en la capa de queries y índices.

---

### 3.3 CodeNode Summaries

**Qué son**: Nodos que representan **resúmenes de módulos de código** — NO código completo. Cada CodeNode pertenece a UN Project.

**Propiedades del CodeNode**:
```
CodeNode
  properties:
    project_id: "proj-zyroagentcli-001"   // PERTENECE a un proyecto
    name: "client.go"
    path: "internal/db/helix/client.go"
    module_type: "file"     // file | package | function | type
    language: "go"
    summary: "Wrapper sobre HelixDB SDK con inyección automática de project_id."
    dependencies: ["helixdb/helix-db/sdks/go"]
    embedding: [0.45, ...]
    hash: "abc123..."       // hash del contenido para detectar cambios
```

**Pipeline de generación**:
```
zyrocli absorb --code [path]
  │
  ├── Walk del directorio
  │   ├── Detectar archivos por extensión (.go, .ts, .py)
  │   ├── Parsear AST (Go: go/ast) → extraer funciones, types, imports
  │   └── Para otros lenguajes: regex + filename patterns
  │
  ├── Generar summary textual
  │   ├── Opción A: Template + AST (más rápido, menos inteligente)
  │   ├── Opción B: LLM (más lento, mejor quality)
  │   └── Opción C: Híbrido (AST para estructura, LLM para "qué hace")
  │
  ├── Generar embedding del summary
  │
  ├── Crear/upsert nodo CodeNode en HelixDB
  │   └── upsert por (project_id + path) — no duplicar
  │
  └── Crear edge HAS_CODENODE desde el Project
```

**Actualización**: Cada CodeNode tiene un `hash`. `zyrocli absorb --code` compara hashes — si cambió, regenera summary + embedding.

**Archivos nuevos**:

| Archivo | Qué hace |
|---------|----------|
| `internal/codeparse/go_ast.go` | Parseo de `go/ast` — extraer funciones, types, imports |
| `internal/codeparse/detector.go` | Detector de lenguaje por extensión |
| `internal/codeparse/summary.go` | Generación de summary textual + embedding |
| `internal/db/helix/nodes.go` | Nuevo método `UpsertCodeNode(ctx, projectID, props)` — upsert por (project_id, path) |

**Esfuerzo**: 🔴 Alto — Requiere parseo de AST, generación de summaries, manejo de embeddings. ~150-200 líneas nuevas.

---

### 3.4 Task → CodeNode Graph

**Qué es**: Conexión entre tareas de desarrollo y el código que tocan. Permite trazabilidad: "¿qué tareas tocaron este módulo?" y "¿qué código cambió por esta task?"

**Modelo**:
```
Task
  properties:
    project_id: "proj-zyroagentcli-001"
    description: "Implementar context injection"
    phase: "phase3"
    status: "done"
    created_at: "2026-06-14"

  ↓ REFERENCES → CodeNode ("internal/context/helix_query.go")
  ↓ REQUIRES → Skill ("Go")
```

**Construcción automática (git diff)**:
```
zyrocli task link [task-id]
  │
  ├── git diff --name-only HEAD~1
  │   → ["internal/context/helix_query.go", "cmd/zyrocli/context.go"]
  │
  ├── Para cada archivo modificado:
  │   ├── Buscar CodeNode existente por (project_id, path)
  │   ├── Si NO existe → crearlo (absorb automático de ese archivo)
  │   └── Crear edge REFERENCES (Task → CodeNode)
  │
  └── Output: "Task #12 ahora referencia 2 CodeNodes"
```

**Archivos nuevos**:

| Archivo | Qué hace |
|---------|----------|
| `cmd/zyrocli/task.go` | Comandos `task create`, `task link`, `task list` |
| `internal/git/diff.go` | Wrapper sobre `git diff` para detectar archivos modificados |
| `internal/db/helix/nodes.go` | Método `LinkTaskToCodeNodes(ctx, taskID, paths []string)` |

**Esfuerzo**: 🟡 Medio — La infraestructura base ya existe. ~100-150 líneas nuevas.

---

### 3.5 `zyrocli context [task]`

**Qué hace**: Dado un task ID, recupera de HelixDB: Skills relacionados, CodeNodes tocados, Docs relevantes, Patterns usados. Output: texto plano (para inyectar en prompt de subagente) o JSON.

**Formato de output**:

```
$ zyrocli context task-42

Context for Task #42: "Implementar auth JWT"

Skills (2):
  • TypeScript (FE, advanced)
  • JWT (BE, intermediate)

CodeNodes (3):
  • internal/auth/middleware.go — "JWT middleware con refresh token logic"
  • internal/auth/jwt.go — "Utilidades de signing y verification"
  • internal/db/helix/client.go — "Wrapper HelixDB con project injection"

Documents (1):
  • spec-auth.md — "Especificación del módulo de autenticación"

Patterns (1):
  • Repository Pattern — "Separación de acceso a datos"
```

**Flujo**:
```
zyrocli context task-42 --format=prompt
  │
  1. Resolver task_id → task-42
  │
  2. Consultar HelixDB:
  │  Task(task-42).REQUIRES → [Skill nodes]
  │  Task(task-42).REFERENCES → [CodeNode nodes]
  │  Project(task-42.project_id).HAS_DOC → [Document nodes]
  │  Project(task-42.project_id).HAS_PATTERN → [Pattern nodes]
  │
  3. Formatear output (--format=text|json|prompt)
  │
  4. Return to caller
```

**Conexión con OpenCode**:
```
Orquestador → zyrocli context task-42 --format=prompt → HelixDB → resultado
  → inyectar en prompt del subagente
  → subagente trabaja con contexto preciso
```

**Archivos nuevos**:

| Archivo | Qué hace |
|---------|----------|
| `cmd/zyrocli/context.go` | Comando `context` con flags `--format` |
| `internal/context/helix_query.go` | Queries a HelixDB para contexto de task |
| `internal/context/formatter.go` | Formateadores (text, json, prompt) |

**Esfuerzo**: 🟡 Medio — Las queries están en `helixdb-schema-hql.md` (líneas 389-446). ~150-200 líneas nuevas.

---

## 4. Lo que NO Cambia

| Capa | Se mantiene igual | Por qué |
|------|-------------------|---------|
| `edges.go` | Sin cambios | Los edges no tienen project_id — se resuelven por traversal |
| Context bridge (`bridge.go`) | Sin cambios | Sigue siendo MCP client para Context7. HelixDB es adicional |
| Scheduler F1 | Sin cambios | Parseo de handoff + approval gates intactos |
| Handoff parser | Sin cambios | Sigue parseando `handoff.yaml` |
| Scaffold | Sin cambios | Estructura de carpetas no cambia |
| os/exec para scripts Python | Sin cambios | No hay razón para cambiar |
| `zyrocli db init` | Función no cambia | Solo se agregan/modifican índices en `schema.go` |
| `zyrocli absorb` (documentos) | Función no cambia | Solo se le agrega flag `--code` |
| `errors.go` | Solo se elimina `ErrTenantMismatch` | Los demás errores se mantienen |

---

## 5. Diagrama: Conexión entre Componentes

```
┌─────────────────────────────────────────────────────────────────────┐
│                  FASE 3 — Multi-Project (Un Developer)              │
└─────────────────────────────────────────────────────────────────────┘

                    ┌──────────────────────┐
                    │     HelixDB          │
                    │  (UNA base, un dev)  │
                    └──────────┬───────────┘
                               │
              ┌────────────────┼────────────────┐
              │                                  │
              ▼                                  ▼
    ┌──────────────────┐              ┌──────────────────┐
    │ Project          │              │ Skill (shared)   │
    │ "ZyroAgentCLI"   │              │ "TypeScript"     │
    │ project_id: UUID │              │ SIN project_id   │
    └────────┬─────────┘              │ (es global)      │
             │                        └────────┬─────────┘
             │                                 │
    ┌────────┼────────┬──────────┐     ┌───────┴───────┐
    │        │        │          │     │               │
    ▼        ▼        ▼          ▼     ▼               ▼
┌───────┐ ┌───────┐ ┌───────┐ ┌──────────┐    ┌──────────┐
│  Doc  │ │Pattern│ │CodeNode│ │  Task    │    │  Task    │
│(spec) │ │(arch) │ │(summ) │ │(current)│    │ (past)   │
└───────┘ └───────┘ └───┬───┘ └────┬─────┘    └────┬─────┘
                        │          │                │
                        │          └────────────────┘
                        │              REFERENCES
                        ▼
                   ┌──────────┐
                   │ CodeNode │  ← "¿qué tareas tocaron este módulo?"
                   └──────────┘
```

### Flujo de `zyrocli context`:

```
$ zyrocli context task-42 --format=prompt

    Task #42 (project: ZyroAgentCLI)
        │
        ├── REQUIRES ──→ Skill "Go" (shared)
        │                 Skill "HelixDB" (shared)
        │
        ├── REFERENCES ─→ CodeNode "internal/db/helix/client.go"
        │                 CodeNode "internal/context/helix_query.go"
        │
        └── (via Project) ─→ Document "spec-context.md"
                              Pattern "Repository Pattern"
                              Pattern "Context Bridge"

    → Output formateado (text/json/prompt)
```

---

## 6. Orden de Implementación

```
Paso 1:  Refactor tenant_id → project_id
         └── Base para todo lo demás. Sin esto, nada funciona correctamente.
         └── ~30 líneas cambiadas, 0 líneas nuevas.
         └── Validación: todos los tests existentes pasan (si los hay).

Paso 2:  Cross-project Skill sharing
         └── Dependencia: Paso 1 (necesita project_id en su lugar).
         └── Cambios en schema.go (quitar partición de índices Skill).
         └── Método FindSharedSkills en nodes.go.
         └── ~50 líneas nuevas/modificadas.

Paso 3:  CodeNode summaries
         └── Dependencia: Paso 1 (project_id en CodeNodes).
         └── Nuevo paquete internal/codeparse/.
         └── Método UpsertCodeNode en nodes.go.
         └── ~150-200 líneas nuevas.

Paso 4:  Task → CodeNode graph
         └── Dependencia: Paso 3 (necesita CodeNodes existentes).
         └── Nuevo cmd/zyrocli/task.go.
         └── Nuevo internal/git/diff.go.
         └── ~100-150 líneas nuevas.

Paso 5:  zyrocli context [task]
         └── Dependencia: Pasos 3 + 4 (necesita CodeNodes y Tasks con edges).
         └── Nuevo cmd/zyrocli/context.go.
         └── Nuevo internal/context/helix_query.go + formatter.go.
         └── ~150-200 líneas nuevas.
```

**Total estimado**: ~400-600 líneas de código nuevo (Go).

---

## 7. Esfuerzo Estimado por Componente

| # | Componente | Esfuerzo | Dependencias | Líneas estimadas |
|---|-----------|----------|--------------|------------------|
| 1 | Refactor `tenant_id` → `project_id` | 🟢 Bajo | — | ~30 (cambios) |
| 2 | Cross-project Skill sharing | 🟡 Medio | #1 | ~50 |
| 3 | CodeNode summaries | 🔴 Alto | #1 | ~150-200 |
| 4 | Task → CodeNode graph | 🟡 Medio | #3 | ~100-150 |
| 5 | `zyrocli context [task]` | 🟡 Medio | #3, #4 | ~150-200 |

---

## 8. Riesgos

| Riesgo | Impacto | Probabilidad | Mitigación |
|--------|---------|-------------|------------|
| **Rename incompleto** — queda algún `tenant_id` suelto en queries o índices | 🟡 Medio | 🟡 Media | Búsqueda global `grep "tenant_id"` post-refactor. Tests de integración. |
| **Skill sharing — query sin filtro devuelve Skills de otros "developers"** | N/A | N/A | **No aplica** — hay un solo developer. No hay "otros". |
| **CodeNode summary quality** — summaries malos confunden al agente | 🟡 Medio | 🟡 Media | Empezar con template + AST (no LLM). Expandir a LLM después. |
| **Embedding dimension mismatch** — mezclar modelos genera vectores no comparables | 🟡 Medio | 🟢 Baja | Estandarizar un modelo (1536-dim). Documentar en README. |
| **HelixDB schema drift** — agregar propiedades sin migración | 🟢 Baja | 🟢 Baja | Schemaless = no hay migración. Validar en capa Go. |
| **Complejidad del pipeline CodeNodes** — AST + summary + embedding = mucho código | 🟡 Medio | 🟡 Media | Empezar con Go solamente. Expander lenguajes después. |
| **git diff no captura renames** — `task link` pierde archivos renombrados | 🟢 Baja | 🟢 Baja | Usar `git diff --name-status` para detectar renames. |

---

## 9. Resumen para el Usuario

**¿Qué es la Fase 3?** Es agregar **multi-project** a una base de conocimiento de un solo developer. No es "multi-tenant" — es "multi-proyecto".

**¿Qué gana tu proyecto?**
1. **Project isolation real** — Cada proyecto tiene sus Docs, CodeNodes, Tasks aislados por `project_id`.
2. **Skills compartidos** — Un nodo "TypeScript" se crea una vez y lo usan todos los proyectos. No duplicación.
3. **CodeNodes** — El agente "sabe" qué hace cada módulo sin leer todo el código.
4. **Trazabilidad** — Sabés exactamente qué código tocó cada task (via git diff automático).
5. **Context injection** — `zyrocli context` le da al subagente solo la información que necesita.

**¿Qué NO hay que tocar?** Todo lo que ya funciona (bridge, scheduler, scaffold, absorb de docs) se mantiene igual. Solo se agregan capas encima y se renombra `tenant_id` → `project_id`.
