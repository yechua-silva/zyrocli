#!/usr/bin/env bash
# Configura entorno: ZyroCLI + Boomerang + HelixDB
set -euo pipefail
echo "🔧 [zyro] Configurando ZyroCLI..."

# Verificar zyrocli
if command -v zyrocli &>/dev/null; then
    echo "✅ zyrocli disponible: $(zyrocli --version 2>&1)"
else
    echo "⚠️  zyrocli no en PATH. Usando binario local..."
    if [ -f "../zyrocli" ]; then
        export PATH="$PATH:$(cd .. && pwd)"
    fi
fi

# Iniciar servicios si no están corriendo
if command -v zyrocli &>/dev/null; then
    zyrocli db status 2>/dev/null || zyrocli db start 2>/dev/null || echo "⚠️  HelixDB no disponible"
fi

echo "✅ [zyro] Entorno listo"
