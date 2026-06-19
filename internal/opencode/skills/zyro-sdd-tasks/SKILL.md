---
name: zyro-sdd-tasks
description: "F2: Divide el diseño en tareas atómicas — crea nodos Task en HelixDB"
---
# F2: Planificación de Tareas

## ⚠️ REGLAS
- **NO implementes código. NO diseñes.**
- **NO saltes el diseño.** Sin Design aprobado no hay tareas.
- **NO crees tasks sin leer Design + Spec de HelixDB.**
- **Cada tarea = una unidad implementable y testeable.** No más de 1 componente por tarea.

## ACCIÓN
1. Lee Spec + Design de HelixDB: `task_context(project_id)`
2. Divide en tareas atómicas (cada una ≤ 1 componente/función)
3. Ordena tareas por dependencias
4. Por cada tarea: `save_to_helix(label="Task", properties={...})`

## OUTPUT
Crear N nodos Task en HelixDB (mínimo 1).

`save_to_helix(label="Task", properties={
  project_id: int,
  name: string,
  description: string,
  depends_on: [int],
  acceptance_criteria: string,
  status: "pending",
  phase: "F2"
})`

## NOTIFICACIÓN (OBLIGATORIA)
Al terminar, guardá un nodo Notification en HelixDB:
`save_to_helix(label="Notification", properties={
  agent: "zyro-sdd-tasks",
  task_id: "<task-id>",
  summary: "Resumen breve de lo que se completó",
  read: false
})`
