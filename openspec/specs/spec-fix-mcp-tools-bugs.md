# Fix MCP Tools Bugs — Especificación Técnica

> **Documento**: Spec Técnica
> **Basado en**: `openspec/proposals/fix-mcp-tools-bugs.md`
> **Estado**: Borrador
> **Fase**: F1 (Spec)

---

## 1. Propósito

Las MCP tools de `helix-integration` (Python) tienen **8 bugs confirmados** que las hacen parcial o totalmente inoperables. El bug más crítico es que `text_search()` hardcodea `property: "name"`, mientras que nodos como `Fact` almacenan su texto en `property: "content"`, resultando en 0 resultados. Adicionalmente hay inconsistencias de labels canónicos (`CodeNode` vs `CodeModule`), edge labels divergentes entre las dos copias del código, un `runner.py` en el proyecto raíz con imports rotos, y datos insuficientes en los retornos de las queries.

Esta especificación detalla los cambios necesarios para que las 7 MCP tools funcionen correctamente: `search_facts`, `search_code`, `search_skills`, `task_context`, `save_to_helix`, `link_to_project`, `find_project`.

---

## 2. Arquitectura

### 2.1 Diagrama de flujo de datos (actual, con bugs)

```
                    +---------------------------------+
                    |       opencode.json (MCP)        |
                    |  uv run ~/.config/zyrocli/...    |
                    +----------+----------------------+
                               | stdio
                    +----------v----------------------+
                    |       runner.py (FastMCP)        |
                    |  helix-integration MCP server    |
                    +--+-----------+----------+-------+
                       |           |          |
              +--------v--+  +-----v----+  +--v--------+
              |search_*.py|  |task_     |  |helix_     |
              |(code/facts|  |context.py|  |write.py   |
              | /skills)  |  |          |  |           |
              +-----+-----+  +----+-----+  +------+----+
                    |             |               |
                    +-------------+---------------+
                                  |
                    +-------------v------------------+
                    |       helix_client.py           |
                    |  text_search() -- property       |
                    |  hardcodeado a "name" X         |
                    |  _get_ids() -- falsy ID bug X   |
                    |  retorna solo IDs X             |
                    +-------------+------------------+
                                  | HTTP POST /v1/query
                    +-------------v------------------+
                    |         HelixDB v3              |
                    |  TextSearchNodes -> 0 resultados |
                    |  si property != campo real      |
                    +--------------------------------+
```

### 2.2 Diagrama de flujo de datos (propuesto, con fixes)

```
                    +---------------------------------+
                    |       opencode.json (MCP)        |
                    +----------+----------------------+
                               | stdio
                    +----------v----------------------+
                    |       runner.py (FastMCP)        |
                    +--+-----------+----------+-------+
                       |           |          |
              +--------v--+  +-----v----+  +--v--------+
              |search_*.py|  |task_     |  |helix_     |
              | -> pasa    |  |context.py|  |write.py   |
              |   property |  | -> edge   |  | -> CodeNode|
              |   correcto |  |   labels |  |   +4 labels|
              +-----+-----+  |   canon. |  +------+----+
                    |        +----+-----+         |
                    |             |               |
                    |   +---------v--------------+ |
                    |   | Fallback edge labels    | |
                    |   | (try canonical, then   | |
                    |   |  legacy)               | |
                    |   +------------------------+ |
                    +--------------+---------------+
                                  |
                    +-------------v------------------+
                    |       helix_client.py           |
                    |  text_search(property=...) V    |
                    |  _get_ids() -- check "$id" in p V|
                    |  retorna dicts completos V      |
                    |  + ProjectReturn step V         |
                    +-------------+------------------+
                                  | HTTP POST /v1/query
                    +-------------v------------------+
                    |         HelixDB v3              |
                    |  TextSearchNodes -> resultados   |
                    |  con propiedades completas      |
                    +--------------------------------+
```

### 2.3 Mapa de archivos afectados

| Archivo | Ruta (LIVE) | Ruta (proyecto raiz) | Ruta (internal mirror) |
|---------|-------------|---------------------|----------------------|
| `helix_client.py` | `~/.config/zyrocli/mcp-tools/` | `./mcp-tools/` | `./internal/opencode/mcptools/` |
| `search_facts.py` | `~/.config/zyrocli/mcp-tools/` | `./mcp-tools/` | `./internal/opencode/mcptools/` |
| `search_code.py` | `~/.config/zyrocli/mcp-tools/` | `./mcp-tools/` | `./internal/opencode/mcptools/` |
| `search_skills.py` | `~/.config/zyrocli/mcp-tools/` | `./mcp-tools/` | `./internal/opencode/mcptools/` |
| `helix_write.py` | `~/.config/zyrocli/mcp-tools/` | `./mcp-tools/` (stub) | `./internal/opencode/mcptools/` |
| `task_context.py` | `~/.config/zyrocli/mcp-tools/` | `./mcp-tools/` | `./internal/opencode/mcptools/` |
| `runner.py` | `~/.config/zyrocli/mcp-tools/` | `./mcp-tools/` | `./internal/opencode/mcptools/` |

