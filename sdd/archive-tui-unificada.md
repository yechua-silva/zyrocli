# Archive — TUI Unificada

## Fecha
2026-06-18

## Fases
- F0: Exploración de TUIs existentes (Go Bubbletea + Python Rich)
- F1: Plan de unificación + investigación GPU Mesa Vulkan
- F2: Diseño del menú principal y flujos
- F3: Implementación
- F4: Archive

## Cambios realizados

### Menú principal interactivo
- `zyrocli` sin args ahora muestra menú Bubbletea con 5 opciones
- Navegación con ↑/↓, selección con Enter, salida con q
- Opciones: Instalación, Servicios, Modelos IA, Auto-inicio, Salir

### Color de marca cambiado
- Violeta (#7C3AED) → Naranja (#F97316)
- Consistente en toda la TUI: logo, menú, selectores, confirm, steps

### Nuevos flujos TUI (Go Bubbletea)
- `models_flow.go`: selector de modelos embeddings + chat
- `gpu_flow.go`: detección y resumen de GPU
- `test_flow.go`: prueba de modelos contra Ollama
- `services_flow.go`: verificación de HelixDB + Ollama
- `autostart_flow.go`: configuración systemd para inicio automático
- `select_model.go`: componente reutilizable de lista seleccionable

### Eliminado
- Python TUI bridge (tui_launcher.go, tui_launcher_test.go)
- Dependencia de Python para el usuario (install/setup ya no llaman a install_tui.py)
- scripts/install_tui.py ya no se invoca desde Go (se mantiene como respaldo)

### Investigación GPU
- Documentada en docs/gpu-vulkan-drivers-research.md
- Guardada en HelixDB como ResearchFinding ID 5014
- Estrategia: Vulkan con ollama-vulkan como universal, ROCm/CUDA como avanzado

## Estado final
- Build: go build ./... OK
- Tests: 23/24 OK (1 preexistente por falta de TTY)
- Color: Naranja (#F97316) uniforme en toda la TUI
- Dependencias: Sin Python para el usuario
