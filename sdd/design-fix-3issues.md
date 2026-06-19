# Design & Tasks — Fix 3 Issues

## Issue 1: GPU installer automático

### Diseño

**Archivo:** `scripts/install_tui.py`

**Función a modificar:** `paso5_gpu()` 

**Flujo nuevo post-instalación ROCm:**

```
[Usuario elige opción 2 ROCm]
  → yay -S ollama-rocm-bin (YA EXISTE)
  → sudo modprobe amdkfd (NUEVO)
  → echo amdkfd | sudo tee /etc/modules-load.d/amdkfd.conf (NUEVO)
  → pkill ollama (YA EXISTE)
  → Preparar env vars: HSA_OVERRIDE_GFX_VERSION=8.0.3 + OLLAMA_GPU_DRIVER=rocm (NUEVO)
  → ollama serve con esas env vars (NUEVO)
  → Esperar hasta que responda (max 20s) (NUEVO)
  → Verificar backend ROCm (YA EXISTE)
  → Si ok: escribir env vars a ~/.bashrc (NUEVO)
  → Si no: mostrar error claro (MEJORADO)
```

**Nuevas funciones a agregar:**

```python
def _auto_configure_rocm():
    """Ejecuta post-instalación ROCm automáticamente."""
    # 1. Modprobe amdkfd
    if not _check_amdkfd_module():
        subprocess.run(["sudo", "modprobe", "amdkfd"])
        subprocess.run(
            ["sudo", "tee", "/etc/modules-load.d/amdkfd.conf"],
            input=b"amdkfd\n",
        )
    
    # 2. Matar ollama
    subprocess.run(["pkill", "ollama"], capture_output=True)
    
    # 3. Setear envs y arrancar
    env = os.environ.copy()
    env["HSA_OVERRIDE_GFX_VERSION"] = "8.0.3"
    env["OLLAMA_GPU_DRIVER"] = "rocm"
    proc = subprocess.Popen(
        ["ollama", "serve"],
        env=env,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    
    # 4. Esperar que responda
    for i in range(10):
        time.sleep(2)
        try:
            httpx.get("http://localhost:11434/api/tags", timeout=3)
            break
        except:
            continue
    else:
        return False, "Ollama no respondió después de 20s"
    
    # 5. Verificar backend
    backend = _check_ollama_backend()
    if backend == "rocm":
        # Persistir env vars
        shell_rc = os.path.expanduser("~/.bashrc")
        with open(shell_rc, "a") as f:
            f.write("\n# ROCm for Ollama (ZyroCLI)\n")
            f.write('export HSA_OVERRIDE_GFX_VERSION=8.0.3\n')
            f.write('export OLLAMA_GPU_DRIVER=rocm\n')
        return True, "ROCm configurado correctamente"
    
    return False, f"Backend es {backend}, no ROCm"

def _auto_configure_vulkan():
    """Configura Vulkan post-instalación."""
    subprocess.run(["pkill", "ollama"], capture_output=True)
    proc = subprocess.Popen(
        ["ollama", "serve"],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    for i in range(10):
        time.sleep(2)
        try:
            httpx.get("http://localhost:11434/api/tags", timeout=3)
            break
        except:
            continue
    backend = _check_ollama_backend()
    return backend == "vulkan", f"Backend: {backend}"
```

**Para Vulkan (opción 1):** Similar pero más simple, solo matar ollama y arrancarlo de nuevo (Vulkan se activa automáticamente si el driver está disponible).

### Tasks
1. **T-1.1**: Agregar `_auto_configure_rocm()` a `scripts/install_tui.py`
2. **T-1.2**: Agregar `_auto_configure_vulkan()` a `scripts/install_tui.py`
3. **T-1.3**: Modificar `paso5_gpu()` para llamar las funciones automáticas en vez de imprimir instrucciones
4. **T-1.4**: Mejorar mensajes post-instalación (que muestre "✅ ROCm configurado" o "❌ Falló: ...")

---

## Issue 2: HelixDB post-install

### Diseño

**Archivo:** `internal/db/helix/client.go`

