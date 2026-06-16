# Investigación: "Engram" como Sistema de Memoria Causal sobre HelixDB

**Fecha**: 2026-06-15
**Proyecto**: ZyroAgentCLI
**Contexto**: Investigación técnica para CONSTRUIR un sistema de memoria causal persistente para agentes de IA sobre HelixDB.
**Aclaración**: Este documento NO trata sobre el MCP server `engram-mcp-server` (keggan-std/Engram). "Engram" aquí se usa como concepto genérico — la traza física de una memoria en el cerebro — y se refiere al sistema CUSTOM que Zyro construye sobre HelixDB.
**Estado**: Propuesta de arquitectura

---

## Índice

1. [¿Qué es "Engram" como Concepto?](#1-qué-es-engram-como-concepto)
2. [El Problema: Zwischenlösung (Sesión en Blanco)](#2-el-problema-zwischenlösung-sesión-en-blanco)
3. [Diferencias con el Engram MCP Server Existente](#3-diferencias-con-el-engram-mcp-server-existente)
4. [Arquitectura de Memoria Causal Propuesta](#4-arquitectura-de-memoria-causal-propuesta)
5. [Implementación sobre HelixDB](#5-implementación-sobre-helixdb)
6. [Agente Extractor de Hechos](#6-agente-extractor-de-hechos)
7. [Flujo de Integración en el Pipeline Zyro](#7-flujo-de-integración-en-el-pipeline-zyro)
8. [Comparativa de Enfoques](#8-comparativa-de-enfoques)
9. [Código de Referencia](#9-código-de-referencia)
10. [Recomendaciones](#10-recomendaciones)
11. [Referencias](#11-referencias)

---

## 1. ¿Qué es "Engram" como Concepto?

### Definición Neurocientífica

**Engram** (del alemán *Engramm*, acuñado por Richard Semon en 1904) es el término neurocientífico para la **traza física de una memoria** — el cambio bioquímico y estructural en el cerebro que codifica una experiencia. En neurociencia moderna, el engram es un conjunto de neuronas que se activan juntas para evocar un recuerdo específico.

### En el Contexto de Zyro

En ZyroAgentCLI, "engram" es un **sistema de memoria causal** que:

- **Captura** hechos, decisiones, errores, preferencias, patrones y dependencias durante la ejecución del agente
- **Almacena** estos hechos como nodos en un grafo con aristas causales (qué causó qué, qué contradice qué, qué soporta qué)
- **Recupera** memoria relevante antes de cada fase del pipeline, inyectándola en el prompt del agente
- **Resuelve** contradicciones automáticamente cuando se detectan hechos opuestos
- **Olvida** gradualmente usando una curva de decaimiento temporal (inspirada en la curva de olvido de Ebbinghaus)

### Principios de Diseño

| Principio | Descripción |
|-----------|-------------|
| **Causal** | Cada hecho registra su contexto causal (qué lo originó, qué consecuencias tuvo) |
| **Estructural** | Los hechos no son texto plano — son nodos en un grafo con relaciones semánticas |
| **Persistente** | La memoria sobrevive reinicios, sesiones, y fases del pipeline |
| **Recuperable** | Búsqueda híbrida (vector + BM25 + grafo) para encontrar el hecho relevante |
| **Consistente** | Detección y resolución de contradicciones entre hechos |
| **Evolving** | Los hechos tienen ciclos de vida: nacen, son referenciados, decaen, son olvidados |
| **No-determinista** | El sistema no fuerza consistencia absoluta — registra ambigüedad cuando existe |

---

## 2. El Problema: Zwischenlösung (Sesión en Blanco)

### El Síntoma

Cada vez que ZyroAgentCLI inicia una fase del pipeline (F0→F1→F2→F3→F4), el agente comienza **sin contexto**:
- No sabe qué decisiones se tomaron en la fase anterior
- No sabe qué errores se cometieron y cómo se resolvieron
- No sabe qué prefirió el usuario (estilo, patrones, libraries)
- No sabe qué patrones funcionaron y cuáles no
- No sabe qué dependencias existen entre decisiones

### La Raíz

OpenCode/Claude Code no tiene memoria persistente entre sesiones de agente. Cada invocación es un "reset cognitivo". El agente puede repetir errores, contradecir decisiones previas, o perder aprendizaje valioso.

### La Solución: Capa de Memoria Causal

Una capa que:
1. En **pre-fase**: consulta la memoria relevante y la inyecta en el prompt del agente
2. Durante la fase: el agente puede consultar y escribir memoria vía herramientas MCP propias
3. En **post-fase**: extrae hechos nuevos de la conversación y los almacena en el grafo causal

---

## 3. Diferencias con el Engram MCP Server Existente

Existe un producto llamado Engram (`keggan-std/Engram`) que es un MCP server de memoria para agentes. **Zyro NO va a usar ese producto.** Las diferencias son fundamentales:

| Aspecto | Engram MCP Server (keggan-std) | Engram Zyro (propuesto) |
|---------|-------------------------------|------------------------|
| **Naturaleza** | Producto pre-construido (npm package) | Sistema CUSTOM construido sobre HelixDB |
| **Stack** | TypeScript + SQLite (WAL mode) | Go + HelixDB (grafo + vectores) |
| **Almacenamiento** | SQLite plano (tablas relacionales) | Grafo de propiedades con nodos + edges |
| **Búsqueda** | FTS5 (texto) | Híbrida: BM25 + Vector ANN + Traversal de grafo |
| **Modelo de datos** | Tablas: sesiones, decisiones, tareas, notas | Nodos: Fact, Session, Decision; Edges: CAUSED, PRECEDES, CONTRADICTS, SUPPORTS, REQUIRES |
| **Causalidad** | Implícita (por sesión/timestamp) | Explícita (aristas causales con tipo) |
| **Contradicciones** | No resuelve | Resolución automática con estrategias configurables |
| **Olvido** | No implementado | Curva de Ebbinghaus + decaimiento configurable |
| **Embeddings** | No tiene | Vector search semántico con embeddings externos |
| **Escalabilidad** | Limitada a SQLite (single-node) | Horizontal con HelixDB Cloud |
| **Multi-tenencia** | Por proyecto (directorio) | Nativa con `tenant_id` en índices vector/text |
| **Licencia** | MIT | Propietario (Zyro) |
| **Dependencia externa** | npm + Node.js + better-sqlite3 build | HelixDB (ya integrada en Zyro) |

### ¿Por Qué NO Usar el Engram MCP Server?

| Razón | Explicación |
|-------|-------------|
| **Ya tenemos HelixDB** | Zyro ya tiene HelixDB integrada. Agregar SQLite + Node.js es duplicar infraestructura |
| **Stack unificado** | Go + HelixDB es el stack objetivo de Zyro. TypeScript + SQLite añade complejidad operacional |
| **Grafo + vectores** | La memoria causal se beneficia del grafo (relaciones causales) y vectores (búsqueda semántica). SQLite no ofrece ni lo uno ni lo otro |
| **Control total** | Construir nuestro propio sistema permite adaptar el modelo de datos, las estrategias de resolución de contradicciones, y la curva de olvido exactamente a nuestras necesidades |
| **Sin dependencias frágiles** | El Engram MCP server depende de `better-sqlite3` (bindings nativos C++), Node.js, y mantenimiento externo |
| **Costo de integración** | Integrar el Engram MCP server requiere configurar MCP, mantener schema compatibility, y lidiar con divergencia de datos entre Engram y HelixDB |

---

## 4. Arquitectura de Memoria Causal Propuesta

### Visión General

```
┌──────────────────────────────────────────────────────────────────┐
│                      ZyroAgentCLI                                 │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────┐    ┌─────────────────────┐    ┌─────────────┐  │
│  │  Agente IA    │    │  Orquestador Go      │    │  Extractor  │  │
│  │  (OpenCode)   │◄──►│  (zyrocli agent)     │◄──►│  de Hechos  │  │
│  │               │    │                      │    │  (Python)   │  │
│  └──────┬───────┘    └──────────┬──────────┘    └──────┬──────┘  │
│         │                       │                       │         │
│         │  MCP Tools            │  Go SDK               │  API    │
│         ▼                       ▼                       ▼         │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │              HelixDB (localhost:6969)                         │ │
│  │  ┌────────────────────────────────────────────────────────┐  │ │
│  │  │  Memoria Causal (Engram)                                │  │ │
│  │  │  • Nodos Fact (decisión, error, preferencia, patrón)   │  │ │
│  │  │  • Edges causales (CAUSED, PRECEDES, CONTRADICTS...)  │  │ │
│  │  │  • Índice vectorial (búsqueda semántica)               │  │ │
│  │  │  • Índice BM25 (búsqueda textual)                      │  │ │
│  │  │  • Ciclo de vida: salience, decay, expiresAt          │  │ │
│  │  └────────────────────────────────────────────────────────┘  │ │
│  └──────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

### Componentes

#### 1. Nodos `Fact` — La Unidad de Memoria

Cada hecho capturado es un nodo en HelixDB con esta estructura:

```json
{
  "label": "Fact",
  "properties": {
    "fact_id": "uuid-v7",
    "type": "decision | error | preference | pattern | dependency | observation",
    "content": "Descripción textual del hecho",
    "embedding": [0.12, 0.85, -0.04, ...],
    "salience": 0.85,
    "confidence": 0.92,
    "source": "agent:F0 | user:input | extractor:llm",
    "created_at": "2026-06-15T10:30:00Z",
    "last_accessed_at": "2026-06-15T10:30:00Z",
    "access_count": 1,
    "decay_rate": 0.05,
    "expires_at": "2026-07-15T10:30:00Z",
    "is_active": true,
    "tenant_id": "proyecto-zyro",
    "phase": "F0 | F1 | F2 | F3 | F4",
    "metadata": {
      "model": "claude-sonnet-4",
      "tokens_used": 4500,
      "files_touched": ["internal/db/helix/helix.go"]
    }
  }
}
```

#### 2. Tipos de Hecho

| Tipo | Descripción | Ejemplo |
|------|-------------|---------|
| `decision` | Decisión arquitectónica o de diseño | "Usamos Go SDK oficial de HelixDB en lugar de cliente raw JSON" |
| `error` | Error encontrado y su resolución | "El cliente raw JSON devolvía 404 por type mismatch en $id" |
| `preference` | Preferencia explícita del usuario | "El usuario quiere SQLC para queries, no GORM" |
| `pattern` | Patrón de código identificado | "Usamos repository pattern con interfaces en internal/db/" |
| `dependency` | Dependencia entre decisiones o módulos | "Fase F2 depende de la migración a Go SDK oficial" |
| `observation` | Observación factual sin juicio de valor | "El proyecto tiene 3 archivos en internal/db/helix/ con lógica duplicada" |

#### 3. Aristas Causales — El Grafo de Memoria

| Edge Type | Significado | Ejemplo |
|-----------|-------------|---------|
| `CAUSED` | A causó directamente B | Decisión A → Error B |
| `PRECEDES` | A ocurrió antes que B | Fase F0 → Fase F1 |
| `CONTRADICTS` | A contradice a B | Pref "usa GORM" → Dec "usamos SQLC" |
| `SUPPORTS` | A soporta o refuerza B | Pattern repo → Dec usar interfaces |
| `REQUIRES` | A requiere B para ser válido | Dec migrar SDK → Dep Go SDK instalado |
| `DERIVES_FROM` | A se deriva o infiere de B | Pattern A → Observation B |
| `REFERENCES` | A referencia a B (relación débil) | Fact → CodeNode, Fact → Doc |

#### 4. Resolución de Contradicciones

Cuando dos hechos se contradicen (detectado por embedding similarity + tipo opuesto), el sistema:

1. **Marca ambos** como `in_conflict: true`
2. **Crea un edge** `CONTRADICTS` entre ellos
3. **Aplica estrategia configurable**:
   - `newest_wins` — el hecho más reciente tiene prioridad
   - `highest_confidence` — el hecho con mayor `confidence` prevalece
   - `user_override` — pregunta al usuario (vía MCP tool)
   - `keep_both` — registra la ambigüedad, ambos son válidos en contexto
4. **Expone la contradicción** en la consulta de memoria para que el agente decida

#### 5. Curva de Olvido (Decaimiento Temporal)

Inspirado en la curva de olvido de Ebbinghaus:

```
salience(t) = salience_0 * e^(-decay_rate * t)
```

- Cada hecho tiene `salience_0` (importancia inicial), `decay_rate` (velocidad de olvido), y `access_count`
- Cada vez que se accede al hecho, `salience` se refuerza: `salience += boost * (1 - salience)`
- Hechos con `salience < threshold` son marcados como `stale: true`
- Hechos con `expires_at < now` son marcados como `is_active: false`
- En consulta, los hechos se ordenan por `salience * freshness` donde `freshness = 1 - (now - created_at) / (expires_at - created_at)`

**Ejemplo**:
```python
# Configuración por defecto
DECAY_CONFIG = {
    "base_decay_rate": 0.05,       # Por día
    "access_boost": 0.3,           # Refuerzo al acceder
    "salience_threshold": 0.15,    # Mínimo para considerar recordado
    "max_salience": 1.0,
    "default_expiry_days": 90,     # Hechos sin access expiran en 90 días
}
```

---

## 5. Implementación sobre HelixDB

### 5.1 Schema HelixQL (HQL)

```sql
-- Schema de memoria causal para Engram Zyro

-- Crear índices para Facts
CREATE VECTOR INDEX IF NOT EXISTS idx_fact_embedding ON Fact(embedding) WITH tenant_id;
CREATE TEXT INDEX IF NOT EXISTS idx_fact_content ON Fact(content) WITH tenant_id;
CREATE EQUALITY INDEX IF NOT EXISTS idx_fact_type ON Fact(type);

-- Crear índices para edges causales
CREATE EQUALITY INDEX IF NOT EXISTS idx_edge_caused ON Fact edge CAUSED;
CREATE EQUALITY INDEX IF NOT EXISTS idx_edge_precedes ON Fact edge PRECEDES;
CREATE EQUALITY INDEX IF NOT EXISTS idx_edge_contradicts ON Fact edge CONTRADICTS;
CREATE EQUALITY INDEX IF NOT EXISTS idx_edge_supports ON Fact edge SUPPORTS;
CREATE EQUALITY INDEX IF NOT EXISTS idx_edge_requires ON Fact edge REQUIRES;
CREATE EQUALITY INDEX IF NOT EXISTS idx_edge_derives_from ON Fact edge DERIVES_FROM;
CREATE EQUALITY INDEX IF NOT EXISTS idx_edge_references ON Fact edge REFERENCES;
```

### 5.2 Go SDK — Operaciones de Memoria

El orquestador Go usa el SDK oficial de HelixDB (`github.com/helixdb/helix-db/sdks/go`) para todas las operaciones de memoria.

#### Guardar un Hecho

```go
import helix "github.com/helixdb/helix-db/sdks/go"

type FactInput struct {
    Type       string
    Content    string
    Embedding  []float32
    Salience   float64
    Confidence float64
    Source     string
    TenantID   string
    Phase      string
    Metadata   map[string]any
}

func (s *EngramStore) SaveFact(ctx context.Context, input FactInput) (int64, error) {
    q := helix.WriteQuery("save_fact")
    tenant := q.ParamString("tenant_id", input.TenantID)
    embedding := q.ParamArray("embedding", input.Embedding, helix.ParamTypeF32())
    metadata := q.ParamJSON("metadata", input.Metadata)
    now := q.ParamDateTime("now", time.Now())
    expiresAt := q.ParamDateTime("expires_at", time.Now().Add(90*24*time.Hour))

    return q.
        VarAs("fact",
            helix.G().AddN("Fact", helix.Props{
                helix.Prop("type", input.Type),
                helix.Prop("content", input.Content),
                helix.Prop("embedding", embedding),
                helix.Prop("salience", input.Salience),
                helix.Prop("confidence", input.Confidence),
                helix.Prop("source", input.Source),
                helix.Prop("created_at", now),
                helix.Prop("last_accessed_at", now),
                helix.Prop("access_count", 1),
                helix.Prop("decay_rate", 0.05),
                helix.Prop("expires_at", expiresAt),
                helix.Prop("is_active", true),
                helix.Prop("tenant_id", tenant),
                helix.Prop("phase", input.Phase),
                helix.Prop("metadata", metadata),
            }).
            Project(helix.ProjectPropAs("$id", "fact_id")),
        ).
        Returning("fact").
        ExecAndReturnID(ctx, s.client)
}
```

#### Crear Arista Causal

```go
func (s *EngramStore) AddCausalEdge(ctx context.Context, fromID, toID int64, edgeType string, props map[string]any) error {
    q := helix.WriteQuery("add_causal_edge")

    return q.
        VarAs("edge",
            helix.G().
                N(helix.NodeVarFromID(fromID)).
                AddE(edgeType, helix.NodeVarFromID(toID), helix.PropsFromMap(props)).
                Count(),
        ).
        Exec(ctx, s.client)
}
```

#### Consultar Memoria Relevante (Búsqueda Híbrida)

```go
type MemoryQuery struct {
    QueryText   string
    QueryVector []float32
    TenantID    string
    K           int64
    FactTypes   []string   // opcional: filtrar por tipo
    MinSalience float64    // opcional: mínimo de salience
    IncludeStale bool      // opcional: incluir facts stale
}

type MemoryResult struct {
    FactID    int64   `json:"fact_id"`
    Type      string  `json:"type"`
    Content   string  `json:"content"`
    Salience  float64 `json:"salience"`
    Score     float64 `json:"score"`      // RRF score
    VectorScore float64 `json:"vector_score,omitempty"`
    TextScore   float64 `json:"text_score,omitempty"`
}

func (s *EngramStore) RecallMemories(ctx context.Context, q MemoryQuery) ([]MemoryResult, error) {
    // 1. Vector search
    vecQuery := helix.ReadQuery("vector_recall").
        VarAs("vector_hits",
            helix.G().
                VectorSearchNodes("Fact", "embedding", q.QueryVector, q.K, q.TenantID).
                Where(helix.PredEq("is_active", true)).
                Project(
                    helix.ProjectPropAs("$id", "fact_id"),
                    helix.ProjectPropAs("$distance", "score"),
                    helix.ProjectProp("type"),
                    helix.ProjectProp("content"),
                    helix.ProjectProp("salience"),
                ),
        ).
        Returning("vector_hits")

    // 2. Text search (BM25)
    textQuery := helix.ReadQuery("text_recall").
        VarAs("text_hits",
            helix.G().
                TextSearchNodes("Fact", "content", q.QueryText, q.K, q.TenantID).
                Where(helix.PredEq("is_active", true)).
                Project(
                    helix.ProjectPropAs("$id", "fact_id"),
                    helix.ProjectPropAs("$distance", "score"),
                    helix.ProjectProp("type"),
                    helix.ProjectProp("content"),
                    helix.ProjectProp("salience"),
                ),
        ).
        Returning("text_hits")

    // 3. Fusion with RRF
    var vecResults, textResults []MemoryResult
    s.client.Exec(ctx, vecQuery, &vecResults)
    s.client.Exec(ctx, textQuery, &textResults)

    return fuseRRF(vecResults, textResults, q.K), nil
}

func fuseRRF(vec, txt []MemoryResult, k int64) []MemoryResult {
    scores := make(map[int64]float64)
    seen := make(map[int64]*MemoryResult)
    items := make(map[int64]MemoryResult)

    for i, r := range vec {
        scores[r.FactID] += 1.0 / (float64(k) + float64(i) + 1)
        items[r.FactID] = r
    }
    for i, r := range txt {
        scores[r.FactID] += 1.0 / (float64(k) + float64(i) + 1)
        items[r.FactID] = r
    }

    result := make([]MemoryResult, 0, len(scores))
    for id, score := range scores {
        item := items[id]
        item.Score = score
        result = append(result, item)
    }
    sort.Slice(result, func(i, j int) bool {
        return result[i].Score > result[j].Score
    })
    return result
}
```

#### Detectar y Resolver Contradicciones

```go
type ContradictionStrategy string

const (
    StrategyNewestWins       ContradictionStrategy = "newest_wins"
    StrategyHighestConfidence ContradictionStrategy = "highest_confidence"
    StrategyKeepBoth          ContradictionStrategy = "keep_both"
)

func (s *EngramStore) ResolveContradictions(ctx context.Context, factID int64, strategy ContradictionStrategy) error {
    // 1. Buscar facts similares semánticamente (embedding cerca) que sean de tipo opuesto
    similar := s.findSemanticConflicts(ctx, factID, 0.85)

    for _, conflict := range similar {
        // 2. Crear edge CONTRADICTS
        s.AddCausalEdge(ctx, factID, conflict.FactID, "CONTRADICTS", map[string]any{
            "detected_at": time.Now(),
            "strategy":    string(strategy),
        })

        // 3. Aplicar estrategia
        switch strategy {
        case StrategyNewestWins:
            // El más nuevo se marca como active, el otro como superseded
            if conflict.CreatedAt.After(s.getFactCreation(ctx, factID)) {
                s.setFactActive(ctx, factID, false, "superseded_by_conflict")
            } else {
                s.setFactActive(ctx, conflict.FactID, false, "superseded_by_conflict")
            }
        case StrategyHighestConfidence:
            // El de mayor confidence gana
            factConf := s.getFactConfidence(ctx, factID)
            confConf := s.getFactConfidence(ctx, conflict.FactID)
            if confConf > factConf {
                s.setFactActive(ctx, factID, false, "superseded_by_conflict")
            } else {
                s.setFactActive(ctx, conflict.FactID, false, "superseded_by_conflict")
            }
        case StrategyKeepBoth:
            // Marcar ambos como in_conflict pero mantener activos
            s.markConflict(ctx, factID, conflict.FactID)
        }
    }
    return nil
}
```

---

## 6. Agente Extractor de Hechos

### Propósito

El extractor de hechos es un agente Python que:
1. Toma el log de una conversación entre el usuario y el agente IA
2. Parsea e identifica hechos: decisiones, errores, preferencias, patrones, dependencias, observaciones
3. Calcula embeddings para cada hecho
4. Los envía al orquestador Go para almacenarlos en HelixDB

### Arquitectura

```
Log de conversación (JSONL / markdown)
    │
    ▼
┌──────────────────────────────────────┐
│  Extractor de Hechos (Python)         │
│                                      │
│  1. Pre-procesamiento                │
│     - Tokenización                   │
│     - Segmentación por turnos        │
│                                      │
│  2. Clasificación (LLM local)        │
│     - ¿Es un hecho? Si/No            │
│     - Tipo: decision/error/...       │
│     - Confianza                      │
│                                      │
│  3. Extracción de estructura         │
│     - Content (texto del hecho)       │
│     - Contexto causal                │
│     - Metadatos (archivos, fases)    │
│                                      │
│  4. Embedding                        │
│     - LLM local (Ollama/Optimized)   │
│     - text-embedding-3-small via API │
│                                      │
│  5. Envío a orquestador Go           │
│     - POST /api/v1/facts             │
└──────────────────────────────────────┘
    │
    ▼
Orquestador Go → HelixDB
```

### Implementación de Referencia

```python
"""
extractor.py — Agente extractor de hechos para memoria causal Zyro.

Uso:
    python extractor.py --input conversation.jsonl --tenant proyecto-zyro --phase F0

Dependencias:
    - ollama (para LLM local: llama3.2, qwen2.5, etc.)
    - openai (opcional, para text-embedding-3-small)
    - httpx (para comunicación con orquestador Go)
"""

import json
import uuid
import argparse
from datetime import datetime, timedelta
from typing import Optional
import httpx
from ollama import Client as OllamaClient

# ─── Configuración ───────────────────────────────────────────────────

FACT_TYPES = ["decision", "error", "preference", "pattern", "dependency", "observation"]

EXTRACTION_PROMPT = """Eres un extractor de hechos para un sistema de memoria causal.
Analiza la siguiente conversación entre un usuario y un agente de IA
e identifica TODOS los hechos relevantes.

Para cada hecho, extrae:
1. type: uno de [decision, error, preference, pattern, dependency, observation]
2. content: descripción clara y concisa del hecho
3. confidence: 0.0 a 1.0 (qué tan seguro estás de que esto es un hecho)
4. salience: 0.0 a 1.0 (qué tan importante es este hecho)
5. causal_context: qué causó este hecho (si es identificable)
6. metadata: {files, concepts, phase}

Conversación:
{conversation}

Responde SOLO con un JSON array de hechos. Si no hay hechos, responde [].
"""

# ─── Extractor ───────────────────────────────────────────────────────

class FactExtractor:
    def __init__(self, ollama_model: str = "llama3.2", openai_api_key: Optional[str] = None):
        self.ollama = OllamaClient()
        self.model = ollama_model
        self.openai_key = openai_api_key

    def extract_facts(self, conversation: str) -> list[dict]:
        """Extrae hechos de una conversación usando LLM local."""
        prompt = EXTRACTION_PROMPT.format(conversation=conversation[:8000])

        response = self.ollama.chat(
            model=self.model,
            messages=[{"role": "user", "content": prompt}],
            format="json",
        )

        try:
            facts = json.loads(response["message"]["content"])
            if not isinstance(facts, list):
                return []
            return [self._enrich_fact(f) for f in facts if f.get("type") in FACT_TYPES]
        except (json.JSONDecodeError, KeyError):
            return []

    def _enrich_fact(self, fact: dict) -> dict:
        """Añade campos automáticos a un hecho extraído."""
        now = datetime.utcnow()
        return {
            "fact_id": str(uuid.uuid4()),
            "type": fact.get("type", "observation"),
            "content": fact.get("content", "").strip(),
            "confidence": min(fact.get("confidence", 0.5), 1.0),
            "salience": min(fact.get("salience", 0.5), 1.0),
            "source": "extractor:llm",
            "created_at": now.isoformat() + "Z",
            "decay_rate": 0.05,
            "expires_at": (now + timedelta(days=90)).isoformat() + "Z",
            "is_active": True,
            "causal_context": fact.get("causal_context", ""),
            "metadata": fact.get("metadata", {}),
        }

    def compute_embedding(self, text: str) -> list[float]:
        """Calcula embedding para un texto."""
        if self.openai_key:
            # Usar OpenAI API
            import openai
            client = openai.OpenAI(api_key=self.openai_key)
            resp = client.embeddings.create(
                model="text-embedding-3-small",
                input=text,
            )
            return resp.data[0].embedding

        # Fallback: usar Ollama (modelo como nomic-embed-text)
        resp = self.ollama.embeddings(model="nomic-embed-text", prompt=text)
        return resp["embedding"]


# ─── Orquestador Go Client ───────────────────────────────────────────

class GoOrchestratorClient:
    """Cliente HTTP para el orquestador Go de Zyro."""

    def __init__(self, base_url: str = "http://localhost:8080"):
        self.client = httpx.AsyncClient(base_url=base_url, timeout=30)

    async def save_fact(self, fact: dict, tenant_id: str, phase: str) -> dict:
        """Envía un hecho al orquestador Go para persistir en HelixDB."""
        payload = {
            "fact": fact,
            "tenant_id": tenant_id,
            "phase": phase,
        }
        resp = await self.client.post("/api/v1/facts", json=payload)
        resp.raise_for_status()
        return resp.json()

    async def save_facts_batch(self, facts: list[dict], tenant_id: str, phase: str) -> list[dict]:
        """Envía múltiples hechos en batch."""
        payload = {
            "facts": facts,
            "tenant_id": tenant_id,
            "phase": phase,
        }
        resp = await self.client.post("/api/v1/facts/batch", json=payload)
        resp.raise_for_status()
        return resp.json()


# ─── CLI ─────────────────────────────────────────────────────────────

async def main():
    parser = argparse.ArgumentParser(description="Zyro Engram Fact Extractor")
    parser.add_argument("--input", required=True, help="Archivo de conversación (JSONL o markdown)")
    parser.add_argument("--tenant", default="default", help="Tenant ID")
    parser.add_argument("--phase", default="F0", help="Fase del pipeline")
    parser.add_argument("--ollama-model", default="llama3.2", help="Modelo Ollama para extracción")
    parser.add_argument("--orchestrator", default="http://localhost:8080", help="URL del orquestador Go")
    parser.add_argument("--dry-run", action="store_true", help="No enviar, solo mostrar")
    args = parser.parse_args()

    # Leer conversación
    with open(args.input) as f:
        conversation = f.read()

    # Extraer hechos
    extractor = FactExtractor(ollama_model=args.ollama_model)
    facts = extractor.extract_facts(conversation)
    print(f"Extraídos {len(facts)} hechos de {args.input}")

    if not facts:
        print("No se encontraron hechos.")
        return

    # Calcular embeddings
    for fact in facts:
        fact["embedding"] = extractor.compute_embedding(fact["content"])

    if args.dry_run:
        print("\n--- DRY RUN: Hechos extraídos ---")
        for i, f in enumerate(facts, 1):
            print(f"\n[{i}] {f['type'].upper()}: {f['content'][:120]}...")
            print(f"    confidence={f['confidence']:.2f}, salience={f['salience']:.2f}")
        return

    # Enviar al orquestador
    client = GoOrchestratorClient(base_url=args.orchestrator)
    results = await client.save_facts_batch(facts, args.tenant, args.phase)
    print(f"Guardados {len(results)} hechos en HelixDB (tenant={args.tenant}, phase={args.phase})")


if __name__ == "__main__":
    import asyncio
    asyncio.run(main())
```

---

## 7. Flujo de Integración en el Pipeline Zyro

### Ciclo Completo por Fase

```
┌────────────────────────────────────────────────────────────────┐
│                    PIPELINE ZYRO                                │
│                                                                │
│  FASE F0:                                                       │
│    Pre-fase:  consultar memoria relevante (inyectar en prompt) │
│    Durante:   el agente trabaja + consulta/escribe memoria     │
│    Post-fase: extraer hechos nuevos → almacenar en grafo       │
│                                                                │
│       │  handoff (contexto causal pasa a F1)                   │
│       ▼                                                        │
│  FASE F1:                                                       │
│    Pre-fase:  consultar memoria (decía F0, errores, prefs)     │
│    Durante:   ...                                               │
│    Post-fase: extraer hechos nuevos                             │
│                                                                │
│       │  ...                                                    │
│       ▼                                                        │
│  FASE F4:                                                       │
│    Post-fase: consolidación final de memoria                   │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

### Pre-Fase: Inyección de Contexto

Antes de que el agente comience una fase, el orquestador Go:

1. Construye un query vectorial con el contexto de la fase (nombre, descripción, objetivos)
2. Consulta HelixDB: `RecallMemories(ctx, MemoryQuery{...})`
3. Filtra resultados por relevancia (score > 0.3, salience > 0.2)
4. Formatea como contexto para el prompt del agente:

```
─── MEMORIA CAUSAL (fase actual: F1) ───

Decisiones activas:
  • [alta] Usamos Go SDK oficial de HelixDB (F0, confianza 0.95)
  • [media] Preferencia del usuario: SQLC para queries (F0, confianza 0.88)

Errores documentados:
  • Cliente raw JSON devolvía 404 por type mismatch en $id (F0)
    → Resuelto: migrar a Go SDK oficial

Patrones identificados:
  • Repository pattern con interfaces en internal/db/ (F0)
  • Los embeddings se calculan con text-embedding-3-small (F0)

Dependencias activas:
  • Fase F2 requiere migración a Go SDK oficial (completada en F1)
  • Fase F3 requiere índices vectoriales en HelixDB (pendiente)

Contradicciones activas:
  • Pref "usa GORM" vs Dec "usamos SQLC" → resuelto: SQLC gana (newest_wins)
  • Confianza en resolución: 0.92
```

### Durante la Fase: Consulta y Escritura de Memoria

El agente puede:
- **Consultar**: vía MCP tools `recall_memories(query, type_filter)`
- **Escribir**: vía MCP tool `save_fact(type, content, confidence, causal_context)`
- **Consultar contradicciones**: qué hechos contradicen a qué
- **Ver cronología**: línea de tiempo causal de decisiones

### Post-Fase: Extracción Automática

1. El orquestador toma el log completo de la fase
2. Llama al extractor Python: `python extractor.py --input fase_F1.log --tenant zyro --phase F1`
3. El extractor devuelve hechos con embeddings
4. El orquestador los guarda en HelixDB
5. El orquestador ejecuta resolución de contradicciones
6. El orquestador refuerza salience de hechos relevantes que fueron accedidos

---

## 8. Comparativa de Enfoques

### Construir sobre HelixDB (RECOMENDADO) vs Usar Engram MCP Server

| Criterio | HelixDB (Zyro Engram) | Engram MCP Server (keggan-std) |
|----------|----------------------|-------------------------------|
| **Stack** | Go + HelixDB (grafo+vector) | TypeScript + SQLite (relacional) |
| **Modelo de datos** | Nodos + edges causales | Tablas planas con FTS5 |
| **Búsqueda** | Híbrida (vector + BM25 + grafo) | Solo FTS5 (texto) |
| **Causalidad** | Explícita (edgetipos) | Implícita (timestamp/sesión) |
| **Contradicciones** | Automática (configurable) | No existe |
| **Decaimiento temporal** | Curva de Ebbinghaus configurable | No existe |
| **Escalabilidad** | Horizontal (HelixDB Cloud) | Single-node (SQLite) |
| **Multi-tenencia** | Nativa (tenant_id en índices) | Por proyecto (directorio) |
| **Embeddings** | Sí (API externa → HelixDB) | No |
| **Ciclo de sesión** | Custom (orquestador Go) | Built-in (start/work/end) |
| **File notes** | Custom (nodos FileNote) | Built-in con staleness |
| **Checkpoints** | Custom (nodos Checkpoint) | Built-in |
| **Handoff entre fases** | Custom (edges PRECEDES + contexto) | Built-in (handoff action) |
| **Dashboard** | No planeado (CLI-only) | React SPA con Recharts |
| **Tiempo de implementación** | 2-4 semanas (construir custom) | 1 día (integrar) |
| **Mantenimiento** | Interno (control total) | Externo (dependencia npm) |
| **Licencia** | Propietario | MIT |

### Pros y Contras Detallados

#### Enfoque A: Construir sobre HelixDB

**Pros:**
- Modelo de datos expresivo (grafo causal con tipos de arista semánticos)
- Búsqueda semántica real (embeddings + ANN)
- Resolución de contradicciones automática
- Curva de olvido configurable (memoria adaptativa)
- Sin dependencias externas (todo va contra HelixDB que ya tenemos)
- Escalable horizontalmente
- Control total sobre el schema y comportamiento

**Contras:**
- Requiere implementar extractor de hechos (LLM local)
- Requiere implementar orquestador Go
- No tiene ciclo de sesión pre-construido
- No tiene dashboard visual
- 2-4 semanas de implementación inicial

#### Enfoque B: Usar Engram MCP Server

**Pros:**
- Ciclo de vida de sesión completo (start/work/end)
- File notes con staleness tracking
- Handoff entre agentes
- Multi-agente con locks y broadcasts
- Dashboard React
- Instalación en 1 día

**Contras:**
- No hay grafo causal (solo tablas planas)
- No hay búsqueda semántica vectorial
- No hay resolución de contradicciones
- No hay curva de olvido
- Stack paralelo (Node.js + SQLite) cuando ya tenemos Go + HelixDB
- Datos fragmentados entre Engram y HelixDB
- Dependencia de mantenimiento externo
- SQLite no escala a millones de hechos
- No hay tenancy nativa

### Veredicto Final

**Construir sobre HelixDB es la opción correcta para Zyro.**

El Engram MCP server es una solución excelente para equipos que NO tienen una base de datos de grafos. Pero Zyro ya invirtió en HelixDB. Duplicar la capa de datos con SQLite solo introduce complejidad, fragmentación, y dependencias innecesarias.

La memoria causal es una extensión natural del grafo de conocimiento de Zyro. No es un sistema separado — es un label `Fact` más con edges causales. Los mismos embeddings que se usan para búsqueda de código sirven para memoria causal. El mismo Go SDK que se usa para skills sirve para guardar hechos.

---

## 9. Código de Referencia

### 9.1 Schema HelixQL Completo para Memoria Causal

```sql
-- Schema de memoria causal "Engram Zyro"
-- Ejecutar al inicializar proyecto

-- ============================================
-- NODOS
-- ============================================

-- Nodo Fact (hecho atómico de memoria)
CREATE LABEL Fact IF NOT EXISTS;
ALTER LABEL Fact ADD PROPERTIES (
    fact_id       STRING,       -- UUID v7 único
    type          STRING,       -- decision|error|preference|pattern|dependency|observation
    content       STRING,       -- Descripción textual del hecho
    embedding     F32_ARRAY,    -- Vector semántico 1536-dim
    salience      FLOAT64,      -- Importancia actual (0.0-1.0)
    confidence    FLOAT64,      -- Confianza en la veracidad (0.0-1.0)
    source        STRING,       -- Origen: agent:F1, user:input, extractor:llm
    created_at    DATETIME,
    last_accessed_at DATETIME,
    access_count  INT64,
    decay_rate    FLOAT64,      -- Velocidad de olvido (por día)
    expires_at    DATETIME,     -- Fecha de expiración
    is_active     BOOL,         -- false si fue olvidado o superseded
    is_stale      BOOL,         -- true si salience < threshold
    tenant_id     STRING,       -- Multi-tenencia
    phase         STRING,       -- Fase del pipeline: F0..F4
    metadata      JSON          -- Datos adicionales (modelo, archivos, etc.)
);

-- Nodo Session (sesión de agente)
CREATE LABEL Session IF NOT EXISTS;
ALTER LABEL Session ADD PROPERTIES (
    session_id    STRING,
    phase         STRING,
    agent_name    STRING,
    started_at    DATETIME,
    ended_at      DATETIME,
    summary       STRING,
    tenant_id     STRING
);

-- ============================================
-- ÍNDICES
-- ============================================

CREATE VECTOR INDEX IF NOT EXISTS idx_fact_embedding
    ON Fact(embedding)
    WITH TENANT tenant_id;

CREATE TEXT INDEX IF NOT EXISTS idx_fact_content
    ON Fact(content)
    WITH TENANT tenant_id;

CREATE EQUALITY INDEX IF NOT EXISTS idx_fact_type
    ON Fact(type);

CREATE EQUALITY INDEX IF NOT EXISTS idx_fact_tenant
    ON Fact(tenant_id);

CREATE EQUALITY INDEX IF NOT EXISTS idx_fact_active
    ON Fact(is_active);

CREATE EQUALITY INDEX IF NOT EXISTS idx_session_tenant
    ON Session(tenant_id);

-- ============================================
-- EDGES CAUSALES
-- ============================================

CREATE EDGE CAUSED IF NOT EXISTS;       -- A causó directamente B
CREATE EDGE PRECEDES IF NOT EXISTS;     -- A ocurrió antes que B
CREATE EDGE CONTRADICTS IF NOT EXISTS;  -- A contradice a B
CREATE EDGE SUPPORTS IF NOT EXISTS;     -- A soporta/refuerza B
CREATE EDGE REQUIRES IF NOT EXISTS;     -- A requiere B
CREATE EDGE DERIVES_FROM IF NOT EXISTS; -- A se deriva de B
CREATE EDGE REFERENCES IF NOT EXISTS;   -- A referencia a B (débil)

-- Índices en edges
CREATE EQUALITY INDEX IF NOT EXISTS idx_edge_caused ON CAUSED;
CREATE EQUALITY INDEX IF NOT EXISTS idx_edge_precedes ON PRECEDES;
CREATE EQUALITY INDEX IF NOT EXISTS idx_edge_contradicts ON CONTRADICTS;
CREATE EQUALITY INDEX IF NOT EXISTS idx_edge_supports ON SUPPORTS;
CREATE EQUALITY INDEX IF NOT EXISTS idx_edge_requires ON REQUIRES;
CREATE EQUALITY INDEX IF NOT EXISTS idx_edge_derives ON DERIVES_FROM;
CREATE EQUALITY INDEX IF NOT EXISTS idx_edge_references ON REFERENCES;
```

### 9.2 Ejemplo de Consulta de Memoria en Go

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    helix "github.com/helixdb/helix-db/sdks/go"
)

type EngramClient struct {
    client *helix.Client
}

func NewEngramClient(baseURL string) *EngramClient {
    client, err := helix.NewClient(baseURL)
    if err != nil {
        log.Fatalf("failed to create helix client: %v", err)
    }
    return &EngramClient{client: client}
}

// RecallForPhase recupera la memoria relevante para una fase del pipeline.
// Combina búsqueda vectorial (semántica), BM25 (textual), y traversal de grafo (causal).
func (e *EngramClient) RecallForPhase(ctx context.Context, phase string, tenantID string, query string, embedding []float32) (map[string]any, error) {
    q := helix.ReadQuery("recall_for_phase")
    phaseParam := q.ParamString("phase", phase)
    tenant := q.ParamString("tenant_id", tenantID)
    queryParam := q.ParamString("query", query)
    vecParam := q.ParamArray("embedding", embedding, helix.ParamTypeF32())
    kParam := q.ParamI64("k", 15)
    now := q.ParamDateTime("now", time.Now())

    return q.
        // 1. Búsqueda vectorial: hechos semánticamente similares, activos, no expirados
        VarAs("vector_memories",
            helix.G().
                VectorSearchNodes("Fact", "embedding", vecParam, kParam, tenant).
                Where(helix.PredEq("is_active", true)).
                Where(helix.PredGt("expires_at", now)).
                Where(helix.PredGt("salience", 0.15)).
                Project(
                    helix.ProjectPropAs("$id", "fact_id"),
                    helix.ProjectPropAs("$distance", "score"),
                    helix.ProjectProp("type"),
                    helix.ProjectProp("content"),
                    helix.ProjectProp("salience"),
                    helix.ProjectProp("phase"),
                    helix.ProjectProp("created_at"),
                ),
        ).

        // 2. Búsqueda BM25: hechos textualmente relevantes
        VarAs("text_memories",
            helix.G().
                TextSearchNodes("Fact", "content", queryParam, kParam, tenant).
                Where(helix.PredEq("is_active", true)).
                Where(helix.PredGt("expires_at", now)).
                Project(
                    helix.ProjectPropAs("$id", "fact_id"),
                    helix.ProjectPropAs("$distance", "score"),
                    helix.ProjectProp("type"),
                    helix.ProjectProp("content"),
                ),
        ).

        // 3. Traversal causal: decisiones que CAUSED errores o SUPPORTS patrones
        VarAs("causal_chain",
            helix.G().
                NWithLabel("Fact").
                Where(helix.PredEq("tenant_id", tenant)).
                Where(helix.PredEq("is_active", true)).
                Out("CAUSED", "SUPPORTS", "PRECEDES").
                Where(helix.PredEq("is_active", true)).
                Project(
                    helix.ProjectPropAs("$id", "related_id"),
                    helix.ProjectProp("type"),
                    helix.ProjectProp("content"),
                    helix.ProjectProp("salience"),
                ),
        ).

        // 4. Contradicciones activas
        VarAs("contradictions",
            helix.G().
                NWithLabel("Fact").
                Where(helix.PredEq("tenant_id", tenant)).
                Where(helix.PredEq("is_active", true)).
                Both("CONTRADICTS").
                Project(
                    helix.ProjectPropAs("$id", "fact_id"),
                    helix.ProjectProp("type"),
                    helix.ProjectProp("content"),
                ),
        ).

        Returning("vector_memories", "text_memories", "causal_chain", "contradictions").
        ExecAndReturn(ctx, e.client)
}

func main() {
    ctx := context.Background()
    client := NewEngramClient("http://localhost:6969")

    // Ejemplo: recuperar memoria para fase F1
    // En producción, embedding vendría de computeEmbedding(query)
    result, err := client.RecallForPhase(ctx, "F1", "proyecto-zyro",
        "base de datos queries go sdk migración",
        []float32{0.1, 0.2, 0.3}, // placeholder
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Vector memories: %v\n", result["vector_memories"])
    fmt.Printf("Text memories: %v\n", result["text_memories"])
    fmt.Printf("Causal chain: %v\n", result["causal_chain"])
    fmt.Printf("Contradictions: %v\n", result["contradictions"])
}
```

### 9.3 Actualización de Salience con Decaimiento

```go
// DecayAndRefresh actualiza la salience de todos los hechos según la curva de olvido.
// Se ejecuta como un cron job diario o al inicio de cada fase.
func (e *EngramClient) DecayAndRefresh(ctx context.Context, tenantID string) error {
    q := helix.WriteQuery("decay_and_refresh")
    tenant := q.ParamString("tenant_id", tenantID)
    now := q.ParamDateTime("now", time.Now())
    threshold := q.ParamF64("threshold", 0.15)

    // Cargar todos los facts activos
    type FactRow struct {
        ID           int64   `json:"$id"`
        Salience     float64 `json:"salience"`
        DecayRate    float64 `json:"decay_rate"`
        AccessCount  int64   `json:"access_count"`
        LastAccessed string  `json:"last_accessed_at"`
    }

    var facts []FactRow
    loadQ := helix.ReadQuery("load_active_facts").
        VarAs("facts",
            helix.G().
                NWithLabel("Fact").
                Where(helix.PredEq("tenant_id", tenant)).
                Where(helix.PredEq("is_active", true)).
                ValueMap("$id", "salience", "decay_rate", "access_count", "last_accessed_at"),
        ).
        Returning("facts")

    if err := e.client.Exec(ctx, loadQ, &facts); err != nil {
        return fmt.Errorf("load facts: %w", err)
    }

    for _, f := range facts {
        lastAccess, _ := time.Parse(time.RFC3339, f.LastAccessed)
        daysSinceAccess := time.Since(lastAccess).Hours() / 24.0

        // Curva de Ebbinghaus: salience(t) = salience_0 * e^(-decay_rate * t)
        newSalience := f.Salience * math.Exp(-f.DecayRate*daysSinceAccess)
        if newSalience < threshold {
            // Marcar como stale
            e.setFactActive(ctx, f.ID, false, "stale_below_threshold")
        } else {
            // Actualizar salience
            updateQ := helix.WriteQuery("update_salience").
                VarAs("updated",
                    helix.G().
                        NWhere(helix.SourceEq("$id", f.ID)).
                        SetProperty("salience", newSalience),
                )
            e.client.Exec(ctx, updateQ, nil)
        }
    }

    return nil
}
```

---

## 10. Recomendaciones

### Inmediatas (Sprint Actual)

1. **Definir el schema HelixQL** para nodos `Fact` y edges causales (ya está en este documento)
2. **Implementar el store de memoria** en Go usando el SDK oficial de HelixDB
3. **Crear MCP tools** para que el agente consulte y escriba memoria:
   - `recall_memories(query, type_filter, k)`
   - `save_fact(type, content, confidence, causal_context)`
   - `get_causal_chain(fact_id, direction, edge_types)`

### Corto Plazo (1-2 Semanas)

4. **Implementar el extractor de hechos** en Python con LLM local
5. **Implementar el flujo pre-fase**: inyectar memoria relevante en el prompt del agente
6. **Implementar el flujo post-fase**: extraer hechos del log y persistir
7. **Implementar resolución básica de contradicciones** (estrategia `newest_wins`)

### Mediano Plazo (3-4 Semanas)

8. **Implementar la curva de olvido** con cron job de decaimiento
9. **Implementar estrategias avanzadas de resolución** (`highest_confidence`, `keep_both`, `user_override`)
10. **Agregar causal context tracking**: cuando el agente registra un hecho, registrar también qué archivos/tareas/decisiones lo causaron
11. **Dashboard CLI**: `zyrocli engram stats`, `zyrocli engram contradictions`, `zyrocli engram timeline`

### Largo Plazo

12. **Memoria compartida multi-proyecto**: usar `tenant_id` para aislar o compartir memoria entre proyectos
13. **Federación de memoria**: varios desarrolladores compitiendo/colaborando sobre la misma base de conocimiento
14. **Refuerzo por feedback del usuario**: cuando el usuario dice "eso no es correcto", el sistema ajusta confianza y crea edge CONTRADICTS

---

## 11. Referencias

- **HelixDB Official Docs**: `docs/helixdb-official-docs.md` (ZyroAgentCLI)
- **HelixDB Deep Integration**: `docs/explorations/investigacion-02-helixdb-deep-integration.md` (ZyroAgentCLI)
- **HelixDB Go SDK**: `github.com/helixdb/helix-db/sdks/go`
- **HelixDB Skills**: https://github.com/HelixDB/skills
- **Curva de Olvido de Ebbinghaus**: https://en.wikipedia.org/wiki/Forgetting_curve
- **Engram (neurociencia)**: https://en.wikipedia.org/wiki/Engram_(neuropsychology)
- **Richard Semon (1904)**: Die Mneme als erhaltendes Prinzip im Wechsel des organischen Geschehens
- **Arquitectura Decisional V2**: `docs/explorations/fase1-arquitectura-decisional-v2.md` (ZyroAgentCLI)
- **Boundari y Políticas de Seguridad**: `docs/explorations/investigacion-03-boundari-politicas-seguridad.md` (ZyroAgentCLI)

---

> **Nota final**: "Engram" en ZyroAgentCLI no es un producto externo. Es el nombre conceptual para nuestro sistema de memoria causal construido sobre HelixDB. El término se usa en sentido neurocientífico, no como referencia al MCP server `engram-mcp-server`.
