"""Boundari policy wrapper — controls tool execution per phase.

Loads YAML policies and wraps agent tools to enforce boundaries.
"""

from __future__ import annotations

import json
import os
import yaml
from dataclasses import dataclass, field
from datetime import datetime, UTC
from pathlib import Path
from typing import Any, Callable


@dataclass
class ToolRule:
    name: str
    action: str  # "allow" | "deny" | "require_approval"
    require_approval: bool = False
    conditions: dict[str, Any] = field(default_factory=dict)


@dataclass
class Budget:
    max_tool_calls: int = 50
    max_runtime_seconds: int = 300
    max_cost_usd: float = 0.0


@dataclass
class Policy:
    version: str = "1.0"
    phase: str = ""
    description: str = ""
    budget: Budget = field(default_factory=Budget)
    tools: list[ToolRule] = field(default_factory=list)


class BoundariWrapper:
    """Wraps agent tools to enforce phase-specific policies."""

    def __init__(
        self,
        phase: str,
        policies_dir: str | None = None,
        audit_dir: str | None = None,
    ):
        self.phase = phase
        self.policies_dir = policies_dir or str(
            Path(__file__).parent.parent / "internal" / "boundari"
        )
        self.audit_dir = audit_dir or str(
            Path.home() / ".zyro" / "audit"
        )
        self.tool_calls = 0
        self.start_time = datetime.now(UTC)
        self.audit_log: list[dict] = []
        self.policy = self._load_policy()

    def _load_policy(self) -> Policy:
        """Carga la política YAML para la fase actual."""
        # Strip "F" prefix if present (e.g. "F0" -> "0") to match filename
        phase_num = self.phase.lstrip("F")
        filename = f"phase{phase_num}-boundari.yaml"
        path = os.path.join(self.policies_dir, filename)

        if os.path.exists(path):
            with open(path) as f:
                data = yaml.safe_load(f)
            policy = Policy(
                version=data.get("version", "1.0"),
                phase=data.get("phase", self.phase),
                description=data.get("description", ""),
                budget=Budget(**data.get("budget", {})),
                tools=[ToolRule(**t) for t in data.get("tools", [])],
            )
        else:
            policy = self._fallback_policy()

        return policy

    def _fallback_policy(self) -> Policy:
        """Política hardcodeada por si no encuentra el YAML."""
        if self.phase == "F0":
            return Policy(
                phase="F0", description="Investigación (fallback)",
                budget=Budget(max_tool_calls=50, max_runtime_seconds=300),
                tools=[
                    ToolRule(name="search_code", action="allow"),
                    ToolRule(name="search_skills", action="allow"),
                    ToolRule(name="task_context", action="allow"),
                    ToolRule(name="write_file", action="deny"),
                    ToolRule(name="edit_file", action="deny"),
                    ToolRule(name="execute_command", action="deny"),
                ],
            )
        elif self.phase == "F3":
            return Policy(
                phase="F3", description="Implementación (fallback)",
                budget=Budget(max_tool_calls=200, max_runtime_seconds=1800),
                tools=[
                    ToolRule(name="write_file", action="allow"),
                    ToolRule(name="edit_file", action="allow"),
                    ToolRule(name="execute_command", action="require_approval"),
                ],
            )
        else:
            return Policy(
                phase=self.phase, description="Modo lectura (fallback)",
                tools=[
                    ToolRule(name="write_file", action="deny"),
                    ToolRule(name="edit_file", action="deny"),
                    ToolRule(name="execute_command", action="deny"),
                ],
            )

    def wrap_tool(
        self,
        tool_name: str,
        tool_func: Callable,
        raise_on_denied: bool = True,
    ) -> Callable:
        """Envuelve una tool con verificación de política."""

        async def wrapper(*args: Any, **kwargs: Any) -> Any:
            # Budget check
            if self.tool_calls >= self.policy.budget.max_tool_calls:
                self._audit(tool_name, False, "budget_exceeded")
                raise PermissionError(
                    f"Boundari: max tool calls exceeded ({self.policy.budget.max_tool_calls})"
                )

            elapsed = (datetime.now(UTC) - self.start_time).total_seconds()
            if elapsed > self.policy.budget.max_runtime_seconds:
                self._audit(tool_name, False, "timeout_exceeded")
                raise TimeoutError(
                    f"Boundari: max runtime exceeded ({self.policy.budget.max_runtime_seconds}s)"
                )

            # Find tool rule
            rule = None
            for r in self.policy.tools:
                if r.name == tool_name:
                    rule = r
                    break

            if rule is None:
                self._audit(tool_name, False, "not_in_policy")
                if raise_on_denied:
                    raise PermissionError(
                        f"Boundari: tool '{tool_name}' not in policy for phase {self.phase}"
                    )
                return None

            if rule.action == "deny":
                self._audit(tool_name, False, "denied")
                if raise_on_denied:
                    raise PermissionError(
                        f"Boundari: tool '{tool_name}' denied in phase {self.phase}"
                    )
                return None

            if rule.action == "require_approval" or rule.require_approval:
                self._audit(tool_name, False, "requires_approval")
                raise PermissionError(
                    f"Boundari: tool '{tool_name}' requires approval in phase {self.phase}"
                )

            # Allow
            self.tool_calls += 1
            self._audit(tool_name, True, "allowed")
            return await tool_func(*args, **kwargs) if tool_func else None

        return wrapper

    def _audit(self, tool: str, allowed: bool, reason: str) -> None:
        self.audit_log.append({
            "timestamp": datetime.now(UTC).isoformat(),
            "phase": self.phase,
            "tool": tool,
            "allowed": allowed,
            "reason": reason,
        })

    def save_audit(self, phase: str | None = None) -> str:
        """Guarda el log de auditoría como JSONL."""
        phase_name = phase or self.phase
        os.makedirs(self.audit_dir, exist_ok=True)
        filename = f"{phase_name}-{datetime.now(UTC).strftime('%Y%m%dT%H%M%S')}.jsonl"
        path = os.path.join(self.audit_dir, filename)

        with open(path, "w") as f:
            for event in self.audit_log:
                f.write(json.dumps(event) + "\n")

        return path
