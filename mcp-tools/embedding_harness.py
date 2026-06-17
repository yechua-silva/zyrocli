"""Zyro Embedding Harness — MCP server para generar embeddings.

Pipeline de prioridad:
1. Ollama + mxbai-embed-large (local, CPU/GPU, 1024 dims)
2. Scaleway qwen3-embedding-8b (free API, 1M tokens)
3. GitHub Models / Cohere (fallback terciario)
4. BM25 puro (degradación graceful)

Uso:
    uv run --directory mcp-tools python embedding_harness.py
"""

from __future__ import annotations

import hashlib
import json
import math
import os
import sqlite3
import sys
from pathlib import Path
from mcp.server.fastmcp import FastMCP

server = FastMCP("zyro-embedding-harness")

# Cache LRU en disco
CACHE_DIR = Path.home() / ".zyro" / "embedding-cache"
CACHE_DIR.mkdir(parents=True, exist_ok=True)
CACHE_DB = CACHE_DIR / "embeddings.db"


def _init_cache():
    conn = sqlite3.connect(str(CACHE_DB))
    conn.execute("""
        CREATE TABLE IF NOT EXISTS embeddings (
            hash TEXT PRIMARY KEY,
            vector TEXT NOT NULL,
            model TEXT NOT NULL,
            created_at TEXT NOT NULL
        )
    """)
    conn.commit()
    return conn


def _get_cached(hash: str) -> list[float] | None:
    try:
        conn = sqlite3.connect(str(CACHE_DB))
        row = conn.execute(
            "SELECT vector FROM embeddings WHERE hash = ?", (hash,)
        ).fetchone()
        conn.close()
        if row:
            return json.loads(row[0])
    except Exception as e:
        print(f"[embedding_harness] _get_cached error: {e}", file=sys.stderr)
        return None


def _set_cache(hash: str, vector: list[float], model: str):
    try:
        conn = sqlite3.connect(str(CACHE_DB))
        conn.execute(
            "INSERT OR REPLACE INTO embeddings (hash, vector, model, created_at) VALUES (?, ?, ?, datetime('now'))",
            (hash, json.dumps(vector), model),
        )
        conn.commit()
        conn.close()
    except Exception as e:
        print(f"[embedding_harness] _set_cache error: {e}", file=sys.stderr)


def _bm25_fallback(text: str) -> list[float]:
    """BM25-like pseudo-embedding usando feature hashing de 1024 dims.

    Último recurso cuando todos los proveedores fallan.
    Solo usa la librería estándar de Python (sin dependencias externas).
    """
    # 1. Tokenización: split por whitespace + lowercase
    tokens = text.lower().split()
    if not tokens:
        return [0.0] * 1024

    # 2. Frecuencias de tokens
    freq: dict[str, int] = {}
    for t in tokens:
        freq[t] = freq.get(t, 0) + 1

    # 3. Pseudo-IDF: log(1 + avg_len / len(tokens))
    avg_len = 100.0  # longitud promedio asumida del "corpus"
    doc_len = len(tokens)
    idf = math.log(1.0 + avg_len / doc_len)

    # 4. Feature hashing a 1024 dimensiones (hashing trick)
    dim = 1024
    vector = [0.0] * dim

    for token, count in freq.items():
        # BM25-like term frequency saturation (k1 = 1.5)
        tf = count / (count + 1.5)
        weight = tf * idf

        # Hash determinista a índice + signo
        h = hashlib.md5(token.encode()).hexdigest()
        idx = int(h[:8], 16) % dim
        sign = 1 if int(h[8:16], 16) % 2 == 0 else -1

        vector[idx] += sign * weight

    # 5. Normalizar a unit norm
    norm = math.sqrt(sum(v * v for v in vector))
    if norm > 0:
        vector = [v / norm for v in vector]

    return vector


