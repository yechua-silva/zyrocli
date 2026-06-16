"""Skills search — async function, not MCP tool."""

from __future__ import annotations

from helix_client import HelixClient


async def search_skills(query: str, limit: int = 10) -> list[dict]:
    """Search skill nodes in HelixDB by text."""
    client = HelixClient()
    results = await client.text_search("Skill", query, limit)
    return results
