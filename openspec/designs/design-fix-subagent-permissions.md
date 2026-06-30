# Fix Subagent Permissions — Design Técnico

> **Documento**: Design Técnico (F2)
> **Basado en**: `openspec/specs/spec-fix-subagent-permissions.md`
> **Conexiones**: `openspec/specs/spec-acceptance-criteria-tracking.md` (§4.2), `openspec/specs/spec-activate-boundary-enforcement.md`
> **Cambio**: `fix-subagent-permissions`
> **Proyecto HelixDB ID**: 1005

---

## 1. Arquitectura Actual

### 1.1 Generación de opencode.jsonc

La función `buildInstallConfig()` en `cmd/zyrocli/install.go` (líneas 232–423) construye la configuración global de OpenCode. Retorna un `*opencode.Config` con:

- `Agent`: map[string]opencode.Agent — 15 agentes subagentes + 1 orquestador
- `MCP`: servidores MCP (helix-integration, zyro-task-board, context, gitmcp)
- `Skills`: rutas de skills

Cada agente tiene un campo `Permission map[string]any` que OpenCode interpreta como reglas de capability:

```go
type Agent struct {
    Mode        string         `json:"mode"`
    Model       string         `json:"model"`
    Description string         `json:"description,omitempty"`
    Prompt      string         `json:"prompt,omitempty"`
    Hidden      bool           `json:"hidden,omitempty"`
    Permission  map[string]any `json:"permission,omitempty"`
}
```

### 1.2 El problema del map compartido `sddPerms`

```go
sddPerms := map[string]any{
    "read": "allow", "write": "deny", "edit": "deny", "bash": "allow",
}
```

En Go, `map` es un tipo referencia. Las 4 referencias:

| Variable | Línea | Agente |
|----------|-------|--------|
| `Permission: sddPerms` | 351 | zyro-sdd-propose |
| `Permission: sddPerms` | 367 | zyro-sdd-design |
| `Permission: sddPerms` | 373 | zyro-sdd-tasks |
| `Permission: sddPerms` | 379 | zyro-sdd-archive |

**Apuntan al mismo objeto en memoria.** Cualquier mutación al map `sddPerms` se refleja en los 4 agentes simultáneamente. Esto es el núcleo del problema: no se puede cambiar un permiso para un solo agente sin afectar a los demás.

### 1.3 Árbol de dependencias de permisos actual

```
buildInstallConfig()
  ├── sddPerms (map compartido, bash:allow)
  │     ├── sdd-propose
  │     ├── sdd-design
  │     ├── sdd-tasks
  │     └── sdd-archive ← necesita bash:allow (separación necesaria)
  ├── maps inline (12 agentes con permisos individuales)
  └── return cfg → opencode.WriteGlobalConfig(cfg)
```

### 1.4 Discrepancias actuales

| Problema | Agentes afectados | Severidad |
|----------|-------------------|-----------|
| bash:allow sin necesitarlo | pre-f0, phase-0-patterns, phase-0-libraries, sdd-design, sdd-tasks, sdd-propose | **ALTA** (riesgo de ejecución no autorizada) |
| write:allow donde el skill lo prohíbe | sdd-verify | **ALTA** (puede modificar archivos del proyecto) |
| task:{"*":allow} excesivo | 9 agentes | **MEDIA** (orquestación no autorizada) |
| Sin bloque task explícito | 6 agentes | **BAJA** (depende del default de OpenCode) |
| Sin webfetch donde se necesita | phase-0-patterns, phase-0-libraries | **MEDIA** (no pueden investigar) |
| Sin question donde se necesita | sdd-explore | **BAJA** (no puede entrevistar) |

---

## 2. Propuesta de Diseño

### 2.1 Principios rectores

1. **Principio de mínimo privilegio**: cada agente tiene solo los permisos que su SKILL.md declara explícitamente.
2. **Explicitud > herencia**: prohibiciones explícitas (`deny`) son preferibles a defaults implícitos.
3. **Seguridad por aislamiento**: agentes con distintas necesidades de permiso NO comparten el mismo objeto map.
4. **Un solo archivo modificado**: todos los cambios se concentran en `cmd/zyrocli/install.go` función `buildInstallConfig()`.

