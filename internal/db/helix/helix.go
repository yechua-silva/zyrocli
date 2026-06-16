package helix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	helixsdk "github.com/helixdb/helix-db/sdks/go"
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

// EnsureStarted checks if HelixDB is reachable. If not, it tries to start it
// via `helix start dev --disk`. Returns nil when HelixDB is ready.
func (c *Client) EnsureStarted(ctx context.Context) error {
	// Quick health check
	if err := c.ping(ctx); err == nil {
		return nil // already running
	}

	// Ensure we have a global helix project to run commands from
	helixHome := os.Getenv("HOME") + "/.config/zyrocli/helix"
	os.MkdirAll(helixHome, 0755)
	if _, err := os.Stat(helixHome + "/helix.toml"); os.IsNotExist(err) {
		helixToml := `[project]
name = "zyrocli-global"
queries = "db"
container_runtime = "docker"

[local.dev]
port = 6969
image = "ghcr.io/helixdb/enterprise-dev"
tag = "latest"

[enterprise]
`
		if err := os.WriteFile(helixHome+"/helix.toml", []byte(helixToml), 0644); err != nil {
			return fmt.Errorf("helix: create config: %w", err)
		}
	}

	// Start HelixDB via its CLI (handles port mapping and container lifecycle correctly)
	cmd := exec.CommandContext(ctx, "helix", "start", "dev", "--disk")
	cmd.Dir = helixHome
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("helix: auto-start failed: %w\n%s", err, string(out))
	}

	// Wait for it to be ready (up to 30s)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if err := c.ping(ctx); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("helix: auto-started but not ready after 15s")
}

