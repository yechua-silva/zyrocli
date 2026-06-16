package codeparse

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ParseFile
// ---------------------------------------------------------------------------

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	src := `package mypkg

import "fmt"

// Hello returns a greeting.
func Hello(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}
`
	path := filepath.Join(dir, "test.go")
	err := os.WriteFile(path, []byte(src), 0644)
	require.NoError(t, err)

	result, err := ParseFile(path)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "mypkg", result.Package)
	assert.Equal(t, path, result.File)
	assert.Len(t, result.Functions, 1)
	assert.Len(t, result.Imports, 1)
	assert.Empty(t, result.Types)
}

func TestParseFile_WithFunctions(t *testing.T) {
	dir := t.TempDir()
	src := `package service

import "context"

// GetUser retrieves a user by ID.
func GetUser(ctx context.Context, id int64) (*User, error) {
	return nil, nil
}

// internal helper — not exported
func validate(id int64) bool {
	return id > 0
}

// (Client).Connect establishes a connection.
func (c *Client) Connect(ctx context.Context) error {
	return nil
}
`
	path := filepath.Join(dir, "service.go")
	err := os.WriteFile(path, []byte(src), 0644)
	require.NoError(t, err)

	result, err := ParseFile(path)
	require.NoError(t, err)
	require.Len(t, result.Functions, 3)

	// Exported function
	f1 := result.Functions[0]
	assert.Equal(t, "GetUser", f1.Name)
	assert.True(t, f1.Exported)
	assert.Empty(t, f1.Receiver)
	assert.Len(t, f1.Params, 2)
	assert.Equal(t, "ctx context.Context", f1.Params[0])
	assert.Equal(t, "id int64", f1.Params[1])
	assert.Len(t, f1.Returns, 2)
	assert.Contains(t, f1.Returns[1], "error")
	assert.Contains(t, f1.DocComment, "retrieves a user")

	// Unexported function
	f2 := result.Functions[1]
	assert.Equal(t, "validate", f2.Name)
	assert.False(t, f2.Exported)
	assert.Empty(t, f2.Receiver)

	// Method
	f3 := result.Functions[2]
	assert.Equal(t, "Connect", f3.Name)
	assert.True(t, f3.Exported)
	assert.Equal(t, "*Client", f3.Receiver)
}

func TestParseFile_WithTypes(t *testing.T) {
	dir := t.TempDir()
	src := `package model

// User represents a system user.
type User struct {
	ID   int64  ` + "`" + `json:"id"` + "`" + `
	Name string ` + "`" + `json:"name"` + "`" + `
}

// Repository defines data access operations.
type Repository interface {
	Find(id int64) (*User, error)
	Save(u *User) error
}

// Status is a string alias.
type Status string
`
	path := filepath.Join(dir, "model.go")
	err := os.WriteFile(path, []byte(src), 0644)
	require.NoError(t, err)

	result, err := ParseFile(path)
	require.NoError(t, err)
	require.Len(t, result.Types, 3)

	// Struct
	typ := result.Types[0]
	assert.Equal(t, "User", typ.Name)
	assert.True(t, typ.Exported)
	assert.Equal(t, "struct", typ.Kind)
	assert.Len(t, typ.Fields, 2)
	assert.Contains(t, typ.Fields[0], "ID")
	assert.Contains(t, typ.Fields[0], "int64")

	// Interface
	iface := result.Types[1]
	assert.Equal(t, "Repository", iface.Name)
	assert.True(t, iface.Exported)
	assert.Equal(t, "interface", iface.Kind)
	assert.Len(t, iface.Methods, 2)
	assert.Equal(t, "Find", iface.Methods[0])
	assert.Equal(t, "Save", iface.Methods[1])

	// Alias
	alias := result.Types[2]
	assert.Equal(t, "Status", alias.Name)
	assert.True(t, alias.Exported)
	assert.Equal(t, "alias", alias.Kind)
}

