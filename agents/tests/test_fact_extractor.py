"""Tests for fact_extractor.py"""

import json
import sys
from pathlib import Path

# Asegurar que agents/ está en el path
sys.path.insert(0, str(Path(__file__).parent.parent))

from fact_extractor import extract_facts, extract_facts_llm


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


class TestExtractFactsExtra:
    """Tests adicionales B6+D2 — tareas, límites, LLM fallback."""

    def test_extract_task(self):
        """"TODO: refactor this" detecta tipo "task"."""
        facts = extract_facts("TODO: refactor this module ASAP", "F1")
        tasks = [f for f in facts if f["type"] == "task"]
        assert len(tasks) >= 1
        assert "refactor" in tasks[0]["content"]

    def test_extract_task_fixme(self):
        """"# fixme: bug crítico" detecta tipo "task"."""
        facts = extract_facts("# fixme: bug crítico en el login", "F2")
        tasks = [f for f in facts if f["type"] == "task"]
        assert len(tasks) >= 1
        assert "fixme" in tasks[0]["content"].lower()

    def test_ignore_common_todo_spanish(self):
        """"todo el código" NO detecta task (porque "todo" suelto no es task)."""
        facts = extract_facts("todo el código está bien documentado", "F0")
        tasks = [f for f in facts if f["type"] == "task"]
        assert len(tasks) == 0, (
            f"'todo' sin ':' ni '#' no debería detectarse como task. "
            f"Se encontraron: {[f['content'] for f in tasks]}"
        )

    def test_case_insensitive(self):
        """"ERROR en el server" detecta error (case insensitive)."""
        facts = extract_facts("ERROR en el server de producción", "F3")
        errors = [f for f in facts if f["type"] == "error"]
        assert len(errors) >= 1
        assert "ERROR" in errors[0]["content"] or "error" in errors[0]["content"].lower()

    def test_word_boundary_error(self):
        """"error404" NO detecta error (por el \\b en el patrón)."""
        facts = extract_facts("error404 en la página de resultados", "F0")
        errors = [f for f in facts if f["type"] == "error"]
        assert len(errors) == 0, (
            f"'error404' no debería coincidir con \\berror\\b. "
            f"Se encontraron: {[f['content'] for f in errors]}"
        )

    def test_llm_mode_fallback(self):
        """Cuando Ollama no está, extract_facts_llm devuelve [] sin crashear."""
        facts = extract_facts_llm("test input", "F0", "non-existent-model")
        assert facts == [], (
            "Si Ollama no responde debe retornar lista vacía, "
            f"no crashear. Se obtuvo: {facts}"
        )
