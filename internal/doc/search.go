package doc

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Search types
// ---------------------------------------------------------------------------

// SearchQuery represents a query against the documentation index.
type SearchQuery struct {
	// TopicKey is the exact topic key to look up (fast path).
	TopicKey string

	// Query is free-text for content search (slow path).
	Query string

	// Type filters entries by type (e.g., "project", "change", "graph").
	Type string

	// ChangeName filters entries by the originating SDD change.
	ChangeName string

	// Project filters entries by project scope.
	Project string
}

// SearchResult holds a single index search match.
type SearchResult struct {
	Entry   IndexEntry
	Content string // populated when the entry is found locally
}

// ---------------------------------------------------------------------------
// SearchIndex implements the documentation search protocol.
//
// Protocol:
//  1. Fast path — exact topic_key match returns the entry directly
//  2. Slow path — search by query text across all entry fields
//  3. Fallback — return all entries for a given type or change
//  4. Last resort — return no results (caller should ask the human)
func SearchIndex(idx *DocIndex, q SearchQuery) ([]SearchResult, error) {
	if idx == nil {
		return nil, fmt.Errorf("doc: index is nil")
	}

	// 1. Fast path: exact topic key
	if q.TopicKey != "" {
		for _, entry := range idx.Entries {
			if entry.TopicKey == q.TopicKey {
				return []SearchResult{{Entry: entry}}, nil
			}
		}
		return nil, fmt.Errorf("doc: topic_key %q not found in index", q.TopicKey)
	}

	// 2. Slow path: search by query text
	if q.Query != "" {
		var results []SearchResult
		qLower := strings.ToLower(q.Query)
		for _, entry := range idx.Entries {
			if strings.Contains(strings.ToLower(entry.TopicKey), qLower) ||
				strings.Contains(strings.ToLower(entry.Type), qLower) ||
				strings.Contains(strings.ToLower(entry.ChangeName), qLower) {
				results = append(results, SearchResult{Entry: entry})
			}
		}
		if len(results) > 0 {
			return results, nil
		}
	}

	// 3. Fallback: filter by type or change
	var results []SearchResult
	for _, entry := range idx.Entries {
		if q.Type != "" && !strings.HasPrefix(entry.Type, q.Type) &&
			!strings.Contains(entry.Type, q.Type) {
			continue
		}
		if q.ChangeName != "" && entry.ChangeName != q.ChangeName {
			continue
		}
		results = append(results, SearchResult{Entry: entry})
	}

	if len(results) > 0 {
		return results, nil
	}

	// 4. Last resort: empty
	return nil, nil
}

// MustFindByTopicKey is a convenience wrapper that panics on error.
// Use in tests and CLI tooling where a missing key is a hard failure.
func MustFindByTopicKey(idx *DocIndex, topicKey string) SearchResult {
	results, err := SearchIndex(idx, SearchQuery{TopicKey: topicKey})
	if err != nil {
		panic(err)
	}
	if len(results) == 0 {
		panic(fmt.Sprintf("doc: topic_key %q not found", topicKey))
	}
	return results[0]
}
