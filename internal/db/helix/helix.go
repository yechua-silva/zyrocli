package helix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Option configures a Client.
type Option func(*Client)

// WithBaseURL sets the HelixDB server URL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithProjectID sets the default project ID for scoped queries.
func WithProjectID(pid string) Option {
	return func(c *Client) {
		c.projectID = pid
	}
}

// Client manages connections to HelixDB. It is stateless — each method creates
// its own HTTP request. Use NewClient to create a Client with functional options.
type Client struct {
	baseURL    string
	projectID  string
	httpClient *http.Client
}

// NewClient creates a new HelixDB client with a 30-second HTTP timeout.
// No health check is performed on creation (R-HELIX-001).
func NewClient(ctx context.Context, opts ...Option) (*Client, error) {
	c := &Client{
		baseURL:    "http://localhost:6969",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	_ = ctx
	return c, nil
}

// Close releases client resources.
func (c *Client) Close() error {
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// queryError represents a structured error from HelixDB.
type queryError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// queryResponse is the top-level response envelope from HelixDB.
type queryResponse struct {
	Results map[string]json.RawMessage `json:"results,omitempty"`
	Error   *queryError                `json:"error,omitempty"`
}

// doQuery sends a JSON query to HelixDB's /v1/query endpoint and returns the
// parsed response. It maps HTTP and application-level errors to sentinel errors.
func (c *Client) doQuery(ctx context.Context, body map[string]any) (*queryResponse, error) {
	reqBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("helix: marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/query", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("helix: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.projectID != "" {
		req.Header.Set("x-project-id", c.projectID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrConnectionFailed, err.Error())
	}
	defer resp.Body.Close()

	// Read full body for error diagnostics.
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("helix: read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%w: HTTP %d %s", ErrConnectionFailed, resp.StatusCode, string(respBody))
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: HTTP %d %s", ErrInvalidRequest, resp.StatusCode, string(respBody))
	}

	var qr queryResponse
	if err := json.Unmarshal(respBody, &qr); err != nil {
		return nil, fmt.Errorf("helix: decode response: %w", err)
	}

	if qr.Error != nil {
		switch qr.Error.Code {
		case "NOT_FOUND":
			return nil, ErrNotFound
		default:
			return nil, fmt.Errorf("%w: %s", ErrInvalidRequest, qr.Error.Message)
		}
	}

	return &qr, nil
}

// buildBatch creates the standard batch query envelope used by all methods.
func buildBatch(ops []map[string]any, returning []string) map[string]any {
	return map[string]any{
		"batch":     ops,
		"returning": returning,
	}
}

// singleOp creates a single-operation batch entry with a variable name.
func singleOp(varName string, op string, params map[string]any) []map[string]any {
	return []map[string]any{
		{
			"var": varName,
			op:    params,
		},
	}
}

// parseNode extracts a single Node from the "n" result key.
func parseNode(qr *queryResponse) (*Node, error) {
	raw, ok := qr.Results["n"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return nil, ErrNotFound
	}
	var nodes []*Node
	if err := json.Unmarshal(raw, &nodes); err != nil {
		return nil, fmt.Errorf("helix: parse node: %w", err)
	}
	if len(nodes) == 0 {
		return nil, ErrNotFound
	}
	return nodes[0], nil
}

// parseNodes extracts a slice of Nodes from the "n" result key.
func parseNodes(qr *queryResponse) ([]*Node, error) {
	raw, ok := qr.Results["n"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return nil, ErrNotFound
	}
	var nodes []*Node
	if err := json.Unmarshal(raw, &nodes); err != nil {
		return nil, fmt.Errorf("helix: parse nodes: %w", err)
	}
	return nodes, nil
}

// parseEdge extracts a single Edge from the "e" result key.
func parseEdge(qr *queryResponse) (*Edge, error) {
	raw, ok := qr.Results["e"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return nil, ErrNotFound
	}
	var edges []*Edge
	if err := json.Unmarshal(raw, &edges); err != nil {
		return nil, fmt.Errorf("helix: parse edge: %w", err)
	}
	if len(edges) == 0 {
		return nil, ErrNotFound
	}
	return edges[0], nil
}

// parseEdges extracts a slice of Edges from the "e" result key.
func parseEdges(qr *queryResponse) ([]*Edge, error) {
	raw, ok := qr.Results["e"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return nil, ErrNotFound
	}
	var edges []*Edge
	if err := json.Unmarshal(raw, &edges); err != nil {
		return nil, fmt.Errorf("helix: parse edges: %w", err)
	}
	return edges, nil
}

// ---------------------------------------------------------------------------
// Node CRUD
// ---------------------------------------------------------------------------

// CreateNode creates a new node with the given label and properties.
func (c *Client) CreateNode(ctx context.Context, label string, props map[string]any) (*Node, error) {
	if props == nil {
		props = make(map[string]any)
	}
	query := buildBatch(
		singleOp("n", "add_node", map[string]any{
			"label":      label,
			"properties": props,
		}),
		[]string{"n"},
	)
	qr, err := c.doQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	return parseNode(qr)
}

// GetNode retrieves a node by label and ID.
func (c *Client) GetNode(ctx context.Context, label string, id int64) (*Node, error) {
	query := buildBatch(
		singleOp("n", "get_node", map[string]any{
			"label": label,
			"id":    id,
		}),
		[]string{"n"},
	)
	qr, err := c.doQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	return parseNode(qr)
}

// UpdateNode updates properties on an existing node.
func (c *Client) UpdateNode(ctx context.Context, label string, id int64, props map[string]any) (*Node, error) {
	if props == nil {
		props = make(map[string]any)
	}
	query := buildBatch(
		singleOp("n", "update_node", map[string]any{
			"label":      label,
			"id":         id,
			"properties": props,
		}),
		[]string{"n"},
	)
	qr, err := c.doQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	return parseNode(qr)
}

// DeleteNode deletes a node by label and ID.
func (c *Client) DeleteNode(ctx context.Context, label string, id int64) error {
	query := buildBatch(
		singleOp("n", "delete_node", map[string]any{
			"label": label,
			"id":    id,
		}),
		[]string{},
	)
	_, err := c.doQuery(ctx, query)
	return err
}

// ---------------------------------------------------------------------------
// Edge CRUD
// ---------------------------------------------------------------------------

// CreateEdge creates a directed edge between two nodes with the given label and properties.
func (c *Client) CreateEdge(ctx context.Context, fromID, toID int64, label string, props map[string]any) (*Edge, error) {
	if props == nil {
		props = make(map[string]any)
	}
	query := buildBatch(
		singleOp("e", "add_edge", map[string]any{
			"from":       fromID,
			"to":         toID,
			"label":      label,
			"properties": props,
		}),
		[]string{"e"},
	)
	qr, err := c.doQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	return parseEdge(qr)
}

// GetOutgoing retrieves the target nodes of outgoing edges from a node,
// filtered by edge label.
func (c *Client) GetOutgoing(ctx context.Context, nodeID int64, edgeLabel string) ([]*Node, error) {
	query := buildBatch(
		singleOp("n", "get_outgoing", map[string]any{
			"id":    nodeID,
			"label": edgeLabel,
		}),
		[]string{"n"},
	)
	qr, err := c.doQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	return parseNodes(qr)
}

// GetIncoming retrieves the source nodes of incoming edges to a node,
// filtered by edge label.
func (c *Client) GetIncoming(ctx context.Context, nodeID int64, edgeLabel string) ([]*Node, error) {
	query := buildBatch(
		singleOp("n", "get_incoming", map[string]any{
			"id":    nodeID,
			"label": edgeLabel,
		}),
		[]string{"n"},
	)
	qr, err := c.doQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	return parseNodes(qr)
}

// DeleteEdge deletes an edge by its ID.
func (c *Client) DeleteEdge(ctx context.Context, edgeID int64) error {
	query := buildBatch(
		singleOp("e", "delete_edge", map[string]any{
			"id": edgeID,
		}),
		[]string{},
	)
	_, err := c.doQuery(ctx, query)
	return err
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

// TextSearch searches nodes of the given label for a text query on a property.
func (c *Client) TextSearch(ctx context.Context, label, property, query string, limit int) ([]*Node, error) {
	q := buildBatch(
		singleOp("n", "text_search", map[string]any{
			"label":    label,
			"property": property,
			"query":    query,
			"limit":    limit,
		}),
		[]string{"n"},
	)
	qr, err := c.doQuery(ctx, q)
	if err != nil {
		return nil, err
	}
	return parseNodes(qr)
}

// VectorSearch performs a vector similarity search on a node property.
func (c *Client) VectorSearch(ctx context.Context, label, property string, vector []float32, k int) ([]*Node, error) {
	q := buildBatch(
		singleOp("n", "vector_search", map[string]any{
			"label":    label,
			"property": property,
			"vector":   vector,
			"k":        k,
		}),
		[]string{"n"},
	)
	qr, err := c.doQuery(ctx, q)
	if err != nil {
		return nil, err
	}
	return parseNodes(qr)
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

// CreateIndex creates a new index with the given specification.
func (c *Client) CreateIndex(ctx context.Context, spec IndexSpec) error {
	q := buildBatch(
		singleOp("i", "create_index", map[string]any{
			"name":   spec.Name,
			"type":   spec.Type,
			"fields": spec.Fields,
		}),
		[]string{"i"},
	)
	_, err := c.doQuery(ctx, q)
	return err
}

// ListIndexes lists all indexes in the project.
func (c *Client) ListIndexes(ctx context.Context) ([]IndexSpec, error) {
	q := buildBatch(
		singleOp("i", "list_indexes", map[string]any{}),
		[]string{"i"},
	)
	qr, err := c.doQuery(ctx, q)
	if err != nil {
		return nil, err
	}
	raw, ok := qr.Results["i"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return nil, ErrNotFound
	}
	var specs []IndexSpec
	if err := json.Unmarshal(raw, &specs); err != nil {
		return nil, fmt.Errorf("helix: parse index specs: %w", err)
	}
	return specs, nil
}
