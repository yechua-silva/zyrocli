# Investigación: Sistema de Embeddings para Memoria Causal

> Fecha: 2026-06-15
> Propósito: Elegir el mejor modelo/API para generar embeddings en ZyroAgentCLI
> Contexto: Local-first, CPU-friendly, fallback a free APIs, skill autogestionada

---

## 1. El Problema

La memoria causal (Sprint 4) necesita embeddings para:
- Búsqueda semántica (vector ANN) sobre Facts
- Detección de contradicciones (cosine similarity entre Facts)
- Ranking de relevancia (hybrid search: vector + BM25 + RRF)

Sin embeddings: solo BM25 (texto). Funciona, pero menos preciso.

## 2. Criterios de Evaluación

| Criterio | Peso | Explicación |
|----------|------|-------------|
| Calidad embedding | Alto | Afecta recall de búsqueda semántica y detección de contradicciones |
| Velocidad CPU | Alto | Debe correr en laptops sin GPU |
| Tamaño descarga | Medio | <500MB ideal para instalación rápida |
| Sin dependencias externas | Alto | Local-first siempre que sea posible |
| Fallback gratuito | Medio | Si no hay Ollama, que funcione con API gratis |
| Dimensionalidad | Medio | Afecta performance de índice vectorial y precisión |

## 3. Modelos Evaluados

### 3.1 Modelos Locales (Ollama)

| Modelo | Params | Tamaño | Dims | Velocidad CPU | Calidad | Veredicto |
|--------|--------|--------|------|---------------|---------|-----------|
| all-minilm (L6) | 23M | ~80MB | 384 | ⚡ Ultrarrápido | 🟡 Suficiente | Bueno para starter |
| all-minilm (L12) | 33M | ~120MB | 384 | ⚡ Rápido | 🟡 Medio | Alternativa ligera |
| nomic-embed-text | 137M | ~274MB | 768 | 🟢 Rápido | 🟢 Bueno | Default actual |
| mxbai-embed-large | 334M | ~350MB | 768 | 🟡 Moderado | 🔴 Excelente | ★ RECOMENDADO |
| bge-m3 | 567M | ~2.2GB | 1024 | 🔴 Lento | 🔴 Excelente | Multilenguaje, pesado |

**RECOMENDADO: mxbai-embed-large**
- Mejor calidad de los que corren bien en CPU
- 768 dimensiones (buen balance con índice vectorial)
- Reconocido como el mejor embedding model local en benchmarks MTEB
- ~350MB descarga razonable

### 3.2 Free APIs (Fallback)

| Proveedor | Modelo | Dims | Límite | Calidad | Veredicto |
|-----------|--------|------|--------|---------|-----------|
| Scaleway | qwen3-embedding-8b | 1024 | 1M tokens | 🔴 Excelente (8B params) | ★ MEJOR CALIDAD |
| GitHub Models | OpenAI Text Embedding 3 small | 1536 | Según tier Copilot | 🔴 Excelente | Ideal si tiene GitHub Copilot |
| Cohere | embed-english-v3.0 | 1024 | 1000 req/mes | 🔴 Excelente | Multilingual support |
| NVIDIA NIM | Various | varies | 40 req/min | 🟢 Bueno | Rápido, confiable |
| Google AI Studio | Gemini embedding | 768 | 20 req/día | 🟢 Bueno | Fácil de usar |
| OpenRouter | varies | varies | 50 req/día | 🟢 Bueno | Muchos modelos |

**RECOMENDADO: Scaleway qwen3-embedding-8b** (mejor calidad gratuita, 1M tokens)

## 4. Arquitectura Propuesta: Embedding Harness

El sistema de embeddings debe ser una **skill/harness autogestionada** que:

1. Corre en segundo plano como MCP server propio
2. El agente la llama vía tool MCP (`generate_embedding(text) -> vector`)
3. No interrumpe el flujo del agente principal (es async, fire-and-forget)
4. Tiene su propio sistema de caché para no regenerar embeddings duplicados

### Diagrama

```
OpenCode Agent
  │
  ├── llama a zyro-sdd-tasks (normal)
  │
  └── llama a zyro-embedding-harness (MCP tool)
        │
        ├── Si Ollama disponible → mxbai-embed-large (local, CPU)
        ├── Si no, pero hay API key → Scaleway/GitHub/Cohere (free API)
        └── Si no → error graceful (BM25 fallback)
```

### Caché LRU
- En memoria: últimas 1000 consultas
- En disco: `~/.zyro/embedding-cache/` (SQLite o JSONL)
- Embeddings duplicados no se regeneran

## 5. Instalación Interactiva (zyro setup)

La instalación debe ser:

