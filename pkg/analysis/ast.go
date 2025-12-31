// Package analysis provides static analysis tools for Go source code.
// It demonstrates the use of go/ast, go/parser, go/types, and go/token
// for building custom linters and code generators.
//
// LEARN: Static analysis is a powerful technique that examines code
// without executing it. Go's standard library provides excellent
// tools for parsing and analyzing Go code.
package analysis

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// FunctionInfo holds metadata about a function declaration.
// LEARN: This type aggregates information extracted from ast.FuncDecl
type FunctionInfo struct {
	Name       string         // Function name
	Position   token.Position // File, line, and column
	Receiver   string         // Receiver type (empty for non-methods)
	NumParams  int            // Number of parameters
	NumResults int            // Number of return values
	IsExported bool           // Starts with uppercase
	Comments   string         // Associated documentation
}

// StructInfo holds metadata about a struct type declaration.
type StructInfo struct {
	Name     string         // Struct name
	Position token.Position // File, line, and column
	Fields   []FieldInfo    // Struct fields
	Comments string         // Associated documentation
}

// FieldInfo holds metadata about a struct field.
type FieldInfo struct {
	Name     string // Field name (empty for embedded)
	Type     string // Type as string
	Tag      string // Struct tag
	Embedded bool   // Is this an embedded field?
}

// ImportInfo holds metadata about an import declaration.
type ImportInfo struct {
	Path     string         // Import path
	Alias    string         // Alias (empty if none)
	Position token.Position // File, line, and column
}

// InterfaceInfo holds metadata about an interface type declaration.
type InterfaceInfo struct {
	Name     string // Interface name
	Position token.Position
	Methods  []MethodInfo // Interface methods
	Comments string
}

// MethodInfo holds metadata about an interface method.
type MethodInfo struct {
	Name    string
	Params  string // Parameter list as string
	Results string // Return types as string
}

// FileAnalysis contains all extracted information from a Go file.
type FileAnalysis struct {
	Package    string
	Filename   string
	Functions  []FunctionInfo
	Structs    []StructInfo
	Interfaces []InterfaceInfo
	Imports    []ImportInfo
}

// ParseFile parses a Go source file and returns its AST.
//
// LEARN: token.FileSet is crucial for tracking position information.
// It maps AST node positions back to file:line:column format.
// Always create one FileSet and reuse it for related files.
func ParseFile(filename string) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()

	// LEARN: parser.ParseFile accepts:
	// - filename: for error messages and position info
	// - src: nil to read from filename, or string/[]byte for source
	// - mode: parser.ParseComments to preserve documentation
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", filename, err)
	}

	return file, fset, nil
}

// ParseSource parses Go source code from a string.
//
// LEARN: This is useful for testing and for tools that receive
// source code through channels other than files.
func ParseSource(src string) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "source.go", src, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse source: %w", err)
	}

	return file, fset, nil
}

// ParseDir parses all Go files in a directory.
//
// LEARN: parser.ParseDir returns a map of package name to *ast.Package.
// Most directories contain a single package, but test files may
// create a separate "_test" package.
func ParseDir(dir string) (map[string]*ast.Package, *token.FileSet, error) {
	fset := token.NewFileSet()

	// LEARN: The filter function controls which files to parse
	// Returning nil parses all .go files
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse dir %s: %w", dir, err)
	}

	return pkgs, fset, nil
}

// AnalyzeFile extracts structured information from a Go file.
//
// LEARN: This function demonstrates the visitor pattern using ast.Inspect.
// ast.Inspect walks the entire AST, calling the provided function
// for every node. Return true to visit children, false to skip.
func AnalyzeFile(file *ast.File, fset *token.FileSet) *FileAnalysis {
	analysis := &FileAnalysis{
		Package:  file.Name.Name,
		Filename: fset.Position(file.Pos()).Filename,
	}

	// Extract imports
	for _, imp := range file.Imports {
		info := ImportInfo{
			Position: fset.Position(imp.Pos()),
		}

		// LEARN: Import paths are string literals, including quotes
		// We need to strip them for clean output
		info.Path = strings.Trim(imp.Path.Value, `"`)

		if imp.Name != nil {
			info.Alias = imp.Name.Name
		}

		analysis.Imports = append(analysis.Imports, info)
	}

	// LEARN: ast.Inspect traverses the AST depth-first
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			analysis.Functions = append(analysis.Functions, extractFunctionInfo(node, fset))

		case *ast.GenDecl:
			// LEARN: GenDecl covers import, const, type, and var declarations
			// We only care about type declarations here
			if node.Tok == token.TYPE {
				for _, spec := range node.Specs {
					typeSpec := spec.(*ast.TypeSpec)

					switch t := typeSpec.Type.(type) {
					case *ast.StructType:
						analysis.Structs = append(analysis.Structs,
							extractStructInfo(typeSpec, t, node, fset))

					case *ast.InterfaceType:
						analysis.Interfaces = append(analysis.Interfaces,
							extractInterfaceInfo(typeSpec, t, node, fset))
					}
				}
			}
		}

		return true // Continue visiting children
	})

	return analysis
}

