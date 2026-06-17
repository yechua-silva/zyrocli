"""Fact search — async function, not MCP tool."""
from __future__ import annotations

from helix_client import HelixClient


async def search_facts(query: str, limit: int = 10) -> list[dict]:
    """Search Fact nodes in HelixDB by text content."""
    client = HelixClient()
    results = await client.text_search("Fact", query, limit)
    return results
