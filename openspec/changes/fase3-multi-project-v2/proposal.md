# Proposal: Fase 3 — Multi-Project v2

## Intent

HelixDB almacena conocimiento de un solo developer con múltiples proyectos. Fase 3 completa el modelo: Skills compartidos entre proyectos (sin project_id), CodeNodes con summaries automáticos vía AST, trazabilidad Task → CodeNode vía git diff, y `zyrocli context [task]` para inyectar contexto preciso en subagentes.

## Scope

### In Scope (4 pasos, ~600 líneas)

| Paso | Descripción | Líneas |
|------|-------------|--------|
| 2 | Cross-project Skill sharing: quitar partición `project_id` de índices Skill, `FindSharedSkills()`, `VectorSearchGlobal()` | ~50 |
| 3 | CodeNode summaries: nuevo `internal/codeparse/` (AST Go → summary textual, upsert por project_id+path) | ~200 |
| 4 | Task → CodeNode graph: `zyrocli task` (create/link/list), `internal/git/diff.go`, edges REFERENCES automáticos | ~150 |
| 5 | `zyrocli context [task]` — queries HelixDB (Task→Skills, Task→CodeNodes, Project→Docs/Patterns), formateadores text/json/prompt | ~200 |

### Out of Scope
- Parseo de lenguajes no-Go (TypeScript, Python) — futuro
- LLM para summaries — template + AST es suficiente para MVP
- Community detection en HelixDB (no existe como feature)
- TUI/web para `zyrocli context`

## Capabilities

### New Capabilities
- `codeparse-ast`: AST Go → extraer funciones, tipos, imports; generar summary textual
- `helixdb-cross-project`: FindSharedSkills, VectorSearchGlobal, UpsertCodeNode, LinkTaskToCodeNodes
- `zyrocli-task`: Subcomandos `task create`, `task link`, `task list`
- `zyrocli-context`: Comando `context [task-id] --format=text|json|prompt`

### Modified Capabilities
- `helixdb-core`: Nuevos métodos en nodes.go/search.go; Skill sin project_id en schema
- `zyrocli-run`: Añade subcomandos `task`, `context`

## Approach

4 PRs encadenados a main (stacked-to-main):
```
main ← PR 1: Skills cross-project (~50 ln)
  ← PR 2: CodeNode summaries (~200 ln)
  ← PR 3: Task→CodeNode graph (~150 ln)
  ← PR 4: zyrocli context (~200 ln)
```
Cada PR es autónomo: PR1 refactoriza índices, PR2 introduce `internal/codeparse/`, PR3 agrega `internal/git/diff.go` + `cmd/zyrocli/task.go`, PR4 cierra con `cmd/zyrocli/context.go` + `internal/context/`.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/db/helix/schema.go` | Modified | Skill indexes: quitar `project_id` partition |
| `internal/db/helix/nodes.go` | Modified | +FindSharedSkills, UpsertCodeNode, LinkTaskToCodeNodes |
| `internal/db/helix/search.go` | Modified | +VectorSearchGlobal |
| `internal/codeparse/` | New | go_ast.go, detector.go, summary.go |
| `internal/git/diff.go` | New | git diff --name-only wrapper |
| `internal/context/` | New | helix_query.go, formatter.go |
| `cmd/zyrocli/task.go` | New | task subcommands |
| `cmd/zyrocli/context.go` | New | context subcommand |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| CodeNode summary quality (AST sin contexto) | Medium | Template + AST para MVP; expandir con LLM después |
| git diff no captura renames | Low | Usar `--name-status` para detectar renames |
| Embedding dim mismatch entre modelos | Low | Estandarizar 1536-dim, documentar modelo |

## Rollback Plan

Por PR: `git revert <merge-commit>` de cada PR individualmente. PR1 (Skills) es el único con riesgo de datos — requiere regenerar índices Skill tras revert. PRs 2-4 son puramente aditivos (nuevos paquetes/comandos) y se revierten sin side effects.

## Dependencies

- `go/ast`, `go/parser` (stdlib) — parseo AST
- `github.com/helixdb/helix-db/sdks/go` v0.1.1 — existente
- Paso 1 completo (`tenant_id` → `project_id` ya aplicado)

## Success Criteria

- [ ] `go build ./...` compila sin errores
- [ ] `go test ./internal/db/helix/...` pasa con HelixDB corriendo
- [ ] `go test ./internal/codeparse/...` pasa (parseo AST Go)
- [ ] `go test ./internal/git/...` pasa
- [ ] `zyrocli context task-42 --format=text` retorna contexto formateado
- [ ] `zyrocli task link` detecta archivos via git diff y crea edges REFERENCES
- [ ] Un nodo Skill puede ser referenciado por múltiples Projects
