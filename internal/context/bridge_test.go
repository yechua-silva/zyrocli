package context

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// ––––– JSON-RPC Message Tests –––––

func TestJSONRPCMessage_Marshal(t *testing.T) {
	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "query_docs",
		Params:  json.RawMessage(`{"library_id":"/vercel/next.js","query":"app router"}`),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	if !strings.Contains(string(data), `"jsonrpc":"2.0"`) {
		t.Errorf("missing jsonrpc field in %s", string(data))
	}
	if !strings.Contains(string(data), `"method":"query_docs"`) {
		t.Errorf("missing method field in %s", string(data))
	}
}

func TestJSONRPCError_Unmarshal(t *testing.T) {
	raw := `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`
	var msg jsonRPCMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if msg.Error == nil {
		t.Fatal("expected error field")
	}
	if msg.Error.Code != -32601 {
		t.Errorf("Error.Code = %d, want -32601", msg.Error.Code)
	}
	if msg.Error.Message != "Method not found" {
		t.Errorf("Error.Message = %q, want %q", msg.Error.Message, "Method not found")
	}
}

// ––––– Bridge sendRequest via pipe –––––

func TestBridge_sendRequest_Success(t *testing.T) {
	expectedResult := json.RawMessage(`{"library_id":"/vercel/next.js","content":"docs here"}`)
	response := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Result:  expectedResult,
	}
	respRaw, _ := json.Marshal(response)

	// Simulate a bridge with pipes instead of real process
	stdin := new(bytes.Buffer)
	stdout := bytes.NewReader(append(respRaw, '\n'))

	b := &Bridge{
		queryTimeout: defaultQueryTimeout,
		stopGraceful: defaultStopGraceful,
	}

	result, err := b.sendRequest(context.Background(), stdin, stdout, 1, "query_docs", map[string]string{
		"library_id": "/vercel/next.js",
		"query":      "app router",
	})
	if err != nil {
		t.Fatalf("sendRequest() error = %v", err)
	}

	if !bytes.Contains(result, []byte("docs here")) {
		t.Errorf("result = %s, want docs here", string(result))
	}
}

func TestBridge_sendRequest_JSONRPCError(t *testing.T) {
	response := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Error: &jsonRPCError{
			Code:    -32601,
			Message: "Method not found",
		},
	}
	respRaw, _ := json.Marshal(response)

	stdin := new(bytes.Buffer)
	stdout := bytes.NewReader(append(respRaw, '\n'))

	b := &Bridge{
		queryTimeout: defaultQueryTimeout,
		stopGraceful: defaultStopGraceful,
	}

	_, err := b.sendRequest(context.Background(), stdin, stdout, 1, "query_docs", map[string]string{})
	if err == nil {
		t.Fatal("expected error for JSON-RPC error response")
	}
	if !strings.Contains(err.Error(), "Method not found") {
		t.Errorf("error = %v, want Method not found", err)
	}
}

func TestBridge_sendRequest_Timeout(t *testing.T) {
	stdin := new(bytes.Buffer)
	// Never write to stdout — the read will hang, triggering timeout
	stdout := bytes.NewReader(nil)

	b := &Bridge{
		queryTimeout: 1, // 1 nanosecond — immediate timeout
		stopGraceful: defaultStopGraceful,
	}

	_, err := b.sendRequest(context.Background(), stdin, stdout, 1, "query_docs", map[string]string{})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("error = %v, want timeout", err)
	}
}

// ––––– LibraryID –––––

func TestLibraryID_String(t *testing.T) {
	tests := []struct {
		id   LibraryID
		want string
	}{
		{LibraryID{Org: "vercel", Project: "next.js"}, "/vercel/next.js"},
		{LibraryID{Org: "vercel", Project: "next.js", Version: "v14.0.0"}, "/vercel/next.js/v14.0.0"},
		{LibraryID{Org: "mongodb", Project: "docs"}, "/mongodb/docs"},
	}

	for _, tt := range tests {
		got := tt.id.String()
		if got != tt.want {
			t.Errorf("LibraryID(%+v).String() = %q, want %q", tt.id, got, tt.want)
		}
	}
}

// ––––– Bridge start/stop edge cases –––––

func TestBridge_StopNotStarted(t *testing.T) {
	b := NewBridge()
	err := b.Stop()
	if err != nil {
		t.Errorf("Stop() on unstarted bridge should not error, got = %v", err)
	}
}

func TestBridge_IsRunning_NotStarted(t *testing.T) {
	b := NewBridge()
	if b.IsRunning() {
		t.Error("IsRunning() should be false before Start()")
	}
}

