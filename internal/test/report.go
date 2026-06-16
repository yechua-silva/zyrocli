package test

import "fmt"

// DefaultDiffThreshold is the default threshold for considering a diff
// as significant. Only diffs exceeding this threshold are flagged.
const DefaultDiffThreshold = 5

// DiffEntry represents a single structural change detected during comparison.
type DiffEntry struct {
	Type   string // "node_added" | "node_removed" | "edge_added" | "edge_removed"
	Label  string // human-readable label of the changed element
	Detail string // optional additional context
}

// GraphifyDiff compares contract results with previous graph state.
// It captures structural changes between an expected and actual graph,
// such as added or removed nodes and edges.
type GraphifyDiff struct {
	NodesAdded   int
	NodesRemoved int
	EdgesAdded   int
	EdgesRemoved int
	TotalDiffs   int
	Significant  bool
	Entries      []DiffEntry
}

// NewGraphifyDiff creates a GraphifyDiff from two sets of node/edge counts.
// expected and actual represent the previous and current graph states.
func NewGraphifyDiff(expectedNodes, expectedEdges, actualNodes, actualEdges int) *GraphifyDiff {
	nodeDiff := actualNodes - expectedNodes
	edgeDiff := actualEdges - expectedEdges

	diff := &GraphifyDiff{
		Entries: make([]DiffEntry, 0),
	}

	if nodeDiff > 0 {
		diff.NodesAdded = nodeDiff
		diff.Entries = append(diff.Entries, DiffEntry{
			Type:   "node_added",
			Label:  fmt.Sprintf("%d node(s) added", nodeDiff),
			Detail: fmt.Sprintf("Expected %d nodes, got %d nodes (+%d)", expectedNodes, actualNodes, nodeDiff),
		})
	} else if nodeDiff < 0 {
		diff.NodesRemoved = -nodeDiff
		diff.Entries = append(diff.Entries, DiffEntry{
			Type:   "node_removed",
			Label:  fmt.Sprintf("%d node(s) removed", -nodeDiff),
			Detail: fmt.Sprintf("Expected %d nodes, got %d nodes (%d)", expectedNodes, actualNodes, nodeDiff),
		})
	}

	if edgeDiff > 0 {
		diff.EdgesAdded = edgeDiff
		diff.Entries = append(diff.Entries, DiffEntry{
			Type:   "edge_added",
			Label:  fmt.Sprintf("%d edge(s) added", edgeDiff),
			Detail: fmt.Sprintf("Expected %d edges, got %d edges (+%d)", expectedEdges, actualEdges, edgeDiff),
		})
	} else if edgeDiff < 0 {
		diff.EdgesRemoved = -edgeDiff
		diff.Entries = append(diff.Entries, DiffEntry{
			Type:   "edge_removed",
			Label:  fmt.Sprintf("%d edge(s) removed", -edgeDiff),
			Detail: fmt.Sprintf("Expected %d edges, got %d edges (%d)", expectedEdges, actualEdges, edgeDiff),
		})
	}

	diff.TotalDiffs = diff.NodesAdded + diff.NodesRemoved + diff.EdgesAdded + diff.EdgesRemoved
	diff.Significant = diff.TotalDiffs > DefaultDiffThreshold

	return diff
}

// IsSignificant returns true if the total number of changes exceeds the given threshold.
func (d *GraphifyDiff) IsSignificant(threshold int) bool {
	return d.TotalDiffs > threshold
}

// Summary returns a one-line summary of the diff.
func (d *GraphifyDiff) Summary() string {
	if d.TotalDiffs == 0 {
		return "No structural changes detected"
	}
	return fmt.Sprintf(
		"%d structural change(s): +%d/-%d nodes, +%d/-%d edges",
		d.TotalDiffs, d.NodesAdded, d.NodesRemoved, d.EdgesAdded, d.EdgesRemoved,
	)
}

// String returns a formatted multi-line report of the diff.
func (d *GraphifyDiff) String() string {
	if d.TotalDiffs == 0 {
		return "✓ No structural changes detected."
	}

	s := fmt.Sprintf("GraphifyDiff: %s\n", d.Summary())
	for _, entry := range d.Entries {
		s += fmt.Sprintf("  [%s] %s\n", entry.Type, entry.Label)
	}
	if d.Significant {
		s += fmt.Sprintf("  ⚠ Exceeds default threshold (%d)\n", DefaultDiffThreshold)
	}
	return s
}

// ---------------------------------------------------------------------------
// Report aggregates contract test results with optional GraphifyDiff data.
// ---------------------------------------------------------------------------

// Report captures the output of a contract test run, including graphify diff data.
type Report struct {
	Passed   int
	Failed   int
	Results  []ContractResult
	Diffs    []string
	GraphDiff *GraphifyDiff
}

// NewReport creates a Report from a set of contract results.
func NewReport(results []ContractResult) *Report {
	r := &Report{
		Results: results,
		Diffs:   make([]string, 0),
	}
	for _, res := range results {
		if res.Passed {
			r.Passed++
		} else {
			r.Failed++
			r.Diffs = append(r.Diffs, fmt.Sprintf("%s: %s", res.Name, res.Error))
		}
	}
	return r
}

// WithGraphDiff attaches a GraphifyDiff to the report.
func (r *Report) WithGraphDiff(d *GraphifyDiff) *Report {
	r.GraphDiff = d
	return r
}

// Summary returns a one-line summary of the report.
func (r *Report) Summary() string {
	s := fmt.Sprintf("Contracts: %d passed, %d failed", r.Passed, r.Failed)
	if r.GraphDiff != nil {
		s += " | " + r.GraphDiff.Summary()
	}
	return s
}
