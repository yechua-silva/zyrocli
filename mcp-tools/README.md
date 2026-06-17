# Helix MCP Tools — Python MCP Server for HelixDB

Exposes four MCP tools that query HelixDB from OpenCode or any MCP-compatible client:

| Tool | Description |
|------|-------------|
| `task_context(id)` | Returns full context for a task: skills, code, docs, patterns, dependents, dependencies |
| `search_code(query, limit)` | Text search over `CodeNode` entries |
| `search_facts(query, limit)` | Text search over `Fact` entries |
| `search_skills(query, limit)` | Text search over global `Skill` entries |

## Registration

Add the following entry to `~/.config/opencode/opencode.json` (replace `/path/to` with the absolute path to this directory):

```json
{
  "mcpTools": {
    "helix-integration": {
      "command": "uv",
      "args": ["run", "--directory", "/path/to/mcp-tools", "runner.py"]
    }
  }
}
```

After adding the entry, restart OpenCode. The four tools will be available as MCP tools.

## Requirements

- Python 3.10+
- `uv` package manager (installed by default with OpenCode)

## Running manually

```bash
uv run --directory /path/to/mcp-tools runner.py
```

The server listens on stdio (JSON-RPC messages) — the default MCP transport.
