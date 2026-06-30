# Fix MCP Tools Bugs — Diseño Técnico

> **Documento**: Design Técnico  
> **Fase**: F2 (Design)  
> **Basado en**: `openspec/specs/spec-fix-mcp-tools-bugs.md`, `openspec/specs/spec-acceptance-criteria-tracking.md`  
> **Estado**: Final  

---

## 1. Resumen Ejecutivo

Se diseñan los cambios para corregir **8 bugs** en las MCP tools de `helix-integration` (Python). El bug más crítico es que `text_search()` hardcodea `property: "name"`, haciendo que `search_facts` retorne 0 resultados siempre (los Facts usan `content`). Adicionalmente hay labels inconsistentes (`CodeModule` vs `CodeNode`), edge labels divergentes entre copias (`REQUIRES_SKILL` vs `has_skill`), un `runner.py` con imports rotos, y datos insuficientes en los retornos de queries.

El diseño cubre 2 copias del código (`internal/opencode/mcptools/` como fuente canónica y `mcp-tools/` como copia del proyecto raíz que debe sincronizarse) más la copia LIVE en `~/.config/zyrocli/mcp-tools/`.

**Conexión con acceptance-criteria-tracking**: Este diseño habilita que `task_context` retorne `acceptance_criteria` de nodos Task y que `search_facts` pueda buscar acceptance criteria por contenido (cuando se persistan como Facts en el cambio de tracking).

---

## 2. Componentes Afectados — Mapa de Archivos

### 2.1 Las 3 copias

| Archivo | `internal/opencode/mcptools/` (canónica) | `mcp-tools/` (proyecto raíz) | `~/.config/zyrocli/` (LIVE) |
|---------|------------------------------------------|------------------------------|-----------------------------|
| `helix_client.py` | 304L ✅ | 304L ✅ idéntica | ✅ idéntica |
| `search_facts.py` | 30L ✅ MCP tool | 11L ❌ simple fn sin `_tool` | ✅ idéntica a canónica |
| `search_code.py` | 31L ✅ MCP tool | 12L ❌ simple fn + `project_id` | ✅ idéntica a canónica |
| `search_skills.py` | 31L ✅ MCP tool | 12L ❌ simple fn | ✅ idéntica a canónica |
| `helix_write.py` | 87L ✅ full impl | 22L ❌ stub deprecado | ✅ idéntica a canónica |
| `task_context.py` | 72L ✅ 6 secciones + canónico | 28L ❌ legacy edges (REQUIRES_SKILL) | ✅ idéntica a canónica |
| `runner.py` | 32L ✅ funcional | 32L ⚠️ imports rotos (mismo contenido pero imports fallan) | ✅ funcional |

### 2.2 Archivos NO afectados

| Archivo | Razón |
|---------|-------|
| `agent.py` | Usa `client.text_search()` inline, no las MCP tools. Se beneficia de los fixes en `helix_client.py` sin cambios. |
| `models.py` | Modelos de datos — sin cambios. `HelixSearchResult` sigue siendo compatible (usa `n["id"]`). |
| `capabilities.py` | Lista de labels permitidos — ya correcta. |
| `boundari_wrapper.py` | Enforce policies — sin cambios directos. |
| `approval.py` | Approval gates — no interactúa con search tools. |
| `embedding_harness.py` | MCP server separado — sin cambios. |

---

## 3. Diagrama de Flujo de Datos (Post-Fix)

