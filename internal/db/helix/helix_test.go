package helix

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------------------------------------------------------------------------
// Error mapping (table-driven)
// ---------------------------------------------------------------------------

func TestErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		respBody   string
		method     func(*Client) error
		wantErr    error
	}{
		{
			name:       "CreateNode success",
			statusCode: http.StatusOK,
			respBody:   `{"n":{"properties":[{"id":1}]}}`,
			method: func(c *Client) error {
				_, err := c.CreateNode(context.Background(), "Skill", map[string]any{"name": "go"})
				return err
			},
			wantErr: nil,
		},
		{
			name:       "GetNode not found (404)",
			statusCode: http.StatusNotFound,
			respBody:   `{"error":{"code":"NOT_FOUND","message":"node not found"}}`,
			method: func(c *Client) error {
				_, err := c.GetNode(context.Background(), "Skill", 999)
				return err
			},
			wantErr: ErrNotFound,
		},
		{
			name:       "GetNode not found (empty ids)",
			statusCode: http.StatusOK,
			respBody:   `{"n":{"ids":[]}}`,
			method: func(c *Client) error {
				_, err := c.GetNode(context.Background(), "Skill", 999)
				return err
			},
			wantErr: ErrNotFound,
		},
		{
			name:       "TextSearch success",
			statusCode: http.StatusOK,
			respBody:   `{"n":{"ids":[10]}}`,
			method: func(c *Client) error {
				_, err := c.TextSearch(context.Background(), "Skill", "name", "testing", 10)
				return err
			},
			wantErr: nil,
		},
		{
			name:       "Server error (500)",
			statusCode: http.StatusInternalServerError,
			respBody:   `{"error":{"code":"INTERNAL","message":"server error"}}`,
			method: func(c *Client) error {
				_, err := c.CreateNode(context.Background(), "Skill", nil)
				return err
			},
			wantErr: ErrConnectionFailed,
		},
		{
			name:       "Bad request (400)",
			statusCode: http.StatusBadRequest,
			respBody:   `{"error":{"code":"INVALID","message":"bad label"}}`,
			method: func(c *Client) error {
				_, err := c.CreateNode(context.Background(), "", nil)
				return err
			},
			wantErr: ErrInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/v1/query" {
					t.Errorf("expected /v1/query, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.respBody))
			}))
			defer ts.Close()

			client, err := NewClient(context.Background(), WithBaseURL(ts.URL))
			if err != nil {
				t.Fatal(err)
			}

			got := tt.method(client)
			if tt.wantErr == nil {
				if got != nil {
					t.Fatalf("unexpected err = %v", got)
				}
			} else if !errors.Is(got, tt.wantErr) {
				t.Fatalf("err = %v, wantErr wrapping %v", got, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Connection failed
// ---------------------------------------------------------------------------

func TestConnectionFailed(t *testing.T) {
	client, err := NewClient(context.Background(), WithBaseURL("http://localhost:19999"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.CreateNode(context.Background(), "Skill", map[string]any{"name": "x"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Functional: CreateNode + GetNode round-trip
// ---------------------------------------------------------------------------

func TestCreateAndGetNode(t *testing.T) {
	var capturedBody map[string]any
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		callCount++
		if callCount == 1 {
			// CreateNode response: AddN + Project returns properties with id
			w.Write([]byte(`{"n":{"properties":[{"id":42}]}}`))
		} else {
			// GetNode response: NWhere returns ids
			w.Write([]byte(`{"n":{"ids":[42]}}`))
		}
	}))
	defer ts.Close()

	client, err := NewClient(context.Background(), WithBaseURL(ts.URL), WithProjectID("test-proj"))
	if err != nil {
		t.Fatal(err)
	}

	// Create
	node, err := client.CreateNode(context.Background(), "Skill", map[string]any{"name": "go"})
	if err != nil {
		t.Fatal(err)
	}
	if node.ID != 42 {
		t.Fatalf("id = %d, want 42", node.ID)
	}
	if node.Type != "Skill" {
		t.Fatalf("type = %q, want Skill", node.Type)
	}

	// Verify request body uses v3 envelope
	reqType, ok := capturedBody["request_type"].(string)
	if !ok || reqType != "write" {
		t.Fatalf("request_type = %v, want write", capturedBody["request_type"])
	}
	queries, ok := capturedBody["query"].(map[string]any)["queries"].([]any)
	if !ok || len(queries) != 1 {
		t.Fatalf("expected 1 query, got %v", queries)
	}
	q := queries[0].(map[string]any)["Query"].(map[string]any)
	if q["name"] != "n" {
		t.Fatalf("query name = %v, want n", q["name"])
	}
	steps := q["steps"].([]any)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}

	// Get
	node2, err := client.GetNode(context.Background(), "Skill", 42)
	if err != nil {
		t.Fatal(err)
	}
	if node2.ID != 42 {
		t.Fatalf("id = %d, want 42", node2.ID)
	}
	if node2.Type != "Skill" {
		t.Fatalf("type = %q, want Skill", node2.Type)
	}
}

// ---------------------------------------------------------------------------
// UpdateNode
// ---------------------------------------------------------------------------

func TestUpdateNode(t *testing.T) {
	var capturedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"n":{"ids":[42]}}`))
	}))
	defer ts.Close()

	client, err := NewClient(context.Background(), WithBaseURL(ts.URL))
	if err != nil {
		t.Fatal(err)
	}

	node, err := client.UpdateNode(context.Background(), "Skill", 42, map[string]any{"name": "updated"})
	if err != nil {
		t.Fatal(err)
	}
	if node.ID != 42 {
		t.Fatalf("id = %d, want 42", node.ID)
	}

	// Verify request uses SetProperty steps
	steps := capturedBody["query"].(map[string]any)["queries"].([]any)[0].(map[string]any)["Query"].(map[string]any)["steps"].([]any)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps (NWhere + SetProperty), got %d", len(steps))
	}
	setStep, ok := steps[1].(map[string]any)
	if !ok {
		t.Fatalf("step[1] should be a map, got %T", steps[1])
	}
	if _, ok := setStep["SetProperty"]; !ok {
		t.Fatalf("expected SetProperty step, got %v", setStep)
	}
}

// ---------------------------------------------------------------------------
// GetOutgoing / GetIncoming
// ---------------------------------------------------------------------------

func TestGetOutgoing(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"n":{"ids":[1,2]}}`))
	}))
	defer ts.Close()

	client, err := NewClient(context.Background(), WithBaseURL(ts.URL))
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := client.GetOutgoing(context.Background(), 42, "REQUIRES_SKILL")
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}
	if nodes[0].ID != 1 {
		t.Fatalf("id = %d, want 1", nodes[0].ID)
	}
	if nodes[1].ID != 2 {
		t.Fatalf("id = %d, want 2", nodes[1].ID)
	}
}

