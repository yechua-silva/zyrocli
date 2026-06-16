package investigation

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// ResearchEngine tests
// ---------------------------------------------------------------------------

func TestResearchEngine_EmptyConfig(t *testing.T) {
	engine := NewResearchEngine(ResearchEngineConfig{Project: "test-proj"})
	report := engine.Run(context.Background())
	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if report.Project != "test-proj" {
		t.Errorf("expected project 'test-proj', got %q", report.Project)
	}
	if !report.HasData() {
		t.Error("expected HasData() to be true (git analysis always runs)")
	}
}

func TestResearchEngine_WithDocQuery(t *testing.T) {
	engine := NewResearchEngine(ResearchEngineConfig{
		Project:      "test-proj",
		DocLibraries: []string{"/golang/go"},
		DocQueryFn: func(_ context.Context, lib, query string) ([]byte, error) {
			return []byte("# Go Documentation\n\nBest practices for Go 1.22+"), nil
		},
	})
	report := engine.Run(context.Background())
	if len(report.DocSources) != 1 {
		t.Fatalf("expected 1 doc source, got %d", len(report.DocSources))
	}
	if report.DocSources[0].LibraryID != "/golang/go" {
		t.Errorf("expected library /golang/go, got %q", report.DocSources[0].LibraryID)
	}
	if !strings.Contains(report.DocSources[0].Content, "Go Documentation") {
		t.Error("expected doc content to contain 'Go Documentation'")
	}
}

func TestResearchEngine_DocQueryError(t *testing.T) {
	engine := NewResearchEngine(ResearchEngineConfig{
		Project:      "test-proj",
		DocLibraries: []string{"/invalid/lib"},
		DocQueryFn: func(_ context.Context, lib, query string) ([]byte, error) {
			return nil, context.DeadlineExceeded
		},
	})
	report := engine.Run(context.Background())
	if len(report.DocSources) != 1 {
		t.Fatalf("expected 1 doc source, got %d", len(report.DocSources))
	}
	if report.DocSources[0].Error == "" {
		t.Error("expected error message for failed doc query")
	}
}

func TestResearchEngine_WithWebFetch(t *testing.T) {
	engine := NewResearchEngine(ResearchEngineConfig{
		Project:      "test-proj",
		ExternalURLs: []string{"https://example.com/api"},
		WebFetchFn: func(_ context.Context, url string) (string, error) {
			return "# API Docs\n\nExample API documentation", nil
		},
	})
	report := engine.Run(context.Background())
	if len(report.WebSources) != 1 {
		t.Fatalf("expected 1 web source, got %d", len(report.WebSources))
	}
	if report.WebSources[0].URL != "https://example.com/api" {
		t.Errorf("expected URL https://example.com/api, got %q", report.WebSources[0].URL)
	}
}

func TestResearchEngine_WebFetchError(t *testing.T) {
	engine := NewResearchEngine(ResearchEngineConfig{
		Project:      "test-proj",
		ExternalURLs: []string{"https://invalid.example.com"},
		WebFetchFn: func(_ context.Context, url string) (string, error) {
			return "", context.DeadlineExceeded
		},
	})
	report := engine.Run(context.Background())
	if len(report.WebSources) != 1 {
		t.Fatalf("expected 1 web source, got %d", len(report.WebSources))
	}
	if report.WebSources[0].Error == "" {
		t.Error("expected error message for failed web fetch")
	}
}

func TestResearchEngine_MultipleSourcesConcurrent(t *testing.T) {
	engine := NewResearchEngine(ResearchEngineConfig{
		Project:      "test-proj",
		DocLibraries: []string{"/lib/a", "/lib/b", "/lib/c"},
		DocQueryFn: func(_ context.Context, lib, query string) ([]byte, error) {
			return []byte("doc for " + lib), nil
		},
		ExternalURLs: []string{"https://a.example.com", "https://b.example.com"},
		WebFetchFn: func(_ context.Context, url string) (string, error) {
			return "content from " + url, nil
		},
	})
	report := engine.Run(context.Background())

	if len(report.DocSources) != 3 {
		t.Errorf("expected 3 doc sources, got %d", len(report.DocSources))
	}
	if len(report.WebSources) != 2 {
		t.Errorf("expected 2 web sources, got %d", len(report.WebSources))
	}
	if report.GitAnalysis == nil {
		t.Error("expected git analysis to run")
	}
}

// ---------------------------------------------------------------------------
// Report tests
// ---------------------------------------------------------------------------

func TestReport_Markdown(t *testing.T) {
	report := &Report{
		Project: "test-proj",
		DocSources: []DocSource{
			{LibraryID: "/golang/go", Query: "testing", Content: "# Go Testing\n\nTable-driven tests"},
		},
		GitAnalysis: &GitSource{
			Structure: "main.go\ninternal/\n",
			Languages: ".go (10), .yaml (2)",
		},
		GeneratedAt: time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC),
	}
	md := report.Markdown()
	if !strings.Contains(md, "Investigation Report") {
		t.Error("expected 'Investigation Report' header")
	}
	if !strings.Contains(md, "/golang/go") {
		t.Error("expected doc library reference")
	}
	if !strings.Contains(md, "main.go") {
		t.Error("expected git structure content")
	}
}

