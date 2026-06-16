"""
DEPRECATED: Writes handled by Go SDK.

The Python agent MUST NOT write to HelixDB. All write operations
are performed by the Go orchestrator after validating the agent's
AgentDecision output. This file is kept as a stub for backward
compatibility but should not be imported or used.

See: docs/spec-zyrov2.md (Agent-as-Validator pattern)
"""

from __future__ import annotations

import warnings

warnings.warn(
    "helix_write.py is deprecated. Writes handled by Go SDK.",
    DeprecationWarning,
    stacklevel=2,
)

__all__: list[str] = []
