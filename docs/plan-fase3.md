# Plan de Ejecución: Fase 3 — Multi-tenant / Open Source Ready

> Junio 2026 · Basado en Arquitectura Decisional v2 + Estado actual Fase 2

---

## 1. ¿Qué es "Multi-tenant"?

Hoy, el sistema tiene **tenancy básico**: cada nodo lleva un `tenant_id` (string plano) que se inyecta automáticamente en el cliente HelixDB. Pero el aislamiento es frágil:

- **No hay autenticación real** — cualquiera que conozca el ID de un nodo puede leerlo
- **No hay permisos** — Developer A puede teóricamente encontrar nodos de Developer B si no filtra correctamente
- **El Skill sharing es ambiguo** — el schema dice "pool compartido del Developer" pero no aclara qué pasa entre Developers

### Ejemplo concreto: 2 developers

**Hoy (Fase 2):**
```
Developer "Sebastian"
  └── Project "ZyroAgentCLI"
        └── REQUIRES_SKILL → Skill "TypeScript" (tenant_id: "sebastian")

Developer "Ana"
  └── Project "DataPiper"
        └── REQUIRES_SKILL → Skill "TypeScript" (tenant_id: "ana")
```

Ana y Sebastian **NO comparten** el nodo TypeScript. Cada uno tiene el suyo duplicado con distinto `tenant_id`. Si Sebastian descubre una nueva versión de TypeScript, Ana no se entera.

**Fase 3 (multi-tenant real):**
```
Developer "Sebastian" (tenant: sebastian)
  └── Project "ZyroAgentCLI"
        └── REQUIRES_SKILL → Skill "TypeScript" (owner: global)
        └── HAS_CODENODE → CodeNode "internal/db/helix/client.go"

Developer "Ana" (tenant: ana)
  └── Project "DataPiper"
        └── REQUIRES_SKILL → Skill "TypeScript" (owner: global)
        └── HAS_CODENODE → CodeNode "src/pipeline.ts"

Skill "TypeScript" (owner: global, type: FE)
  ← REQUIRES_SKILL ← ZyroAgentCLI
  ← REQUIRES_SKILL ← DataPiper
```

Un solo nodo Skill, compartido entre proyectos de diferentes Developers.

---

## 2. Estado Actual del Tenant (Fase 2)

### Cómo funciona hoy

El `Client` de HelixDB (`internal/db/helix/client.go`) inyecta `tenant_id` de forma automática:

```go
// InjectTenant agrega tenant_id si no existe
func (c *Client) InjectTenant(props map[string]interface{}) map[string]interface{} {
    if c.tenantID != "" {
        if _, exists := props["tenant_id"]; !exists {
            props["tenant_id"] = c.tenantID
        }
    }
    return props
}
```

**Puntos clave:**
1. `CreateNode` llama `InjectTenant` automáticamente — todo nodo nuevo recibe `tenant_id`
2. `GetNode` verifica que el nodo pertenezca al tenant del cliente — si no, retorna `ErrTenantMismatch`
3. `FindNodes` agrega filtro `tenant_id` a la query — solo retorna nodos del tenant
4. `VectorSearch` y `TextSearch` pasan `tenant_id` como partition key al índice
5. `UpdateNode` **previene** la sobreescritura de `tenant_id` (lo borra del input)
6. Los índices vectoriales y de texto están particionados por `tenant_id`

**Lo que falta para ser "multi-tenant" real:**

| Capa | Hoy | Fase 3 |
|------|-----|--------|
| Identificación | String plano `"sebastian"` | Developer node como root entity |
| Aislamiento | Filtro en cada query (frágil) | Wrapper que garantiza aislamiento |
| Auth | Ninguno | Developer key o similar |
| Skill sharing | Duplicado por tenant | Un solo nodo global |
| CodeNodes | No existen | Summaries automatizados |
| Context query | No existe | `zyrocli context [task]` |

---

## 3. Los 5 Componentes de la Fase 3

### 3.1 Developer Node como Tenant Aislado

**Qué es:** El nodo `Developer` pasa de ser "solo otro nodo" a ser la **raíz de aislamiento**. Todo nodo en el grafo pertenece a un Developer (directa o indirectamente).

