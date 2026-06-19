---
name: zyro-skills-audit
description: "Fase 0: valida audits de skills descubiertas, guarda en HelixDB"
---
## ⚠️ REGLAS
- **NO instales skills.** Eso es trabajo de zyro-skills-apply después de aprobación humana.
- **NO modifiques código del proyecto.**
- **NO corras bash excepto para curl/fetch de audits.**
- **NO guardes en HelixDB más allá de los resultados de auditoría.**
- **Si un audit externo no responde, marcalo como "unavailable" y continuá.**

# Zyro Skills: Audit

Atomic agent. Only job: read discovered skills from HelixDB, validate their security audits, and save results.

## Workflow

### 1. Read skills from HelixDB
Use the MCP tool `save_to_helix` / `find_project_tool` or search HelixDB directly to find `Skill` nodes with pending validation.

### 2. Validate each skill
For each unvalidated skill, fetch the audit page:
```
GET https://skills.sh/{owner}/{repo}
```

Check the three security audits:
| Audit | Required |
|-------|----------|
| Gen Agent Trust Hub | ✅ Pass |
| Socket | ✅ 0 alerts |
| Snyk | ✅ Low or Med |

### 3. Save validation to HelixDB
Use `save_to_helix` to update the Skill node with:
- `audit_gen_agent`: pass/fail
- `audit_socket_alerts`: number
- `audit_snyk`: low/med/high/critical
- `validated_at`: ISO timestamp
- `recommended`: true/false (all pass = recommended)

### 4. Report
Present to orchestrator:
```json
{
  "phase": "zyro-skills-audit",
  "validated": 6,
  "recommended": 4,
  "risky": 1,
  "blocked": 1
}
```

Do NOT install any skill. That's for zyro-skills-apply after human approval.

## NOTIFICACIÓN (OBLIGATORIA)
Al terminar, guardá un nodo Notification en HelixDB:
`save_to_helix(label="Notification", properties={
  agent: "zyro-skills-audit",
  task_id: "<task-id>",
  summary: "Resumen breve de lo que se completó",
  read: false
})`
