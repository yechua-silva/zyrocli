# Investigación: opencode-sdd-engram-manage (plugin de referencia)

## Fecha
2026-06-19

## Fuente
- https://github.com/j0k3r-dev-rgl/sdd-engram-plugin
- https://www.npmjs.com/package/opencode-sdd-engram-manage

## Qué hace
Plugin TUI de OpenCode que permite gestionar perfiles SDD y asignar modelos a cada fase.
Es el plugin que gentle-ai recomienda para configurar modelos por fase SDD.

## Cómo se instala
```json
// ~/.config/opencode/tui.json
{ "plugin": ["opencode-sdd-engram-manage"] }
```
OpenCode lo instala automáticamente desde npm.

## Cómo se abre
- Atajo: `Alt+K` (Linux/Win) o `Cmd+K` (macOS)
- Comando: `/sdd-model`

## Flujo de selección de modelos (2 pasos)

### Paso 1: Provider Picker
```tsx
<api.ui.DialogSelect
  title={`Provider for ${agentName}`}
  options={providers.map(p => ({
    title: p.name || p.id,
    value: p.id,
    description: `${Object.keys(p.models).length} models available`
  }))}
  onSelect={(opt) => showModelPicker(api, agentName, provider)}
/>
```

### Paso 2: Model Picker
```tsx
<api.ui.DialogSelect
  title={`${provider.name} › ${agentName}`}
  options={Object.keys(models).map(key => ({
    title: models[key].name || key,
    value: `${provider.id}/${key}`,
    description: formatContext(models[key].limit?.context)
  }))}
  onSelect={(opt) => updateAgentModel(api, agentName, opt.value)}
/>
```

## Cómo guarda los perfiles
Archivos JSON en `~/.config/opencode/profiles/<nombre>.json`:
```json
{
  "models": {
    "sdd-init": "anthropic/claude-sonnet-4-6",
    "sdd-spec": "anthropic/claude-haiku-4",
    "sdd-apply": "openai/gpt-4o"
  }
}
```

## Cómo activa en runtime (sin reinicio)
```typescript
const result = await api.client.global.config.update({ config: finalConfig })
fs.writeFileSync(configPath, JSON.stringify(finalConfig, null, 2))
```

## APIs de OpenCode utilizadas
| API | Uso |
|---|---|
| `api.state.config` | Leer config actual |
| `api.state.provider` | Listar providers con modelos |
| `api.client.global.config.update()` | Actualizar config en runtime |
| `api.ui.dialog.replace()` | Mostrar diálogos modales |
| `api.ui.dialog.clear()` | Cerrar diálogos |
| `api.ui.toast()` | Notificaciones |
| `api.keymap.registerLayer()` | Registrar atajos de teclado |
| `api.command.register()` | Registrar slash commands |
| `api.slots.register()` | Renderizar en slots de UI |
| `api.kv.get/set` | Persistir preferencias |
