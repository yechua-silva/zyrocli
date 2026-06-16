"""Capabilities and dependencies for Zyro agents."""

from __future__ import annotations

from dataclasses import dataclass, field


@dataclass
class HelixReadCapability:
    """Define qué puede leer el agente."""
    max_results: int = 10
    allowed_nodes: tuple[str, ...] = (
        "Project", "Technology", "Pattern", "Library",
        "Skill", "CodeNode", "Task", "Document",
        "Decision", "Fact",
    )


@dataclass
class AgentDependencies:
    """Dependencies injected into the PydanticAI agent."""
    read_cap: HelixReadCapability = field(default_factory=HelixReadCapability)
    phase: str = ""
    task_description: str = ""
    memory_context: str = ""
    boundari_phase: str | None = None
    request_id: str = ""
