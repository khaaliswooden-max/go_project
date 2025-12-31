# Phase 6: Static Analysis - Learning Guide

This document covers Go static analysis techniques: the `go/ast` package for parsing, `go/types` for type checking, custom linter development, and code generation. Phase 6 builds on all previous phases to enable powerful compile-time tooling.

---

## Prerequisites

Before starting Phase 6, ensure you've completed:
- [x] Phase 1: Foundations (interfaces, error handling, testing)
- [x] Phase 2: Concurrency (goroutines, channels, sync primitives)
- [x] Phase 3: Generics (type parameters, generic data structures)
- [x] Phase 4: Performance (profiling, benchmarks, memory pooling)
- [x] Phase 5: Systems Programming (unsafe, mmap, CGO)

---

## Why Static Analysis?

Static analysis examines code without executing it. This enables:
1. **Linters:** Find bugs before runtime
2. **Code Generation:** Automate boilerplate
3. **Refactoring Tools:** Automated code transformations
4. **Documentation:** Generate docs from code structure
5. **Security Scanning:** Find vulnerabilities statically

Go's standard library provides excellent tools for this in `go/ast`, `go/parser`, `go/types`, and `go/token`.

---

## 1. The `go/ast` Package

**Files:** `pkg/analysis/ast.go`, `pkg/analysis/ast_test.go`

### Concept

The `go/ast` package represents Go source code as an Abstract Syntax Tree (AST). Every Go construct has a corresponding AST node type, from files down to individual expressions.

### What You'll Learn

```go
// LEARN: Parsing Go source code into an AST
import (
    "go/ast"
    "go/parser"
    "go/token"
)

func parseSource(src string) (*ast.File, *token.FileSet, error) {
    // LEARN: FileSet tracks position information
    // This allows mapping AST nodes back to line/column numbers
    fset := token.NewFileSet()
    
    // Parse the source string
    file, err := parser.ParseFile(fset, "example.go", src, parser.ParseComments)
    if err != nil {
        return nil, nil, fmt.Errorf("parse error: %w", err)
    }
    
    return file, fset, nil
}
```

### AST Node Types

```go
// LEARN: Key AST node types you'll work with

// Declarations
*ast.FuncDecl     // Function declaration
*ast.GenDecl      // Generic declaration (import, const, type, var)
*ast.TypeSpec     // Type specification in GenDecl

// Statements
*ast.AssignStmt   // Assignment: x = y, x := y
*ast.IfStmt       // If statement
*ast.ForStmt      // For loop
*ast.RangeStmt    // Range loop
*ast.ReturnStmt   // Return statement
*ast.ExprStmt     // Expression as statement

// Expressions
*ast.CallExpr     // Function call: foo(x, y)
*ast.SelectorExpr // Selector: pkg.Name, obj.Method
*ast.Ident        // Identifier: variable name
*ast.BasicLit     // Literal: "string", 42, 3.14
*ast.CompositeLit // Composite literal: []int{1, 2, 3}
```

### AST Traversal with ast.Inspect

```go
// LEARN: ast.Inspect walks the entire AST
// It calls your function for every node

func findFunctions(file *ast.File) []string {
    var functions []string
    
    ast.Inspect(file, func(n ast.Node) bool {
        // Type switch on node type
        switch x := n.(type) {
        case *ast.FuncDecl:
            functions = append(functions, x.Name.Name)
        }
        // Return true to continue visiting children
        return true
    })
    
    return functions
}
```

### AST Visitor Pattern

```go
// LEARN: ast.Visitor interface for stateful traversal
// More flexible than ast.Inspect

type functionVisitor struct {
    fset      *token.FileSet
    functions []FunctionInfo
}

type FunctionInfo struct {
    Name     string
    Position token.Position
    NumArgs  int
}

func (v *functionVisitor) Visit(node ast.Node) ast.Visitor {
    if fn, ok := node.(*ast.FuncDecl); ok {
        info := FunctionInfo{
            Name:     fn.Name.Name,
            Position: v.fset.Position(fn.Pos()),
        }
        if fn.Type.Params != nil {
            info.NumArgs = fn.Type.Params.NumFields()
        }
        v.functions = append(v.functions, info)
    }
    // Return self to continue traversal
    return v
}

func analyzeFunctions(fset *token.FileSet, file *ast.File) []FunctionInfo {
    v := &functionVisitor{fset: fset}
    ast.Walk(v, file)
    return v.functions
}
```

