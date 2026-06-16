# Contract Testing Specification

## Purpose

Define the Given/When/Then contract testing framework that validates agent-spec contracts against implementation behavior, executes contracts sequentially via a ContractExecutor, and produces structured pass/fail reports with GraphifyDiff structural comparison data.

## Requirements

### Requirement: Contract Types

The test package MUST define a `Contract` type with `Given`, `When`, and `Then` function fields. `GivenFunc` MUST receive `context.Context` and return `(interface{}, error)`. `WhenFunc` MUST receive `(context.Context, interface{})` and return `(interface{}, error)`. `ThenFunc` MUST receive `(context.Context, interface{}, interface{})` and return `error`. A `ContractResult` MUST carry the name, passed boolean, and error string.

#### Scenario: Contract creation
- GIVEN a Contract with all three functions set
- WHEN the contract is created
- THEN it has Name, Given, When, and Then fields

### Requirement: ContractExecutor with Given/When/Then

The test package MUST provide a `ContractExecutor` type with an `Execute(ctx, Contract) ContractResult` method. The executor MUST run the three phases sequentially: Given → When → Then. Each phase MUST receive the output of the previous phase. If Given fails, When and Then MUST be skipped. If When fails, Then MUST be skipped.

#### Scenario: All phases succeed
- GIVEN a Contract where all three functions return success
- WHEN `Execute(ctx, contract)` is called
- THEN Passed is true

#### Scenario: Given fails
- GIVEN a Contract where Given returns an error
- WHEN `Execute(ctx, contract)` is called
- THEN Passed is false and the error contains "GIVEN failed"

#### Scenario: When fails
- GIVEN a Contract where When returns an error
- WHEN `Execute(ctx, contract)` is called
- THEN Passed is false and the error contains "WHEN failed"

#### Scenario: Then fails
- GIVEN a Contract where Then returns an error
- WHEN `Execute(ctx, contract)` is called
- THEN Passed is false and the error contains "THEN failed"

#### Scenario: Nil Then
- GIVEN a Contract with Then=nil
- WHEN `Execute(ctx, contract)` is called
- THEN the contract passes without verification

#### Scenario: Phase ordering
- GIVEN a Contract with order-tracking functions
- WHEN `Execute(ctx, contract)` is called
- THEN the phases execute in order: given, when, then

#### Scenario: State pass-through
- GIVEN a Contract with state mutation across phases
- WHEN `Execute(ctx, contract)` is called
- THEN Given state is passed to When, When result is passed to Then

### Requirement: ExecuteBatch

The ContractExecutor MUST provide an `ExecuteBatch(ctx, []Contract) []ContractResult` method that runs multiple contracts sequentially. All contracts MUST execute regardless of individual failures. Nil and empty contract slices MUST return nil and empty respectively.

#### Scenario: Batch execution
- GIVEN 3 contracts (pass, fail, pass)
- WHEN `ExecuteBatch(ctx, contracts)` is called
- THEN 3 results are returned: pass, fail, pass

#### Scenario: Nil batch
- GIVEN nil contracts
- WHEN `ExecuteBatch(ctx, nil)` is called
- THEN nil is returned

#### Scenario: Empty batch
- GIVEN empty contracts slice
- WHEN `ExecuteBatch(ctx, [])` is called
- THEN empty results slice is returned

### Requirement: GraphifyDiff for Structural Comparison

The test package MUST provide a `GraphifyDiff` type that computes structural differences between expected and actual graph states. It MUST capture nodes added, nodes removed, edges added, and edges removed. A `DefaultDiffThreshold` of 5 MUST be defined. A diff is "significant" when TotalDiffs exceeds the default threshold.

#### Scenario: No changes
- GIVEN expected and actual graphs with same node/edge counts
- WHEN `NewGraphifyDiff(10, 5, 10, 5)` is called
- THEN TotalDiffs is 0 and Significant is false

#### Scenario: Nodes added
- GIVEN actual has 5 more nodes than expected
- WHEN `NewGraphifyDiff(10, 5, 15, 5)` is called
- THEN NodesAdded is 5

#### Scenario: Nodes removed
- GIVEN actual has 3 fewer nodes than expected
- WHEN `NewGraphifyDiff(10, 5, 7, 5)` is called
- THEN NodesRemoved is 3

#### Scenario: Mixed changes
- GIVEN 5 nodes added and 5 edges removed
- WHEN `NewGraphifyDiff(20, 15, 25, 10)` is called
- THEN TotalDiffs is 10 and Significant is true

### Requirement: Diff Threshold

`GraphifyDiff.IsSignificant(threshold)` MUST return true when TotalDiffs exceeds the given threshold. The constructor MUST set `Significant=true` when TotalDiffs > `DefaultDiffThreshold` (5).

#### Scenario: Custom threshold
- GIVEN a diff with 8 total changes
- WHEN `IsSignificant(5)` is called
- THEN it returns true
- WHEN `IsSignificant(10)` is called
- THEN it returns false

### Requirement: Report Aggregation

The test package MUST provide a `Report` type that aggregates `ContractResult` values. It MUST count passed and failed, collect error diffs, and optionally attach a `GraphifyDiff`. The `Summary()` method MUST return a one-line summary of contract results and any graph diff.

#### Scenario: Mixed results
- GIVEN 2 passed and 1 failed contract result
- WHEN `NewReport(results)` is called
- THEN Passed is 2, Failed is 1, and Diffs has 1 entry

#### Scenario: With graph diff
- GIVEN a Report with an attached GraphifyDiff
- WHEN `Summary()` is called
- THEN it includes both contract counts and graph diff info
