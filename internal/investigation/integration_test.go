package investigation

import (
	"context"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Integration-style tests: ResearchEngine + Advisor pipeline
// ---------------------------------------------------------------------------

func TestIntegration_ResearchThenAdvise(t *testing.T) {
	// Simulate a full research → advisory pipeline
	engine := NewResearchEngine(ResearchEngineConfig{
		Project:      "zyroagentcli",
		DocLibraries: []string{"/golang/go", "/spf13/cobra"},
		DocQueryFn: func(_ context.Context, lib, query string) ([]byte, error) {
			if lib == "/golang/go" {
				return []byte("# Go 1.22\n\nBest practices and idioms"), nil
			}
			return []byte("# Cobra CLI\n\nStandard CLI framework"), nil
		},
		ExternalURLs: []string{"https://example.com/docs"},
		WebFetchFn: func(_ context.Context, url string) (string, error) {
			return "# Example Docs\n\nExternal API reference", nil
		},
	})

	report := engine.Run(context.Background())

	if !report.HasData() {
		t.Fatal("expected report to have data")
	}
	if len(report.DocSources) != 2 {
		t.Errorf("expected 2 doc sources, got %d", len(report.DocSources))
	}
	if len(report.WebSources) != 1 {
		t.Errorf("expected 1 web source, got %d", len(report.WebSources))
	}

	// Now feed into advisor
	advisor := NewAdvisor(AdvisorConfig{
		Project:     "zyroagentcli",
		Language:    "go",
		Framework:   "cobra",
		ProjectType: "cli",
		MVPBoundaries: []string{
			"Parse handoff.yaml",
			"Run SDD phases",
			"Generate docs",
		},
		AvailableSkills: []string{"go-testing"},
	})

	advisory := advisor.Analyze(report)

	if advisory.Project != "zyroagentcli" {
		t.Errorf("expected project zyroagentcli, got %q", advisory.Project)
	}

	// Should have stack recs for Go + Cobra CLI
	foundCobra := false
	for _, s := range advisory.Stack {
		if s.Name == "Cobra" {
			foundCobra = true
			break
		}
	}
	if !foundCobra {
		t.Error("expected Cobra stack recommendation")
	}

	// Should have patterns
	if len(advisory.Patterns) == 0 {
		t.Error("expected pattern recommendations")
	}

	// Should have skills excluding already-installed ones
	for _, sl := range advisory.SkillsByLayer {
		for _, s := range sl.Skills {
			if s == "go-testing" {
				t.Errorf("go-testing should be excluded (already installed)")
			}
		}
	}

	// MVP approach should include boundaries
	if !strings.Contains(advisory.MVPApproach, "Parse handoff.yaml") {
		t.Error("expected MVP approach to mention boundaries")
	}
}

func TestIntegration_ReportMarkdownPipeline(t *testing.T) {
	engine := NewResearchEngine(ResearchEngineConfig{
		Project:      "test",
		DocLibraries: []string{"/lib/a"},
		DocQueryFn: func(_ context.Context, lib, query string) ([]byte, error) {
			return []byte("Doc content for " + lib), nil
		},
	})
	report := engine.Run(context.Background())

	md := report.Markdown()
	if !strings.Contains(md, "Investigation Report") {
		t.Error("expected Investigation Report header")
	}
	if !strings.Contains(md, "/lib/a") {
		t.Error("expected library reference in markdown")
	}
}

func TestIntegration_AdvisoryMarkdownPipeline(t *testing.T) {
	advisor := NewAdvisor(AdvisorConfig{
		Project:     "test",
		Language:    "go",
		ProjectType: "cli",
	})
	advisory := advisor.Analyze(nil)

	md := advisory.Markdown()
	if !strings.Contains(md, "Advisor Recommendations") {
		t.Error("expected Advisor Recommendations header")
	}
}

func TestIntegration_PartialFailure(t *testing.T) {
	// Some sources fail, others succeed
	engine := NewResearchEngine(ResearchEngineConfig{
		Project:      "test",
		DocLibraries: []string{"/lib/a", "/lib/b"},
		DocQueryFn: func(_ context.Context, lib, query string) ([]byte, error) {
			if lib == "/lib/b" {
				return nil, context.DeadlineExceeded
			}
			return []byte("content for a"), nil
		},
		ExternalURLs: []string{"https://good.example.com", "https://bad.example.com"},
		WebFetchFn: func(_ context.Context, url string) (string, error) {
			if strings.Contains(url, "bad") {
				return "", context.DeadlineExceeded
			}
			return "good content", nil
		},
	})

	report := engine.Run(context.Background())

	// Check by LibraryID (concurrent execution means order isn't guaranteed)
	foundSuccess := false
	foundFailed := false
	for _, d := range report.DocSources {
		if d.LibraryID == "/lib/a" && d.Error == "" && d.Content == "content for a" {
			foundSuccess = true
		}
		if d.LibraryID == "/lib/b" && d.Error != "" {
			foundFailed = true
		}
	}
	if !foundSuccess {
		t.Error("expected /lib/a doc source to succeed")
	}
	if !foundFailed {
		t.Error("expected /lib/b doc source to have error")
	}
	// Good web source
	foundGood := false
	for _, w := range report.WebSources {
		if w.URL == "https://good.example.com" && w.Content == "good content" {
			foundGood = true
			break
		}
	}
	if !foundGood {
		t.Error("expected good.example.com to have content")
	}
	// Git analysis ran regardless
	if report.GitAnalysis == nil {
		t.Error("expected git analysis to run even with partial failures")
	}
}

// Test the wrapper concept: the integration test validates that the
// zyro-sdd-* wrapper pattern (inject topic keys + protocol) works
// by simulating what the wrapper does: fetch topic keys from conventions,
// inject into context, run the phase, persist the result.
func TestIntegration_WrapperPatternExplore(t *testing.T) {
	// Simulate what a zyro-sdd-explore wrapper does:
	// 1. Read topic keys (simulated)
	topicKey := "sdd/test-change/explore"
	project := "zyroagentcli"
	changeName := "test-change"

	// 2. Run exploration phase (ResearchEngine + Advisor as simulated "explore")
	engine := NewResearchEngine(ResearchEngineConfig{
		Project: project,
		DocLibraries: []string{"/golang/go"},
		DocQueryFn: func(_ context.Context, lib, query string) ([]byte, error) {
			return []byte("# Go investigation result"), nil
		},
	})
	report := engine.Run(context.Background())

	// 3. Format with standard Engram entry format (simulated)
	engramEntry := formatExploreEntry(project, changeName, topicKey, report)

	// 4. Verify the format
	if !strings.Contains(engramEntry, project) {
		t.Error("expected project in engram entry")
	}
	if !strings.Contains(engramEntry, changeName) {
		t.Error("expected change name in engram entry")
	}
	if !strings.Contains(engramEntry, topicKey) {
		t.Error("expected topic key in engram entry")
	}
}

func TestIntegration_WrapperPatternPropose(t *testing.T) {
	// Simulate what a zyro-sdd-propose wrapper does:
	// 1. Prerequisite check: explore must exist (simulated)
	exploreExists := true
	if !exploreExists {
		t.Fatal("exploration must exist before proposal")
	}

	// 2. Read topic keys
	topicKey := "sdd/test-change/proposal"
	project := "zyroagentcli"
	changeName := "test-change"

	// 3. Run proposal phase using advisor
	advisor := NewAdvisor(AdvisorConfig{
		Project:     project,
		Language:    "go",
		ProjectType: "cli",
		MVPBoundaries: []string{
			"Implement feature X",
		},
	})
	advisory := advisor.Analyze(nil)

	// 4. Format with standard Engram entry format
	engramEntry := formatProposeEntry(project, changeName, topicKey, advisory)

	// 5. Verify the format
	if !strings.Contains(engramEntry, project) {
		t.Error("expected project in engram entry")
	}
	if !strings.Contains(engramEntry, "Proposal") {
		t.Error("expected Proposal in entry title")
	}
}

// ---------------------------------------------------------------------------
// Simulated wrapper formatters (what the SKILL.md wrapper would produce)
// ---------------------------------------------------------------------------

func formatExploreEntry(project, change, topicKey string, report *Report) string {
	var b strings.Builder
	b.WriteString("## Explore: " + change + "\n")
	b.WriteString("- **project**: " + project + "\n")
	b.WriteString("- **change**: " + change + "\n")
	b.WriteString("- **artifact**: explore\n")
	b.WriteString("- **topic_key**: " + topicKey + "\n")
	b.WriteString("- **status**: draft\n\n")
	b.WriteString("### What\nInvestigation completed for " + change + "\n\n")
	b.WriteString("### Where\ninternal/investigation/\n\n")
	if report != nil {
		b.WriteString("### Findings\n")
		b.WriteString(report.Markdown())
	}
	return b.String()
}

func formatProposeEntry(project, change, topicKey string, advisory *Advisory) string {
	var b strings.Builder
	b.WriteString("## Proposal: " + change + "\n")
	b.WriteString("- **project**: " + project + "\n")
	b.WriteString("- **change**: " + change + "\n")
	b.WriteString("- **artifact**: proposal\n")
	b.WriteString("- **topic_key**: " + topicKey + "\n")
	b.WriteString("- **status**: draft\n\n")
	b.WriteString("### What\nProposal created for " + change + "\n\n")
	b.WriteString("### Where\nBased on investigation recommendations\n\n")
	if advisory != nil {
		b.WriteString("### Recommendations\n")
		b.WriteString(advisory.Markdown())
	}
	return b.String()
}
