"""Tests for approval.py."""

from approval import ApprovalGate


class TestApprovalGate:
    def test_console_approve(self, monkeypatch):
        monkeypatch.setattr("builtins.input", lambda _: "y")
        gate = ApprovalGate(mode="console")
        from models import AgentDecision
        decision = AgentDecision(action="search", reasoning="x" * 10)
        assert gate.request_approval(decision, "F0") is True
    
    def test_console_reject(self, monkeypatch):
        monkeypatch.setattr("builtins.input", lambda _: "n")
        gate = ApprovalGate(mode="console")
        from models import AgentDecision
        decision = AgentDecision(action="search", reasoning="x" * 10)
        assert gate.request_approval(decision, "F0") is False
