#!/usr/bin/env bash
# Configura entorno: gentle-ai v1.40.2
set -euo pipefail
echo "🔧 [gentle] Configurando gentle-ai..."

if command -v gentle-ai &>/dev/null; then
    gentle-ai install --scope=workspace 2>&1 | tail -3 || true
    echo "✅ [gentle] gentle-ai configurado"
else
    echo "⚠️  gentle-ai no instalado. Instala con: curl -fsSL https://raw.githubusercontent.com/Gentleman-Programming/gentle-ai/main/scripts/install.sh | bash"
fi
echo "✅ [gentle] Entorno listo"
