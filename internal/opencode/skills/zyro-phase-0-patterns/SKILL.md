---
name: zyro-phase-0-patterns
description: "Fase 0: busca patrones similares en internet y guarda en HelixDB"
---
# F0: Patrones de Referencia

## ⚠️ REGLAS
- **Cada patrón encontrado DEBE guardarse en HelixDB.**
- **No vuelvas sin haber creado al menos 1 nodo Pattern en HelixDB.**
- Usa `save_to_helix` para cada patrón, no texto libre.
- No reportes solo al orquestador. El output real es HelixDB.

## ACCIÓN
1. Lee `handoff.yaml` para entender lenguaje, dominio, tipo de proyecto
2. Busca en internet proyectos similares con webfetch
3. Para CADA patrón encontrado:
   `save_to_helix(label="Pattern", properties={
     name: string,
     description: string,
     language: string,
     confidence: "alta"|"media"|"baja",
     source_url: string
   })`
4. Vincula al proyecto: `link_to_project(project_id, "Pattern", pattern_id, "HAS_PATTERN")`

## OUTPUT
Lista de IDs de patrones creados en HelixDB. Mínimo 1.

## NOTIFICACIÓN (OBLIGATORIA)
Al terminar, guardá un nodo Notification en HelixDB:
`save_to_helix(label="Notification", properties={
  agent: "zyro-phase-0-patterns",
  task_id: "<task-id>",
  summary: "Resumen breve de lo que se completó",
  read: false
})`
