"""Tests for embedding_harness.py — unit tests for BM25 fallback and helpers."""
import math
import sys
from pathlib import Path

# Add mcp-tools to path
sys.path.insert(0, str(Path(__file__).parent.parent))

from embedding_harness import _bm25_fallback


class TestBM25Fallback:
    """Tests for _bm25_fallback — BM25 pseudo-embedding de 1024 dims."""

    def test_bm25_returns_vector(self):
        """_bm25_fallback devuelve una lista."""
        result = _bm25_fallback("hello world")
        assert isinstance(result, list)

    def test_bm25_1024_dims(self):
        """El vector tiene exactamente 1024 dimensiones."""
        result = _bm25_fallback("hello world")
        assert len(result) == 1024

    def test_bm25_not_all_zeros(self):
        """Para texto normal, no todos los valores son 0."""
        result = _bm25_fallback("hello world this is a test with multiple tokens")
        non_zero = sum(1 for v in result if v != 0.0)
        assert non_zero > 0, "Se esperaba al menos un valor distinto de cero"

    def test_bm25_empty_text(self):
        """Texto vacío retorna vector de 1024 ceros."""
        result = _bm25_fallback("")
        assert len(result) == 1024
        assert all(v == 0.0 for v in result)

    def test_bm25_deterministic(self):
        """Mismo input produce exactamente el mismo vector."""
        text = "deterministic test input for embedding"
        r1 = _bm25_fallback(text)
        r2 = _bm25_fallback(text)
        assert r1 == r2

    def test_bm25_different_inputs(self):
        """Inputs distintos producen vectores distintos."""
        r1 = _bm25_fallback("the cat sat on the mat")
        r2 = _bm25_fallback("the dog ran in the park")
        assert r1 != r2

    def test_bm25_unit_norm(self):
        """La norma L2 del vector es aproximadamente 1.0 (tolerancia 0.01)."""
        result = _bm25_fallback("some text to compute l2 norm for testing")
        norm = math.sqrt(sum(v * v for v in result))
        assert abs(norm - 1.0) < 0.01, f"Norma L2 = {norm}, se esperaba ~1.0"

    def test_bm25_single_token(self):
        """Un solo token también produce vector normalizado."""
        result = _bm25_fallback("python")
        norm = math.sqrt(sum(v * v for v in result))
        assert abs(norm - 1.0) < 0.01
        non_zero = sum(1 for v in result if v != 0.0)
        assert non_zero > 0

    def test_bm25_repeated_words(self):
        """Palabras repetidas (BM25 tf saturation) sigue siendo válido."""
        result = _bm25_fallback("test " * 50)
        assert len(result) == 1024
        norm = math.sqrt(sum(v * v for v in result))
        assert abs(norm - 1.0) < 0.01

    def test_bm25_spanish_text(self):
        """Texto en español funciona correctamente."""
        result = _bm25_fallback("esto es una prueba del sistema de embeddings en español")
        assert len(result) == 1024
        norm = math.sqrt(sum(v * v for v in result))
        assert abs(norm - 1.0) < 0.01

    def test_bm25_special_chars(self):
        """Caracteres especiales no causan errores."""
        result = _bm25_fallback("hello! @world #python 3.14 $100 &more")
        assert len(result) == 1024
        norm = math.sqrt(sum(v * v for v in result))
        assert abs(norm - 1.0) < 0.01

    def test_bm25_unicode(self):
        """Texto con Unicode funciona correctamente."""
        result = _bm25_fallback("café français 中文 español русский")
        assert len(result) == 1024
        norm = math.sqrt(sum(v * v for v in result))
        assert abs(norm - 1.0) < 0.01
