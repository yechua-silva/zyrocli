#!/usr/bin/env bash
# Ejecuta: Plain OpenCode (sin memoria, sin SDD)
set -euo pipefail
TASK_FILE="${1:-../tasks/jwt-auth.md}"
echo "🚀 [plain] Ejecutando tarea con OpenCode (plain)..."
echo "   Tarea: $TASK_FILE"
echo "   HTTP_PROXY=$HTTP_PROXY"

# Crear proyecto temporal
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

# Inicializar proyecto Go básico
cat > go.mod << 'GO.mod'
module github.com/benchmark/jwt-api
go 1.24
GO.mod

cat > main.go << 'MAIN.go'
package main

import (
    "encoding/json"
    "log"
    "net/http"
)

func main() {
    http.HandleFunc("/ping", pingHandler)
    http.HandleFunc("/status", statusHandler)
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
MAIN.go

# Ejecutar opencode con la tarea
opencode "$TMP_DIR" --input "$TASK_FILE" 2>&1 | tee /tmp/benchmark-plain-output.log

# Limpiar
cd / && rm -rf "$TMP_DIR"
echo "✅ [plain] Completado"
