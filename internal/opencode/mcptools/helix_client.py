"""HelixDB HTTP client — wraps httpx for querying HelixDB at /v1/query and /health."""

from __future__ import annotations

import json
from dataclasses import dataclass, field

import httpx

# Edge label fallbacks: canonical -> [legacy labels] for backward compatibility.
# When a canonical label returns no results the legacy labels are tried in order.
EDGE_LABEL_FALLBACKS: dict[str, list[str]] = {
    "has_skill": ["REQUIRES_SKILL"],
    "has_code": ["REFERENCES"],
    "has_doc": [],
    "has_pattern": [],
    "depends_on": [],
}


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
                    p.get("$id") if "$id" in p else p.get("id")
                    for p in data["properties"]
                    if p.get("$id") is not None or p.get("id") is not None
                ]
        return []

    def _clean_props(self, p: dict) -> dict:
        """Normalize: rename $id to id, drop internal keys starting with ``$``."""
        return {
            ("id" if k == "$id" else k): v
            for k, v in p.items()
            if not k.startswith("$") or k == "$id"
        }

    def _get_properties(self, result: dict, name: str = "n") -> list[dict]:
        """Extract full property dicts from a v3 query result.

        Used by ``text_search()`` to return complete properties (via ValueMap).
        When the result has no properties, falls back to ``[{"id": i}]``
        for backward compatibility.
        """
        data = result.get(name, {})
        if isinstance(data, dict) and "properties" in data:
            return [self._clean_props(p) for p in data["properties"] if p is not None]
        return [{"id": i} for i in self._get_ids(result, name)]

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

    async def create_edge(self, from_id: int, to_id: int, label: str, properties: dict | None = None) -> dict:
        """Create a directed edge from from_id -> to_id with label and properties.

        Uses the v3 AddE step: N(source) -> AddE(label, target, props) -> Project($id).

        Response v3 format: {name: {"properties": [{"id": <edge_id>}]}}
        """
        if properties is None:
            properties = {}

        payload = self._v3_envelope(
            [
                {
                    "name": "e",
                    "steps": [
                        {
                            "N": {
                                "Ids": [from_id],
                            }
                        },
                        {
                            "AddE": {
                                "label": label,
                                "to": {"Ids": [to_id]},
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
        ids = self._get_ids(result, "e")
        return {"id": ids[0]} if ids else {}

    async def get_node(self, label: str, id: int, include_properties: bool = False) -> dict | None:
        """Fetch a single node by label and id.

        When ``include_properties=True``, adds a ``ValueMap`` step so the
        response includes all properties of the node (acceptance_criteria, etc.).

        Response v3 format: {name: {"ids": [<id>]}} or with ValueMap:
        {name: {"properties": [{"$id": <id>, "name": ..., ...}]}}
        """
        steps: list[dict] = [
            {
                "NWhere": {
                    "Eq": ["$id", {"I64": id}],
                }
            },
        ]
        if include_properties:
            steps.append(
                {
                    "ValueMap": [
                        "$id", "name", "content", "language",
                        "path", "summary", "description", "source",
                    ]
                }
            )

        payload = self._v3_envelope(
            [
                {
                    "name": "n",
                    "steps": steps,
                }
            ],
            request_type="read",
        )
        result = await self.query(payload)
        if include_properties:
            props = self._get_properties(result, "n")
            return props[0] if props else None
        ids = self._get_ids(result, "n")
        if not ids:
            return None
        return {"id": ids[0]}

    async def text_search(
        self, label: str, query: str, limit: int = 10, property: str = "name"
    ) -> list[dict]:
        """Convenience: text-search nodes of a given label.

        Response includes full properties via ``ValueMap``.
        Default ``property="name"`` provides backward compatibility;
        pass ``property="content"`` for ``Fact`` / ``Document`` nodes.

        Response format: {name: {"properties": [{"$id": 1, "name": "...", ...}, ...]}}
        """
        payload = self._v3_envelope(
            [
                {
                    "name": "n",
                    "steps": [
                        {
                            "TextSearchNodes": {
                                "label": label,
                                "property": property,
                                "query_text": {"Value": {"String": query}},
                                "k": {"Literal": limit},
                            }
                        },
                        {"ValueMap": None},
                    ],
                }
            ],
            request_type="read",
        )
        result = await self.query(payload)
        return self._get_properties(result, "n")

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

    async def get_outgoing_with_fallback(self, node_id: int, canonical: str) -> list[dict]:
        """Traverse outgoing edges, trying canonical label first then legacy fallbacks.

        Uses ``EDGE_LABEL_FALLBACKS`` to locate legacy labels when the canonical
        label returns no results. See Bug #8 in spec-fix-mcp-tools-bugs.md.
        """
        result = await self.get_outgoing(node_id, canonical)
        if result:
            return result
        for legacy in EDGE_LABEL_FALLBACKS.get(canonical, []):
            result = await self.get_outgoing(node_id, legacy)
            if result:
                return result
        return []

    async def get_incoming_with_fallback(self, node_id: int, canonical: str) -> list[dict]:
        """Traverse incoming edges, trying canonical label first then legacy fallbacks."""
        result = await self.get_incoming(node_id, canonical)
        if result:
            return result
        for legacy in EDGE_LABEL_FALLBACKS.get(canonical, []):
            result = await self.get_incoming(node_id, legacy)
            if result:
                return result
        return []
