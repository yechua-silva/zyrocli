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
        # Traverse outgoing edges for related resources (returns nodes directly)
        sections["skills"] = await client.get_outgoing(id, "has_skill")
        sections["code"] = await client.get_outgoing(id, "has_code")
        sections["docs"] = await client.get_outgoing(id, "has_doc")
        sections["patterns"] = await client.get_outgoing(id, "has_pattern")

        # Traverse incoming / outgoing for dependents / dependencies
        sections["dependents"] = await client.get_incoming(id, "depends_on")
        sections["dependencies"] = await client.get_outgoing(id, "depends_on")

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
