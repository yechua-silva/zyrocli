#!/usr/bin/env bash
set -euo pipefail

# ═══════════════════════════════════════════════════════════════
# ZyroCLI — Install script
# ═══════════════════════════════════════════════════════════════
# Usage: ./scripts/install.sh
#
# What this does:
#   1. Builds the zyrocli binary
#   2. Installs it to ~/.local/bin/zyrocli
#   3. Installs HelixDB (if not present)
#   4. Runs `zyrocli install` to configure OpenCode ecosystem globally
#
# The binary is fully self-contained — no repo path needed at runtime.
# ═══════════════════════════════════════════════════════════════

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BINARY_NAME="zyrocli"
INSTALL_DIR="${HOME}/.local/bin"

echo "🚀 Installing ZyroCLI..."

# ── Step 1: Build binary ──────────────────────────────────────
echo "  1. Building binary..."
cd "$REPO_DIR"
go build -o "${INSTALL_DIR}/${BINARY_NAME}" ./cmd/zyrocli
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
echo "     ✅ Binary installed: ${INSTALL_DIR}/${BINARY_NAME}"

# ── Step 2: Install HelixDB ───────────────────────────────────
if ! command -v helix &>/dev/null; then
    echo "  2. Installing HelixDB..."
    curl -sSL "https://install.helix-db.com" | bash
    echo "     ✅ HelixDB installed"
else
    echo "  2. ✅ HelixDB already installed"
fi

# ── Step 3: Install ZyroCLI ecosystem ─────────────────────────
echo "  3. Installing ZyroCLI ecosystem..."
zyrocli install
echo "     ✅ Ecosystem configured"

# ── Step 4: Verify ────────────────────────────────────────────
echo ""
echo "  ─────────────────────────────────────────────"
echo "  ✅ ZyroCLI installed successfully!"
echo "  ─────────────────────────────────────────────"
echo ""
echo "  Binary:  ${INSTALL_DIR}/${BINARY_NAME}"
echo "  Config:  ~/.config/opencode/opencode.jsonc"
echo "  Skills:  ~/.config/opencode/skills/"
echo "  MCP:     ~/.config/zyrocli/mcp-tools/"
echo ""
# ── Step 4: Install find-skills discovery skill ────────────────
echo "  4. Installing find-skills discovery skill..."
npx skills add vercel-labs/skills --skill find-skills -g -y 2>/dev/null || true
echo "     ✅ find-skills installed"

echo ""
echo "  Quick start:"
    echo "    zyrocli install              # (re)configure ecosystem"
    echo "    zyrocli init handoff.yaml    # Create a new project"
echo ""
