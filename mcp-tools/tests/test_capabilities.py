"""Tests for capabilities.py."""

from capabilities import AgentDependencies, HelixReadCapability


class TestHelixReadCapability:
    def test_defaults(self):
        cap = HelixReadCapability()
        assert cap.max_results == 10
        assert "Project" in cap.allowed_nodes
        assert "Skill" in cap.allowed_nodes
    
    def test_custom(self):
        cap = HelixReadCapability(max_results=5, allowed_nodes=("CodeNode",))
        assert cap.max_results == 5
        assert cap.allowed_nodes == ("CodeNode",)


class TestAgentDependencies:
    def test_defaults(self):
        deps = AgentDependencies()
        assert deps.read_cap.max_results == 10
    
    def test_custom(self):
        deps = AgentDependencies(
            phase="F0", task_description="test", boundari_phase="F0"
        )
        assert deps.phase == "F0"
        assert deps.boundari_phase == "F0"