**Nota**: `opencode.json` apunta a la copia LIVE (`~/.config/zyrocli/mcp-tools/runner.py`), NO a la del proyecto raiz.

### 2.4 Estado de sincronizacion actual

| Archivo | LIVE vs Internal | LIVE vs Proyecto raiz |
|---------|-----------------|----------------------|
| `helix_client.py` | OK Identicos (304L) | OK Identicos (304L) |
| `search_facts.py` | OK Identicos (30L) | OK Identicos (30L) |
| `search_code.py` | OK Identicos (31L) | OK Identicos (31L) |
| `search_skills.py` | OK Identicos (31L) | OK Identicos (31L) |
| `helix_write.py` | OK Identicos (87L) | X Raiz tiene stub deprecado (22L) |
| `task_context.py` | OK Identicos (72L) | X Raiz usa edge labels legacy (28L) |
| `runner.py` | WARN Identico pero funcional | X Raiz tiene imports rotos |

---

## 3. Especificacion Detallada

### 3.1 Bug #1: `text_search` -- hacer `property` configurable

**Severidad**: CRITICAL
**Archivos**: `helix_client.py` -- TODAS las copias (LIVE, proyecto raiz, internal mirror)

#### Estado actual

```python
# helix_client.py:210-238
async def text_search(
    self, label: str, query: str, limit: int = 10
) -> list[dict]:
    ...
    payload = self._v3_envelope(
        [
            {
                "name": "n",
                "steps": [
                    {
                        "TextSearchNodes": {
                            "label": label,
                            "property": "name",          # <- L226: HARDCODEADO X
                            "query_text": {"Value": {"String": query}},
                            "k": {"Literal": limit},
                        }
                    },
                ],
            }
        ],
        request_type="read",
    )
```

#### Cambio propuesto

```python
# helix_client.py:210-238
async def text_search(
    self, label: str, query: str, limit: int = 10, property: str = "name"
) -> list[dict]:
    ...
    payload = self._v3_envelope(
        [
            {
                "name": "n",
                "steps": [
                    {
                        "TextSearchNodes": {
                            "label": label,
                            "property": property,         # <- L226: DINAMICO V
                            "query_text": {"Value": {"String": query}},
                            "k": {"Literal": limit},
                        }
                    },
                ],
            }
        ],
        request_type="read",
    )
```

#### Justificacion tecnica

- `TextSearchNodes` en HelixDB v3 requiere el campo `property` para saber en que campo hacer busqueda textual.
- Diferentes labels almacenan texto en diferentes propiedades:
  - `Fact` -> `content`
  - `Skill` -> `name`
  - `CodeNode` -> `name`
  - `Project` -> `name`
  - `Pattern` -> `name`, `description`
  - `Document` -> `content`
- El default `"name"` mantiene compatibilidad hacia atras.
- No hay riesgo de breaking change porque:
  - Los callers actuales NO pasan `property` -> usan default `"name"` -> mismo comportamiento que antes.
  - Solo `search_facts()` cambiara para pasar `property="content"` explicitamente.

---

### 3.2 Bug #2: Unificar label `CodeNode` como canonico

**Severidad**: HIGH
**Archivos**: `helix_write.py` -- LIVE e internal mirror (NO el stub del proyecto raiz)

#### Estado actual

```python
# helix_write.py:11-21 (LIVE e internal mirror)
REQUIRED_FIELDS = {
    ...
    "CodeModule": ["path", "language", "summary"],   # <- L19: LABEL ERRONEO X
    ...
}
```

#### Cambio propuesto

```python
# helix_write.py:11-21
REQUIRED_FIELDS = {
    ...
    "CodeNode": ["path", "language", "summary"],     # <- L19: LABEL CANONICO V
    ...
}
```

#### Justificacion tecnica

- `search_code.py` usa `"CodeNode"` como label en `text_search("CodeNode", ...)`.
- `capabilities.py` lista `"CodeNode"` en `allowed_nodes`.
- `helix_write.py` validaba `"CodeModule"`, que no existe como label en HelixDB.
- Al crear un nodo de codigo con `save_to_helix(label="CodeNode", ...)`, la validacion fallaba porque no encontraba el label en REQUIRED_FIELDS (o peor: validaba contra un label que no se usa).
- El label canonico es `CodeNode` porque es el que HelixDB reconoce y el que usan las tools de lectura.

---

### 3.3 Bug #3: `search_facts` -- buscar por `content`

**Severidad**: HIGH
**Archivos**: `search_facts.py` -- TODAS las copias

#### Estado actual

```python
# search_facts.py:18
nodes = await client.text_search("Fact", query, limit=limit)
#                                      ^ Sin property -> usa default "name" X
```