def _get_embedding(text: str) -> list[float]:
    """Genera embedding con el mejor proveedor disponible."""
    cache_key = hashlib.sha256(text.encode()).hexdigest()

    # 1. Check cache
    cached = _get_cached(cache_key)
    if cached:
        return cached

    vector = None
    model = "none"

    # 2. Intentar Ollama (mxbai-embed-large)
    try:
        import httpx

        resp = httpx.post(
            "http://localhost:11434/api/embeddings",
            json={"model": "mxbai-embed-large", "prompt": text},
            timeout=10,
        )
        if resp.status_code == 200:
            data = resp.json()
            vector = data.get("embedding", [])
            model = "mxbai-embed-large"
    except Exception as e:
        print(f"[embedding_harness] Ollama error: {e}", file=sys.stderr)

    # 3. Fallback: Scaleway / GitHub Models / Cohere
    if not vector:
        for provider, config in _get_providers().items():
            try:
                vector = _call_provider(provider, config, text)
                if vector:
                    model = config.get("model", provider)
                    break
            except Exception as e:
                print(f"[embedding_harness] Provider '{provider}' error: {e}", file=sys.stderr)
                continue

    # 4. Cache y retorno
    if vector:
        _set_cache(cache_key, vector, model)
        return vector

    # 5. BM25 fallback (degradación graceful)
    vector = _bm25_fallback(text)
    model = "bm25-fallback"
    _set_cache(cache_key, vector, model)
    return vector


def _get_providers() -> dict:
    """Lee proveedores configurados en ~/.zyro/config.yaml."""
    config_path = Path.home() / ".zyro" / "config.yaml"
    if not config_path.exists():
        return {}

    try:
        import yaml

        with open(config_path) as f:
            config = yaml.safe_load(f)
        return config.get("embeddings", {}).get("providers", {})
    except Exception as e:
        print(f"[embedding_harness] _get_providers error: {e}", file=sys.stderr)
        return {}


def _call_provider(provider: str, config: dict, text: str) -> list[float]:
    """Llama a un proveedor de embeddings."""
    import httpx

    if provider == "scaleway":
        resp = httpx.post(
            config.get("url", "https://api.scaleway.ai/v1/embeddings"),
            json={
                "model": config.get("model", "qwen3-embedding-8b"),
                "input": text,
            },
            headers={"Authorization": f"Bearer {config.get('api_key', '')}"},
            timeout=15,
        )
        if resp.status_code == 200:
            data = resp.json()
            return data.get("data", [{}])[0].get("embedding", [])

    elif provider == "github_models":
        resp = httpx.post(
            "https://models.inference.ai.azure.com/embeddings",
            json={"model": "text-embedding-3-small", "input": text},
            headers={"Authorization": f"Bearer {config.get('api_key', '')}"},
            timeout=15,
        )
        if resp.status_code == 200:
            data = resp.json()
            return data.get("data", [{}])[0].get("embedding", [])

    return []


@server.tool()
async def embed(text: str) -> list[float]:
    """Genera embedding para un texto."""
    return _get_embedding(text)


@server.tool()
async def embed_batch(texts: list[str]) -> list[list[float]]:
    """Genera embeddings para múltiples textos."""
    return [_get_embedding(t) for t in texts]


@server.tool()
async def status() -> dict:
    """Estado del sistema de embeddings."""
    # Probar qué proveedor está disponible
    provider = "none"
    model = "none"
    try:
        import httpx

        resp = httpx.post(
            "http://localhost:11434/api/embeddings",
            json={"model": "mxbai-embed-large", "prompt": "test"},
            timeout=5,
        )
        if resp.status_code == 200:
            provider = "ollama"
            model = "mxbai-embed-large"
    except Exception as e:
        print(f"[embedding_harness] status Ollama test error: {e}", file=sys.stderr)

    # Contar cache
    cache_count = 0
    try:
        conn = sqlite3.connect(str(CACHE_DB))
        cache_count = conn.execute("SELECT COUNT(*) FROM embeddings").fetchone()[0]
        conn.close()
    except Exception as e:
        print(f"[embedding_harness] status cache count error: {e}", file=sys.stderr)

    return {
        "provider": provider,
        "model": model,
        "cache_size": cache_count,
        "cache_dir": str(CACHE_DIR),
        "available": provider != "none",
    }


if __name__ == "__main__":
    _init_cache()
    server.run(transport="stdio")