### 2.2 Diagrama de flujo post-cambio

```
buildInstallConfig()
  │
  ├── sddPerms (bash: deny) ──────────► sdd-propose  (via permsWithTaskDeny)
  │                                     ├── sdd-design   (via permsWithTaskDeny)
  │                                     └── sdd-tasks    (via permsWithTaskDeny)
  │
  ├── sddArchivePerms (bash: allow) ───► sdd-archive  (con task:{"*":"deny"} inline)
  │
  ├── maps inline ─────────────────────► 12 agentes con permisos individuales
  │
  └── opencode.WriteGlobalConfig(cfg)
          │
          ▼
    ~/.config/opencode/opencode.jsonc
          │
          ▼
    OpenCode aplica permisos al ejecutar subagentes
```

### 2.3 Mapa de cambios por agente

A continuación el delta completo entre permisos actuales y post-cambio:

| # | Agente | read | write | edit | bash | task | webfetch | question | Cambios |
|---|---|---|---|---|---|---|---|---|---|
| 1 | orchestrator | allow | deny | deny | deny | restrictivo | allow | allow | — |
| 2 | **pre-f0** | allow | deny | deny | **deny** | — | allow | allow | bash↓; task– |
| 3 | **phase-0-patterns** | allow | deny | deny | **deny** | — | **allow** | — | bash↓; wf+; task– |
| 4 | **phase-0-libraries** | allow | deny | deny | **deny** | — | **allow** | — | bash↓; wf+; task– |
| 5 | skills-find | allow | deny | deny | allow | — | allow | — | task– |
| 6 | skills-audit | allow | deny | deny | deny | — | allow | — | task– |
| 7 | skills-apply | allow | deny | deny | allow | — | — | — | task– |
| 8 | sdd-apply | allow | allow | allow | allow | — | — | — | task– |
| 9 | **sdd-verify** | allow | **deny** | deny | allow | — | — | — | write↓; task– |
| 10 | **sdd-explore** | allow | deny | deny | allow | **{"*":"deny"}** | — | **allow** | q+; task+ |
| 11 | **sdd-propose** | allow | deny | deny | **deny** (vía sddPerms) | **{"*":"deny"}** | — | — | bash↓; task+ |
| 12 | sdd-spec | allow | deny | deny | deny | — | — | — | task– |
| 13 | **sdd-design** | allow | deny | deny | **deny** (vía sddPerms) | **{"*":"deny"}** | — | — | bash↓; task+ |
| 14 | **sdd-tasks** | allow | deny | deny | **deny** (vía sddPerms) | **{"*":"deny"}** | — | — | bash↓; task+ |
| 15 | **sdd-archive** | allow | deny | deny | allow (map propio) | **{"*":"deny"}** | — | — | separado; task+ |
| 16 | to-issues | allow | deny | deny | allow | **{"*":"deny"}** | allow | — | task+ |

**Leyenda**: ↓ = allow→deny, + = agregado, – = eliminado, wf = webfetch, q = question

### 2.4 Especificación de cambios en install.go

#### 2.4.1 Cambio en `sddPerms` (línea 237)

```go
// Antes:
sddPerms := map[string]any{
    "read": "allow", "write": "deny", "edit": "deny", "bash": "allow",
}
// Después:
sddPerms := map[string]any{
    "read": "allow", "write": "deny", "edit": "deny", "bash": "deny",
}
```

#### 2.4.2 Nueva variable `sddArchivePerms` (insertar después de sddPerms)

```go
sddArchivePerms := map[string]any{
    "read": "allow", "write": "deny", "edit": "deny", "bash": "allow",
    "task": map[string]any{"*": "deny"},
}
```

#### 2.4.3 Función helper `permsWithTaskDeny`

