# Tasks: TUI Plugin `/zyro-model`

> **Fecha:** 2026-06-20  
> **Basado en:** `docs/spec-zyro-model-tui-plugin.md` + `docs/spec-zyro-model-routing.md`  
> **Pipeline:** SDD Fase 2 (Task Breakdown)  
> **Total tareas:** 7

---

## Convenciones

- **CA** = Criterio de Aceptación funcional (CA1–CA10)
- **R** = Criterio de Regresión (R1–R5)

---

## Task 1: Crear TUI Plugin zyro-model.tsx

**ID:** TASK-1  
**Estimación:** Alta  
**Dependencias:** Ninguna  
**Fase:** F3 (Implementación)  
**Archivo resultado:** `~/.config/opencode/tui-plugins/zyro-model.tsx`

### Descripción

Crear el archivo base del TUI plugin con la estructura mínima necesaria para que OpenCode lo cargue y registre los entry points. Este archivo contiene toda la lógica del plugin (las tasks 2–5 se implementan dentro del mismo archivo, no en archivos separados).

### Contenido del archivo

El archivo debe incluir:

1. **Header y tipos**
   - `// @ts-nocheck` y `/** @jsxImportSource @opentui/solid */`
   - Import de `TuiPlugin` desde `@opencode-ai/plugin/tui`
   - Import de `DialogSelect` desde `@opencode-ai/plugin/tui`
   - Import de `createSignal` desde `solid-js` (opcional, para estado local)
   - Interfaz `AgentInfo` con: `name`, `description`, `phase`, `currentModel`

2. **Constante `ZYRO_AGENTS`**
   - Array de 16 objetos `AgentInfo` con datos de los 16 agentes SDD
   - Orden: PRE-F0 → F0 → F1 → F2 → F3 → F4 → (sin fase)
   - Cada entrada: `{ name, description, phase }`
   - NO incluir `currentModel` aquí (se agrega en runtime)

3. **Registro del comando** via `api.command.register()`
   - `id`: `"zyro-model"`
   - `name`: `"zyro-model"`
   - `description`: `"Asignar modelos LLM por agente SDD"`
   - `handler`: llama a `start(api)`

4. **Registro del keymap** via `api.keymap.registerLayer()`
   - `id`: `"zyro-model-keymap"`
   - `keymap.Alt+K`: `{ name: "Zyro: Asignar modelo a agente", command: "zyro-model" }`

5. **Función `start(api)`** — esqueleto del flujo principal
   - Debe ser `async`
   - Por ahora solo debe llamar a `showAgentSelector()` (implementación en Task 2)

6. **Export default**
   - `export default { id: "zyro-model", tui }`

### Criterios de aceptación

| ID | Criterio | Verificación |
|----|----------|-------------|
| CA1 | El comando `/zyro-model` aparece en autocompletado de OpenCode | Escribir `/z` → ver "zyro-model" en la lista |
| CA2 | `Alt+K` abre el selector de agentes | Presionar Alt+K → se abre DialogSelect #1 |

### Notas técnicas

- El `// @ts-nocheck` es necesario porque OpenCode no expone tipos públicos para el TUI sandbox
- El keymap `Alt+K` debe registrarse con `registerLayer` (no `registerKeymap`) para no pisar keymaps existentes
- No usar `client.config.providers()` (API de server plugin) — usar `api.state.provider` (API de TUI plugin)

### Archivos afectados

```
~/.config/opencode/tui-plugins/zyro-model.tsx  (NUEVO)
```

---

## Task 2: Implementar DialogSelect #1 — Lista de agentes

**ID:** TASK-2  
**Estimación:** Media  
**Dependencias:** Task 1  
**Fase:** F3  
**Archivo:** `~/.config/opencode/tui-plugins/zyro-model.tsx`

### Descripción

Implementar la función `showAgentSelector()` que construye y muestra el `DialogSelect` con los 16 agentes agrupados por fase.

### Comportamiento detallado

