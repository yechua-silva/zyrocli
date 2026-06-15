"""HelixDB HTTP client — wraps httpx for querying HelixDB at /v1/query and /health."""

from __future__ import annotations

import json
from dataclasses import dataclass, field

import httpx


@dataclass
class HelixClient:
    """Stateless HTTP client for HelixDB v3.

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

    def _v3_envelope(
        self, queries: list[dict], request_type: str = "read"
    ) -> dict:
        """Build a v3 query envelope from a list of query dicts.

        Each query dict must have ``name`` (str) and ``steps`` (list).
        """
        query_list = [
            {
                "Query": {
                    "name": q["name"],
                    "steps": q["steps"],
                    "condition": None,
                }
            }
            for q in queries
        ]
        return_names = [q["name"] for q in queries]
        return {
            "request_type": request_type,
            "parameters": {},
            "query": {
                "queries": query_list,
                "returns": return_names,
            },
        }

    def _value(self, v: str | int | float) -> dict:
        """Wrap a Python value in the v3 Value envelope."""
        if isinstance(v, str):
            return {"Value": {"String": v}}
        elif isinstance(v, int):
            return {"Value": {"I64": v}}
        elif isinstance(v, float):
            return {"Value": {"F64": v}}
        return {"Value": {"String": str(v)}}

    def _props(self, properties: dict) -> list[list]:
        """Convert a dict to v3 properties array format."""
        return [[k, self._value(v)] for k, v in properties.items()]

    def _get_ids(self, result: dict, name: str = "n") -> list[int]:
        """Extract IDs from a v3 query result.

        v3 returns {name: {"ids": [1,2,3]}} or {name: {"properties": [{"$id": 1}, ...]}}
        """
        data = result.get(name, {})
        if isinstance(data, dict):
            if "ids" in data:
                return data["ids"]
            if "properties" in data:
                return [
                    p.get("$id") or p.get("id")
                    for p in data["properties"]
                    if p.get("$id") is not None or p.get("id") is not None
                ]
        return []

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

    # ------------------------------------------------------------------
    # v3 helper methods
    # ------------------------------------------------------------------

    async def create_node(self, label: str, properties: dict) -> dict:
        """Create a node and return it with its ID.

        Response v3 format: {name: {"properties": [{"id": <id>}]}}
        """
        payload = self._v3_envelope(
            [
                {
                    "name": "n",
                    "steps": [
                        {
                            "AddN": {
                                "label": label,
                                "properties": self._props(properties),
                            }
                        },
                        {
                            "Project": [
                                {"source": "$id", "alias": "id"},
                            ]
                        },
                    ],
                }
            ],
            request_type="write",
        )
        result = await self.query(payload)
        ids = self._get_ids(result, "n")
        return {"id": ids[0]} if ids else {}

    async def get_node(self, label: str, id: int) -> dict | None:
        """Fetch a single node by label and id.

        Response v3 format: {name: {"ids": [<id>]}}
        """
        payload = self._v3_envelope(
            [
                {
                    "name": "n",
                    "steps": [
                        {
                            "NWhere": {
                                "Eq": ["$id", {"I64": id}],
                            }
                        },
                    ],
                }
            ],
            request_type="read",
        )
        result = await self.query(payload)
        ids = self._get_ids(result, "n")
        if not ids:
            return None
        return {"id": ids[0]}

    async def text_search(
        self, label: str, query: str, limit: int = 10
    ) -> list[dict]:
        """Convenience: text-search nodes of a given label.

        Requires a text index on (label, name). Response format can be
        {name: {"ids": [...]}} or {name: {"properties": [...]}}.
        """
        payload = self._v3_envelope(
            [
                {
                    "name": "n",
                    "steps": [
                        {
                            "TextSearchNodes": {
                                "label": label,
                                "property": "name",
                                "query_text": {"Value": {"String": query}},
                                "k": {"Literal": limit},
                            }
                        },
                    ],
                }
            ],
            request_type="read",
        )
        result = await self.query(payload)
        ids = self._get_ids(result, "n")
        return [{"id": i} for i in ids]

    async def get_outgoing(
        self, node_id: int, relation: str
    ) -> list[dict]:
        """Traverse outgoing edges by relation label, returning target node IDs.

        Response v3 format: {name: {"ids": [<id>, ...]}}
        """
        payload = self._v3_envelope(
            [
                {
                    "name": "src",
                    "steps": [
                        {
                            "NWhere": {
                                "Eq": ["$id", {"I64": node_id}],
                            }
                        },
                    ],
                },
                {
                    "name": "n",
                    "steps": [
                        {"N": {"Var": "src"}},
                        {"Out": relation},
                    ],
                },
            ],
            request_type="read",
        )
        result = await self.query(payload)
        ids = self._get_ids(result, "n")
        return [{"id": i} for i in ids]

    async def get_incoming(
        self, node_id: int, relation: str
    ) -> list[dict]:
        """Traverse incoming edges by relation label, returning source node IDs.

        Response v3 format: {name: {"ids": [<id>, ...]}}
        """
        payload = self._v3_envelope(
            [
                {
                    "name": "src",
                    "steps": [
                        {
                            "NWhere": {
                                "Eq": ["$id", {"I64": node_id}],
                            }
                        },
                    ],
                },
                {
                    "name": "n",
                    "steps": [
                        {"N": {"Var": "src"}},
                        {"In": relation},
                    ],
                },
            ],
            request_type="read",
        )
        result = await self.query(payload)
        ids = self._get_ids(result, "n")
        return [{"id": i} for i in ids]