**Cómo funciona hoy:**
- `Developer` existe como label en el schema (`schema.go` línea 30)
- Tiene `NodeUniqueEqualityIndex("Developer", "name")`
- Pero NO tiene `tenant_id` (porque EL Developer ES el tenant)

**Qué cambiaría:**
```go
// Hoy: el Developer no tiene tenant_id
"Developer" → name:"Sebastian", default_tech_stack:"Go,TS", culture:"clean-arch"

// Fase 3: se agrega ID estable y nivel de aislamiento
"Developer" → id:"dev-seb-001", name:"Sebastian", email:"seb@...",
               isolation_level:"strict", // o "shared"
               created_at:"2026-06-14"
```

**Regla de aislamiento:**
- Cada nodo hijo (Project, Task, CodeNode, Document) hereda `tenant_id = Developer.id`
- Los Skills pueden tener `owner = "global"` o `owner = "dev-seb-001"`
- La query layer del cliente **siempre** filtra por `tenant_id` — nunca se desactiva

**Archivos a crear/modificar:**
| Archivo | Acción |
|---------|--------|
| `internal/db/helix/schema.go` | Agregar índices para Developer (ya existe el unique) |
| `internal/db/helix/nodes.go` | Modificar `FindNodes` para que NUNCA devuelva nodos de otros tenants (doble verificación) |
| `cmd/zyrocli/init.go` | Cuando hace `init`, crear nodo Developer si no existe |
| `internal/db/helix/client.go` | El `WithTenantID` ahora recibe `Developer.id` como valor |

**Esfuerzo:** 🟡 Medio — El patrón ya existe, hay que reforzarlo y agregar validación en `FindNodes` para que siempre filtre por tenant.

---

### 3.2 Cross-project Skill Sharing

**Qué es:** Un nodo Skill compartido entre múltiples Projects de diferentes Developers.

**Modelo actual:**
```
Skill "TypeScript" (tenant_id: "sebastian")  ← duplicado
Skill "TypeScript" (tenant_id: "ana")         ← duplicado
```

**Modelo Fase 3:**
```
Skill "TypeScript"
  properties:
    name: "TypeScript"
    type: "FE"
    source_url: "https://typescriptlang.org"
    version: "5.4"
    owner: "global"          // NUEVO: quién creó el skill
    visibility: "public"     // NUEVO: public | private
    validated_at: "2026-06-14"
    embedding: [0.12, ...]
  
  ← REQUIRES_SKILL ← Project "ZyroAgentCLI" (required_level: "advanced")
  ← REQUIRES_SKILL ← Project "DataPiper" (required_level: "intermediate")
```

**¿Cómo se modela?**
- **El Skill es global por defecto** — si `visibility = "public"`, cualquier Developer puede referenciarlo
- **Skills privados** — si `visibility = "private"`, solo el owner puede verlos
- **Edge REQUIRES_SKILL** ahora tiene propiedad `required_level` que varía por proyecto
- El `NodeUniqueEqualityIndex("Skill", "name")` se mantiene — un skill por nombre

**¿Qué pasa con permisos?**
- Developer A **no puede ver** Skills privados de Developer B
- Developer A **puede ver** Skills públicos (globales)
- La query layer agrega filtro: `WHERE visibility = "public" OR owner = :tenant_id`

**Archivos a crear/modificar:**
| Archivo | Acción |
|---------|--------|
| `internal/db/helix/schema.go` | Agregar index para `Skill.visibility` + `Skill.owner` |
| `internal/db/helix/nodes.go` | Nuevo método `FindGlobalSkills(ctx, query)` que busca Skills públicos |
| `internal/db/helix/search.go` | Modificar `VectorSearch` para Skills: buscar en globales + locales |
| `cmd/zyrocli/skill.go` | Nuevo comando `zyrocli skill list [--scope=global\|mine]` |

**Esfuerzo:** 🟡 Medio — El modelo de edges ya soporta esto. El cambio principal es en la capa de queries.

---

### 3.3 CodeNode Summaries Automatizados

**Qué son:** Nodos que representan **resúmenes de módulos de código** — NO código completo. Son la clave para que el agente entienda qué hace cada parte sin leer todo el codebase.

