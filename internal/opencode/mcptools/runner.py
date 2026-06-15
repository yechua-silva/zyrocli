"""PydanticAI MCP server entry point — registers helix-integration tools.

Run with::

    uv run --directory mcp-tools runner.py

The server speaks the MCP stdio transport so OpenCode can register it as an
external MCP tool server in ``opencode.json``.
"""

from __future__ import annotations

from mcp.server.fastmcp import FastMCP

from search_code import search_code_tool
from search_skills import search_skills_tool
from task_context import task_context_tool
from helix_write import save_to_helix_tool, link_to_project_tool, find_project_tool

server = FastMCP("helix-integration")

server.add_tool(task_context_tool)
server.add_tool(search_code_tool)
server.add_tool(search_skills_tool)
server.add_tool(save_to_helix_tool)
server.add_tool(link_to_project_tool)
server.add_tool(find_project_tool)

if __name__ == "__main__":
    server.run(transport="stdio")
