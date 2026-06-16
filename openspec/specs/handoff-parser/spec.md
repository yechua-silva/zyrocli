# Handoff Parser Specification

## Purpose

Define the handoff.yaml v2.0 parser, validator, and CLI entry point. This capability is the contract between Holdin Admin and ZyroCLI — without it, the 4-phase SDD pipeline cannot start.

## Requirements

### Requirement: Handoff Structs

The `internal/handoff/payload.go` payload struct MUST represent all sections of the handoff.yaml v2.0 contract: `version`, `source`, `project`, `validated_idea`, `user_stories`, `mvp`, `governance`, `testing`, `limits`, including the new `Capabilities []string` and `Dependencies []string` fields for C-I-O traceability. Each field MUST carry a `yaml` tag matching the YAML key. Each new field MUST carry a `yaml:"...,omitempty"` tag.

#### Scenario: All sections round-trip
- GIVEN a handoff.yaml v2.0 with all sections populated
- WHEN `yaml.Unmarshal` reads the content into `Payload`
- THEN all fields are populated without data loss

#### Scenario: Zero-value optionals
- GIVEN a handoff.yaml with only required fields
- WHEN unmarshalled into `Payload`
- THEN optional fields are zero-valued (nil/empty string)

#### Scenario: Capabilities parsed from YAML
- GIVEN a handoff YAML with `capabilities: ["testing", "deployment"]`
- WHEN unmarshalled into `Payload`
- THEN `Capabilities` contains 2 entries: `"testing"` and `"deployment"`

#### Scenario: Dependencies parsed from YAML
- GIVEN a handoff YAML with `dependencies: ["auth", "payments"]`
- WHEN unmarshalled into `Payload`
- THEN `Dependencies` contains 2 entries: `"auth"` and `"payments"`

#### Scenario: Capabilities and dependencies optional (absent)
- GIVEN a handoff YAML without `capabilities` or `dependencies` keys
- WHEN unmarshalled into `Payload`
- THEN both `Capabilities` and `Dependencies` are nil (zero value)

### Requirement: YAML Parsing

`Parse(path string) (*Payload, error)` in `internal/handoff/parser.go` MUST read YAML from disk if path is a file, or from stdin if path is `"-"`. It MUST return an error wrapping the underlying read or unmarshal failure.

#### Scenario: File parse succeeds
- GIVEN a well-formed handoff.yaml at `testdata/valid.yaml`
- WHEN `Parse("testdata/valid.yaml")` is called
- THEN a `*Payload` is returned with nil error

#### Scenario: Stdin parse succeeds
- GIVEN valid YAML on stdin and path `"-"`
- WHEN `Parse("-")` is called
- THEN stdin is consumed and a `*Payload` is returned

#### Scenario: Missing file returns error
- GIVEN a non-existent path
- WHEN `Parse("/nonexistent.yaml")` is called
- THEN an error wrapping `os.ErrNotExist` is returned

#### Scenario: Invalid YAML syntax
- GIVEN a file with malformed YAML
- WHEN `Parse("bad.yaml")` is called
- THEN an error referencing the syntax issue is returned

### Requirement: Business Validation

`Validate(p *Payload) error` in `internal/handoff/validate.go` MUST enforce: `version` is required and MUST be `"2.0"`; `source.system` is required; `project.name` and `project.language` are required; `governance.mode` is required; `testing.strategy` is required. All other fields MAY be empty.

#### Scenario: Valid payload passes
- GIVEN a payload with all required fields and `version: "2.0"`
- WHEN `Validate(p)` is called
- THEN nil error is returned

#### Scenario: Wrong version rejected
- GIVEN `version` is `"1.0"`
- WHEN `Validate(p)` is called
- THEN the error states `version` MUST be `"2.0"`

#### Scenario: All missing fields reported
- GIVEN a payload missing `project.name`, `source.system`, and `governance.mode`
- WHEN `Validate(p)` is called
- THEN the error contains all 3 violations, not just the first

### Requirement: CLI Integration

The root Cobra command MUST register a subcommand `init` that accepts one positional argument (path or `"-"`). `zyrocli init <path>` MUST call `Parse` then `Validate`, printing a success message on success or the error on failure.

#### Scenario: Init with valid file
- GIVEN a valid `testdata/valid.yaml`
- WHEN `zyrocli init testdata/valid.yaml` runs
- THEN exit code is 0 and stdout contains "OK" or "success"

#### Scenario: Init with invalid file
- GIVEN a file with missing required fields
- WHEN `zyrocli init broken.yaml` runs
- THEN exit code is non-zero and stderr lists validation errors

### Requirement: Test Fixtures Extended

The `validYAMLv20` test fixture MUST include capabilities (testing, deployment, monitoring) and dependencies (auth, payments, storage) to verify parsing. The `requiredOnlyYAML` fixture MUST verify zero-value for both slices.

#### Scenario: Full fixture parsed correctly
- GIVEN the `validYAMLv20` fixture with capabilities and dependencies
- WHEN `Parse` runs
- THEN `Capabilities` has 3 entries and `Dependencies` has 3 entries

#### Scenario: Minimal fixture has zero values
- GIVEN the `requiredOnlyYAML` fixture without capabilities or dependencies
- WHEN `Parse` runs
- THEN both `Capabilities` and `Dependencies` are empty/nil

### Requirement: Error Handling

Parse errors MUST wrap the underlying cause clearly. Validate errors MUST accumulate ALL violations. Invalid YAML MUST produce a message referencing the syntax problem.

#### Scenario: Multi-error returns all failures
- GIVEN a payload with 3 missing required fields
- WHEN `Validate(p)` returns
- THEN the error string contains all 3 violations

### Requirement: Stdin Pipe

`zyrocli init -` MUST read YAML from stdin. The command MUST support `holdin validate . | zyrocli init -` as a working pipeline.

#### Scenario: Stdin pipeline succeeds
- GIVEN valid YAML piped to stdin
- WHEN `echo "version: \"2.0\"" | zyrocli init -` runs
- THEN exit code is 0

#### Scenario: Empty stdin produces error
- GIVEN stdin is closed immediately with no input
- WHEN `zyrocli init -` runs
- THEN a "empty input" or EOF error is returned
