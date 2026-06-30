# Propuesta: Fix MCP Tools Bugs — helix-integration

## Intento

Las MCP tools de `helix-integration` (Python) tienen **8 bugs confirmados** que las hacen parcial o totalmente inoperables. El más crítico es que `text_search()` hardcodea `property: "name"`, mientras que nodos como `Fact` almacenan su texto en `property: "content"`, resultando en 0 resultados siempre. Adicionalmente hay inconsistencias de labels, edge labels divergentes entre `mcp-tools/` y `internal/opencode/mcptools/`, un `runner.py` roto, y datos insuficientes en los retornos.

Este cambio apunta a que las MCP tools funcionen correctamente: que `search_facts`, `search_code`, `search_skills`, `task_context` y las tools de escritura (`save_to_helix`, `link_to_project`, `find_project`) devuelvan datos reales de HelixDB con labels, edge labels y properties correctos.

## Alcance

### Incluye

| # | Bug | Severidad | Archivos afectados |
|---|-----|-----------|--------------------|
| 1 | `text_search` hardcodea `property: "name"` en vez de ser configurable | CRITICAL | `helix_client.py` (ambas copias) |
| 2 | Label mismatch: `search_code` usa `CodeNode`, `helix_write` valida `CodeModule` | HIGH | `helix_write.py` (internal), `search_code.py` (ambas) |
| 3 | `search_facts` busca por `name` en vez de `content` | HIGH | `search_facts.py` (ambas) |
| 4 | `REQUIRED_FIELDS` incompleto: faltan `Fact`, `Project`, `CodeNode`, `Document` | MEDIUM | `helix_write.py` (internal) |
| 5 | `text_search` retorna solo IDs sin propiedades del nodo | MEDIUM | `helix_client.py` (ambas) |
| 6 | Falsy ID bug: `p.get("$id") or p.get("id")` falla si ID=0 | MEDIUM | `helix_client.py` (ambas) |
| 7 | `runner.py` del proyecto raíz importa funciones que no existen | CRITICAL | `mcp-tools/runner.py` |
| 8 | Edge labels inconsistentes entre global y proyecto | HIGH | `task_context.py` (ambas) |

### Excluye

- Migrar las MCP tools a Go (siguen en Python, que es su hogar actual)
- Reescribir el MCP server en Go (`internal/mcp/`) — eso es Fase 4 y está en otro cambio
- Agregar nuevas MCP tools (solo se corrigen las existentes)
- Refactor mayor de `helix_client.py` (la estructura actual se mantiene)
- Tests unitarios (se dejan para un cambio posterior de testing)
- Integración con OpenCode o cambios en `opencode.json`

## Enfoque técnico

### Bug #1: `text_search` — hacer `property` configurable

**Problema**: `TextSearchNodes` en `helix_client.py` línea 226 tiene `"property": "name"` hardcodeado. Los Facts, Skills, Patterns y otros labels pueden usar `content`, `description`, etc.

**Solución**:
- Agregar parámetro `property: str = "name"` a `text_search()` (default "name" preserva compatibilidad donde funciona)
- Pasar ese valor dinámicamente al step `TextSearchNodes`
- Cada `search_*.py` pasa el property correcto según su label:
  - `search_facts` → `property="content"`
  - `search_skills` → `property="name"` (Skills sí usan `name`)
  - `search_code` → `property="name"` (CodeNode usa `name`)
  - `find_project_tool` → `property="name"` (Project usa `name`)

**Archivos**: `helix_client.py` (ambas copias: `mcp-tools/` e `internal/opencode/mcptools/`)

### Bug #2: Unificar label `CodeNode` como canónico

**Problema**: `search_code.py` usa label `"CodeNode"` para text search. `helix_write.py` (internal) valida `"CodeModule"` en `REQUIRED_FIELDS`. `capabilities.py` lista `"CodeNode"` como label permitido. No hay consistencia.

**Solución**:
- Definir `"CodeNode"` como label canónico en todo el sistema
- Cambiar `REQUIRED_FIELDS["CodeModule"]` → `REQUIRED_FIELDS["CodeNode"]` en `helix_write.py` (internal)
- Los campos requeridos para `CodeNode`: `["path", "language", "summary"]` (se mantienen)

**Archivos**: `helix_write.py` (internal)

### Bug #3: `search_facts` — buscar por `content` en vez de `name`

**Problema**: `search_facts()` llama a `client.text_search("Fact", query, limit)` que internamente hardcodea `property: "name"`. Los Facts guardan su texto en el campo `content`.

**Solución**:
- Pasar `property="content"` explícitamente desde `search_facts()`
- Esto funciona gracias al fix del Bug #1

**Archivos**: `search_facts.py` (ambas copias)

### Bug #4: Completar `REQUIRED_FIELDS`

**Problema**: Faltan validaciones para labels que ya existen y se usan: `Fact`, `Project`, `CodeNode`, `Document`.

