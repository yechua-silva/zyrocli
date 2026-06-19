---
name: zyro-sdd-archive
description: "Archiva cambios completados y cierra el ciclo del proyecto"
---
# F4: Cierre y Archive

## ⚠️ REGLAS
- **NO modifiques código.** Solo cerrás el ciclo del proyecto.
- **NO borres archivos del proyecto.** Solo temporales de .zyro/.
- **NO corras bash excepto para limpiar archivos temporales.**

## ACCIÓN
1. Verificá que todas las tareas estén completadas en HelixDB
2. Marcá el proyecto como archivado: `update Project.status = "archived"`
3. Limpiá archivos temporales de `.zyro/` (task.yaml, result.yaml)
4. Guardá el registro de cierre: `save_to_helix(label="Archive", properties={project_id, completed_at, summary})`

## OUTPUT
`save_to_helix(label="Archive", properties={
  project_id: int,
  tasks_completed: int,
  summary: string,
  status: "archived"
})`

## NOTIFICACIÓN (OBLIGATORIA)
Al terminar, guardá un nodo Notification en HelixDB:
`save_to_helix(label="Notification", properties={
  agent: "zyro-sdd-archive",
  task_id: "<task-id>",
  summary: "Resumen breve de lo que se completó",
  read: false
})`
