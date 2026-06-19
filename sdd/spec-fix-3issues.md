# Spec — Fix 3 Issues del Instalador

## Issue 1: GPU installer automático

### Problema
En `scripts/install_tui.py`, función `paso5_gpu()`, cuando el usuario selecciona ROCm (opción 2):
1. ✅ Instala `ollama-rocm-bin` con `yay`
2. ❌ NO ejecuta los pasos post-instalación automáticamente
3. ❌ Solo IMPRIME instrucciones como `sudo modprobe amdkfd`, `export HSA_OVERRIDE_GFX_VERSION=8.0.3`
4. ❌ Verifica backend SIN haber reiniciado `ollama serve` → siempre muestra "Backend sigue siendo CPU"

### Solución
Modificar `paso5_gpu()` en `scripts/install_tui.py` para que después de instalar `ollama-rocm-bin`:

1. **Cargar módulo amdkfd si no está:**
   ```python
   if not _check_amdkfd_module():
       subprocess.run(["sudo", "modprobe", "amdkfd"])
       subprocess.run(["sudo", "tee", "/etc/modules-load.d/amdkfd.conf"], input=b"amdkfd\n")
   ```

2. **Matar proceso ollama:**
   ```python
   subprocess.run(["pkill", "ollama"], capture_output=True)
   ```

3. **Setear variables de entorno y arrancar ollama:**
   ```python
   env = os.environ.copy()
   env["HSA_OVERRIDE_GFX_VERSION"] = "8.0.3"
   env["OLLAMA_GPU_DRIVER"] = "rocm"
   proc = subprocess.Popen(["ollama", "serve"], env=env, ...)
   ```

4. **Esperar a que esté listo y verificar backend:**
   ```python
   import time, httpx
   for i in range(10):
       time.sleep(2)
       try:
           httpx.get("http://localhost:11434/api/tags", timeout=3)
           break
       except: pass
   nuevo_backend = _check_ollama_backend()
   ```

5. **Si funciona, persistir env vars al perfil del usuario:**
   ```python
   shell_rc = os.path.expanduser("~/.bashrc")
   with open(shell_rc, "a") as f:
       f.write('\nexport HSA_OVERRIDE_GFX_VERSION=8.0.3\n')
       f.write('export OLLAMA_GPU_DRIVER=rocm\n')
   ```

### Archivos a modificar
- `scripts/install_tui.py` — función `paso5_gpu()` (líneas ~814-865)

---

## Issue 2: HelixDB post-install

### Problema
En `internal/db/helix/client.go`, `EnsureStarted()` solo hace PING al servidor:
```go
func (c *Client) EnsureStarted(ctx context.Context) error {
    if !c.Ping(ctx) {
        return fmt.Errorf("%w: server not reachable", ErrConnection)
    }
    return nil
}
```
Si HelixDB no está corriendo, no lo inicia. El paso de install.go "Verificando HelixDB" es no-fatal, así que la instalación termina ok pero HelixDB no funciona.

### Solución
1. Agregar función `StartHelixDB()` que ejecute `helix up` (CLI) o `docker compose up -d` para levantar el container Docker.
2. `EnsureStarted()` debe:
   - Intentar ping
   - Si falla, ejecutar `StartHelixDB()`
   - Reintentar ping hasta 3 veces
   - Si aún falla, retornar error

```go
func (c *Client) EnsureStarted(ctx context.Context) error {
    if c.Ping(ctx) {
        return nil
    }
    // No responde → intentar iniciar
    if err := startHelixContainer(ctx); err != nil {
        return fmt.Errorf("%w: %v", ErrConnection, err)
    }
    // Reintentar
    for i := 0; i < 5; i++ {
        time.Sleep(2 * time.Second)
        if c.Ping(ctx) {
            return nil
        }
    }
    return fmt.Errorf("%w: server not reachable after start", ErrConnection)
}

func startHelixContainer(ctx context.Context) error {
    cmd := exec.CommandContext(ctx, "docker", "compose", "-f", "/home/secko/.config/zyrocli/docker-compose.yml", "up", "-d")
    return cmd.Run()
}
```

Pero mejor: usar el `helix up` command si helix CLI está disponible, o docker compose directo. La ruta del compose file debe ser configurable.

### Archivos a modificar
- `internal/db/helix/client.go` — modificar `EnsureStarted()`, agregar `startHelixContainer()`
- `cmd/zyrocli/install.go` — hacer el paso HelixDB fatal (no non-fatal) si falla

---

## Issue 3: Subagentes arreglados

### Problema
Los archivos `internal/boomerang/task_manager.go` y `internal/boomerang/delegate.go` intentaban ejecutar `opencode subagent <nombre>` como CLI, pero OpenCode nunca tuvo ese comando. Siempre fallaba mostrando el help.

### Solución (ya aplicada)
1. `executeTask()` ya no ejecuta CLI. Solo marca la tarea como "running".
2. Agregada `CompleteTask()` para que el orquestador marque tareas como completadas.
3. `DelegateStep()` ya no ejecuta CLI. Solo registra tareas.
4. Agregado tool `complete_task` al MCP server `zyro-task-board`.

### Archivos modificados
- `internal/boomerang/task_manager.go`
- `internal/boomerang/delegate.go`
- `cmd/zyrocli/mcp_server.go`

---

## Prioridad
1. **Issue 3**: ✅ YA ARREGLADO
2. **Issue 2**: HelixDB post-install — MEDIA
3. **Issue 1**: GPU installer auto — ALTA (es lo que más afecta al usuario)

## Riesgos
- GPU: `sudo modprobe` requiere permisos elevados. Si el usuario no tiene sudo, falla.
- HelixDB: buscar el docker-compose.yml automáticamente puede fallar si helix no está instalado.
- Subagentes: ya probado, pasa tests.

## Criterios de éxito
- [ ] GPU: usuario elige ROCm → el script carga amdkfd, persiste config, reinicia ollama, verifica backend automáticamente
- [ ] HelixDB: `EnsureStarted()` inicia el container Docker si no está corriendo
- [ ] Subagentes: dispatch_task + complete_task funcionan sin errores