func TestParseFile_WithImports(t *testing.T) {
	dir := t.TempDir()
	src := `package app

import (
	"context"
	"fmt"
	helixsdk "github.com/helixdb/helix-db/sdks/go"
	"strings"
)
`
	path := filepath.Join(dir, "app.go")
	err := os.WriteFile(path, []byte(src), 0644)
	require.NoError(t, err)

	result, err := ParseFile(path)
	require.NoError(t, err)
	require.Len(t, result.Imports, 4)

	// Stdlib import (no alias)
	assert.Equal(t, `"context"`, result.Imports[0].Path)
	assert.Empty(t, result.Imports[0].Alias)

	// Third-party import with alias
	assert.Equal(t, `"github.com/helixdb/helix-db/sdks/go"`, result.Imports[2].Path)
	assert.Equal(t, "helixsdk", result.Imports[2].Alias)
}

func TestParseFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.go")
	err := os.WriteFile(path, []byte("package empty\n"), 0644)
	require.NoError(t, err)

	result, err := ParseFile(path)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "empty", result.Package)
	assert.Empty(t, result.Functions)
	assert.Empty(t, result.Types)
	assert.Empty(t, result.Imports)
}

func TestParseFile_NonExistent(t *testing.T) {
	_, err := ParseFile("/nonexistent/path.go")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "codeparse:")
}

// ---------------------------------------------------------------------------
// ParseDir
// ---------------------------------------------------------------------------

func TestParseDir(t *testing.T) {
	dir := t.TempDir()

	// Create two .go files
	f1 := filepath.Join(dir, "a.go")
	os.WriteFile(f1, []byte("package pkg\nconst X = 1\n"), 0644)

	f2 := filepath.Join(dir, "b.go")
	os.WriteFile(f2, []byte("package pkg\nfunc F() {}\n"), 0644)

	// Create a non-.go file (should be skipped)
	os.WriteFile(filepath.Join(dir, "note.md"), []byte("# readme"), 0644)

	results, err := ParseDir(dir)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Both should have the same package
	assert.Equal(t, "pkg", results[0].Package)
	assert.Equal(t, "pkg", results[1].Package)
}

func TestParseDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	results, err := ParseDir(dir)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestParseDir_NonExistent(t *testing.T) {
	_, err := ParseDir("/nonexistent/dir")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// GenerateSummary
// ---------------------------------------------------------------------------

func TestGenerateSummary(t *testing.T) {
	result := &ParseResult{
		Package: "mypkg",
		Functions: []FunctionInfo{
			{Name: "Hello", Exported: true},
			{Name: "internal", Exported: false},
			{Name: "Connect", Exported: true, Receiver: "*Client"},
		},
		Types: []TypeInfo{
			{Name: "User", Exported: true, Kind: "struct"},
			{Name: "config", Exported: false, Kind: "struct"},
		},
		Imports: []ImportInfo{
			{Path: `"fmt"`},
			{Path: `"github.com/helixdb/helix-db/sdks/go"`, Alias: "helixsdk"},
		},
	}

	summary := GenerateSummary(result)
	assert.Contains(t, summary, "Package mypkg")
	assert.Contains(t, summary, "provides 2 funcs: Hello, (*Client).Connect")
	assert.Contains(t, summary, "types: User")
	assert.Contains(t, summary, "github.com/helixdb/helix-db/sdks/go")
	assert.NotContains(t, summary, "fmt") // stdlib excluded
	assert.NotContains(t, summary, "internal") // unexported func excluded
	assert.NotContains(t, summary, "config") // unexported type excluded
}

func TestGenerateSummary_Empty(t *testing.T) {
	assert.Empty(t, GenerateSummary(nil))
	// A result with only a package still produces a minimal summary
	summary := GenerateSummary(&ParseResult{Package: "empty"})
	assert.Equal(t, "Package empty.", summary)
}

func TestGenerateSummary_NoExports(t *testing.T) {
	result := &ParseResult{
		Package: "internal",
		Functions: []FunctionInfo{
			{Name: "helper", Exported: false},
		},
		Types: []TypeInfo{
			{Name: "config", Exported: false},
		},
	}

	summary := GenerateSummary(result)
	assert.Contains(t, summary, "Package internal")
	assert.NotContains(t, summary, "funcs")
	assert.NotContains(t, summary, "types")
}

// ---------------------------------------------------------------------------
// GenerateSummaryMulti
// ---------------------------------------------------------------------------

func TestGenerateSummaryMulti(t *testing.T) {
	results := []*ParseResult{
		{Package: "pkg1", File: "a.go", Functions: []FunctionInfo{{Name: "F1", Exported: true}}},
		{Package: "pkg2", File: "b.go", Functions: []FunctionInfo{{Name: "F2", Exported: true}}},
	}

	summary := GenerateSummaryMulti(results)
	assert.Contains(t, summary, "a.go")
	assert.Contains(t, summary, "b.go")
	assert.Contains(t, summary, "pkg1")
	assert.Contains(t, summary, "pkg2")
}

func TestGenerateSummaryMulti_Empty(t *testing.T) {
	assert.Empty(t, GenerateSummaryMulti(nil))
	assert.Empty(t, GenerateSummaryMulti([]*ParseResult{}))
}

// ---------------------------------------------------------------------------
// DetectLanguage
// ---------------------------------------------------------------------------

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected Language
	}{
		{"go file", "main.go", LangGo},
		{"go test file", "main_test.go", LangGo},
		{"typescript", "component.ts", LangTypeScript},
		{"typescript react", "page.tsx", LangTypeScript},
		{"python", "script.py", LangPython},
		{"unknown ext", "readme.md", LangUnknown},
		{"no ext", "Makefile", LangUnknown},
		{"go in path", "/path/to/server.go", LangGo},
		{"ts in path", "/path/to/handler.ts", LangTypeScript},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, DetectLanguage(tt.path))
		})
	}
}

// ---------------------------------------------------------------------------
// IsParseable
// ---------------------------------------------------------------------------

func TestIsParseable(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"go file", "main.go", true},
		{"go test", "main_test.go", true},
		{"typescript", "component.ts", false},
		{"python", "script.py", false},
		{"unknown", "readme.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsParseable(tt.path))
		})
	}
}

// ---------------------------------------------------------------------------
// isStdlib
// ---------------------------------------------------------------------------

func TestIsStdlib(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"fmt", "fmt", true},
		{"context", "context", true},
		{"net/http", "net/http", true},
		{"external no sub", "github.com/foo/bar", false},
		{"external with sub", "github.com/helixdb/helix-db/sdks/go", false},
		{"custom domain", "example.com/pkg", false},
		{"stdlib io/ioutil", "io/ioutil", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isStdlib(tt.path))
		})
	}
}

// ---------------------------------------------------------------------------
// edge cases
// ---------------------------------------------------------------------------

func TestParseFile_SyntaxError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.go")
	err := os.WriteFile(path, []byte("package broken\n\nfunc broken("), 0644)
	require.NoError(t, err)

	_, err = ParseFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "codeparse:")
}

func TestParseDir_SkipParseErrors(t *testing.T) {
	dir := t.TempDir()

	// Valid file
	os.WriteFile(filepath.Join(dir, "valid.go"), []byte("package pkg\nfunc F(){}\n"), 0644)
	// Invalid file (should be skipped gracefully)
	os.WriteFile(filepath.Join(dir, "broken.go"), []byte("package broken\n\nfunc broken("), 0644)

	results, err := ParseDir(dir)
	require.NoError(t, err)
	// Should still return the valid file even though the broken one failed
	require.Len(t, results, 1)
	assert.Equal(t, "pkg", results[0].Package)
}
