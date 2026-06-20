# Fix Subagent Permissions — Especificación Técnica

## Propósito

Alinear los permisos de los subagentes de ZyroCLI declarados en `opencode.jsonc` con las responsabilidades reales definidas en cada `SKILL.md`, eliminando riesgos de seguridad operativa. Este spec concreta el diseño detallado para implementar la propuesta validada en `openspec/proposals/fix-subagent-permissions.md`.

## Arquitectura actual

### Generación de `opencode.jsonc`

La configuración de agentes se construye en `cmd/zyrocli/install.go`, función `buildInstallConfig()` (líneas 232-423). Esta función retorna un `*opencode.Config` que contiene un map `Agent` con 15 agentes. Cada agente tiene un bloque `Permission` con las capabilities que OpenCode autoriza.

### Mecanismo de permisos compartidos

Existe una variable `sddPerms` (líneas 236-238) que se asigna directamente a 4 agentes:

```go
sddPerms := map[string]any{
    "read": "allow", "write": "deny", "edit": "deny", "bash": "allow",
}
```

**Importante**: En Go, los maps son tipos referencia. Las variables `zyro-sdd-propose`, `zyro-sdd-design`, `zyro-sdd-tasks`, y `zyro-sdd-archive` **comparten el mismo objeto map** (líneas 351, 367, 373, 379). Cualquier mutación de `sddPerms` impacta a los 4 simultáneamente.

### Permisos actuales (pre-cambio)

| Agente | read | write | edit | bash | task | webfetch | question |
|---|---|---|---|---|---|---|---|
| orchestrator | allow | deny | deny | deny | mapa restrictivo | allow | allow |
| pre-f0 | allow | deny | deny | **allow** | **{"*":"allow"}** | allow | allow |
| phase-0-patterns | allow | deny | deny | **allow** | **{"*":"allow"}** | — | — |
| phase-0-libraries | allow | deny | deny | **allow** | **{"*":"allow"}** | — | — |
| skills-find | allow | deny | deny | allow | **{"*":"allow"}** | allow | — |
| skills-audit | allow | deny | deny | deny | **{"*":"allow"}** | allow | — |
| skills-apply | allow | deny | deny | allow | **{"*":"allow"}** | — | — |
| sdd-apply | allow | allow | allow | allow | **{"*":"allow"}** | — | — |
| sdd-verify | allow | **allow** | deny | allow | **{"*":"allow"}** | — | — |
| sdd-explore | allow | deny | deny | allow | *(sin bloque)* | — | — |
| sdd-propose (vía sddPerms) | allow | deny | deny | **allow** | *(sin bloque)* | — | — |
| sdd-spec | allow | deny | deny | deny | **{"*":"allow"}** | — | — |
| sdd-design (vía sddPerms) | allow | deny | deny | **allow** | *(sin bloque)* | — | — |
| sdd-tasks (vía sddPerms) | allow | deny | deny | **allow** | *(sin bloque)* | — | — |
| sdd-archive (vía sddPerms) | allow | deny | deny | **allow** | *(sin bloque)* | — | — |
| to-issues | allow | deny | deny | allow | *(sin bloque)* | allow | — |

### Problemas identificados

1. **6 agentes con `bash: allow` que su skill no necesita** (líneas 277, 286, 386, 237 a través de sddPerms). El más crítico: `zyro-sdd-design` cuyo SKILL.md dice textualmente **"NO corras bash. NO instales nada."** pero tiene `bash: allow` porque comparte `sddPerms`.

2. **1 agente con `write: allow` que su skill prohíbe**: `zyro-sdd-verify` dice "NO modifiques archivos del proyecto" pero tiene `write: allow` (línea 334).

3. **9 agentes con `task: {"*": "allow"}`**: capacidad de despachar tareas a cualquier agente, incluyendo agentes que no deberían orquestar. Solo el orquestador debe tener facultad de orquestación.

4. **6 agentes sin bloque `task`**: dejan el comportamiento por defecto a criterio de OpenCode (actualmente `deny` pero no explícito).

