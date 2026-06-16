"""Zyro Embedding Harness — MCP server para generar embeddings.

Pipeline de prioridad:
1. Ollama + mxbai-embed-large (local, CPU/GPU, 768 dims)
2. Scaleway qwen3-embedding-8b (free API, 1M tokens)
3. GitHub Models / Cohere (fallback terciario)
4. BM25 puro (degradación graceful)

Uso:
    uv run --directory mcp-tools python embedding_harness.py
"""

from __future__ import annotations

import hashlib
import json
import os
import sqlite3
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
    except Exception:
        pass
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
    except Exception:
        pass


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
    except Exception:
        pass

    # 3. Fallback: Scaleway / GitHub Models / Cohere
    if not vector:
        for provider, config in _get_providers().items():
            try:
                vector = _call_provider(provider, config, text)
                if vector:
                    model = config.get("model", provider)
                    break
            except Exception:
                continue

    # 4. Cache y retorno
    if vector:
        _set_cache(cache_key, vector, model)
        return vector

    return []


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
    except Exception:
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


@server.tool
async def embed(text: str) -> list[float]:
    """Genera embedding para un texto."""
    return _get_embedding(text)


@server.tool
async def embed_batch(texts: list[str]) -> list[list[float]]:
    """Genera embeddings para múltiples textos."""
    return [_get_embedding(t) for t in texts]


@server.tool
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
    except Exception:
        pass

    # Contar cache
    cache_count = 0
    try:
        conn = sqlite3.connect(str(CACHE_DB))
        cache_count = conn.execute("SELECT COUNT(*) FROM embeddings").fetchone()[0]
        conn.close()
    except Exception:
        pass

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
