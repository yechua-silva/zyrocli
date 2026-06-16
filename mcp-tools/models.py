"""Pydantic models for Zyro agent communication protocol."""

from __future__ import annotations

from enum import Enum
from pydantic import BaseModel, Field, model_validator


class Action(str, Enum):
    create = "create"
    update = "update"
    search = "search"
    skip = "skip"


class HelixNodeOutput(BaseModel):
    label: str = Field(..., min_length=1, description="HelixDB node label")
    properties: dict = Field(default_factory=dict)
    project_id: str | None = None
    requires_approval: bool = False

    model_config = {"extra": "forbid"}


class AgentDecision(BaseModel):
    action: Action
    reasoning: str = Field(..., min_length=10, description="Justificación de la decisión")
    nodes: list[HelixNodeOutput] = Field(default_factory=list)
    requires_approval: bool = False
    metadata: dict = Field(default_factory=dict)

    model_config = {"extra": "forbid"}


class ZyroAgentInput(BaseModel):
    protocol: str = "zyro-agent-v2"
    version: str = "2.0.0"
    request_id: str = ""
    phase: str = Field(..., description="Fase actual: F0-F4")
    task: str = Field(..., min_length=1)
    memory_context: str = ""
    boundari_phase: str | None = None
    timeout_seconds: int = 30
    read_cap: dict = Field(default_factory=lambda: {"max_results": 10})

    model_config = {"extra": "forbid"}


class HelixSearchResult(BaseModel):
    id: int
    label: str
    content: str
    score: float = 0.0
    source: str = "helixdb"

    model_config = {"extra": "forbid"}


class HelixReadInput(BaseModel):
    query: str = Field(..., min_length=1)
    limit: int = Field(default=10, ge=1, le=50)
    node_labels: list[str] = Field(default_factory=list)

    model_config = {"extra": "forbid"}
