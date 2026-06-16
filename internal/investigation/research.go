// Package investigation implements Macro 1 of the SDD pipeline: automated
// research combining Context MCP documentation queries, repository analysis,
// and web fetching to produce a comprehensive investigation report.
package investigation

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Source types
// ---------------------------------------------------------------------------

// DocSource represents a documentation query result from Context MCP.
type DocSource struct {
	LibraryID string // canonical library ID from Context MCP, e.g. "/golang/go"
	Query     string // the research query sent
	Content   string // markdown documentation result
	Error     string // non-empty if the query failed
}

// GitSource represents repository analysis results from GitMCP or native git.
type GitSource struct {
	Structure    string // directory tree summary
	Languages    string // detected languages breakdown
	RecentCommits []string // last N commit subjects
	Patterns     string // detected patterns (e.g. "table-driven tests")
}

// WebSource represents a fetched external URL result.
type WebSource struct {
	URL     string // the fetched URL
	Content string // markdown-converted content
	Error   string // non-empty if the fetch failed
}

// ---------------------------------------------------------------------------
// Research report
// ---------------------------------------------------------------------------

// Report holds the complete output of an investigation run.
type Report struct {
	Project      string      `json:"project"`
	DocSources   []DocSource `json:"doc_sources,omitempty"`
	GitAnalysis  *GitSource  `json:"git_analysis,omitempty"`
	WebSources   []WebSource `json:"web_sources,omitempty"`
	GeneratedAt  time.Time   `json:"generated_at"`
	ErrorMessage string      `json:"error_message,omitempty"`
}

// HasData returns true when at least one source produced content.
func (r *Report) HasData() bool {
	return len(r.DocSources) > 0 || r.GitAnalysis != nil || len(r.WebSources) > 0
}

// Markdown renders the report as a structured markdown string.
func (r *Report) Markdown() string {
	var b strings.Builder
	b.WriteString("## Investigation Report\n\n")
	b.WriteString(fmt.Sprintf("- **Project**: %s\n", r.Project))
	b.WriteString(fmt.Sprintf("- **Generated**: %s\n", r.GeneratedAt.Format(time.RFC3339)))
	b.WriteString("\n")

	if len(r.DocSources) > 0 {
		b.WriteString("### Context Documentation\n\n")
		for _, d := range r.DocSources {
			b.WriteString(fmt.Sprintf("**Library**: `%s`\n", d.LibraryID))
			b.WriteString(fmt.Sprintf("**Query**: `%s`\n", d.Query))
			if d.Error != "" {
				b.WriteString(fmt.Sprintf("**Error**: %s\n", d.Error))
			}
			if d.Content != "" {
				// Truncate very long content
				content := d.Content
				if len(content) > 1000 {
					content = content[:1000] + "...\n[truncated]"
				}
				b.WriteString(fmt.Sprintf("\n%s\n\n", content))
			}
		}
	}

	if r.GitAnalysis != nil {
		b.WriteString("### Repository Analysis\n\n")
		if r.GitAnalysis.Structure != "" {
			b.WriteString(fmt.Sprintf("**Structure**:\n```\n%s\n```\n\n", r.GitAnalysis.Structure))
		}
		if r.GitAnalysis.Languages != "" {
			b.WriteString(fmt.Sprintf("**Languages**: %s\n\n", r.GitAnalysis.Languages))
		}
		if len(r.GitAnalysis.RecentCommits) > 0 {
			b.WriteString("**Recent commits**:\n")
			for _, c := range r.GitAnalysis.RecentCommits {
				b.WriteString(fmt.Sprintf("- %s\n", c))
			}
			b.WriteString("\n")
		}
		if r.GitAnalysis.Patterns != "" {
			b.WriteString(fmt.Sprintf("**Patterns**: %s\n\n", r.GitAnalysis.Patterns))
		}
	}

	if len(r.WebSources) > 0 {
		b.WriteString("### Web Sources\n\n")
		for _, w := range r.WebSources {
			b.WriteString(fmt.Sprintf("**URL**: %s\n", w.URL))
			if w.Error != "" {
				b.WriteString(fmt.Sprintf("**Error**: %s\n", w.Error))
			} else {
				content := w.Content
				if len(content) > 500 {
					content = content[:500] + "...\n[truncated]"
				}
				b.WriteString(fmt.Sprintf("\n%s\n\n", content))
			}
		}
	}

	if r.ErrorMessage != "" {
		b.WriteString("### Errors\n\n")
		b.WriteString(fmt.Sprintf("%s\n\n", r.ErrorMessage))
	}

	return b.String()
}

// ---------------------------------------------------------------------------
// ResearchEngine
// ---------------------------------------------------------------------------

// DocQueryFn is the signature for Context MCP documentation queries.
type DocQueryFn func(ctx context.Context, libraryID, query string) ([]byte, error)

// WebFetchFn is the signature for external URL fetching.
type WebFetchFn func(ctx context.Context, url string) (string, error)

// ResearchEngineConfig configures the research engine sources.
type ResearchEngineConfig struct {
	Project      string
	DocLibraries []string // Context MCP library ID list
	ExternalURLs []string // URLs to fetch
	WebFetchFn   WebFetchFn
	DocQueryFn   DocQueryFn
}

