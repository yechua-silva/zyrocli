#!/usr/bin/env bash
# Ejecuta: gentle-ai con SDD + Engram
set -euo pipefail
TASK_FILE="${1:-../tasks/jwt-auth.md}"
echo "🚀 [gentle] Ejecutando tarea con gentle-ai (SDD + Engram)..."
echo "   Tarea: $TASK_FILE"

TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

# Mismo proyecto base que plain
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

# Inicializar SDD
opencode "$TMP_DIR" --input "Ejecuta /sdd-init y luego completa esta tarea: $(cat $TASK_FILE)" 2>&1 | tee /tmp/benchmark-gentle-output.log

cd / && rm -rf "$TMP_DIR"
echo "✅ [gentle] Completado"
