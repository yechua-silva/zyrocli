# zyrocli

Orquestador Go para desarrollo asistido por IA con pipeline F0-F4.

## Instalación

```bash
npx zyrocli setup
```

O global:

```bash
npm install -g zyrocli
zyrocli setup
```

## Uso

```bash
zyrocli setup              # Instalar todo (Go, uv, HelixDB, Ollama, etc.)
zyrocli install            # Configurar skills, MCP tools y OpenCode
zyrocli doctor             # Diagnosticar el entorno
zyrocli init handoff.yaml  # Inicializar proyecto desde un contrato
zyrocli run --phase F0     # Ejecutar fase de investigación
```
