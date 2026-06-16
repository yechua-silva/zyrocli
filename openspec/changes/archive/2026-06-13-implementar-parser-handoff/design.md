# Design: Implementar parser de handoff.yaml + comando init

## Technical Approach

Replace the incomplete v1 structs in `payload.go` with a full v2.0 contract aligned to AGENT.md. Add three new files: `parser.go` (YAML deserialization + stdin), `validate.go` (business rule enforcement), and `cmd/zyrocli/init.go` (Cobra subcommand). The parser does one thing (deserialize YAML). The validator does one thing (enforce required fields). They compose via `Parse → Validate` pipeline.

## Architecture Decisions

### Decision: Parse/Validate Separation

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Single `Load(path) (*Payload, error)` | Simpler API, but conflates deserialization with business rules | Rejected |
| Separate `Parse()` + `Validate()` | Two functions to test independently; clear responsibility boundary | **Chosen** |

**Rationale**: `Parse()` tests focus on YAML syntax (file I/O, malformed YAML, stdin). `Validate()` tests focus on business rules (missing fields, wrong version). Independent test surfaces, no mocking needed.

### Decision: Stdin via `"-"` Convention

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Separate `--stdin` flag | Explicit, but adds Cobra flag plumbing | Rejected |
| Path == `"-"` triggers os.Stdin | POSIX convention (curl, tar), zero extra deps | **Chosen** |

**Rationale**: `os.Stdin` is a `*os.File` implementing `io.Reader`. `io.ReadAll(os.Stdin)` works identically to `os.ReadFile(path)`. No special libraries needed.

### Decision: Multi-Error Validation with `errors.Join()`

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Fail-fast (return first error) | Simpler code, but user fixes one at a time | Rejected |
| `errors.Join()` collects all | User sees ALL missing fields at once, faster iteration | **Chosen** |

**Rationale**: Go 1.20+ provides `errors.Join()`. Returning all failures saves the user multiple round-trips. No custom error type needed.

### Decision: Struct Design — No Pointers, `omitempty`

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Pointer fields (`*string`) | Distinguish nil vs empty, but verbose nil-checks everywhere | Rejected |
| Value fields + `omitempty` yaml tags | Zero value = "not provided"; simpler code | **Chosen** |

**Rationale**: For this use case, empty string `""` and absent are semantically equivalent — both mean "user didn't provide this". Pointer indirection adds complexity with no benefit.

### Decision: Subcommand in Separate File

| Option | Tradeoff | Decision |
|--------|----------|----------|
| All commands in `main.go` | Single file, easy to find | Rejected |
| `cmd/zyrocli/init.go` with `init()` + `rootCmd.AddCommand()` | Cobra convention, scalable to many subcommands | **Chosen** |

**Rationale**: Existing `main.go` has only `rootCmd`. Adding `initCmd` there bloats the file. Cobra supports multi-file registration via package-level `init()` functions.

## Data Flow

```
Holdin Admin ──stdout──► handoff.yaml (pipe)
                           │
                    zyrocli init <path> o "-"
                           │
                    os.ReadFile(path) / io.ReadAll(os.Stdin)
                           │
                    yaml.Unmarshal ──► *Payload
                           │
                    Validate(payload) ──► errors.Join(...)
                           │
                    ├── len(errors) == 0 → Success message
                    └── len(errors) > 0  → Print all errors, exit 1
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/handoff/payload.go` | Modify | Replace v1 structs with v2.0: Payload with 8 sections (Source, Project, ValidatedIdea, UserStory, MVP, Governance, Testing, Limits) + Phase |
| `internal/handoff/parser.go` | Create | `Parse(path string) (*Payload, error)` — file or stdin via `"-"` |
| `internal/handoff/validate.go` | Create | `Validate(p *Payload) error` — required field checks, version == `"2.0"` |
| `internal/handoff/payload_test.go` | Create | 8+ table-driven cases covering parse, validate, and integration |
| `cmd/zyrocli/init.go` | Create | Cobra subcommand: `initCmd`, args validation, calls Parse → Validate → prints summary |
| `handoff.yaml` | Modify | Upgrade from v1.0 to v2.0 with all 8 sections |

## Interfaces / Contracts

```go
// internal/handoff/payload.go
type Payload struct {
    Version      string       `yaml:"version"`
    Source       Source       `yaml:"source"`
    Project      Project      `yaml:"project"`
    ValidatedIdea ValidatedIdea `yaml:"validated_idea"`
    UserStory    UserStory    `yaml:"user_story"`
    MVP          MVP          `yaml:"mvp"`
    Governance   Governance   `yaml:"governance"`
    Testing      Testing      `yaml:"testing"`
    Limits       Limits       `yaml:"limits"`
}

// internal/handoff/parser.go
func Parse(path string) (*Payload, error)

// internal/handoff/validate.go
func Validate(p *Payload) error
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit — parser | Valid file, stdin (`"-"`), malformed YAML, file not found | Table-driven with `t.TempDir()`, mock stdin via pipe |
| Unit — validator | All fields present, each required field missing one-by-one, version `"1.0"` rejected | Table-driven, assert `errors.Join` message contains expected field |
| Integration | Full YAML string → Parse → Validate → zero errors | Single table-driven case with complete v2.0 payload |

## Migration / Rollout

No migration required — greenfield feature. No existing data to transform.

## Open Questions

- None — all technical decisions resolved. Proposal provides clear contract.
