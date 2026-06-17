"""MCP tool: ``search_facts(query, limit)`` — search Fact nodes by text query."""
from __future__ import annotations

import json

from helix_client import HelixClient


async def search_facts_tool(query: str, limit: int = 10) -> str:
    """Search Fact entries in HelixDB by text.

    Args:
        query: The text to search for in Fact nodes.
        limit: Maximum number of results to return (default 10).
    """
    client = HelixClient()
    try:
        nodes = await client.text_search("Fact", query, limit=limit)
        return json.dumps(
            {"query": query, "count": len(nodes), "results": nodes},
            indent=2,
        )
    except Exception as exc:
        return json.dumps(
            {
                "error": f"HelixDB connection failed: {exc}",
                "helix_url": client.base_url,
            },
            indent=2,
        )