**Propiedades del CodeNode:**
```
CodeNode
  properties:
    name: "client.go"
    path: "internal/db/helix/client.go"
    module_type: "file"     // file | package | function | type
    language: "go"
    summary: "Wrapper sobre HelixDB SDK con inyección automática de tenant_id.
              Provee CreateNode, GetNode, FindNodes, VectorSearch, TextSearch.
              Tenant isolation via row-level filtering."
    dependencies: ["helixdb/helix-db/sdks/go"]  // imports relevantes
    embedding: [0.45, ...]  // para búsqueda semántica
    tenant_id: "sebastian"
    hash: "abc123..."       // hash del contenido para detectar cambios
```

**Cómo se generarían:**

| Método | Pros | Cons | ¿Cuándo usar? |
|--------|------|------|----------------|
| **Parseo de AST** (Go `go/ast`) | Preciso, sabe qué funciones/types exporta | Solo Go, no captura "qué hace" | Para `module_type: "function"` y `"type"` |
| **Lectura de directorio + regex** | Funciona con cualquier lenguaje | Superficial, no entiende semántica | Para `module_type: "file"` y `"package"` |
| **Generado por IA** (embedding + summary) | Entiende "qué hace", no solo "qué tiene" | Lento, costoso en tokens, necesita validación | Para el campo `summary` textual |
| **Híbrido** (AST + IA) | Lo mejor de ambos mundos | Más complejo de implementar | **Recomendado** |

**Pipeline propuesto:**
```
1. zyrocli absorb --code [path]    ← nuevo flag
   │
   ├── Walk del directorio de código
   │   ├── Detectar archivos por extensión (.go, .ts, .py, ...)
   │   ├── Parsear AST (si es Go) → extraer funciones, types, imports
   │   └── Para otros lenguajes: regex + filename patterns
   │
   ├── Generar summary textual (IA)
   │   ├── Prompt: "Resume en 3 líneas qué hace este módulo"
   │   ├── Genera embedding del summary
   │   └── Crea nodo CodeNode en HelixDB
   │
   └── Crear edge HAS_CODENODE desde el Project
```

**¿Dónde se almacenan?** En HelixDB, como cualquier otro nodo. El `summary` es un string, el `embedding` es un `[]float32` de 1536 dimensiones.

**¿Cómo se mantienen actualizados?**
1. Cada CodeNode tiene un `hash` del contenido fuente
2. `zyrocli absorb --code` compara hashes — si el hash cambió, regenera el summary
3. Opcionalmente: un `zyrocli sync --code` que corre en background (futuro)

**Archivos a crear/modificar:**
| Archivo | Acción |
|---------|--------|
| `internal/codeparse/` | **Nuevo paquete** — parseador de AST Go, detector de lenguajes |
| `internal/codeparse/go_ast.go` | **Nuevo** — parseo de `go/ast` para extraer funciones, types, imports |
| `internal/codeparse/summary.go` | **Nuevo** — generación de summary textual + embedding |
| `internal/db/helix/schema.go` | Agregar índices para CodeNode (`NodeEqualityIndex("CodeNode", "path")` ya existe) |
| `cmd/zyrocli/absorb.go` | Agregar flag `--code` para absorber código |
| `internal/db/helix/nodes.go` | Método `UpsertCodeNode` (upsert por path, no crear duplicados) |

**Esfuerzo:** 🔴 Alto — Requiere parseo de AST, generación de summaries, manejo de embeddings, y un pipeline completo.

---

### 3.4 Task → References → CodeNode Graph

**Qué es:** Conexión entre tareas de desarrollo y el código que tocan. Permite trazabilidad: "¿por qué cambió este archivo?" → "porque la Task X lo requirió."

**Modelo actual:**
```
Task (properties: description, phase, status, created_at)
  ↓ REFERENCES → CodeNode    ← el edge ya está definido en el schema
  ↓ REQUIRES   → Skill
```

**Cómo se construye:**

| Método | Pros | Cons | Complejidad |
|--------|------|------|------------|
| **Manual** (dev asigna) | Preciso, humano decide | Tedioso, olvidable | Baja |
| **Automático** (detectar qué módulos toca un diff de git) | No requiere esfuerzo humano | Puede ser ruidoso (mucha gente en un commit) | Media |
| **Híbrido** (detectar + humano confirma) | Lo mejor de ambos | Requiere UI/CLI para confirmar | Media |

