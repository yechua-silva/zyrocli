# Delta for CIO DSL Compile

## MODIFIED Requirements

### Requirement: CIO Compile — Engram Key Emission

`Compile(cio *CIO, changeName string)` MUST translate a CIO struct into Engram-compatible entries. The output MUST be `[]EngramEntry` where each entry has a `TopicKey` (format: `sdd/{change}/cio-{component}`) and markdown `Content`. The function MUST NOT emit OpenAPI or protobuf.

#### Scenario: Full CIO compiles to Engram entry
- GIVEN a CIO with Contract.Name="auth-model" and 2 IOMethods
- WHEN `Compile(cio, "scheduler-harness")` is called
- THEN one `EngramEntry` is returned with TopicKey `sdd/scheduler-harness/cio-auth-model` and Content containing markdown serialization of all populated fields

#### Scenario: Nil CIO returns error
- GIVEN a nil CIO pointer
- WHEN `Compile(nil, "change")` is called
- THEN an error is returned wrapping "nil CIO"

#### Scenario: Empty contract name uses "unnamed" fallback
- GIVEN a CIO with empty Contract.Name
- WHEN `Compile(cio, "change")` is called
- THEN the topic key uses "unnamed" as component name

### Requirement: Markdown Serialization

`CIO.ToMarkdown()` MUST produce a markdown document from the CIO struct. Each non-zero section produces a corresponding markdown heading. Zero-value fields MUST be omitted. The method MUST NOT panic on a zero-value CIO.

#### Scenario: All sections serialized
- GIVEN a CIO with all 6 sections populated
- WHEN `ToMarkdown()` is called
- THEN the output contains headings for Interface, Behavior, Constraints, Operation, and Testing

#### Scenario: Zero-value safety
- GIVEN a `CIO{}` with zero values
- WHEN `ToMarkdown()` is called
- THEN no panic occurs and a non-empty string is returned

### Requirement: Stable Topic Key Generation

`GenerateTopicKey(cio *CIO, changeName string)` MUST produce a deterministic topic key in the format `sdd/{change}/cio-{component}`. The same CIO MUST always produce the same key.

#### Scenario: Stable key
- GIVEN a CIO with Contract.Name="auth-model"
- WHEN `GenerateTopicKey(cio, "change")` is called twice
- THEN both calls return `sdd/change/cio-auth-model`

### Requirement: `EngramEntry` Output Struct

The `EngramEntry` struct MUST be exported and contain `TopicKey string` and `Content string` fields. This struct is the output contract for `Compile`.

#### Scenario: Struct fields accessible
- GIVEN an `EngramEntry{TopicKey: "k", Content: "v"}`
- WHEN fields are accessed
- THEN `TopicKey` returns "k" and `Content` returns "v"
