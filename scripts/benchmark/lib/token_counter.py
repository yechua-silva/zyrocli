"""Token counter exacto usando tiktoken + Anthropic SDK.

Uso:
    from token_counter import count_tokens, TokenCounter
    counter = TokenCounter("gpt-4o")
    tokens = counter.count("texto a medir")
    print(counter.summary())
"""

from __future__ import annotations

import json
from typing import Any


class TokenCounter:
    """Contador de tokens exacto con soporte para OpenAI y Anthropic."""

    def __init__(self, model: str = "gpt-4o"):
        self.model = model
        self.log: list[dict[str, Any]] = []
        self._enc = None
        self._init_encoder()

    def _init_encoder(self):
        """Inicializa el encoder según el modelo."""
        try:
            import tiktoken
            if "gpt" in self.model or "o1" in self.model or "o3" in self.model:
                encoding = "o200k_base"
            elif "claude" in self.model:
                encoding = "cl100k_base"  # fallback para Claude via tiktoken
            else:
                encoding = "cl100k_base"
            self._enc = tiktoken.get_encoding(encoding)
            self.method = "tiktoken"
        except ImportError:
            try:
                from anthropic import Anthropic
                self._anthropic = Anthropic()
                self.method = "anthropic"
            except ImportError:
                self.method = "char_div_4"

    def count(self, text: str) -> int:
        """Cuenta tokens exactos de un texto."""
        if not text:
            return 0
        if self._enc:
            return len(self._enc.encode(text))
        elif hasattr(self, '_anthropic'):
            return self._anthropic.count_tokens(text)
        else:
            return (len(text) + 3) // 4  # fallback

    def count_messages(self, messages: list[dict]) -> int:
        """Cuenta tokens de una lista de mensajes (formato API)."""
        total = 0
        for msg in messages:
            content = msg.get("content", "")
            if isinstance(content, str):
                total += self.count(content)
            elif isinstance(content, list):
                for block in content:
                    if isinstance(block, dict):
                        total += self.count(block.get("text", ""))
        return total

    def log_request(self, role: str, content: str, metadata: dict | None = None):
        """Registra un turno en el log."""
        entry = {
            "role": role,
            "tokens": self.count(content),
            "chars": len(content),
            "model": self.model,
            "method": self.method,
        }
        if metadata:
            entry["metadata"] = metadata
        self.log.append(entry)

    def summary(self) -> dict:
        """Retorna resumen de tokens."""
        input_tokens = sum(
            t["tokens"] for t in self.log
            if t["role"] in ("system", "user", "tool_result")
        )
        output_tokens = sum(
            t["tokens"] for t in self.log
            if t["role"] in ("assistant", "tool_use")
        )
        return {
            "input_tokens": input_tokens,
            "output_tokens": output_tokens,
            "total_tokens": input_tokens + output_tokens,
            "turns": len(self.log),
            "method": self.method,
        }

    def save(self, path: str):
        """Guarda el log completo a un archivo JSON."""
        data = {
            "model": self.model,
            "method": self.method,
            "summary": self.summary(),
            "log": self.log,
        }
        with open(path, "w") as f:
            json.dump(data, f, indent=2)

    @staticmethod
    def load(path: str) -> dict:
        """Carga un log guardado."""
        with open(path) as f:
            return json.load(f)