// extractFunctionInfo extracts metadata from a function declaration.
func extractFunctionInfo(fn *ast.FuncDecl, fset *token.FileSet) FunctionInfo {
	info := FunctionInfo{
		Name:       fn.Name.Name,
		Position:   fset.Position(fn.Pos()),
		IsExported: ast.IsExported(fn.Name.Name),
	}

	// LEARN: Method receiver is in fn.Recv
	// Regular functions have nil receiver
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		info.Receiver = exprToString(fn.Recv.List[0].Type)
	}

	// Count parameters
	if fn.Type.Params != nil {
		info.NumParams = countFields(fn.Type.Params)
	}

	// Count results
	if fn.Type.Results != nil {
		info.NumResults = countFields(fn.Type.Results)
	}

	// Extract doc comment
	if fn.Doc != nil {
		info.Comments = fn.Doc.Text()
	}

	return info
}

// extractStructInfo extracts metadata from a struct type.
func extractStructInfo(spec *ast.TypeSpec, st *ast.StructType, gen *ast.GenDecl, fset *token.FileSet) StructInfo {
	info := StructInfo{
		Name:     spec.Name.Name,
		Position: fset.Position(spec.Pos()),
	}

	// LEARN: GenDecl.Doc contains the doc comment for the declaration group
	// TypeSpec.Doc contains inline comments
	if gen.Doc != nil {
		info.Comments = gen.Doc.Text()
	} else if spec.Doc != nil {
		info.Comments = spec.Doc.Text()
	}

	// Extract fields
	if st.Fields != nil {
		for _, field := range st.Fields.List {
			fi := FieldInfo{
				Type: exprToString(field.Type),
			}

			// LEARN: Named fields have names in field.Names
			// Anonymous (embedded) fields have empty Names
			if len(field.Names) > 0 {
				fi.Name = field.Names[0].Name
			} else {
				fi.Embedded = true
			}

			// Extract struct tag
			if field.Tag != nil {
				fi.Tag = field.Tag.Value
			}

			info.Fields = append(info.Fields, fi)
		}
	}

	return info
}

// extractInterfaceInfo extracts metadata from an interface type.
func extractInterfaceInfo(spec *ast.TypeSpec, iface *ast.InterfaceType, gen *ast.GenDecl, fset *token.FileSet) InterfaceInfo {
	info := InterfaceInfo{
		Name:     spec.Name.Name,
		Position: fset.Position(spec.Pos()),
	}

	if gen.Doc != nil {
		info.Comments = gen.Doc.Text()
	}

	// Extract methods
	if iface.Methods != nil {
		for _, method := range iface.Methods.List {
			// Skip embedded interfaces
			if len(method.Names) == 0 {
				continue
			}

			mi := MethodInfo{
				Name: method.Names[0].Name,
			}

			// LEARN: Interface methods have FuncType
			if ft, ok := method.Type.(*ast.FuncType); ok {
				mi.Params = fieldListToString(ft.Params)
				mi.Results = fieldListToString(ft.Results)
			}

			info.Methods = append(info.Methods, mi)
		}
	}

	return info
}

// FindFunctions returns all function names in the file.
//
// LEARN: This is a simpler alternative to AnalyzeFile when you
// only need function names. Demonstrates focused AST traversal.
func FindFunctions(file *ast.File) []string {
	var functions []string

	ast.Inspect(file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			functions = append(functions, fn.Name.Name)
		}
		return true
	})

	return functions
}

