# Investigación: Cómo medir tokens realmente en coding agents

**Fecha:** 2026-06-16
**Fuente:** HelixDB node 3039

## Herramientas de tokenización

| Herramienta | Precisión | Dependencias | Modelos |
|------------|-----------|-------------|---------|
| **tiktoken** (OpenAI, 18.5k ⭐) | **EXACTA** (tokenizer real) | `pip install tiktoken` | GPT-4o, GPT-4, GPT-3 |
| **Anthropic SDK** | **EXACTA** | `pip install anthropic` | Claude 3/4 |
| **Nuestro `internal/tokens.Count()`** | ±20% | 0 (Go stdlib) | Aproximación |

## El problema

OpenCode, Claude Code, gentle-ai **NO exponen** cuántos tokens usan por turno.
No hay API como `get_last_token_count()`.

## La solución: Proxy MITM

```
Agente → Proxy HTTP → API (Anthropic/OpenAI)
             ↓
        Captura: system prompt + mensajes + respuestas
        Cuenta: tiktoken EXACTO
        Guarda: log JSON por turno
```

## Referencias
- https://github.com/openai/tiktoken — BPE tokenizer oficial (MIT)
- https://github.com/openai/openai-cookbook — How to count tokens with tiktoken
- https://platform.openai.com/tokenizer — Visualizador online