```
                    +--------------------------------------+
                    |       opencode.json (MCP)             |
                    |  uv run ~/.config/zyrocli/mcptools/   |
                    |  runner.py (FastMCP server)          |
                    +----------+---------------------------+
                               | stdio
                    +----------v---------------------------+
                    |         runner.py                      |
                    |  Registra tools: search_facts_tool,   |
                    |  search_code_tool, search_skills_tool, |
                    |  task_context_tool, save_to_helix_tool,|
                    |  link_to_project_tool, find_project_tool|
                    +--+-----------+----------+------------+
                       |           |          |
              +--------v--+  +-----v----+  +--v------------+
              |search_*.py|  |task_     |  |helix_write.py  |
              | -> pasa    |  |context.py|  | -> REQUIRED_   |
              |   property |  | -> 6 secc |  |   FIELDS fixed |
              |   correcto |  |   + AC    |  | + CodeNode     |
              |   + _tool  |  | + fallback|  | + Fact/Project/|
              |   + error  |  |   edges  |  |   Document     |
              |   handling |  |          |  |                |
              +-----+------+  +----+-----+  +-------+--------+
                    |              |                |
                    |   +----------v--------------+ |
                    |   | get_outgoing_with_fallback| |
                    |   | (has_skill→REQUIRES_SKILL)| |
                    |   | (has_code→REFERENCES)    | |
                    |   +--------------------------+ |
                    +--------------+-----------------+
                                   |
                     +-------------v--------------------+
                     |       helix_client.py              |
                     |  text_search(property=...) ✅      |
                     |  _get_ids falsy ID fix ✅          |
                     |  _get_properties + ProjectReturn ✅|
                     |  get_node(include_properties) ✅   |
                     |  get_outgoing_with_fallback() ✅   |
                     +-------------+---------------------+
                                   | HTTP POST /v1/query
                     +-------------v---------------------+
                     |         HelixDB v3                  |
                     |  TextSearchNodes → resultados       |
                     |  con propiedades completas          |
                     |  Edge traversal → nodos relacionados|
                     +-----------------------------------+
```

---

## 4. Decisiones de Diseño

### 4.1 Firma de `text_search()` con `property` configurable

**Decisión**: Agregar parámetro `property: str = "name"` a `text_search()`.

```python
async def text_search(
    self, label: str, query: str, limit: int = 10, property: str = "name"
) -> list[dict]:
```

**Fundamento**:
- `"name"` como default mantiene compatibilidad hacia atrás (todos los callers existentes siguen funcionando).
- Solo `search_facts` cambia para pasar `property="content"`.
- El nombre `property` deliberadamente coincide con el field de HelixDB TextSearchNodes, aunque sombrea el built-in de Python. Es el nombre más reconocible para quien lea el código.
- Alternativa considerada y **descartada**: `prop` o `field` — rompen la correspondencia con la API de HelixDB.

**Riesgo mitigado**: Ningún caller actual pasa `property` por nombre, por lo que no hay breaking changes en el paso del parámetro.

### 4.2 `_get_properties` y `ProjectReturn` para retorno completo

**Decisión**: Agregar dos métodos privados y un step `ProjectReturn` en `text_search()`.

```python
def _get_properties(self, result: dict, name: str = "n") -> list[dict]:
    """Extrae dicts completos de properties desde respuesta v3.
    Retorna [{id, name, content, ...}, ...] o fallback a [{"id": i}].
    """
    data = result.get(name, {})
    if isinstance(data, dict) and "properties" in data:
        return [self._clean_props(p) for p in data["properties"] if p is not None]
    return [{"id": i} for i in self._get_ids(result, name)]

def _clean_props(self, p: dict) -> dict:
    """Normaliza: $id → id, elimina claves internas que empiezan con $."""
    return {
        ("id" if k == "$id" else k): v
        for k, v in p.items()
        if not k.startswith("$") or k == "$id"
    }
```

**Fundamento**:
- `_get_ids()` **NO se modifica** — sigue retornando `list[int]`. Los callers de `create_node`, `create_edge`, `get_node` (modo legacy) no se ven afectados.
- `_get_properties()` es el único método que usa `text_search`.
- `ProjectReturn` fields: lista explícita de 8 campos (id, name, content, language, path, summary, description, source). HelixDB retorna `null` para campos que no existen en el nodo.
- **Alternativa descartada**: `ProjectReturn` con `"*"` — no está confirmado que HelixDB soporte wildcards.

### 4.3 Mapeo label→property para cada search

| Archivo | Label | Property | Argumento |
|---------|-------|----------|-----------|
| `search_facts.py` | `Fact` | `"content"` | Facts almacenan texto buscable en `content` |
| `search_code.py` | `CodeNode` | `"name"` (default) | CodeNode usa `name` como título buscable |
| `search_skills.py` | `Skill` | `"name"` (default) | Skills usan `name` |
| `helix_write.py:find_project_tool` | `Project` | `"name"` (default) | Projects usan `name` |
| `agent.py:search_code` | `CodeNode` | `"name"` (default) | inline en agent.py |
| `agent.py:search_skills` | `Skill` | `"name"` (default) | inline en agent.py |

