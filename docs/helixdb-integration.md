# Integración de HelixDB — ZyroAgentCLI

**Fecha**: 2026-06-14
**Propósito**: Reemplazar Engram + Graphify por HelixDB (Rust) como storage unificado.
**NO modifica**: El flujo de 4 macro-fases (F1→F4), el scheduler Go, ni los comandos CLI existentes.

---

## Stack Modificado

```
ANTES:                          DESPUÉS:
OpenCode MCP (engram, graphify) → HelixDB (Rust v3.0.5) + gRPC + Python Harness
```

El stack actualizado es:
- **Go 1.26+** — Orquestador, scheduler, CLI
- **HelixDB (Rust v3.0.5)** — grafo + vector (ANN) + texto (BM25) → reemplaza Engram + Graphify
- **gRPC (connectrpc.com/connect)** — comunicación Go ↔ Python
- **Python PydanticAI Harness** — tools, MCP, sandbox
- **skills.sh** — descubrimiento de skills
- **Context (Neuledge)** + GitMCP — documentación de librerías

---

## ¿Qué Reemplaza HelixDB?

| Componente Actual | Reemplazado Por | Detalle Técnico |
|------------------|----------------|-----------------|
| **Engram** (MCP memory) | HelixDB nodos + vectores | Observaciones → nodos `Observation` con embedding; topic_key → equality index; semantic search → vector search ANN |
| **Graphify** (JSON graph) | HelixDB grafo navegable | Nodos AST → `CodeEntity` nodes; edges `CALLS`/`DECLARES`; graph.json → queries en vivo sobre el grafo |
| **YAML configs** | HelixDB nodos | `.zyro/conventions.yaml` → nodos `Convention`; state → `Project` node |
| **mem_save/mem_search** | Go SDK `client.Exec()` | SDK oficial: `github.com/helixdb/helix-db/sdks/go` |

**NO se reemplaza**: El scheduler Go, las 4 macro-fases, los comandos CLI init/run/doc, los subagentes SDD.

---

## Las 4 Macro-Fases (sin cambios en el flujo)

### F1: Planificación
Hand-off → Exploración (Python) + Skill Advisor (Go) → [VALIDACIÓN HUMANA]

```
ANTES:  SaveState(engram, topic: "zyro/{project}/handoff")
DESPUÉS: helixClient.SaveProject(ctx, project) + helixClient.SaveSkills(ctx, skills)
```

Lo que se guarda en HelixDB:
- Nodo `Project` con tecnologías detectadas
- Nodos `Capability` para skills validadas
- Edge `Project ──USES──→ Technology`
- Edge `Project ──HAS_CHANGE──→ Change`

**No cambia**: handoff parse, skill advisor scoring, validación humana.

### F2: Especificación
Contextualización → Especificación C-I-O → [VALIDACIÓN HUMANA]

```
ANTES:  compile.go → EngramEntry (topic key + markdown)
DESPUÉS: compile.go → helixClient.SaveArtifact(ctx, artifact)
```

Lo que se guarda en HelixDB:
- Nodo `Change` con estado actualizado
- Nodos `Artifact` para specs, designs, ADRs
- Nodos `TechnicalDecision` con embedding para búsqueda semántica
- Edge `Change ──HAS_ARTIFACT──→ Artifact`
- Edge `Project ──HAS_DECISION──→ TechnicalDecision`

**No cambia**: context MCP bridge, C-I-O DSL, generación AGENT.md, validación humana.

### F3: Implementación
Aplicación (paralelo por tarea) → Testing de Contratos → loop hasta pasar → [VALIDACIÓN HUMANA]

```
ANTES:  test/report.go genera diff con graphify JSON
DESPUÉS: test/report.go genera diff con consulta en vivo a HelixDB
```

Lo que se guarda en HelixDB:
- Nodos `CodeEntity` para funciones/tipos implementados
- Nodos `Capability` con status "implemented"
- Edge `CodeEntity ──IMPLEMENTS──→ Capability`

**No cambia**: apply runner, contract testing given/when/then, loop max_loops, validación humana.

### F4: Cierre
Archivo + Automática (lint/build) + Revisión Final (opcional)

```
ANTES:  mem_save a Engram topic_key "zyro/{project}/archive-report"
DESPUÉS: helixClient.SaveArchive(ctx, project, report)
```

