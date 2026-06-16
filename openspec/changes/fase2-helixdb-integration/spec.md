# helixdb-core Specification

**Purpose**: HelixDB access layer with tenant injection, idempotent schema, CRUD, traversals, semantic search.

## Requirements

### Requirement: InitSchema
`CreateSchemaIndexes()` MUST create all indexes idempotently: 9 equality (3 unique), 3 vector 1536-d (tenant-scoped), 1 text BM25, 1 range. Repeated calls MUST NOT error.

#### Scenario: Idempotent - GIVEN indexes exist, WHEN called twice, THEN second call returns no error

#### Scenario: Fresh init - GIVEN no indexes, WHEN called, THEN all 14 indexes created

### Requirement: Node CRUD
CreateNode MUST inject `tenant_id`. GetNode/UpdateNode/DeleteNode MUST verify tenant_id.

#### Scenario: Create returns ID > 0 - GIVEN valid props, WHEN CreateNode, THEN ID > 0

#### Scenario: Tenant injection - GIVEN client with tenant "proj-alpha", WHEN CreateNode, THEN node has `tenant_id: "proj-alpha"`

#### Scenario: Tenant isolation - GIVEN node from tenant A, WHEN tenant B calls GetNode, THEN error (not found)

### Requirement: Traversals
GetOutgoing, GetIncoming, CreateEdge MUST filter by tenant_id.

#### Scenario: Outgoing edges - GIVEN edges labeled "depends_on" N1→N2, WHEN GetOutgoing(N1, "depends_on"), THEN N2 returned

#### Scenario: Cross-tenant isolation - GIVEN edge from tenant A, WHEN tenant B queries, THEN edge invisible

### Requirement: Semantic Search
VectorSearch (ANN cosine) and TextSearch (BM25) MUST be tenant-scoped.

#### Scenario: Vector search ranked - GIVEN nodes with embeddings, WHEN VectorSearch("Pattern", emb, 5), THEN ≤5 results with scores

#### Scenario: Text search scoped - GIVEN same content for tenants A and B, WHEN TextSearch for A, THEN only A matches

### Requirement: Client Lifecycle
NewClient MUST retry 3 times on failure. Close MUST be idempotent. Ping MUST return false when DB unreachable.

#### Scenario: Retry on failure - GIVEN HelixDB down, WHEN NewClient, THEN retries 3 times then error

#### Scenario: Ping false - GIVEN HelixDB not running, THEN Ping() returns false

#### Scenario: Idempotent close - GIVEN connected client, WHEN Close() called twice, THEN second call no panic

---

# zyrocli-db Specification

**Purpose**: `zyrocli db` subcommands for HelixDB administration.

## Requirements

### Requirement: db init
`zyrocli db init` MUST call CreateSchemaIndexes(), accept `--url` (default http://localhost:6969). If unreachable, print error and exit 1.

#### Scenario: Succeeds - GIVEN HelixDB running, WHEN `zyrocli db init`, THEN exit 0

#### Scenario: Unreachable - GIVEN HelixDB not running, THEN clear error and exit 1

### Requirement: db status
MUST ping HelixDB and report connected/unreachable.

#### Scenario: Connected - GIVEN DB running, THEN output "connected", exit 0

#### Scenario: Down - GIVEN DB not running, THEN output "unreachable", exit 1

### Requirement: db reset
MUST delete all data. MUST prompt confirmation. --force bypasses prompt.

#### Scenario: Confirmed - GIVEN prompt, WHEN user answers "y", THEN data deleted

#### Scenario: Cancelled - GIVEN prompt, WHEN user answers "n", THEN exit 0, no deletion

---

# Delta for doc-tools

## ADDED Requirements

### Requirement: Absorb Documents
`zyrocli absorb` MUST glob `.docs/` (.md, .yaml, .json, .txt), infer `doc_type` from content, upsert Document nodes by `topic_key: "docs/{filename}"`. Binaries, .git, node_modules MUST be skipped.

#### Scenario: Creates nodes - GIVEN 3 .md files, WHEN `zyrocli absorb`, THEN 3 Document nodes created

#### Scenario: Idempotent - GIVEN file absorbed before, WHEN re-run, THEN existing node updated, not duplicated

#### Scenario: Skip binary - GIVEN .md and .png, WHEN absorb, THEN only .md creates a node

#### Scenario: Doc type inference - GIVEN file with "## Design" in content, THEN doc_type = "design"

---

# Delta for zyrocli-run

## ADDED Requirements

### Requirement: Register root-level subcommands
`zyrocli db` and `zyrocli absorb` MUST be registered as cobra root commands. Existing `zyrocli run` MUST be unchanged.

#### Scenario: db registered - GIVEN binary, WHEN `zyrocli db --help`, THEN db help shown

#### Scenario: absorb registered - GIVEN binary, WHEN `zyrocli absorb --help`, THEN absorb help shown

#### Scenario: run unchanged - GIVEN binary, WHEN `zyrocli run --help`, THEN output unchanged (scaffold+opencode)
