# Investigación: gentle-ai SDD Profiles para OpenCode

## Fecha
2026-06-19

## Fuente
https://github.com/Gentleman-Programming/gentle-ai
https://github.com/Gentleman-Programming/gentle-ai/blob/main/docs/opencode-profiles.md

## ¿Cómo funciona?
gentle-ai permite crear PERFILES NOMBRADOS de configuración de modelos para OpenCode.
Cada perfil asigna un modelo diferente a cada fase SDD (orchestrator, explore, spec, design, tasks, apply, verify, archive).

## Mecanismo

### Creación de perfiles
- Via TUI: `gentle-ai` → "OpenCode SDD Profiles" → Create → nombre → provider/model por fase
- Via CLI: `gentle-ai sync --profile cheap:anthropic/claude-haiku-3.5-20241022`
- Via CLI por fase: `gentle-ai sync --profile-phase cheap:sdd-apply:anthropic/claude-sonnet-4-20250514`

### Cómo se almacenan
- Cada perfil genera 11 entradas de agente en opencode.json:
  - 1 orchestrator: `sdd-orchestrator-{name}` (mode: primary)
  - 10 sub-agentes: `sdd-{phase}-{name}` (mode: subagent, hidden)
- Base conductor: `gentle-orchestrator` (canonical)
- Sub-agent prompts compartidos via `{file:~/.config/opencode/prompts/sdd/sdd-apply.md}`
- Solo el `model` difiere entre perfiles

### Cómo se usan en OpenCode
- El usuario presiona **Tab** en OpenCode para cambiar entre perfiles
- Cada perfil aparece como un orchestrator seleccionable
- Todos los comandos `/sdd-*` apuntan al orchestrator activo
- El orchestrador delega a sus sub-agentes con sufijo (e.g., `sdd-apply-cheap`)

### Diferencia con lo que queremos
- gentle-ai crea MULTIPLES perfiles nombrados y el usuario switchea con Tab
- Nuestro /zyro-model necesita un solo perfil configurable dentro de OpenCode
- Ambos enfoques son compatibles: podríamos tener /zyro-model configurando el perfil activo