// ping checks if HelixDB is reachable by sending a minimal query.
func (c *Client) ping(ctx context.Context) error {
	body := `{"request_type":"read","parameters":{},"query":{"queries":[{"Query":{"name":"t","steps":[{"NWhere":{"Eq":["$label",{"String":"_ping"}]}}],"condition":null}}],"returns":["t"]}}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/query", bytes.NewReader([]byte(body)))
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	req.Header.Set("Content-Type", "application/json")
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return nil
	}
	return fmt.Errorf("helix: ping returned %d", resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// v3Query defines a single named query within a v3 request.
type v3Query struct {
	Name  string
	Steps []any
}

// buildV3Envelope creates a v3 API query request body.
func buildV3Envelope(queries []v3Query, requestType string) map[string]any {
	queryList := make([]map[string]any, len(queries))
	returnNames := make([]string, len(queries))
	for i, q := range queries {
		queryList[i] = map[string]any{
			"Query": map[string]any{
				"name":      q.Name,
				"steps":     q.Steps,
				"condition": nil,
			},
		}
		returnNames[i] = q.Name
	}
	return map[string]any{
		"request_type": requestType,
		"parameters":   map[string]any{},
		"query": map[string]any{
			"queries": queryList,
			"returns": returnNames,
		},
	}
}

// propsToV3 converts a map of properties to the v3 [[name, {"Value": {TYPE: val}}], ...] format.
func propsToV3(props map[string]any) [][]any {
	arr := make([][]any, 0, len(props))
	for k, v := range props {
		arr = append(arr, []any{k, valueToV3(v)})
	}
	return arr
}

// valueToV3 wraps a Go value in the v3 Value envelope.
func valueToV3(v any) map[string]any {
	switch val := v.(type) {
	case string:
		return map[string]any{"Value": map[string]any{"String": val}}
	case int64:
		return map[string]any{"Value": map[string]any{"I64": val}}
	case int:
		return map[string]any{"Value": map[string]any{"I64": val}}
	case float64:
		return map[string]any{"Value": map[string]any{"F64": val}}
	default:
		return map[string]any{"Value": map[string]any{"String": fmt.Sprintf("%v", v)}}
	}
}

// ---------------------------------------------------------------------------
// v3 response types and helpers
// ---------------------------------------------------------------------------

// v3QueryResult holds the possible fields in a v3 query response.
type v3QueryResult struct {
	Properties []map[string]any `json:"properties,omitempty"`
	IDs        []int64          `json:"ids,omitempty"`
	Edges      []v3EdgeRaw      `json:"edges,omitempty"`
}

// v3EdgeRaw matches the raw edge JSON from v3 AddE.
type v3EdgeRaw struct {
	From    int64 `json:"from"`
	To      int64 `json:"to"`
	Context int64 `json:"context"`
	EdgeID  int64 `json:"edge_id"`
}

// doQuery sends a JSON query to HelixDB's /v1/query endpoint and returns the
// response as a raw map keyed by query name. It maps HTTP and application-level
// errors to sentinel errors.
func (c *Client) doQuery(ctx context.Context, body map[string]any) (map[string]json.RawMessage, error) {
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

	var result map[string]json.RawMessage
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("helix: decode response: %w", err)
	}

	return result, nil
}

// getQueryResult extracts and parses a single named query result.
func getQueryResult(result map[string]json.RawMessage, name string) (*v3QueryResult, error) {
	raw, ok := result[name]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return nil, ErrNotFound
	}
	var qr v3QueryResult
	if err := json.Unmarshal(raw, &qr); err != nil {
		return nil, fmt.Errorf("helix: parse %s result: %w", name, err)
	}
	return &qr, nil
}

// parseSingleNode extracts a single Node from the named result key.
func parseSingleNode(result map[string]json.RawMessage, name, label string) (*Node, error) {
	qr, err := getQueryResult(result, name)
	if err != nil {
		return nil, err
	}
	if len(qr.Properties) > 0 {
		return propsToNode(qr.Properties[0], label), nil
	}
	if len(qr.IDs) > 0 {
		return &Node{ID: qr.IDs[0], Type: label}, nil
	}
	return nil, ErrNotFound
}

// parseNodeList extracts a list of Nodes from the named result key.
func parseNodeList(result map[string]json.RawMessage, name, label string) ([]*Node, error) {
	qr, err := getQueryResult(result, name)
	if err != nil {
		return nil, err
	}
	if len(qr.Properties) > 0 {
		nodes := make([]*Node, len(qr.Properties))
		for i, p := range qr.Properties {
			nodes[i] = propsToNode(p, label)
		}
		return nodes, nil
	}
	if len(qr.IDs) > 0 {
		nodes := make([]*Node, len(qr.IDs))
		for i, id := range qr.IDs {
			nodes[i] = &Node{ID: id, Type: label}
		}
		return nodes, nil
	}
	// Valid response with no results.
	return []*Node{}, nil
}

// propsToNode converts a v3 properties map to a Node, extracting the ID from $id or id.
func propsToNode(props map[string]any, label string) *Node {
	n := &Node{Type: label, Properties: make(map[string]any)}
	for k, v := range props {
		if k == "$id" || k == "id" {
			switch val := v.(type) {
			case float64:
				n.ID = int64(val)
			case int64:
				n.ID = val
			}
		} else {
			n.Properties[k] = v
		}
	}
	return n
}

// parseEdgeResult extracts the first Edge from the "e" result key.
func parseEdgeResult(result map[string]json.RawMessage, label string) (*Edge, error) {
	qr, err := getQueryResult(result, "e")
	if err != nil {
		return nil, err
	}
	if len(qr.Edges) > 0 {
		re := qr.Edges[0]
		return &Edge{
			ID:       re.EdgeID,
			SourceID: re.From,
			TargetID: re.To,
			Relation: label,
		}, nil
	}
	if len(qr.Properties) > 0 {
		raw := qr.Properties[0]
		e := &Edge{Relation: label}
		for k, v := range raw {
			switch k {
			case "id":
				switch val := v.(type) {
				case float64:
					e.ID = int64(val)
				case int64:
					e.ID = val
				}
			case "source_id":
				switch val := v.(type) {
				case float64:
					e.SourceID = int64(val)
				case int64:
					e.SourceID = val
				}
			case "target_id":
				switch val := v.(type) {
				case float64:
					e.TargetID = int64(val)
				case int64:
					e.TargetID = val
				}
			}
		}
		return e, nil
	}
	return nil, ErrNotFound
}

// ---------------------------------------------------------------------------
// Node CRUD
// ---------------------------------------------------------------------------

// CreateNode creates a new node with the given label and properties.
func (c *Client) CreateNode(ctx context.Context, label string, props map[string]any) (*Node, error) {
	if props == nil {
		props = make(map[string]any)
	}
	payload := buildV3Envelope([]v3Query{
		{
			Name: "n",
			Steps: []any{
				map[string]any{
					"AddN": map[string]any{
						"label":      label,
						"properties": propsToV3(props),
					},
				},
				map[string]any{
					"Project": []map[string]any{
						{"source": "$id", "alias": "id"},
					},
				},
			},
		},
	}, "write")
	result, err := c.doQuery(ctx, payload)
	if err != nil {
		return nil, err
	}
	return parseSingleNode(result, "n", label)
}

// GetNode retrieves a node by label and ID. In v3, only IDs are returned;
// properties are not fetched by default.
func (c *Client) GetNode(ctx context.Context, label string, id int64) (*Node, error) {
	payload := buildV3Envelope([]v3Query{
		{
			Name: "n",
			Steps: []any{
				map[string]any{
					"NWhere": map[string]any{
						"Eq": []any{"$id", map[string]any{"I64": id}},
					},
				},
			},
		},
	}, "read")
	result, err := c.doQuery(ctx, payload)
	if err != nil {
		return nil, err
	}
	return parseSingleNode(result, "n", label)
}

// UpdateNode sets properties on an existing node. Properties are added or
// updated individually — existing properties not in the map are left untouched.
func (c *Client) UpdateNode(ctx context.Context, label string, id int64, props map[string]any) (*Node, error) {
	if props == nil {
		props = make(map[string]any)
	}
	steps := []any{
		map[string]any{
			"NWhere": map[string]any{
				"Eq": []any{"$id", map[string]any{"I64": id}},
			},
		},
	}
	for k, v := range props {
		steps = append(steps, map[string]any{
			"SetProperty": []any{k, valueToV3(v)},
		})
	}
	payload := buildV3Envelope([]v3Query{{Name: "n", Steps: steps}}, "write")
	result, err := c.doQuery(ctx, payload)
	if err != nil {
		return nil, err
	}
	return parseSingleNode(result, "n", label)
}

// DeleteNode removes a node from the query result set. In v3, Drop does not
// persist to the database; the node remains but is excluded from query results.
func (c *Client) DeleteNode(ctx context.Context, label string, id int64) error {
	payload := buildV3Envelope([]v3Query{
		{
			Name: "n",
			Steps: []any{
				map[string]any{
					"NWhere": map[string]any{
						"Eq": []any{"$id", map[string]any{"I64": id}},
					},
				},
				"Drop",
			},
		},
	}, "write")
	_, err := c.doQuery(ctx, payload)
	return err
}

// ---------------------------------------------------------------------------
// Edge CRUD
// ---------------------------------------------------------------------------

// CreateEdge creates a directed edge between two nodes with the given label.
func (c *Client) CreateEdge(ctx context.Context, fromID, toID int64, label string, props map[string]any) (*Edge, error) {
	if props == nil {
		props = make(map[string]any)
	}
	payload := buildV3Envelope([]v3Query{
		{
			Name: "src",
			Steps: []any{
				map[string]any{"NWhere": map[string]any{"Eq": []any{"$id", map[string]any{"I64": fromID}}}},
			},
		},
		{
			Name: "target",
			Steps: []any{
				map[string]any{"NWhere": map[string]any{"Eq": []any{"$id", map[string]any{"I64": toID}}}},
			},
		},
		{
			Name: "e",
			Steps: []any{
				map[string]any{"N": map[string]any{"Var": "src"}},
				map[string]any{"AddE": map[string]any{
					"label":      label,
					"to":         map[string]any{"Var": "target"},
					"properties": propsToV3(props),
				}},
			},
		},
	}, "write")
	result, err := c.doQuery(ctx, payload)
	if err != nil {
		return nil, err
	}
	e, err := parseEdgeResult(result, label)
	if err != nil {
		return nil, err
	}
	return e, nil
}

// GetOutgoing retrieves the target node IDs of outgoing edges from a node,
// filtered by edge label. Returns Node objects with at minimum the ID set.
func (c *Client) GetOutgoing(ctx context.Context, nodeID int64, edgeLabel string) ([]*Node, error) {
	payload := buildV3Envelope([]v3Query{
		{
			Name: "src",
			Steps: []any{
				map[string]any{"NWhere": map[string]any{"Eq": []any{"$id", map[string]any{"I64": nodeID}}}},
			},
		},
		{
			Name: "n",
			Steps: []any{
				map[string]any{"N": map[string]any{"Var": "src"}},
				map[string]any{"Out": edgeLabel},
			},
		},
	}, "read")
	result, err := c.doQuery(ctx, payload)
	if err != nil {
		return nil, err
	}
	return parseNodeList(result, "n", "")
}

// GetIncoming retrieves the source node IDs of incoming edges to a node,
// filtered by edge label. Returns Node objects with at minimum the ID set.
func (c *Client) GetIncoming(ctx context.Context, nodeID int64, edgeLabel string) ([]*Node, error) {
	payload := buildV3Envelope([]v3Query{
		{
			Name: "src",
			Steps: []any{
				map[string]any{"NWhere": map[string]any{"Eq": []any{"$id", map[string]any{"I64": nodeID}}}},
			},
		},
		{
			Name: "n",
			Steps: []any{
				map[string]any{"N": map[string]any{"Var": "src"}},
				map[string]any{"In": edgeLabel},
			},
		},
	}, "read")
	result, err := c.doQuery(ctx, payload)
	if err != nil {
		return nil, err
	}
	return parseNodeList(result, "n", "")
}

// DeleteEdge deletes an edge by its ID.
func (c *Client) DeleteEdge(ctx context.Context, edgeID int64) error {
	payload := buildV3Envelope([]v3Query{
		{
			Name: "e",
			Steps: []any{
				map[string]any{"DropEdgeById": map[string]any{"Ids": []int64{edgeID}}},
			},
		},
	}, "write")
	_, err := c.doQuery(ctx, payload)
	return err
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

// TextSearch searches nodes of the given label for a text query on a property.
// Requires a text index on (label, property) created via CreateIndex.
func (c *Client) TextSearch(ctx context.Context, label, property, query string, limit int) ([]*Node, error) {
	payload := buildV3Envelope([]v3Query{
		{
			Name: "n",
			Steps: []any{
				map[string]any{
					"TextSearchNodes": map[string]any{
						"label":      label,
						"property":   property,
						"query_text": map[string]any{"Value": map[string]any{"String": query}},
						"k":          map[string]any{"Literal": limit},
					},
				},
			},
		},
	}, "read")
	result, err := c.doQuery(ctx, payload)
	if err != nil {
		return nil, err
	}
	return parseNodeList(result, "n", label)
}

// VectorSearch performs a vector similarity search on a node property.
func (c *Client) VectorSearch(ctx context.Context, label, property string, vector []float32, k int) ([]*Node, error) {
	payload := buildV3Envelope([]v3Query{
		{
			Name: "n",
			Steps: []any{
				map[string]any{
					"VectorSearchNodes": map[string]any{
						"label":      label,
						"property":   property,
						"vector":     vector,
						"k":          map[string]any{"Literal": k},
					},
				},
			},
		},
	}, "read")
	result, err := c.doQuery(ctx, payload)
	if err != nil {
		return nil, err
	}
	return parseNodeList(result, "n", label)
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

// CreateIndex creates a new index for the given label+property.
func (c *Client) CreateIndex(ctx context.Context, spec IndexSpec) error {
	var nodeType string
	switch spec.IndexType {
	case IndexVector:
		nodeType = "NodeVector"
	case IndexText:
		nodeType = "NodeText"
	case IndexEquality:
		nodeType = "NodeEquality"
	case IndexRange:
		nodeType = "NodeRange"
	case IndexUnique:
		nodeType = "NodeUnique"
	default:
		nodeType = "NodeText"
	}
	specMap := map[string]any{
		nodeType: map[string]any{
			"label":    spec.Label,
			"property": spec.Property,
		},
	}
	payload := buildV3Envelope([]v3Query{
		{
			Name: "i",
			Steps: []any{
				map[string]any{
					"CreateIndex": map[string]any{
						"spec":          specMap,
						"if_not_exists": true,
					},
				},
			},
		},
	}, "write")
	_, err := c.doQuery(ctx, payload)
	return err
}

// ListIndexes lists all indexes in the project.
func (c *Client) ListIndexes(ctx context.Context) ([]IndexSpec, error) {
	// v3 does not expose a "list indexes" operation.
	return []IndexSpec{}, nil
}

// ---------------------------------------------------------------------------
// SDK Client Wrapper (T-2.1)
// ---------------------------------------------------------------------------

// SDKClientOptions configura el cliente HelixDB SDK.
type SDKClientOptions struct {
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
}

// DefaultSDKClientOptions retorna opciones por defecto.
func DefaultSDKClientOptions() SDKClientOptions {
	return SDKClientOptions{
		BaseURL:    "http://localhost:6969",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}
}

// SDKClient es el wrapper sobre el SDK oficial de HelixDB.
type SDKClient struct {
	inner *helixsdk.Client
	opts  SDKClientOptions
}

// NewSDKClient crea un nuevo cliente HelixDB SDK.
func NewSDKClient(opts SDKClientOptions) (*SDKClient, error) {
	if opts.BaseURL == "" {
		opts.BaseURL = "http://localhost:6969"
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.MaxRetries == 0 {
		opts.MaxRetries = 3
	}

	// Configurar HTTP client con timeout personalizado
	httpClient := &http.Client{Timeout: opts.Timeout}
	inner, err := helixsdk.NewClient(opts.BaseURL, helixsdk.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("helix: new sdk client: %w", err)
	}

	return &SDKClient{inner: inner, opts: opts}, nil
}

// Exec ejecuta una query con reintentos.
func (c *SDKClient) Exec(ctx context.Context, q helixsdk.Request, out interface{}) error {
	var lastErr error
	for i := 0; i <= c.opts.MaxRetries; i++ {
		if i > 0 {
			time.Sleep(time.Duration(100*i) * time.Millisecond)
		}
		if err := c.inner.Exec(ctx, q, out); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("helix: exec after %d retries: %w", c.opts.MaxRetries, lastErr)
}

// HealthCheck verifica que HelixDB responda.
func (c *SDKClient) HealthCheck(ctx context.Context) error {
	// Use a simple read query to check health
	q := helixsdk.ReadQuery("health").
		VarAs("h",
			helixsdk.G().NWithLabel("_ping").Limit(1),
		).
		Returning("h")
	return c.Exec(ctx, q, nil)
}

// Close cierra el cliente. No-op ya que el SDK maneja su propio cleanup.
func (c *SDKClient) Close() {
	// SDK maneja su propio cleanup
}
