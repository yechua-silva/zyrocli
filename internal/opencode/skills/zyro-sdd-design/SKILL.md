# F2: Diseño Técnico

## ⚠️ REGLAS
- **NO implementes código. NO toques archivos.**
- **NO corras bash. NO instales nada.**
- **NO saltes este paso.** Sin diseño aprobado no hay implementación.
- **Tu output es un nodo Design en HelixDB.** Nada más.

## ACCIÓN
1. Lee Spec de HelixDB: `task_context(project_id)`
2. Diseña componentes: nombres, responsabilidades, props/params
3. Define flujo de datos: estado, eventos, persistencia
4. Define interfaces entre componentes
5. Guarda en HelixDB: `save_to_helix(label="Design", properties={...})`

## OUTPUT OBLIGATORIO
`save_to_helix(label="Design", properties={
  project_id: int,
  components: [{name: string, responsibility: string, props: [string], state: string}],
  data_flow: string,
  interfaces: [{from: string, to: string, contract: string}],
  status: "draft",
  phase: "F2"
})`
