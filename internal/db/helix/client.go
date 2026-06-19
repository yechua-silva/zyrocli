package helix

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	helixsdk "github.com/helixdb/helix-db/sdks/go"
	"github.com/secko/zyrocli/internal/setup"
)

// Client wraps helixsdk.Client with project-level isolation.
type Client struct {
	inner     *helixsdk.Client
	projectID string
}

// Options configures the Client.
type Options struct {
	ProjectID string
	BaseURL   string // default: http://localhost:6969
}

// Option is a functional option for NewClient.
type Option func(*Options)

// WithProjectID sets the project isolation key.
func WithProjectID(id string) Option {
	return func(o *Options) { o.ProjectID = id }
}

// WithBaseURL sets the HelixDB server URL.
func WithBaseURL(url string) Option {
	return func(o *Options) { o.BaseURL = url }
}

// Node represents a HelixDB graph node with its label and properties.
type Node struct {
	ID         int64                  `json:"$id"`
	Label      string                 `json:"$label"`
	Properties map[string]interface{} `json:"properties"`
}

// NewClient creates a new HelixDB client with project injection and a best-effort
// connection warmup (3 retries). It always returns a client even when no server
// is reachable; callers should check Ping separately for readiness.
func NewClient(ctx context.Context, opts ...Option) (*Client, error) {
	options := &Options{
		BaseURL: setup.GetHelixDBURL(),
	}
	for _, o := range opts {
		o(options)
	}

	inner, err := helixsdk.NewClient(options.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnection, err)
	}

	c := &Client{inner: inner, projectID: options.ProjectID}

	// Best-effort connection warmup — 3 retries, respects context cancellation.
	for i := 0; i < 3; i++ {
		if c.Ping(ctx) {
			return c, nil
		}
		if ctx.Err() != nil {
			// Context cancelled — stop retrying but still return the client.
			break
		}
		time.Sleep(time.Second * time.Duration(i+1))
	}

	return c, nil
}

// Close releases the client resources.
func (c *Client) Close() error {
	// The underlying SDK client has no close in its current interface.
	return nil
}

// Ping performs a health check against the HelixDB server.
// Returns true if the server responds (any non-network response is considered alive).
func (c *Client) Ping(ctx context.Context) bool {
	if c == nil || c.inner == nil {
		return false
	}
	// A minimal read query used as health check.
	req := helixsdk.ReadQuery("ping").Returning()
	if err := c.inner.Exec(ctx, req, nil); err != nil {
		// Only network errors indicate the server is unreachable.
		var helixErr *helixsdk.HelixError
		if errors.As(err, &helixErr) && helixErr.Kind == helixsdk.ErrorNetwork {
			return false
		}
		// Any other error (e.g. "query not found", auth) means the server is up.
		return true
	}
	return true
}

// InjectProject adds the project_id property to a properties map if not already set.
func (c *Client) InjectProject(props map[string]interface{}) map[string]interface{} {
	if props == nil {
		props = make(map[string]interface{})
	}
	if c.projectID != "" {
		if _, exists := props["project_id"]; !exists {
			props["project_id"] = c.projectID
		}
	}
	return props
}

// EnsureStarted checks that the HelixDB server is reachable by performing a ping.
// If the server is not reachable, it attempts to start the Docker container
// automatically via startHelixContainer.
func (c *Client) EnsureStarted(ctx context.Context) error {
	if c.Ping(ctx) {
		return nil
	}

	// Server not reachable — try to start it
	if err := startHelixContainer(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrConnection, err)
	}

	// Wait for it to be ready (up to 15s)
	for i := 0; i < 5; i++ {
		time.Sleep(2 * time.Second)
		if c.Ping(ctx) {
			return nil
		}
	}

	return fmt.Errorf("%w: server not reachable after start attempt", ErrConnection)
}

// startHelixContainer attempts to start the HelixDB Docker container.
// It tries the helix CLI first, then falls back to docker compose.
func startHelixContainer(ctx context.Context) error {
	// Method 1: helix CLI (helix up)
	if helixPath, err := exec.LookPath("helix"); err == nil {
		cmd := exec.CommandContext(ctx, helixPath, "up")
		if output, err := cmd.CombinedOutput(); err == nil {
			return nil
		} else {
			_ = output // ignore error, try docker
		}
	}

	// Method 2: docker compose with helix.toml config
	// The helix.toml defines the project's container setup
	cwd, _ := os.Getwd()
	composeFiles := []string{}
	if cwd != "" {
		composeFiles = append(composeFiles, filepath.Join(cwd, "docker-compose.yml"))
	}
	// Also check home config
	home, _ := os.UserHomeDir()
	if home != "" {
		composeFiles = append(composeFiles,
			filepath.Join(home, ".config", "zyrocli", "docker-compose.yml"),
		)
	}

	for _, path := range composeFiles {
		if _, err := os.Stat(path); err == nil {
			cmd := exec.CommandContext(ctx, "docker", "compose",
				"-f", path, "up", "-d")
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("no helix container config found — install HelixDB first or start it manually")
}

// Exec executes a raw HelixDB query against the underlying SDK client.
func (c *Client) Exec(ctx context.Context, q helixsdk.Request, out interface{}) error {
	return c.inner.Exec(ctx, q, out)
}

// Inner returns the underlying HelixDB SDK client.
func (c *Client) Inner() *helixsdk.Client {
	return c.inner
}

// ProjectID returns the project isolation identifier.
func (c *Client) ProjectID() string {
	return c.projectID
}

// TextSearch searches for nodes by label and an exact property value match, with limit.
// This is a convenience method used by legacy callers; prefer FindNodes for new code.
func (c *Client) TextSearch(ctx context.Context, label, property, value string, limit int) ([]*Node, error) {
	conds := []helixsdk.SourcePredicate{
		helixsdk.SourceEq("$label", label),
		helixsdk.SourceEq(property, value),
	}

	var where helixsdk.SourcePredicate
	if len(conds) == 1 {
		where = conds[0]
	} else {
		where = helixsdk.SourceAnd(conds...)
	}

	q := helixsdk.ReadQuery("text_search").
		VarAs("results",
			helixsdk.G().NWhere(where).
				Project(
					helixsdk.ProjectPropAs("$id", "id"),
					helixsdk.ProjectPropAs("$label", "label"),
				),
		).
		Returning("results")

	var result struct {
		Results []struct {
			ID    int64  `json:"id"`
			Label string `json:"label"`
		} `json:"results"`
	}

	if err := c.inner.Exec(ctx, q, &result); err != nil {
		return nil, fmt.Errorf("helix: text search: %w", err)
	}

	// Aplicar límite
	if limit > 0 && len(result.Results) > limit {
		result.Results = result.Results[:limit]
	}

	nodes := make([]*Node, 0, len(result.Results))
	for _, r := range result.Results {
		nodes = append(nodes, &Node{
			ID:    r.ID,
			Label: r.Label,
		})
	}
	return nodes, nil
}

// NodeFromResult creates a Node from its raw components.
func NodeFromResult(id int64, label string, props map[string]interface{}) *Node {
	return &Node{
		ID:         id,
		Label:      label,
		Properties: props,
	}
}
