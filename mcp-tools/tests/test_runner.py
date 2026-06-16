"""Tests for runner.py — stdin/stdout JSON protocol."""

import json

from runner import error_output


class TestErrorOutput:
    def test_prints_json(self, capsys):
        error_output("test error message")
        captured = capsys.readouterr()
        data = json.loads(captured.out)
        assert data["type"] == "error"
        assert data["message"] == "test error message"
        assert data["protocol"] == "zyro-agent-v2"
