#!/usr/bin/env bash
# Ejecuta: ZyroCLI + Boomerang (F0→F4)
set -euo pipefail
TASK_FILE="${1:-../tasks/jwt-auth.md}"
echo "🚀 [zyro] Ejecutando tarea con ZyroCLI (Boomerang + HelixDB)..."
echo "   Tarea: $TASK_FILE"

TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

# Crear handoff.yaml desde la tarea
TASK_CONTENT=$(cat "$TASK_FILE")
cat > handoff.yaml << HANDOFF
version: "2.0"
source: { system: "benchmark" }
project:
  name: jwt-benchmark
  language: go
  repository: ""
validated_idea:
  problem: "Benchmark task: add JWT auth"
  success_criteria:
    - "go build ./... compila"
    - "go test ./... pasa"
    - "POST /login returns JWT"
    - "GET /profile works with valid token"
user_story:
  story: "Add JWT authentication to Go API"
  acceptance: "All 4 criteria met"
mvp: { scope: "jwt auth", features: ["login", "middleware", "profile"] }
governance: { mode: "auto" }
testing: { strategy: "unit" }
limits: { max_loops: 3, phase_timeout: "10m" }
HANDOFF

# Inicializar proyecto
zyrocli init handoff.yaml 2>&1 || true

# Ejecutar pipeline completo
zyrocli run 2>&1 | tee /tmp/benchmark-zyro-output.log

cd / && rm -rf "$TMP_DIR"
echo "✅ [zyro] Completado"