**Pipeline automático (recomendado):**
```
1. Dev crea Task "Implementar auth JWT"
2. Dev trabaja en el código (cambia archivos)
3. zyrocli task link [task-id] 
   │
   ├── git diff --name-only HEAD~1  (o desde la Task)
   │   → ["internal/auth/middleware.go", "internal/auth/jwt.go"]
   │
   ├── Para cada archivo modificado:
   │   ├── Buscar CodeNode existente por path
   │   ├── Si NO existe → crearlo (absorb automático)
   │   └── Crear edge REFERENCES (Task → CodeNode)
   │
   └── Output: "Task #12 ahora referencia 2 CodeNodes"
```

**Archivos a crear/modificar:**
| Archivo | Acción |
|---------|--------|
| `cmd/zyrocli/task.go` | **Nuevo** — comandos `task create`, `task link`, `task list` |
| `internal/git/diff.go` | **Nuevo** — wrapper sobre `git diff` para detectar archivos modificados |
| `internal/db/helix/nodes.go` | Método `LinkTaskToCodeNodes(ctx, taskID, paths []string)` |

**Esfuerzo:** 🟡 Medio — La infraestructura base ya existe (edges, CodeNodes). Lo nuevo es el parseo de git diff y el CLI.

---

### 3.5 CLI: `zyrocli context [task]`

**Qué debería hacer:** Dado un task ID, devolver **todos los nodos relevantes** para trabajar en esa task: Skills necesarias, CodeNodes relacionados, Docs de contexto, Patterns aplicables.

**Formato de output:**

Opción A — Texto plano (simple, legible):
```
$ zyrocli context task-42

Context for Task #42: "Implementar auth JWT"

Skills (2):
  • TypeScript (FE, advanced)
  • JWT (BE, intermediate)

CodeNodes (3):
  • internal/auth/middleware.go — "JWT middleware con refresh token logic"
  • internal/auth/jwt.go — "Utilidades de signing y verification"
  • internal/db/helix/client.go — "Wrapper HelixDB con tenant injection"

Documents (1):
  • spec-auth.md — "Especificación del módulo de autenticación"

Patterns (1):
  • Repository Pattern — "Separación de acceso a datos"
```

Opción B — JSON (para alimentar subagentes):
```json
{
  "task_id": "task-42",
  "task_description": "Implementar auth JWT",
  "skills": [
    {"name": "TypeScript", "type": "FE", "level": "advanced"}
  ],
  "code_nodes": [
    {"path": "internal/auth/middleware.go", "summary": "JWT middleware..."}
  ],
  "documents": [
    {"title": "spec-auth.md", "content": "..."}
  ],
  "patterns": [
    {"name": "Repository Pattern", "description": "..."}
  ]
}
```

**Opción C — Ambos** (flag `--format=text|json|prompt`):
- `--format=text` (default) — output legible para humano
- `--format=json` — para consumo programático
- `--format=prompt` — genera un bloque de texto listo para copiar al prompt de un subagente

**Conexión con context bridge existente:**
- El `internal/context/bridge.go` es un MCP client que habla con el binary `context`
- El bridge actual **NO consulta HelixDB** — solo consulta Context7 (docs de librerías)
- `zyrocli context` consulta HelixDB directamente (sin pasar por el bridge)
- El bridge se mantiene para consultas a librerías externas; HelixDB es para contexto interno

**Flujo de integración con OpenCode:**
```
Subagente SDD pide contexto
    ↓
Orquestador llama: zyrocli context task-42 --format=prompt
    ↓
ZyroCLI consulta HelixDB → retorna nodos relevantes
    ↓
Orquestador inyecta resultado en el prompt del subagente
    ↓
Subagente trabaja con contexto preciso (sin leer todo el codebase)
```

**Archivos a crear/modificar:**
| Archivo | Acción |
|---------|--------|
| `cmd/zyrocli/context.go` | **Nuevo** — comando `context` con subcomandos |
| `internal/context/helix_query.go` | **Nuevo** — queries a HelixDB para contexto de task |
| `internal/context/formatter.go` | **Nuevo** — formateadores de output (text, json, prompt) |

**Esfuerzo:** 🟡 Medio — Las queries HelixDB ya están pensadas (schema HQL línea 389-446). Lo nuevo es el CLI y los formatters.

---

