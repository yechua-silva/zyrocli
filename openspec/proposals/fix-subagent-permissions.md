# Propuesta: Corrección de permisos de subagentes OpenCode

## Intento

Los permisos declarados en `opencode.jsonc` para los subagentes de ZyroCLI no están alineados con las responsabilidades reales que define cada SKILL.md. Esto genera riesgos de seguridad operativa:

1. **Agentes con `bash: allow` innecesario**: 6 subagentes pueden ejecutar comandos arbitrarios en el shell cuando su skill explícitamente dice que no deben hacerlo. El caso más grave es `zyro-sdd-design`, cuyo SKILL.md dice textualmente **"NO corras bash. NO instales nada."** pero tiene `bash: allow`.

2. **Agente con `write: allow` que viola su skill**: `zyro-sdd-verify` dice "NO modifiques archivos del proyecto" pero tiene `write: allow`.

3. **Task dispatch sin restricciones**: 9 subagentes tienen `task: {"*": "allow"}`, lo que les permite despachar tareas a **cualquier agente**, incluyendo agentes que no deberían invocar. Solo el orquestador debería tener capacidad de orquestación.

4. **Permisos faltantes**: `zyro-phase-0-patterns` y `zyro-phase-0-libraries` necesitan `webfetch` para investigar, pero no lo tienen. `zyro-sdd-explore` necesita `question` para entrevistar al usuario.

5. **Permisos no declarados explícitamente**: 6 agentes no declaran `task` en su bloque de permisos, dejando comportamiento por defecto incierto.

El problema duele hoy porque cualquier agente con bash puede, por error o desvío, ejecutar comandos destructivos en el sistema. Cerrar estos permisos es una operación de hardening necesaria antes de escalar el uso de ZyroCLI.

## Alcance

### Incluye

1. **`bash: allow` → `bash: deny`** para estos 6 agentes:
   - `zyro-pre-f0` — no necesita bash (skill: solo entrevistar + escribir archivos markdown)
   - `zyro-phase-0-patterns` — no necesita bash (skill: solo webfetch + HelixDB)
   - `zyro-phase-0-libraries` — no necesita bash (skill: solo webfetch + Context + GitMCP + HelixDB)
   - `zyro-sdd-design` — **contradicción directa** (skill: "NO corras bash. NO instales nada.")
   - `zyro-sdd-tasks` — no necesita bash (skill: solo leer HelixDB + guardar tasks)
   - `zyro-sdd-propose` — no necesita bash (skill: solo leer archivos + escribir propuesta)

2. **`write: allow` → `write: deny`** para `zyro-sdd-verify`:
   - Skill dice: "NO modifiques archivos del proyecto." — contradicción directa.

3. **`task: {"*": "allow"}` → sin bloque task** (hereda deny por defecto) para 9 agentes:
   - `zyro-pre-f0`, `zyro-phase-0-patterns`, `zyro-phase-0-libraries`
   - `zyro-skills-find`, `zyro-skills-audit`, `zyro-skills-apply`
   - `zyro-sdd-spec`, `zyro-sdd-apply`, `zyro-sdd-verify`
   - La orquestación es potestad exclusiva del orquestador.

4. **Agregar `webfetch: allow`** a:
   - `zyro-phase-0-patterns` — skill: "Busca en internet proyectos similares con webfetch"
   - `zyro-phase-0-libraries` — skill: "Investigá librerías recomendadas... (webfetch, Context MCP, GitMCP)"

5. **Agregar `question: allow`** a:
   - `zyro-sdd-explore` — skill: "Entrevistá al usuario... Una pregunta a la vez..."

6. **Agregar `task: {"*": "deny"}` explícito** a 6 agentes que actualmente no declaran task:
   - `zyro-sdd-design`, `zyro-sdd-tasks`, `zyro-sdd-archive`
   - `zyro-sdd-explore`, `zyro-sdd-propose`, `to-issues`
   - Mejor práctica de seguridad: declarar explícitamente, no confiar en defaults.

### Excluye

- **No se modifican** los permisos de agentes que **SÍ** necesitan bash por su skill:
  - `zyro-skills-find` (bash: allow — necesita `npx skills find`)
  - `zyro-skills-apply` (bash: allow — necesita `npx skills add`)
  - `zyro-sdd-apply` (bash: allow — necesita correr tests)
  - `zyro-sdd-archive` (bash: allow — skill dice "NO corras bash excepto para limpiar archivos temporales")
  - `zyro-sdd-explore` (bash: allow — skill dice "NO corras bash excepto para grep, find, o leer archivos")
  - `to-issues` (bash: allow — necesita `gh` o `curl`)
- **No se rediseña** el modelo de permisos granular (OpenCode no soporta allow-except-for-list). Eso queda para futura versión de OpenCode.
- **No se toca** la config del orquestador (`zyro-orchestrator`), que ya está correcta.
- **No se modifica** comportamiento de agentes, solo permisos.

## Enfoque

### Archivo a modificar

`cmd/zyrocli/install.go` — función `buildInstallConfig()`, que genera la estructura de permisos que luego se escribe en `opencode.jsonc`.

### Estrategia

Actualmente existe una variable compartida `sddPerms` usada por 3 agentes (sdd-propose, sdd-design, sdd-tasks, sdd-archive) que define:

```go
sddPerms := map[string]any{
    "read": "allow", "write": "deny", "edit": "deny", "bash": "allow",
}
```

El enfoque es:

1. **Refactorizar `sddPerms`**: cambiar `"bash": "allow"` → `"bash": "deny"`. Como `sddPerms` es compartida, este cambio impacta a todos los agentes que la usan (propose, design, tasks) automáticamente.

