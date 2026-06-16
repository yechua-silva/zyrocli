"""Zyro agent runner — Agent-as-Validator pattern.

Reads ZyroAgentInput from stdin, runs the PydanticAI agent,
outputs AgentDecision as JSON to stdout.

Usage:
    echo '<json>' | uv run --directory mcp-tools python runner.py
"""

from __future__ import annotations

import asyncio
import json
import sys
import traceback

from capabilities import AgentDependencies, HelixReadCapability
from models import AgentDecision, ZyroAgentInput


async def main() -> None:
    """Read input, run agent, output decision."""
    line = sys.stdin.readline()
    if not line:
        error_output("No input received on stdin")
        sys.exit(1)

    try:
        input_data = ZyroAgentInput.model_validate_json(line)
    except Exception as e:
        error_output(f"Invalid input: {e}")
        sys.exit(1)

    deps = AgentDependencies(
        read_cap=HelixReadCapability(
            max_results=input_data.read_cap.get("max_results", 10)
        ),
        phase=input_data.phase,
        task_description=input_data.task,
        memory_context=input_data.memory_context,
        boundari_phase=input_data.boundari_phase,
        request_id=input_data.request_id,
    )

    try:
        from agent import run_agent

        decision = await run_agent(input_data.task, deps)
        print(decision.model_dump_json())
    except Exception as e:
        error_output(f"Agent error: {e}\n{traceback.format_exc()}")
        sys.exit(1)


def error_output(message: str) -> None:
    """Print error as JSON to stdout for the Go orchestrator."""
    print(
        json.dumps(
            {
                "type": "error",
                "message": message,
                "protocol": "zyro-agent-v2",
            }
        )
    )


if __name__ == "__main__":
    asyncio.run(main())
