"""Tests for models.py — Pydantic validation."""

import pytest
from pydantic import ValidationError

from models import (
    Action, AgentDecision, HelixNodeOutput, 
    HelixReadInput, ZyroAgentInput,
)


class TestAgentDecision:
    def test_valid_minimal(self):
        d = AgentDecision(action="search", reasoning="x" * 10)
        assert d.action == Action.search
        assert len(d.reasoning) >= 10
    
    def test_invalid_short_reasoning(self):
        with pytest.raises(ValidationError, match="String should have at least 10 characters"):
            AgentDecision(action="search", reasoning="short")


class TestZyroAgentInput:
    def test_valid_minimal(self):
        inp = ZyroAgentInput(phase="F0", task="analizar")
        assert inp.protocol == "zyro-agent-v2"
    
    def test_missing_required(self):
        with pytest.raises(ValidationError):
            ZyroAgentInput()


class TestHelixReadInput:
    def test_valid_defaults(self):
        inp = HelixReadInput(query="test")
        assert inp.limit == 10
    
    def test_invalid_limit(self):
        with pytest.raises(ValidationError):
            HelixReadInput(query="test", limit=0)


class TestHelixNodeOutput:
    def test_valid(self):
        n = HelixNodeOutput(label="Project", properties={"name": "test"})
        assert n.label == "Project"
    
    def test_empty_label(self):
        with pytest.raises(ValidationError):
            HelixNodeOutput(label="")
    
    def test_extra_fields_forbidden(self):
        with pytest.raises(ValidationError):
            HelixNodeOutput(label="Test", extra_field="bad")