5. **2 agentes de Fase 0 sin `webfetch`**: phase-0-patterns y phase-0-libraries necesitan `webfetch` para investigar.

6. **1 agente sin `question`**: sdd-explore necesita `question` para entrevistar al usuario.

## Especificación detallada

### 1. Refactor de `sddPerms`

**Cambio**: Mutar el map compartido para cambiar bash de `allow` a `deny`.

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

**Impacto**: Los 3 agentes que siguen compartiendo `sddPerms` heredan `bash: deny`:
- `zyro-sdd-propose`
- `zyro-sdd-design`
- `zyro-sdd-tasks`

### 2. Separación de `zyro-sdd-archive`

**Motivo**: sdd-archive es el único agente SDD que necesita `bash: allow` (skill: "NO corras bash excepto para limpiar archivos temporales").

**Cambio**: Crear un map independiente `sddArchivePerms`:

```go
sddArchivePerms := map[string]any{
    "read": "allow", "write": "deny", "edit": "deny", "bash": "allow",
}
```

Y asignarlo a `zyro-sdd-archive` en lugar de `sddPerms`.

### 3. Tabla completa de permisos post-cambio

| # | Agente | read | write | edit | bash | task | webfetch | question | Cambios respecto al actual |
|---|---|---|---|---|---|---|---|---|---|
| 1 | orchestrator | allow | deny | deny | deny | mapa restrictivo | allow | allow | Sin cambios |
| 2 | **pre-f0** | allow | deny | deny | **deny** | *(sin bloque)* | allow | allow | bash: allow→deny; task: eliminar bloque |
| 3 | **phase-0-patterns** | allow | deny | deny | **deny** | *(sin bloque)* | **allow** | — | bash: allow→deny; webfetch: agregar; task: eliminar bloque |
| 4 | **phase-0-libraries** | allow | deny | deny | **deny** | *(sin bloque)* | **allow** | — | bash: allow→deny; webfetch: agregar; task: eliminar bloque |
| 5 | **skills-find** | allow | deny | deny | allow | *(sin bloque)* | allow | — | task: eliminar bloque |
| 6 | **skills-audit** | allow | deny | deny | deny | *(sin bloque)* | allow | — | task: eliminar bloque |
| 7 | **skills-apply** | allow | deny | deny | allow | *(sin bloque)* | — | — | task: eliminar bloque |
| 8 | **sdd-apply** | allow | allow | allow | allow | *(sin bloque)* | — | — | task: eliminar bloque |
| 9 | **sdd-verify** | allow | **deny** | deny | allow | *(sin bloque)* | — | — | write: allow→deny; task: eliminar bloque |
| 10 | **sdd-explore** | allow | deny | deny | allow | **{"*":"deny"}** | — | **allow** | question: agregar; task: agregar bloque explícito |
| 11 | **sdd-propose** | allow | deny | deny | **deny** (vía sddPerms) | **{"*":"deny"}** | — | — | bash: allow→deny (vía sddPerms); task: agregar bloque explícito |
| 12 | **sdd-spec** | allow | deny | deny | deny | *(sin bloque)* | — | — | task: eliminar bloque |
| 13 | **sdd-design** | allow | deny | deny | **deny** (vía sddPerms) | **{"*":"deny"}** | — | — | bash: allow→deny (vía sddPerms); task: agregar bloque explícito |
| 14 | **sdd-tasks** | allow | deny | deny | **deny** (vía sddPerms) | **{"*":"deny"}** | — | — | bash: allow→deny (vía sddPerms); task: agregar bloque explícito |
| 15 | **sdd-archive** | allow | deny | deny | **allow** (map propio) | **{"*":"deny"}** | — | — | Separado de sddPerms; task: agregar bloque explícito |
| 16 | **to-issues** | allow | deny | deny | allow | **{"*":"deny"}** | allow | — | task: agregar bloque explícito |

### 4. Especificación línea por línea de `install.go`

A continuación se detalla cada cambio concreto en el código:

#### 4.1 `sddPerms` (línea 237)
```
- "bash": "allow"
+ "bash": "deny"
```