**Solución**:
- Agregar entradas en `REQUIRED_FIELDS` en `helix_write.py` (internal):
  - `"Fact": ["content", "source"]`
  - `"Project": ["name", "path"]`
  - `"CodeNode": ["path", "language", "summary"]` (reemplaza `CodeModule`)
  - `"Document": ["topic_key", "doc_type", "content"]`

**Archivos**: `helix_write.py` (internal)

### Bug #5: `text_search` — incluir propiedades en el retorno

**Problema**: Líneas 236-238 retornan `[{"id": i} for i in ids]`. El agente recibe IDs opacos sin contexto.

**Solución**:
- Agregar un step `ProjectReturn` después de `TextSearchNodes` para traer propiedades del nodo
- El step adicional lista los fields a retornar: `["$id", "name", "content", "language", "path", "summary", "description", "source"]` (lo que exista en cada nodo)
- La respuesta de HelixDB v3 devuelve `{name: {"properties": [{"$id": 1, "name": "...", ...}]}}` en vez de solo IDs
- `_get_ids()` ya maneja este formato; se adapta `text_search()` para retornar dicts completos en vez de solo `{"id": i}`

**Formato de retorno nuevo**:
```python
# Antes:
[{"id": 1}, {"id": 2}]

# Después:
[{"id": 1, "name": "...", "content": "...", ...}, {"id": 2, ...}]
```

**Archivos**: `helix_client.py` (ambas copias)

### Bug #6: Falsy ID bug

**Problema**: Línea 83: `p.get("$id") or p.get("id")`. Si `$id` es 0 (falsy en Python), cae a `p.get("id")` aunque el ID real sea 0.

**Solución**:
- Reemplazar con: `p.get("$id") if p.get("$id") is not None else p.get("id")`
- O usar: `p.get("$id") if "$id" in p else p.get("id")`

**Archivos**: `helix_client.py` (ambas copias)

### Bug #7: Arreglar/eliminar `runner.py` del proyecto raíz

**Problema**: `mcp-tools/runner.py` importa `search_code_tool`, `search_facts_tool` etc., pero esos nombres no existen en ese directorio — solo existen en `internal/opencode/mcptools/`. Además `helix_write.py` del proyecto raíz es un stub vacío del que intenta importar `save_to_helix_tool`, `link_to_project_tool`, `find_project_tool`.

**Solución**:
- Opción recomendada: **eliminar** `mcp-tools/runner.py` por completo, ya que:
  - El runner real y funcional está en `internal/opencode/mcptools/runner.py`
  - El proyecto raíz (`mcp-tools/`) contiene funciones planas (`search_code()`, `search_facts()`) que NO son MCP tools — son helpers internos
  - Mantener el runner roto crea confusión y falsas expectativas
- Alternativamente: convertirlo en un re-export que delegue a `internal/opencode/mcptools/`

**Archivos**: `mcp-tools/runner.py`

### Bug #8: Unificar edge labels

**Problema**: Dos versiones de `task_context.py` usan edge labels distintos para el mismo concepto:

| Concepto | `internal/opencode/mcptools/` (global) | `mcp-tools/` (proyecto) |
|----------|----------------------------------------|--------------------------|
| Skills asociados | `has_skill` | `REQUIRES_SKILL` |
| Código asociado | `has_code` | `REFERENCES` |
| Docs asociados | `has_doc` | — |
| Patterns asociados | `has_pattern` | — |
| Dependencias | `depends_on` | — |

**Solución**:
- Adoptar el estilo **lowercase con underscores** usado en `internal/opencode/mcptools/task_context.py` como canónico (`has_skill`, `has_code`, `has_doc`, `has_pattern`, `depends_on`)
- Actualizar `mcp-tools/task_context.py` para usar los mismos edge labels
- Registrar esta decisión como edge label convention en el proyecto

**Archivos**: `mcp-tools/task_context.py`

## Evidencia

Toda la evidencia proviene de investigación directa con curl a HelixDB y revisión de código:

1. **Bug #1**: `curl` query demostró que `TextSearchNodes(Fact, property="content")` retorna resultados (nodo #3025, score 4.65), mientras `TextSearchNodes(Fact, property="name")` retorna 0. Sin `property`, HelixDB rejects con "missing field `property`".

2. **Bug #2**: Lectura directa de tres archivos revela tres labels distintos para el mismo concepto: `CodeNode` (search_code.py:11), `CodeModule` (helix_write.py:19), `CodeNode` (capabilities.py:14).

3. **Bug #3**: Consecuencia directa de Bug #1. `search_facts.py:10` llama `client.text_search("Fact", query, limit)` sin especificar `property`. Facts tienen campo `content` no `name`.

4. **Bug #4**: `helix_write.py` internal lista 9 labels en `REQUIRED_FIELDS`. `capabilities.py` lista 10 labels permitidos. Faltan: `Fact`, `Project`, `CodeNode`, `Document`.

