// Package planning implements Macro 2 of the SDD pipeline: decomposing
// investigation results into atomic features and ordering them into a
// dependency-respecting schedule.
package planning

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Feature types
// ---------------------------------------------------------------------------

// Feature represents a single atomic feature extracted from user stories.
type Feature struct {
	ID               string   `json:"id"`                 // e.g. "F1"
	Name             string   `json:"name"`               // short name, e.g. "handoff-parser"
	Description      string   `json:"description"`         // what this feature does
	Acceptance       string   `json:"acceptance"`          // acceptance criteria
	Dependencies     []string `json:"dependencies"`        // feature IDs this depends on
	Complexity       string   `json:"complexity"`          // "small", "medium", "large"
	Story            string   `json:"story,omitempty"`     // original user story
}

// ---------------------------------------------------------------------------
// Decomposer
// ---------------------------------------------------------------------------

// DecomposerConfig configures feature decomposition.
type DecomposerConfig struct {
	ProjectName string
}

// Decomposer breaks down user stories into atomic features.
type Decomposer struct {
	config DecomposerConfig
}

// NewDecomposer creates a Decomposer with the given config.
func NewDecomposer(cfg DecomposerConfig) *Decomposer {
	return &Decomposer{config: cfg}
}

// DecomposeResult holds the output of a decomposition.
type DecomposeResult struct {
	Project  string    `json:"project"`
	Features []Feature `json:"features"`
	Errors   []string  `json:"errors,omitempty"`
}

// HasFeatures returns true when at least one feature was produced.
func (r *DecomposeResult) HasFeatures() bool {
	return len(r.Features) > 0
}

// Decompose breaks a user story string into atomic features.
// It parses the story text to identify distinct functional units.
func (d *Decomposer) Decompose(story, acceptance string) *DecomposeResult {
	result := &DecomposeResult{
		Project: d.config.ProjectName,
	}

	if story == "" {
		result.Errors = append(result.Errors, "cannot plan without a user story")
		return result
	}

	features := d.identifyFeatures(story, acceptance)
	if len(features) == 0 {
		result.Errors = append(result.Errors, fmt.Sprintf("no features could be decomposed from story: %q", story))
		return result
	}

	result.Features = features
	return result
}

// identifyFeatures parses a user story into atomic features based on
// common patterns: lists, "and" conjunctions, comma-separated items.
func (d *Decomposer) identifyFeatures(story, acceptance string) []Feature {
	var features []Feature
	seen := make(map[string]bool) // deduplicate by name
	nextID := 1

	// Try splitting by common delimiters
	segments := splitStory(story)

	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		// Normalize to a feature name (kebab-case, short)
		name := toFeatureName(seg)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true

		// Infer dependencies from cross-references
		deps := inferDependencies(name, segments, seen)

		complexity := estimateComplexity(seg)

		features = append(features, Feature{
			ID:           fmt.Sprintf("F%d", nextID),
			Name:         name,
			Description:  seg,
			Acceptance:   acceptance,
			Dependencies: deps,
			Complexity:   complexity,
			Story:        story,
		})
		nextID++
	}

	return features
}

// splitStory splits a user story into candidate feature segments.
// It tries list markers first, then "and" conjunction, then returns the whole.
func splitStory(story string) []string {
	// Try bullet list
	if strings.Contains(story, "\n- ") {
		var segments []string
		for _, line := range strings.Split(story, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "- ") {
				segments = append(segments, strings.TrimPrefix(line, "- "))
			}
		}
		if len(segments) > 0 {
			return segments
		}
	}

	// Try numbered list
	if strings.Contains(story, "\n1. ") || strings.Contains(story, "\n1)") {
		var segments []string
		for _, line := range strings.Split(story, "\n") {
			line = strings.TrimSpace(line)
			if len(line) > 0 && (line[0] >= '0' && line[0] <= '9') {
				if idx := strings.IndexAny(line, ". )"); idx >= 0 {
					segments = append(segments, strings.TrimSpace(line[idx+1:]))
				}
			}
		}
		if len(segments) > 0 {
			return segments
		}
	}

	// Try splitting by " and " for compound stories
	if strings.Contains(story, " and ") {
		parts := strings.Split(story, " and ")
		if len(parts) >= 2 {
			// Verify each part is substantial enough
			valid := true
			for _, p := range parts {
				if len(strings.Fields(p)) < 3 {
					valid = false
					break
				}
			}
			if valid {
				return parts
			}
		}
	}

	return []string{story}
}

// toFeatureName converts a story segment to a short kebab-case name.
func toFeatureName(segment string) string {
	// Remove common prefixes
	cleaned := segment
	prefixes := []string{"I want to ", "I want the ", "I need to ", "I need a ", "Ability to ",
		"As a user, ", "As an operator, ", "As an admin, "}
	for _, p := range prefixes {
		if strings.HasPrefix(strings.ToLower(cleaned), strings.ToLower(p)) {
			cleaned = cleaned[len(p):]
			break
		}
	}

	// Take first 3-4 meaningful words
	words := strings.Fields(cleaned)
	if len(words) > 4 {
		words = words[:4]
	}

	// Clean up words
	var cleanWords []string
	for _, w := range words {
		w = strings.Trim(w, ",.;:!?()[]{}")
		if w != "" {
			cleanWords = append(cleanWords, strings.ToLower(w))
		}
	}

	if len(cleanWords) == 0 {
		return ""
	}

	return strings.Join(cleanWords, "-")
}

// inferDependencies checks if a segment references previously seen features.
func inferDependencies(name string, allSegments []string, seen map[string]bool) []string {
	var deps []string
	// Simple heuristic: check if the segment name (words) match any preceding feature
	words := strings.Fields(name)
	for _, w := range words {
		if len(w) < 4 {
			continue // skip short words
		}
		for other := range seen {
			if other == name {
				continue
			}
			if strings.Contains(other, w) {
				deps = append(deps, other)
			}
		}
	}
	return deps
}

// estimateComplexity estimates feature complexity based on description length
// and key terms.
func estimateComplexity(desc string) string {
	wordCount := len(strings.Fields(desc))
	if wordCount > 20 {
		return "large"
	}
	if wordCount > 8 {
		return "medium"
	}
	return "small"
}

// ---------------------------------------------------------------------------
// Feature list helpers
// ---------------------------------------------------------------------------

// FeatureByID finds a feature by its ID in a slice. Returns nil if not found.
func FeatureByID(features []Feature, id string) *Feature {
	for i, f := range features {
		if f.ID == id {
			return &features[i]
		}
	}
	return nil
}

// SummarizeFeatures returns a compact text summary of features.
func SummarizeFeatures(features []Feature) string {
	if len(features) == 0 {
		return "No features defined."
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Total features: %d\n\n", len(features)))
	for _, f := range features {
		deps := "none"
		if len(f.Dependencies) > 0 {
			deps = strings.Join(f.Dependencies, ", ")
		}
		b.WriteString(fmt.Sprintf("- %s [%s] (%s) — deps: %s\n  %s\n",
			f.ID, f.Name, f.Complexity, deps, f.Description))
	}
	return b.String()
}
