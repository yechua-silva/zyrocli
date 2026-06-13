package scaffold

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"text/template"
)

//go:embed templates/go-project/*
var templateFS embed.FS

// Renderer handles template parsing and execution from the embedded filesystem.
type Renderer struct {
	funcMap template.FuncMap
	fs      fs.FS
}

// NewRenderer creates a Renderer with the embedded template FS and standard FuncMap.
func NewRenderer() *Renderer {
	return &Renderer{
		fs: templateFS,
		funcMap: template.FuncMap{
			"lower":     strings.ToLower,
			"kebab":     toKebabCase,
			"pascal":    toPascalCase,
			"normalize": normalizeName,
		},
	}
}

// Render parses and executes a template by name, returning the rendered output.
func (r *Renderer) Render(name string, cfg Config) (string, error) {
	tmplBytes, err := fs.ReadFile(r.fs, name)
	if err != nil {
		return "", fmt.Errorf("scaffold: read template %s: %w", name, err)
	}

	tmpl, err := template.New(name).Funcs(r.funcMap).Parse(string(tmplBytes))
	if err != nil {
		return "", fmt.Errorf("scaffold: parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", fmt.Errorf("scaffold: execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// toKebabCase converts a string to kebab-case: lowercase, non-alnum becomes hyphens.
func toKebabCase(s string) string {
	s = strings.ToLower(s)
	var result strings.Builder
	lastWasHyphen := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			result.WriteByte(c)
			lastWasHyphen = false
		} else {
			if !lastWasHyphen {
				result.WriteByte('-')
				lastWasHyphen = true
			}
		}
	}
	return strings.Trim(result.String(), "-")
}

// toPascalCase converts a string to PascalCase: split on hyphens/spaces, title-case each word.
func toPascalCase(s string) string {
	replaced := strings.ReplaceAll(s, "-", " ")
	words := strings.Fields(replaced)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(string(w[0])) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, "")
}

// normalizeName lowercases, replaces spaces with hyphens, and strips special characters.
func normalizeName(s string) string {
	s = strings.ToLower(s)
	s = strings.Join(strings.Fields(s), "-")
	return toKebabCase(s)
}
