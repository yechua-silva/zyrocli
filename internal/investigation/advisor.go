package investigation

import (
	"fmt"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Recommendation types
// ---------------------------------------------------------------------------

// StackRecommendation captures a recommended library or framework with tradeoffs.
type StackRecommendation struct {
	Name        string // library / framework name
	Category    string // "library", "framework", "tool"
	Pro         string // key advantage
	Con         string // key disadvantage
	Confidence  float64 // 0.0–1.0
}

// PatternRecommendation captures an architectural pattern recommendation.
type PatternRecommendation struct {
	Pattern     string // e.g. "table-driven tests", "hexagonal architecture"
	Rationale   string // why recommended in this context
	Effort      string // "low", "medium", "high"
}

// SkillRecommendation captures a skill installation recommendation per layer.
type SkillRecommendation struct {
	Layer  string // "frontend", "backend", "cli", "testing", "documentation"
	Skills []string // recommended skills to install
}

// Advisory holds consolidated recommendations from the Advisor.
type Advisory struct {
	Project       string                 `json:"project"`
	Stack         []StackRecommendation  `json:"stack,omitempty"`
	Patterns      []PatternRecommendation `json:"patterns,omitempty"`
	SkillsByLayer []SkillRecommendation  `json:"skills_by_layer,omitempty"`
	MVPApproach   string                 `json:"mvp_approach"`
	OpenQuestions []string               `json:"open_questions,omitempty"`
}

// Markdown renders the advisory as structured markdown.
func (a *Advisory) Markdown() string {
	var b strings.Builder
	b.WriteString("## Advisor Recommendations\n\n")
	b.WriteString(fmt.Sprintf("- **Project**: %s\n\n", a.Project))

	if len(a.Stack) > 0 {
		b.WriteString("### Stack & Libraries\n\n")
		b.WriteString("| Library | Category | Pro | Con | Confidence |\n")
		b.WriteString("|---------|----------|-----|-----|------------|\n")
		for _, s := range a.Stack {
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %.0f%% |\n",
				s.Name, s.Category, s.Pro, s.Con, s.Confidence*100))
		}
		b.WriteString("\n")
	}

	if len(a.Patterns) > 0 {
		b.WriteString("### Architectural Patterns\n\n")
		for _, p := range a.Patterns {
			b.WriteString(fmt.Sprintf("- **%s**: %s (effort: %s)\n", p.Pattern, p.Rationale, p.Effort))
		}
		b.WriteString("\n")
	}

	if len(a.SkillsByLayer) > 0 {
		b.WriteString("### Skills by Layer\n\n")
		for _, sl := range a.SkillsByLayer {
			if len(sl.Skills) > 0 {
				b.WriteString(fmt.Sprintf("- **%s**: %s\n", sl.Layer, strings.Join(sl.Skills, ", ")))
			}
		}
		b.WriteString("\n")
	}

	if a.MVPApproach != "" {
		b.WriteString("### MVP Approach\n\n")
		b.WriteString(fmt.Sprintf("%s\n\n", a.MVPApproach))
	}

	if len(a.OpenQuestions) > 0 {
		b.WriteString("### Open Questions\n\n")
		for _, q := range a.OpenQuestions {
			b.WriteString(fmt.Sprintf("- %s\n", q))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// ---------------------------------------------------------------------------
// Advisor
// ---------------------------------------------------------------------------

// AdvisorConfig configures how the Advisor generates recommendations.
type AdvisorConfig struct {
	Project        string
	Language       string
	Framework      string
	ProjectType    string
	MVPBoundaries  []string // from handoff.MVP.Features
	AvailableSkills []string // skill names already installed
}

// Advisor consolidates investigation results into structured recommendations.
type Advisor struct {
	config AdvisorConfig
}

// NewAdvisor creates an Advisor with the given configuration.
func NewAdvisor(cfg AdvisorConfig) *Advisor {
	return &Advisor{config: cfg}
}

// Analyze produces an Advisory from current config and optional Report data.
func (a *Advisor) Analyze(report *Report) *Advisory {
	advisory := &Advisory{
		Project:     a.config.Project,
		MVPApproach: a.generateMVPApproach(),
	}

	// --- Stack recommendations based on project context ---
	advisory.Stack = a.recommendStack()

	// --- Pattern recommendations ---
	advisory.Patterns = a.recommendPatterns()

	// --- Skills by layer ---
	advisory.SkillsByLayer = a.recommendSkills()

	// --- Open questions from gaps in report ---
	advisory.OpenQuestions = a.identifyOpenQuestions(report)

	return advisory
}

func (a *Advisor) recommendStack() []StackRecommendation {
	var recs []StackRecommendation

	switch a.config.Language {
	case "go", "Go", "golang":
		recs = append(recs,
			StackRecommendation{
				Name: "Cobra", Category: "framework",
				Pro:  "Standard CLI framework in Go ecosystem",
				Con:  "Heavy for small CLIs",
				Confidence: 0.95,
			},
			StackRecommendation{
				Name: "go-yaml/v3", Category: "library",
				Pro:  "Required for handoff.yaml parsing",
				Con:  "No schema validation built-in",
				Confidence: 0.90,
			},
			StackRecommendation{
				Name: "table-driven tests", Category: "pattern",
				Pro:  "Go standard testing idiom, deterministic",
				Con:  "More verbose for simple cases",
				Confidence: 0.85,
			},
		)
	case "python", "Python":
		recs = append(recs,
			StackRecommendation{
				Name: "Click", Category: "framework",
				Pro:  "Simple CLI framework",
				Con:  "Less ergonomic than Typer",
				Confidence: 0.80,
			},
		)
	}

	// If framework is set, add specific recommendations
	if a.config.Framework != "" {
		switch strings.ToLower(a.config.Framework) {
		case "cobra", "cobra-cli":
			recs = append(recs, StackRecommendation{
				Name: "pterm", Category: "library",
				Pro:  "Beautiful terminal output for CLI",
				Con:  "Additional dependency",
				Confidence: 0.75,
			})
		}
	}

	return recs
}

func (a *Advisor) recommendPatterns() []PatternRecommendation {
	patterns := []PatternRecommendation{
		{
			Pattern:   "Table-driven tests",
			Rationale: "Standard Go testing idiom: deterministic, self-documenting, easy to extend",
			Effort:    "low",
		},
	}

	switch a.config.ProjectType {
	case "cli", "CLI":
		patterns = append(patterns,
			PatternRecommendation{
				Pattern:   "Command pattern",
				Rationale: "Each subcommand is a self-contained handler — matches Cobra design",
				Effort:    "low",
			},
			PatternRecommendation{
				Pattern:   "Package-local interfaces",
				Rationale: "Decouple implementation from CLI handlers; enables testing without exec",
				Effort:    "medium",
			},
		)
	case "library", "Library":
		patterns = append(patterns,
			PatternRecommendation{
				Pattern:   "Functional options",
				Rationale: "Extensible constructor pattern for Go libraries",
				Effort:    "low",
			},
		)
	}

	return patterns
}

func (a *Advisor) recommendSkills() []SkillRecommendation {
	installed := make(map[string]bool, len(a.config.AvailableSkills))
	for _, s := range a.config.AvailableSkills {
		installed[s] = true
	}

	var layers []SkillRecommendation

	allByLayer := map[string][]struct {
		Name string
		Desc string
	}{
		"cli": {
			{"go-testing", "Go test patterns, golden files, teatest"},
			{"branch-pr", "PR creation with issue-first checks"},
			{"chained-pr", "Split oversized PRs into reviewable slices"},
		},
		"testing": {
			{"go-testing", "Go test patterns, teatest, golden files"},
			{"judgment-day", "Dual review + fix + re-judge cycle"},
		},
		"documentation": {
			{"cognitive-doc-design", "Docs that reduce cognitive load"},
			{"graphify", "Persistent knowledge graph for codebase"},
		},
	}

	for layer, skills := range allByLayer {
		var toInstall []string
		for _, s := range skills {
			if !installed[s.Name] {
				toInstall = append(toInstall, s.Name)
			}
		}
		sort.Strings(toInstall)
		layers = append(layers, SkillRecommendation{
			Layer:  layer,
			Skills: toInstall,
		})
	}

	// Sort layers for deterministic output
	sort.Slice(layers, func(i, j int) bool {
		return layers[i].Layer < layers[j].Layer
	})

	return layers
}

func (a *Advisor) generateMVPApproach() string {
	if len(a.config.MVPBoundaries) == 0 {
		return "No MVP boundaries defined — recommend defining scope before implementation."
	}

	var b strings.Builder
	b.WriteString("MVP scope based on defined boundaries:\n")
	for i, f := range a.config.MVPBoundaries {
		b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, f))
	}
	b.WriteString(fmt.Sprintf("\nRecommendation: implement in order — each boundary is a self-contained work unit."))
	return b.String()
}

func (a *Advisor) identifyOpenQuestions(report *Report) []string {
	var questions []string
	if report == nil || !report.HasData() {
		questions = append(questions, "No investigation data available — recommendations are based on project config only")
	}
	if len(a.config.MVPBoundaries) == 0 {
		questions = append(questions, "MVP boundaries not specified — define scope from handoff.yaml")
	}
	return questions
}
