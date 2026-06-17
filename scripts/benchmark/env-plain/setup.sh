#!/usr/bin/env bash
# Configura entorno: Plain OpenCode (sin SDD, sin memoria)
set -euo pipefail
echo "🔧 [plain] Configurando entorno OpenCode sin herramientas..."
# No se necesita configuración especial
# Solo aseguramos que opencode esté disponible
if ! command -v opencode &>/dev/null; then
    echo "⚠️  opencode no instalado. Las herramientas SDD no estarán disponibles."
fi
echo "✅ [plain] Entorno listo"