1. **Construir opciones del DialogSelect:**
   - Opción inicial: `{ id: "SET_ALL", label: "★ Set All" }`
   - Por cada fase (PRE-F0, F0, F1, F2, F3, F4, —):
     - Opción de separador/título: `{ id: "__group_PRE-F0", label: "— PRE-F0 —", disabled: true }`
     - Opciones de agentes: `{ id: agent.name, label: " nombre — descripción (modelo)" }`
   - Opción final: `{ id: "DONE", label: "✓ Done" }`

2. **Modelo actual:** Cada agente muestra su modelo actual entre paréntesis. Si no tiene modelo asignado, muestra `(hereda del orchestrator)`.

3. **Flujo:**
   - Si usuario selecciona "★ Set All" → retorna `"SET_ALL"`
   - Si usuario selecciona un agente → retorna el `name` del agente
   - Si usuario selecciona "✓ Done" → retorna `"DONE"`
   - Si usuario cancela (Esc) → retorna `null`

### Criterios de aceptación

| ID | Criterio | Verificación |
|----|----------|-------------|
| CA3 | Muestra 16 agentes agrupados con descripciones y modelo actual | Verificar visualmente cada grupo |
| CA7 | "✓ Done" sale del selector y termina el flujo | Seleccionar → diálogos desaparecen |
| CA8 | Después de asignar, el loop vuelve al selector de agentes | Asignar modelo a un agente → ver la lista otra vez |

### Firma de la función

```typescript
async function showAgentSelector(api: any, agents: AgentInfo[]): Promise<string | null>
```

Retorna:
- `"SET_ALL"` → flujo Set All
- `"zyro-sdd-apply"` (o cualquier agent name) → flujo individual
- `"DONE"` → terminar
- `null` → cancelar

---

## Task 3: Implementar DialogSelect #2 — Lista de proveedores

**ID:** TASK-3  
**Estimación:** Media  
**Dependencias:** Task 1, API `api.state.provider`  
**Fase:** F3  
**Archivo:** `~/.config/opencode/tui-plugins/zyro-model.tsx`

### Descripción

Implementar la función `showProviderSelector()` que lee los proveedores disponibles desde `api.state.provider` y muestra un DialogSelect para elegir uno.

### Comportamiento detallado

1. **Obtener proveedores:** Llamar a `api.state.provider` y obtener `state.providers`
2. **Filtrar:** Solo mostrar proveedores que tengan `models.length > 0`
3. **Construir opciones:**
   - Cada proveedor: `{ id: provider.id, label: "provider.name (N modelos)" }`
   - Opción de volver: `{ id: "BACK", label: "← Volver" }`
4. **Caso sin proveedores:** Si `providers.filter(p => p.models.length > 0).length === 0`:
   - Mostrar mensaje: `"No hay proveedores configurados. Usá /connect en OpenCode para agregar uno."`
   - Solo opción "← Volver"

### Criterios de aceptación

| ID | Criterio | Verificación |
|----|----------|-------------|
| CA4 | Muestra SOLO proveedores configurados en OpenCode con modelos disponibles | Comparar con `/connect` |
| CA5 | "← Volver" regresa al selector de agentes (Task 2) | Seleccionar → volver a agentes |
| R1 | Sin proveedores → mensaje informativo y "← Volver" | Desconectar todos los providers y probar |

### Firma de la función

```typescript
async function showProviderSelector(api: any, providers: Provider[]): Promise<Provider | "BACK" | null>
```

### Implementación de `getAvailableProviders()`

```typescript
async function getAvailableProviders(api: any): Promise<Provider[]> {
  const state = await api.state.provider()
  return state.providers.filter((p: any) => p.models?.length > 0)
}
```

---

## Task 4: Implementar DialogSelect #3 — Lista de modelos

**ID:** TASK-4  
**Estimación:** Media  
**Dependencias:** Task 3  
**Fase:** F3  
**Archivo:** `~/.config/opencode/tui-plugins/zyro-model.tsx`

### Descripción

Implementar la función `showModelSelector()` que recibe un proveedor y muestra un DialogSelect con los modelos disponibles.

### Comportamiento detallado

1. **Construir opciones:**
   - Cada modelo del proveedor: `{ id: model.id, label: "model.name (model.id)" }`
   - Opción de volver: `{ id: "BACK", label: "← Volver" }`

