"""MCP tool: ``task_context(id)`` — return full task context from HelixDB.

Returns six sections (skills, code, docs, patterns, dependents, dependencies)
as a structured JSON string.
"""

from __future__ import annotations

import json

from helix_client import HelixClient


async def task_context_tool(id: int) -> str:
    """Get full context for a task by ID: skills, code, docs, patterns.

    Args:
        id: The HelixDB node ID of the task to fetch context for.
    """
    client = HelixClient()

    # Fetch the task node first
    task_node = await client.get_node("Task", id)
    if task_node is None:
        return json.dumps(
            {"error": f"Task {id} not found", "helix_url": client.base_url},
            indent=2,
        )

    # Run six traversals
    sections: dict[str, list[dict]] = {
        "skills": [],
        "code": [],
        "docs": [],
        "patterns": [],
        "dependents": [],
        "dependencies": [],
    }

    try:
        # Traverse outgoing edges for related resources
        outgoing_edges = await client.get_outgoing(id, "has_skill")
        for edge in outgoing_edges:
            target = edge.get("target", {})
            if target and isinstance(target, dict):
                sections["skills"].append(target)

        code_edges = await client.get_outgoing(id, "has_code")
        for edge in code_edges:
            target = edge.get("target", {})
            if target and isinstance(target, dict):
                sections["code"].append(target)

        doc_edges = await client.get_outgoing(id, "has_doc")
        for edge in doc_edges:
            target = edge.get("target", {})
            if target and isinstance(target, dict):
                sections["docs"].append(target)

        pattern_edges = await client.get_outgoing(id, "has_pattern")
        for edge in pattern_edges:
            target = edge.get("target", {})
            if target and isinstance(target, dict):
                sections["patterns"].append(target)

        # Traverse incoming / outgoing for dependents / dependencies
        inbound = await client.get_incoming(id, "depends_on")
        for edge in inbound:
            source = edge.get("source", {})
            if source and isinstance(source, dict):
                sections["dependents"].append(source)

        outbound = await client.get_outgoing(id, "depends_on")
        for edge in outbound:
            target = edge.get("target", {})
            if target and isinstance(target, dict):
                sections["dependencies"].append(target)

    except Exception as exc:
        return json.dumps(
            {
                "error": f"HelixDB query failed: {exc}",
                "helix_url": client.base_url,
                "task_id": id,
            },
            indent=2,
        )

    return json.dumps(
        {
            "task_id": id,
            "skills": sections["skills"],
            "code": sections["code"],
            "docs": sections["docs"],
            "patterns": sections["patterns"],
            "dependents": sections["dependents"],
            "dependencies": sections["dependencies"],
        },
        indent=2,
    )
