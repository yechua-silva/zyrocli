---
name: zyro-skills-find
description: "Fase 0: descubre skills en skills.sh y guarda en HelixDB"
---
# F0: Descubrimiento de Skills

## ⚠️ REGLAS
- **PASO 1 OBLIGATORIO: Ejecuta `npx skills find <lenguaje>`.** No uses skills ya instaladas.
- **NO instales skills.** Eso es trabajo de zyro-skills-apply con aprobación humana.
- **Cada skill descubierta DEBE guardarse en HelixDB.**
- Si el lenguaje está vacío en handoff.yaml, detectalo del proyecto: package.json, go.mod, etc.

## ACCIÓN
1. Detecta lenguaje: handoff.yaml o archivos del proyecto
2. Ejecuta: `npx skills find <lenguaje>`
3. Para CADA skill del resultado:
   `save_to_helix(label="Skill", properties={
     name: string,
     language: string,
     stars: int,
     description: string,
     source_url: string
   })`
4. Reporta IDs de skills creadas al orquestador

## NO HACER
No instalar. No validar audits. Solo descubrir y guardar en HelixDB.

## NOTIFICACIÓN (OBLIGATORIA)
Al terminar, guardá un nodo Notification en HelixDB:
`save_to_helix(label="Notification", properties={
  agent: "zyro-skills-find",
  task_id: "<task-id>",
  summary: "Resumen breve de lo que se completó",
  read: false
})`
