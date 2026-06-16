"""Tests for agent.py — tools and configuration."""

import os

# Set dummy API key so the OpenAI client doesn't fail at import time
os.environ.setdefault("OPENAI_API_KEY", "test-key")

import pytest
from agent import zyro_agent


class TestAgentTools:
    @property
    def _tools(self):
        return zyro_agent._function_toolset.tools

    def test_has_search_code(self):
        assert "search_code" in self._tools

    def test_has_search_skills(self):
        assert "search_skills" in self._tools

    def test_has_task_context(self):
        assert "task_context" in self._tools

    def test_save_to_helix_readonly(self):
        tool = self._tools["save_to_helix"]
        assert tool.requires_approval is True
        assert tool.max_retries == 0