```go
// permsWithTaskDeny clona un map base y agrega task:{"*":"deny"}.
// Se usa para sdd-propose, sdd-design, sdd-tasks que comparten sddPerms
// pero necesitan bloque task explícito.
permsWithTaskDeny := func(base map[string]any) map[string]any {
    m := make(map[string]any, len(base)+1)
    for k, v := range base {
        m[k] = v
    }
    m["task"] = map[string]any{"*": "deny"}
    return m
}
```

**Justificación**: Go no permite "merge" de maps en literales. Esta función evita duplicar las 4 líneas de permisos base para cada uno de los 3 agentes, manteniendo un solo punto de cambio para permisos base SDD.

#### 2.4.4 Cambios inline por agente

**zyro-phase-0-patterns** (líneas 276-279):
```go
// Antes:
"read": "allow", "write": "deny", "edit": "deny",
"bash": "allow", "task": map[string]any{"*": "allow"},
// Después:
"read": "allow", "write": "deny", "edit": "deny",
"bash": "deny", "webfetch": "allow",
```

**zyro-phase-0-libraries** (líneas 285-289):
```go
// Antes:
"read": "allow", "bash": "allow",
"write": "deny", "edit": "deny",
"task": map[string]any{"*": "allow"},
// Después:
"read": "allow", "bash": "deny", "webfetch": "allow",
"write": "deny", "edit": "deny",
```

**zyro-skills-find** (líneas 295-298):
```go
// Eliminar: "task": map[string]any{"*": "allow"},
```

**zyro-skills-audit** (líneas 305-308):
```go
// Eliminar: "task": map[string]any{"*": "allow"},
```

**zyro-skills-apply** (líneas 315-318):
```go
// Eliminar: "task": map[string]any{"*": "allow"},
```

**zyro-sdd-apply** (líneas 324-327):
```go
// Eliminar: "task": map[string]any{"*": "allow"},
```

**zyro-sdd-verify** (líneas 333-335):
```go
// Antes:
"read": "allow", "write": "allow", "edit": "deny",
"bash": "allow", "task": map[string]any{"*": "allow"},
// Después:
"read": "allow", "write": "deny", "edit": "deny",
"bash": "allow",
```

> **NOTA**: Se elimina `"write": "allow"` → `"write": "deny"`. Ver §5 para implicaciones con acceptance criteria tracking.

**zyro-sdd-explore** (líneas 342-344):
```go
// Antes:
"read": "allow", "bash": "allow",
"write": "deny", "edit": "deny",
// Después:
"read": "allow", "bash": "allow", "question": "allow",
"write": "deny", "edit": "deny",
"task": map[string]any{"*": "deny"},
```

**zyro-sdd-propose** (línea 351):
```go
// Antes:
Permission: sddPerms,
// Después:
Permission: permsWithTaskDeny(sddPerms),
```

**zyro-sdd-spec** (líneas 357-360):
```go
// Eliminar: "task": map[string]any{"*": "allow"},
```

**zyro-sdd-design** (línea 367):
```go
// Antes:
Permission: sddPerms,
// Después:
Permission: permsWithTaskDeny(sddPerms),
```

**zyro-sdd-tasks** (línea 373):
```go
// Antes:
Permission: sddPerms,
// Después:
Permission: permsWithTaskDeny(sddPerms),
```

**zyro-sdd-archive** (línea 379):
```go
// Antes:
Permission: sddPerms,
// Después:
Permission: sddArchivePerms,
```

**zyro-pre-f0** (líneas 385-388):
```go
// Antes:
"read": "allow", "bash": "allow", "webfetch": "allow", "question": "allow",
"write": "deny", "edit": "deny",
"task": map[string]any{"*": "allow"},
// Después:
"read": "allow", "bash": "deny", "webfetch": "allow", "question": "allow",
"write": "deny", "edit": "deny",
```

**to-issues** (líneas 395-398):
```go
// Antes:
"read": "allow", "bash": "allow", "webfetch": "allow",
"write": "deny", "edit": "deny",
// Después:
"read": "allow", "bash": "allow", "webfetch": "allow",
"write": "deny", "edit": "deny",
"task": map[string]any{"*": "deny"},
```

