# Design: Fase 3 — Multi-Project v2

## Technical Approach

4 PRs stacked-to-main, each autonomous. Paso 1 (tenant→project rename) is already complete. This design covers Pasos 2–5: cross-project Skill sharing, CodeNode AST summaries, Task→CodeNode graph, and `zyrocli context`.

**Naming collision resolved**: `internal/context/` already exists (Context7 MCP bridge). New task-context queries go in `internal/taskcontext/` to avoid conflicts.

## Architecture Decisions

### Decision: AST Go via stdlib `go/ast` + `go/parser`

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `go/ast` stdlib | Only Go; no TS/Python. Fast, zero deps. | ✅ Go MVP |
| `treesitter` bindings | Multi-lang. CGo, heavy deps, complex build. | ❌ Overkill for MVP |
| Regex + filename | Fast but fragile, misses semantics. | ❌ Rejected |

**Rationale**: Proposal scopes non-Go parsing as out-of-scope. stdlib gives exact type/function/import extraction with no dependency risk.

### Decision: Package naming — `internal/taskcontext/` not `internal/context/`

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Extend `internal/context/` | No new pkg. But mixes Context7 MCP bridge with HelixDB queries. | ❌ Mixed concerns |
| `internal/taskcontext/` | Clear separation. `context` cmd imports this. | ✅ Chosen |

**Rationale**: `internal/context/bridge.go` is a Context7 JSON-RPC process manager. Task context queries are HelixDB graph traversals — fundamentally different. Renaming the existing package is unnecessary churn.

### Decision: UpsertCodeNode by (project_id, path) with hash check

| Option | Tradeoff | Decision |
|--------|----------|----------|
| FindNodes + compare | N queries. Slow for large codebases. | ❌ |
| Unique index (project_id, path) | HelixDB doesn't support composite unique. | ❌ N/A |
| FindNodes by path + client-side filter | 1 query + filter. Acceptable for single-developer scale. | ✅ Chosen |

**Rationale**: Single-developer context means <1000 CodeNodes per project. One query + linear scan is fine. No need for composite indexes.

### Decision: `internal/git/` for diff wrapper

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `os/exec` inline in cmd | No reuse, hard to test. | ❌ |
| `internal/git/diff.go` | Testable, reusable by `task link` and future commands. | ✅ Chosen |

## Data Flow

```
zyrocli task link [task-id] --ref HEAD~1
    │
    ├── internal/git/diff.go
    │   └── exec: git diff --name-status HEAD~1
    │       → []ChangedFile{Path, Status, OldPath}
    │
    ├── internal/db/helix/nodes.go
    │   ├── UpsertCodeNode(projectID, path, name, summary, hash)
    │   │   ├── FindNodes("CodeNode", {path}) → check hash
    │   │   ├── CreateNode("CodeNode", props) if new
    │   │   └── UpdateNode(id, {summary, hash}) if changed
    │   │
    │   └── LinkTaskToCodeNodes(taskID, changedFiles)
    │       ├── For each file: UpsertCodeNode (minimal if missing)
    │       └── CreateEdge(taskID, codeNodeID, "REFERENCES")
    │
    └── Output: "Task #12 references 2 CodeNodes"

zyrocli context [task-id] --format=text
    │
    ├── internal/taskcontext/helix_query.go
    │   └── GetTaskContext(taskID):
    │       ├── Task.REQUIRES → []Skill
    │       ├── Task.REFERENCES → []CodeNode
    │       ├── Project.HAS_DOC → []Document
    │       └── Project.HAS_PATTERN → []Pattern
    │
    └── internal/taskcontext/formatter.go
        └── FormatText/FormatJSON/FormatPrompt(TaskContext)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/codeparse/go_ast.go` | Create | `ParseFile()`, `ParseDir()`, `GenerateSummary()` — AST Go analysis |