**Nota**: Si en el futuro se necesita buscar Patterns por `description`, `helix_client.text_search("Pattern", query, property="description")` funcionará sin cambios en el cliente.

### 4.4 Estrategia de Fallback para Edge Labels Legacy

**Decisión**: Agregar métodos `get_outgoing_with_fallback()` y `get_incoming_with_fallback()` en `HelixClient`.

```python
EDGE_LABEL_FALLBACKS: dict[str, list[str]] = {
    "has_skill": ["REQUIRES_SKILL"],   # canonico → legacy
    "has_code": ["REFERENCES"],
    "has_doc": [],                     # sin legacy
    "has_pattern": [],                 # sin legacy
    "depends_on": [],                  # sin legacy
}
```

**Flujo**:
1. Intentar con label canónico (`has_skill`).
2. Si retorna vacío, intentar cada label legacy en orden (`REQUIRES_SKILL`).
3. Retornar resultados del primer label que tenga datos.
4. Si todos vacíos, retornar `[]`.

**Ubicación**: Métodos en `HelixClient` (en `helix_client.py`) para reutilización por cualquier tool futura.

```python
async def get_outgoing_with_fallback(self, node_id: int, canonical: str) -> list[dict]:
    result = await self.get_outgoing(node_id, canonical)
    if result:
        return result
    for legacy in EDGE_LABEL_FALLBACKS.get(canonical, []):
        result = await self.get_outgoing(node_id, legacy)
        if result:
            return result
    return []
```

**Uso en `task_context.py`**:
```python
sections["skills"] = await client.get_outgoing_with_fallback(id, "has_skill")
sections["code"] = await client.get_outgoing_with_fallback(id, "has_code")
# ... resto de secciones igual
```

**Migración futura**:
- Fase 1 (este cambio): Fallback en código. Edges legacy siguen siendo encontrados.
- Fase 2 (post-cambio): Script de migración en HelixDB renombra `REQUIRES_SKILL`→`has_skill`, `REFERENCES`→`has_code`.
- Fase 3 (futuro): Una vez migrados todos los datos, remover fallbacks de `EDGE_LABEL_FALLBACKS`.

### 4.5 `get_node` y Retorno de `acceptance_criteria`

**Decisión**: Extender `get_node()` con parámetro `include_properties: bool = False`.

```python
async def get_node(
    self, label: str, id: int, include_properties: bool = False
) -> dict | None:
```

**Comportamiento**:
- `include_properties=False` (default): comportamiento actual — retorna `{"id": id}`.
- `include_properties=True`: agrega step `ProjectReturn` con fields completos, retorna `{"id": ..., "name": ..., "description": ..., "acceptance_criteria": ..., ...}`.

**En `task_context.py`**:
```python
task_node = await client.get_node("Task", id, include_properties=True)
acceptance_criteria = task_node.get("acceptance_criteria", []) if task_node else []
```

**Formato en respuesta de `task_context_tool`**:
```json
{
  "task_id": 42,
  "acceptance_criteria": [
    {"id": "AC-001", "description": "El middleware debe rechazar requests sin token", "status": "pending"},
    {"id": "AC-002", "description": "Debe retornar 401 con mensaje", "status": "verified"}
  ],
  "skills": [...],
  "code": [...],
  "docs": [...],
  "patterns": [...],
  "dependents": [...],
  "dependencies": [...]
}
```

**Justificación**:
- Backward compatible: los 3 callers existentes (`task_context.py` ×2 + `agent.py`) no pasan `include_properties` y reciben el mismo formato de siempre.
- El campo `acceptance_criteria` en la respuesta es un array JSON libre; su estructura interna depende del schema de HelixDB (definido en `spec-acceptance-criteria-tracking.md`).

### 4.6 `search_facts` y Búsqueda de Acceptance Criteria

**Decisión**: `search_facts` busca exclusivamente en nodos `Fact` con `property="content"`. Los acceptance criteria se vuelven searchables cuando se persisten como nodos `Fact` (responsabilidad del cambio de acceptance-criteria-tracking).

**Flujo**:
1. `spec-acceptance-criteria-tracking.md` define que cada criterion se persistirá como nodo `Fact` con `content = descripción del criterion` y `source = "acceptance_criteria"`.
2. `search_facts("reject requests without token")` ejecuta `text_search("Fact", query, property="content")`.
3. Los Facts de tipo acceptance_criteria aparecen en los resultados porque comparten el campo `content`.
4. El campo `source: "acceptance_criteria"` permite filtrar en el cliente si solo se quieren criteria.

