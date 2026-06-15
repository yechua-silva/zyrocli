"""MCP tool: ``search_skills(query, limit)`` — search global skills by text query."""

from __future__ import annotations

import json

from helix_client import HelixClient


async def search_skills_tool(query: str, limit: int = 10) -> str:
    """Search global skills by text query.

    Args:
        query: The text to search for in skill nodes.
        limit: Maximum number of results to return (default 10).
    """
    client = HelixClient()
    try:
        nodes = await client.text_search("Skill", query, limit=limit)
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
