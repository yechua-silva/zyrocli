#!/usr/bin/env bash
# Benchmark: 3 Jaulas
# Uso: bash run.sh [--dry-run] [--jaula plain|gentle|zyro]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORT_DIR="$SCRIPT_DIR/report"
LIB_DIR="$SCRIPT_DIR/lib"
TASKS_DIR="$SCRIPT_DIR/tasks"
DRY_RUN=false
JAULA="all"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --dry-run) DRY_RUN=true; shift ;;
        --jaula) JAULA="$2"; shift 2 ;;
        *) shift ;;
    esac
done

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${CYAN}ℹ️${NC} $1"; }
ok()    { echo -e "${GREEN}✅${NC} $1"; }
warn()  { echo -e "${YELLOW}⚠️${NC} $1"; }
error() { echo -e "${RED}❌${NC} $1"; }

# Verificar dependencias
check_deps() {
    info "Verificando dependencias..."
    python3 -c "import tiktoken" 2>/dev/null || warn "tiktoken no instalado (pip install tiktoken)"
    python3 -c "import matplotlib" 2>/dev/null || warn "matplotlib no instalado (pip install matplotlib)"
    command -v opencode &>/dev/null || warn "opencode no instalado"
    command -v gentle-ai &>/dev/null || warn "gentle-ai no instalado"
    command -v zyrocli &>/dev/null || warn "zyrocli no en PATH"
    command -v go &>/dev/null || error "go no instalado"
}

# Iniciar proxy
start_proxy() {
    local jaula="$1"
    local port="$2"
    local log_dir="$REPORT_DIR/logs/$jaula"
    mkdir -p "$log_dir"
    
    cd "$LIB_DIR"
    HTTP_PROXY="" HTTPS_PROXY="" NO_PROXY="" \
    python3 proxy.py --port "$port" --jaula "$jaula" --model "gpt-4o" &
    PROXY_PID=$!
    cd "$SCRIPT_DIR"
    
    # Esperar a que proxy esté listo
    sleep 2
    info "Proxy en puerto $port (PID: $PROXY_PID)"
    export HTTP_PROXY="http://127.0.0.1:$port"
    export HTTPS_PROXY="http://127.0.0.1:$port"
}

# Detener proxy
stop_proxy() {
    if [ -n "${PROXY_PID:-}" ]; then
        kill "$PROXY_PID" 2>/dev/null || true
        wait "$PROXY_PID" 2>/dev/null || true
        unset HTTP_PROXY HTTPS_PROXY
        ok "Proxy detenido"
    fi
}

# Jaula 1: Plain OpenCode
run_plain() {
    info "=== Jaula 1: Plain OpenCode ==="
    start_proxy "plain" 8081
    
    cd "$SCRIPT_DIR/env-plain"
    # Configurar y ejecutar
    if [ -f setup.sh ]; then bash setup.sh; fi
    if [ -f run.sh ]; then bash run.sh "$TASKS_DIR/jwt-auth.md"; fi
    cd "$SCRIPT_DIR"
    
    stop_proxy
    ok "Jaula 1 completada"
}

# Jaula 2: gentle-ai
run_gentle() {
    info "=== Jaula 2: gentle-ai v1.40.2 ==="
    start_proxy "gentle" 8082
    
    cd "$SCRIPT_DIR/env-gentle"
    if [ -f setup.sh ]; then bash setup.sh; fi
    if [ -f run.sh ]; then bash run.sh "$TASKS_DIR/jwt-auth.md"; fi
    cd "$SCRIPT_DIR"
    
    stop_proxy
    ok "Jaula 2 completada"
}

# Jaula 3: ZyroCLI
run_zyro() {
    info "=== Jaula 3: ZyroCLI + Boomerang ==="
    start_proxy "zyro" 8083
    
    cd "$SCRIPT_DIR/env-zyro"
    if [ -f setup.sh ]; then bash setup.sh; fi
    if [ -f run.sh ]; then bash run.sh "$TASKS_DIR/jwt-auth.md"; fi
    cd "$SCRIPT_DIR"
    
    stop_proxy
    ok "Jaula 3 completada"
}

# Generar reporte
generate_report() {
    info "Generando reporte..."
    cd "$LIB_DIR"
    python3 -c "
import sys, json, os
sys.path.insert(0, '.')
from pathlib import Path

# Leer datos de las 3 jaulas
report_dir = Path('$REPORT_DIR')
logs_dir = report_dir / 'logs'
data = {}

for jaula in ['plain', 'gentle', 'zyro']:
    data_path = logs_dir / jaula / 'data.json'
    if data_path.exists():
        with open(data_path) as f:
            data[jaula] = json.load(f)
    else:
        data[jaula] = {'summary': {'total_tokens': 0, 'input_tokens': 0, 'output_tokens': 0, 'turns': 0}}

# Guardar datos combinados
with open(report_dir / 'data.json', 'w') as f:
    json.dump(data, f, indent=2)

print(f'Datos guardados: {list(data.keys())}')
"
    
    # Generar HTML
    python3 "$LIB_DIR/generate_report.py" "$REPORT_DIR"
    
    ok "Reporte generado en $REPORT_DIR/index.html"
}

# Main
main() {
    echo "═══════════════════════════════════════════════"
    echo "  🏋️  Benchmark: 3 Jaulas"
    echo "  Plain OpenCode vs gentle-ai vs ZyroCLI"
    echo "═══════════════════════════════════════════════"
    echo ""
    
    check_deps
    echo ""
    
    if [ "$DRY_RUN" = "true" ]; then
        info "Dry-run: verificando estructura..."
        ls -la "$LIB_DIR/proxy.py" && ok "proxy.py existe"
        ls -la "$LIB_DIR/token_counter.py" && ok "token_counter.py existe"
        ls -la "$TASKS_DIR/jwt-auth.md" && ok "jwt-auth.md existe"
        info "Dry-run completo ✅"
        return 0
    fi
    
    case "$JAULA" in
        plain)  run_plain ;;
        gentle) run_gentle ;;
        zyro)   run_zyro ;;
        all)
            run_plain
            echo ""
            run_gentle
            echo ""
            run_zyro
            echo ""
            generate_report
            ;;
    esac
}

trap stop_proxy EXIT
main "$@"