**Limitación conocida**: Los acceptance criteria almacenados como JSON array dentro del nodo Task (`acceptance_criteria: [...]`) NO son searchables por `TextSearchNodes` porque HelixDB busca en propiedades planas (string), no en sub-campos de arrays.

**Mitigación**: Documentar que para searchabilidad, los criteria DEBEN persistirse como nodos Fact además de incluirse en el Task node properties. Esto es responsabilidad del cambio de tracking, no de este fix.

### 4.7 Estrategia de Sincronización de las 3 Copias

**Diagnóstico actual**:

| Copia | Estado |
|-------|--------|
| `internal/opencode/mcptools/` | ✅ **Canónica** — contiene versiones MCP tool con error handling, 6 secciones en task_context, full helix_write |
| `mcp-tools/` | ❌ **Divergida** — 5/7 archivos son diferentes (versiones simples/deprecadas) |
| `~/.config/zyrocli/mcp-tools/` | ✅ **Idéntica a canónica** — se despliega desde canónica |

**Decisión**: Convertir `internal/opencode/mcptools/` en la fuente canónica única y reemplazar los archivos divergentes en `mcp-tools/` con copias de la canónica.

**Pasos**:
1. Aplicar todos los fixes en `internal/opencode/mcptools/` primero.
2. Reemplazar archivos en `mcp-tools/`:
   - `mcp-tools/search_facts.py` → copia de `internal/opencode/mcptools/search_facts.py`
   - `mcp-tools/search_code.py` → copia de `internal/opencode/mcptools/search_code.py`
   - `mcp-tools/search_skills.py` → copia de `internal/opencode/mcptools/search_skills.py`
   - `mcp-tools/helix_write.py` → copia de `internal/opencode/mcptools/helix_write.py`
   - `mcp-tools/task_context.py` → copia de `internal/opencode/mcptools/task_context.py`
3. Propagar cambios a LIVE (manual: `cp internal/opencode/mcptools/* ~/.config/zyrocli/mcp-tools/`).
4. Verificar con `diff` que las 3 copias sean idénticas.

**Mecanismo preventivo**: Agregar un alias/make target `make sync-mcptools` que ejecute:
```bash
for f in helix_client.py search_facts.py search_code.py search_skills.py helix_write.py task_context.py runner.py; do
  cp internal/opencode/mcptools/$f mcp-tools/$f
done
```

### 4.8 `runner.py` del Proyecto Raíz

**Decisión**: **Opción A (Eliminar)**.

**Justificación**:
1. `opencode.json` referencia la copia LIVE (`~/.config/zyrocli/mcp-tools/runner.py`), no la del proyecto raíz.
2. El proyecto raíz tiene `agent.py` que es su propio entry point para el agente PydanticAI, no necesita un MCP server duplicado.
3. La canónica en `internal/opencode/mcptools/runner.py` es el MCP server de desarrollo/testing.
4. Eliminar elimina la falsa expectativa de que se pueda ejecutar `uv run --directory mcp-tools runner.py` (falla porque los imports no existen).

**Verificación pre-eliminación**:
- Buscar referencias a `mcp-tools/runner.py` en scripts, docs, `Makefile`, `opencode.json`.
- La única referencia conocida está en `opencode.json` línea ~163, que apunta a `~/.config/zyrocli/mcp-tools/runner.py` (LIVE), no al proyecto raíz.

**Alternativa considerada y descartada (Opción B)**: Convertir en re-export que delegue a `internal/opencode/mcptools/runner.py`. Se descarta porque agrega complejidad innecesaria (un wrapper que redirige a `sys.path.insert`) sin beneficio real, ya que nadie ejecuta el runner del proyecto raíz.

---

## 5. Módulos/Archivos a Modificar — Cambios Específicos

### 5.1 `helix_client.py` — Ambos archivos (mcp-tools/ + internal/)

