# Tasks: Corrección de 7 bugs en PRD Upgrade

## Orden de implementación
Las dependencias ya están resueltas en el diseño. Se implementan en orden ascendente.

---

### Task 1: Fix sanitizeArt() — no trimear espacios en ASCII art

**Archivo**: internal/tui/brand.go (~línea 96-101)
**Depende de**: Nada
**Estimación**: 5 min

**Cambio**: 
- Modificar sanitizeArt() para que solo normalice \r\n a \n, sin hacer TrimRight
- Modificar BrandLines() para que no trimee el \n final

**Criterio de aceptación**:
- [ ] Dado el arte ASCII en brand.txt, cuando se renderiza, entonces los espacios del diseño 3D se preservan
- [ ] Dado BrandLines(), cuando se llama, entonces retorna estructura exacta sin pérdida de líneas

---

### Task 2: Fix WriteGlobalConfig() — merge en vez de pisar

**Archivo**: internal/opencode/config.go (~línea 71-88)
**Depende de**: Nada
**Estimación**: 15 min

**Cambio**:
- Leer JSON existente si el archivo ya existe
- Unmarshal a Config
- Fusionar MCP servers: los del usuario tienen prioridad, solo agregar los que no existen
- Fusionar agentes: agregar nuevos sin borrar existentes
- Escribir JSON final

**Criterio de aceptación**:
- [ ] Dado un opencode.jsonc existente con MCP servers manuales, cuando se ejecuta WriteGlobalConfig, entonces los MCP servers manuales se preservan
- [ ] Dado un opencode.jsonc existente con agentes, cuando se ejecuta WriteGlobalConfig, entonces los agentes nuevos se agregan sin borrar los existentes
- [ ] Dado un archivo JSON mal formado, cuando se ejecuta WriteGlobalConfig, entonces NO falla y escribe la config nueva limpia

---

### Task 3: Fix buildInstallConfig() — agregar zyro-pre-f0 y to-issues

**Archivo**: cmd/zyrocli/install.go (~línea 175-357)
**Depende de**: Nada
**Estimación**: 10 min

**Cambio**:
1. Agregar entrada "zyro-pre-f0" al mapa de agentes con permisos: read allow, bash allow, webfetch allow, question allow, write deny, edit deny, task * allow
2. Agregar entrada "to-issues" al mapa de agentes con permisos: read allow, bash allow, webfetch allow, write deny, edit deny
3. Agregar "zyro-pre-f0": "allow" a la lista de subagentes permitidos del orquestador (en la sección "task" del permission map)

**Criterio de aceptación**:
- [ ] Dado buildInstallConfig(), cuando se ejecuta, entonces el mapa de agentes contiene "zyro-pre-f0"
- [ ] Dado buildInstallConfig(), cuando se ejecuta, entonces el mapa de agentes contiene "to-issues"
- [ ] Dado el permission del orquestador, cuando se lista task, entonces contiene "zyro-pre-f0": "allow"

---

### Task 4: Fix gitMCP — reemplazar remote URL por servidor local

**Archivo**: cmd/zyrocli/install.go (~línea 340-343)
**Depende de**: Nada
**Estimación**: 5 min

**Cambio**:
Reemplazar:
```go
"gitmcp": { Type: "remote", URL: "https://gitmcp.io/docs" }
```
Por:
```go
"gitmcp": { Type: "local", Command: []string{"uvx", "mcp-server-git", "--repository", "."} }
```

**Criterio de aceptación**:
- [ ] Dado el MCP map en buildInstallConfig(), cuando se consulta "gitmcp", entonces Type es "local" y Command contiene "uvx" y "mcp-server-git"

---

### Task 5: Fix scroll acumulativo en TUI installer

**Archivo**: internal/tui/install.go (~línea 175, línea 207-233)
**Depende de**: Nada
**Estimación**: 10 min

**Cambios**:
1. Eliminar tea.Printf de la línea 175 — reemplazar por tea.Quit sin Printf
2. En View(), limitar salida por m.height:
   ```go
   if m.height > 0 {
       lines := strings.Split(content, "\n")
       maxLines := m.height - 3
       if len(lines) > maxLines {
           lines = lines[:maxLines]
           lines = append(lines, helpStyle.Render("  [... más ...]"))
           content = strings.Join(lines, "\n")
       }
   }
   ```

**Criterio de aceptación**:
- [ ] Dado el installer TUI, cuando se completa, entonces NO hay mensaje "✓ All N steps completed" en el log
- [ ] Dado el installer TUI con altura limitada, cuando el contenido excede la pantalla, entonces se trunca con indicador

---

### Task 6: Verificar y corregir comando de context MCP

**Archivo**: cmd/zyrocli/install.go (~línea 336-339)
**Depende de**: Task 2 (merge)
**Estimación**: 10 min + verificación

**Cambio**:
1. Verificar cuál es el comando MCP real de @neuledge/context ejecutando `context --help` y buscando subcomando MCP
2. Si el comando actual (`context serve`) es incorrecto, actualizarlo
3. El merge de Task 2 ya preserva config personalizada del usuario

**Criterio de aceptación**:
- [ ] Dado el MCP map en buildInstallConfig(), cuando se consulta "context", entonces el comando es el correcto para servir MCP
- [ ] Dado WriteGlobalConfig con merge, cuando el usuario tenía context configurado, entonces se preserva

---

## Resumen

| Task | Archivo | Depende de | Estimación |
|------|---------|-----------|-----------|
| 1 | internal/tui/brand.go | — | 5 min |
| 2 | internal/opencode/config.go | — | 15 min |
| 3 | cmd/zyrocli/install.go | — | 10 min |
| 4 | cmd/zyrocli/install.go | — | 5 min |
| 5 | internal/tui/install.go | — | 10 min |
| 6 | cmd/zyrocli/install.go | Task 2 | 10 min |

**Total estimado**: 55 min
