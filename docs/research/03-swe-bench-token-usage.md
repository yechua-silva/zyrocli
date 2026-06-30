# Investigación: Token usage en SWE-bench (Anthropic, 2025)

**Fecha:** 2026-06-16
**Fuente:** https://www.anthropic.com/research/swe-bench-sonnet

## Hallazgos clave

1. **"Many successful runs took hundreds of turns and >100k tokens"**
2. El agente primero explora toda la estructura del repo (view directory)
3. Luego busca archivos específicos (cat -n)
4. El contexto se llena con código irrelevante
5. "The model kept trying until it ran out of context" (límite 200k tokens)

## El problema sin memoria

- Cada fase empieza desde cero
- El agente gasta tokens descubriendo lo que ya existe
- Contexto lleno de código irrelevante entre archivos que busca

## Cómo lo resuelve ZyroCLI + Boomerang

- MemoryStep: inyecta solo hechos relevantes (2k-4k tokens)
- Agente no necesita explorar codebase
- Hechos ya contienen decisiones, errores, preferencias
- SaveStep: escribe nuevos hechos para siguientes fases

## Diferencia clave

Sin HelixDB: agente en F2 necesita re-leer proyecto para entender F1
Con HelixDB: F2 recibe hechos de F1 ya filtrados

## Referencias
- https://www.anthropic.com/research/swe-bench-sonnet
- https://swebench.com/
- https://www.anthropic.com/research/building-effective-agents