func TestReport_HasData_Empty(t *testing.T) {
	r := &Report{Project: "p"}
	if r.HasData() {
		t.Error("expected HasData() to be false for empty report")
	}
}

func TestReport_HasData_WithDocs(t *testing.T) {
	r := &Report{
		Project: "p",
		DocSources: []DocSource{
			{LibraryID: "/lib/x", Content: "data"},
		},
	}
	if !r.HasData() {
		t.Error("expected HasData() to be true with doc source")
	}
}

func TestReport_HasData_WithGit(t *testing.T) {
	r := &Report{
		Project: "p",
		GitAnalysis: &GitSource{
			Structure: "src/",
		},
	}
	if !r.HasData() {
		t.Error("expected HasData() to be true with git analysis")
	}
}

// ---------------------------------------------------------------------------
// Advisor tests
// ---------------------------------------------------------------------------

func TestAdvisor_Analyze_GoCLI(t *testing.T) {
	advisor := NewAdvisor(AdvisorConfig{
		Project:     "test-cli",
		Language:    "go",
		Framework:   "cobra",
		ProjectType: "cli",
		MVPBoundaries: []string{
			"Parse handoff.yaml",
			"Generate tasks.md",
			"Run verification",
		},
	})
	advisory := advisor.Analyze(nil)

	if len(advisory.Stack) == 0 {
		t.Error("expected stack recommendations for Go CLI project")
	}

	foundCobra := false
	for _, s := range advisory.Stack {
		if s.Name == "Cobra" {
			foundCobra = true
			break
		}
	}
	if !foundCobra {
		t.Error("expected Cobra in stack recommendations for Go project")
	}

	if len(advisory.Patterns) == 0 {
		t.Error("expected pattern recommendations")
	}

	if advisory.MVPApproach == "" {
		t.Error("expected MVP approach to be generated")
	}
}

func TestAdvisor_Analyze_EmptyConfig(t *testing.T) {
	advisor := NewAdvisor(AdvisorConfig{Project: "empty"})
	advisory := advisor.Analyze(nil)

	if advisory.Project != "empty" {
		t.Errorf("expected project 'empty', got %q", advisory.Project)
	}
	// No stack recs for unknown language
	if len(advisory.Stack) > 0 {
		t.Errorf("expected no stack recs for empty config, got %d", len(advisory.Stack))
	}
	// At least table-driven tests pattern
	if len(advisory.Patterns) == 0 {
		t.Error("expected at least table-driven tests pattern")
	}
}

func TestAdvisor_Analyze_InstalledSkills(t *testing.T) {
	advisor := NewAdvisor(AdvisorConfig{
		Project:        "test-skills",
		Language:       "go",
		ProjectType:    "cli",
		AvailableSkills: []string{"go-testing", "branch-pr"},
	})
	advisory := advisor.Analyze(nil)

	// go-testing and branch-pr should NOT appear in recommendations
	for _, sl := range advisory.SkillsByLayer {
		for _, s := range sl.Skills {
			if s == "go-testing" || s == "branch-pr" {
				t.Errorf("skill %q should not appear — it's already installed", s)
			}
		}
	}
}

func TestAdvisor_Analyze_NoMVP(t *testing.T) {
	advisor := NewAdvisor(AdvisorConfig{
		Project:  "test-nomvp",
		Language: "go",
	})
	advisory := advisor.Analyze(nil)
	if advisory.MVPApproach == "" {
		t.Error("expected MVP approach even without boundaries")
	}
	if !strings.Contains(advisory.MVPApproach, "No MVP boundaries") {
		t.Error("expected MVP approach to mention missing boundaries")
	}
}

func TestAdvisory_Markdown(t *testing.T) {
	advisory := &Advisory{
		Project: "test",
		Stack: []StackRecommendation{
			{Name: "Cobra", Category: "framework", Pro: "Great", Con: "Heavy", Confidence: 0.95},
		},
		Patterns: []PatternRecommendation{
			{Pattern: "Table-driven tests", Rationale: "Go standard", Effort: "low"},
		},
		MVPApproach: "Build feature by feature.",
	}
	md := advisory.Markdown()
	if !strings.Contains(md, "Advisor Recommendations") {
		t.Error("expected 'Advisor Recommendations' header")
	}
	if !strings.Contains(md, "Cobra") {
		t.Error("expected stack recommendation content")
	}
	if !strings.Contains(md, "Table-driven tests") {
		t.Error("expected pattern recommendation content")
	}
}

func TestReport_Markdown_Error(t *testing.T) {
	report := &Report{
		Project:      "test",
		ErrorMessage: "something went wrong",
		GeneratedAt:  time.Now(),
	}
	md := report.Markdown()
	if !strings.Contains(md, "Errors") {
		t.Error("expected Errors section in markdown")
	}
	if !strings.Contains(md, "something went wrong") {
		t.Error("expected error message in markdown")
	}
}