### Key Principles

- **Position Tracking:** Use `token.FileSet` to get line/column info
- **Immutable AST:** The AST represents source as-is; modifications create copies
- **Comments Preserved:** Parse with `parser.ParseComments` to include comments
- **Package Scope:** Multiple files can be parsed into one package

---

## 2. The `go/types` Package

**Files:** `pkg/analysis/checker.go`, `pkg/analysis/checker_test.go`

### Concept

While `go/ast` gives you syntax, `go/types` gives you semantics. It performs type checking and provides type information for every expression in the AST.

### What You'll Learn

```go
// LEARN: Type checking requires configuration
import (
    "go/types"
    "go/importer"
)

func typeCheck(fset *token.FileSet, files []*ast.File) (*types.Package, *types.Info, error) {
    // LEARN: types.Config controls type checking behavior
    conf := types.Config{
        // Importer resolves import paths to packages
        Importer: importer.Default(),
        
        // Error handler (optional - defaults to panicking)
        Error: func(err error) {
            // Collect errors instead of failing immediately
            fmt.Println("type error:", err)
        },
    }
    
    // LEARN: types.Info collects type information
    // Fill in the fields you need
    info := &types.Info{
        Types:      make(map[ast.Expr]types.TypeAndValue),
        Defs:       make(map[*ast.Ident]types.Object),
        Uses:       make(map[*ast.Ident]types.Object),
        Selections: make(map[*ast.SelectorExpr]*types.Selection),
    }
    
    // Perform type checking
    pkg, err := conf.Check("mypackage", fset, files, info)
    if err != nil {
        return nil, nil, err
    }
    
    return pkg, info, nil
}
```

### Using Type Information

```go
// LEARN: types.Info provides rich type data

func analyzeTypes(info *types.Info) {
    // Get type of any expression
    for expr, tv := range info.Types {
        if tv.IsValue() {
            fmt.Printf("Expression %T has type %s\n", expr, tv.Type)
        }
    }
    
    // Track where identifiers are defined
    for id, obj := range info.Defs {
        if obj != nil {
            fmt.Printf("Defined %s at %s\n", id.Name, obj.Pos())
        }
    }
    
    // Track where identifiers are used
    for id, obj := range info.Uses {
        fmt.Printf("Used %s (defined at %s)\n", id.Name, obj.Pos())
    }
}
```

### Type Comparisons and Assertions

```go
// LEARN: Working with types.Type

func analyzeType(t types.Type) {
    // Type switch for specific handling
    switch underlying := t.Underlying().(type) {
    case *types.Basic:
        fmt.Printf("Basic type: %s\n", underlying.Name())
        
    case *types.Struct:
        fmt.Printf("Struct with %d fields\n", underlying.NumFields())
        for i := 0; i < underlying.NumFields(); i++ {
            f := underlying.Field(i)
            fmt.Printf("  %s: %s\n", f.Name(), f.Type())
        }
        
    case *types.Slice:
        fmt.Printf("Slice of %s\n", underlying.Elem())
        
    case *types.Interface:
        fmt.Printf("Interface with %d methods\n", underlying.NumMethods())
        
    case *types.Signature:
        fmt.Printf("Function: %s\n", underlying.String())
    }
    
    // Check type relationships
    if types.Implements(t, someInterface) {
        fmt.Println("Type implements interface")
    }
    
    if types.AssignableTo(t1, t2) {
        fmt.Println("t1 is assignable to t2")
    }
}
```

### Key Principles

- **Semantic Analysis:** Types give meaning beyond syntax
- **Object Resolution:** Every identifier resolves to a types.Object
- **Type Identity:** Use types.Identical() for comparison
- **Underlying Types:** Named types have underlying basic types

---

## 3. Building a Custom Linter

**Files:** `pkg/analysis/linter.go`, `pkg/analysis/linter_test.go`

### Concept

A linter combines AST traversal with type information to detect code patterns that are likely bugs or style violations.

### What You'll Learn

```go
// LEARN: Linter structure
// Combine all analysis components

type Linter struct {
    fset  *token.FileSet
    info  *types.Info
    rules []Rule
}

type Rule interface {
    Name() string
    Check(node ast.Node) []Diagnostic
}

type Diagnostic struct {
    Pos      token.Position
    Severity Severity
    Message  string
    Rule     string
}

type Severity int

const (
    SeverityWarning Severity = iota
    SeverityError
)
```

