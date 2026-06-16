package doc

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// ---------------------------------------------------------------------------
// Embedded templates
// ---------------------------------------------------------------------------

//go:embed ARCHITECTURE.md.tmpl
var architectureTmpl string

//go:embed CHANGELOG.md.tmpl
var changelogTmpl string

// ---------------------------------------------------------------------------
// Export generates project documentation from the doc index.
//
// It renders ARCHITECTURE.md and CHANGELOG.md at the project root using
// embedded Go templates populated with index data.
func Export(projectRoot string, idx *DocIndex) error {
	if idx == nil {
		return fmt.Errorf("doc: cannot export nil index")
	}

	// Render ARCHITECTURE.md
	if err := renderTemplate(projectRoot, "ARCHITECTURE.md", architectureTmpl, idx); err != nil {
		return fmt.Errorf("doc: architecture export: %w", err)
	}

	// Render CHANGELOG.md
	if err := renderTemplate(projectRoot, "CHANGELOG.md", changelogTmpl, idx); err != nil {
		return fmt.Errorf("doc: changelog export: %w", err)
	}

	return nil
}

// renderTemplate renders a Go template to a file at the project root.
func renderTemplate(projectRoot, filename, tmplContent string, data interface{}) error {
	tmpl, err := template.New(filename).
		Funcs(template.FuncMap{
			"now":      func() string { return time.Now().UTC().Format(time.RFC3339) },
			"hasPrefix": strings.HasPrefix,
		}).
		Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("parsing template %s: %w", filename, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template %s: %w", filename, err)
	}

	path := filepath.Join(projectRoot, filename)
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", filename, err)
	}

	return nil
}