```
zyro setup
  ...
  ❓ ¿Querés instalar Ollama para búsqueda semántica local? (recomendado)
     [Y/n]: y
     → Detectando GPU... NVIDIA RTX 3060 detectada
     → Instalando Ollama...
     → Pulling mxbai-embed-large (334MB)...
     → Embeddings locales activados ✅
     → ¿Querés configurar fallback a API gratuita?
       [y/N]: y
       → Opciones:
         1. Scaleway (qwen3-embedding-8b, 1M tokens gratis)
         2. GitHub Models (OpenAI Embedding 3, requiere Copilot)
         3. Cohere (embed-english-v3.0, 1000 req/mes)
       → Seleccioná (1-3): 1
       → Configurando Scaleway API...
     → Sistema de embeddings listo ✅
  ...
```

### Detección de GPU
```bash
# NVIDIA
if nvidia-smi &>/dev/null; then
    echo "NVIDIA GPU detectada"
    # Instalar Ollama con GPU support
    ollama serve &
    ollama pull mxbai-embed-large
fi

# AMD ROCm
if rocminfo &>/dev/null; then
    echo "AMD GPU detectada"
    # Instalar Ollama con ROCm
fi

# CPU
else
    echo "CPU mode"
    ollama pull mxbai-embed-large  # corre en CPU
fi
```

## 6. Skill/Harness Propuesta

Crear `mcp-tools/embedding_harness.py`:

```python
"""Zyro Embedding Harness — MCP tool para generar embeddings.

Corre como servidor MCP separado para no interrumpir el agente principal.
"""

from mcp.server.fastmcp import FastMCP
import hashlib, json, os, sqlite3
from pathlib import Path

server = FastMCP("zyro-embedding-harness")

# Cache LRU en disco
CACHE_DIR = Path.home() / ".zyro" / "embedding-cache"
CACHE_DIR.mkdir(parents=True, exist_ok=True)

def _get_embedding(text: str) -> list[float]:
    """Genera embedding con el mejor proveedor disponible."""
    # 1. Intentar Ollama (mxbai-embed-large)
    # 2. Fallback a Scaleway/GitHub/Cohere
    # 3. Si nada funciona, retornar vacío (BM25 fallback)
    pass

@server.tool
async def embed(text: str) -> list[float]:
    """Genera embedding para un texto."""
    cache_key = hashlib.sha256(text.encode()).hexdigest()
    # Check cache...
    embedding = _get_embedding(text)
    # Store cache...
    return embedding

@server.tool
async def embed_batch(texts: list[str]) -> list[list[float]]:
    """Genera embeddings para múltiples textos."""
    return [_get_embedding(t) for t in texts]

@server.tool
async def status() -> dict:
    """Estado del sistema de embeddings."""
    return {
        "provider": "ollama" or "scaleway" or "none",
        "model": "mxbai-embed-large" or "qwen3-embedding-8b",
        "cache_size": 1234,
        "available": True,
    }
```

## 7. Pipeline de Prioridad (decidido)

```
┌─ ¿Ollama + mxbai-embed-large disponible?
│   ✅ → Usar (local, CPU/GPU, 768 dims)
│
├─ ¿No? → ¿Scaleway/GitHub Models API configurado?
│   ✅ → Usar como fallback (free, mayor calidad)
│
├─ ¿No? → ¿Cohere/NVIDIA NIM configurado?
│   ✅ → Usar como fallback terciario
│
└─ ¿Nada disponible?
    → Degradación graceful: solo BM25 text search
    → EmbeddingService devuelve nil
    → El sistema funciona sin vectores
```

## 8. Recomendaciones

1. **Modelo default local**: mxbai-embed-large (mejor calidad/speed en CPU)
2. **Fallback gratis**: Scaleway qwen3-embedding-8b (1M tokens, 8B params)
3. **Harness**: MCP server separado (no interrumpe agente principal)
4. **Instalación**: `zyro setup` interactivo pregunta y configura automáticamente
5. **Detección GPU**: nvidia-smi → Ollama GPU; rocminfo → Ollama ROCm; nada → CPU
6. **Cache**: LRU en disco en ~/.zyro/embedding-cache/
7. **Degradación**: Sin embeddings → BM25 puro, sin errores

## 9. Referencias

- https://ollama.com/blog/embedding-models (modelos oficiales Ollama)
- https://github.com/cheahjs/free-llm-api-resources (free APIs)
- https://build.nvidia.com/explore/discover (NVIDIA NIM)
- https://ollama.com/library/mxbai-embed-large (modelo recomendado)
- MTEB Leaderboard: https://huggingface.co/spaces/mteb/leaderboard
