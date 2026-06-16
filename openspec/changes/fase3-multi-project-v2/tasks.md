# Tasks: Fase 3 — Multi-Project v2

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 550–650 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 → PR 3 → PR 4 (stacked-to-main) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Skills cross-project: schema + queries + search | PR 1 | ~50 ln, base=main, tests included |
| 2 | CodeNode AST summaries + upsert | PR 2 | ~200 ln, base=main after PR1 merges |
| 3 | Task→CodeNode graph + git diff wrapper | PR 3 | ~150 ln, base=main after PR2 merges |
| 4 | zyrocli context command + formatters | PR 4 | ~200 ln, base=main after PR3 merges |

## Phase 1: PR 1 — Skills Cross-Project

- [ ] 1.1 Modify `internal/db/helix/schema.go` line 68: remove `"project_id"` partition arg from `idx_skill_emb` vector index
- [ ] 1.2 Add `FindSharedSkills(ctx, label string) ([]*Node, error)` to `internal/db/helix/nodes.go` — queries Skill nodes WITHOUT project_id filter
- [ ] 1.3 Add `LinkSkillToProject(ctx, skillID, projectID int64, level string) (int64, error)` to `internal/db/helix/nodes.go` — creates REQUIRES_SKILL edge with required_level property
- [ ] 1.4 Add `GetProjectSkills(ctx, projectID int64) ([]*Node, error)` to `internal/db/helix/nodes.go` — traverses REQUIRES_SKILL from Project
- [ ] 1.5 Add `VectorSearchGlobal(ctx, label string, embedding []float32, k int) ([]*Node, error)` to `internal/db/helix/search.go` — ANN search WITHOUT project_id partition
- [ ] 1.6 Add tests in `internal/db/helix/helix_test.go`: TestFindSharedSkills, TestVectorSearchGlobal, TestLinkSkillToProject — mock httptest pattern
- [ ] 1.7 Verify: `go build ./...` and `go test ./internal/db/helix/...` pass

## Phase 2: PR 2 — CodeNode AST Summaries

- [x] 2.1 Create `internal/codeparse/go_ast.go`: ParseFile(path) → ParseResult, ParseDir(dir) → []ParseResult using go/ast + go/parser
- [x] 2.2 Define types in `internal/codeparse/go_ast.go`: ParseResult, FunctionInfo, TypeInfo, ImportInfo structs per design.md interfaces
- [x] 2.3 Create `internal/codeparse/detector.go`: DetectLanguage(filePath string) returns "go"|"unknown" by extension
- [x] 2.4 Create `internal/codeparse/summary.go`: GenerateSummary(result *ParseResult) string — template-based textual summary
- [x] 2.5 Add `UpsertCodeNode(ctx, projectID uint64, path, name, summary, hash string, imports []string) (int64, bool, error)` to `internal/db/helix/nodes.go` — custom query by (project_id, path) + hash check → create/update + HAS_CODENODE edge
- [x] 2.6 Add `findCodeNodeByPath(ctx, projectID uint64, path string) (*Node, error)` as private helper in `internal/db/helix/nodes.go` (per design.md — lowercase, used internally by UpsertCodeNode)
- [x] 2.7 CodeNode equality index `idx_codenode_path` already exists in `internal/db/helix/schema.go` — verified, no project_id partition needed
- [x] 2.8 Create `internal/codeparse/codeparse_test.go`: 20 table-driven tests covering ParseFile, ParseDir, GenerateSummary, DetectLanguage, edge cases
- [x] 2.9 Add tests in `internal/db/helix/helix_test.go`: TestUpsertCodeNode (Create, Update, NoChange), TestGetCodeNodesByProject (full + empty)
- [x] 2.10 Verified: `go build ./...` and `go test ./internal/codeparse/... ./internal/db/helix/...` pass (all 15 packages green)

## Phase 3: PR 3 — Task → CodeNode Graph

- [x] 3.1 Create `internal/git/diff.go`: ChangedFile struct (Path, Status, OldPath), ChangedFiles(ref string) ([]ChangedFile, error) wrapping `git diff --name-status`
- [x] 3.2 Create `internal/git/diff_test.go`: tests using temp git repos (init, commit files, modify, assert diff output)
- [x] 3.3 Add `LinkTaskToCodeNodes(ctx, taskID uint64, changedFiles []ChangedFile) (int, error)` to `internal/db/helix/nodes.go` — for each file: UpsertCodeNode (minimal if missing) + CreateEdge REFERENCES
- [x] 3.4 Create `cmd/zyrocli/task.go`: Cobra subcommands `task create`, `task link [task-id] --ref HEAD~1`, `task list`
- [x] 3.5 Add tests in `internal/db/helix/helix_test.go`: TestLinkTaskToCodeNodes
- [x] 3.6 Verify: `go build ./...` and `go test ./internal/git/... ./internal/db/helix/...` pass

## Phase 4: PR 4 — zyrocli context

- [x] 4.1 Create `internal/taskcontext/types.go`: TaskContext struct (Skills, CodeNodes, Documents, Patterns []ContextItem), ContextItem struct (Name, Summary, Type)
- [x] 4.2 Create `internal/taskcontext/queries.go`: GetTaskContext(ctx, client, taskID) → *TaskContext — traverses REQUIRES→Skill, REFERENCES→CodeNode, via Project: HAS_DOC→Document, HAS_PATTERN→Pattern
- [x] 4.3 Create `internal/taskcontext/formatter.go`: FormatText(ctx TaskContext) string, FormatJSON(ctx TaskContext) string, FormatPrompt(ctx TaskContext) string
- [x] 4.4 Create `cmd/zyrocli/context.go`: Cobra command `context [task-id] --format=text|json|prompt`, resolves taskID, calls GetTaskContext, formats output
- [x] 4.5 Create `internal/taskcontext/context_test.go`: table-driven tests for all 3 formatters (empty context, full context, single item)
- [x] 4.6 Verify: `go build ./...` and `go test ./internal/taskcontext/...` pass

## Phase 5: Integration Verification

- [x] 5.1 Full build: `go build ./...` — all 4 PRs merged
- [x] 5.2 Full test suite: `go test ./...` — all packages green
- [ ] 5.3 Smoke test: `zyrocli task create "test task"`, `zyrocli task list`, `zyrocli context <task-id> --format=text`