5. **Bug #5**: `text_search()` en helix_client.py:236-238 itera IDs y retorna `[{"id": i}]`. Sin `ProjectReturn` step, HelixDB no incluye properties.

6. **Bug #6**: `helix_client.py:83` — patrón clásico de falsy en Python. `$id=0` es improbable pero el patrón es incorrecto.

7. **Bug #7**: `runner.py:15-19` importa `search_code_tool`, `search_facts_tool`, `search_skills_tool`, `task_context_tool`, `save_to_helix_tool`, `link_to_project_tool`, `find_project_tool` — ninguno existe en `mcp-tools/`. `helix_write.py` en `mcp-tools/` es un stub deprecado de 22 líneas sin esas funciones.

8. **Bug #8**: `internal/opencode/mcptools/task_context.py:42-49` usa `has_skill`, `has_code`, `has_doc`, `has_pattern`, `depends_on`. `mcp-tools/task_context.py:20-24` usa `REQUIRES_SKILL`, `REFERENCES`. Misma función, aristas distintas.

## Riesgos

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|-------------|---------|------------|
| Cambiar `property` default rompe código que depende de `text_search("Fact", ...)` sin property explícito | Media | Alto | Mantener default `"name"` consistente con comportamiento actual; solo `search_facts` pasa `"content"` |
| `ProjectReturn` step puede traer data sensible si no se filtra bien | Baja | Medio | Especificar fields explícitos en vez de traer todo. Los datos en HelixDB son conocimiento técnico, no secretos. |
| Eliminar `runner.py` rompe scripts que lo referencian | Media | Medio | Verificar si hay referencias externas (opencode.json, docs). Si existen, actualizar. Si no, eliminación segura. |
| Unificar edge labels invalida edges existentes en BD | Alta | Alto | Los edges en HelixDB se crearon con labels viejos. Solo se cambia el código que *lee* edges; se debe hacer migrate de edges existentes O soportar ambos labels temporalmente con fallback. |
| Dos copias de `helix_client.py` pueden desincronizarse | Alta | Medio | Espejo manual. Idealmente extraer shared lib, pero fuera de alcance. Documentar que ambas copias deben mantenerse iguales. |

### Riesgo especial: Edge labels existentes en BD

El Bug #8 es engañosamente simple. No basta con cambiar las strings en `task_context.py` — los edges ya creados en HelixDB tienen los labels viejos. Si cambiamos el código para buscar `has_skill` pero los edges se crearon como `REQUIRES_SKILL`, las queries retornarán 0 resultados.

**Estrategia recomendada**:
1. Cambiar el código para que **intente el label canónico primero**, y si retorna vacío, **haga fallback al label legacy**
2. En paralelo, ejecutar un script de migrate en HelixDB que renombre edges viejos al nuevo label
3. Documentar la convención de edge labels (`lowercase_with_underscores`) como ADR

## Esfuerzo estimado

**Medium** (~150-200 líneas totales de cambio, 8 archivos modificados, complejidad baja-media).

Desglose:
- Bug #1 (property configurable): ~5 líneas en helix_client.py
- Bug #2 (CodeNode canónico): ~2 líneas en helix_write.py
- Bug #3 (search_facts content): ~1 línea por archivo (2 archivos)
- Bug #4 (REQUIRED_FIELDS): ~8 líneas en helix_write.py
- Bug #5 (ProjectReturn): ~8 líneas en helix_client.py + ~2 en _get_ids
- Bug #6 (falsy ID): ~1 línea en helix_client.py
- Bug #7 (runner.py): ~1 archivo eliminado o ~10 líneas si se convierte en re-export
- Bug #8 (edge labels): ~4 líneas en mcp-tools/task_context.py + migrate script

El esfuerzo principal no está en escribir código sino en verificar que los cambios no rompan la integración existente y en ejecutar el migrate de edge labels.

## Criterios de éxito

- [ ] `search_facts("pattern")` retorna Facts con contenido (no solo IDs)
- [ ] `search_code("auth")` retorna CodeNodes con propiedades
- [ ] `search_skills("react")` retorna Skills con nombre y metadata
- [ ] `task_context(1)` retorna contexto con skills, code, docs, patterns usando edge labels consistentes
- [ ] `save_to_helix` valida campos requeridos para `Fact`, `Project`, `CodeNode`, `Document`
- [ ] `helix_client.text_search()` acepta `property` como parámetro
- [ ] `runner.py` del proyecto raíz no rompe imports (eliminado o convertido)
- [ ] Falsy ID no causa bug si `$id` fuera 0
- [ ] Edges legacy (`REQUIRES_SKILL`, `REFERENCES`) siguen siendo encontrados via fallback
- [ ] Ambas copias de `helix_client.py` (`mcp-tools/` e `internal/opencode/mcptools/`) quedan sincronizadas