func TestGetIncoming(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"n":{"ids":[10]}}`))
	}))
	defer ts.Close()

	client, err := NewClient(context.Background(), WithBaseURL(ts.URL))
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := client.GetIncoming(context.Background(), 42, "HAS_TASK")
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(nodes))
	}
	if nodes[0].ID != 10 {
		t.Fatalf("id = %d, want 10", nodes[0].ID)
	}
}

// ---------------------------------------------------------------------------
// DeleteNode, CreateEdge, DeleteEdge
// ---------------------------------------------------------------------------

func TestDeleteNode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	client, err := NewClient(context.Background(), WithBaseURL(ts.URL))
	if err != nil {
		t.Fatal(err)
	}

	if err := client.DeleteNode(context.Background(), "Skill", 1); err != nil {
		t.Fatal(err)
	}
}

func TestCreateEdge(t *testing.T) {
	var capturedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"e":{"edges":[{"from":1,"to":2,"context":1,"edge_id":100}]}}`))
	}))
	defer ts.Close()

	client, err := NewClient(context.Background(), WithBaseURL(ts.URL))
	if err != nil {
		t.Fatal(err)
	}

	edge, err := client.CreateEdge(context.Background(), 1, 2, "REQUIRES_SKILL", nil)
	if err != nil {
		t.Fatal(err)
	}
	if edge.ID != 100 {
		t.Fatalf("id = %d, want 100", edge.ID)
	}
	if edge.Relation != "REQUIRES_SKILL" {
		t.Fatalf("relation = %q, want REQUIRES_SKILL", edge.Relation)
	}
	if edge.SourceID != 1 {
		t.Fatalf("source_id = %d, want 1", edge.SourceID)
	}
	if edge.TargetID != 2 {
		t.Fatalf("target_id = %d, want 2", edge.TargetID)
	}

	// Verify the request has 3 queries (src, target, e)
	queries := capturedBody["query"].(map[string]any)["queries"].([]any)
	if len(queries) != 3 {
		t.Fatalf("expected 3 queries, got %d", len(queries))
	}
}

func TestDeleteEdge(t *testing.T) {
	var capturedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	client, err := NewClient(context.Background(), WithBaseURL(ts.URL))
	if err != nil {
		t.Fatal(err)
	}

	if err := client.DeleteEdge(context.Background(), 100); err != nil {
		t.Fatal(err)
	}

	// Verify the request uses DropEdgeById
	steps := capturedBody["query"].(map[string]any)["queries"].([]any)[0].(map[string]any)["Query"].(map[string]any)["steps"].([]any)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	step := steps[0].(map[string]any)
	if _, ok := step["DropEdgeById"]; !ok {
		t.Fatalf("expected DropEdgeById step, got %v", step)
	}
}