| # | Línea(s) | Cambio | Bug |
|---|----------|--------|-----|
| 1 | 83 | `p.get("$id") or p.get("id")` → `p.get("$id") if "$id" in p else p.get("id")` | #6 |
| 2 | 210-212 | Firma: agregar `property: str = "name"` | #1 |
| 3 | 226 | `"property": "name"` → `"property": property` | #1 |
| 4 | 230-244 | Agregar step `ProjectReturn` después de `TextSearchNodes` con 8 fields | #5 |
| 5 | 236-238 | `return [{"id": i} for i in ids]` → `return self._get_properties(result, "n")` | #5 |
| 6 | Nueva | Agregar método `_clean_props(p: dict) -> dict` | #5 |
| 7 | Nueva | Agregar método `_get_properties(result, name) -> list[dict]` | #5 |
| 8 | 184-208 | `get_node`: agregar `include_properties: bool = False`, `ProjectReturn` condicional | AC |
| 9 | Nueva | Agregar `EDGE_LABEL_FALLBACKS` dict (módulo, no clase) | #8 |
| 10 | Nueva | Agregar método `get_outgoing_with_fallback(node_id, canonical) -> list[dict]` | #8 |
| 11 | Nueva | Agregar método `get_incoming_with_fallback(node_id, canonical) -> list[dict]` | #8 |

### 5.2 `search_facts.py` — Ambos archivos

| # | Línea | Cambio | Bug |
|---|-------|--------|-----|
| 1 | 18 | `client.text_search("Fact", query, limit=limit)` → `client.text_search("Fact", query, limit=limit, property="content")` | #3 |

> **Nota**: En `mcp-tools/search_facts.py` (proyecto raíz, 11L), también se debe reemplazar con la versión MCP tool completa (30L) de la canónica.

### 5.3 `search_code.py`, `search_skills.py`

**Sin cambios en el contenido de las funciones**. Ambos ya usan labels correctos (`CodeNode`, `Skill`) y `property="name"` es default correcto.

> **Acción**: Reemplazar `mcp-tools/search_code.py` (12L) y `mcp-tools/search_skills.py` (12L) con las versiones canónicas (31L c/u) que incluyen error handling y firma MCP tool.

### 5.4 `helix_write.py` — Canónica (internal/)

| # | Línea | Cambio | Bug |
|---|-------|--------|-----|
| 1 | 19 | `"CodeModule": ["path", "language", "summary"]` → `"CodeNode": ["path", "language", "summary"]` | #2 |
| 2 | Después de 21 | Agregar `"Fact": ["content", "source"]` | #4 |
| 3 | | Agregar `"Project": ["name", "path"]` | #4 |
| 4 | | Agregar `"CodeNode": ["path", "language", "summary"]` | #4 |
| 5 | | Agregar `"Document": ["topic_key", "doc_type", "content"]` | #4 |

> **Nota**: `mcp-tools/helix_write.py` (proyecto raíz, 22L stub deprecado) debe reemplazarse por la versión canónica (87L) como parte de la sincronización.

### 5.5 `task_context.py` — Canónica (internal/) + Proyecto raíz

| # | Línea | Cambio | Bug |
|---|-------|--------|-----|
| 1 | 22-28 | Cambiar `client.get_node("Task", id)` → `client.get_node("Task", id, include_properties=True)` | AC |
| 2 | Después de 28 | Extraer `acceptance_criteria` del `task_node` | AC |
| 3 | 42-49 | Reemplazar `client.get_outgoing(id, "has_skill")` por `client.get_outgoing_with_fallback(id, "has_skill")` (y análogos para code, docs, patterns, depends) | #8 |
| 4 | 60-71 | En el JSON de retorno, agregar campo `"acceptance_criteria": acceptance_criteria` | AC |

> **Acción para proyecto raíz**: Reemplazar completamente `mcp-tools/task_context.py` (28L, legacy) con la versión canónica de `internal/opencode/mcptools/task_context.py` (72L, con 6 secciones) y luego aplicar los cambios #1-#4.

### 5.6 `mcp-tools/runner.py` — Proyecto raíz

| # | Acción | Bug |
|---|--------|-----|
| 1 | **Eliminar** `mcp-tools/runner.py` | #7 |

### 5.7 Archivos a Reemplazar por Sincronización

| Archivo destino (mcp-tools/) | Reemplazar por (internal/opencode/mcptools/) |
|------------------------------|---------------------------------------------|
| `mcp-tools/search_facts.py` | `internal/opencode/mcptools/search_facts.py` |
| `mcp-tools/search_code.py` | `internal/opencode/mcptools/search_code.py` |
| `mcp-tools/search_skills.py` | `internal/opencode/mcptools/search_skills.py` |
| `mcp-tools/helix_write.py` | `internal/opencode/mcptools/helix_write.py` |
| `mcp-tools/task_context.py` | `internal/opencode/mcptools/task_context.py` (post-fixes) |

