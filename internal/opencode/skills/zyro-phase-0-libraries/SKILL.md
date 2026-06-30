---
name: zyro-phase-0-libraries
description: "Fase 0: investiga librerías con Context + GitMCP, guarda en HelixDB"
---
# F0: Descubrimiento de Librerías

## ⚠️ REGLAS
- **Cada librería encontrada DEBE guardarse en HelixDB.**
- **No vuelvas sin haber creado al menos 1 nodo Library en HelixDB.**
- Usa `save_to_helix` para cada librería, no texto libre.
- No reportes solo al orquestador. El output real es HelixDB.

## ACCIÓN
1. Detectá el stack del proyecto: package.json, go.mod, Cargo.toml, etc.
2. Investigá librerías recomendadas para ese stack (webfetch, Context MCP, GitMCP)
3. Para CADA librería:
   `save_to_helix(label="Library", properties={
     name: string,
     version: string,
     category: string,
     description: string,
     confidence: "alta"|"media"|"baja",
     docs_url: string
   })`
4. Vinculá al proyecto: `link_to_project(project_id, "Library", library_id, "USES_LIB")`

## OUTPUT
Lista de IDs de librerías creadas en HelixDB. Mínimo 1.

## NOTIFICACIÓN (OBLIGATORIA)
Al terminar, guardá un nodo Notification en HelixDB:
`save_to_helix(label="Notification", properties={
  agent: "zyro-phase-0-libraries",
  task_id: "<task-id>",
  summary: "Resumen breve de lo que se completó",
  read: false
})`
