# Investigación: Drivers Vulkan (Mesa) para aceleración GPU en Ollama

## Fecha
2026-06-18

## Contexto
El instalador de ZyroCLI debe configurar Ollama para usar la GPU en cualquier hardware.
Actualmente soporta CUDA (NVIDIA), ROCm (AMD) y Vulkan (genérico). Esta investigación
explora los drivers Vulkan de Mesa (RADV, NVK, ANV) como alternativa para simplificar
la configuración y dar soporte universal.

## Drivers Vulkan de Mesa

### RADV — AMD (estándar open-source)
- **Driver Vulkan oficial de Mesa para GPUs AMD Radeon**
- Usado por Steam Deck, ampliamente probado
- Paquete Arch: `vulkan-radeon` (o `lib32-vulkan-radeon`)
- No requiere ROCm, funciona con solo el driver amdgpu del kernel
- Soporta desde GCN1 en adelante (con configuraciones especiales para tarjetas viejas)
- **Ideal para usuarios AMD que no quieren instalar ROCm**

### NVK — NVIDIA (Mesa, open-source)
- Driver Vulkan open-source de Mesa para GPUs NVIDIA
- Rápido desarrollo, alternativa a `nvidia-utils` (proprietario)
- Paquete Arch: `vulkan-nouveau` (o `lib32-vulkan-nouveau`)
- **Nota**: Requiere configuración adicional del sistema (ver wiki de Nouveau)
- La mayoría de usuarios NVIDIA siguen usando `nvidia-utils` (CUDA)

### ANV — Intel (oficial)
- Driver Vulkan oficial de Mesa para GPUs Intel
- Soporta Intel HD, Iris, Arc Graphics
- Paquete Arch: `vulkan-intel` (o `lib32-vulkan-intel`)
- Para iGPUs Intel no se espera alto rendimiento en ML

## Ollama y los backends GPU

### Paquetes oficiales (Arch Linux)

| Backend | Paquete Arch | GPU | Notas |
|---------|-------------|-----|-------|
| **Vulkan** | `ollama-vulkan` | NVIDIA, AMD, Intel | Genérico, easy |
| **CUDA** | `ollama-cuda` | NVIDIA | Propietario, máximo rendimiento |
| **ROCm** | `ollama-rocm` | AMD | Alto rendimiento, configurable |

### Lo que usa el instalador actual
Actualmente el instalador (install_tui.py) instala desde AUR:
- `ollama-vulkan-bin` (Vulkan fácil)
- `ollama-rocm-bin` (ROCm más rendimiento)

### Alternativa con paquetes oficiales
Arch Linux tiene paquetes oficiales (`ollama-vulkan`, `ollama-rocm`, `ollama-cuda`)
que son más estables que los AUR bins. Se podrían usar en vez de los AUR.

## Estrategia GPU para todos

### Tabla actualizada

| GPU | Detección | Backend primario | Backend fallback | Paquete |
|-----|-----------|-----------------|------------------|---------|
| **NVIDIA** | `nvidia-smi` | CUDA (`ollama-cuda`) | Vulkan (`ollama-vulkan`) | oficial |
| **AMD** | `rocm-smi` / `hipconfig` | ROCm (`ollama-rocm`) | Vulkan/RADV (`ollama-vulkan`) | oficial |
| **AMD (nueva)** | `vulkaninfo` (RADV) | Vulkan/RADV (`ollama-vulkan`) | — | oficial |
| **Intel Arc** | `vulkaninfo` (ANV) | Vulkan/ANV (`ollama-vulkan`) | — | oficial |
| **Apple** | `sysctl` | Metal (built-in) | — | built-in |
| **Otra** | `vulkaninfo` | Vulkan genérico | — | oficial |

### Flujo de decisión (nuevo)

```
1. Detectar GPU
   ├── ¿NVIDIA? → ¿nvidia-smi funciona?
   │   ├── Sí → ollama-cuda ✅
   │   └── No → Verificar NVK (vulkan-nouveau) → ollama-vulkan
   │
   ├── ¿AMD? → ¿rocm-smi funciona?
   │   ├── Sí → ollama-rocm + auto-config (HSA_OVERRIDE_GFX_VERSION)
   │   └── No → Verificar RADV (vulkan-radeon) → ollama-vulkan ✅
   │
   ├── ¿Intel? → ollama-vulkan (ANV)
   │
   ├── ¿Apple? → Metal (built-in, no requiere instalación)
   │
   └── ¿No detectada? → ollama-vulkan (fallback genérico)
```

### Ventajas de priorizar Vulkan
1. **Universal**: misma solución para NVIDIA, AMD, Intel
2. **Sin dependencias pesadas**: no requiere ROCm (6GB+) ni CUDA toolkit
3. **RADV está preinstalado** en la mayoría de distros con Mesa
4. **Menos config**: no requiere modprobe amdkfd, ni HSA_OVERRIDE_GFX_VERSION
5. **Rendimiento**: RADV tiene rendimiento competitivo con ROCm para inferencia

### Desventajas
1. **ROCm puede dar 10-20% más rendimiento** en GPUs AMD compatibles
2. **NVK aún no es tan maduro** como nvidia-utils para gaming
3. **No todos los modelos de GPU tienen soporte Vulkan óptimo**

## Recomendación para el instalador
1. Detectar GPU
2. Si NVIDIA → intentar CUDA (`ollama-cuda`), fallback a Vulkan
3. Si AMD → intentar ROCm (`ollama-rocm`), fallback a Vulkan/RADV (`ollama-vulkan`)
4. Si Intel u otra → Vulkan (`ollama-vulkan`)
5. Si nada detectado → CPU mode

Esto cubre NVIDIA, AMD, Intel, Apple, y cualquier GPU con soporte Vulkan.

## Fuentes
- https://wiki.archlinux.org/title/Vulkan
- https://wiki.archlinux.org/title/Ollama
- https://docs.mesa3d.org/