// ResearchEngine orchestrates multiple investigation sources concurrently.
type ResearchEngine struct {
	config ResearchEngineConfig
}

// NewResearchEngine creates a ResearchEngine with the given configuration.
func NewResearchEngine(cfg ResearchEngineConfig) *ResearchEngine {
	return &ResearchEngine{config: cfg}
}

// Run executes all research sources concurrently and returns a combined Report.
func (e *ResearchEngine) Run(ctx context.Context) *Report {
	report := &Report{
		Project:     e.config.Project,
		GeneratedAt: time.Now(),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	// --- Context Docs ---
	if len(e.config.DocLibraries) > 0 && e.config.DocQueryFn != nil {
		for _, lib := range e.config.DocLibraries {
			wg.Add(1)
			lib := lib
			go func() {
				defer wg.Done()
				result := e.queryDoc(ctx, lib)
				mu.Lock()
				report.DocSources = append(report.DocSources, result)
				mu.Unlock()
			}()
		}
	}

	// --- Web Fetch ---
	if len(e.config.ExternalURLs) > 0 && e.config.WebFetchFn != nil {
		for _, url := range e.config.ExternalURLs {
			wg.Add(1)
			url := url
			go func() {
				defer wg.Done()
				result := e.fetchURL(ctx, url)
				mu.Lock()
				report.WebSources = append(report.WebSources, result)
				mu.Unlock()
			}()
		}
	}

	// --- Git Analysis ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		analysis := e.analyzeGit(ctx)
		mu.Lock()
		report.GitAnalysis = analysis
		mu.Unlock()
	}()

	wg.Wait()
	return report
}

func (e *ResearchEngine) queryDoc(ctx context.Context, libraryID string) DocSource {
	slog.Debug("investigation: querying docs", "library", libraryID)
	result := DocSource{LibraryID: libraryID, Query: "best practices"}
	data, err := e.config.DocQueryFn(ctx, libraryID, "")
	if err != nil {
		result.Error = fmt.Sprintf("doc query failed: %v", err)
		slog.Warn("investigation: doc query failed", "library", libraryID, "error", err)
		return result
	}
	result.Content = string(data)
	return result
}

func (e *ResearchEngine) fetchURL(ctx context.Context, url string) WebSource {
	slog.Debug("investigation: fetching URL", "url", url)
	result := WebSource{URL: url}
	content, err := e.config.WebFetchFn(ctx, url)
	if err != nil {
		result.Error = fmt.Sprintf("fetch failed: %v", err)
		slog.Warn("investigation: web fetch failed", "url", url, "error", err)
		return result
	}
	result.Content = content
	return result
}

func (e *ResearchEngine) analyzeGit(ctx context.Context) *GitSource {
	slog.Debug("investigation: analyzing git repository")
	analysis := &GitSource{}

	// Check if .git exists
	gitCheck := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	if err := gitCheck.Run(); err != nil {
		analysis.Patterns = "no git repository detected — skipping git analysis"
		slog.Warn("investigation: no git repo, skipping git analysis")
		return analysis
	}

	// Structure: git ls-tree
	if out, err := exec.CommandContext(ctx, "git", "ls-tree", "--name-only", "-r", "HEAD").Output(); err == nil {
		analysis.Structure = strings.TrimSpace(string(out))
	} else {
		slog.Warn("investigation: git ls-tree failed", "error", err)
	}

	// Languages: simple extension-based detection from ls-tree output
	if analysis.Structure != "" {
		extMap := map[string]int{}
		for _, line := range strings.Split(analysis.Structure, "\n") {
			if idx := strings.LastIndex(line, "."); idx >= 0 {
				ext := line[idx:]
				extMap[ext]++
			}
		}
		var parts []string
		for ext, count := range extMap {
			parts = append(parts, fmt.Sprintf("%s (%d)", ext, count))
		}
		analysis.Languages = strings.Join(parts, ", ")
	}

	// Recent commits
	if out, err := exec.CommandContext(ctx, "git", "log", "--oneline", "-10").Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				analysis.RecentCommits = append(analysis.RecentCommits, trimmed)
			}
		}
	}

	// Patterns: check for common Go testing patterns
	analysis.Patterns = e.detectPatterns(ctx)

	return analysis
}

func (e *ResearchEngine) detectPatterns(ctx context.Context) string {
	var patterns []string

	if data, err := exec.CommandContext(ctx, "grep", "-r", "-l", "func Test", "--include=*.go", ".").Output(); err == nil && len(data) > 0 {
		patternFiles := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(patternFiles) > 0 && patternFiles[0] != "" {
			patterns = append(patterns, fmt.Sprintf("table-driven tests (%d files)", len(patternFiles)))
		}
	}

	if data, err := exec.CommandContext(ctx, "grep", "-r", "-l", "interface{", "--include=*.go", ".").Output(); err == nil && len(data) > 0 {
		ifaceFiles := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(ifaceFiles) > 0 && ifaceFiles[0] != "" {
			patterns = append(patterns, fmt.Sprintf("interfaces defined (%d files)", len(ifaceFiles)))
		}
	}

	if len(patterns) == 0 {
		return "No significant patterns detected"
	}
	return strings.Join(patterns, "; ")
}
