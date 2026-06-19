package context

import (
	"bufio"
	stdctx "context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const (
	defaultQueryTimeout  = 30 * time.Second
	defaultStopGraceful  = 5 * time.Second
	bridgeBinary         = "context"
)

// jsonRPCMessage represents a JSON-RPC 2.0 request/response.
type jsonRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Bridge manages a Context MCP server process lifecycle and JSON-RPC queries.
type Bridge struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	mu     sync.Mutex
	running bool

	queryTimeout time.Duration
	stopGraceful time.Duration
	nextID       int
}

// NewBridge creates a new unstarted Bridge with default timeouts.
func NewBridge() *Bridge {
	return &Bridge{
		queryTimeout: defaultQueryTimeout,
		stopGraceful: defaultStopGraceful,
	}
}

// Start launches the MCP server process via exec.Command.
// The binary used is "context" with arguments "serve --libs".
func (b *Bridge) Start(ctx stdctx.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.running {
		return fmt.Errorf("bridge: already running")
	}

	cmd := exec.CommandContext(ctx, bridgeBinary, "serve", "--libs")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("bridge start: stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("bridge start: stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("bridge start: exec: %w", err)
	}

	b.cmd = cmd
	b.stdin = stdin
	b.stdout = stdout
	b.running = true
	b.nextID = 0

	return nil
}

// Stop sends SIGTERM and waits up to stopGraceful for graceful shutdown,
// then falls back to SIGKILL.
func (b *Bridge) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return nil
	}

	// Close stdin to signal graceful shutdown
	if b.stdin != nil {
		b.stdin.Close()
	}

	// Send SIGTERM
	proc := b.cmd.Process
	if proc != nil {
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			b.forceKill(proc)
			b.cleanup()
			return nil
		}
	}

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		b.cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(b.stopGraceful):
		if proc != nil {
			b.forceKill(proc)
		}
	}

	b.cleanup()
	return nil
}

func (b *Bridge) forceKill(proc *os.Process) {
	_ = proc.Kill()
	b.cmd.Wait()
}

func (b *Bridge) cleanup() {
	b.running = false
	b.cmd = nil
	b.stdin = nil
	b.stdout = nil
}

// IsRunning returns whether the bridge process is currently running.
func (b *Bridge) IsRunning() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.running
}

// QueryDocs sends a JSON-RPC query to the MCP server and returns the documentation result.
// The libraryID is the canonical library ID (e.g. "/vercel/next.js").
func (b *Bridge) QueryDocs(ctx stdctx.Context, libraryID, query string) ([]byte, error) {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return nil, fmt.Errorf("querydocs: bridge not running")
	}
	b.nextID++
	reqID := b.nextID
	stdin := b.stdin
	stdout := b.stdout
	b.mu.Unlock()

	return b.sendRequest(ctx, stdin, stdout, reqID, "query_docs", map[string]string{
		"library_id": libraryID,
		"query":      query,
	})
}

// ResolveLibraryID resolves a package name to a canonical library ID via the Context MCP.
// Returns the ID string in format "/org/project" or "/org/project/version".
func (b *Bridge) ResolveLibraryID(ctx stdctx.Context, packageName string) (string, error) {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return "", fmt.Errorf("resolve: bridge not running")
	}
	b.nextID++
	reqID := b.nextID
	stdin := b.stdin
	stdout := b.stdout
	b.mu.Unlock()

	raw, err := b.sendRequest(ctx, stdin, stdout, reqID, "resolve_library_id", map[string]string{
		"package_name": packageName,
	})
	if err != nil {
		return "", err
	}

	var result struct {
		LibraryID string `json:"library_id"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("resolve: unmarshal result: %w", err)
	}
	if result.LibraryID == "" {
		return "", fmt.Errorf("resolve: empty library ID for package %q", packageName)
	}
	return result.LibraryID, nil
}

// sendRequest sends a JSON-RPC request and returns the response result.
func (b *Bridge) sendRequest(ctx stdctx.Context, stdin io.Writer, stdout io.Reader, id int, method string, paramsMap map[string]string) ([]byte, error) {
	timeoutCtx, cancel := stdctx.WithTimeout(ctx, b.queryTimeout)
	defer cancel()

	paramsRaw, err := json.Marshal(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal params: %w", method, err)
	}

	req := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  paramsRaw,
	}

	reqRaw, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal request: %w", method, err)
	}

	// Send request
	if _, err := stdin.Write(append(reqRaw, '\n')); err != nil {
		return nil, fmt.Errorf("%s: write: %w", method, err)
	}

	// Read response
	type rpcResponse struct {
		resp jsonRPCMessage
		err  error
	}

	responseChan := make(chan rpcResponse, 1)

	go func() {
		var resp jsonRPCMessage
		decoder := json.NewDecoder(bufio.NewReader(stdout))
		if err := decoder.Decode(&resp); err != nil {
			responseChan <- rpcResponse{err: fmt.Errorf("%s: decode: %w", method, err)}
			return
		}
		responseChan <- rpcResponse{resp: resp}
	}()

	select {
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("%s: timeout: %w", method, timeoutCtx.Err())
	case rr := <-responseChan:
		if rr.err != nil {
			return nil, rr.err
		}
		if rr.resp.Error != nil {
			return nil, fmt.Errorf("%s: JSON-RPC error %d: %s", method, rr.resp.Error.Code, rr.resp.Error.Message)
		}
		return rr.resp.Result, nil
	}
}
