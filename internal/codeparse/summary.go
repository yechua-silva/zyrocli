package codeparse

import (
	"fmt"
	"strings"
)

// GenerateSummary generates a textual summary from a ParseResult.
// Template-based, no LLM.
func GenerateSummary(result *ParseResult) string {
	if result == nil {
		return ""
	}

	var parts []string

	// Package
	parts = append(parts, fmt.Sprintf("Package %s", result.Package))

	// Exported functions
	var funcs []string
	for _, fn := range result.Functions {
		if fn.Exported {
			s := fn.Name
			if fn.Receiver != "" {
				s = fmt.Sprintf("(%s).%s", fn.Receiver, fn.Name)
			}
			funcs = append(funcs, s)
		}
	}
	if len(funcs) > 0 {
		parts = append(parts, fmt.Sprintf("provides %d funcs: %s", len(funcs), strings.Join(funcs, ", ")))
	}

	// Exported types
	var types []string
	for _, t := range result.Types {
		if t.Exported {
			types = append(types, t.Name)
		}
	}
	if len(types) > 0 {
		parts = append(parts, fmt.Sprintf("types: %s", strings.Join(types, ", ")))
	}

	// Non-stdlib dependencies
	var deps []string
	for _, imp := range result.Imports {
		path := strings.Trim(imp.Path, "\"")
		if !isStdlib(path) {
			deps = append(deps, path)
		}
	}
	if len(deps) > 0 {
		parts = append(parts, fmt.Sprintf("deps: %s", strings.Join(deps, ", ")))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, ". ") + "."
}

// isStdlib determines whether a package path belongs to the Go standard library.
// Heuristic: standard library paths do not contain a dot.
func isStdlib(path string) bool {
	return !strings.Contains(path, ".")
}

// GenerateSummaryMulti generates a multi-file summary for multiple ParseResults.
func GenerateSummaryMulti(results []*ParseResult) string {
	if len(results) == 0 {
		return ""
	}

	var summaries []string
	for _, r := range results {
		s := GenerateSummary(r)
		if s != "" {
			summaries = append(summaries, fmt.Sprintf("- %s: %s", r.File, s))
		}
	}

	return strings.Join(summaries, "\n")
}
