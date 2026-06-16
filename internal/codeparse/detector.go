package codeparse

import "strings"

// Language represents a programming language.
type Language string

const (
	LangGo         Language = "go"
	LangTypeScript Language = "typescript"
	LangPython     Language = "python"
	LangUnknown    Language = "unknown"
)

// DetectLanguage detects the programming language from a file extension.
func DetectLanguage(path string) Language {
	switch {
	case strings.HasSuffix(path, ".go"):
		return LangGo
	case strings.HasSuffix(path, ".ts"), strings.HasSuffix(path, ".tsx"):
		return LangTypeScript
	case strings.HasSuffix(path, ".py"):
		return LangPython
	default:
		return LangUnknown
	}
}

// IsParseable determines whether this file can be parsed by the current codeparse package.
// Currently only Go (via go/ast) is supported. TypeScript and Python are future targets.
func IsParseable(path string) bool {
	return DetectLanguage(path) == LangGo
}