**Nuevas funciones:**

```go
// startHelixContainer intenta iniciar el container de HelixDB.
// Busca docker-compose.yml en varias ubicaciones conocidas.
func startHelixContainer(ctx context.Context) error {
    // Método 1: helix CLI
    if _, err := exec.LookPath("helix"); err == nil {
        cmd := exec.CommandContext(ctx, "helix", "up")
        if output, err := cmd.CombinedOutput(); err == nil {
            return nil
        } else {
            _ = output // fallback a docker
        }
    }
    
    // Método 2: docker compose con el helix.toml del proyecto
    candidates := []string{
        "/home/secko/.config/zyrocli/docker-compose.yml",
        "/etc/helix/docker-compose.yml",
    }
    for _, path := range candidates {
        if _, err := os.Stat(path); err == nil {
            cmd := exec.CommandContext(ctx, "docker", "compose",
                "-f", path, "up", "-d")
            if err := cmd.Run(); err == nil {
                return nil
            }
        }
    }
    
    return fmt.Errorf("no helix container config found")
}
```

**Modificar `EnsureStarted()`:**

```go
func (c *Client) EnsureStarted(ctx context.Context) error {
    if c.Ping(ctx) {
        return nil
    }
    // Intentar iniciar
    if err := startHelixContainer(ctx); err != nil {
        return fmt.Errorf("%w: %v", ErrConnection, err)
    }
    // Reintentar ping con backoff
    for i := 0; i < 5; i++ {
        time.Sleep(2 * time.Second)
        if c.Ping(ctx) {
            return nil
        }
    }
    return fmt.Errorf("%w: server not reachable after start attempt", ErrConnection)
}
```

**Modificar `cmd/zyrocli/install.go`:** Cambiar el step de HelixDB de no-fatal a fatal:

```go
{
    Name: "Verificando HelixDB",
    Action: func() error {
        client, err := helix.NewClient(cmd.Context())
        if err != nil {
            return err // AHORA ES FATAL
        }
        defer client.Close()
        return client.EnsureStarted(cmd.Context())
    },
},
```

### Tasks
1. **T-2.1**: Agregar `startHelixContainer()` a `internal/db/helix/client.go`
2. **T-2.2**: Modificar `EnsureStarted()` para llamar `startHelixContainer()` si ping falla
3. **T-2.3**: Cambiar el step de HelixDB en `cmd/zyrocli/install.go` de no-fatal a fatal

---

## Issue 3: Subagentes ✅ YA ARREGLADO

### Tasks completadas
- **T-3.1**: `task_manager.go` — `executeTask()` ya no ejecuta CLI inexistente ✅
- **T-3.2**: `task_manager.go` — Nueva función `CompleteTask()` ✅
- **T-3.3**: `delegate.go` — `DelegateStep()` ya no ejecuta CLI ✅
- **T-3.4**: `mcp_server.go` — Nuevo tool `complete_task` ✅

---

## Resumen de tareas

| ID | Descripción | Archivo | Depende de |
|---|---|---|---|
| T-1.1 | `_auto_configure_rocm()` | `scripts/install_tui.py` | — |
| T-1.2 | `_auto_configure_vulkan()` | `scripts/install_tui.py` | — |
| T-1.3 | Modificar `paso5_gpu()` | `scripts/install_tui.py` | T-1.1, T-1.2 |
| T-1.4 | Mejorar mensajes post-instalación | `scripts/install_tui.py` | T-1.3 |
| T-2.1 | `startHelixContainer()` | `internal/db/helix/client.go` | — |
| T-2.2 | Modificar `EnsureStarted()` | `internal/db/helix/client.go` | T-2.1 |
| T-2.3 | HelixDB fatal en install.go | `cmd/zyrocli/install.go` | T-2.2 |
| T-3.1-T-3.4 | Subagentes | varios | ✅ Hecho |

## Orden de implementación sugerido

1. T-3.x (ya hecho) → 2. T-1.1 + T-1.2 (paralelo) → 3. T-1.3 → 4. T-1.4
5. T-2.1 → 6. T-2.2 → 7. T-2.3
