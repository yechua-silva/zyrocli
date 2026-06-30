---
name: to-issues
description: Generate GitHub Issues from PRD specs and task definitions for external visibility
disable-model-invocation: true
---

## ⚠️ REGLAS
- **NO modifiques código del proyecto.** Solo creás Issues en GitHub.
- **NO corras bash excepto para gh (GitHub CLI) o curl.**
- **NO guardes nodos en HelixDB excepto la Notification obligatoria.**
- **NO edites el PRD ni las task definitions.** Solo leélos.
- **Una Issue por user story o tarea.** No combines.

# To Issues

Generate well-structured GitHub Issues from PRD specs and task definitions.

## When to use
- When you need external visibility into the development process
- When stakeholders need to track progress on GitHub
- When you want vertical-slice issues that cross-cut technical boundaries

## Process
1. Read the PRD from `openspec/specs/<feature>/spec.md`
2. Read the task definitions from HelixDB (Task nodes)
3. For each task/user story, create a GitHub Issue with:
   - Title: Feature/component name
   - Description: User story in Given/When/Then format
   - Labels: phase, component, priority
   - Assignee: (optional)
   - Milestone: (optional)
4. Link related issues as dependencies
5. Apply label `ready-for-dev` when the issue is actionable

## Output
GitHub Issues created via GitHub API or CLI.

## NOTIFICACIÓN (OBLIGATORIA)
Al terminar, guardá un nodo Notification en HelixDB:
`save_to_helix(label="Notification", properties={
  agent: "to-issues",
  task_id: "<task-id>",
  summary: "Resumen de issues creados",
  read: false
})`