2. **Búsqueda:** Si hay más de 10 modelos, incluir un campo de búsqueda/filtro (si DialogSelect lo soporta nativamente)

### Criterios de aceptación

| ID | Criterio | Verificación |
|----|----------|-------------|
| CA5 | "← Volver" regresa al selector de proveedores (Task 3) | Seleccionar → volver a proveedores |

### Firma de la función

```typescript
async function showModelSelector(api: any, provider: Provider): Promise<Model | "BACK" | null>
```

---

## Task 5: Implementar asignación (memoria + archivo + toast)

**ID:** TASK-5  
**Estimación:** Media  
**Dependencias:** Task 4  
**Fase:** F3  
**Archivo:** `~/.config/opencode/tui-plugins/zyro-model.tsx`

### Descripción

Implementar la función `assignModel()` que persiste la asignación del modelo en caliente (memoria) y en disco, y muestra un toast de confirmación. También implementar `handleSetAll()` para el caso especial.

### Comportamiento detallado de `assignModel()`

```typescript
async function assignModel(api: any, agentName: string, modelStr: string): Promise<void>
```

1. **En caliente (memoria):**
   ```typescript
   await api.client.global.config.update({
     agent: { [agentName]: { model: modelStr } },
   })
   ```
   Esto actualiza la configuración en el runtime de OpenCode instantáneamente.

2. **En disco (persistencia):**
   ```typescript
   try {
     const result = await Bun.$`zyrocli profile set ${agentName} ${modelStr}`
     if (result.exitCode !== 0) {
       throw new Error(result.stderr.toString())
     }
   } catch (e) {
     await api.ui.toast({
       message: `⚠ Error al persistir: ${e.message}`,
       variant: "error",
     })
     return  // No detener el flujo, pero avisar al usuario
   }
   ```
   `zyrocli profile set` se encarga de leer el JSON, modificarlo y escribirlo de vuelta.

3. **Toast de confirmación:**
   ```typescript
   await api.ui.toast({
     message: `✓ ${agentName} → ${modelStr}`,
     variant: "success",
   })
   ```

### Comportamiento detallado de `handleSetAll()`

```typescript
async function handleSetAll(api: any, providers: Provider[], agents: AgentInfo[]): Promise<void>
```

1. Mostrar DialogSelect #2 (proveedores) — reutilizar `showProviderSelector()`
2. Mostrar DialogSelect #3 (modelo) — reutilizar `showModelSelector()`
3. Hacer un loop sobre los 16 agentes:
   - `api.client.global.config.update()` para cada uno
   - `Bun.$ zyrocli profile set ...` para cada uno (con try/catch individual)
4. Toast: `"✓ Set All: Todos los agentes → provider/model"`

### Criterios de aceptación

| ID | Criterio | Verificación |
|----|----------|-------------|
| CA6 | "★ Set All" asigna el mismo modelo a todos los 16 agentes | Verificar opencode.json después |
| CA9 | La asignación se escribe en caliente (memoria) y en disco (archivo) | `api.client.global.config.get()` + leer opencode.json |
| CA10 | Aparece toast de confirmación con nombre de agente y modelo | Verificar visualmente |
| R2 | Si `zyrocli` no está en PATH, asigna en caliente igual pero muestra toast de advertencia | `mv $(which zyrocli) /tmp/` y probar |
| R5 | Si `zyrocli profile set` falla, se muestra toast error | Corromper opencode.json temporalmente |

### Manejo de errores

| Escenario | Comportamiento |
|-----------|----------------|
| `zyrocli` no encontrado | Toast warning: "⚠ zyrocli no encontrado. La asignación en caliente funciona pero no persistirá al reiniciar OpenCode." |
| `zyrocli profile set` falla | Toast error: "⚠ Error al persistir: <mensaje>" + continuar flujo |
| `api.client.global.config.update()` falla | Mostrar toast error y terminar flujo (error fatal) |

---

## Task 6: Integrar con instalador (zyrocli install)

