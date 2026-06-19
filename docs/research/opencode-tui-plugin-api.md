# Investigación: OpenCode TUI Plugin API

## Fecha
2026-06-19

## Fuentes
- https://opencode.ai/docs/es/plugins/
- https://opencode.ai/docs/es/tui/
- https://opencode.ai/docs/es/sdk/
- Código fuente de sub-agent-statusline (Joaquinvesapa/sub-agent-statusline)
- Código fuente de opencode-sdd-engram-manage (j0k3r-dev-rgl/sdd-engram-plugin)

## Diferencia crítica: Server Plugin vs TUI Plugin

OpenCode tiene DOS sistemas de plugins distintos:

| Aspecto | Server Plugin | TUI Plugin |
|---------|--------------|------------|
| Import | `@opencode-ai/plugin` | `@opencode-ai/plugin/tui` |
| Recibe | `{ client, $, directory }` | `(api: TuiPluginApi, options, meta)` |
| Puede mostrar modales? | ❌ NO | ✅ SÍ |
| Puede registrar `/commands`? | ❌ NO | ✅ SÍ |
| Archivo ejemplo | `engram.ts`, `model-variants.ts` | (ninguno en nuestro proyecto) |

## Estructura de un TUI Plugin

```typescript
import type { TuiPlugin } from "@opencode-ai/plugin/tui"

export const ZyroModel: TuiPlugin = (api, options, meta) => {
  // api: TuiPluginApi - interactúa con OpenCode
  // options: PluginOptions - config del plugin
  // meta: TuiPluginMeta - metadatos
}
```

### package.json (peerDependencies)
```json
{
  "type": "module",
  "main": "./dist/tui.js",
  "exports": { "./tui": { "import": "./dist/tui.js" } },
  "peerDependencies": {
    "@opencode-ai/plugin": ">=1.14.48",
    "@opentui/core": ">=0.2.8 <1",
    "@opentui/keymap": ">=0.2.8 <1",
    "@opentui/solid": ">=0.2.8 <1",
    "solid-js": "1.9.12"
  }
}
```

## API de Diálogos Modales

### DialogSelect — EL COMPONENTE CLAVE para selector provider→modelo
```typescript
api.ui.DialogSelect({
  title: "Select Provider",
  options: [
    { title: "Anthropic", value: "anthropic", description: "10 models" },
    { title: "OpenAI", value: "openai", description: "25 models" }
  ],
  onSelect: (opt) => { /* opt.value = "anthropic" */ }
})
```

### Otros diálogos
- `api.ui.DialogAlert` - Alerta simple
- `api.ui.DialogConfirm` - Confirmación OK/Cancel
- `api.ui.DialogPrompt` - Input de texto

### Dialog Stack (navegación multi-paso)
```typescript
api.ui.dialog.replace(() => <Componente />)  // Reemplazar diálogo actual
api.ui.dialog.clear()                          // Cerrar todos
api.ui.dialog.setSize("large")                 // Cambiar tamaño
```

## Registrar comando `/zyro-model`

### API moderna (keymap):
```typescript
api.keymap.registerLayer({
  priority: 100,
  commands: [{
    name: ":zyro-model",
    title: "Zyro Model Config",
    desc: "Configure models per Zyro-SDD agent",
    category: "Zyro",
    nargs: "0",
    run: () => { openModelConfigDialog(api); return true; }
  }],
  bindings: [{ key: "alt+m", cmd: ":zyro-model" }]
});
```

### API legacy (command):
```typescript
api.command.register(() => [{
  title: "Zyro Model Config",
  value: "zyro-model",
  category: "Zyro",
  slash: { name: "zyro-model", aliases: ["zm"] },
  onSelect: () => openModelConfigDialog(api)
}])
```

## Leer providers y modelos
```typescript
// providers = array de { id, name, models: Record<string, {name, limit}> }
const providers = api.state.provider
// o via SDK:
const { data } = await api.client.config.providers()
```

## Leer y escribir configuración
```typescript
// Leer config actual
const config = api.state.config
const agentModel = config.agent?.["sdd-orchestrator"]?.model

// Escribir en runtime (SIN REINICIO)
await api.client.global.config.update({
  config: { ...config, agent: { ...config.agent, "sdd-orchestrator": { model: "anthropic/claude-sonnet-4" } } }
})
```

## Slots disponibles para UI persistente
```typescript
api.slots.register({
  name: "home_bottom", // sidebar_content, session_prompt_right, etc.
  render: (props) => <div>Status</div>
})
```

## Registro de rutas personalizadas
```typescript
api.route.register([{
  name: "zyro-config",
  render: ({ params }) => <div>Page</div>
}])
api.route.navigate("zyro-config", { agent: "sdd-apply" })
```

## Build (tsup + esbuild-plugin-solid)
```typescript
import { defineConfig } from "tsup";
import solidPlugin from "esbuild-plugin-solid";
export default defineConfig({
  entry: ["src/index.tsx"],
  format: "esm",
  clean: true,
  dts: false,
  external: ["@opencode-ai/plugin", "@opentui/core", "@opentui/solid", "solid-js"],
  esbuildPlugins: [solidPlugin()],
});
```

## Instalación del plugin
En `~/.config/opencode/tui.json`:
```json
{
  "plugin": ["opencode-zyro-model"]
}
```
O como archivo local en `~/.config/opencode/plugins/zyro-model.ts` (carga automática).
