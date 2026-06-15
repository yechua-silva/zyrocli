# AGENT.md — Cambios Mínimos para Integración HelixDB

Solo se modifican las líneas donde se referencia Engram/Graphify. El flujo de 4 macro-fases NO cambia.

---

## Línea 13: Stack

```diff
- OpenCode MCP (engram, graphify)
+ HelixDB (Rust v3.0.5) — grafo+vector+texto + gRPC + Python PydanticAI Harness
```

## Línea 52: F1 — Hand-off

```diff
- SaveState(engram, topic: "zyro/{project}/handoff")
+ SaveProject(helix, project) — crea nodo Project en HelixDB con tecnologías detectadas
```

## Línea 65: F2 — Contextualización

```diff
- graphify mapea estructura
+ CodeEntity nodes en HelixDB mapean estructura
```

## Línea 89: F3 — Testing

```diff
- genera diff con graphify
+ genera diff con HelixDB graph traversal
```

## Línea 94-96: F4 — Archivo

```diff
- Archivo (Engram) + Automática (lint/build) + Revisión Final (opcional)
- Archivo: mem_save a Engram con topic_key zyro/{project}/archive-report
+ Archivo (HelixDB) + Automática (lint/build) + Revisión Final (opcional)
+ Archivo: helix.SaveArchive(project, report) — guarda en HelixDB como nodo Artifact
```

## Línea 126: Patrones Go

```diff
- Engram MCP: mem_save y mem_search via MCP client para persistencia
+ HelixDB Go SDK: client.Exec() para persistencia y búsqueda
```

---

## Resumen

| Línea | Archivo | Cambio |
|-------|---------|--------|
| 13 | AGENT.md | Stack: reemplazar engram+graphify por HelixDB |
| 52 | AGENT.md | F1: SaveState(engram) → SaveProject(helix) |
| 65 | AGENT.md | F2: graphify mapea → HelixDB mapea |
| 89 | AGENT.md | F3: graphify diff → HelixDB traversal |
| 94-96 | AGENT.md | F4: mem_save → helix.SaveArchive() |
| 126 | AGENT.md | Patrón: Engram MCP → HelixDB Go SDK |

**Total: 6 cambios, ~15 líneas modificadas en AGENT.md.**
**Cero cambios en el flujo de fases, scheduler, o subagentes.**
