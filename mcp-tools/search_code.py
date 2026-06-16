"""Code search — async function, not MCP tool."""

from __future__ import annotations

from helix_client import HelixClient


async def search_code(query: str, limit: int = 10, project_id: str | None = None) -> list[dict]:
    """Search code nodes in HelixDB by text."""
    client = HelixClient(project_id=project_id)
    results = await client.text_search("CodeNode", query, limit)
    return results
