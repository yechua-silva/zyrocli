"""MCP tool: ``save_to_helix(label, properties)`` — create/update nodes and edges in HelixDB.
Validates required fields per node label before creating.
"""
from __future__ import annotations

import json

from helix_client import HelixClient

# Required fields per node label. Agents MUST provide all of these.
REQUIRED_FIELDS = {
    "Spec": ["project_id", "architecture", "modules", "dependencies", "testing_strategy"],
    "Pattern": ["name", "description", "language", "confidence"],
    "Library": ["name", "version", "category", "description"],
    "Skill": ["name", "language", "stars", "source_url"],
    "Decision": ["title", "context", "decision"],
    "Design": ["project_id", "components", "data_flow", "status"],
    "Task": ["project_id", "name", "description", "status"],
    "CodeModule": ["path", "language", "summary"],
    "Review": ["status", "findings"],
}


async def save_to_helix_tool(label: str, properties: dict) -> str:
    """Create a node in HelixDB with the given label and properties.

    Validates required fields before creating.

    Args:
        label: The node label (e.g. "Spec", "Skill", "Pattern", "Library").
        properties: Key-value pairs. Must include all required fields for this label.
    """
    # Validate required fields
    required = REQUIRED_FIELDS.get(label, [])
    missing = [f for f in required if f not in properties]
    if missing:
        return json.dumps(
            {"error": f"Missing required fields for {label}: {missing}", "label": label},
            indent=2,
        )

    client = HelixClient()
    try:
        node = await client.create_node(label, properties)
        return json.dumps(
            {"status": "ok", "label": label, "id": node.get("id")},
            indent=2,
        )
    except Exception as exc:
        return json.dumps(
            {
                "error": f"HelixDB write failed: {exc}",
                "helix_url": client.base_url,
            },
            indent=2,
        )


async def link_to_project_tool(project_id: int, target_label: str, target_id: int, edge_label: str, properties: dict | None = None) -> str:
    """Create an edge from a Project node to another node."""
    client = HelixClient()
    try:
        edge = await client.create_edge(project_id, target_id, edge_label, properties or {})
        return json.dumps(
            {"status": "ok", "edge_id": edge.get("id"), "label": edge_label},
            indent=2,
        )
    except Exception as exc:
        return json.dumps(
            {
                "error": f"HelixDB edge failed: {exc}",
                "helix_url": client.base_url,
            },
            indent=2,
        )


async def find_project_tool(name: str) -> str:
    """Find a Project node by name and return its ID."""
    client = HelixClient()
    try:
        nodes = await client.text_search("Project", name, limit=1)
        if nodes:
            return json.dumps({"status": "ok", "project_id": nodes[0].get("id")}, indent=2)
        return json.dumps({"status": "not_found", "name": name}, indent=2)
    except Exception as exc:
        return json.dumps({"error": f"HelixDB search failed: {exc}"}, indent=2)