---

## 3. Decisiones de Diseño

### 3.1 Separar sdd-archive de sddPerms

| | Decisión |
|---|----------|
| **Problema** | sdd-archive necesita `bash: allow` (único agente SDD que limpia archivos temporales) pero comparte `sddPerms` con 3 agentes que deben tener `bash: deny` |
| **Solución** | Crear `sddArchivePerms` independiente |
| **Alternativa** | Mutar `sddPerms` a `bash: deny` y luego mutarlo de vuelta (inviable: map compartido por referencia) |
| **Alternativa 2** | Hacer que sdd-archive ejecute bash vía orquestador (excesivo, viola principio de autonomía) |
| **Riesgo** | Olvidar cambiar la línea 379 de `Permission: sddPerms` a `Permission: sddArchivePerms` |
| **Verificación** | CE10 en el script: `get_perm zyro-sdd-archive bash` debe retornar `"allow"` |

### 3.2 Función helper permsWithTaskDeny

| | Decisión |
|---|----------|
| **Problema** | 3 agentes (sdd-propose, sdd-design, sdd-tasks) comparten `sddPerms` y necesitan `task: {"*":"deny"}` adicional |
| **Solución** | Función helper `permsWithTaskDeny` que clona el map base y agrega task deny |
| **Alternativa** | Expandir a maps inline para los 3 agentes (duplicación, frágil ante cambios futuros) |
| **Alternativa 2** | No agregar task deny (riesgo si OpenCode cambia default a "allow") |
| **Ventaja** | Un solo punto de cambio para permisos base SDD. Si se agrega `"glob": "deny"` a sddPerms, los 3 lo heredan automáticamente |
| **Desventaja** | Indirección: hay que leer la función helper para entender permisos completos |
| **Verificación** | CE9b en el script |

### 3.3 Agentes sin bloque task explícito (heredan default)

| | Decisión |
|---|----------|
| **Problema** | 9 agentes tenían `task: {"*":"allow"}` que se elimina. Quedan sin bloque task, dependiendo del default de OpenCode |
| **Solución** | Eliminar el bloque (no agregar deny explícito) para minimizar cambios |
| **Alternativa** | Agregar `task: {"*":"deny"}` a todos (más cambios, más ruido) |
| **Riesgo** | Si OpenCode cambia su default de deny a allow, estos 9 agentes heredarían task:allow |
| **Mitigación** | Riesgo aceptado como bajo (ningún skill de esos agentes usa task). Si ocurre, es un cambio de 9 líneas |

### 3.4 write:deny en sdd-verify vs. acceptance criteria tracking

Ver §5 a continuación.

---

## 4. Archivos a Modificar

| Archivo | Cambio | Líneas afectadas |
|---------|--------|-----------------|
| `cmd/zyrocli/install.go` | Modificar `sddPerms` bash:allow → deny | 237 |
| `cmd/zyrocli/install.go` | Agregar `sddArchivePerms` (nueva variable) | después de 238 |
| `cmd/zyrocli/install.go` | Agregar función `permsWithTaskDeny` (closure) | después de sddArchivePerms |
| `cmd/zyrocli/install.go` | Modificar zyro-phase-0-patterns | 276-279 |
| `cmd/zyrocli/install.go` | Modificar zyro-phase-0-libraries | 285-289 |
| `cmd/zyrocli/install.go` | Modificar zyro-skills-find | 295-298 |
| `cmd/zyrocli/install.go` | Modificar zyro-skills-audit | 305-308 |
| `cmd/zyrocli/install.go` | Modificar zyro-skills-apply | 315-318 |
| `cmd/zyrocli/install.go` | Modificar zyro-sdd-apply | 324-327 |
| `cmd/zyrocli/install.go` | Modificar zyro-sdd-verify | 333-335 |
| `cmd/zyrocli/install.go` | Modificar zyro-sdd-explore | 342-344 |
| `cmd/zyrocli/install.go` | Modificar zyro-sdd-propose | 351 |
| `cmd/zyrocli/install.go` | Modificar zyro-sdd-spec | 357-360 |
| `cmd/zyrocli/install.go` | Modificar zyro-sdd-design | 367 |
| `cmd/zyrocli/install.go` | Modificar zyro-sdd-tasks | 373 |
| `cmd/zyrocli/install.go` | Modificar zyro-sdd-archive | 379 |
| `cmd/zyrocli/install.go` | Modificar zyro-pre-f0 | 385-388 |
| `cmd/zyrocli/install.go` | Modificar to-issues | 395-398 |
| `openspec/designs/design-fix-subagent-permissions.md` | Este documento | — |

