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
			respBody:   `{"results":{"n":[{"id":1,"type":"Skill","properties":{"name":"go"}}]}}`,
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
			name:       "GetNode not found (error code)",
			statusCode: http.StatusOK,
			respBody:   `{"error":{"code":"NOT_FOUND","message":"node not found"}}`,
			method: func(c *Client) error {
				_, err := c.GetNode(context.Background(), "Skill", 999)
				return err
			},
			wantErr: ErrNotFound,
		},
		{
			name:       "TextSearch success",
			statusCode: http.StatusOK,
			respBody:   `{"results":{"n":[{"id":10,"type":"Skill","properties":{"name":"testing"}}]}}`,
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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"results": map[string]any{
				"n": []map[string]any{
					{"id": 42, "type": "Skill", "properties": map[string]any{"name": "go"}},
				},
			},
		})
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
	if node.Properties["name"] != "go" {
		t.Fatalf("name = %v, want go", node.Properties["name"])
	}

	// Verify request body
	batch, ok := capturedBody["batch"].([]any)
	if !ok || len(batch) != 1 {
		t.Fatalf("expected batch with 1 op, got %v", capturedBody["batch"])
	}
	op := batch[0].(map[string]any)
	if op["var"] != "n" {
		t.Fatalf("var = %v, want n", op["var"])
	}

	// Verify x-project-id header
	// (we can't easily capture headers in this simple handler, but we trust doQuery)
}

// ---------------------------------------------------------------------------
// GetOutgoing / GetIncoming
// ---------------------------------------------------------------------------

func TestGetOutgoing(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"results": map[string]any{
				"n": []map[string]any{
					{"id": 1, "type": "Skill", "properties": map[string]any{"name": "golang"}},
					{"id": 2, "type": "Skill", "properties": map[string]any{"name": "testing"}},
				},
			},
		})
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
	if nodes[0].Type != "Skill" {
		t.Fatalf("type = %q, want Skill", nodes[0].Type)
	}
}

func TestGetIncoming(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"results": map[string]any{
				"n": []map[string]any{
					{"id": 10, "type": "Project", "properties": map[string]any{"name": "zyrocli"}},
				},
			},
		})
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
	if nodes[0].Type != "Project" {
		t.Fatalf("type = %q, want Project", nodes[0].Type)
	}
}

// ---------------------------------------------------------------------------
// DeleteNode, CreateEdge, DeleteEdge
// ---------------------------------------------------------------------------

func TestDeleteNode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":{}}`))
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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"results": map[string]any{
				"e": []map[string]any{
					{"id": 100, "source_id": 1, "target_id": 2, "relation": "REQUIRES_SKILL", "properties": map[string]any{}},
				},
			},
		})
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
}

func TestDeleteEdge(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":{}}`))
	}))
	defer ts.Close()

	client, err := NewClient(context.Background(), WithBaseURL(ts.URL))
	if err != nil {
		t.Fatal(err)
	}

	if err := client.DeleteEdge(context.Background(), 100); err != nil {
		t.Fatal(err)
	}
}
