"""Approval gates for human-in-the-loop decisions.

Two modes:
  - "console": prints decision details and waits for stdin "y/n".
  - "go_bridge": prints JSON approval request to stdout and reads
    JSON response from stdin (for Go orchestrator integration).
"""

from __future__ import annotations

import json
import sys
from dataclasses import dataclass

from models import AgentDecision


@dataclass
class ApprovalGate:
    """Human-in-the-loop approval gate for agent decisions.

    Attributes:
        mode: "console" for interactive terminal, "go_bridge" for Go IPC.
    """

    mode: str = "go_bridge"

    def request_approval(self, decision: AgentDecision, phase: str) -> bool:
        """Request human approval for a decision.

        Args:
            decision: The AgentDecision to be approved.
            phase: Current pipeline phase (F0-F4).

        Returns:
            True if approved, False otherwise.
        """
        if self.mode == "console":
            return self._console_approver(decision, phase)
        return self._go_bridge_approver(decision, phase)

    def _console_approver(self, decision: AgentDecision, phase: str) -> bool:
        """Interactive terminal approver — prints decision, reads y/n."""
        print(f"\n=== Aprobación requerida [Fase {phase}] ===")
        print(f"Acción: {decision.action.value}")
        print(f"Razonamiento: {decision.reasoning}")
        print(f"Nodos: {len(decision.nodes)}")
        respuesta = input("¿Aprobar? (y/n): ").strip().lower()
        return respuesta in ("y", "yes", "s", "si")

    def _go_bridge_approver(self, decision: AgentDecision, phase: str) -> bool:
        """Go bridge approver — prints JSON request, reads JSON response.

        Protocol:
            stdout: {"type": "approval_request", "phase": ..., ...}
            stdin:  {"approved": true/false}
        """
        approval_request = {
            "type": "approval_request",
            "phase": phase,
            "action": decision.action.value,
            "reasoning": decision.reasoning,
            "node_count": len(decision.nodes),
            "requires_approval": decision.requires_approval,
        }
        print(json.dumps(approval_request))
        line = sys.stdin.readline().strip()
        try:
            response = json.loads(line)
            return response.get("approved", False)
        except (json.JSONDecodeError, KeyError):
            return False
