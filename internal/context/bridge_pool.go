// Package context provides a Bridge that manages a Context MCP server process
// (Neuledge's `context` CLI) and enables Go code to query offline documentation
// via JSON-RPC 2.0.
//
// This is the internal bridge for F0 (research phase). Skills like
// zyro-phase-0-libraries use the MCP server directly; Go code should use
// SharedBridge() to get a singleton instance.
package context

import (
	"sync"
)

var (
	globalBridge *Bridge
	bridgeOnce   sync.Once
)

// SharedBridge returns a lazily-initialized singleton Bridge that F0-related
// code can use to query documentation via the Context MCP server.
// The bridge is NOT started automatically — call Start() before first use.
//
// Usage from F0 research code:
//
//	b := context.SharedBridge()
//	if err := b.Start(ctx); err != nil { ... }
//	defer b.Stop()
//	docs, err := b.QueryDocs(ctx, libraryID, query)
func SharedBridge() *Bridge {
	bridgeOnce.Do(func() {
		globalBridge = NewBridge()
	})
	return globalBridge
}
