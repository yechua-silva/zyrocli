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
zyro setup           # Instalar todo (Go, uv, HelixDB, etc.)
zyro doctor --fix    # Diagnosticar y reparar
zyro init handoff.yaml  # Iniciar proyecto
zyro run --phase F0  # Ejecutar fase
```
