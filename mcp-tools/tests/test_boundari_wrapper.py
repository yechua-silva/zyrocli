"""Tests for boundari_wrapper.py."""

import asyncio
import json
import os
from pathlib import Path

import pytest

from boundari_wrapper import BoundariWrapper


@pytest.fixture
def fake_tool():
    async def _tool(**kwargs):
        return "executed"
    return _tool


class TestBoundariWrapper:
    def test_f0_blocks_write(self, fake_tool):
        wrapper = BoundariWrapper(phase="F0")
        wrapped = wrapper.wrap_tool("write_file", fake_tool)

        with pytest.raises(PermissionError, match="denied"):
            asyncio.run(wrapped(path="test.txt"))

    def test_f0_allows_search(self, fake_tool):
        wrapper = BoundariWrapper(phase="F0")
        wrapped = wrapper.wrap_tool("search_code", fake_tool)

        result = asyncio.run(wrapped(query="test"))
        assert result == "executed"

    def test_f3_allows_write(self, fake_tool):
        wrapper = BoundariWrapper(phase="F3")
        wrapped = wrapper.wrap_tool("write_file", fake_tool)

        result = asyncio.run(wrapped(path="test.go"))
        assert result == "executed"

    def test_unlisted_tool_blocked(self, fake_tool):
        wrapper = BoundariWrapper(phase="F0")
        wrapped = wrapper.wrap_tool("unknown_tool", fake_tool)

        with pytest.raises(PermissionError, match="not in policy"):
            asyncio.run(wrapped())

    def test_audit_log(self, fake_tool):
        wrapper = BoundariWrapper(phase="F0", audit_dir="/tmp/zyro-test-audit")
        wrapped = wrapper.wrap_tool("search_code", fake_tool)
        asyncio.run(wrapped(query="test"))

        path = wrapper.save_audit()
        assert os.path.exists(path)

        with open(path) as f:
            line = f.readline()
            event = json.loads(line)
            assert event["tool"] == "search_code"
            assert event["allowed"] is True

        os.remove(path)

    def test_fallback_policy(self):
        wrapper = BoundariWrapper(phase="F0", policies_dir="/nonexistent")
        assert wrapper.policy is not None
        assert len(wrapper.policy.tools) > 0