---

## 6. Pruebas — Estrategia de Verificación

### 6.1 Unit tests automáticos (vía Python inline)

| # | Bug | Prueba | Comando |
|---|-----|--------|---------|
| 1 | #1 | `text_search` acepta `property` en firma | `python -c "from helix_client import HelixClient; import inspect; p = inspect.signature(HelixClient.text_search).parameters; assert 'property' in p and p['property'].default == 'name' ; print('OK')"` |
| 2 | #2 | `REQUIRED_FIELDS` tiene `CodeNode` en vez de `CodeModule` | `python -c "from helix_write import REQUIRED_FIELDS; assert 'CodeNode' in REQUIRED_FIELDS and 'CodeModule' not in REQUIRED_FIELDS; print('OK')"` |
| 3 | #4 | `REQUIRED_FIELDS` completo con Fact/Project/Document | `python -c "from helix_write import REQUIRED_FIELDS; assert all(l in REQUIRED_FIELDS for l in ['Fact','Project','CodeNode','Document']); print('OK')"` |
| 4 | #6 | Falsy ID: `$id=0` retorna `[0]` | `python -c "from helix_client import HelixClient; c=HelixClient(); r=c._get_ids({'n':{'properties':[{'$id':0}]}}); assert r==[0]; print('OK')"` |
| 5 | #5 | `_get_properties` extrae y normaliza | `python -c "from helix_client import HelixClient; c=HelixClient(); r=c._get_properties({'n':{'properties':[{'$id':1,'name':'x'}]}}); assert r==[{'id':1,'name':'x'}]; print('OK')"` |
| 6 | #8 | Fallback edge labels | `python -c "from helix_client import EDGE_LABEL_FALLBACKS; assert EDGE_LABEL_FALLBACKS['has_skill']==['REQUIRES_SKILL']; assert EDGE_LABEL_FALLBACKS['has_code']==['REFERENCES']; print('OK')"` |
| 7 | AC | `get_node` con `include_properties=True` | Prueba manual contra HelixDB: verificar que retorna `acceptance_criteria` si existe. |

### 6.2 Smoke tests de integración (contra HelixDB real)

```bash
# 1. Verificar HelixDB corriendo
curl -s http://localhost:6969/health

# 2. search_facts con property=content
cd ~/.config/zyrocli/mcp-tools
uv run python -c "
import asyncio, json
from search_facts import search_facts_tool
r = asyncio.run(search_facts_tool('test'))
d = json.loads(r)
assert 'results' in d, f'Missing results key: {d}'
if d['results']:
    assert 'content' in d['results'][0], 'Results missing content field'
print(f'OK: {d[\"count\"]} results')
"

# 3. search_code retorna propiedades completas
uv run python -c "
import asyncio, json
from search_code import search_code_tool
r = asyncio.run(search_code_tool('auth'))
d = json.loads(r)
if d['results']:
    for field in ['id', 'name', 'language', 'path', 'summary']:
        assert field in d['results'][0], f'Missing field {field}'
print(f'OK: {d[\"count\"]} code nodes')
"

# 4. task_context retorna acceptance_criteria
uv run python -c "
import asyncio, json
from task_context import task_context_tool
r = asyncio.run(task_context_tool(1))
d = json.loads(r)
assert 'acceptance_criteria' in d, 'Missing acceptance_criteria in response'
assert set(d.keys()) >= {'skills', 'code', 'docs', 'patterns', 'dependents', 'dependencies'}
print(f'OK: task 1 has {len(d[\"acceptance_criteria\"])} criteria')
"

# 5. verify edge label fallback
uv run python -c "
import asyncio
from helix_client import HelixClient
c = HelixClient()
r = asyncio.run(c.get_outgoing_with_fallback(1, 'has_skill'))
print(f'OK: skill edges = {len(r)}')  # should find REQUIRES_SKILL edges
"
```

### 6.3 Verificación de sincronización