## 4. Lo que NO cambiaría

| Capa | Se mantiene igual |
|------|-------------------|
| Context bridge (`bridge.go`) | Sigue siendo MCP client para Context7. HelixDB es adicional, no reemplazo |
| Scheduler F1 | No toca. Parseo de handoff + approval gates intactos |
| Handoff parser | Intacto. Sigue parseando `handoff.yaml` |
| Scaffold | Intacto. Estructura de carpetas no cambia |
| os/exec para scripts Python | Intacto. No hay razón para cambiar |
| `zyrocli db init` | Sigue inicializando schema. Se agregan índices nuevos, pero el comando no cambia |
| `zyrocli absorb` (documentos) | Sigue funcionando igual. Se le agrega flag `--code` |
| Schema HelixDB existente | Los nodos y edges existentes no se rompen. Solo se agregan propiedades nuevas |

---

## 5. Estimación de Esfuerzo por Item

| # | Componente | Esfuerzo | Dependencias | Notas |
|---|-----------|----------|--------------|-------|
| 9 | Developer como tenant aislado | 🟡 Medio | — | Reforzar patrón existente |
| 10 | Cross-project Skill sharing | 🟡 Medio | #9 | Cambios en queries, no en schema |
| 11 | CodeNode summaries automatizados | 🔴 Alto | #9 | Requiere parseo AST + IA + pipeline |
| 12 | Task → References → CodeNode | 🟡 Medio | #11 | Necesita CodeNodes existentes |
| 13 | `zyrocli context [task]` | 🟡 Medio | #11, #12 | CLI + queries + formatters |

**Total estimado:** ~400-600 líneas de código nuevo (Go), distribuidas en:
- `internal/codeparse/`: ~150-200 líneas (nuevo paquete)
- `internal/context/helix_query.go` + `formatter.go`: ~150-200 líneas (nuevo)
- `cmd/zyrocli/task.go` + `context.go` + `skill.go`: ~100-150 líneas (nuevos comandos)
- Modificaciones a archivos existentes: ~50-100 líneas

---

## 6. Dependencias (qué necesita estar listo antes)

### Para empezar la Fase 3:
1. **Fase 2 completada** — HelixDB client funcional con tenant injection ✅ (ya está)
2. **Schema validado** — `zyrocli db init` crea todos los índices ✅ (ya está)
3. **`zyrocli absorb` funcional** — ingesta de documentos ✅ (ya está)

### Para cada componente:
- **#9 (Developer tenant)**: Solo necesita que Fase 2 esté lista
- **#10 (Skill sharing)**: Necesita #9 completado
- **#11 (CodeNodes)**: Necesita Fase 2 + decisiones sobre generación de summaries (¿IA? ¿AST?)
- **#12 (Task→CodeNode)**: Necesita #11 (CodeNodes deben existir)
- **#13 (context CLI)**: Necesita #11 + #12 para tener datos que consultar

### Para Open Source Ready:
- **Configuración de tenant en `.zyro/config.yaml`** — que cada proyecto declare su developer
- **Documentación de setup** — cómo un dev nuevo instala y configura ZyroAgentCLI
- **Tests de aislamiento** — que no se filten datos entre tenants

---

## 7. Riesgos

| Riesgo | Impacto | Probabilidad | Mitigación |
|--------|---------|-------------|------------|
| **Tenant leakage** — bug en query que no filtra `tenant_id` | 🔴 Alto | 🟡 Media | Tests de aislamiento + wrapper que NUNCA desactiva el filtro |
| **CodeNode summary quality** — summaries malos confunden al agente | 🟡 Medio | 🟡 Media | Validación humana + regeneración manual |
| **Performance con muchos tenants** — un solo HelixDB con 100+ developers | 🟡 Medio | 🟢 Baja (para MVP) | HelixDB maneja millones de nodos; row-level filtering es eficiente con índices |
| **Embedding dimension mismatch** — mezclar modelos genera vectores no comparables | 🟡 Medio | 🟢 Baja | Estandarizar un modelo de embedding (OpenAI text-embedding-3-small, 1536-dim) |
| **HelixDB schema drift** — agregar propiedades sin migración puede causar inconsistencias | 🟢 Baja | 🟢 Baja | Schemaless = no hay migración. Validar en capa Go antes de insertar |
| **Complejidad del pipeline de CodeNodes** — parseo de AST + IA + embeddings = mucho código nuevo | 🟡 Medio | 🟡 Media | Empezar con Go solamente, expandir lenguajes después |

