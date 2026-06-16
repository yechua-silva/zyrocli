# Tasks: Fase 2 — HelixDB Integration

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 680–780 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Foundation) → PR 2 (CRUD + Schema) → PR 3 (Search + CLI) |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Foundation: dependency + errors + client wrapper + schema | PR 1 | Base: `main`; ~260 lines; includes go.mod, errors.go, client.go, schema.go |
| 2 | Core: nodes CRUD + edges + traversals | PR 2 | Base: PR 1 branch; ~220 lines; nodes.go + edges.go; depends on PR 1 |
| 3 | Search + Tests + CLI commands | PR 3 | Base: PR 2 branch; ~300 lines; search.go + helix_test.go + db.go + absorb.go; depends on PR 2 |

## Phase 1: Foundation (Dependency + Error Types + Client Wrapper + Schema)

- [x] 1.1 Add HelixDB SDK dependency: run `go get github.com/helixdb/helix-db/sdks/go` to update `go.mod` and `go.sum`
- [x] 1.2 Create `internal/db/helix/errors.go` (~20 lines): define `ErrNotFound`, `ErrConnection`, `ErrUnauthorized`, `ErrTenantMismatch` using `errors.New()` with `helix:` prefix
- [x] 1.3 Create `internal/db/helix/client.go` (~120 lines): `Client` struct wrapping `*helixsdk.Client` + `tenantID string`; `NewClient(ctx, ...Option)` with 3x retry on failure; `Close()` idempotent; `Ping(ctx) bool`; functional options pattern (`WithBaseURL`, `WithTenantID`)
- [x] 1.4 Create `internal/db/helix/schema.go` (~80 lines): `InitSchema()` builds and executes `WriteQuery("create_schema_indexes")` with 6 equality (2 unique: Developer.name, Skill.name), 3 vector 1536-d (Pattern, Document, Skill scoped by tenant_id), 1 text BM25 (Document.content), 1 range (Task.created_at); all via `CreateIndexIfNotExists` — idempotent
- [x] 1.5 Verify: `go build ./internal/db/helix/...` compiles without error

## Phase 2: Node CRUD + Edge Operations

- [x] 2.1 Create `internal/db/helix/nodes.go` (~100 lines): define `Node` struct (`ID int64`, `Label string`, `Props map[string]interface{}`); implement `CreateNode(ctx, label, props)` injecting `tenant_id` from wrapper, returning node ID; `GetNode(ctx, id)` with tenant verification; `UpdateNode(ctx, id, props)` with tenant check; `DeleteNode(ctx, id)` with tenant check; `FindNodes(ctx, label, filters)` adding `tenant_id` filter automatically
- [x] 2.2 Create `internal/db/helix/edges.go` (~60 lines): implement `CreateEdge(ctx, fromID, toID, label, props)` filtering by tenant; `GetOutgoing(ctx, nodeID, edgeLabel)` returning `[]*Node`; `GetIncoming(ctx, nodeID, edgeLabel)` returning `[]*Node`; `DeleteEdge(ctx, edgeID)` with tenant verification; all queries add `tenant_id` filter via wrapper
- [x] 2.3 Verify: `go build ./internal/db/helix/...` compiles; tenant injection logic testable via mock

## Phase 3: Semantic Search

- [x] 3.1 Create `internal/db/helix/search.go` (~60 lines): implement `VectorSearch(ctx, label, embedding []float32, k int)` returning `[]*Node` with similarity scores, tenant-scoped; `TextSearch(ctx, label, query string)` returning `[]*Node` via BM25, tenant-scoped; both use wrapper's `tenantID` for filter injection

## Phase 4: Testing

- [x] 4.1 Create `internal/db/helix/helix_test.go` (~200 lines): unit tests with real SDK client (no mock needed — `NewClient` is connection-free); test `NewClient` default URL/options/invalid URL; test `InjectTenant` with/without tenant_id; test `Ping` returns false when unreachable; test `buildSchemaIndexes` JSON structure; test `InitSchema` nil client guard; test `NodeFromResult`; test all error sentinels; test `Close` idempotency
- [x] 4.2 Verify: `go test ./internal/db/helix/...` passes (15 tests, 5ms, no HelixDB required)

## Phase 5: CLI Commands

- [x] 5.1 Create `cmd/zyrocli/db.go` (~80 lines): cobra command `db` with subcommands `init` (calls `CreateSchemaIndexes`, accepts `--url` default `http://localhost:6969`, exit 1 on unreachable), `status` (pings HelixDB, prints connected/unreachable), `reset` (deletes all data, prompts confirmation, `--force` bypasses prompt); register in root command
- [x] 5.2 Create `cmd/zyrocli/absorb.go` (~100 lines): cobra command `absorb`; globs `.docs/` for `.md`, `.yaml`, `.json`, `.txt`; skips `.git`, `node_modules`, binaries; infers `doc_type` from content keywords (design, spec, adr, handoff, decision); upserts Document nodes by `topic_key: "docs/{filename}"` idempotently; accepts `--url` flag; prints summary (created/updated/skipped counts)
- [x] 5.3 Verify: `go build ./cmd/zyrocli/...` compiles; `zyrocli db --help` and `zyrocli absorb --help` show correct help text; existing `zyrocli run --help` unchanged

## Phase 6: Integration Verification

- [x] 6.1 Run full test suite: `go test ./...` — no regressions in Phase 1 code
- [ ] 6.2 Manual smoke test (requires HelixDB running): `zyrocli db init` exits 0; `zyrocli db status` prints "connected"; `zyrocli absorb` creates Document nodes from `.docs/`
- [ ] 6.3 Verify rollback: `git checkout HEAD -- internal/db/ cmd/zyrocli/db.go cmd/zyrocli/absorb.go go.mod go.sum` + `go build ./...` + `go test ./...` all pass
