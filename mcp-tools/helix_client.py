"""HelixDB HTTP client — wraps httpx for querying HelixDB at /v1/query and /health."""

from __future__ import annotations

import json
from dataclasses import dataclass, field

import httpx


@dataclass
class HelixClient:
    """Stateless HTTP client for HelixDB.

    Each call creates its own ``httpx.AsyncClient``; the client is designed for
    short-lived CLI/agent invocations.
    """

    base_url: str = field(default="http://localhost:6969")
    project_id: str | None = field(default=None)
    timeout: float = field(default=10.0)

    def _headers(self) -> dict[str, str]:
        headers: dict[str, str] = {
            "Content-Type": "application/json",
        }
        if self.project_id:
            headers["x-project-id"] = self.project_id
        return headers

    async def query(self, payload: dict) -> dict:
        """Send a query to ``POST /v1/query`` and return the JSON response."""
        async with httpx.AsyncClient(timeout=self.timeout) as client:
            resp = await client.post(
                f"{self.base_url}/v1/query",
                json=payload,
                headers=self._headers(),
            )
            resp.raise_for_status()
            return resp.json()

    async def health(self) -> bool:
        """Return ``True`` when ``GET /health`` returns 200."""
        try:
            async with httpx.AsyncClient(timeout=self.timeout) as client:
                resp = await client.get(f"{self.base_url}/health")
                return resp.status_code == 200
        except httpx.RequestError:
            return False

    async def text_search(
        self, label: str, query: str, limit: int = 10
    ) -> list[dict]:
        """Convenience: text-search nodes of a given label."""
        result = await self.query(
            {
                "text_search": {
                    "label": label,
                    "query": query,
                    "limit": limit,
                }
            }
        )
        return result.get("nodes", [])

    async def get_node(self, label: str, id: int) -> dict | None:
        """Convenience: fetch a single node by label and id."""
        result = await self.query({"get_node": {"label": label, "id": id}})
        if "node" in result:
            return result["node"]
        return None

    async def get_outgoing(
        self, node_id: int, relation: str
    ) -> list[dict]:
        """Convenience: traverse outgoing edges by relation label."""
        result = await self.query(
            {
                "get_outgoing": {
                    "node_id": node_id,
                    "relation": relation,
                }
            }
        )
        return result.get("edges", [])

    async def get_incoming(
        self, node_id: int, relation: str
    ) -> list[dict]:
        """Convenience: traverse incoming edges by relation label."""
        result = await self.query(
            {
                "get_incoming": {
                    "node_id": node_id,
                    "relation": relation,
                }
            }
        )
        return result.get("edges", [])
