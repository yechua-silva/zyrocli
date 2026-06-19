# Investigación: OpenCode Commands, Skills y Custom Tools

## Fecha
2026-06-19

## Fuentes
- https://opencode.ai/docs/es/commands/
- https://opencode.ai/docs/es/skills/
- https://opencode.ai/docs/es/custom-tools/
- https://opencode.ai/docs/es/sdk/
- https://opencode.ai/docs/es/tui/

## Comandos (/commands)

### Definición en opencode.jsonc
```json
{
  "command": {
    "zyro-model": {
      "template": "zyrocli profile tui",
      "description": "Seleccionar modelos por fase SDD",
      "subtask": true,
      "agent": "model-assigner"
    }
  }
}
```

### Opciones disponibles
| Opción | Obligatorio | Descripción |
|--------|-------------|-------------|
| `template` | ✅ Sí | El prompt que se envía al LLM |
| `description` | ❌ No | Texto mostrado en el menú de comandos |
| `agent` | ❌ No | Qué agente ejecuta el comando |
| `subtask` | ❌ No | Si ejecuta como subagente (no contamina contexto) |
| `model` | ❌ No | Modelo específico para este comando |

### Argumentos
- `$ARGUMENTS` - todos los argumentos
- `$1`, `$2`, `$3` - argumentos posicionales
- `!comando` - output de shell
- `@archivo` - referencia a archivo

### Comandos vs Skills
| Aspecto | Commands | Skills |
|---------|----------|--------|
| Inicio | `/nombre` | Llamada por agente |
| Visibilidad | TUI (autocomplete) | System prompt (<available_skills>) |
| Uso | Acción directa del usuario | Instrucción contextual para el LLM |

## Skills (/skills)

### Formato SKILL.md
```yaml
---
name: zyro-pre-f0
description: PRE-F0: Alineación de dominio — grill-me, domain-model, triage, improve-arch
---
```

### Ubicaciones de skills (orden de búsqueda)
1. Proyecto: `.opencode/skills/<name>/SKILL.md`
2. Global: `~/.config/opencode/skills/<name>/SKILL.md`
3. Claude proyecto: `.claude/skills/<name>/SKILL.md`
4. Claude global: `~/.claude/skills/<name>/SKILL.md`
5. Agents proyecto: `.agents/skills/<name>/SKILL.md`
6. Agents global: `~/.agents/skills/<name>/SKILL.md`

### Reglas de frontmatter
- `name` (obligatorio): 1-64 chars, alfanumérico + guiones, match directory name
- `description` (obligatorio): 1-1024 chars, debe ser específica
- `license` (opcional)
- `compatibility` (opcional)
- `metadata` (opcional, map string→string)

### Permisos de skills
```json
"permission": { "skill": { "*": "allow", "internal-*": "deny" } }
```

## Custom Tools (/custom-tools)

### Definición (TypeScript/JavaScript)
```typescript
import { tool } from "@opencode-ai/plugin"
export default tool({
  description: "Tool description",
  args: { param: tool.schema.string().describe("Parameter") },
  async execute(args, context) {
    return "result"
  },
})
```

### Ubicaciones
- Proyecto: `.opencode/tools/<name>.ts`
- Global: `~/.config/opencode/tools/<name>.ts`

### Características
- El nombre del archivo = nombre de la herramienta
- Múltiples exports = múltiples herramientas: `math_add`, `math_multiply`
- Pueden sobreescribir tools nativas (ej: bash)
- Contexto disponible: agent, sessionID, messageID, directory, worktree
- Pueden ejecutar scripts en cualquier lenguaje

## SDK (@opencode-ai/sdk)

### API relevante para /zyro-model
```typescript
// Obtener proveedores y modelos
const { providers, default: defaults } = await client.config.providers()

// Interacción con TUI
await client.tui.showToast({ body: { message: "texto", variant: "success" } })
await client.tui.appendPrompt({ body: { text: "texto" } })

// Ejecutar comando
await client.tui.executeCommand({ body: { command: "zyro-model" } })
```

## Conclusión para /zyro-model

El camino correcto para implementar /zyro-model es:

1. **El comando `/zyro-model`** ya está registrado y apunta al agente `model-assigner`
2. **El agente `model-assigner`** debe usar la tool `question` para guiar al usuario:
   - Mostrar proveedores disponibles
   - Por cada agente SDD: seleccionar provider → modelo
   - Escribir la configuración en opencode.jsonc
3. **No necesita un modal TUI** - funciona dentro del chat de OpenCode
4. **Opcional**: Un plugin .tsx podría mostrar un modal más visual, pero no es necesario
