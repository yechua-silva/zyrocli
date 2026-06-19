# Archive — Fix 3 Issues del Instalador

## Fecha
2026-06-17

## Fases completadas
- ✅ F0: Exploración de problemas (GPU, HelixDB, subagentes)
- ✅ F1: Spec escrita en sdd/spec-fix-3issues.md
- ✅ F2: Design + Tasks en sdd/design-fix-3issues.md
- ✅ F3: Implementación completa
- ✅ F4: Archive

## Cambios realizados

### 1. GPU installer automático (scripts/install_tui.py)
- Nuevas funciones: `_auto_configure_rocm()`, `_auto_configure_vulkan()`
- Post-instalación ejecuta: modprobe amdkfd → persiste módulo → mata ollama → arranca con HSA_OVERRIDE_GFX_VERSION=8.0.3 + OLLAMA_GPU_DRIVER=rocm → espera → verifica backend → guarda env vars en ~/.bashrc
- Archivos: scripts/install_tui.py (+148 líneas)

### 2. HelixDB post-install (internal/db/helix/client.go + cmd/zyrocli/install.go)
- Nueva función: `startHelixContainer()` — busca helix CLI o docker compose
- `EnsureStarted()` ahora inicia el container si ping falla (con retry 15s)
- El paso de instalación ahora es fatal (no se salta si falla)
- Archivos: internal/db/helix/client.go, cmd/zyrocli/install.go

### 3. Subagentes arreglados (internal/boomerang/ + cmd/zyrocli/mcp_server.go)
- `executeTask()` ya no ejecuta CLI `opencode subagent` que no existe
- Nueva función: `CompleteTask()` en TaskManager
- `DelegateStep()` ya no ejecuta CLI
- Nuevo tool MCP: `complete_task`
- Archivos: task_manager.go, delegate.go, mcp_server.go

### 4. MCP_DIR_PLACEHOLDER resuelto
- Código fuente: cmd/zyrocli/install.go ahora resuelve la ruta dinámicamente
- Config existente: ~/.config/opencode/opencode.jsonc actualizado con ruta real

### 5. Limpieza de sistema
- Binario engram eliminado de ~/.local/bin/
- MCP servers context7 y engram eliminados de ~/.claude/mcp/
- Plugin engram eliminado de ~/.claude/settings.json
- .codex/config.toml MCP engram eliminado

## Estado final
- Build: ✅ go build ./... exitoso
- Tests: ✅ Todos pasan (56/56 boomerang, setup, etc.)
- Binary: 17MB, tools MCP funcionando (6 tools: dispatch, check, wait, list, cancel, complete)
- HelixDB: ✅ Corriendo en :6969