**ID:** TASK-6  
**Estimación:** Media  
**Dependencias:** Tasks 1–5 (el plugin debe existir para embeberlo)  
**Fase:** F3  
**Archivos:**
- `internal/opencode/tui-plugins/zyro-model.tsx` (NUEVO — copia exacta del plugin)
- `internal/opencode/tui_plugins.go` (MODIFICAR)
- `internal/opencode/tui_plugins_test.go` (MODIFICAR)
- `cmd/zyrocli/install.go` (MODIFICAR)

### Descripción

Embeber el plugin en el binario Go de `zyrocli` y agregar su instalación durante `zyrocli install`.

### Subtask 6.1: Copiar plugin a Go embed

1. Copiar `~/.config/opencode/tui-plugins/zyro-model.tsx` a `internal/opencode/tui-plugins/zyro-model.tsx`
2. Agregar directiva `//go:embed tui-plugins/zyro-model.tsx` en `tui_plugins.go`

### Subtask 6.2: Extender `tui_plugins.go`

Agregar:

```go
//go:embed tui-plugins/zyro-model.tsx
var zyroModelPlugin string

func ZyroModelPluginPath() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".config", "opencode", "tui-plugins", "zyro-model.tsx")
}

func WriteZyroModelPlugin() (string, error) {
    pluginPath := ZyroModelPluginPath()
    if err := os.MkdirAll(filepath.Dir(pluginPath), 0755); err != nil {
        return "", fmt.Errorf("opencode: create tui-plugins dir: %w", err)
    }
    if err := os.WriteFile(pluginPath, []byte(zyroModelPlugin), 0644); err != nil {
        return "", fmt.Errorf("opencode: write zyro-model plugin: %w", err)
    }
    return pluginPath, nil
}
```

### Subtask 6.3: Extender `UpdateTuiJSON()`

Modificar la función para:

1. Agregar `ZyroModelPluginPath()` a la lista `wanted` (junto con `ZorroPluginPath()`)
2. Agregar `"zyro-model"` (referencia antigua) a la lista `stale`

Lista `stale` actualizada:
```go
stale := []string{
    "opencode-subagent-statusline",
    "zyro-model",
}
```

Lista `wanted` actualizada:
```go
wanted := []string{
    ZorroPluginPath(),
    ZyroModelPluginPath(),
}
```

### Subtask 6.4: Agregar paso en install.go

Agregar estos pasos al `RunInstall` de `install.go`:

```go
{Name: "Instalando plugin Zyro Model", Action: func() error {
    _, err := opencode.WriteZyroModelPlugin()
    return err
}},
```

Preferiblemente después del paso de zorro-logo.

### Subtask 6.5: Tests

Agregar tests análogos a los existentes en `tui_plugins_test.go`:

- `TestWriteZyroModelPlugin`: verifica que el archivo se crea correctamente
- `TestUpdateTuiJSONWithZyroModel`: verifica que tui.json incluye el nuevo plugin
- `TestUpdateTuiJSONIdempotentWithZyroModel`: verifica que ejecutar dos veces no da error

### Criterios de aceptación

| ID | Criterio | Verificación |
|----|----------|-------------|
| — | El plugin se copia a `~/.config/opencode/tui-plugins/` durante `zyrocli install` | Ejecutar install y verificar |
| — | El plugin se registra en `tui.json` automáticamente | Verificar `tui.json` después de install |
| — | `go test ./internal/opencode/` pasa | Tests unitarios |
| — | Plugin stale `zyro-model` se elimina de tui.json si existe | Verificar limpieza |

---

## Task 7: Limpiar código muerto del server plugin

**ID:** TASK-7  
**Estimación:** Baja  
**Dependencias:** Ninguna (puede ejecutarse en paralelo con Tasks 1–6)  
**Fase:** F3  
**Archivos:**
- `~/.config/opencode/plugins/zyro-model.ts` (ELIMINAR)
- Documentación que referencie `model-assigner` (VARIOS)

### Descripción

Evaluar y limpiar el server plugin legacy `zyro-model.ts` y cualquier referencia obsoleta.

### Subtask 7.1: Evaluar server plugin

El archivo `~/.config/opencode/plugins/zyro-model.ts` actualmente:

```typescript
export const ZyroModelPlugin: Plugin = async ({ client }) => {
  return {
    event: async ({ event }) => {
      if (event.type === "command.executed" && event.properties.name === "zyro-model") {
        // Solo muestra toasts, no tiene UI interactiva
        await client.tui.showToast({ ... })
      }
    },
  }
}
```

**Análisis:** Este plugin es un server plugin (no TUI plugin). Su única función es mostrar toasts cuando se ejecuta `/zyro-model`. Con el nuevo TUI plugin, este server plugin es redundante porque:
- El TUI plugin ya maneja el comando `/zyro-model` via `api.command.register()`
- El TUI plugin ya muestra toasts con `api.ui.toast()`
- Los server plugins y TUI plugins pueden interferir si manejan el mismo comando

**Decisión:** Eliminar el server plugin.

### Subtask 7.2: Eliminar archivo

```bash
rm ~/.config/opencode/plugins/zyro-model.ts
```

### Subtask 7.3: Limpiar referencias

Buscar y limpiar referencias a:

| Referencia obsoleta | Acción |
|---------------------|--------|
| `model-assigner` | Eliminar de documentación (docs/spec-zyro-model-routing.md), scripts, configs |
| `plugins/zyro-model.ts` | Eliminar de cualquier path list |
| `zyro-model.js` (si existe) | Eliminar de stale list si aplica |

### Subtask 7.4: Verificar que `profile_tui.go` no necesita cambios

Revisar `cmd/zyrocli/profile_tui.go`: actualmente redirige al usuario a usar `/zyro-model`. Esto sigue siendo válido después del cambio, así que no necesita modificación. Sin embargo, considerar si vale la pena que `profile_tui.go` detecte que el TUI plugin está instalado y en lugar de mostrar un mensaje estático, lanzar algo interactivo.

**Decisión para ahora:** No cambiar `profile_tui.go`. Se mantiene como está.

### Criterios de aceptación

| ID | Criterio | Verificación |
|----|----------|-------------|
| — | No hay archivos huérfanos de plugins server de zyro-model | `ls ~/.config/opencode/plugins/zyro-model.*` → no existe |
| — | No hay comandos rotos por la eliminación | `/zyro-model` sigue funcionando (TUI plugin) |
| — | No hay referencias a `model-assigner` en docs/code | `grep -r "model-assigner" docs/` → 0 resultados |
| — | `profile_tui.go` sigue funcionando correctamente | `zyrocli profile tui` → muestra mensaje |

---

## Resumen de tareas

| # | Tarea | Estimación | Dependencias | Archivos principales |
|---|-------|-----------|--------------|---------------------|
| 1 | Crear TUI plugin base | Alta | — | `tui-plugins/zyro-model.tsx` |
| 2 | DialogSelect #1 — Agentes | Media | 1 | `tui-plugins/zyro-model.tsx` |
| 3 | DialogSelect #2 — Proveedores | Media | 1 | `tui-plugins/zyro-model.tsx` |
| 4 | DialogSelect #3 — Modelos | Media | 3 | `tui-plugins/zyro-model.tsx` |
| 5 | Asignación + Set All | Media | 4 | `tui-plugins/zyro-model.tsx` |
| 6 | Integración con instalador | Media | 1–5 | `tui_plugins.go`, `install.go` |
| 7 | Limpiar código muerto | Baja | — | `plugins/zyro-model.ts` |

### Dependencias entre tareas

```
Task 1 ──┬── Task 2 ──┬── Task 3 ──┬── Task 4 ──┬── Task 5 ──┐
         │            │            │            │            │
         │            └────────────┘────────────┘            │
         │                                                   │
         └───────────────────────────────────────────────────┼── Task 6
                                                             │
Task 7 (independiente — puede ejecutarse en paralelo) ───────┘
```

### Orden de ejecución recomendado

1. **Task 7 primero** (limpiar código muerto — baja estimación, sin dependencias, despeja el camino)
2. **Task 1** (crear archivo base del plugin)
3. **Task 2** (selector de agentes)
4. **Task 3** (selector de proveedores)
5. **Task 4** (selector de modelos)
6. **Task 5** (asignación + Set All + toast)
7. **Task 6** (integrar con instalador — requiere plugin terminado)