#### 4.2 Nueva variable `sddArchivePerms` (insertar después de `sddPerms`)
```
+ sddArchivePerms := map[string]any{
+     "read": "allow", "write": "deny", "edit": "deny", "bash": "allow",
+ }
```

#### 4.3 `zyro-phase-0-patterns` (líneas 276-279)
```
- "bash": "allow", "task": map[string]any{"*": "allow"},
+ "bash": "deny", "webfetch": "allow",
```
Nota: se elimina el bloque `task` por completo (hereda deny por defecto de OpenCode).

#### 4.4 `zyro-phase-0-libraries` (líneas 285-289)
```
- "read": "allow", "bash": "allow",
- "write": "deny", "edit": "deny",
- "task": map[string]any{"*": "allow"},
+ "read": "allow", "bash": "deny", "webfetch": "allow",
+ "write": "deny", "edit": "deny",
```

#### 4.5 `zyro-skills-find` (líneas 295-298)
```
- "task": map[string]any{"*": "allow"},
+ (eliminar línea)
```

#### 4.6 `zyro-skills-audit` (líneas 305-308)
```
- "task": map[string]any{"*": "allow"},
+ (eliminar línea)
```

#### 4.7 `zyro-skills-apply` (líneas 315-318)
```
- "task": map[string]any{"*": "allow"},
+ (eliminar línea)
```

#### 4.8 `zyro-sdd-apply` (líneas 324-327)
```
- "task": map[string]any{"*": "allow"},
+ (eliminar línea)
```

#### 4.9 `zyro-sdd-verify` (líneas 333-335)
```
- "read": "allow", "write": "allow", "edit": "deny",
- "bash": "allow", "task": map[string]any{"*": "allow"},
+ "read": "allow", "write": "deny", "edit": "deny",
+ "bash": "allow",
```

#### 4.10 `zyro-sdd-explore` (líneas 342-344)
```
- "read": "allow", "bash": "allow",
- "write": "deny", "edit": "deny",
+ "read": "allow", "bash": "allow", "question": "allow",
+ "write": "deny", "edit": "deny",
+ "task": map[string]any{"*": "deny"},
```

#### 4.11 `zyro-sdd-propose` (línea 351)
Sin cambios en la línea (sigue siendo `Permission: sddPerms`). El cambio de bash se hereda del map `sddPerms`. Agregar bloque `task` inline **no es posible** porque compartir el map implicaría que el task block se herede a design y tasks también.

**Decisión de diseño**: Para sdd-propose, sdd-design, sdd-tasks, el bloque `task: {"*": "deny"}` debe agregarse **después de la referencia a sddPerms**, sobrescribiendo/complementando. Sin embargo, Go no permite "merge" de maps en una declaración literal.

**Solución**: En lugar de usar `sddPerms` como valor directo, definir el permission map inline para estos 3 agentes, copiando los valores base de `sddPerms` y agregando `task: {"*": "deny"}`. Alternativamente, mantener `sddPerms` para los permisos base y crear maps separados para cada uno que incluyan `task`.

**Alternativa recomendada (mínimo cambio)**: Expandir los 3 agentes a maps inline:

```go
// zyro-sdd-propose
Permission: map[string]any{
    "read": "allow", "write": "deny", "edit": "deny", "bash": "deny",
    "task": map[string]any{"*": "deny"},
},
// zyro-sdd-design (idéntico)
// zyro-sdd-tasks (idéntico)
```

**Ventaja**: Clara intención, sin dependencias ocultas. **Desventaja**: Duplicación de 3 líneas.

**Alternativa más limpia (recomendada)**: Mantener `sddPerms` solo para sdd-propose, sdd-design, sdd-tasks PERO crear una función helper que agregue task deny:

```go
// helper al inicio de buildInstallConfig
permsWithTaskDeny := func(base map[string]any) map[string]any {
    m := make(map[string]any, len(base)+1)
    for k, v := range base {
        m[k] = v
    }
    m["task"] = map[string]any{"*": "deny"}
    return m
}
```

