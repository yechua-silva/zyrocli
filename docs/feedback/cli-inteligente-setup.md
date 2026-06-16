# Feedback: CLI Inteligente con Misión de Autoinstalación

> **Fecha:** 2026-06-15
> **Fuente:** Prueba en PC del hermano — entorno limpio, dependencias faltantes

---

## 🔍 El Problema Detectado

Cuando se probó Zyro en el PC del hermano, falló porque **faltaban dependencias** (especialmente `uv`). El CLI asumía que todo estaba instalado y configurado, pero la realidad es que un entorno limpio no lo tiene. El usuario tuvo que instalar cosas a mano, rompiendo la experiencia de "funciona de una vez".

## 🧠 La Solución: CLI Inteligente

Un CLI inteligente no es solo el que ejecuta agentes, sino el que **se levanta a sí mismo** en cualquier máquina sin intervención manual.

**Misión única en la fase de instalación:**
> *"Verificar que todas las dependencias necesarias existen; si no, instalarlas automáticamente o guiar al usuario con un solo comando."*

## ⚙️ Flujo Atómico de `zyro setup`

1. **Verificar sistema operativo** y arquitectura.
2. **Comprobar cada dependencia**:
   - `uv` (gestor Python) → si no está, instalar desde `astral.sh/uv`
   - `go` (1.21+) → si no, avisar y sugerir instalación oficial
   - `docker` (opcional, para sandbox) → si no, avisar
   - `helixdb` → si no está en `PATH`, descargar binario desde GitHub releases
   - `git` (opcional pero recomendado)
3. **Crear entorno virtual Python** con `uv venv` y sincronizar `pyproject.toml`
4. **Compilar el binario Go** (`go build -o ~/.local/bin/zyrocli ./cmd/zyrocli`)
5. **Configurar MCP servers** automáticamente (helix-mcp, etc.)
6. **Generar `~/.zyro/config.yaml`** con rutas y preferencias
7. **Ejecutar `zyro doctor --fix`** para reparar problemas residuales

## 🎯 Resultado Esperado

Después de `zyro setup`, el sistema queda **listo para usar** en cualquier máquina, sin leer READMEs ni instalaciones manuales.

## 📌 Conclusión

> *"El CLI no puede depender de que el usuario tenga X o Y preinstalado."*

La misión del CLI inteligente es **ser el propio instalador de su ecosistema**. Todo lo demás (orquestación, fases) viene después.
