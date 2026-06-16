package doc

import (
	"fmt"
)

// ---------------------------------------------------------------------------
// Sync orchestrates the full documentation sync cycle.
//
// Steps:
//  1. GenerateIndex — build a fresh doc index from conventions + filesystem
//  2. SaveIndex — write the index to .zyro/doc-index.yaml
//  3. Export — render ARCHITECTURE.md + CHANGELOG.md from templates
//  4. UpdateGraph — run graphify diff and persist
//
// Sync is idempotent: running it multiple times produces the same result
// unless the underlying conventions or changes have changed.
func Sync(projectRoot string) (*DocIndex, error) {
	// 1. Generate fresh index
	idx, err := GenerateIndex(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("doc sync: generate index: %w", err)
	}

	// 2. Save index to disk
	if err := SaveIndex(projectRoot, idx); err != nil {
		return nil, fmt.Errorf("doc sync: save index: %w", err)
	}

	// 3. Export documentation
	if err := Export(projectRoot, idx); err != nil {
		return nil, fmt.Errorf("doc sync: export: %w", err)
	}

	// 4. Update graph (non-fatal)
	if err := UpdateGraph(projectRoot, idx); err != nil {
		// Graph update failure should not block the sync
		return idx, fmt.Errorf("doc sync: graph update (non-fatal): %w", err)
	}

	return idx, nil
}
