# Design: Corrección de 7 bugs en PRD Upgrade

## Resumen
7 bugs identificados. 3 archivos principales afectados: cmd/zyrocli/install.go, internal/opencode/config.go, internal/tui/brand.go, internal/tui/install.go.

## Bug 1: buildInstallConfig() sin agentes nuevos

### Archivo
cmd/zyrocli/install.go, función buildInstallConfig() (~línea 175)

### Cambio
Agregar al mapa de agentes:
- zyro-pre-f0 como subagent con permisos de lectura, bash, webfetch, question
- to-issues como subagent con permisos de lectura, bash, webfetch
- Agregar "zyro-pre-f0": "allow" a la lista de subagentes del orquestador

### Riesgo
Bajo. Solo agregar entradas a un mapa.

### Dependencias
Ninguna.

## Bug 2: WriteGlobalConfig() pisa config existente

### Archivo
internal/opencode/config.go (~línea 71-88)

### Cambio
Reemplazar os.WriteFile directo por:
1. Leer JSON existente si existe
2. Unmarshal a Config
3. Fusionar MCP servers: los del usuario tienen prioridad, solo agregar nuevos
4. Fusionar agentes: agregar nuevos sin borrar existentes
5. Escribir JSON final

### Riesgo
Medio. Si el JSON existente está mal formado, no fallar sino sobrescribir limpio.

### Dependencias
Ninguna.

## Bug 3: gitMCP como remote URL inservible

### Archivo
cmd/zyrocli/install.go (~línea 340-343)

### Cambio
Reemplazar:
```go
"gitmcp": { Type: "remote", URL: "https://gitmcp.io/docs" }
```
Por:
```go
"gitmcp": { Type: "local", Command: []string{"uvx", "mcp-server-git", "--repository", "."} }
```

### Riesgo
Bajo. Cambio de config inline.

### Dependencias
Requiere que `uvx` esté instalado (ya se usa en helix-integration MCP).

## Bug 3b: Context MCP merge con config existente

### Archivo
cmd/zyrocli/install.go (~línea 336-339) + internal/opencode/config.go

### Cambio
El fix está en Bug 2 (merge). Si el usuario ya tenía context configurado, el merge lo preserva. Si no, se agrega con el default.

Adicionalmente, verificar el comando real de context:
- Si `context serve` no funciona, probar `context mcp` o el comando correcto

### Riesgo
Bajo si merge funciona. El comando exacto de context hay que verificarlo.

### Dependencias
Bug 2 (merge).

## Bug 4: ASCII art 3D se corrompe

### Archivo
internal/tui/brand.go (~línea 96-101)

### Cambio
Modificar sanitizeArt() para NO trimear espacios:
```go
func sanitizeArt(art string) string {
    return strings.ReplaceAll(art, "\r\n", "\n")
}
```

Además modificar BrandLines() para preservar estructura exacta:
```go
func BrandLines() []string {
    return strings.Split(brandArt, "\n")
}
```

### Riesgo
Bajo. Función pura, sin side effects.

### Dependencias
Ninguna.

## Bug 5: Scroll acumulativo en TUI

### Archivo
internal/tui/install.go (~línea 175, 207)

### Cambios
1. Eliminar tea.Printf (línea 175) — reemplazar por return tea.Quit sin Printf
2. En View(), limitar salida por m.height:
   - Si el contenido excede m.height-2, truncar líneas
   - Agregar indicador "[... scroll oculto ...]"

### Riesgo
Bajo. Solo afecta renderizado del installer TUI.

### Dependencias
Ninguna.

## Bug 6: Logo OpenCode trailing spaces

### Archivo
internal/tui/assets/logo.txt

### Cambio
El fix es el mismo que Bug 4. sanitizeArt() ya no trimea espacios, así que el logo se preserva.

### Riesgo
Ninguno.

### Dependencias
Bug 4 (sanitizeArt).

## Resumen de dependencias entre bugs

Bug 4 ─┐
         ├── Bug 6 (misma causa raíz)
         │
Bug 2 ──┼── Bug 3b (merge resuelve ambos)
         │
Bug 1 ──┘ (independiente)
Bug 3 ─── (independiente)
Bug 5 ─── (independiente)

## Orden de implementación sugerido
1. Bug 4 + Bug 6 (sanitizeArt)
2. Bug 2 (merge) → Bug 3b (context)
3. Bug 1 (agentes nuevos)
4. Bug 3 (gitMCP)
5. Bug 5 (scroll TUI)
