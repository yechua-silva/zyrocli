#!/usr/bin/env python3
"""Fact Extractor — extrae hechos atómicos de logs de conversación.

Uso:
    python fact_extractor.py --input <log.json> --phase F1
    cat log.json | python fact_extractor.py --phase F1

Output JSON:
    {{"facts": [{{"type": "decision", "content": "...", "confidence": 0.9, ...}}]}}
"""

import argparse
import json
import re
import sys
from datetime import datetime, timezone
from typing import Any

# Patrones para detectar tipos de hechos
FACT_PATTERNS: dict[str, list[str]] = {
    "task": [
        r"#\s*todo",
        r"#\s*fixme",
        r"todo:",
        r"fixme:",
    ],
    "decision": [
        r"vamos a usar",
        r"decidimos",
        r"hemos decidido",
        r"voy a implementar",
        r"la solución es",
        r"se opta por",
        r"elegimos",
        r"se adopta",
    ],
    "error": [
        r"\berror\b",
        r"\bbug\b",
        r"\bfallo\b",
        r"\bfalla\b",
        r"\bexcepción\b",
        r"\bcrash\b",
        r"no funciona",
        r"tira error",
    ],
    "preference": [
        r"\bprefiero\b",
        r"mejor usar",
        r"en lugar de",
        r"no me gusta",
        r"preferiría",
    ],
    "pattern": [
        r"patrón",
        r"arquitectura",
        r"design pattern",
        r"repository pattern",
        r"arquitectura hexagonal",
        r"clean architecture",
    ],
    "dependency": [
        r"dependemos de",
        r"\brequiere\b",
        r"\bnecesita\b",
        r"depende de",
        r"tiene como dependencia",
    ],
    "observation": [
        r"observo",
        r"noto",
        r"detecto",
        r"veo que",
        r"encuentro que",
        r"hay un",
        r"existe un",
        r"se encuentra",
    ],
}


def extract_facts(log_text: str, phase: str) -> list[dict[str, Any]]:
    """Extrae hechos atómicos del texto de log usando coincidencia de patrones."""
    facts: list[dict[str, Any]] = []
    seen: set[str] = set()

    for fact_type, patterns in FACT_PATTERNS.items():
        for pattern in patterns:
            matches = re.finditer(pattern, log_text, re.IGNORECASE)
            for match in matches:
                start = max(0, match.start() - 50)
                end = min(len(log_text), match.end() + 100)
                context = log_text[start:end].strip()

                # Deduplicar por contexto similar
                if context in seen:
                    continue
                seen.add(context)

                # Filtro: mínimo 15 caracteres de contenido
                if len(context) < 15:
                    continue

                facts.append({
                    "type": fact_type,
                    "content": context,
                    "confidence": 0.8,
                    "salience": 0.7,
                    "source": "extractor:pattern",
                    "phase": phase,
                    "decay_rate": 0.05,
                    "created_at": datetime.now(timezone.utc).isoformat(),
                })

    return facts


def extract_facts_llm(log_text: str, phase: str, model: str = "") -> list[dict[str, Any]]:
    """Extrae hechos usando un LLM local (Ollama) para mejor precisión.

    Requiere: ollama running en localhost:11434
    """
    try:
        import httpx  # type: ignore[import-untyped]

        prompt = f"""Extrae hechos atómicos de esta conversación de desarrollo.
Para cada hecho, indica: tipo (decision|error|preference|pattern|dependency|observation),
contenido, confianza (0-1).

Conversación:
{log_text[:2000]}

Output JSON: {{"facts": [...]}}
"""
        response = httpx.post(
            "http://localhost:11434/api/generate",
            json={
                "model": model or "llama3.2",
                "prompt": prompt,
                "stream": False,
                "format": "json",
            },
            timeout=30,
        )

        if response.status_code == 200:
            data = response.json()
            text = data.get("response", "{}")
            parsed = json.loads(text)
            facts = parsed.get("facts", [])
            for f in facts:
                f.setdefault("source", "extractor:llm")
                f.setdefault("phase", phase)
                f.setdefault("salience", 0.7)
                f.setdefault("decay_rate", 0.05)
            return facts
    except Exception as e:
        print(f"[fact_extractor] Error en LLM mode: {e}", file=sys.stderr)
        return []

    return []


def main() -> None:
    parser = argparse.ArgumentParser(description="Extrae hechos de logs")
    parser.add_argument("--input", "-i", help="Archivo JSON de entrada (si no, lee stdin)")
    parser.add_argument("--phase", "-p", required=True, help="Fase actual (F0-F4)")
    parser.add_argument("--llm", "-l", action="store_true", help="Usar LLM local para mejor precisión")
    parser.add_argument("--model", "-m", default="llama3.2", help="Modelo Ollama para LLM")
    args = parser.parse_args()

    if args.input:
        with open(args.input) as f:
            data = json.load(f)
    else:
        data = json.load(sys.stdin)

    log_text = ""
    if isinstance(data, dict):
        log_text = data.get("conversation", data.get("log", json.dumps(data)))
    else:
        log_text = str(data)

    if args.llm:
        facts = extract_facts_llm(log_text, args.phase, args.model)
    else:
        facts = extract_facts(log_text, args.phase)

    print(json.dumps({"facts": facts}, indent=2))


if __name__ == "__main__":
    main()
