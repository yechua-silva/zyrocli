#!/usr/bin/env python3
"""ZyroAgentCLI — Installation & Configuration TUI.

Flujo de 8 pasos:
  1. Welcome + verificar Python >= 3.10
  2. Verificar Ollama (binario + servidor)
  3. Seleccionar modelo de embeddings
  4. Seleccionar modelo chat (para fact_extractor LLM mode)
  5. Detectar GPU
  6. Probar embeddings
  7. Probar chat model (opcional)
  8. Resumen + escribir ~/.zyro/config.yaml

Uso:
    python scripts/install_tui.py

Dependencias: rich, httpx, yaml (pyyaml) — todo ya instalado vía el proyecto.
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
from datetime import date
from pathlib import Path
from typing import Any

# ── Rich ────────────────────────────────────────────────────────
from rich.console import Console
from rich.markdown import Markdown
from rich.panel import Panel
from rich.progress import (
    BarColumn,
    Progress,
    SpinnerColumn,
    TaskProgressColumn,
    TextColumn,
    TimeElapsedColumn,
)
from rich.prompt import Confirm, Prompt
from rich.table import Table
from rich.text import Text

# ── Logger global ────────────────────────────────────────────────
console = Console()

# ── Constantes ──────────────────────────────────────────────────
OLLAMA_HOST = "http://localhost:11434"
MIN_PYTHON = (3, 10)

EMBEDDING_OPTIONS: list[dict[str, Any]] = [
    {
        "key": "1",
        "model": "mxbai-embed-large:latest",
        "dims": 1024,
        "params": "~0.5B",
        "desc": "Recommended — balance calidad/rendimiento",
    },
    {
        "key": "2",
        "model": "nomic-embed-text:latest",
        "dims": 768,
        "params": "~0.1B",
        "desc": "Más rápido en CPU, bueno para equipos modestos",
    },
    {
        "key": "3",
        "model": "all-minilm:latest",
        "dims": 384,
        "params": "~0.02B",
        "desc": "Ultraligero, ideal para CPU o Raspberry Pi",
    },
]

CHAT_OPTIONS: list[dict[str, Any]] = [
    {
        "key": "1",
        "model": "llama3.2:3b",
        "size": "~2.0 GB",
        "params": "3B",
        "quality": "Alta",
        "cpu_ok": True,
        "desc": "Recommended — Meta Llama 3.2, balance velocidad/calidad",
    },
    {
        "key": "2",
        "model": "phi4-mini:3.8b",
        "size": "~2.5 GB",
        "params": "3.8B",
        "quality": "Muy alta",
        "cpu_ok": True,
        "desc": "Microsoft Phi-4 Mini, excelente para JSON y structured output",
    },
    {
        "key": "3",
        "model": "qwen3.5:0.5b",
        "size": "~0.4 GB",
        "params": "0.5B",
        "quality": "Básica",
        "cpu_ok": True,
        "desc": "Ultraligero, funciona en cualquier parte",
    },
    {
        "key": "4",
        "model": "gemma3:2b",
        "size": "~1.5 GB",
        "params": "2B",
        "quality": "Alta",
        "cpu_ok": True,
        "desc": "Google Gemma 3, 2B params, buena relación calidad/tamaño",
    },
    {
        "key": "5",
        "model": "mistral:7b",
        "size": "~4.1 GB",
        "params": "7B",
        "quality": "Muy alta",
        "cpu_ok": False,
        "desc": "Más preciso pero pesado para CPU — requiere GPU o mucha RAM",
    },
]


# ═══════════════════════════════════════════════════════════════════
#  Helpers
# ═══════════════════════════════════════════════════════════════════


def _ejecutar(cmd: list[str], timeout: int = 30) -> tuple[int, str, str]:
    """Ejecuta un comando y retorna (returncode, stdout, stderr)."""
    try:
        r = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
        return r.returncode, r.stdout.strip(), r.stderr.strip()
    except FileNotFoundError:
        return -1, "", f"comando no encontrado: {cmd[0]}"
    except subprocess.TimeoutExpired:
        return -2, "", f"timeout ({timeout}s) ejecutando: {' '.join(cmd)}"


def _ollama_get(path: str, timeout: int = 10) -> dict[str, Any] | None:
    """GET a la API de Ollama. Retorna dict o None si falla."""
    import httpx

    try:
        resp = httpx.get(f"{OLLAMA_HOST}{path}", timeout=timeout)
        if resp.status_code == 200:
            return resp.json()
    except Exception:
        return None
    return None


def _ollama_post(
    path: str, payload: dict[str, Any], timeout: int = 30
) -> dict[str, Any] | None:
    """POST a la API de Ollama. Retorna dict o None si falla."""
    import httpx

    try:
        resp = httpx.post(
            f"{OLLAMA_HOST}{path}", json=payload, timeout=timeout
        )
        if resp.status_code == 200:
            return resp.json()
    except Exception:
        return None
    return None


def _modelo_sin_tag(modelo: str) -> str:
    """Quita el tag ':latest' u otro tag para el nombre limpio."""
    return modelo.split(":")[0]


def _modelo_instalado(modelo: str) -> bool:
    """Verifica si un modelo ya está descargado en Ollama."""
    data = _ollama_get("/api/tags")
    if not data:
        return False
    name_clean = _modelo_sin_tag(modelo)
    for m in data.get("models", []):
        mname = m.get("name", "")
        if mname == modelo or mname == name_clean or mname.startswith(name_clean + ":"):
            return True
    return False


def _parse_ollama_version(version_str: str) -> tuple[int, int, int] | None:
    """Parsea 'ollama version is 0.24.0' → (0, 24, 0). Retorna None si no puede."""
    import re
    m = re.search(r'(\d+)\.(\d+)\.(\d+)', version_str)
    if m:
        return (int(m.group(1)), int(m.group(2)), int(m.group(3)))
    return None


def _check_amdkfd_module() -> bool:
    """Verifica si el módulo amdkfd está cargado (necesario para ROCm)."""
    rc, out, _ = _ejecutar(["lsmod"])
    if rc == 0:
        for line in out.split("\n"):
            if line.startswith("amdkfd"):
                return True
    return False


def _check_ollama_backend() -> str:
    """Detecta qué backend usa Ollama: 'cpu', 'rocm', 'vulkan', 'cuda', o 'unknown'.
    
    Lee los logs de ollama o el proceso para determinar el backend activo.
    """
    # Método 1: buscar librerías ggml cargadas
    # Buscar en /proc/$(pgrep ollama)/maps o similar
    import subprocess
    try:
        pid = subprocess.check_output(["pgrep", "-x", "ollama"], text=True).strip()
        maps = subprocess.check_output(
            ["grep", "ggml", f"/proc/{pid}/maps"], 
            text=True, 
            stderr=subprocess.DEVNULL
        )
        if "rocm" in maps.lower():
            return "rocm"
        elif "vulkan" in maps.lower():
            return "vulkan"  
        elif "cuda" in maps.lower():
            return "cuda"
        elif "cpu" in maps.lower():
            return "cpu"
    except (subprocess.CalledProcessError, FileNotFoundError):
        pass
    
    # Método 2: buscar en journalctl
    try:
        logs = subprocess.check_output(
            ["journalctl", "-u", "ollama", "--no-pager", "-n", "50"],
            text=True, stderr=subprocess.DEVNULL
        )
        logs_lower = logs.lower()
        if "loaded rocm backend" in logs_lower or "ggml_rocm" in logs_lower:
            return "rocm"
        elif "loaded vulkan backend" in logs_lower or "ggml_vulkan" in logs_lower:
            return "vulkan"
        elif "loaded cuda backend" in logs_lower or "ggml_cuda" in logs_lower:
            return "cuda"
        elif "loaded cpu backend" in logs_lower or "ggml_cpu" in logs_lower:
            return "cpu"
    except (subprocess.CalledProcessError, FileNotFoundError):
        pass
    
    # Método 3: verificar OLLAMA_GPU_DRIVER
    driver = os.environ.get("OLLAMA_GPU_DRIVER", "").lower()
    if driver in ("rocm", "vulkan", "cuda"):
        return driver
    
    # Método 4: buscar binarios de ollama y verificar con qué tags se compilaron
    ollama_bin = shutil.which("ollama")
    if ollama_bin:
        try:
            out = subprocess.check_output(
                [ollama_bin, "--version"], text=True, stderr=subprocess.STDOUT
            )
            # Si se compiló con un tag, podría mencionarlo
            if "vulkan" in out.lower():
                return "vulkan"
        except:
            pass
    
    return "unknown"


def _get_aur_helper() -> str | None:
    """Retorna el nombre del helper AUR disponible: 'yay', 'paru', o None."""
    for helper in ["yay", "paru"]:
        if shutil.which(helper):
            return helper
    return None


# ═══════════════════════════════════════════════════════════════════
#  Paso 1 — Welcome + verificar Python
# ═══════════════════════════════════════════════════════════════════


def paso1_bienvenida() -> bool:
    """Muestra banner y verifica Python >= 3.10. Retorna False si debe abortar."""
    console.clear()
    banner = Panel.fit(
        Text(
            "╔══════════════════════════════════════════╗\n"
            "║     ZyroAgentCLI — Instalación y        ║\n"
            "║          Configuración                   ║\n"
            "╚══════════════════════════════════════════╝",
            style="bold cyan",
            justify="center",
        ),
        border_style="cyan",
        padding=(1, 2),
    )
    console.print(banner)
    console.print()

    # Python version
    v = sys.version_info
    console.print(f"  Python detectado: [bold]{v.major}.{v.minor}.{v.micro}[/bold]")
    console.print(f"  Path: [dim]{sys.executable}[/dim]")
    console.print()

    if (v.major, v.minor) < MIN_PYTHON:
        console.print(
            Panel(
                f"[red]❌ Se requiere Python ≥ {MIN_PYTHON[0]}.{MIN_PYTHON[1]}[/red]\n"
                f"  Versión actual: {v.major}.{v.minor}.{v.micro}\n\n"
                "  [yellow]Actualiza Python e intenta de nuevo:[/yellow]\n"
                "    https://www.python.org/downloads/",
                title="Error",
                border_style="red",
            )
        )
        return False

    console.print(
        Panel(
            "[green]✅ Python 3.10+ detectado correctamente[/green]",
            border_style="green",
        )
    )
    console.print()
    return True


# ═══════════════════════════════════════════════════════════════════
#  Paso 2 — Verificar Ollama
# ═══════════════════════════════════════════════════════════════════


def paso2_ollama() -> bool:
    """Verifica que Ollama esté instalado y el servidor responda.
    Retorna True si todo funciona.
    """
    console.rule("[bold cyan]Paso 2: Verificar Ollama[/bold cyan]")
    console.print()

    # 2a. ¿Binario instalado?
    ollama_bin = shutil.which("ollama")
    if not ollama_bin:
        console.print(
            Panel(
                "[red]❌ Ollama no está instalado.[/red]\n\n"
                "  [yellow]Instálalo desde:[/yellow]\n"
                "    https://ollama.com/download\n\n"
                "  [yellow]O con el script oficial:[/yellow]\n"
                "    curl -fsSL https://ollama.com/install.sh | sh\n\n"
                "  [dim]Una vez instalado, ejecuta este instalador nuevamente.[/dim]",
                title="Ollama no encontrado",
                border_style="red",
            )
        )
        return False

    # Versión
    rc, out, err = _ejecutar([ollama_bin, "--version"])
    if rc == 0:
        console.print(f"  [green]✅[/green] Ollama binario: [bold]{out}[/bold]")
    else:
        console.print(f"  [yellow]⚠️[/yellow]  Ollama binario encontrado pero no se pudo obtener versión: {err}")

    # Detectar versión antigua
    version_info = _parse_ollama_version(out)
    if version_info:
        major, minor, patch = version_info
        es_antigua = major == 0 and minor < 50  # 0.24.0 es MUY antigua
        if es_antigua:
            console.print(
                Panel(
                    f"[yellow]⚠️  Versión {major}.{minor}.{patch} es muy antigua.[/yellow]\n\n"
                    "  Los backends GPU (ROCm, Vulkan) requieren Ollama ≥ 0.5.x.\n"
                    "  Tu versión actual solo soporta CPU.\n\n"
                    "  [bold]Opciones para actualizar:[/bold]\n"
                    "    1. [green]ollama-vulkan-bin[/green] — Usa Vulkan (más fácil, recomendado)\n"
                    "    2. [green]ollama-rocm-bin[/green]  — Usa ROCm (más rendimiento pero más config)\n"
                    "    3. Script oficial: [dim]curl -fsSL https://ollama.com/install.sh | sh[/dim]\n\n"
                    "  [yellow]💡 Puedes actualizar en el Paso 5 (GPU) de este instalador.[/yellow]",
                    title="Versión antigua de Ollama",
                    border_style="yellow",
                )
            )

    console.print()

    # 2b. ¿Servidor responde?
    tags = _ollama_get("/api/tags")
    if tags is not None:
        modelos = tags.get("models", [])
        console.print(f"  [green]✅[/green] Servidor Ollama responde en [bold]{OLLAMA_HOST}[/bold]")
        if modelos:
            console.print(f"  Modelos instalados: [bold]{len(modelos)}[/bold]")
            for m in modelos:
                name = m.get("name", "?")
                size = m.get("size", 0)
                size_str = (
                    f"{size / 1e9:.1f} GB" if size > 1e9 else f"{size / 1e6:.1f} MB"
                )
                console.print(f"    • {name}  [dim]({size_str})[/dim]")
        else:
            console.print("  [dim]No hay modelos instalados todavía.[/dim]")
        console.print()
        return True
    else:
        console.print(
            Panel(
                "[red]❌ No se pudo conectar al servidor Ollama.[/red]\n\n"
                "  El binario está instalado pero el servidor no responde en\n"
                f"  [bold]{OLLAMA_HOST}[/bold].\n\n"
                "  [yellow]¿Quieres iniciarlo ahora?[/yellow]",
                title="Servidor no responde",
                border_style="yellow",
            )
        )
        iniciar = Confirm.ask("  ¿Iniciar 'ollama serve'?", default=True)
        if iniciar:
            with console.status("[yellow]Iniciando servidor Ollama...[/yellow]"):
                try:
                    subprocess.Popen(
                        [ollama_bin, "serve"],
                        stdout=subprocess.DEVNULL,
                        stderr=subprocess.DEVNULL,
                        start_new_session=True,
                    )
                except Exception as e:
                    console.print(f"[red]Error al iniciar: {e}[/red]")
                    return False

            # Reintentar conexión
            import time

            for attempt in range(5):
                time.sleep(1)
                tags = _ollama_get("/api/tags", timeout=3)
                if tags is not None:
                    console.print("[green]✅ Servidor iniciado correctamente[/green]")
                    return True
                console.print(f"  Esperando... ({attempt + 1}/5)")

            console.print("[red]❌ No se pudo conectar después de iniciar el servidor.[/red]")
            console.print("  [yellow]Prueba manualmente:[/yellow] ollama serve")
            return False
        else:
            console.print("[yellow]⚠️  Continuando sin servidor Ollama (algunas funciones no estarán disponibles)[/yellow]")
            console.print()
            return False


# ═══════════════════════════════════════════════════════════════════
#  Paso 3 — Seleccionar modelo de embeddings
# ═══════════════════════════════════════════════════════════════════


def paso3_embeddings() -> str | None:
    """Elige modelo de embeddings. Retorna el nombre del modelo o None si cancela."""
    console.rule("[bold cyan]Paso 3: Modelo de Embeddings[/bold cyan]")
    console.print()
    console.print(
        "Selecciona el modelo que se usará para generar embeddings vectoriales.\n"
        "Estos vectores se usan para búsqueda semántica, memoria y clasificación.\n"
    )

    # Tabla de opciones
    table = Table(title="Modelos de Embeddings disponibles")
    table.add_column("Opción", style="cyan", width=8)
    table.add_column("Modelo", style="bold")
    table.add_column("Dimensiones", justify="right")
    table.add_column("Parámetros", justify="right")
    table.add_column("Descripción", style="dim")

    for opt in EMBEDDING_OPTIONS:
        table.add_row(
            opt["key"],
            opt["model"],
            str(opt["dims"]),
            opt["params"],
            opt["desc"],
        )
    console.print(table)
    console.print()

    # Prompt interactivo
    choices = [o["key"] for o in EMBEDDING_OPTIONS]
    eleccion = Prompt.ask(
        "  [bold]Elige una opción[/bold]",
        choices=choices,
        default="1",
    )

    selected = next(o for o in EMBEDDING_OPTIONS if o["key"] == eleccion)
    modelo = selected["model"]
    dims = selected["dims"]

    console.print(f"\n  Has elegido: [bold cyan]{modelo}[/bold cyan] ({dims} dimensiones)")

    if not _modelo_instalado(modelo):
        console.print(f"\n  [yellow]⚠️  El modelo '{modelo}' no está instalado.[/yellow]")
        instalar = Confirm.ask(f"  ¿Descargarlo ahora? (ollama pull {modelo})", default=True)
        if instalar:
            ok = _instalar_modelo(modelo, "embeddings")
            if not ok:
                console.print(f"[red]❌ No se pudo instalar {modelo}[/red]")
                return None
        else:
            console.print("  [yellow]⚠️  Continuando sin descargar (el modelo no estará disponible)[/yellow]")

    console.print()
    return modelo


def _instalar_modelo(modelo: str, tipo: str) -> bool:
    """Ejecuta 'ollama pull <modelo>' con barra de progreso."""
    import httpx

    console.print(f"\n  [bold]Descargando {modelo}...[/bold]")

    try:
        # Iniciamos el pull vía API de Ollama
        # Ollama streaming API: POST /api/pull
        with Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            BarColumn(),
            TaskProgressColumn(),
            TimeElapsedColumn(),
            console=console,
            transient=False,
        ) as progress:
            task = progress.add_task(
                f"[cyan]Descargando {_modelo_sin_tag(modelo)}...[/cyan]",
                total=None,
            )

            resp = httpx.post(
                f"{OLLAMA_HOST}/api/pull",
                json={"name": modelo, "stream": True},
                timeout=600,
            )
            if resp.status_code != 200:
                progress.remove_task(task)
                console.print(f"[red]Error HTTP {resp.status_code} al descargar[/red]")
                # Fallback a subprocess
                return _instalar_modelo_subprocess(modelo)

            # Parsear streaming
            last_status = ""
            for line in resp.iter_lines():
                if not line:
                    continue
                try:
                    data = json.loads(line)
                    status = data.get("status", "")
                    if status != last_status:
                        progress.update(task, description=f"[cyan]{status}[/cyan]")
                        last_status = status
                    if data.get("completed", False):
                        progress.update(
                            task,
                            description="[green]✓ Descarga completada[/green]",
                            completed=100,
                        )
                        console.print(f"  [green]✅ Modelo {modelo} instalado[/green]")
                        return True
                    if data.get("error"):
                        console.print(f"[red]Error: {data['error']}[/red]")
                        return False
                except json.JSONDecodeError:
                    continue

            progress.update(task, description="[green]✓ Instalado[/green]")
            return True

    except Exception as e:
        console.print(f"[yellow]Streaming API falló ({e}), usando subprocess...[/yellow]")
        return _instalar_modelo_subprocess(modelo)


def _instalar_modelo_subprocess(modelo: str) -> bool:
    """Fallback: instalar modelo via subprocessollama pull."""
    with console.status(f"[yellow]Descargando {modelo} (esto puede tomar varios minutos)...[/yellow]"):
        rc, out, err = _ejecutar(["ollama", "pull", modelo], timeout=600)
    if rc == 0:
        console.print(f"  [green]✅ Modelo {modelo} instalado[/green]")
        return True
    else:
        console.print(f"  [red]❌ Error descargando {modelo}: {err}[/red]")
        return False


# ═══════════════════════════════════════════════════════════════════
#  Paso 4 — Seleccionar modelo chat
# ═══════════════════════════════════════════════════════════════════


def paso4_chat() -> str | None:
    """Elige modelo chat para fact_extractor. Retorna nombre o None."""
    console.rule("[bold cyan]Paso 4: Modelo Chat (LLM)[/bold cyan]")
    console.print()
    console.print(
        "Selecciona el modelo que se usará para el modo LLM del fact_extractor.\n"
        "Este modelo procesa texto y extrae hechos estructurados en JSON.\n"
    )

    # Tabla de opciones
    table = Table(title="Modelos Chat disponibles")
    table.add_column("Opción", style="cyan", width=8)
    table.add_column("Modelo", style="bold")
    table.add_column("Tamaño", justify="right")
    table.add_column("Params", justify="right")
    table.add_column("Calidad")
    table.add_column("CPU", justify="center")
    table.add_column("Descripción", style="dim")

    for opt in CHAT_OPTIONS:
        cpu_icon = "[green]✅[/green]" if opt["cpu_ok"] else "[red]❌[/red]"
        table.add_row(
            opt["key"],
            opt["model"],
            opt["size"],
            opt["params"],
            opt["quality"],
            cpu_icon,
            opt["desc"],
        )
    console.print(table)
    console.print()

    choices = [o["key"] for o in CHAT_OPTIONS]
    eleccion = Prompt.ask(
        "  [bold]Elige una opción[/bold]",
        choices=choices,
        default="1",
    )

    selected = next(o for o in CHAT_OPTIONS if o["key"] == eleccion)
    modelo = selected["model"]
    clean_name = _modelo_sin_tag(modelo)

    console.print(f"\n  Has elegido: [bold cyan]{modelo}[/bold cyan]")

    if not _modelo_instalado(modelo):
        console.print(f"\n  [yellow]⚠️  El modelo '{modelo}' no está instalado.[/yellow]")
        instalar = Confirm.ask(f"  ¿Descargarlo ahora? (ollama pull {modelo})", default=True)
        if instalar:
            ok = _instalar_modelo(modelo, "chat")
            if not ok:
                console.print(f"[red]❌ No se pudo instalar {modelo}[/red]")
                return None
        else:
            console.print("  [yellow]⚠️  Continuando sin descargar[/yellow]")

    console.print()
    return modelo


# ── Auto-configuración GPU post-instalación ───────────────────────────────


def _auto_configure_rocm() -> tuple[bool, str]:
    """Ejecuta post-instalación ROCm automáticamente.
    
    Retorna (éxito, mensaje).
    """
    import time
    import httpx
    
    # 1. Cargar módulo amdkfd si no está
    if not _check_amdkfd_module():
        console.print("  [yellow]Cargando módulo amdkfd...[/yellow]")
        subprocess.run(["sudo", "modprobe", "amdkfd"], capture_output=True)
        subprocess.run(
            ["sudo", "tee", "/etc/modules-load.d/amdkfd.conf"],
            input=b"amdkfd\n", capture_output=True,
        )
        time.sleep(1)
        if _check_amdkfd_module():
            console.print("  [green]✅ amdkfd cargado y persistido[/green]")
        else:
            console.print("  [yellow]⚠️  No se pudo cargar amdkfd (puede requerir reinicio)[/yellow]")
    
    # 2. Matar ollama si está corriendo
    subprocess.run(["pkill", "ollama"], capture_output=True)
    time.sleep(1)
    
    # 3. Preparar entorno y arrancar ollama con ROCm
    console.print("  [yellow]Iniciando ollama con backend ROCm...[/yellow]")
    env = os.environ.copy()
    env["HSA_OVERRIDE_GFX_VERSION"] = "8.0.3"
    env["OLLAMA_GPU_DRIVER"] = "rocm"
    
    proc = subprocess.Popen(
        ["ollama", "serve"],
        env=env,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    
    # 4. Esperar a que responda (max 20s)
    for i in range(10):
        time.sleep(2)
        try:
            httpx.get("http://localhost:11434/api/tags", timeout=3)
            break
        except Exception:
            continue
    else:
        return False, "Ollama no respondió después de 20s"
    
    # 5. Verificar backend
    backend = _check_ollama_backend()
    if backend == "rocm":
        # Persistir variables de entorno al perfil del usuario
        shell_rc = os.path.expanduser("~/.bashrc")
        try:
            with open(shell_rc, "a") as f:
                f.write("\n# ROCm for Ollama (ZyroCLI)\n")
                f.write('export HSA_OVERRIDE_GFX_VERSION=8.0.3\n')
                f.write('export OLLAMA_GPU_DRIVER=rocm\n')
            console.print(f"  [green]✅ Variables guardadas en {shell_rc}[/green]")
        except Exception as e:
            console.print(f"  [yellow]⚠️  No se pudo persistir env vars: {e}[/yellow]")
        
        return True, "ROCm configurado y funcionando"
    
    return False, f"Backend detectado: {backend}, se esperaba ROCm. Prueba reiniciar la terminal y ejecutar 'ollama serve'"


def _auto_configure_vulkan() -> tuple[bool, str]:
    """Configura Vulkan post-instalación (más simple, sin módulos kernel).
    
    Retorna (éxito, mensaje).
    """
    import time
    import httpx
    
    # 1. Matar ollama
    subprocess.run(["pkill", "ollama"], capture_output=True)
    time.sleep(1)
    
    # 2. Iniciar ollama (Vulkan se activa automáticamente)
    console.print("  [yellow]Iniciando ollama con backend Vulkan...[/yellow]")
    proc = subprocess.Popen(
        ["ollama", "serve"],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    
    # 3. Esperar a que responda
    for i in range(10):
        time.sleep(2)
        try:
            httpx.get("http://localhost:11434/api/tags", timeout=3)
            break
        except Exception:
            continue
    else:
        return False, "Ollama no respondió después de 20s"
    
    # 4. Verificar backend
    backend = _check_ollama_backend()
    if backend in ("vulkan", "rocm", "cuda"):
        return True, f"GPU activa: {backend}"
    return False, f"Backend: {backend} (se esperaba GPU)"


# ═══════════════════════════════════════════════════════════════════
#  Paso 5 — Detectar GPU
# ═══════════════════════════════════════════════════════════════════


def paso5_gpu() -> dict[str, Any]:
    """Detecta GPU disponible. Retorna dict con info."""
    console.rule("[bold cyan]Paso 5: Detectar GPU[/bold cyan]")
    console.print()

    info: dict[str, Any] = {
        "detected": False,
        "type": "none",
        "name": None,
    }

    # 1. NVIDIA
    nvidia_bin = shutil.which("nvidia-smi")
    if nvidia_bin:
        rc, out, err = _ejecutar(
            [nvidia_bin, "--query-gpu=name,driver_version", "--format=csv,noheader"]
        )
        if rc == 0 and out:
            parts = out.split(",")
            gpu_name = parts[0].strip() if len(parts) > 0 else "NVIDIA GPU"
            driver = parts[1].strip() if len(parts) > 1 else "?"
            info["detected"] = True
            info["type"] = "nvidia"
            info["name"] = gpu_name
            info["driver"] = driver
            console.print(
                f"  [green]✅ GPU NVIDIA detectada:[/green] [bold]{gpu_name}[/bold]"
            )
            console.print(f"     Driver: {driver}")

    # 2. AMD ROCm
    if not info["detected"]:
        for bin_name in ["rocm-smi", "hipconfig"]:
            amd_bin = shutil.which(bin_name)
            if amd_bin:
                rc, out, err = _ejecutar([amd_bin])
                if rc == 0:
                    info["detected"] = True
                    info["type"] = "amd"
                    # Extraer nombre de GPU del output
                    gpu_name = _extraer_nombre_amd(out, err)
                    info["name"] = gpu_name
                    console.print(
                        f"  [green]✅ GPU AMD detectada:[/green] [bold]{gpu_name}[/bold]"
                    )
                    console.print(
                        "  [yellow]💡 Recomendación:[/yellow] "
                        "configura [bold]OLLAMA_GPU_DRIVER=rocm[/bold] "
                        "en tu shell o .bashrc/.zshrc"
                    )
                    break

    # 3. Ollama logs: buscar menciones de GPU
    if not info["detected"]:
        gpu_ollama = _detectar_gpu_via_ollama()
        if gpu_ollama:
            info["detected"] = True
            info["type"] = gpu_ollama.get("type", "unknown")
            info["name"] = gpu_ollama.get("name", "GPU (Ollama)")
            console.print(
                f"  [green]✅ GPU detectada (vía Ollama):[/green] [bold]{info['name']}[/bold]"
            )

    # 4. vulkaninfo
    if not info["detected"]:
        vk_bin = shutil.which("vulkaninfo")
        if vk_bin:
            rc, out, err = _ejecutar([vk_bin, "--summary"], timeout=15)
            if rc == 0 and ("GPU" in out or "gpu" in out):
                # Extraer nombre de GPU del summary
                for line in out.split("\n"):
                    if "GPU" in line and (":" in line or "=" in line):
                        gpu_name = line.split(":")[-1].split("=")[-1].strip()
                        if gpu_name and gpu_name not in ("", "0", "1"):
                            info["detected"] = True
                            info["type"] = "vulkan"
                            info["name"] = gpu_name
                            console.print(
                                f"  [green]✅ GPU detectada (vía Vulkan):[/green] "
                                f"[bold]{gpu_name}[/bold]"
                            )
                            break

    # Si no se detectó nada
    if not info["detected"]:
        console.print(
            "  [yellow]⚠️  No se detectó GPU dedicada.[/yellow]\n"
            "  El sistema usará la CPU para todos los cálculos.\n"
            "  Los modelos pequeños (≤3B params) funcionan bien en CPU."
        )
        info["name"] = "CPU (no GPU detectada)"

    console.print()

    # ── Diagnóstico avanzado: ¿Ollama está usando la GPU? ─────
    console.print()
    console.rule("[bold cyan]Diagnóstico GPU en Ollama[/bold cyan]")
    console.print()
    
    ollama_backend = _check_ollama_backend()
    backend_names = {
        "rocm": "[green]ROCm[/green] (AMD GPU)",
        "vulkan": "[green]Vulkan[/green] (GPU multiplataforma)",
        "cuda": "[green]CUDA[/green] (NVIDIA GPU)",
        "cpu": "[yellow]CPU[/yellow] (sin aceleración GPU)",
        "unknown": "[red]No detectado[/red]",
    }
    console.print(f"  Backend activo de Ollama: {backend_names.get(ollama_backend, ollama_backend)}")
    
    # Si GPU detectada pero Ollama corre en CPU → ofrecer instalación
    gpu_detectada = info.get("detected", False)
    gpu_type = info.get("type", "none")
    
    if gpu_detectada and ollama_backend in ("cpu", "unknown"):
        console.print(
            Panel(
                "[yellow]⚡ Tu GPU está detectada pero Ollama no la está usando.[/yellow]\n\n"
                "  Para acelerar los modelos con GPU:\n",
                title="GPU infrautilizada",
                border_style="yellow",
            )
        )
        
        # Para AMD
        if gpu_type in ("amd", "vulkan"):
            # Check amdkfd
            amdkfd_ok = _check_amdkfd_module()
            if not amdkfd_ok and gpu_type == "amd":
                console.print("  [red]❌ Módulo 'amdkfd' no cargado[/red] (necesario para ROCm)")
            else:
                console.print("  [green]✅ Módulo 'amdkfd' cargado[/green]")
            
            console.print("\n  [bold]Opciones disponibles:[/bold]\n")
            console.print("    [bold]1.[/bold] [green]ollama-vulkan-bin[/green] — Usa Vulkan (fácil, recomendado)")
            console.print("       Aprovecha tu Vulkan 1.4 sin instalar ROCm")
            console.print("    [bold]2.[/bold] [green]ollama-rocm-bin[/green]  — Usa ROCm (más rendimiento)")
            console.print("       Requiere: amdkfd + HIP + HSA_OVERRIDE_GFX_VERSION=8.0.3")
            console.print("    [bold]3.[/bold] No instalar, seguir usando CPU\n")
            
            instalar = Prompt.ask(
                "  ¿Qué deseas hacer?",
                choices=["1", "2", "3"],
                default="1",
            )
            
            if instalar in ("1", "2"):
                pkg = "ollama-vulkan-bin" if instalar == "1" else "ollama-rocm-bin"
                aur = _get_aur_helper()
                
                if aur:
                    console.print(f"\n  [yellow]Instalando {pkg} con {aur}...[/yellow]")
                    # Primero matar ollama si está corriendo
                    subprocess.run(["pkill", "ollama"], capture_output=True)
                    
                    rc, out, err = _ejecutar(
                        ["sudo", aur, "-S", "--noconfirm", pkg],
                        timeout=300
                    )
                    if rc == 0:
                        console.print(f"  [green]✅ {pkg} instalado correctamente[/green]")
                        info["backend_instalado"] = pkg
                        info["recomendacion"] = "Reinicia ollama serve para usar la GPU"
                        
                        if instalar == "2":  # ROCm
                            console.print()
                            console.print("  [bold]Configuración automática ROCm...[/bold]")
                            ok, msg = _auto_configure_rocm()
                            if ok:
                                console.print(f"  [green]✅ {msg}[/green]")
                                info["backend_verificado"] = "rocm"
                            else:
                                console.print(f"  [yellow]⚠️  {msg}[/yellow]")
                                console.print("  [dim]Puedes configurarlo manualmente siguiendo la documentación de ROCm.[/dim]")
                        else:  # Vulkan
                            console.print()
                            console.print("  [bold]Configuración automática Vulkan...[/bold]")
                            ok, msg = _auto_configure_vulkan()
                            if ok:
                                console.print(f"  [green]✅ {msg}[/green]")
                                info["backend_verificado"] = "vulkan"
                            else:
                                console.print(f"  [yellow]⚠️  {msg}[/yellow]")
                    else:
                        console.print(f"  [red]❌ Error instalando {pkg}: {err}[/red]")
                        console.print("  [yellow]Instalación manual:[/yellow]")
                        console.print(f"    {aur} -S {pkg}")
                else:
                    console.print("\n  [yellow]No se encontró helper AUR (yay/paru).[/yellow]")
                    console.print("  [bold]Instalación manual:[/bold]")
                    if shutil.which("pacman"):
                        console.print(f"    1. sudo pacman -S --needed git base-devel")
                        console.print(f"    2. git clone https://aur.archlinux.org/{pkg}.git")
                        console.print(f"    3. cd {pkg} && makepkg -si")
                    console.print(f"\n     O descarga desde: https://aur.archlinux.org/packages/{pkg}")
            
            # La verificación ya se hizo dentro de _auto_configure_rocm/vulkan
            # No es necesario repetirla aquí
        
        elif gpu_type == "nvidia":
            console.print("  [green]✅ GPU NVIDIA detectada — CUDA debería funcionar[/green]")
            console.print("  [yellow]💡 Verifica que ollama tenga el backend CUDA:[/yellow]")
            console.print("    ollama --version")
    else:
        if gpu_detectada:
            console.print(f"  [green]✅ GPU en uso: {ollama_backend}[/green]")
        else:
            console.print("  [yellow]⚠️  Sin GPU dedicada — modo CPU solamente[/yellow]")

    console.print()
    return info


def _extraer_nombre_amd(rocm_out: str, rocm_err: str) -> str:
    """Extrae nombre de GPU AMD del output de rocm-smi."""
    for line in rocm_out.split("\n"):
        if "GPU" in line and ":" in line:
            parts = line.split(":")
            if len(parts) >= 2:
                name = parts[-1].strip()
                if name and not name.startswith("GPU"):
                    return name
    for line in rocm_err.split("\n"):
        if "Device" in line or "gpu" in line.lower():
            return line.strip()
    return "AMD GPU detectada"


def _detectar_gpu_via_ollama() -> dict[str, Any] | None:
    """Intenta detectar GPU mirando logs/comandos de Ollama."""
    result: dict[str, Any] = {}

    # Ejecutar ollama list (puede mostrar info de GPU en stderr)
    rc, out, err = _ejecutar(["ollama", "list"], timeout=10)

    # Revisar stderr por menciones GPU
    for line in err.split("\n"):
        ll = line.lower()
        if "gpu" in ll or "cuda" in ll or "rocm" in ll or "vulkan" in ll:
            result["detected"] = True
            if "rocm" in ll:
                result["type"] = "amd"
            elif "cuda" in ll or "nvidia" in ll:
                result["type"] = "nvidia"
            else:
                result["type"] = "unknown"
            # Tratar de extraer nombre
            if ":" in line:
                result["name"] = line.split(":", 1)[-1].strip()
            else:
                result["name"] = line.strip()
            return result

    # Verificar variable de entorno
    driver = os.environ.get("OLLAMA_GPU_DRIVER", "").lower()
    if driver in ("rocm", "cuda", "vulkan"):
        result["detected"] = True
        result["type"] = "amd" if driver == "rocm" else driver
        result["name"] = f"GPU via OLLAMA_GPU_DRIVER={driver}"
        return result

    return None


# ═══════════════════════════════════════════════════════════════════
#  Paso 6 — Probar embeddings
# ═══════════════════════════════════════════════════════════════════


def paso6_probar_embeddings(modelo: str | None) -> bool:
    """Genera un embedding de prueba y verifica dimensiones."""
    console.rule("[bold cyan]Paso 6: Probar Embeddings[/bold cyan]")
    console.print()

    if not modelo:
        console.print("  [yellow]⚠️  No hay modelo de embeddings seleccionado. Omitiendo prueba.[/yellow]")
        console.print()
        return False

    with console.status(f"[yellow]Generando embedding de prueba con {modelo}...[/yellow]"):
        import httpx

        try:
            resp = httpx.post(
                f"{OLLAMA_HOST}/api/embeddings",
                json={
                    "model": modelo,
                    "prompt": "ZyroAgentCLI test de embedding — verificación de instalación.",
                },
                timeout=30,
            )
        except httpx.ConnectError:
            console.print(
                f"  [red]❌ No se pudo conectar a Ollama en {OLLAMA_HOST}[/red]"
            )
            console.print("  [yellow]Asegúrate de que 'ollama serve' esté corriendo.[/yellow]")
            console.print()
            return False
        except httpx.TimeoutException:
            console.print(f"  [red]❌ Timeout conectando a Ollama[/red]")
            console.print(f"  [yellow]El modelo {modelo} puede no estar instalado o tardar mucho.[/yellow]")
            console.print()
            return False

    if resp.status_code != 200:
        console.print(f"  [red]❌ Error HTTP {resp.status_code}: {resp.text}[/red]")
        console.print(f"  [yellow]Sugerencias:[/yellow]")
        console.print(f"    • Verifica que 'ollama pull {modelo}' se haya completado")
        console.print(f"    • Revisa los logs de Ollama: ollama serve 2>&1 | tail -20")
        console.print()
        return False

    data = resp.json()
    vector = data.get("embedding", [])
    dims = len(vector)

    if dims == 0:
        console.print("  [red]❌ El vector de embedding está vacío[/red]")
        console.print()
        return False

    # Mostrar info
    console.print(f"  [green]✅ Embedding generado correctamente[/green]")
    console.print(f"  Dimensiones totales: [bold]{dims}[/bold]")
    console.print(
        f"  Primeras 5 dimensiones: "
        f"[dim]{vector[:5]}[/dim]"
    )
    console.print()

    # Verificar dimensión esperada
    dims_esperadas: dict[str, int] = {
        "mxbai-embed-large": 1024,
        "nomic-embed-text": 768,
        "all-minilm": 384,
    }
    modelo_base = _modelo_sin_tag(modelo)
    esperada = dims_esperadas.get(modelo_base, 0)
    if esperada and dims != esperada:
        console.print(
            f"  [yellow]⚠️  Dimensión inesperada: se esperaba {esperada}, "
            f"se obtuvo {dims}[/yellow]"
        )
    else:
        console.print(f"  [green]✅ Dimensión correcta para {modelo_base}[/green]")
    console.print()
    return True


# ═══════════════════════════════════════════════════════════════════
#  Paso 7 — Probar chat model
# ═══════════════════════════════════════════════════════════════════


def paso7_probar_chat(modelo: str | None) -> bool:
    """Prueba el modelo chat con un prompt simple. Opcional, pregunta primero."""
    console.rule("[bold cyan]Paso 7: Probar Chat Model[/bold cyan]")
    console.print()

    if not modelo:
        console.print("  [yellow]⚠️  No hay modelo chat seleccionado. Omitiendo prueba.[/yellow]")
        console.print()
        return False

    probar = Confirm.ask(
        f"  ¿Probar el modelo chat [bold]{_modelo_sin_tag(modelo)}[/bold]?",
        default=True,
    )
    if not probar:
        console.print("  [dim]Omitiendo prueba del modelo chat.[/dim]")
        console.print()
        return False

    with console.status(f"[yellow]Enviando prompt a {modelo}...[/yellow]"):
        import httpx

        try:
            resp = httpx.post(
                f"{OLLAMA_HOST}/api/generate",
                json={
                    "model": modelo,
                    "prompt": "Responde solo 'OK' si funciono.",
                    "stream": False,
                    "format": "json",
                },
                timeout=60,
            )
        except httpx.ConnectError:
            console.print(
                f"  [red]❌ No se pudo conectar a Ollama en {OLLAMA_HOST}[/red]"
            )
            console.print()
            return False
        except httpx.TimeoutException:
            console.print(
                f"  [red]❌ Timeout — el modelo '{modelo}' no respondió en 60s[/red]\n"
                f"  [yellow]Posibles causas:[/yellow]\n"
                f"    • El modelo no está instalado (ejecuta: ollama pull {modelo})\n"
                f"    • El modelo no soporta 'format: json'\n"
                f"    • Hardware insuficiente (necesitas más RAM/GPU)\n\n"
                f"  [yellow]Alternativas:[/yellow]\n"
                f"    • Prueba un modelo más pequeño como 'llama3.2:3b' o 'phi4-mini:3.8b'\n"
                f"    • Si usas CPU, modelos >3B params pueden ser lentos\n"
            )
            console.print()
            return False

    if resp.status_code == 200:
        data = resp.json()
        response_text = data.get("response", "")
        # Verificar que respondió algo
        if response_text:
            console.print(f"  [green]✅ Chat model responde correctamente[/green]")
            console.print(f"  Respuesta: [bold]{response_text[:200]}[/bold]")
            console.print()
            return True
        else:
            console.print(f"  [yellow]⚠️  Respuesta vacía del modelo[/yellow]")
            console.print()
            return False
    else:
        console.print(
            f"  [red]❌ Error HTTP {resp.status_code}[/red]\n"
            f"  Mensaje: {resp.text[:300]}\n\n"
            f"  [yellow]El modelo '{modelo}' podría no soportar format:json[/yellow]\n"
            f"  [yellow]Sugerencias:[/yellow]\n"
            f"    • Prueba 'llama3.2:3b' o 'phi4-mini:3.8b' que tienen buen soporte JSON\n"
            f"    • Modelos muy pequeños como 'qwen3.5:0.5b' pueden fallar con format constraint\n"
            f"    • Verifica que ollama esté actualizado: ollama --version\n"
        )
        console.print()
        return False


# ═══════════════════════════════════════════════════════════════════
#  Paso 8 — Resumen + escribir configuración
# ═══════════════════════════════════════════════════════════════════


def paso8_resumen(config: dict[str, Any]) -> None:
    """Muestra tabla resumen y escribe ~/.zyro/config.yaml."""
    console.rule("[bold cyan]Paso 8: Resumen Final[/bold cyan]")
    console.print()

    # ── Tabla resumen ──────────────────────────────────────────
    estados = config.get("_estados", {})

    def icono(ok: bool | str) -> str:
        if ok is True or ok == "ok":
            return "[green]✅[/green]"
        elif ok is False or ok == "error":
            return "[red]❌[/red]"
        else:
            return f"[yellow]{ok}[/yellow]"

    table = Table(
        title="[bold]ZyroAgentCLI — Resumen de Instalación[/bold]",
        title_style="bold cyan",
        border_style="cyan",
    )
    table.add_column("Componente", style="bold")
    table.add_column("Estado", justify="center")
    table.add_column("Detalle", style="dim")

    table.add_row(
        "Python",
        icono(estados.get("python", False)),
        str(estados.get("python_version", "?")),
    )
    table.add_row(
        "Ollama",
        icono(estados.get("ollama", False)),
        estados.get("ollama_version", ""),
    )
    table.add_row(
        "Embeddings",
        icono(estados.get("embedding", False)),
        config.get("embeddings", {}).get("model", "—"),
    )
    table.add_row(
        "Chat LLM",
        icono(estados.get("chat", False)),
        config.get("chat", {}).get("model", "—"),
    )
    gpu_info = config.get("gpu", {})
    gpu_icon = "[green]✅[/green]" if gpu_info.get("detected") else "[yellow]⚠️[/yellow]"
    gpu_name = gpu_info.get("name", "No detectada")
    table.add_row("GPU", gpu_icon, str(gpu_name))
    table.add_row(
        "Embedding test",
        icono(estados.get("embedding_test", False)),
        f"{estados.get('embedding_dims', '—')} dims",
    )
    table.add_row(
        "Chat test",
        icono(estados.get("chat_test", False)),
        "OK" if estados.get("chat_test") else "—",
    )

    console.print(table)
    console.print()

    # ── Escribir config (formato compatible con Go) ────────────
    config_dir = Path.home() / ".zyro"
    config_dir.mkdir(parents=True, exist_ok=True)
    config_path = config_dir / "config.yaml"

    # Obtener valores de modelos (con defaults)
    embed_model = config.get("embeddings", {}).get("model", "mxbai-embed-large")
    embed_dims = config.get("embeddings", {}).get("dims", 1024)
    chat_model = config.get("chat", {}).get("model", "phi4-mini:3.8b")

    # Construir estructura compatible con el struct Go Config
    go_config = {
        "version": "2.0.0",
        "services": {
            "ollama_url": "",
            "helixdb_url": "",
            "embedding_model": embed_model,
            "embedding_dims": embed_dims,
            "chat_model": chat_model,
        },
    }

    # Agregar metadata adicional que Go ignora pero Python usa
    go_config["install_date"] = config.get("install_date")
    if config.get("gpu"):
        go_config["gpu"] = config["gpu"]

    import yaml

    try:
        with open(config_path, "w") as f:
            yaml.dump(
                go_config,
                f,
                default_flow_style=False,
                sort_keys=False,
                allow_unicode=True,
            )
        console.print(
            f"  [green]✅[/green] Configuración escrita en: [bold]{config_path}[/bold]"
        )
    except Exception as e:
        console.print(f"  [red]❌ Error escribiendo configuración: {e}[/red]")

    console.print()

    # ── Panel final ────────────────────────────────────────────
    all_ok = (
        estados.get("python", False)
        and estados.get("ollama", False)
        and estados.get("embedding", False)
    )

    if all_ok:
        final = Panel.fit(
            Text(
                "🎉  ¡Instalación completada con éxito!  🎉\n\n"
                "ZyroAgentCLI está listo para usar.",
                justify="center",
                style="bold green",
            ),
            border_style="green",
            padding=(1, 4),
        )
        console.print(final)

        # Preguntar si ejecutar prueba rápida
        console.print()
        ejecutar_prueba = Confirm.ask(
            "  ¿Ejecutar una prueba rápida completa?", default=True
        )
        if ejecutar_prueba:
            console.print()
            _ejecutar_prueba_rapida(config)
    else:
        final = Panel.fit(
            Text(
                "⚠️   Instalación parcial   ⚠️\n\n"
                "Algunos componentes no están listos.\n"
                "Revisa los mensajes anteriores para más detalles.",
                justify="center",
                style="yellow",
            ),
            border_style="yellow",
            padding=(1, 4),
        )
        console.print(final)

    console.print()


def _ejecutar_prueba_rapida(config: dict[str, Any]) -> None:
    """Ejecuta una prueba integral de los componentes."""
    console.rule("[bold green]Prueba Rápida[/bold green]")
    console.print()

    import httpx

    modelo_embed = config.get("embeddings", {}).get("model", "mxbai-embed-large")
    modelo_chat = config.get("chat", {}).get("model", "llama3.2")
    errores = 0

    # 1. Embeddings
    with console.status("[yellow]Probando embeddings...[/yellow]"):
        try:
            resp = httpx.post(
                f"{OLLAMA_HOST}/api/embeddings",
                json={
                    "model": _modelo_sin_tag(modelo_embed),
                    "prompt": "ZyroAgentCLI test rápido.",
                },
                timeout=30,
            )
            if resp.status_code == 200:
                dims = len(resp.json().get("embedding", []))
                console.print(f"  [green]✅[/green] Embeddings: {dims} dimensiones")
            else:
                console.print(f"  [red]❌[/red] Embeddings: HTTP {resp.status_code}")
                errores += 1
        except Exception as e:
            console.print(f"  [red]❌[/red] Embeddings: {e}")
            errores += 1

    # 2. Chat
    with console.status("[yellow]Probando chat model...[/yellow]"):
        try:
            resp = httpx.post(
                f"{OLLAMA_HOST}/api/generate",
                json={
                    "model": _modelo_sin_tag(modelo_chat),
                    "prompt": "Responde solo OK.",
                    "stream": False,
                },
                timeout=60,
            )
            if resp.status_code == 200:
                respuesta = resp.json().get("response", "").strip()
                console.print(f"  [green]✅[/green] Chat model responde: {respuesta[:50]}")
            else:
                console.print(f"  [red]❌[/red] Chat model: HTTP {resp.status_code}")
                errores += 1
        except Exception as e:
            console.print(f"  [red]❌[/red] Chat model: {e}")
            errores += 1

    console.print()
    if errores == 0:
        console.print(
            Panel.fit(
                "[green]✅  Todos los componentes funcionan correctamente[/green]",
                border_style="green",
            )
        )
    else:
        console.print(
            Panel.fit(
                f"[yellow]⚠️  {errores} prueba(s) fallaron — revisa los mensajes arriba[/yellow]",
                border_style="yellow",
            )
        )
    console.print()


# ═══════════════════════════════════════════════════════════════════
#  Detección de duplicados npm
# ═══════════════════════════════════════════════════════════════════


def _detectar_paquetes_duplicados() -> None:
    """Detecta si hay paquetes npm duplicados (zyro, zyrocli, zyro-agent-cli) y advierte."""
    console.rule("[bold yellow]Verificación de paquetes npm[/bold yellow]")
    console.print()

    import subprocess, json

    # Lista de paquetes que podrían colisionar
    paquetes_sospechosos = ["zyro", "zyrocli", "zyro-agent-cli"]
    instalados = []

    try:
        # npm list -g --json para ver paquetes globales
        result = subprocess.run(
            ["npm", "list", "-g", "--json", "--depth=0"],
            capture_output=True, text=True, timeout=15
        )
        if result.returncode == 0:
            data = json.loads(result.stdout)
            dependencies = data.get("dependencies", {})
            for pkg_name in paquetes_sospechosos:
                if pkg_name in dependencies:
                    pkg_info = dependencies[pkg_name]
                    instalados.append({
                        "name": pkg_name,
                        "version": pkg_info.get("version", "?"),
                        "path": pkg_info.get("resolved", "?"),
                    })
    except (json.JSONDecodeError, subprocess.TimeoutExpired, FileNotFoundError):
        pass

    # También verificar con which si hay binarios
    for cmd in ["zyro", "zyrocli"]:
        path = shutil.which(cmd)
        if path:
            console.print(f"  ℹ️  Binario '{cmd}' encontrado en: [dim]{path}[/dim]")

    if len(instalados) > 1:
        console.print(
            Panel(
                f"[yellow]⚠️  Se detectaron {len(instalados)} paquetes que pueden solaparse:[/yellow]\n\n"
                + "\n".join(f"    • [bold]{p['name']}[/bold] v{p['version']}" for p in instalados)
                + "\n\n"
                "  [bold]Recomendación:[/bold] mantener solo [green]zyro-agent-cli[/green]\n"
                "  Para limpiar duplicados manualmente:\n"
                + "".join(f"    npm uninstall -g {p['name']}\n" for p in instalados if p['name'] != 'zyro-agent-cli')
                + "\n  O puedes hacerlo ahora.\n",
                title="Paquetes duplicados detectados",
                border_style="yellow",
            )
        )

        from rich.prompt import Confirm
        if Confirm.ask("  ¿Limpiar paquetes duplicados?", default=True):
            for p in instalados:
                if p["name"] != "zyro-agent-cli":
                    with console.status(f"Desinstalando {p['name']}..."):
                        rc, out, err = _ejecutar(
                            ["npm", "uninstall", "-g", p["name"]], timeout=30
                        )
                    if rc == 0:
                        console.print(f"  [green]✅ {p['name']} desinstalado[/green]")
                    else:
                        console.print(f"  [red]❌ Error desinstalando {p['name']}: {err}[/red]")
    elif len(instalados) == 1:
        console.print(f"  [green]✅ Solo hay un paquete instalado: {instalados[0]['name']} v{instalados[0]['version']}[/green]")
    else:
        console.print("  ℹ️  No se detectaron paquetes npm de Zyro.")

    console.print()


# ═══════════════════════════════════════════════════════════════════
#  Main
# ═══════════════════════════════════════════════════════════════════


def main() -> None:
    """Punto de entrada principal."""
    # ── Paso 1 ──────────────────────────────────────────────
    if not paso1_bienvenida():
        sys.exit(1)

    # Contenedor de configuración
    config: dict[str, Any] = {
        "version": "1.0",
        "install_date": date.today().isoformat(),
        "_estados": {},  # campos temporales, se limpian antes de escribir
    }
    estados: dict[str, Any] = config["_estados"]

    estados["python"] = True
    estados["python_version"] = f"{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}"

    # ── Detectar duplicados npm ─────────────────────────────
    _detectar_paquetes_duplicados()

    # ── Paso 2 ──────────────────────────────────────────────
    input("\n  Presiona Enter para continuar...")
    console.print()
    ollama_ok = paso2_ollama()
    estados["ollama"] = ollama_ok
    if ollama_ok:
        rc, out, _ = _ejecutar(["ollama", "--version"])
        estados["ollama_version"] = out if rc == 0 else "?"
    else:
        estados["ollama_version"] = "No disponible"

    # ── Paso 3 ──────────────────────────────────────────────
    input("\n  Presiona Enter para continuar...")
    console.print()
    modelo_embed = paso3_embeddings()
    estados["embedding"] = modelo_embed is not None
    if modelo_embed:
        config["embeddings"] = {
            "provider": "ollama",
            "model": _modelo_sin_tag(modelo_embed),
            "dims": _dimensiones_modelo(modelo_embed),
        }
    else:
        config["embeddings"] = {
            "provider": "ollama",
            "model": "mxbai-embed-large",
            "dims": 1024,
        }

    # ── Paso 4 ──────────────────────────────────────────────
    input("\n  Presiona Enter para continuar...")
    console.print()
    modelo_chat = paso4_chat()
    estados["chat"] = modelo_chat is not None
    if modelo_chat:
        config["chat"] = {
            "provider": "ollama",
            "model": _modelo_sin_tag(modelo_chat),
        }
    else:
        config["chat"] = {
            "provider": "ollama",
            "model": "llama3.2",
        }

    # ── Paso 5 ──────────────────────────────────────────────
    input("\n  Presiona Enter para continuar...")
    console.print()
    gpu_info = paso5_gpu()
    config["gpu"] = {
        "detected": gpu_info.get("detected", False),
        "type": gpu_info.get("type", "none"),
        "name": gpu_info.get("name", None),
    }
    if gpu_info.get("backend_instalado"):
        config["gpu"]["backend_instalado"] = gpu_info["backend_instalado"]
    if gpu_info.get("backend_verificado"):
        config["gpu"]["backend_verificado"] = gpu_info["backend_verificado"]
    if gpu_info.get("recomendacion"):
        config["gpu"]["recomendacion"] = gpu_info["recomendacion"]

    # ── Paso 6 ──────────────────────────────────────────────
    input("\n  Presiona Enter para continuar...")
    console.print()
    test_embed_ok = paso6_probar_embeddings(modelo_embed)
    estados["embedding_test"] = test_embed_ok
    if test_embed_ok and modelo_embed:
        estados["embedding_dims"] = _dimensiones_modelo(modelo_embed)

    # ── Paso 7 ──────────────────────────────────────────────
    input("\n  Presiona Enter para continuar...")
    console.print()
    test_chat_ok = paso7_probar_chat(modelo_chat)
    estados["chat_test"] = test_chat_ok

    # ── Paso 8 ──────────────────────────────────────────────
    input("\n  Presiona Enter para continuar...")
    console.print()
    paso8_resumen(config)


def _dimensiones_modelo(modelo: str) -> int:
    """Retorna las dimensiones esperadas para un modelo de embeddings."""
    dims_map: dict[str, int] = {
        "mxbai-embed-large": 1024,
        "nomic-embed-text": 768,
        "all-minilm": 384,
    }
    base = _modelo_sin_tag(modelo)
    return dims_map.get(base, 0)


if __name__ == "__main__":
    main()
