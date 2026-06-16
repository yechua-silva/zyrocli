#!/usr/bin/env bash
set -euo pipefail

# ═══════════════════════════════════════════════════════════════
# ZyroCLI — Install script
# ═══════════════════════════════════════════════════════════════
# Usage: ./scripts/install.sh [version]
#
# Descarga el binario pre-compilado desde GitHub Releases.
# Si no se especifica versión, descarga la última (latest).
# ═══════════════════════════════════════════════════════════════

REPO="secko/zyrocli"
VERSION="${1:-latest}"
INSTALL_DIR="${HOME}/.local/bin"
BINARY_NAME="zyrocli"

# ── Detectar plataforma ───────────────────────────────────────
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac

PLATFORM="${OS}_${ARCH}"
BINARY="${BINARY_NAME}"
[ "$OS" = "windows" ] && BINARY="${BINARY}.exe"

# ── Obtener última versión si es latest ────────────────────────
if [ "$VERSION" = "latest" ]; then
    echo "🔍 Fetching latest version..."
    VERSION=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | cut -d'"' -f4 \
        | sed 's/^v//')
    if [ -z "$VERSION" ]; then
        echo "❌ Could not determine latest version"
        exit 1
    fi
    echo "   Latest version: v${VERSION}"
fi

URL="https://github.com/${REPO}/releases/download/v${VERSION}/zyrocli_${VERSION}_${PLATFORM}.tar.gz"

echo "🚀 Installing ZyroCLI v${VERSION} for ${PLATFORM}..."
echo "   Downloading from ${URL}"

mkdir -p "$INSTALL_DIR"

# Descargar y extraer tarball
curl -sSL "$URL" | tar -xz -C "$INSTALL_DIR" "${BINARY}" 2>/dev/null || {
    # Fallback: descarga directa del binario (sin tarball)
    echo "   ⚠️  Tarball not found, trying direct download..."
    curl -sSL "https://github.com/${REPO}/releases/download/v${VERSION}/${BINARY}" \
        -o "${INSTALL_DIR}/${BINARY}"
}
chmod +x "${INSTALL_DIR}/${BINARY}"

echo ""
echo "  ─────────────────────────────────────────────"
echo "  ✅ ZyroCLI v${VERSION} installed!"
echo "  ─────────────────────────────────────────────"
echo ""
echo "  Binary:  ${INSTALL_DIR}/${BINARY}"
echo ""
echo "  Quick start:"
echo "    zyrocli --help"
echo "    zyrocli setup"
echo ""
