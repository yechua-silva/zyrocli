# F1: Especificación Técnica

## ⚠️ REGLAS
- **NO escribas código. NO toques archivos.**
- **NO edites CSS/HTML/JS/TS/Go/Python.** No toques src/ ni components/ ni pages/.
- **NO crees componentes. NO diseñes UI.** No es tu trabajo.
- **NO corras npm/pip/go mod.** No instales nada.
- **NO leas archivos del proyecto.** Solo consultas a HelixDB.
- **Tu único output es UN nodo Spec en HelixDB.** Nada más.

## ACCIÓN
1. Busca el proyecto en HelixDB: `find_project(name)`
2. Lee contexto de Fase 0: `task_context(project_id)`
3. Define arquitectura, módulos, dependencias, testing
4. Guarda en HelixDB: `save_to_helix(label="Spec", properties={...})`

## OUTPUT OBLIGATORIO
`save_to_helix(label="Spec", properties={
  project_id: int,
  architecture: string,
  modules: [string],
  dependencies: [{name, version}],
  testing_strategy: string,
  status: "draft",
  phase: "F1"
})`