### Example Rules

```go
// LEARN: Rule 1 - Find empty error checks
// Pattern: if err != nil { return nil } (drops the error!)

type EmptyErrorCheck struct{}

func (r *EmptyErrorCheck) Name() string { return "empty-error-check" }

func (r *EmptyErrorCheck) Check(node ast.Node) []Diagnostic {
    ifStmt, ok := node.(*ast.IfStmt)
    if !ok {
        return nil
    }
    
    // Check if condition is: err != nil
    binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr)
    if !ok || binExpr.Op != token.NEQ {
        return nil
    }
    
    ident, ok := binExpr.X.(*ast.Ident)
    if !ok || ident.Name != "err" {
        return nil
    }
    
    // Check if body is: return nil
    if len(ifStmt.Body.List) == 1 {
        if ret, ok := ifStmt.Body.List[0].(*ast.ReturnStmt); ok {
            if len(ret.Results) == 1 {
                if id, ok := ret.Results[0].(*ast.Ident); ok && id.Name == "nil" {
                    return []Diagnostic{{
                        Severity: SeverityWarning,
                        Message:  "error checked but not returned or wrapped",
                        Rule:     r.Name(),
                    }}
                }
            }
        }
    }
    
    return nil
}
```

```go
// LEARN: Rule 2 - Find unhandled errors
// Pattern: someFunc()  (return value ignored when func returns error)

type UnhandledError struct {
    info *types.Info
}

func (r *UnhandledError) Name() string { return "unhandled-error" }

func (r *UnhandledError) Check(node ast.Node) []Diagnostic {
    // Look for expression statements (not assignments)
    exprStmt, ok := node.(*ast.ExprStmt)
    if !ok {
        return nil
    }
    
    // Check if it's a function call
    call, ok := exprStmt.X.(*ast.CallExpr)
    if !ok {
        return nil
    }
    
    // Get the type of the call expression
    tv, ok := r.info.Types[call]
    if !ok {
        return nil
    }
    
    // Check if return type includes error
    if returnsError(tv.Type) {
        return []Diagnostic{{
            Severity: SeverityError,
            Message:  "function returns error but it is not handled",
            Rule:     r.Name(),
        }}
    }
    
    return nil
}

func returnsError(t types.Type) bool {
    // Check for single error return
    if types.Identical(t, types.Universe.Lookup("error").Type()) {
        return true
    }
    
    // Check for tuple containing error
    if tuple, ok := t.(*types.Tuple); ok {
        for i := 0; i < tuple.Len(); i++ {
            v := tuple.At(i)
            if types.Identical(v.Type(), types.Universe.Lookup("error").Type()) {
                return true
            }
        }
    }
    
    return false
}
```

```go
// LEARN: Rule 3 - Context as first parameter
// Pattern: func foo(x int, ctx context.Context) - ctx should be first!

type ContextFirst struct {
    info *types.Info
}

func (r *ContextFirst) Name() string { return "context-first" }

func (r *ContextFirst) Check(node ast.Node) []Diagnostic {
    fn, ok := node.(*ast.FuncDecl)
    if !ok || fn.Type.Params == nil {
        return nil
    }
    
    params := fn.Type.Params.List
    if len(params) < 2 {
        return nil
    }
    
    // Check if first param is NOT context.Context
    firstType := r.info.TypeOf(params[0].Type)
    if isContextType(firstType) {
        return nil  // Good: context is first
    }
    
    // Check if any other param IS context.Context
    for i := 1; i < len(params); i++ {
        paramType := r.info.TypeOf(params[i].Type)
        if isContextType(paramType) {
            return []Diagnostic{{
                Severity: SeverityWarning,
                Message:  "context.Context should be the first parameter",
                Rule:     r.Name(),
            }}
        }
    }
    
    return nil
}

func isContextType(t types.Type) bool {
    return t != nil && t.String() == "context.Context"
}
```

### Running the Linter

```go
// LEARN: Linter runner orchestrates analysis

func (l *Linter) Run(file *ast.File) []Diagnostic {
    var diagnostics []Diagnostic
    
    ast.Inspect(file, func(n ast.Node) bool {
        if n == nil {
            return true
        }
        
        for _, rule := range l.rules {
            if diags := rule.Check(n); len(diags) > 0 {
                // Add position information
                for i := range diags {
                    diags[i].Pos = l.fset.Position(n.Pos())
                }
                diagnostics = append(diagnostics, diags...)
            }
        }
        
        return true
    })
    
    return diagnostics
}
```