#### Cambio propuesto

```python
# search_facts.py:18
nodes = await client.text_search("Fact", query, limit=limit, property="content")
#                                                                 ^ EXPLICITO V
```

#### Justificacion tecnica

- Los nodos `Fact` en HelixDB almacenan el texto buscable en el campo `content`, no en `name`.
- `TextSearchNodes` con `property: "name"` busca en el campo `name`, que en Facts esta vacio o es un titulo corto.
- Esto hace que `search_facts` retorne 0 resultados aunque existan Facts relevantes.
- El fix requiere que `text_search()` acepte `property` configurable (Bug #1).
- `property="content"` permite buscar en el contenido real del Fact.

**Nota**: `skills`, `code`, `project` siguen usando `property="name"` (default), que es correcto para esos labels.

---

### 3.4 Bug #4: Completar `REQUIRED_FIELDS`

**Severidad**: MEDIUM
**Archivos**: `helix_write.py` -- LIVE e internal mirror

#### Estado actual

```python
# helix_write.py:11-21
REQUIRED_FIELDS = {
    "Spec": ["project_id", "architecture", "modules", "dependencies", "testing_strategy"],
    "Pattern": ["name", "description", "language", "confidence"],
    "Library": ["name", "version", "category", "description"],
    "Skill": ["name", "language", "stars", "source_url"],
    "Decision": ["title", "context", "decision"],
    "Design": ["project_id", "components", "data_flow", "status"],
    "Task": ["project_id", "name", "description", "status"],
    "CodeModule": ["path", "language", "summary"],   # <- se reemplaza por CodeNode
    "Review": ["status", "findings"],
    # FALTAN: Fact, Project, CodeNode, Document X
}
```

#### Cambio propuesto

```python
# helix_write.py:11-21
REQUIRED_FIELDS = {
    "Spec": ["project_id", "architecture", "modules", "dependencies", "testing_strategy"],
    "Pattern": ["name", "description", "language", "confidence"],
    "Library": ["name", "version", "category", "description"],
    "Skill": ["name", "language", "stars", "source_url"],
    "Decision": ["title", "context", "decision"],
    "Design": ["project_id", "components", "data_flow", "status"],
    "Task": ["project_id", "name", "description", "status"],
    "Review": ["status", "findings"],
    "CodeNode": ["path", "language", "summary"],       # <- renombrado + agregado
    "Fact": ["content", "source"],                     # <- NUEVO
    "Project": ["name", "path"],                       # <- NUEVO
    "Document": ["topic_key", "doc_type", "content"],  # <- NUEVO
    # TOTAL: 12 labels (era 9)
}
```

#### Justificacion tecnica

- `capabilities.py` lista 10 labels permitidos para lectura, pero `helix_write.py` solo validaba 9.
- Los labels `Fact`, `Project`, `CodeNode`, `Document` ya existen en HelixDB y se usan activamente.
- Sin validacion, `save_to_helix` para estos labels no verifica campos requeridos, permitiendo nodos incompletos.
- Los campos requeridos se determinaron segun el esquema real de HelixDB:
  - `Fact`: `content` (texto del hecho), `source` (origen: "exploracion", "decision", "conocimiento")
  - `Project`: `name` (nombre del proyecto), `path` (ruta en disco)
  - `CodeNode`: `path` (ruta del archivo), `language` (lenguaje), `summary` (resumen de la funcion/clase)
  - `Document`: `topic_key` (clave del tema), `doc_type` (tipo: "api", "guide", "reference"), `content` (contenido del doc)

---

### 3.5 Bug #5: `text_search` -- incluir propiedades en el retorno

**Severidad**: MEDIUM
**Archivos**: `helix_client.py` -- TODAS las copias

#### Estado actual

```python
# helix_client.py:236-238
result = await self.query(payload)
ids = self._get_ids(result, "n")
return [{"id": i} for i in ids]
#      ^ Solo IDs, sin propiedades X
```

#### Cambio propuesto

Agregar un step `ProjectReturn` despues de `TextSearchNodes` para que HelixDB incluya las propiedades completas en la respuesta. Ademas, crear un metodo separado `_get_properties` (distinto de `_get_ids`) para no romper los callers existentes que esperan IDs numericos.

```python
# helix_client.py -- nuevo metodo
def _get_properties(self, result: dict, name: str = "n") -> list[dict]:
    """Extract full property dicts from a v3 query result.
    Usado por text_search() para retornar propiedades completas.
    """
    data = result.get(name, {})
    if isinstance(data, dict) and "properties" in data:
        return [self._clean_props(p) for p in data["properties"] if p is not None]
    # Fallback: convertir IDs a dicts si no hay properties
    return [{"id": i} for i in self._get_ids(result, name)]

def _clean_props(self, p: dict) -> dict:
    """Normalize: rename $id to id, drop internal keys."""
    return {
        ("id" if k == "$id" else k): v
        for k, v in p.items()
        if not k.startswith("$") or k == "$id"
    }
```

```python
# helix_client.py:218-238 (text_search modificado)
async def text_search(
    self, label: str, query: str, limit: int = 10, property: str = "name"
) -> list[dict]:
    """
    ...
    Response format: {name: {"properties": [{"$id": 1, "name": "...", ...}, ...]}}
    """
    payload = self._v3_envelope(
        [
            {
                "name": "n",
                "steps": [
                    {
                        "TextSearchNodes": {
                            "label": label,
                            "property": property,
                            "query_text": {"Value": {"String": query}},
                            "k": {"Literal": limit},
                        }
                    },
                    {
                        "ProjectReturn": [             # <- NUEVO STEP
                            {"source": "$id", "alias": "id"},
                            {"source": "name", "alias": "name"},
                            {"source": "content", "alias": "content"},
                            {"source": "language", "alias": "language"},
                            {"source": "path", "alias": "path"},
                            {"source": "summary", "alias": "summary"},
                            {"source": "description", "alias": "description"},
                            {"source": "source", "alias": "source"},
                        ]
                    },
                ],
            }
        ],
        request_type="read",
    )
    result = await self.query(payload)
    return self._get_properties(result, "n")  # <- usa _get_properties, no _get_ids
```

#### Formato de retorno

```python
# Antes:
[{"id": 1}, {"id": 2}]

# Despues:
[
    {"id": 1, "name": "AuthMiddleware", "language": "go", "path": "internal/auth/middleware.go", "summary": "JWT validation middleware"},
    {"id": 2, "name": "UserModel", "language": "go", "path": "internal/models/user.go", "summary": "User data model with roles"},
]
```

#### Justificacion tecnica

- Sin `ProjectReturn`, HelixDB v3 retorna solo los IDs de los nodos encontrados.
- El agente MCP recibe IDs opacos sin contexto, forzandolo a hacer N queries adicionales para obtener propiedades.
- `ProjectReturn` trae las propiedades en la misma query, eliminando round-trips.
- Los fields listados son un superconjunto de los campos usados por todos los labels. HelixDB retorna `null` para campos que no existen en un nodo particular.
- Se usa `_get_properties` en vez de modificar `_get_ids` para evitar romper `create_node`, `create_edge` y `get_node` que dependen de `_get_ids` retornando `list[int]`.

---

### 3.6 Bug #6: Falsy ID bug

**Severidad**: MEDIUM
**Archivos**: `helix_client.py` -- TODAS las copias

#### Estado actual

```python
# helix_client.py:82-85
return [
    p.get("$id") or p.get("id")
    for p in data["properties"]
    if p.get("$id") is not None or p.get("id") is not None
]
#       ^ Si $id es 0 (falsy), Python evalua 0 como False y cae a p.get("id") X
```

#### Cambio propuesto

```python
# helix_client.py:82-85
return [
    p.get("$id") if "$id" in p else p.get("id")
    for p in data["properties"]
    if p.get("$id") is not None or p.get("id") is not None
]
#       ^ Verifica presencia de key, no truthiness de valor V
```

#### Justificacion tecnica

- En Python, `0`, `None`, `""`, `[]`, `{}` son falsy.
- `p.get("$id") or p.get("id")` con `$id=0` retorna `p.get("id")` aunque el ID real sea 0.
- Aunque HelixDB no asigna IDs=0 en la practica (los IDs son autoincrementales >0), el patron es incorrecto y puede causar bugs dificiles de depurar si la respuesta contiene `$id` como 0 (ej: edges temporales, nodos virtuales).
- `"$id" in p` es la verificacion correcta: mira si la clave existe en el dict, sin importar su valor.

---

### 3.7 Bug #7: Arreglar/eliminar `runner.py` del proyecto raiz

**Severidad**: CRITICAL
**Archivos**: `mcp-tools/runner.py` (proyecto raiz)

#### Estado actual

```python
# mcp-tools/runner.py (proyecto raiz) -- 32 lineas
from search_code import search_code_tool
from search_facts import search_facts_tool
from search_skills import search_skills_tool
from task_context import task_context_tool
from helix_write import save_to_helix_tool, link_to_project_tool, find_project_tool

server = FastMCP("helix-integration")
server.add_tool(task_context_tool)
...
```

**Problema**: En el proyecto raiz, `helix_write.py` es un stub deprecado de 22 lineas que NO exporta `save_to_helix_tool`, `link_to_project_tool` ni `find_project_tool`. El runner no puede ejecutarse desde el proyecto raiz.

#### Cambio propuesto: **Opcion A (recomendada)**

Eliminar `mcp-tools/runner.py` del proyecto raiz completamente.

**Justificacion**:
1. El runner funcional esta en `~/.config/zyrocli/mcp-tools/runner.py` (LIVE) y en `internal/opencode/mcptools/runner.py` (internal mirror).
2. `opencode.json` referencia la copia LIVE, no la del proyecto raiz.
3. El proyecto raiz contiene codigo adicional (agentes, approvals, wrappers) que no necesita su propio runner duplicado.
4. Eliminar `runner.py` elimina la falsa expectativa de que el proyecto raiz es ejecutable como MCP server.

**Verificacion pre-eliminacion**:
- Buscar referencias a `mcp-tools/runner.py` en scripts, docs, y configs del proyecto raiz.
- La referencia conocida esta en `opencode.json` linea 163, apuntando a `~/.config/zyrocli/mcp-tools/runner.py` -- NO al proyecto raiz.
- Docs como `mcp-tools/README.md` mencionan `runner.py` pero describen su uso generico, no una ruta especifica.

#### Cambio propuesto: **Opcion B (alternativa)**

Convertir `runner.py` en un re-export que delegue a `internal/opencode/mcptools/`:

```python
# mcp-tools/runner.py (proyecto raiz)
"""Re-export: delegate to internal/opencode/mcptools/runner.py"""
import sys
from pathlib import Path

# Add internal mirror to path
internal = Path(__file__).resolve().parent.parent / "internal" / "opencode" / "mcptools"
sys.path.insert(0, str(internal))

from runner import server  # noqa: E402

if __name__ == "__main__":
    server.run(transport="stdio")
```

**Decision**: Se recomienda **Opcion A** (eliminar) porque el proyecto raiz no necesita su propio runner. Si hay scripts que referencian `mcp-tools/runner.py` desde el proyecto raiz, se actualizaran para apuntar a la copia LIVE.

---

### 3.8 Bug #8: Unificar edge labels

**Severidad**: HIGH
**Archivos**: `task_context.py` -- proyecto raiz (`mcp-tools/task_context.py`), LIVE e internal mirror

#### Estado actual

```python
# mcp-tools/task_context.py:20-24 (proyecto raiz -- LEGACY X)
skills = await client.get_outgoing(task_id, "REQUIRES_SKILL")
code_nodes = await client.get_outgoing(task_id, "REFERENCES")
# Faltan: docs, patterns, dependents, dependencies
```

```python
# ~/.config/zyrocli/mcp-tools/task_context.py:42-49 (LIVE -- CANONICO V)
sections["skills"] = await client.get_outgoing(id, "has_skill")
sections["code"] = await client.get_outgoing(id, "has_code")
sections["docs"] = await client.get_outgoing(id, "has_doc")
sections["patterns"] = await client.get_outgoing(id, "has_pattern")
sections["dependents"] = await client.get_incoming(id, "depends_on")
sections["dependencies"] = await client.get_outgoing(id, "depends_on")
```

#### Mapa de edge labels

| Concepto | Label canonico (lowercase) | Label legacy (UPPER) |
|----------|---------------------------|---------------------|
| Skills asociados | `has_skill` | `REQUIRES_SKILL` |
| Codigo asociado | `has_code` | `REFERENCES` |
| Docs asociados | `has_doc` | -- |
| Patterns asociados | `has_pattern` | -- |
| Dependencias | `depends_on` | -- |
| Dependientes (inversa) | `depends_on` (incoming) | -- |

#### Cambio propuesto: agregar logica de fallback en LIVE e internal

En `task_context.py` de LIVE e internal mirror, reemplazar llamadas directas por llamadas con fallback:

```python
# task_context.py:40-49 (con fallbacks)
try:
    sections["skills"] = await _get_outgoing_with_fallback(client, id, "has_skill")
    sections["code"] = await _get_outgoing_with_fallback(client, id, "has_code")
    sections["docs"] = await _get_outgoing_with_fallback(client, id, "has_doc")
    sections["patterns"] = await _get_outgoing_with_fallback(client, id, "has_pattern")
    sections["dependents"] = await _get_incoming_with_fallback(client, id, "depends_on")
    sections["dependencies"] = await _get_outgoing_with_fallback(client, id, "depends_on")
except Exception as exc:
    ...
```

#### Funciones helper de fallback

Agregar en `helix_client.py` como metodos de `HelixClient`, o en cada `task_context.py` como funciones modulares:

```python
EDGE_LABEL_FALLBACKS: dict[str, list[str]] = {
    "has_skill": ["REQUIRES_SKILL"],
    "has_code": ["REFERENCES"],
    "has_doc": [],
    "has_pattern": [],
    "depends_on": [],
}

async def _get_outgoing_with_fallback(
    client: "HelixClient", node_id: int, canonical: str
) -> list[dict]:
    """Try canonical edge label first, fall back to legacy labels if empty."""
    result = await client.get_outgoing(node_id, canonical)
    if result:
        return result
    for legacy in EDGE_LABEL_FALLBACKS.get(canonical, []):
        result = await client.get_outgoing(node_id, legacy)
        if result:
            return result
    return []

async def _get_incoming_with_fallback(
    client: "HelixClient", node_id: int, canonical: str
) -> list[dict]:
    """Try canonical edge label first, fall back to legacy labels if empty."""
    result = await client.get_incoming(node_id, canonical)
    if result:
        return result
    for legacy in EDGE_LABEL_FALLBACKS.get(canonical, []):
        result = await client.get_incoming(node_id, legacy)
        if result:
            return result
    return []
```

#### Cambio propuesto en `mcp-tools/task_context.py` (proyecto raiz)

Reemplazar completamente el contenido del proyecto raiz para que coincida con el de LIVE/internal, usando la misma estructura de 6 secciones y los helpers de fallback.

#### Estrategia de migracion de edges existentes

1. **Fase 1 (este cambio)**: Implementar fallback en codigo. Todos los edges legacy siguen siendo encontrados.
2. **Fase 2 (post-cambio)**: Ejecutar script de migracion en HelixDB que renombre edges `REQUIRES_SKILL` -> `has_skill` y `REFERENCES` -> `has_code`.
3. **Fase 3 (futuro)**: Una vez migrados todos los datos, remover los fallbacks.

#### Justificacion tecnica

- Los edges ya existentes en HelixDB usan labels legacy (`REQUIRES_SKILL`, `REFERENCES`).
- Cambiar el codigo para buscar solo labels canonicos romperia la lectura de edges existentes.
- La estrategia de fallback (try canonico -> try legacy) garantiza que:
  - Los edges nuevos se crean con labels canonicos.
  - Los edges legacy siguen siendo encontrados.
  - No hay downtime ni perdida de datos.
- Los labels canonicos (`has_skill`, `has_code`, `has_doc`, `has_pattern`, `depends_on`) siguen la convencion `snake_case` del sistema HelixDB v3.

---

## 4. Resumen de cambios por archivo

### 4.1 `helix_client.py` -- TODAS las copias (3 archivos)

| Linea | Cambio |
|-------|--------|
| 83 | `p.get("$id") or p.get("id")` -> `p.get("$id") if "$id" in p else p.get("id")` |
| 210-212 | Firma: agregar `property: str = "name"` |
| 226 | `"property": "name"` -> `"property": property` |
| 230-244 | Agregar step `ProjectReturn` con fields |
| 236-238 | Retornar `self._get_properties(result, "n")` en vez de `[{"id": i} for i in ids]` |
| 72-87 | `_get_ids` mantiene retorno de `list[int]` (SIN CAMBIO) |
| Nueva | Agregar metodo `_clean_props(p: dict) -> dict` |
| Nueva | Agregar metodo `_get_properties(result, name) -> list[dict]` |

### 4.2 `search_facts.py` -- TODAS las copias (3 archivos)

| Linea | Cambio |
|-------|--------|
| 18 | `client.text_search("Fact", query, limit=limit)` -> `client.text_search("Fact", query, limit=limit, property="content")` |

### 4.3 `helix_write.py` -- LIVE e internal mirror (2 archivos, NO el stub del proyecto raiz)

| Linea | Cambio |
|-------|--------|
| 19 | `"CodeModule": ["path", "language", "summary"]` -> `"CodeNode": ["path", "language", "summary"]` |
| Despues de 21 | Agregar: `"Fact": ["content", "source"]` |
| | Agregar: `"Project": ["name", "path"]` |
| | Agregar: `"Document": ["topic_key", "doc_type", "content"]` |

### 4.4 `task_context.py` -- todas las copias (3 archivos)

| Archivo | Cambio |
|---------|--------|
| proyecto raiz (`mcp-tools/task_context.py`) | Reemplazar contenido completo para coincidir con LIVE/internal + agregar fallback |
| LIVE (`~/.config/zyrocli/mcp-tools/task_context.py`) | Agregar logica de fallback en `get_outgoing`/`get_incoming` |
| internal (`internal/opencode/mcptools/task_context.py`) | Agregar logica de fallback en `get_outgoing`/`get_incoming` |

### 4.5 `mcp-tools/runner.py` -- proyecto raiz

| Cambio |
|--------|
| **Opcion A (recomendada)**: Eliminar archivo |
| **Opcion B**: Convertir en re-export |

### 4.6 `search_code.py`, `search_skills.py` -- TODAS las copias

Sin cambios. Estos archivos ya usan labels correctos (`CodeNode`, `Skill`) y `property="name"` es el default correcto.

---

## 5. Criterios de Exito

### 5.1 `search_facts("pattern")` retorna Facts con contenido

**Verificacion**:
```bash
cd ~/.config/zyrocli/mcp-tools
uv run python -c "
import asyncio
from search_facts import search_facts_tool
result = asyncio.run(search_facts_tool('pattern', limit=5))
print(result)
"
```

**Criterio**: El JSON retornado debe incluir `results` con al menos un Fact, y cada Fact debe tener campo `content` con texto, no solo `id`.

### 5.2 `search_code("auth")` retorna CodeNodes con propiedades

**Criterio**: Cada CodeNode en `results` debe incluir `path`, `language`, `summary` ademas de `id`.

### 5.3 `search_skills("react")` retorna Skills con nombre y metadata

**Criterio**: Cada Skill en `results` debe incluir `name`, `language`, `stars`, `source_url`.

### 5.4 `task_context(1)` retorna contexto con edge labels consistentes

**Criterio**: El JSON debe incluir secciones `skills`, `code`, `docs`, `patterns`, `dependents`, `dependencies`. Cada seccion debe contener nodos si existen en BD (usando labels canonicos o legacy via fallback).

### 5.5 `save_to_helix` valida campos para Fact, Project, CodeNode, Document

**Verificacion**:
```bash
uv run python -c "
import asyncio
from helix_write import save_to_helix_tool
# Debe fallar con missing fields
result = asyncio.run(save_to_helix_tool('Fact', {'title': 'incompleto'}))
print(result)
# Debe pasar con fields completos
result2 = asyncio.run(save_to_helix_tool('Fact', {'content': 'test', 'source': 'test'}))
print(result2)
"
```

**Criterio**: Primer intento retorna error con `missing fields`. Segundo retorna `status: ok`.

### 5.6 `helix_client.text_search()` acepta `property` como parametro

**Criterio**: `inspect.signature(HelixClient.text_search)` incluye `property` con default `"name"`.

### 5.7 `runner.py` del proyecto raiz no rompe imports

**Criterio**: (si se elimina) El archivo ya no existe. (si se convierte) No hay errores de import.

### 5.8 Falsy ID no causa bug

**Verificacion**: `python -c "from helix_client import HelixClient; c=HelixClient(); r=c._get_ids({'n':{'properties':[{'$id':0}]}}); assert r==[0], f'Got {r}'; print('OK')"`

**Criterio**: `_get_ids` retorna `[0]`, no `[]`.

### 5.9 Edges legacy siguen siendo encontrados via fallback

**Criterio**: `_get_outgoing_with_fallback(client, node, "has_skill")` retorna resultados aunque los edges esten etiquetados como `REQUIRES_SKILL`.

### 5.10 Todas las copias sincronizadas

**Verificacion**:
```bash
for f in helix_client.py search_facts.py search_code.py search_skills.py helix_write.py task_context.py runner.py; do
  diff ~/.config/zyrocli/mcp-tools/$f ~/Projects/ZyroAgentCLI/internal/opencode/mcptools/$f && echo "$f: OK" || echo "$f: DIFFERS"
done
```

**Criterio**: Todos los `diff` retornan vacio.

---

## 6. Pruebas

### 6.1 Pruebas unitarias (manuales)

| Bug | Prueba | Comando |
|-----|--------|---------|
| #1 | text_search acepta property | `python -c "from helix_client import HelixClient; import inspect; print(inspect.signature(HelixClient.text_search))"` |
| #2 | CodeNode canonico | `python -c "from helix_write import REQUIRED_FIELDS; assert 'CodeNode' in REQUIRED_FIELDS; print('OK')"` |
| #3 | search_facts pasa property=content | Inspeccion visual de `search_facts.py` |
| #4 | REQUIRED_FIELDS completo | `python -c "from helix_write import REQUIRED_FIELDS; assert all(l in REQUIRED_FIELDS for l in ['Fact','Project','CodeNode','Document']); print('OK')"` |
| #5 | text_search retorna dicts completos | Ejecutar contra HelixDB real |
| #6 | Falsy ID | `python -c "from helix_client import HelixClient; c=HelixClient(); r=c._get_ids({'n':{'properties':[{'$id':0}]}}); assert r==[0]; print('OK')"` |
| #7 | runner.py eliminado/funcional | `ls mcp-tools/runner.py` debe fallar |
| #8 | Edge labels con fallback | Mock de HelixDB que retorne vacio para canonico y resultados para legacy |

### 6.2 Smoke test de integracion

```bash
# 1. Verificar HelixDB corriendo
curl -s http://localhost:6969/health

# 2. Probar search_facts
uv run --directory ~/.config/zyrocli/mcp-tools python -c "
import asyncio, json
from search_facts import search_facts_tool
r = asyncio.run(search_facts_tool('test'))
print('search_facts OK:', len(json.loads(r)['results']), 'results')"

# 3. Probar search_code
uv run --directory ~/.config/zyrocli/mcp-tools python -c "
import asyncio, json
from search_code import search_code_tool
r = asyncio.run(search_code_tool('auth'))
print('search_code OK:', len(json.loads(r)['results']), 'results')"

# 4. Probar search_skills
uv run --directory ~/.config/zyrocli/mcp-tools python -c "
import asyncio, json
from search_skills import search_skills_tool
r = asyncio.run(search_skills_tool('python'))
print('search_skills OK:', len(json.loads(r)['results']), 'results')"

# 5. Probar task_context
uv run --directory ~/.config/zyrocli/mcp-tools python -c "
import asyncio, json
from task_context import task_context_tool
r = asyncio.run(task_context_tool(1))
print('task_context OK:', json.loads(r).get('skills', 'no skills'))"

# 6. Probar text_search con property explicito
uv run --directory ~/.config/zyrocli/mcp-tools python -c "
import asyncio, json
from helix_client import HelixClient
c = HelixClient()
r = asyncio.run(c.text_search('Fact', 'test', property='content'))
print('text_search property=content OK:', len(r), 'results')"
```

---

## 7. Riesgos y Mitigaciones

| ID | Riesgo | Probabilidad | Impacto | Mitigacion |
|----|--------|-------------|---------|------------|
| R1 | Cambiar `property` default rompe callers | Baja | Alto | Mantener default `"name"`. Solo `search_facts` pasa `"content"`. |
| R2 | `ProjectReturn` trae datos sensibles | Baja | Medio | Fields explicitos y publicos (path, language, summary). HelixDB no almacena secretos. |
| R3 | Eliminar `runner.py` rompe scripts | Media | Medio | Verificar referencias. Solo `opencode.json` referencia LIVE. |
| R4 | Edge labels legacy se pierden | Alta | Alto | **Mitigacion principal**: fallback en codigo. Canonico -> Legacy. |
| R5 | Dos copias se desincronizan en futuro | Alta | Medio | Documentar sync como parte del workflow. Ideal: shared lib. |
| R6 | `_get_properties` cambia formato de retorno | Media | Alta | `_get_ids` NO se modifica (sigue retornando `list[int]`). `text_search` usa `_get_properties` que retorna `list[dict]`. Callers de `_get_ids` no se ven afectados. |
| R7 | `_get_ids` con `$id=0` mal manejado en otros metodos | Baja | Baja | `_get_ids` se usa en `create_node`, `create_edge`, `get_node`. Estos reciben respuestas con `"properties"` donde `$id` siempre es >0. El fix es preventivo. |

---

## 8. Dependencias y Orden de Implementacion

### 8.1 Orden recomendado

1. **Bug #1 + Bug #6 + Bug #5**: Modificar `helix_client.py` una sola vez (property configurable, falsy ID, ProjectReturn + _get_properties)
2. **Bug #3**: Pasar `property="content"` desde `search_facts.py` (depende de #1)
3. **Bug #2 + Bug #4**: Modificar `helix_write.py` (labels + REQUIRED_FIELDS)
4. **Bug #8**: Unificar edge labels con fallback en `task_context.py` (LIVE, internal, proyecto raiz)
5. **Bug #7**: Eliminar `runner.py` del proyecto raiz (ultimo porque no tiene dependencias)
6. **Sincronizacion**: Propagar cambios a las 3 copias y verificar con `diff`

### 8.2 Archivos a modificar por paso

| Paso | Archivos | Bugs |
|------|----------|------|
| 1 | `helix_client.py` (x3) | #1, #5, #6 |
| 2 | `search_facts.py` (x3) | #3 |
| 3 | `helix_write.py` (x2: LIVE, internal) | #2, #4 |
| 4 | `task_context.py` (x3) | #8 |
| 5 | `mcp-tools/runner.py` (x1: proyecto raiz) | #7 |

### 8.3 Post-implementacion

1. Ejecutar todas las pruebas de la Seccion 6
2. Verificar sincronizacion de las 3 copias
3. Ejecutar script de migracion de edge labels en HelixDB
4. Marcar `REQUIRES_SKILL` y `REFERENCES` como labels legacy deprecados

---

## 9. Convenciones adoptadas

### 9.1 Edge labels

Queda establecida la convencion de edge labels en `snake_case` (lowercase con underscores):

| Concepto | Label | Direccion |
|----------|-------|-----------|
| Skills asociados | `has_skill` | Task -> Skill |
| Codigo asociado | `has_code` | Task -> CodeNode |
| Docs asociados | `has_doc` | Task -> Document |
| Patterns asociados | `has_pattern` | Task -> Pattern |
| Dependencias | `depends_on` | Task -> Task |

### 9.2 Labels canonicos de nodos

| Label canonico | Uso |
|---------------|-----|
| `CodeNode` | Nodos de codigo (reemplaza `CodeModule`) |
| `Fact` | Hechos y conocimiento |
| `Project` | Proyectos |
| `Document` | Documentacion |
| `Skill`, `Pattern`, `Library`, `Task`, `Spec`, `Design`, `Decision`, `Review` | Sin cambios |

### 9.3 Sincronizacion

Ambas copias de MCP tools (`~/.config/zyrocli/mcp-tools/` y `internal/opencode/mcptools/`) deben mantenerse identicas. Cualquier cambio en una debe reflejarse en la otra inmediatamente. Como mecanismo de verificacion, ejecutar `diff` entre ambas copias despues de cada cambio.