**Total**: 1 archivo de código fuente, 17 modificaciones puntuales.

---

## 5. Conexión con Acceptance Criteria Tracking

### 5.1 Tensiones identificadas

El spec `spec-acceptance-criteria-tracking.md` (§4.2) identifica conflictos directos con este cambio de permisos:

#### 5.1.1 sdd-verify: write:deny vs. actualización de criteria

```
spec-fix-subagent-permissions.md          spec-acceptance-criteria-tracking.md
         │                                          │
         ▼                                          ▼
  sdd-verify: write:deny (este diseño)    sdd-verify necesita write:allow
                                          para actualizar acceptance_criteria
                                          en HelixDB (CE10 del spec de criteria)
         │                                          │
         └──────────────────┬───────────────────────┘
                            ▼
                    CONFLICTO IDENTIFICADO
```

**Resolución recomendada** (alineada con §4.2 Opción A):

1. Este cambio de permisos establece `write: deny` para sdd-verify (correcto: no debe modificar archivos del proyecto).
2. La escritura de acceptance criteria en HelixDB no debe pasar por `write` permission sino por el MCP tool `save_to_helix` — que se controla mediante **Boundari policy** (spec-activate-boundary-enforcement.md).
3. Cuando se implemente acceptance criteria tracking, la política Boundari para sdd-verify debe incluir:

```yaml
# En phase-boundari.yaml (fase F3/F4)
tool_rules:
  - name: "save_to_helix"
    action: allow
    params:
      label: "Task"
      # Solo puede actualizar acceptance_criteria, no crear nodos arbitrarios
```

**Estado**: Aceptado como deuda de integración. Este cambio de permisos es correcto para la situación actual. La conexión se resolverá cuando se implemente `spec-activate-boundary-enforcement.md`.

#### 5.1.2 sdd-tasks: permisos para crear nodos Task con acceptance_criteria

```
sdd-tasks:
  - write: deny  (vía sddPerms) → correcto: no escribe archivos del proyecto
  - task: {"*":"deny"}           → correcto: no despacha subagentes
  - bash: deny                   → correcto: skill lo prohíbe
  - → ¿Cómo crea nodos Task en HelixDB?
```

**Mecanismo**: sdd-tasks no escribe archivos ni ejecuta bash directamente. Crea nodos Task en HelixDB a través del **MCP tool `save_to_helix`** proporcionado por el servidor `helix-integration`. Este tool se controla mediante permisos de MCP, no mediante el permission map de OpenCode.

**Decisión**: No se requiere ningún permiso adicional en el permission map de sdd-tasks. La capacidad de crear nodos en HelixDB se gobierna por la Boundari policy en fase de ejecución.

### 5.2 Mapa de dependencias entre specs

```
spec-fix-subagent-permissions.md (ESTE)
  │
  ├── define permisos base
  │
  ├──► spec-acceptance-criteria-tracking.md (§4.2)
  │     └── sdd-verify.write:deny → mitigado vía Boundari policy
  │     └── sdd-tasks.create → ya funciona vía MCP tool
  │
  └──► spec-activate-boundary-enforcement.md
        └── tool_rules para save_to_helix en sdd-verify
```

---

## 6. Pruebas

### 6.1 Verificación post-instalación (script bash)

