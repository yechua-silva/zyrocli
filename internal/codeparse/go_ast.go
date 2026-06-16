package codeparse

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ParseResult represents the result of parsing a Go source file.
type ParseResult struct {
	Package   string         // package name
	File      string         // file path
	Functions []FunctionInfo
	Types     []TypeInfo
	Imports   []ImportInfo
}

// FunctionInfo describes a function or method declaration.
type FunctionInfo struct {
	Name       string   // function name
	Receiver   string   // "Client" for methods, "" for functions
	Params     []string // e.g. "ctx context.Context", "id int64"
	Returns    []string // e.g. "error", "(*Node, error)"
	Exported   bool     // true if name starts with uppercase
	DocComment string   // doc comment text
}

// TypeInfo describes a type declaration.
type TypeInfo struct {
	Name     string   // type name
	Kind     string   // "struct", "interface", "alias", "other"
	Exported bool
	Fields   []string // "Name  string" for structs
	Methods  []string // method names for interfaces
}

// ImportInfo describes an import statement.
type ImportInfo struct {
	Path  string // e.g. "fmt", "github.com/helixdb/helix-db/sdks/go"
	Alias string // import alias, empty if no alias
}

// ParseFile parses a single .go file and returns structured information.
func ParseFile(path string) (*ParseResult, error) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("codeparse: %s: %w", path, err)
	}

	result := &ParseResult{
		Package: f.Name.Name,
		File:    path,
	}

	ast.Inspect(f, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.FuncDecl:
			fi := extractFunction(decl)
			result.Functions = append(result.Functions, fi)
		case *ast.GenDecl:
			if decl.Tok == token.TYPE {
				for _, spec := range decl.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					ti := extractType(ts)
					result.Types = append(result.Types, ti)
				}
			}
		case *ast.ImportSpec:
			imp := extractImport(decl)
			result.Imports = append(result.Imports, imp)
		}
		return true
	})

	return result, nil
}

// extractFunction extracts information from an *ast.FuncDecl.
func extractFunction(fn *ast.FuncDecl) FunctionInfo {
	info := FunctionInfo{
		Name:     fn.Name.Name,
		Exported: fn.Name.IsExported(),
	}

	// Receiver (method vs function)
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recv := fn.Recv.List[0]
		switch t := recv.Type.(type) {
		case *ast.Ident:
			info.Receiver = t.Name
		case *ast.StarExpr:
			if ident, ok := t.X.(*ast.Ident); ok {
				info.Receiver = "*" + ident.Name
			}
		}
	}

	// Parameters
	if fn.Type.Params != nil {
		for _, p := range fn.Type.Params.List {
			info.Params = append(info.Params, formatField(p))
		}
	}

	// Return values
	if fn.Type.Results != nil {
		for _, r := range fn.Type.Results.List {
			info.Returns = append(info.Returns, formatField(r))
		}
	}

	// Doc comment
	if fn.Doc != nil {
		info.DocComment = fn.Doc.Text()
	}

	return info
}

// extractType extracts information from an *ast.TypeSpec.
func extractType(ts *ast.TypeSpec) TypeInfo {
	info := TypeInfo{
		Name:     ts.Name.Name,
		Exported: ts.Name.IsExported(),
	}

	switch t := ts.Type.(type) {
	case *ast.StructType:
		info.Kind = "struct"
		if t.Fields != nil {
			for _, f := range t.Fields.List {
				info.Fields = append(info.Fields, formatField(f))
			}
		}
	case *ast.InterfaceType:
		info.Kind = "interface"
		if t.Methods != nil {
			for _, m := range t.Methods.List {
				if len(m.Names) > 0 {
					info.Methods = append(info.Methods, m.Names[0].Name)
				}
			}
		}
	case *ast.Ident:
		info.Kind = "alias"
	default:
		info.Kind = "other"
	}

	return info
}

// extractImport extracts information from an *ast.ImportSpec.
func extractImport(is *ast.ImportSpec) ImportInfo {
	info := ImportInfo{
		Path: is.Path.Value, // includes quotes
	}
	if is.Name != nil {
		info.Alias = is.Name.Name
	}
	return info
}

// formatField formats a field as a readable string: "Name  type".
func formatField(f *ast.Field) string {
	typeStr := exprToString(f.Type)
	if len(f.Names) > 0 {
		names := make([]string, len(f.Names))
		for i, n := range f.Names {
			names[i] = n.Name
		}
		return strings.Join(names, ", ") + " " + typeStr
	}
	return typeStr
}

// exprToString converts an ast.Expr to its string representation.
func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	case *ast.Ellipsis:
		return "..." + exprToString(t.Elt)
	default:
		return fmt.Sprintf("%T", t)
	}
}

// ParseDir parses all .go files in a directory (non-recursive).
func ParseDir(dir string) ([]*ParseResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var results []*ParseResult
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
			continue
		}
		result, err := ParseFile(filepath.Join(dir, e.Name()))
		if err != nil {
			// Log error but continue with other files
			continue
		}
		results = append(results, result)
	}

	return results, nil
}
