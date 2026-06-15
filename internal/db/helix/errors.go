// Package helix provides a client for HelixDB — the persistent knowledge graph
// backing MCP tools (task_context, search_code, search_skills).
package helix

import "errors"

// Sentinel errors returned by Client methods.
var (
	ErrNotFound         = errors.New("helix: not found")
	ErrConnectionFailed = errors.New("helix: connection failed")
	ErrInvalidRequest   = errors.New("helix: invalid request")
	ErrTaskNotFound     = errors.New("helix: task not found")
)
