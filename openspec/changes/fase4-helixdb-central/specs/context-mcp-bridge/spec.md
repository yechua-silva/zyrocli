# Context MCP Bridge Specification (Delta)

## MODIFIED Requirements

### Requirement: Coexistence Clarification

The existing `context` bridge (`internal/context/bridge.go`) MUST remain unchanged. It continues as the connector to the external `context` binary for library documentation queries. The new MCP server (`internal/mcp/`) is a separate, complementary process — it does not replace bridge.go.
(Previously: bridge.go was the only MCP client in the system)

#### Scenario: Bridge unchanged after Fase 4
- GIVEN the existing bridge.go implementation
- WHEN Fase 4 is deployed
- THEN bridge.go has zero lines changed
- AND the `context` binary queries still work via `Bridge.QueryDocs()`

#### Scenario: No stdio conflict
- GIVEN both bridge.go and the new MCP server are running
- WHEN the bridge sends a JSON-RPC request to the `context` binary
- THEN the response comes from the `context` binary, not the MCP server
- (They are separate processes with separate stdin/stdout)