func TestBridge_Start_AlreadyRunning(t *testing.T) {
	b := &Bridge{running: true}
	err := b.Start(context.Background())
	if err == nil {
		t.Fatal("expected error for double start")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("error = %v, want already running", err)
	}
}

// ––––– Complete pipe-based integration test –––––

func TestBridge_QueryDocs_PipeMock(t *testing.T) {
	// Full integration test: write a request to stdin, verify it's valid JSON-RPC,
	// then mock the response on stdout.
	stdinBuf := new(bytes.Buffer)
	stdoutResponse := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Result:  json.RawMessage(`{"content":"test docs"}`),
	}
	respRaw, _ := json.Marshal(stdoutResponse)
	stdoutReader := bytes.NewReader(append(respRaw, '\n'))

	b := &Bridge{
		stdin:        writeCloser{stdinBuf},
		stdout:       io.NopCloser(stdoutReader),
		running:      true,
		nextID:       0,
		queryTimeout: defaultQueryTimeout,
		stopGraceful: defaultStopGraceful,
	}

	result, err := b.QueryDocs(context.Background(), "/vercel/next.js", "app router")
	if err != nil {
		t.Fatalf("QueryDocs() error = %v", err)
	}

	if !bytes.Contains(result, []byte("test docs")) {
		t.Errorf("result = %s, want test docs", string(result))
	}

	// Verify the request framing
	sent := stdinBuf.String()
	if !strings.Contains(sent, `"jsonrpc":"2.0"`) {
		t.Errorf("request missing jsonrpc: %s", sent)
	}
	if !strings.Contains(sent, `"method":"query_docs"`) {
		t.Errorf("request missing method: %s", sent)
	}
}

func TestBridge_ResolveLibraryID_PipeMock(t *testing.T) {
	stdinBuf := new(bytes.Buffer)
	stdoutResponse := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Result:  json.RawMessage(`{"library_id":"/vercel/next.js"}`),
	}
	respRaw, _ := json.Marshal(stdoutResponse)
	stdoutReader := bytes.NewReader(append(respRaw, '\n'))

	b := &Bridge{
		stdin:        writeCloser{stdinBuf},
		stdout:       io.NopCloser(stdoutReader),
		running:      true,
		nextID:       0,
		queryTimeout: defaultQueryTimeout,
		stopGraceful: defaultStopGraceful,
	}

	libID, err := b.ResolveLibraryID(context.Background(), "next.js")
	if err != nil {
		t.Fatalf("ResolveLibraryID() error = %v", err)
	}

	if libID != "/vercel/next.js" {
		t.Errorf("ResolveLibraryID() = %q, want %q", libID, "/vercel/next.js")
	}
}

func TestBridge_QueryDocs_NotRunning(t *testing.T) {
	b := NewBridge()
	_, err := b.QueryDocs(context.Background(), "/vercel/next.js", "app router")
	if err == nil {
		t.Fatal("expected error when bridge not running")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("error = %v, want 'not running'", err)
	}
}

func TestBridge_ResolveLibraryID_NotRunning(t *testing.T) {
	b := NewBridge()
	_, err := b.ResolveLibraryID(context.Background(), "next.js")
	if err == nil {
		t.Fatal("expected error when bridge not running")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("error = %v, want 'not running'", err)
	}
}

func TestBridge_ResolveLibraryID_EmptyID(t *testing.T) {
	stdinBuf := new(bytes.Buffer)
	stdoutResponse := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Result:  json.RawMessage(`{"library_id":""}`),
	}
	respRaw, _ := json.Marshal(stdoutResponse)
	stdoutReader := bytes.NewReader(append(respRaw, '\n'))

	b := &Bridge{
		stdin:        writeCloser{stdinBuf},
		stdout:       io.NopCloser(stdoutReader),
		running:      true,
		nextID:       0,
		queryTimeout: defaultQueryTimeout,
		stopGraceful: defaultStopGraceful,
	}

	_, err := b.ResolveLibraryID(context.Background(), "unknown")
	if err == nil {
		t.Fatal("expected error for empty library ID")
	}
	if !strings.Contains(err.Error(), "empty library ID") {
		t.Errorf("error = %v, want 'empty library ID'", err)
	}
}

// ––––– Helpers –––––

// writeCloser wraps a bytes.Buffer to implement io.WriteCloser.
type writeCloser struct {
	*bytes.Buffer
}

func (wc writeCloser) Close() error { return nil }
