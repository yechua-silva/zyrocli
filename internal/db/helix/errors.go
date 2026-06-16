// Package helix provides a client for HelixDB — the persistent knowledge graph
// backing MCP tools (task_context, search_code, search_skills).
package helix

import (
	"errors"

	helixsdk "github.com/helixdb/helix-db/sdks/go"
)

// Sentinel errors returned by Client methods.
var (
	ErrNotFound         = errors.New("helix: not found")
	ErrConnection       = errors.New("helix: connection")
	ErrConnectionFailed = errors.New("helix: connection failed")
	ErrInvalidRequest   = errors.New("helix: invalid request")
	ErrTaskNotFound     = errors.New("helix: task not found")
	ErrConflict         = errors.New("helix: conflict")
)

// IsHelixNotFound checkea si un error es de tipo "not found".
func IsHelixNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrNotFound) {
		return true
	}
	var helixErr *helixsdk.HelixError
	if errors.As(err, &helixErr) {
		return helixErr.StatusCode == 404
	}
	return false
}

// IsHelixConflict checkea si un error es de tipo "conflict".
func IsHelixConflict(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrConflict) {
		return true
	}
	if helixsdk.IsConflict(err) {
		return true
	}
	var helixErr *helixsdk.HelixError
	if errors.As(err, &helixErr) {
		return helixErr.StatusCode == 409
	}
	return false
}

// IsHelixConnectionFailed checkea si un error es de conexión.
func IsHelixConnectionFailed(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrConnectionFailed) {
		return true
	}
	var helixErr *helixsdk.HelixError
	if errors.As(err, &helixErr) {
		return helixErr.Kind == helixsdk.ErrorNetwork
	}
	return false
}