Y usarlo como:
```go
"zyro-sdd-propose": { ..., Permission: permsWithTaskDeny(sddPerms), },
"zyro-sdd-design":  { ..., Permission: permsWithTaskDeny(sddPerms), },
"zyro-sdd-tasks":   { ..., Permission: permsWithTaskDeny(sddPerms), },
```

**Decisión final**: Usar la función helper `permsWithTaskDeny` para evitar duplicación y mantener un solo punto de cambio para permisos base SDD. Esto se especifica porque:

1. Los 3 agentes comparten la misma base de permisos (read/write/edit/bash) y solo difieren en el bloque task.
2. Si en el futuro se agrega un permiso a sddPerms, los 3 lo heredan automáticamente.
3. Reduce el riesgo de que un agente quede con permisos inconsistentes.

#### 4.12 `zyro-sdd-spec` (líneas 357-360)
```
- "task": map[string]any{"*": "allow"},
+ (eliminar línea)
```

#### 4.13 `zyro-sdd-design` (línea 367)
Cambiar de `Permission: sddPerms` a `Permission: permsWithTaskDeny(sddPerms)` (o inline).

#### 4.14 `zyro-sdd-tasks` (línea 373)
Ídem sdd-design.

#### 4.15 `zyro-sdd-archive` (línea 379)
```
- Permission: sddPerms,
+ Permission: sddArchivePerms,
```
Y agregar bloque task inline al map `sddArchivePerms`:
```go
sddArchivePerms := map[string]any{
    "read": "allow", "write": "deny", "edit": "deny", "bash": "allow",
    "task": map[string]any{"*": "deny"},
}
```

#### 4.16 `zyro-pre-f0` (líneas 385-388)
```
- "read": "allow", "bash": "allow", "webfetch": "allow", "question": "allow",
- "write": "deny", "edit": "deny",
- "task": map[string]any{"*": "allow"},
+ "read": "allow", "bash": "deny", "webfetch": "allow", "question": "allow",
+ "write": "deny", "edit": "deny",
```

#### 4.17 `to-issues` (líneas 395-398)
```
- "read": "allow", "bash": "allow", "webfetch": "allow",
- "write": "deny", "edit": "deny",
+ "read": "allow", "bash": "allow", "webfetch": "allow",
+ "write": "deny", "edit": "deny",
+ "task": map[string]any{"*": "deny"},
```

### 5. Resumen de cambios por categoría

| Categoría | Cantidad | Agentes |
|---|---|---|
| bash: allow → deny | 6 | pre-f0, phase-0-patterns, phase-0-libraries, sdd-design, sdd-tasks, sdd-propose |
| write: allow → deny | 1 | sdd-verify |
| task block eliminado (hereda deny) | 9 | pre-f0, phase-0-patterns, phase-0-libraries, skills-find, skills-audit, skills-apply, sdd-apply, sdd-spec, sdd-verify |
| task: {"*": "deny"} agregado explícito | 6 | sdd-explore, sdd-propose, sdd-design, sdd-tasks, sdd-archive, to-issues |
| webfetch: allow agregado | 2 | phase-0-patterns, phase-0-libraries |
| question: allow agregado | 1 | sdd-explore |
| Separado de sddPerms | 1 | sdd-archive |

### 6. Flujo de datos post-cambio

```
buildInstallConfig()
  ├── sddPerms (bash: deny) ──────────────► sdd-propose (con permsWithTaskDeny)
  │                                          ├── sdd-design (con permsWithTaskDeny)
  │                                          └── sdd-tasks (con permsWithTaskDeny)
  ├── sddArchivePerms (bash: allow) ────────► sdd-archive
  ├── maps inline c/ permiso específico ───► los otros 12 agentes
  │
  └── opencode.WriteGlobalConfig(cfg)
        │
        ▼
  ~/.config/opencode/opencode.jsonc
        │
        ▼
  OpenCode aplica permisos al ejecutar subagentes
```

## Criterios de éxito

