# Delta for Handoff Parser

## ADDED Requirements

### Requirement: Capabilities Field in Payload

The `Payload` struct MUST gain a `Capabilities []string` field for C-I-O traceability, representing project capabilities (e.g., testing, deployment, monitoring).

#### Scenario: Capabilities parsed from YAML
- GIVEN a handoff YAML with `capabilities: ["testing", "deployment"]`
- WHEN unmarshalled into `Payload`
- THEN `Capabilities` contains 2 entries: `"testing"` and `"deployment"`

#### Scenario: Capabilities optional (absent)
- GIVEN a handoff YAML without a `capabilities` key
- WHEN unmarshalled into `Payload`
- THEN `Capabilities` is nil (zero value)

### Requirement: Dependencies Field in Payload

The `Payload` struct MUST gain a `Dependencies []string` field for C-I-O traceability, representing external project dependencies (e.g., auth, payments, storage).

#### Scenario: Dependencies parsed from YAML
- GIVEN a handoff YAML with `dependencies: ["auth", "payments"]`
- WHEN unmarshalled into `Payload`
- THEN `Dependencies` contains 2 entries: `"auth"` and `"payments"`

#### Scenario: Dependencies optional (absent)
- GIVEN a handoff YAML without a `dependencies` key
- WHEN unmarshalled into `Payload`
- THEN `Dependencies` is nil (zero value)

## MODIFIED Requirements

### Requirement: Handoff Structs Extended

The `internal/handoff/payload.go` payload struct MUST represent all sections of handoff.yaml v2.0 including the new `Capabilities []string` and `Dependencies []string` fields. Each new field MUST carry a `yaml:"...,omitempty"` tag.
(Previously: capabilities and dependencies were not present)

#### Scenario: Capabilities and dependencies round-trip
- GIVEN a handoff.yaml v2.0 with capabilities and dependencies
- WHEN `yaml.Unmarshal` reads the content
- THEN `Payload.Capabilities` and `Payload.Dependencies` are populated
- WHEN `yaml.Marshal` writes the content
- THEN the output includes the capabilities and dependencies

### Requirement: Test Fixtures Extended

The `validYAMLv20` test fixture MUST include capabilities (testing, deployment, monitoring) and dependencies (auth, payments, storage) to verify parsing. The `requiredOnlyYAML` fixture MUST verify zero-value for both slices.
(Previously: fixtures did not include these fields)

#### Scenario: Full fixture parsed correctly
- GIVEN the `validYAMLv20` fixture with capabilities and dependencies
- WHEN `Parse` runs
- THEN `Capabilities` has 3 entries and `Dependencies` has 3 entries

#### Scenario: Minimal fixture has zero values
- GIVEN the `requiredOnlyYAML` fixture without capabilities or dependencies
- WHEN `Parse` runs
- THEN both `Capabilities` and `Dependencies` are empty/nil