### Key Principles

- **Composable Rules:** Each rule checks one pattern
- **Position Reporting:** Always include file:line:column
- **Type-Aware:** Use types.Info for semantic analysis
- **Configurable Severity:** Warn vs error distinction

---

## 4. Code Generation

**Files:** `pkg/analysis/generator.go`, `pkg/analysis/generator_test.go`

### Concept

Code generation uses AST manipulation or templates to automatically produce Go code. This eliminates boilerplate and reduces errors.

### What You'll Learn

```go
// LEARN: Code generation with text/template
import (
    "text/template"
    "bytes"
)

const interfaceTemplate = `// Code generated by glr-gen. DO NOT EDIT.

package {{.Package}}

import (
{{- range .Imports}}
    "{{.}}"
{{- end}}
)

// {{.Name}}Mock is a mock implementation of {{.Name}}
type {{.Name}}Mock struct {
{{- range .Methods}}
    {{.Name}}Func func({{.ParamsString}}) {{.ResultsString}}
{{- end}}
}

{{range .Methods}}
func (m *{{$.Name}}Mock) {{.Name}}({{.ParamsString}}) {{.ResultsString}} {
    if m.{{.Name}}Func != nil {
        return m.{{.Name}}Func({{.ArgsString}})
    }
    {{.ZeroReturn}}
}
{{end}}
`

type InterfaceSpec struct {
    Package string
    Name    string
    Imports []string
    Methods []MethodSpec
}

type MethodSpec struct {
    Name    string
    Params  []ParamSpec
    Results []ParamSpec
}

func generateMock(spec InterfaceSpec) ([]byte, error) {
    tmpl, err := template.New("mock").Parse(interfaceTemplate)
    if err != nil {
        return nil, err
    }
    
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, spec); err != nil {
        return nil, err
    }
    
    return buf.Bytes(), nil
}
```

### AST-Based Code Generation

```go
// LEARN: Build AST nodes programmatically

import (
    "go/ast"
    "go/format"
    "go/token"
    "bytes"
)

// Create a simple function: func Name() string { return "value" }
func createSimpleFunc(name, returnValue string) *ast.FuncDecl {
    return &ast.FuncDecl{
        Name: ast.NewIdent(name),
        Type: &ast.FuncType{
            Results: &ast.FieldList{
                List: []*ast.Field{
                    {Type: ast.NewIdent("string")},
                },
            },
        },
        Body: &ast.BlockStmt{
            List: []ast.Stmt{
                &ast.ReturnStmt{
                    Results: []ast.Expr{
                        &ast.BasicLit{
                            Kind:  token.STRING,
                            Value: fmt.Sprintf(`"%s"`, returnValue),
                        },
                    },
                },
            },
        },
    }
}

// Generate formatted Go source from AST
func generateFromAST(file *ast.File) ([]byte, error) {
    var buf bytes.Buffer
    fset := token.NewFileSet()
    
    // format.Node produces formatted Go code
    if err := format.Node(&buf, fset, file); err != nil {
        return nil, err
    }
    
    return buf.Bytes(), nil
}
```

### Extract Interface from Type

```go
// LEARN: Generate interface from concrete type
// Useful for: creating test interfaces, refactoring

func extractInterface(pkg *types.Package, typeName string) (*InterfaceSpec, error) {
    obj := pkg.Scope().Lookup(typeName)
    if obj == nil {
        return nil, fmt.Errorf("type %s not found", typeName)
    }
    
    named, ok := obj.Type().(*types.Named)
    if !ok {
        return nil, fmt.Errorf("%s is not a named type", typeName)
    }
    
    spec := &InterfaceSpec{
        Package: pkg.Name(),
        Name:    typeName,
    }
    
    // Extract all methods
    for i := 0; i < named.NumMethods(); i++ {
        method := named.Method(i)
        if !method.Exported() {
            continue  // Skip unexported methods
        }
        
        sig := method.Type().(*types.Signature)
        spec.Methods = append(spec.Methods, methodSpecFromSignature(method.Name(), sig))
    }
    
    return spec, nil
}
```