- [ ] **CE1**: `zyro pre-f0` ejecuta sin errores de permiso (bash:deny, pero tiene question:allow para entrevistar)
- [ ] **CE2**: `zyro phase-0-patterns` ejecuta con webfetch:allow y bash:deny
- [ ] **CE3**: `zyro phase-0-libraries` ejecuta con webfetch:allow y bash:deny
- [ ] **CE4**: `zyro sdd-design` ejecuta con bash:deny (ya no puede correr bash, alineado con su skill)
- [ ] **CE5**: `zyro sdd-tasks` ejecuta con bash:deny
- [ ] **CE6**: `zyro sdd-propose` ejecuta con bash:deny
- [ ] **CE7**: `zyro sdd-verify` ejecuta con write:deny (ya no puede modificar archivos, alineado con su skill)
- [ ] **CE8**: `zyro sdd-explore` puede hacer preguntas al usuario (question:allow)
- [ ] **CE9**: Ningún subagente (excepto el orquestador) puede despachar tareas a otros agentes
- [ ] **CE10**: `zyro sdd-archive` mantiene bash:allow para limpiar archivos temporales
- [ ] **CE11**: `zyrocli install` genera opencode.jsonc con permisos correctos

## Pruebas

### Prueba de regresión (manual / automatizable)

**Comando**: `zyrocli install --dry-run` (si existe) o inspeccionar el archivo generado:

```bash
zyrocli install 2>/dev/null
cat ~/.config/opencode/opencode.jsonc | grep -A 10 '"zyro-sdd-design"' | grep '"bash"'
# Debe mostrar: "bash": "deny"
```

### Script de verificación post-instalación

Script bash que verifica los 11 criterios de éxito parseando `~/.config/opencode/opencode.jsonc`:

```bash
#!/bin/bash
# verify-permissions.sh
CONFIG=~/.config/opencode/opencode.jsonc

# Helper: extraer valor de permiso para un agente
get_perm() {
  agent=$1
  perm=$2
  python3 -c "
import json
with open('$CONFIG') as f:
    cfg = json.load(f)
agent = cfg.get('agent', {}).get('$agent', {})
perms = agent.get('permission', {})
val = perms.get('$perm', '<not found>')
print(val)
"
}

# Helper: verificar si task block existe
has_task_block() {
  agent=$1
  python3 -c "
import json
with open('$CONFIG') as f:
    cfg = json.load(f)
agent = cfg.get('agent', {}).get('$agent', {})
perms = agent.get('permission', {})
print('task' in perms)
"
}

# CE1: pre-f0 bash=deny, question=allow
[[ "$(get_perm zyro-pre-f0 bash)" == "deny" ]] && echo "✓ CE1a" || echo "✗ CE1a"
[[ "$(get_perm zyro-pre-f0 question)" == "allow" ]] && echo "✓ CE1b" || echo "✗ CE1b"

# CE2: phase-0-patterns bash=deny, webfetch=allow
[[ "$(get_perm zyro-phase-0-patterns bash)" == "deny" ]] && echo "✓ CE2a" || echo "✗ CE2a"
[[ "$(get_perm zyro-phase-0-patterns webfetch)" == "allow" ]] && echo "✓ CE2b" || echo "✗ CE2b"

# CE3: phase-0-libraries bash=deny, webfetch=allow
[[ "$(get_perm zyro-phase-0-libraries bash)" == "deny" ]] && echo "✓ CE3a" || echo "✗ CE3a"
[[ "$(get_perm zyro-phase-0-libraries webfetch)" == "allow" ]] && echo "✓ CE3b" || echo "✗ CE3b"

# CE4: sdd-design bash=deny
[[ "$(get_perm zyro-sdd-design bash)" == "deny" ]] && echo "✓ CE4" || echo "✗ CE4"

# CE5: sdd-tasks bash=deny
[[ "$(get_perm zyro-sdd-tasks bash)" == "deny" ]] && echo "✓ CE5" || echo "✗ CE5"

# CE6: sdd-propose bash=deny
[[ "$(get_perm zyro-sdd-propose bash)" == "deny" ]] && echo "✓ CE6" || echo "✗ CE6"

# CE7: sdd-verify write=deny
[[ "$(get_perm zyro-sdd-verify write)" == "deny" ]] && echo "✓ CE7" || echo "✗ CE7"

# CE8: sdd-explore question=allow
[[ "$(get_perm zyro-sdd-explore question)" == "allow" ]] && echo "✓ CE8" || echo "✗ CE8"

# CE9: Ningún subagente (excepto orchestrator) tiene task block
for agent in zyro-pre-f0 zyro-phase-0-patterns zyro-phase-0-libraries \
             zyro-skills-find zyro-skills-audit zyro-skills-apply \
             zyro-sdd-apply zyro-sdd-spec zyro-sdd-verify; do
  [[ "$(has_task_block $agent)" == "False" ]] && echo "✓ CE9: $agent sin task" || echo "✗ CE9: $agent tiene task"
done

# CE9b: agentes con task explícito deny
for agent in zyro-sdd-explore zyro-sdd-propose zyro-sdd-design \
             zyro-sdd-tasks zyro-sdd-archive to-issues; do
  val=$(get_perm $agent task)
  [[ "$val" == "deny" ]] || [[ "$val" == \{\"*\":\ \"deny\"\} ]] && echo "✓ CE9b: $agent task deny" || echo "✗ CE9b: $agent task no es deny"
done

# CE10: sdd-archive bash=allow
[[ "$(get_perm zyro-sdd-archive bash)" == "allow" ]] && echo "✓ CE10" || echo "✗ CE10"

echo "---"
echo "Verificación completa"
```