---

## 8. Diagrama de Flujo: Conexión entre Componentes

```
┌─────────────────────────────────────────────────────────────────────┐
│                     FASE 3 — Multi-tenant / Open Source Ready        │
└─────────────────────────────────────────────────────────────────────┘

┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Developer   │────→│   Project    │────→│    Task       │
│  (tenant_id) │     │  (hereda     │     │ (references   │
│              │     │   tenant_id) │     │  → CodeNodes) │
└──────────────┘     └──────┬───────┘     └──────┬───────┘
                            │                     │
                            │                     │
              ┌─────────────┼─────────────┐       │
              │             │             │       │
              ▼             ▼             ▼       ▼
       ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
       │  Skill   │  │  CodeNode │  │   Doc    │  │  Pattern │
       │(global/  │  │ (summary  │  │  (spec,  │  │(arch,    │
       │ private) │  │  + embed) │  │  handoff)│  │ design)  │
       └──────────┘  └──────────┘  └──────────┘  └──────────┘
              │             │             │             │
              └─────────────┴──────┬──────┴─────────────┘
                                   │
                                   ▼
                          ┌──────────────┐
                          │  zyrocli     │
                          │  context     │
                          │  [task]      │
                          └──────┬───────┘
                                 │
                    ┌────────────┼────────────┐
                    │            │            │
                    ▼            ▼            ▼
              ┌──────────┐ ┌──────────┐ ┌──────────┐
              │   Text   │ │   JSON   │ │  Prompt  │
              │  (human) │ │  (API)   │ │ (subagent)│
              └──────────┘ └──────────┘ └──────────┘
```

### Flujo de datos para `zyrocli context task-42`:

```
$ zyrocli context task-42 --format=prompt

1. Resolver task_id → task-42
   │
2. Consultar HelixDB:
   │  Task(task-42).REQUIRES_SKILL → [Skill nodes]
   │  Task(task-42).REFERENCES → [CodeNode nodes]
   │  Project(task-42.project).HAS_DOC → [Document nodes]
   │  Project(task-42.project).HAS_PATTERN → [Pattern nodes]
   │
3. Para cada CodeNode, recuperar:
   │  • path
   │  • summary (textual)
   │  • module_type
   │
4. Formatear output (text | json | prompt)
   │
5. Return to caller
```

---

## 9. Orden de Implementación Recomendado

```
Semana 1-2:  #9  Developer como tenant aislado
             └── Reforzar aislamiento en FindNodes, agregar validación

Semana 3-4:  #10 Cross-project Skill sharing
             └── Agregar visibility/owner a Skills, modificar queries

Semana 5-7:  #11 CodeNode summaries automatizados
             └── Crear internal/codeparse/, pipeline de ingestión
             └── Decidir: ¿AST Go primero? ¿IA para summaries?

Semana 8:    #12 Task → References → CodeNode
             └── Comandos CLI + git diff integration

Semana 9:    #13 zyrocli context [task]
             └── Queries + formatters + tests de integración

Semana 10:   Tests de aislamiento + documentación Open Source
```

---

## 10. Resumen para el Usuario

**¿Qué gana tu proyecto con la Fase 3?**

1. **Multi-tenant real**: Cada developer tiene su espacio aislado. No se pisan datos.
2. **Skills compartidos**: Si TypeScript es útil para 10 proyectos, se crea una vez y todos lo usan.
3. **CodeNodes**: El agente "sabe" qué hace cada módulo sin leer todo el código.
4. **Trazabilidad**: Sabés exactamente qué código tocó cada task.
5. **Context injection**: `zyrocli context` le da al subagente solo la información que necesita.

**¿Cuánto falta?** ~400-600 líneas de Go nuevo, distribuidas en 5 componentes. Los 3 primeros (#9, #10, #11) son los más pesados. Los últimos 2 (#12, #13) son más livianos porque ya tienen la infraestructura.

**¿Qué NO hay que tocar?** Todo lo que ya funciona (bridge, scheduler, scaffold, absorb) se mantiene igual. Solo se agregan capas encima.