// FindCalls returns all function calls in the file.
//
// LEARN: Useful for call graph analysis or finding usages
// of specific functions.
func FindCalls(file *ast.File, fset *token.FileSet) []CallInfo {
	var calls []CallInfo

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		info := CallInfo{
			Position: fset.Position(call.Pos()),
			NumArgs:  len(call.Args),
		}

		// LEARN: Call target can be:
		// - Ident: simple function call like fmt.Println
		// - SelectorExpr: method call like obj.Method()
		// - Other expressions: function literals, type assertions, etc.
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			info.Name = fn.Name

		case *ast.SelectorExpr:
			info.Name = exprToString(fn.X) + "." + fn.Sel.Name

		default:
			info.Name = "<expr>"
		}

		calls = append(calls, info)
		return true
	})

	return calls
}

// CallInfo holds metadata about a function call.
type CallInfo struct {
	Name     string
	Position token.Position
	NumArgs  int
}

// FindTODOs returns all TODO comments in the file.
//
// LEARN: Comments are stored separately from the AST.
// Use file.Comments to access them.
func FindTODOs(file *ast.File, fset *token.FileSet) []TODOInfo {
	var todos []TODOInfo

	for _, cg := range file.Comments {
		for _, c := range cg.List {
			text := c.Text
			// Check for TODO, FIXME, HACK, XXX patterns
			upper := strings.ToUpper(text)
			for _, marker := range []string{"TODO", "FIXME", "HACK", "XXX"} {
				if strings.Contains(upper, marker) {
					todos = append(todos, TODOInfo{
						Text:     strings.TrimSpace(strings.TrimPrefix(text, "//")),
						Position: fset.Position(c.Pos()),
						Marker:   marker,
					})
					break
				}
			}
		}
	}

	return todos
}

// TODOInfo holds metadata about a TODO comment.
type TODOInfo struct {
	Text     string
	Position token.Position
	Marker   string // TODO, FIXME, HACK, or XXX
}

// WalkDir analyzes all Go files in a directory tree.
//
// LEARN: Useful for project-wide analysis. Handles multiple packages
// and test files gracefully.
func WalkDir(root string, fn func(analysis *FileAnalysis) error) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files and test files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		file, fset, err := ParseFile(path)
		if err != nil {
			return err
		}

		analysis := AnalyzeFile(file, fset)
		return fn(analysis)
	})
}

// Helper functions

// countFields counts the total number of names in a field list.
// LEARN: A field list like (a, b int, c string) has 3 names but 2 fields
func countFields(fl *ast.FieldList) int {
	count := 0
	for _, f := range fl.List {
		if len(f.Names) == 0 {
			count++ // Anonymous parameter
		} else {
			count += len(f.Names)
		}
	}
	return count
}

// exprToString converts an expression to its string representation.
// LEARN: This is a simplified version - a full implementation would
// use go/printer or go/format for accurate output.
func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name

	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name

	case *ast.StarExpr:
		return "*" + exprToString(e.X)

	case *ast.ArrayType:
		if e.Len == nil {
			return "[]" + exprToString(e.Elt)
		}
		return "[...]" + exprToString(e.Elt)

	case *ast.MapType:
		return "map[" + exprToString(e.Key) + "]" + exprToString(e.Value)

	case *ast.ChanType:
		return "chan " + exprToString(e.Value)

	case *ast.FuncType:
		return "func" + fieldListToString(e.Params) + fieldListToString(e.Results)

	case *ast.InterfaceType:
		return "interface{}"

	case *ast.Ellipsis:
		return "..." + exprToString(e.Elt)

	default:
		return fmt.Sprintf("<%T>", expr)
	}
}

// fieldListToString converts a field list to a parenthesized string.
func fieldListToString(fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return ""
	}

	var parts []string
	for _, f := range fl.List {
		typ := exprToString(f.Type)
		if len(f.Names) > 0 {
			var names []string
			for _, n := range f.Names {
				names = append(names, n.Name)
			}
			parts = append(parts, strings.Join(names, ", ")+" "+typ)
		} else {
			parts = append(parts, typ)
		}
	}

	return "(" + strings.Join(parts, ", ") + ")"
}
