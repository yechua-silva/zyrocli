# Investigación: gentle-ai v1.40.2 — Ecosystem Configurator

**Fecha:** 2026-06-16
**Fuente:** HelixDB nodes 3037, 3038

## Datos del proyecto
- **URL:** https://github.com/Gentleman-Programming/gentle-ai
- **Estrellas:** 4,000+
- **Versión:** v1.40.2 (216 releases)
- **Lenguaje:** Go 95.5%
- **Licencia:** MIT

## ¿Qué es?
"NO es un instalador de agentes. Es un configurador de ecosistema — toma cualquier
AI coding agent y lo supercarga con memoria persistente, SDD, skills, MCP servers,
provider switcher, persona teaching-oriented, y per-phase model assignment."

## 15 Agentes soportados
Claude Code, OpenCode, Kilo Code, Gemini CLI, Cursor, VS Code Copilot,
Codex, Windsurf, Antigravity, Kimi Code, Kiro IDE, Qwen Code, OpenClaw, Trae, Pi

## Capacidades clave
- **Engram:** memoria persistente cross-session (archivos + topic keys)
- **SDD/OpenSpec:** workflow completo con init→onboard→explore→...→archive
- **Per-phase model routing:** modelo distinto por fase SDD
- **Skill registry:** index-first skill discovery
- **Persona modes:** gentleman (Rioplatense) / neutral
- **Backup & rollback:** automático con dedup y pruning

## Diferencias con ZyroCLI

| Aspecto | gentle-ai | ZyroCLI |
|---------|-----------|---------|
| Enfoque | Configura agente existente | Orquestador propio (Go) |
| Memoria | Engram (archivos) | HelixDB (grafo+vectores+BM25) |
| Micro-ciclo | SDD lineal | Boomerang 6 pasos |
| Token counter | ❌ No tiene | ✅ internal/tokens/ |
| Medición | ❌ No tiene | ✅ zyro doctor --tokens |
| Servicios | ❌ No tiene | ✅ zyro services + TUI |
| GPU detection | ❌ No tiene | ✅ TUI + ROCm/Vulkan |
| MCP tools | Genéricas | search_facts, embed, task_context |
| Madurez | v1.40.2 (216 releases) | v2.0.4 (en desarrollo) |

## Referencias
- https://github.com/Gentleman-Programming/gentle-ai
- https://www.npmjs.com/package/gentle-pi
