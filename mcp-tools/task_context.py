"""Task context retrieval — async function, not MCP tool."""

from __future__ import annotations

from helix_client import HelixClient


async def get_task_context(task_id: int, project_id: str | None = None) -> str:
    """Fetch full task context from HelixDB: skills, code, docs, patterns."""
    client = HelixClient(project_id=project_id)
    result_parts = []

    # Get the task node
    task = await client.get_node("Task", task_id)
    if not task:
        return f"Task {task_id} not found"
    result_parts.append(f"Task ID: {task_id}")

    # Traverse outgoing edges
    skills = await client.get_outgoing(task_id, "REQUIRES_SKILL")
    if skills:
        result_parts.append(f"Skills: {len(skills)}")

    code_nodes = await client.get_outgoing(task_id, "REFERENCES")
    if code_nodes:
        result_parts.append(f"Code nodes: {len(code_nodes)}")

    return "\n".join(result_parts) if result_parts else f"Task {task_id}: no context"