Lo que se guarda en HelixDB:
- Nodo `Artifact` con reporte de archive
- Edge `Change ──HAS_ARTIFACT──→ Artifact` (archivo final)
- Actualización de `Change.status = "archived"`

**No cambia**: lint/build final, doc sync, revisión opcional.

---

## Diagrama de Arquitectura

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        TERMINAL                                        │
│  zyrocli init | run | doc sync | db start | project add | search      │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────────┐
│                        ZYROCLI (Go)                                     │
│                                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐   │
│  │ cmd/     │  │ scheduler│  │ doc/     │  │ db/helix/ (NUEVO)    │   │
│  │ zyrocli/ │  │ F1→F4   │  │ sync     │  │ client.go            │   │
│  │ main.go  │  │ state    │  │ index    │  │ schema.go            │   │
│  │ init.go  │  │ machine  │  │ export   │  │ project.go           │   │
│  │ run.go   │  │          │  │ search   │  │ decision.go          │   │
│  └──────────┘  └──────────┘  └──────────┘  │ search.go            │   │
│                                              │ detect.go            │   │
│  ┌─────────────────────────────────────┐    └──────────┬───────────┘   │
│  │ internal/ (EXISTENTE, sin cambios)  │               │               │
│  │ handoff  scaffold  skilladvisor     │    ┌──────────▼───────────┐   │
│  │ context  spec      apply           │    │ grpc/ (NUEVO)        │   │
│  │ investigation  planning  test      │    │ server + client      │   │
│  └─────────────────────────────────────┘    └──────────┬───────────┘   │
└─────────────────────────────────┬─────────────────────┼───────────────┘
                                  │                     │
                     ┌────────────┼─────────────────────┼──────┐
                     │  HTTP:8500 │      gRPC unix      │      │
                     ▼            ▼                      ▼      │
          ┌──────────────────┐  ┌────────────────────────────┐ │
          │  HELIXDB (Rust)  │  │  PYTHON HARNESS            │ │
          │  v3.0.5          │◄─┤  (PydanticAI + gRPC)      │ │
          │                  │  │                            │ │
          │  ┌────────────┐  │  │  ┌────────┐ ┌──────────┐  │ │
          │  │  Graph     │  │  │  │gRPC    │ │Pydantic  │  │ │
          │  │  Nodes+    │  │  │  │Server  │ │AI Agents │  │ │
          │  │  Edges     │  │  │  └────────┘ └──────────┘  │ │
          │  ├────────────┤  │  └────────────────────────────┘ │
          │  │  Vector    │  │              ▲                  │
          │  │  ANN Index │  │              │ MCP stdio        │
          │  ├────────────┤  │              ▼                  │
          │  │  Text BM25 │  │    ┌────────────────────┐      │
          │  └────────────┘  │    │   OPENCODE AGENT    │      │
          └──────────────────┘    └────────────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### Protocolos de Comunicación

| Desde | Hacia | Protocolo | Transporte |
|-------|-------|-----------|------------|
| Go (zyrocli) | HelixDB | HTTP REST | `localhost:8500` |
| Go (zyrocli) | Python Harness | gRPC (connectrpc) | Unix socket `/tmp/zyro.sock` |
| OpenCode | Python Harness | MCP stdio | stdin/stdout |

---

## Esquema Universal de Datos

### Nodos Globales (cross-project)

```
Ecosystem     {name, description, embedding}
Technology    {name, category(language|database|framework|tool|protocol), version, description, embedding}
Pattern       {name, category(architectural|design|testing), description, embedding}
IndustryDomain{name, description, embedding}
Convention    {name, scope(global|project), rule, embedding}
```

### Nodos por Proyecto

```
Project           {name, root, language, module, description, status, embedding, topic_key}
TechnicalDecision {title, context, decision, consequences, status, embedding, topic_key}
Capability        {name, description, status(implemented|planned|deprecated), embedding}
Change            {change_id, title, status, embedding}
Artifact          {artifact_id, type(spec|design|proposal|adr|doc), title, content, embedding, topic_key}
CodeEntity        {type(function|type|file|module), name, path, language, signature, embedding}
```

### Edges