| `internal/codeparse/go_ast_test.go` | Create | Table-driven tests with fixture `.go` files |
| `internal/git/diff.go` | Create | `ChangedFiles(ref)` — `git diff --name-status` wrapper |
| `internal/git/diff_test.go` | Create | Tests with temp git repos |
| `internal/taskcontext/helix_query.go` | Create | `GetTaskContext()` — graph traversal queries |
| `internal/taskcontext/types.go` | Create | `TaskContext`, `ContextItem` structs |
| `internal/taskcontext/formatter.go` | Create | `FormatText()`, `FormatJSON()`, `FormatPrompt()` |
| `internal/taskcontext/formatter_test.go` | Create | Tests for all 3 formatters |
| `internal/db/helix/nodes.go` | Modify | +`FindSharedSkills()`, +`UpsertCodeNode()`, +`LinkTaskToCodeNodes()` |
| `internal/db/helix/search.go` | Modify | +`VectorSearchGlobal()` — search without project_id filter |
| `internal/db/helix/schema.go` | Modify | `idx_skill_emb`: remove `"project_id"` partition arg |
| `cmd/zyrocli/task.go` | Create | `task create`, `task link`, `task list` subcommands |
| `cmd/zyrocli/context.go` | Create | `context [task-id] --format=text|json|prompt` command |

## Interfaces / Contracts

```go
// --- internal/codeparse ---

type ParseResult struct {
    Package   string
    FilePath  string
    Functions []FunctionInfo
    Types     []TypeInfo
    Imports   []ImportInfo
}

type FunctionInfo struct {
    Name, Receiver, DocComment string
    Params, Returns            []string
    Exported                   bool
}

func ParseFile(path string) (*ParseResult, error)
func ParseDir(dir string) ([]*ParseResult, error)
func GenerateSummary(result *ParseResult) string

// --- internal/git ---

type ChangedFile struct {
    Path, Status, OldPath string
}

func ChangedFiles(ref string) ([]ChangedFile, error)

// --- internal/db/helix (new methods) ---

func (c *Client) FindSharedSkills(ctx context.Context, label string) ([]*Node, error)
func (c *Client) VectorSearchGlobal(ctx context.Context, label string, embedding []float32, k int) ([]*Node, error)
func (c *Client) UpsertCodeNode(ctx context.Context, projectID uint64, path, name, summary, hash string, imports []string) (int64, error)
func (c *Client) LinkTaskToCodeNodes(ctx context.Context, taskID uint64, changedFiles []ChangedFile) (int, error)

// --- internal/taskcontext ---

type TaskContext struct {
    Skills, CodeNodes, Documents, Patterns []ContextItem
}

type ContextItem struct {
    Name, Summary, Type string
}

func GetTaskContext(ctx context.Context, client *helix.Client, taskID int64) (*TaskContext, error)
func FormatText(ctx TaskContext) string
func FormatJSON(ctx TaskContext) string
func FormatPrompt(ctx TaskContext) string
```

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | `codeparse.ParseFile` | Table-driven: valid `.go`, syntax errors, anonymous funcs, no exports. Fixture files in `testdata/`. |
| Unit | `codeparse.GenerateSummary` | Golden file tests comparing output string. |
| Unit | `git.ChangedFiles` | Temp git repos via `os.Executable` — init, commit, modify, assert diff output. |
| Unit | `taskcontext` formatters | Table-driven: empty context, full context, single item. |
| Unit | `helix.FindSharedSkills`, `VectorSearchGlobal` | Mock `httptest.NewServer` (same pattern as existing `helix_test.go`). |
| Unit | `helix.UpsertCodeNode` | Mock server: 2 calls (find + create/update). Assert hash comparison logic. |
| Integration | `task link` end-to-end | Temp git repo + mock HelixDB server. Create task, link, assert edges created. |
| E2E | `zyrocli context` | Requires running HelixDB. `go test -tags=integration`. |

## Migration / Rollout

**Schema change (PR1)**: `idx_skill_emb` loses `project_id` partition. HelixDB schemaless — `InitSchema()` is idempotent. After deploy, run `zyrocli db init` to recreate the index. Existing Skill nodes lose nothing (properties are untouched, only the vector index changes).

**No data migration required**: Skill nodes don't have `project_id` in the new model. The index rebuild is automatic via `InitSchema()`.

## Open Questions

- [ ] Should `zyrocli task create` accept `--phase` and `--status` flags, or just the description?
- [ ] Should `LinkTaskToCodeNodes` create minimal CodeNodes (path+name only) or full AST-parse on the spot?
- [ ] Should `context` command default to `--format=text` or `--format=prompt` for subagent use?
