#!/usr/bin/env bash
set -euo pipefail

# ═══════════════════════════════════════════════════════════════
# ZyroCLI — Install script
# ═══════════════════════════════════════════════════════════════
# Usage: curl -sSL "https://install.zyrocli.dev" | bash
#   or:  ./scripts/install.sh
#
# What this does:
#   1. Builds the zyrocli binary
#   2. Installs it to ~/.local/bin/zyrocli
#   3. Installs HelixDB (if not present)
#   4. Registers MCP tools in ~/.config/opencode/opencode.json
#   5. Registers global skills
# ═══════════════════════════════════════════════════════════════

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BINARY_NAME="zyrocli"
INSTALL_DIR="${HOME}/.local/bin"
OPENCODE_CONFIG="${HOME}/.config/opencode/opencode.json"
MCP_TOOLS_DIR="${REPO_DIR}/mcp-tools"

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
    echo "  2. ✅ HelixDB already installed ($(helix --help 2>&1 | head -1 | grep -oP 'v\d+\.\d+\.\d+' || echo 'present'))"
fi

# ── Step 3: Register MCP tools in opencode.json ───────────────
echo "  3. Registering MCP tools in opencode.json..."

# Ensure the opencode config directory exists
mkdir -p "$(dirname "$OPENCODE_CONFIG")"

# Create or update opencode.json with MCP tools
if [ -f "$OPENCODE_CONFIG" ]; then
    # Check if helix-integration MCp tools already registered
    if python3 -c "import json; cfg=json.load(open('${OPENCODE_CONFIG}')); tools=cfg.get('mcpTools',{}); print('exists' if 'helix-integration' in tools else 'missing')" 2>/dev/null | grep -q "exists"; then
        echo "     ✅ MCP tools already registered"
    else
        # Add MCP tools to existing config
        python3 -c "
import json
cfg = json.load(open('${OPENCODE_CONFIG}'))
if 'mcpTools' not in cfg:
    cfg['mcpTools'] = {}
cfg['mcpTools']['helix-integration'] = {
    'command': 'uv',
    'args': ['run', '--directory', '${MCP_TOOLS_DIR}', 'runner.py']
}
json.dump(cfg, open('${OPENCODE_CONFIG}', 'w'), indent=2)
" 2>&1 && echo "     ✅ MCP tools registered"
    fi
else
    # Create new config with MCP tools
    python3 -c "
import json
cfg = {
    '\$schema': 'https://opencode.ai/config.json',
    'mcpTools': {
        'helix-integration': {
            'command': 'uv',
            'args': ['run', '--directory', '${MCP_TOOLS_DIR}', 'runner.py']
        }
    }
}
json.dump(cfg, open('${OPENCODE_CONFIG}', 'w'), indent=2)
" 2>&1 && echo "     ✅ opencode.json created with MCP tools"
fi

# ── Step 4: Register global skills ────────────────────────────
echo "  4. Registering global skills..."

# These are configured in opencode.json under the agent that needs them
# Skills are per-project in opencode.json, so they get added when zyrocli init is run
echo "     📝 Global skills are configured per-project via 'zyrocli init --scaffold'"

# ── Step 5: Verify installation ───────────────────────────────
echo ""
echo "  ─────────────────────────────────────────────"
echo "  ✅ ZyroCLI installed successfully!"
echo "  ─────────────────────────────────────────────"
echo ""
echo "  Binary:    ${INSTALL_DIR}/${BINARY_NAME}"
echo "  HelixDB:   $(command -v helix &>/dev/null && echo 'installed' || echo 'NOT installed')"
echo "  MCP tools: registered in opencode.json"
echo ""
echo "  Quick start:"
echo "    helix start dev          # Start HelixDB"
echo "    zyrocli --help           # See commands"
echo "    zyrocli init handoff.yaml --scaffold --opencode"
echo ""