```bash
# Verificar que mcp-tools/ e internal/opencode/mcptools/ son idénticos
for f in helix_client.py search_facts.py search_code.py search_skills.py helix_write.py task_context.py runner.py; do
  diff internal/opencode/mcptools/$f mcp-tools/$f > /dev/null 2>&1 && echo "$f: OK" || echo "$f: DIFFERS"
done

# Verificar que runner.py no existe (si se eliminó)
test ! -f mcp-tools/runner.py && echo "runner.py: DELETED OK" || echo "runner.py: STILL EXISTS"
```

### 6.4 Verificación backward compatibility

```bash
# agent.py sigue funcionando con text_search()
cd mcp-tools
uv run python -c "
import asyncio
from helix_client import HelixClient
c = HelixClient()
# Llamada sin property (usa default 'name')
r = asyncio.run(c.text_search('Skill', 'python', limit=3))
assert isinstance(r, list)
if r:
    assert 'id' in r[0]  # backward compatible: tiene id
print(f'OK: text_search returns {len(r)} results with id field')
"
```

---

## 7. Riesgos Técnicos y Mitigaciones

| ID | Riesgo | Probabilidad | Impacto | Mitigación |
|----|--------|-------------|---------|------------|
| R1 | `text_search` ahora retorna `list[dict]` con más campos (no solo `{"id": i}`). Código que esperaba solo `id` podría romperse si usaba `**node`. | Baja | Medio | Todos los callers existentes acceden a `n["id"]` o `n.get("id")` — sigue funcionando. El `id` está presente en el nuevo formato. Verificar en `agent.py` (líneas 67-77, 105-115) que usan `n["id"]`. |
| R2 | `get_node` con `include_properties=True` cambia formato de retorno. Callers existentes (`agent.py:147`, `task_context.py:23`) no pasan el flag y reciben el mismo formato de siempre. | Baja | Baja | Default `False` garantiza backward compat. |
| R3 | El campo `project_id` que `mcp-tools/search_code.py` aceptaba se pierde al reemplazar por la versión canónica que no lo soporta. | Media | Media | Verificar que ningún caller usa `search_code(query, project_id=...)`. Si existe, agregar `project_id` a la tool canónica. |
| R4 | `mcp-tools/search_facts.py` (11L) es usado por algún script externo que espera `list[dict]` en vez de `str` JSON. | Baja | Alta | Buscar imports de `from search_facts import search_facts` (sin `_tool`). El grep solo encontró imports en `runner.py`. `agent.py` no lo usa. |
| R5 | LIVE no se sincroniza automáticamente. | Alta | Alta | Agregar `make sync-mcptools` y documentar en `README.md`. Por ahora, sincronización manual vía `diff` + `cp`. |
| R6 | `ProjectReturn` trae `acceptance_criteria` como JSON, pero HelixDB podría truncar o no soportar arrays en ProjectReturn. | Media | Media | Verificar en staging. Si falla, omitir `acceptance_criteria` de ProjectReturn y agregar una query separada. |
| R7 | AL eliminar `runner.py`, alguien intenta `uv run --directory mcp-tools runner.py` y falla con `FileNotFoundError`. | Media | Baja | El archivo ya está roto (imports fallan). Eliminarlo es honesto. Documentar en `mcp-tools/README.md` si existe. |
| R8 | Edge labels legacy pueden tener datos inconsistentes (mismos edges con ambos labels). | Baja | Baja | Estrategia "first non-empty" garantiza que se usen los canónicos si existen. Los legacy solo se consultan si canónicos retornan vacío. |
| R9 | `agent.py` llama `client.text_search("CodeNode", ...)` — sin `property` → usa `"name"` default. Esto es correcto, pero si en futuro se necesitara `"content"` para CodeNode, `agent.py` no lo pasa. | Baja | Baja | Por ahora CodeNode usa `name`. Si cambia en futuro, actualizar `agent.py` también. |

### 7.1 Riesgo específico: `search_code.py` con `project_id`

El `mcp-tools/search_code.py` del proyecto raíz (12L) acepta un parámetro `project_id` que la versión canónica (31L) NO tiene:

```python
# mcp-tools/search_code.py (actual)
async def search_code(query: str, limit: int = 10, project_id: str | None = None) -> list[dict]:
    client = HelixClient(project_id=project_id)
```

```python
# internal/opencode/mcptools/search_code.py (canónica)
async def search_code_tool(query: str, limit: int = 10) -> str:
    client = HelixClient()
```