Ubicación propuesta: `openspec/changes/fix-subagent-permissions/verify-permissions.sh`

El script verifica los 11 criterios de éxito parseando `~/.config/opencode/opencode.jsonc`:

| CE | Verificación | Comando |
|----|-------------|---------|
| CE1 | pre-f0 bash=deny, question=allow | `get_perm zyro-pre-f0 bash` == "deny" |
| CE2 | phase-0-patterns bash=deny, webfetch=allow | `get_perm zyro-phase-0-patterns webfetch` == "allow" |
| CE3 | phase-0-libraries bash=deny, webfetch=allow | `get_perm zyro-phase-0-libraries bash` == "deny" |
| CE4 | sdd-design bash=deny | `get_perm zyro-sdd-design bash` == "deny" |
| CE5 | sdd-tasks bash=deny | `get_perm zyro-sdd-tasks bash` == "deny" |
| CE6 | sdd-propose bash=deny | `get_perm zyro-sdd-propose bash` == "deny" |
| CE7 | sdd-verify write=deny | `get_perm zyro-sdd-verify write` == "deny" |
| CE8 | sdd-explore question=allow | `get_perm zyro-sdd-explore question` == "allow" |
| CE9 | Ningún subagente (excepto orchestrator) tiene task:allow | `has_task_block` == False para 9 agentes |
| CE9b | 6 agentes tienen task:{"*":"deny"} explícito | `get_perm $agent task` contiene "deny" |
| CE10 | sdd-archive bash=allow | `get_perm zyro-sdd-archive bash` == "allow" |
| CE11 | opencode.jsonc se genera sin errores | `zyrocli install` exitoso |

### 6.2 Prueba de regresión (compilación)

```bash
go build ./cmd/zyrocli/...   # debe compilar sin errores
```

### 6.3 Verificación manual de un agente específico

```bash
# Caso más crítico: sdd-design (skill: "NO corras bash")
python3 -c "
import json
cfg = json.load(open('/home/secko/.config/opencode/opencode.jsonc'))
a = cfg['agent']['zyro-sdd-design']
print(json.dumps(a['permission'], indent=2))
# Output esperado:
# {
#   "read": "allow",
#   "write": "deny",
#   "edit": "deny",
#   "bash": "deny",
#   "task": {"*": "deny"}
# }
"
```

---

## 7. Riesgos y Mitigaciones

### 7.1 Regresión por mapas compartidos (ALTO)

| | |
|---|---|
| **Riesgo** | Si `sdd-archive` NO se separa de `sddPerms` (olvidar línea 379), hereda `bash: deny` y pierde capacidad de limpiar temporales |
| **Probabilidad** | Baja (el cambio es explícito) |
| **Impacto** | Alto (sdd-archive no puede completar su función) |
| **Mitigación** | CE10 verifica `sdd-archive bash=allow` |
| **Mitigación 2** | Revisión de código: verificar que la línea `Permission: sddPerms` para sdd-archive cambió |

### 7.2 sdd-verify sin write para acceptance criteria (MEDIO)

| | |
|---|---|
| **Riesgo** | Cuando se implemente acceptance criteria tracking, sdd-verify necesitará escribir en HelixDB |
| **Probabilidad** | Alta (spec ya está definido) |
| **Impacto** | Medio (Bloquea CE10 de acceptance-criteria-tracking) |
| **Mitigación** | Documentado en §5. Se resolverá vía Boundari policy (tool rule save_to_helix) |

### 7.3 Cambio de default en OpenCode (BAJO)

| | |
|---|---|
| **Riesgo** | Si OpenCode cambia el default de task de `deny` a `allow`, 9 agentes sin bloque task explícito heredarían comportamiento incorrecto |
| **Probabilidad** | Baja |
| **Impacto** | Medio (9 agentes ganan task:allow implícito) |
| **Mitigación** | Los skills de esos agentes no usan task. Si ocurre, agregar `task: {"*":"deny"}` a todos |

### 7.4 sddPerms compartida sigue siendo frágil (BAJO)

