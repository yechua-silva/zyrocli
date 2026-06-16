package context

import "fmt"

// QueryResult holds the result of a documentation query against a library.
type QueryResult struct {
	LibraryID string
	Content   string
	Relevance float64
}

// LibraryID represents a Context MCP library identifier (the local `context` binary).
type LibraryID struct {
	Org     string
	Project string
	Version string
}

// String returns the canonical library ID string,
// e.g. "/vercel/next.js" or "/vercel/next.js/v14.0.0".
func (l LibraryID) String() string {
	if l.Version != "" {
		return fmt.Sprintf("/%s/%s/%s", l.Org, l.Project, l.Version)
	}
	return fmt.Sprintf("/%s/%s", l.Org, l.Project)
}