2. **`sdd-archive` se separa** de `sddPerms` porque necesita `"bash": "allow"` (skill: "NO corras bash excepto para limpiar archivos temporales"). Se le da su propio map con bash allow.

3. **Permisos individuales** que deben modificarse uno por uno:

| Agente | Cambio |
|---|---|
| `zyro-pre-f0` | `bash: allow`→`deny`, task: eliminar (heredar deny) |
| `zyro-phase-0-patterns` | `bash: allow`→`deny`, agregar `webfetch: allow`, task: eliminar |
| `zyro-phase-0-libraries` | `bash: allow`→`deny`, agregar `webfetch: allow`, task: eliminar |
| `zyro-skills-find` | task: eliminar |
| `zyro-skills-audit` | task: eliminar (ya tiene bash:deny correcto) |
| `zyro-skills-apply` | task: eliminar (bash:allow correcto) |
| `zyro-sdd-apply` | task: eliminar (bash:allow correcto) |
| `zyro-sdd-spec` | task: eliminar (bash:deny correcto) |
| `zyro-sdd-verify` | `write: allow`→`deny`, `bash: allow`→mantener (skill permite bash para tests), task: eliminar |
| `zyro-sdd-explore` | agregar `question: allow`, agregar `task: {"*": "deny"}` |
| `zyro-sdd-design` | ya cambia via sddPerms, agregar `task: {"*": "deny"}` |
| `zyro-sdd-tasks` | ya cambia via sddPerms, agregar `task: {"*": "deny"}` |
| `zyro-sdd-archive` | separar de sddPerms, mantener bash:allow, agregar `task: {"*": "deny"}` |
| `zyro-sdd-propose` | ya cambia via sddPerms, agregar `task: {"*": "deny"}` |
| `to-issues` | agregar `task: {"*": "deny"}` |

### Flujo de datos

```
buildInstallConfig() → genera map[string]Agent con permissions corregidas
         ↓
opencode.WriteGlobalConfig(cfg) → serializa a JSON → ~/.config/opencode/opencode.jsonc
         ↓
OpenCode lee al iniciar → subagentes ejecutan con permisos restringidos
```

### Nota sobre `sdd-archive`

Aunque sdd-archive necesita bash (para limpiar `.zyro/` temporales), esto es seguro porque su alcance es muy limitado. Se mantiene `bash: allow` pero se le agrega `task: {"*": "deny"}` explícito.

## Riesgos

### Trade-offs identificados

1. **`sdd-verify` pierde write**: Si en el futuro `sdd-verify` necesita escribir reportes locales, habrá que reconsiderar. Hoy su skill dice explícitamente que no modifica archivos — su output es HelixDB.

2. **`sdd-archive` separado de sddPerms**: Aumenta ligeramente el código duplicado (un map extra), pero es necesario porque sdd-archive es el único de los 4 agentes SDD que sí necesita bash.

3. **Task sin bloque explícito**: Si OpenCode cambia su default de `deny` a `allow` en el futuro, los agentes sin task block explícito heredarían behavior incorrecto. El estándar actual de OpenCode es deny por defecto, pero el enfoque propuesto **sí** agrega `task: {"*": "deny"}` explícito para 6 agentes, mitigando este riesgo.

### Impacto en otras áreas

- **Ningún agente de Fase 0** (patterns, libraries, skills-find, skills-audit, skills-apply) podrá despachar tareas — correcto, porque esa responsabilidad es del orquestador.
- **zyro-sdd-design pierde bash** — esto es explícitamente requerido por su SKILL.md. Si alguien necesita que design corra bash, es un error en el diseño del agente, no en el permiso.
- **zyro-pre-f0 pierde bash** — este agente solo entrevista y escribe markdown. No hay scenario donde necesite bash.

### Deuda técnica que se genera

- `sddPerms` compartida es frágil: un cambio afecta a 3 agentes. Si en el futuro un agente necesita un permiso distinto, habrá que separarlo. Esto es aceptable porque está documentado.
- No hay tests que verifiquen que los permisos en `install.go` coincidan con los SKILL.md. Idealmente se agregaría un test de snapshot, pero está fuera de este alcance.

## Esfuerzo estimado

**Small** — ~15-20 líneas netas de cambio en un solo archivo (`install.go`). Los cambios son mecánicos (cambiar strings en un map). No hay lógica nueva, no hay refactor profundo, no hay cambios en el schema de OpenCode.

Archivos afectados: 1 (`cmd/zyrocli/install.go`)
Complejidad: Baja (solo permutación de permisos en maps)
Riesgo de regresión: Bajo-Moderado (los permisos solo afectan lo que el agente puede hacer, no lo que hace)

## Criterios de éxito

- [ ] `zyro pre-f0` ejecuta sin errores de permiso (bash:deny, pero tiene question:allow para entrevistar)
- [ ] `zyro phase-0-patterns` ejecuta con webfetch:allow y bash:deny
- [ ] `zyro phase-0-libraries` ejecuta con webfetch:allow y bash:deny
- [ ] `zyro sdd-design` ejecuta con bash:deny (ya no puede correr bash, alineado con su skill)
- [ ] `zyro sdd-tasks` ejecuta con bash:deny
- [ ] `zyro sdd-propose` ejecuta con bash:deny
- [ ] `zyro sdd-verify` ejecuta con write:deny (ya no puede modificar archivos, alineado con su skill)
- [ ] `zyro sdd-explore` puede hacer preguntas al usuario (question:allow)
- [ ] Ningún subagente (excepto el orquestador) puede despachar tareas a otros agentes
- [ ] `zyro sdd-archive` mantiene bash:allow para limpiar archivos temporales
- [ ] Test de regresión: `zyrocli install` genera opencode.jsonc con permisos correctos