**Decisión**: Mantener la versión canónica (sin `project_id`) porque:
1. `agent.py` es el único caller de `search_code` como función, y usa su propia implementación inline.
2. `runner.py` importa `search_code_tool` que es la versión canónica.
3. Si surge necesidad, `project_id` se puede agregar a `HelixClient()` via `AgentDependencies`.

---

## 8. Orden de Implementación Recomendado

| Paso | Archivos | Bugs | Depende de |
|------|----------|------|------------|
| 1 | `internal/opencode/mcptools/helix_client.py` | #1, #5, #6, #8 | — |
| 2 | `mcp-tools/helix_client.py` | #1, #5, #6, #8 | Paso 1 (copiar cambios) |
| 3 | `internal/opencode/mcptools/search_facts.py` | #3 | Paso 1 (text_search property) |
| 4 | `internal/opencode/mcptools/task_context.py` | #8, AC | Paso 1 (fallback helpers) |
| 5 | `internal/opencode/mcptools/helix_write.py` | #2, #4 | — |
| 6 | Reemplazar `mcp-tools/search_facts.py` con canónica | — | Paso 3 |
| 7 | Reemplazar `mcp-tools/search_code.py` con canónica | — | Paso 1 |
| 8 | Reemplazar `mcp-tools/search_skills.py` con canónica | — | Paso 1 |
| 9 | Reemplazar `mcp-tools/helix_write.py` con canónica | — | Paso 5 |
| 10 | Reemplazar `mcp-tools/task_context.py` con canónica + fallback | — | Paso 4 |
| 11 | Eliminar `mcp-tools/runner.py` | #7 | — |
| 12 | Sincronizar LIVE: `cp internal/opencode/mcptools/* ~/.config/zyrocli/mcp-tools/` | — | Pasos 1-11 |
| 13 | Verificar sincronización con `diff` | — | Paso 12 |
| 14 | Ejecutar smoke tests | — | Pasos 1-13 |

---

## 9. Convenciones Adoptadas

### 9.1 Edge labels canónicos

| Concepto | Label | Dirección | Legacy |
|----------|-------|-----------|--------|
| Skills asociados | `has_skill` | Task → Skill | `REQUIRES_SKILL` |
| Código asociado | `has_code` | Task → CodeNode | `REFERENCES` |
| Docs asociados | `has_doc` | Task → Document | — |
| Patterns asociados | `has_pattern` | Task → Pattern | — |
| Dependencias | `depends_on` | Task → Task | — |

### 9.2 Labels canónicos de nodos

| Label | Propiedades requeridas | Search property |
|-------|----------------------|-----------------|
| `CodeNode` | `path`, `language`, `summary` | `name` |
| `Fact` | `content`, `source` | `content` |
| `Project` | `name`, `path` | `name` |
| `Document` | `topic_key`, `doc_type`, `content` | `content` |
| `Task` | `project_id`, `name`, `description`, `status` | `name` |
| `Skill`, `Pattern`, `Library`, `Spec`, `Design`, `Decision`, `Review` | (sin cambios) | `name` |

### 9.3 Nombres de métodos MCP (firma canónica)

| Tool | Firma | Retorno |
|------|-------|---------|
| `search_facts_tool` | `(query: str, limit: int = 10) -> str` | JSON string |
| `search_code_tool` | `(query: str, limit: int = 10) -> str` | JSON string |
| `search_skills_tool` | `(query: str, limit: int = 10) -> str` | JSON string |
| `task_context_tool` | `(id: int) -> str` | JSON string con 7 secciones |
| `save_to_helix_tool` | `(label: str, properties: dict) -> str` | JSON string |
| `link_to_project_tool` | `(project_id: int, target_label: str, target_id: int, edge_label: str, properties: dict\|None) -> str` | JSON string |
| `find_project_tool` | `(name: str) -> str` | JSON string |

---

## 10. Referencias

- `openspec/specs/spec-fix-mcp-tools-bugs.md` — Especificación técnica de los 8 bugs
- `openspec/specs/spec-acceptance-criteria-tracking.md` — Especificación de tracking de acceptance criteria (relación con task_context y search_facts)
- `openspec/proposals/fix-mcp-tools-bugs.md` — Propuesta original con alcance e intento
- Archivos canónicos: `internal/opencode/mcptools/*.py`
- Archivos a sincronizar: `mcp-tools/*.py`