| | |
|---|---|
| **Riesgo** | Si en el futuro un agente (propose, design o tasks) necesita un permiso diferente, habrá que separarlo de sddPerms |
| **Probabilidad** | Media |
| **Impacto** | Bajo (el cambio de separación es mecánico) |
| **Mitigación** | Deuda técnica aceptada y documentada. El map compartido reduce duplicación para el caso actual |

### 7.5 17 cambios en un solo archivo (MEDIO)

| | |
|---|---|
| **Riesgo** | Error humano: permiso incorrecto, coma mal puesta, olvidar un cambio |
| **Probabilidad** | Media |
| **Impacto** | Alto (JSON inválido, permisos incorrectos) |
| **Mitigación** | `go build` detecta errores de sintaxis Go. Script de verificación detecta permisos incorrectos. |

---

## 8. Orden de Implementación

| Paso | Cambio | Verificación |
|------|--------|-------------|
| 1 | Modificar `sddPerms` (bash:allow→deny) | Compila |
| 2 | Agregar `sddArchivePerms` | Compila |
| 3 | Agregar `permsWithTaskDeny` | Compila |
| 4 | Cambiar sdd-archive a `sddArchivePerms` | Compila + CE10 |
| 5 | Cambiar sdd-propose/design/tasks a `permsWithTaskDeny(sddPerms)` | Compila + CE4/5/6 |
| 6 | Cambiar phase-0-patterns y phase-0-libraries | Compila + CE2/3 |
| 7 | Cambiar pre-f0 | Compila + CE1 |
| 8 | Cambiar sdd-verify | Compila + CE7 |
| 9 | Cambiar sdd-explore | Compila + CE8 |
| 10 | Eliminar task blocks en 7 agentes | Compila + CE9 |
| 11 | Agregar task blocks en to-issues | Compila + CE9b |
| 12 | Compilar: `go build ./cmd/zyrocli/...` | Compilación exitosa |
| 13 | Ejecutar: `zyrocli install` | Genera opencode.jsonc |
| 14 | Ejecutar script de verificación | 11/11 criterios exitosos |

---

## 9. Estado de los Acceptance Criteria

| ID | Descripción | Estado | Verificado por |
|----|-------------|--------|---------------|
| CE1 | pre-f0 bash=deny, question=allow | 🔲 Pendiente | Script §6.1 |
| CE2 | phase-0-patterns bash=deny, webfetch=allow | 🔲 Pendiente | Script §6.1 |
| CE3 | phase-0-libraries bash=deny, webfetch=allow | 🔲 Pendiente | Script §6.1 |
| CE4 | sdd-design bash=deny | 🔲 Pendiente | Script §6.1 |
| CE5 | sdd-tasks bash=deny | 🔲 Pendiente | Script §6.1 |
| CE6 | sdd-propose bash=deny | 🔲 Pendiente | Script §6.1 |
| CE7 | sdd-verify write=deny | 🔲 Pendiente | Script §6.1 |
| CE8 | sdd-explore question=allow | 🔲 Pendiente | Script §6.1 |
| CE9 | Subagentes sin task:allow (excepto orchestrator) | 🔲 Pendiente | Script §6.1 |
| CE10 | sdd-archive bash=allow | 🔲 Pendiente | Script §6.1 |
| CE11 | opencode.jsonc generado sin errores | 🔲 Pendiente | `zyrocli install` |

---

## 10. Glosario

| Término | Significado |
|---------|-------------|
| sddPerms | Variable Go que contiene el map de permisos base para agentes SDD (compartido) |
| sddArchivePerms | Nuevo map independiente para sdd-archive (bash:allow) |
| permsWithTaskDeny | Función helper que clona un map base y agrega task:{"*":"deny"} |
| CE | Criterion of Evaluation / Criterio de Éxito |
| Boundari policy | Sistema de políticas de enforcement del módulo `internal/boundari/` |
| HelixDB | Base de datos gráfica local para persistencia de nodos SDD |
