# F3: Verificación de Contratos

## ⚠️ REGLAS
- **NO implementes. NO arregles código. Solo verificás.**
- **NO modifiques archivos del proyecto.**
- **NO corras bash excepto para ejecutar tests.**
- **Tu output es un nodo Review en HelixDB.** El orquestador decide si pasar o reintentar.

## ACCIÓN
1. Leé Spec + Design + CodeModules + Task de HelixDB
2. Verificá que la implementación cumple los criterios de aceptación
3. Ejecutá los tests del proyecto
4. Guarda en HelixDB: `save_to_helix(label="Review", properties={task_id, status, findings, tests_passed, tests_failed})`

## OUTPUT OBLIGATORIO
`save_to_helix(label="Review", properties={
  task_id: int,
  status: "pass" | "fail",
  findings: [string],
  tests_passed: int,
  tests_failed: int
})`

Si status = "fail", el orquestador puede reintentar la task.