### Key Principles

- **Template Clarity:** Use text/template for readable generation
- **AST for Precision:** Build AST when structure matters
- **Format Output:** Always use go/format for generated code
- **DO NOT EDIT:** Add generated code markers

---

## 5. Building Analysis Tools

**Files:** `cmd/glr-lint/main.go` (optional CLI tool)

### Concept

Combine all techniques into a standalone analysis tool that can be run as part of CI.

### What You'll Learn

```go
// LEARN: CLI structure for analysis tool

package main

import (
    "flag"
    "fmt"
    "go/parser"
    "go/token"
    "os"
    "path/filepath"
)

func main() {
    var dir string
    flag.StringVar(&dir, "dir", ".", "Directory to analyze")
    flag.Parse()
    
    fset := token.NewFileSet()
    
    // Parse all Go files in directory
    packages, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
        os.Exit(1)
    }
    
    linter := NewLinter(fset)
    var totalDiagnostics int
    
    for _, pkg := range packages {
        for filename, file := range pkg.Files {
            diagnostics := linter.Run(file)
            for _, d := range diagnostics {
                fmt.Printf("%s:%d:%d: %s: %s\n",
                    filename, d.Pos.Line, d.Pos.Column,
                    d.Rule, d.Message)
            }
            totalDiagnostics += len(diagnostics)
        }
    }
    
    if totalDiagnostics > 0 {
        os.Exit(1)
    }
}
```

### Using golang.org/x/tools/go/analysis

```go
// LEARN: The analysis framework provides standardized linter API
// Used by: staticcheck, golint, go vet

import "golang.org/x/tools/go/analysis"

var ContextFirstAnalyzer = &analysis.Analyzer{
    Name: "contextfirst",
    Doc:  "checks that context.Context is the first parameter",
    Run:  runContextFirst,
}

func runContextFirst(pass *analysis.Pass) (interface{}, error) {
    for _, file := range pass.Files {
        ast.Inspect(file, func(n ast.Node) bool {
            fn, ok := n.(*ast.FuncDecl)
            if !ok || fn.Type.Params == nil {
                return true
            }
            
            // Analysis logic here...
            // Use pass.Reportf() to report issues
            
            pass.Reportf(fn.Pos(), "context should be first parameter")
            return true
        })
    }
    return nil, nil
}
```

### Key Principles

- **Incremental:** Process file-by-file for large codebases
- **Configurable:** Allow disabling rules
- **CI Integration:** Exit non-zero on errors
- **Standard Format:** Match existing tools (golint, go vet)

---

## Summary: Phase 6 Competencies

After completing Phase 6, you can:

| Skill | Assessment |
|-------|------------|
| Parse Go code | Using go/parser into AST |
| Traverse AST | ast.Inspect, ast.Walk, ast.Visitor |
| Type check | Using go/types for semantic info |
| Build linter rules | Combining AST + types |
| Generate code | Templates and AST construction |
| Build CLI tools | Standalone analysis utilities |

---

## Exercises Checklist

- [ ] **Exercise 6.1:** AST Explorer
  - Implements: Function and type discovery tool
  - File: `pkg/analysis/ast.go`
  - Test: `pkg/analysis/ast_test.go`
  - Requirements:
    - Parse Go source files
    - Extract function declarations with positions
    - Find all struct definitions
    - List all imports

- [ ] **Exercise 6.2:** Custom Linter
  - Implements: Three linting rules
  - File: `pkg/analysis/linter.go`
  - Test: `pkg/analysis/linter_test.go`
  - Requirements:
    - Rule 1: context.Context should be first parameter
    - Rule 2: Error return values must be checked
    - Rule 3: Find TODO comments with no issue reference
    - Output diagnostics with position info

- [ ] **Exercise 6.3:** Mock Generator
  - Implements: Interface mock generator
  - File: `pkg/analysis/generator.go`
  - Test: `pkg/analysis/generator_test.go`
  - Requirements:
    - Extract methods from interface type
    - Generate mock struct with function fields
    - Format output with go/format
    - Add "Code generated" header

---

## Next Phase Preview

**Phase 7: Distributed Systems** will cover:
- Consensus algorithms (Raft basics)
- gRPC and protocol buffers
- Distributed tracing
- Service discovery

The static analysis skills from Phase 6 will help you understand how Go tooling validates distributed system contracts and generates RPC stubs.