### Verificación manual de un agente específico

```bash
# Verificar permisos de sdd-design (el caso más crítico)
python3 -c "
import json
cfg = json.load(open('/home/secko/.config/opencode/opencode.jsonc'))
a = cfg['agent']['zyro-sdd-design']
print(json.dumps(a['permission'], indent=2))
"
# Output esperado:
# {
#   "read": "allow",
#   "write": "deny",
#   "edit": "deny",
#   "bash": "deny",
#   "task": {"*": "deny"}
# }
```

## Riesgos

### 1. Regresión por mapas compartidos (ALTO)

`sddPerms` es un map compartido por referencia en Go. Si la implementación no separa correctamente a `sdd-archive`, este heredará `bash: deny` y perderá su capacidad de limpiar temporales.

**Mitigación**: Verificar explícitamente que `sdd-archive` tenga `bash: allow` en el script de verificación (CE10).

### 2. `sdd-verify` sin write (BAJO)

Si en el futuro `sdd-verify` necesita escribir reportes locales, este permiso deberá reconsiderarse. Hoy su skill dice explícitamente que no modifica archivos — su output es HelixDB.

**Mitigación**: Documentado en el spec. Si surge la necesidad, es un cambio de 1 línea.

### 3. Task default de OpenCode cambie a `allow` (BAJO)

Si OpenCode cambia su default de `deny` a `allow`, los 9 agentes sin bloque task explícito heredarían comportamiento incorrecto.

**Mitigación**: Este spec agrega `task: {"*": "deny"}` explícito para 6 agentes (los que no tenían bloque task). Los otros 9 tenían `task: {"*": "allow"}` que se elimina — si OpenCode cambia su default, estos 9 quedarían con `allow` implícito. Para mitigar completamente, se podría agregar `task: {"*": "deny"}` explícito a todos, pero la propuesta opta por eliminarlo donde estaba `allow` para minimizar cambios.

### 4. `sddPerms` compartida sigue siendo frágil (BAJO)

Si en el futuro un agente (propose, design o tasks) necesita un permiso diferente, habrá que separarlo de `sddPerms`.

**Mitigación**: Aceptado como deuda técnica documentada. El map compartido reduce duplicación para el caso actual donde los 3 agentes tienen permisos idénticos.

### 5. Riesgo de error humano al hacer 17 cambios en un solo archivo (MEDIO)

Aunque los cambios son mecánicos, hay 17 modificaciones puntuales en un solo archivo. Un descuido (olvidar cambiar un permiso, dejar una coma mal puesta, etc.) puede generar un JSON inválido.

**Mitigación**: `go build` detectará errores de sintaxis Go. El script de verificación post-instalación (CE11) detectará permisos incorrectos. Se recomienda ejecutar `zyrocli install` y verificar con el script antes de hacer commit.
