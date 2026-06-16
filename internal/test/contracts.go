package test

import (
	"context"
	"fmt"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// GivenFunc sets up the initial state for a contract test.
// It returns the state object that is passed to When and Then.
type GivenFunc func(ctx context.Context) (interface{}, error)

// WhenFunc performs the action under test.
// It receives the state from Given and returns a result for Then to verify.
type WhenFunc func(ctx context.Context, state interface{}) (interface{}, error)

// ThenFunc verifies the result produced by When.
// It receives both the original state and the result for assertions.
type ThenFunc func(ctx context.Context, state, result interface{}) error

// Contract defines a single Given/When/Then test case for validating
// agent-spec contracts against implementation behavior.
type Contract struct {
	Name  string
	Given GivenFunc
	When  WhenFunc
	Then  ThenFunc
}

// ContractResult captures the outcome of a single contract execution.
type ContractResult struct {
	Name   string
	Passed bool
	Error  string
}

// ---------------------------------------------------------------------------
// ContractExecutor
// ---------------------------------------------------------------------------

// ContractExecutor runs Given/When/Then contracts in sequence.
// Each phase feeds its output into the next: Given → state → When → result → Then.
type ContractExecutor struct{}

// NewContractExecutor creates a new ContractExecutor.
func NewContractExecutor() *ContractExecutor {
	return &ContractExecutor{}
}

// Execute runs a single contract through the Given/When/Then pipeline.
// Returns a ContractResult indicating success or failure with error details.
func (e *ContractExecutor) Execute(ctx context.Context, contract Contract) ContractResult {
	// Phase 1: Given — set up initial state
	state, err := contract.Given(ctx)
	if err != nil {
		return ContractResult{
			Name:   contract.Name,
			Passed: false,
			Error:  fmt.Sprintf("GIVEN failed: %v", err),
		}
	}

	// Phase 2: When — perform the action
	result, err := contract.When(ctx, state)
	if err != nil {
		return ContractResult{
			Name:   contract.Name,
			Passed: false,
			Error:  fmt.Sprintf("WHEN failed: %v", err),
		}
	}

	// Phase 3: Then — verify the outcome
	if contract.Then == nil {
		return ContractResult{
			Name:   contract.Name,
			Passed: true,
		}
	}

	err = contract.Then(ctx, state, result)
	if err != nil {
		return ContractResult{
			Name:   contract.Name,
			Passed: false,
			Error:  fmt.Sprintf("THEN failed: %v", err),
		}
	}

	return ContractResult{
		Name:   contract.Name,
		Passed: true,
	}
}

// ExecuteBatch runs multiple contracts sequentially.
// All contracts are executed regardless of individual failures.
func (e *ContractExecutor) ExecuteBatch(ctx context.Context, contracts []Contract) []ContractResult {
	if contracts == nil {
		return nil
	}

	results := make([]ContractResult, 0, len(contracts))
	for _, c := range contracts {
		result := e.Execute(ctx, c)
		results = append(results, result)
	}
	return results
}