```
Project             ──USES──→         Technology
Project             ──FOLLOWS──→      Pattern
Project             ──HAS_DECISION──→ TechnicalDecision
Project             ──HAS_CHANGE──→   Change
Project             ──BELONGS_TO──→   IndustryDomain
Project             ──EMPLOYS──→      Convention
TechnicalDecision   ──REFERENCES──→   Technology
TechnicalDecision   ──SUPERSEDES──→   TechnicalDecision
Change              ──HAS_ARTIFACT──→ Artifact
Change              ──HAS_CAPABILITY──→ Capability
Technology          ──COMPATIBLE_WITH──→ Technology
Technology          ──ALTERNATIVE_TO──→ Technology
CodeEntity          ──CALLS──→         CodeEntity
CodeEntity          ──IMPLEMENTS──→    Capability
Artifact            ──REFERENCES──→    Technology
```

---

## Índices en HelixDB

| Tipo | Cantidad | Propósito |
|------|----------|-----------|
| **Equality** | 22 | Fast path: lookup por topic_key, name, status, path |
| **Vector ANN** | 8 | Búsqueda semántica: decisiones, proyectos, tecnologías, código |
| **Texto BM25** | 4 | Full-text search: contenido de specs, decisiones, titles |

---

## Nuevos Comandos CLI

```bash
# Gestión de HelixDB
zyrocli db init           # Crear schema + índices (idempotente)
zyrocli db start          # docker run helixdb/helix-db:v3.0.5
zyrocli db stop           # docker stop
zyrocli db status         # Health check + node/edge count

# Gestión de proyectos
zyrocli project add .     # Escanear proyecto actual + registrar en grafo
zyrocli project status    # Mostrar contexto del proyecto desde HelixDB
zyrocli project similar   # Proyectos similares (vector search)

# Búsqueda global
zyrocli search "consulta" # Búsqueda semántica en TODOS los proyectos

# Decisiones
zyrocli decision add      # Registrar decisión técnica en HelixDB
  --title "Usar HelixDB"
  --context "Storage fragmentation"
  --decision "HelixDB unified graph+vector"
```

---

## Detección Automática de Tecnologías

Cuando se ejecuta `zyrocli project add .`, se escanean estos archivos:

| Archivo | Tecnología Detectada |
|---------|---------------------|
| `go.mod` | Go + dependencias (cobra→CLI, gin→web, etc.) |
| `Cargo.toml` | Rust + crates |
| `package.json` | Node/TypeScript + librerías |
| `pyproject.toml` | Python + frameworks |
| `*.proto` | gRPC / Protobuf |
| `Dockerfile` | Docker |
| `.github/` | GitHub Actions |
| `Makefile` | Make |
| `*.rs` | Rust |
| `*.ex` / `*.exs` | Elixir |
| `docker-compose.yml` | Docker Compose |

---

## Ejemplo de Integración en Go

```go
import helix "github.com/helixdb/helix-db/sdks/go"

// En vez de: mem_save(engram, topic, content)
// Ahora:    helixClient.SaveDecision(ctx, input)

client, _ := helix.NewClient("http://localhost:8500")
client.Exec(ctx, helix.WriteQuery("save_decision").
    VarAs("decision",
        helix.G().AddN("TechnicalDecision", helix.Props{
            helix.Prop("title", "Usar HelixDB"),
            helix.Prop("context", "Engram + Graphify fragmentados"),
            helix.Prop("decision", "HelixDB unifica storage"),
            helix.Prop("embedding", embedding),
        }),
    ).
    Returning(), nil)
```

---

## Resumen de Cambios

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `AGENT.md` | Stack: engram→HelixDB; F1/F4 storage | ~3 líneas |
| `internal/spec/compile.go` | `EngramEntry` → guardar en HelixDB | ~10 líneas |
| `internal/doc/index.go` | Topic keys → HelixDB queries | ~5 líneas |
| `internal/test/report.go` | Graphify diff → HelixDB traversal | ~15 líneas |
| `.zyro/conventions.yaml` | Topic keys → HelixDB refs | ~5 líneas |
| **NUEVO**: `internal/db/helix/` | Cliente HelixDB + schema + CRUD | ~1,200 líneas |
| **NUEVO**: `internal/grpc/` | Servidor connectrpc | ~500 líneas |

**NO se modifican**: `cmd/zyrocli/main.go`, `cmd/zyrocli/init.go`, `cmd/zyrocli/run.go`,
`internal/scheduler/scheduler.go`, `internal/handoff/`, `internal/scaffold/`,
`internal/skilladvisor/`, `internal/context/bridge.go`, `internal/apply/runner.go`,
`internal/test/contracts.go`, `internal/investigation/`, `internal/planning/`.
