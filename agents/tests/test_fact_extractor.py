"""Tests for fact_extractor.py"""

import json
import sys
from pathlib import Path

# Asegurar que agents/ está en el path
sys.path.insert(0, str(Path(__file__).parent.parent))

from fact_extractor import extract_facts


class TestExtractFacts:
    def test_extract_decision(self):
        facts = extract_facts("decidimos usar mxbai-embed-large para embeddings", "F1")
        decisions = [f for f in facts if f["type"] == "decision"]
        assert len(decisions) >= 1
        assert "mxbai-embed-large" in decisions[0]["content"]

    def test_extract_error(self):
        facts = extract_facts("hay un error en el cliente HTTP: conexión fallida", "F2")
        errors = [f for f in facts if f["type"] == "error"]
        assert len(errors) >= 1

    def test_extract_preference(self):
        facts = extract_facts("prefiero mxbai-embed-large sobre nomic-embed-text", "F0")
        prefs = [f for f in facts if f["type"] == "preference"]
        assert len(prefs) >= 1

    def test_no_match(self):
        facts = extract_facts("el sol brilla en el cielo hoy", "F0")
        assert len(facts) == 0

    def test_short_content_filtered(self):
        facts = extract_facts("error x", "F0")
        assert len(facts) == 0  # menos de 15 caracteres

    def test_output_format(self):
        facts = extract_facts("decidimos usar Go", "F0")
        for f in facts:
            assert "type" in f
            assert "content" in f
            assert "confidence" in f
            assert "source" in f
            assert f["phase"] == "F0"
