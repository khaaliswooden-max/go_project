package analysis

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"regexp"
	"strings"
)

// LEARN: A linter combines AST traversal with optional type information
// to detect code patterns that may indicate bugs or style issues.

// Severity indicates how serious a diagnostic is.
type Severity int

const (
	SeverityHint Severity = iota
	SeverityWarning
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityHint:
		return "hint"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}

// Diagnostic represents a single issue found by the linter.
type Diagnostic struct {
	Position token.Position // File, line, column
	Severity Severity       // How serious is this?
	Rule     string         // Which rule triggered this
	Message  string         // Human-readable description
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("%s:%d:%d: %s [%s]: %s",
		d.Position.Filename, d.Position.Line, d.Position.Column,
		d.Severity, d.Rule, d.Message)
}

// Rule defines a single lint check.
//
// LEARN: The Rule interface allows composable, testable lint checks.
// Each rule focuses on one specific pattern, making it easy to
// enable/disable individual checks.
type Rule interface {
	// Name returns the rule identifier (e.g., "context-first")
	Name() string

	// Description returns a human-readable explanation
	Description() string

	// Check examines a node and returns diagnostics (if any)
	Check(node ast.Node, ctx *LintContext) []Diagnostic
}

// LintContext provides context for lint rules.
//
// LEARN: Rules may need access to:
// - FileSet for position information
// - Type information for semantic analysis
// - Configuration for severity overrides
type LintContext struct {
	Fset *token.FileSet
	Info *types.Info // May be nil if type checking failed
	Pkg  *types.Package
}

// Linter coordinates running multiple rules over source code.
type Linter struct {
	fset  *token.FileSet
	rules []Rule
	info  *types.Info
	pkg   *types.Package
}

// NewLinter creates a linter with default rules.
//
// LEARN: Default rules represent common Go best practices.
// Users can customize by adding/removing rules.
func NewLinter(fset *token.FileSet) *Linter {
	l := &Linter{
		fset: fset,
		rules: []Rule{
			&ContextFirstRule{},
			&ErrorCheckRule{},
			&TODOWithoutIssueRule{},
			&EmptyErrorHandlerRule{},
			&NakedReturnRule{},
		},
	}
	return l
}

// AddRule adds a custom rule to the linter.
func (l *Linter) AddRule(r Rule) {
	l.rules = append(l.rules, r)
}

// SetTypeInfo provides type information for semantic analysis.
func (l *Linter) SetTypeInfo(info *types.Info, pkg *types.Package) {
	l.info = info
	l.pkg = pkg
}

// TypeCheck performs type checking on the given files.
//
// LEARN: Type checking is optional but enables more powerful analysis.
// Rules can work with just AST, but types give semantic information.
func (l *Linter) TypeCheck(files []*ast.File, pkgPath string) error {
	conf := types.Config{
		Importer: importer.Default(),
		Error: func(err error) {
			// Ignore type errors - we're linting, not compiling
		},
	}

	l.info = &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}

	pkg, _ := conf.Check(pkgPath, l.fset, files, l.info)
	l.pkg = pkg

	return nil
}

// Run executes all lint rules on the given file.
//
// LEARN: The linter walks the AST once, applying all rules at each node.
// This is more efficient than walking once per rule.
func (l *Linter) Run(file *ast.File) []Diagnostic {
	var diagnostics []Diagnostic

	ctx := &LintContext{
		Fset: l.fset,
		Info: l.info,
		Pkg:  l.pkg,
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		for _, rule := range l.rules {
			if diags := rule.Check(n, ctx); len(diags) > 0 {
				// Add position information
				for i := range diags {
					if diags[i].Position.Filename == "" {
						diags[i].Position = l.fset.Position(n.Pos())
					}
					if diags[i].Rule == "" {
						diags[i].Rule = rule.Name()
					}
				}
				diagnostics = append(diagnostics, diags...)
			}
		}

		return true
	})

	// Also check comments for TODO rule
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			for _, rule := range l.rules {
				// Create a synthetic node for comments
				// (comments aren't part of the regular AST walk)
				if todoRule, ok := rule.(*TODOWithoutIssueRule); ok {
					if diags := todoRule.CheckComment(c, ctx); len(diags) > 0 {
						diagnostics = append(diagnostics, diags...)
					}
				}
			}
		}
	}

	return diagnostics
}

// RunRules runs only the specified rules.
func (l *Linter) RunRules(file *ast.File, ruleNames ...string) []Diagnostic {
	// Filter rules
	enabledRules := make(map[string]bool)
	for _, name := range ruleNames {
		enabledRules[name] = true
	}

	var diagnostics []Diagnostic
	ctx := &LintContext{
		Fset: l.fset,
		Info: l.info,
		Pkg:  l.pkg,
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		for _, rule := range l.rules {
			if !enabledRules[rule.Name()] {
				continue
			}

			if diags := rule.Check(n, ctx); len(diags) > 0 {
				for i := range diags {
					if diags[i].Position.Filename == "" {
						diags[i].Position = l.fset.Position(n.Pos())
					}
					if diags[i].Rule == "" {
						diags[i].Rule = rule.Name()
					}
				}
				diagnostics = append(diagnostics, diags...)
			}
		}

		return true
	})

	return diagnostics
}

// ============================================================================
// Built-in Rules
// ============================================================================

// ContextFirstRule checks that context.Context is the first parameter.
//
// LEARN: Go convention is that context.Context should be the first parameter.
// This rule enforces that convention.
type ContextFirstRule struct{}

func (r *ContextFirstRule) Name() string { return "context-first" }

func (r *ContextFirstRule) Description() string {
	return "context.Context should be the first parameter of a function"
}

func (r *ContextFirstRule) Check(node ast.Node, ctx *LintContext) []Diagnostic {
	fn, ok := node.(*ast.FuncDecl)
	if !ok || fn.Type.Params == nil {
		return nil
	}

	params := fn.Type.Params.List
	if len(params) < 2 {
		return nil
	}

	// Check if first param is already context.Context
	if isContextParam(params[0]) {
		return nil
	}

	// Check if any other param is context.Context
	for i := 1; i < len(params); i++ {
		if isContextParam(params[i]) {
			return []Diagnostic{{
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("context.Context should be the first parameter in %s", fn.Name.Name),
			}}
		}
	}

	return nil
}

// isContextParam checks if a field is of type context.Context
func isContextParam(field *ast.Field) bool {
	sel, ok := field.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "context" && sel.Sel.Name == "Context"
}

// ErrorCheckRule checks that error return values are handled.
//
// LEARN: Ignoring errors is a common source of bugs in Go.
// This rule flags function calls where error returns are discarded.
type ErrorCheckRule struct{}

func (r *ErrorCheckRule) Name() string { return "error-check" }

func (r *ErrorCheckRule) Description() string {
	return "error return values must be checked or explicitly ignored with _"
}

func (r *ErrorCheckRule) Check(node ast.Node, ctx *LintContext) []Diagnostic {
	// Look for expression statements (function calls not assigned)
	exprStmt, ok := node.(*ast.ExprStmt)
	if !ok {
		return nil
	}

	call, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return nil
	}

	// Without type info, we can only check known error-returning functions
	// With type info, we can check the actual return types
	if ctx.Info != nil {
		tv, ok := ctx.Info.Types[call]
		if !ok {
			return nil
		}

		if hasErrorReturn(tv.Type) {
			callName := getCallName(call)
			return []Diagnostic{{
				Severity: SeverityError,
				Message:  fmt.Sprintf("error return value of %s is not checked", callName),
			}}
		}
	} else {
		// Fallback: check common error-returning function names
		callName := getCallName(call)
		if isLikelyErrorReturning(callName) {
			return []Diagnostic{{
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("%s likely returns an error that is not checked", callName),
			}}
		}
	}

	return nil
}

// hasErrorReturn checks if a type includes error in its returns
func hasErrorReturn(t types.Type) bool {
	if t == nil {
		return false
	}

	// Single error return
	if isErrorType(t) {
		return true
	}

	// Tuple with error
	if tuple, ok := t.(*types.Tuple); ok {
		for i := 0; i < tuple.Len(); i++ {
			if isErrorType(tuple.At(i).Type()) {
				return true
			}
		}
	}

	return false
}

// isErrorType checks if t is the error type
func isErrorType(t types.Type) bool {
	if t == nil {
		return false
	}
	// Check if it's the error interface
	return t.String() == "error"
}

// isLikelyErrorReturning guesses if a function returns an error
func isLikelyErrorReturning(name string) bool {
	// Common patterns that likely return errors
	patterns := []string{
		"Open", "Create", "Read", "Write", "Close",
		"Parse", "Marshal", "Unmarshal",
		"Dial", "Connect", "Send", "Receive",
		"Do", "Execute", "Run",
	}

	for _, p := range patterns {
		if strings.Contains(name, p) {
			return true
		}
	}
	return false
}

// getCallName extracts the function name from a call expression
func getCallName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		return exprToString(fn.X) + "." + fn.Sel.Name
	default:
		return "<call>"
	}
}

// TODOWithoutIssueRule checks that TODO comments reference an issue.
//
// LEARN: TODOs without issue references tend to be forgotten.
// This rule encourages linking TODOs to tracked issues.
type TODOWithoutIssueRule struct{}

func (r *TODOWithoutIssueRule) Name() string { return "todo-with-issue" }

func (r *TODOWithoutIssueRule) Description() string {
	return "TODO comments should reference an issue (e.g., TODO(#123): ...)"
}

func (r *TODOWithoutIssueRule) Check(node ast.Node, ctx *LintContext) []Diagnostic {
	// This rule works on comments, not AST nodes
	return nil
}

// issueRefPattern matches issue references like #123, GH-123, JIRA-123
var issueRefPattern = regexp.MustCompile(`(?i)(#\d+|[A-Z]+-\d+|issue\s*\d+)`)

// CheckComment checks a comment for TODO without issue reference
func (r *TODOWithoutIssueRule) CheckComment(c *ast.Comment, ctx *LintContext) []Diagnostic {
	text := c.Text

	// Check for TODO/FIXME/HACK/XXX markers
	upper := strings.ToUpper(text)
	markers := []string{"TODO", "FIXME", "HACK", "XXX"}

	var foundMarker string
	for _, marker := range markers {
		if strings.Contains(upper, marker) {
			foundMarker = marker
			break
		}
	}

	if foundMarker == "" {
		return nil
	}

	// Check if there's an issue reference
	if issueRefPattern.MatchString(text) {
		return nil
	}

	return []Diagnostic{{
		Position: ctx.Fset.Position(c.Pos()),
		Severity: SeverityHint,
		Rule:     r.Name(),
		Message:  fmt.Sprintf("%s comment should reference an issue (e.g., %s(#123): ...)", foundMarker, foundMarker),
	}}
}

// EmptyErrorHandlerRule checks for error checks that don't handle the error.
//
// LEARN: A common anti-pattern is `if err != nil { return nil }` which
// drops the error. This rule detects that pattern.
type EmptyErrorHandlerRule struct{}

func (r *EmptyErrorHandlerRule) Name() string { return "empty-error-handler" }

func (r *EmptyErrorHandlerRule) Description() string {
	return "error should be returned or wrapped, not silently discarded"
}

func (r *EmptyErrorHandlerRule) Check(node ast.Node, ctx *LintContext) []Diagnostic {
	ifStmt, ok := node.(*ast.IfStmt)
	if !ok {
		return nil
	}

	// Check if condition is: err != nil
	if !isErrorNilCheck(ifStmt.Cond) {
		return nil
	}

	// Check if body is just: return nil
	if isReturnNilOnly(ifStmt.Body) {
		return []Diagnostic{{
			Severity: SeverityWarning,
			Message:  "error is checked but not returned or wrapped",
		}}
	}

	return nil
}

// isErrorNilCheck checks if expr is "err != nil"
func isErrorNilCheck(expr ast.Expr) bool {
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return false
	}

	// Check left side is identifier named "err"
	ident, ok := bin.X.(*ast.Ident)
	if !ok || ident.Name != "err" {
		return false
	}

	// Check right side is nil
	nilIdent, ok := bin.Y.(*ast.Ident)
	return ok && nilIdent.Name == "nil"
}

// isReturnNilOnly checks if block is `{ return nil }`
func isReturnNilOnly(block *ast.BlockStmt) bool {
	if len(block.List) != 1 {
		return false
	}

	ret, ok := block.List[0].(*ast.ReturnStmt)
	if !ok {
		return false
	}

	// Single nil return
	if len(ret.Results) == 1 {
		if id, ok := ret.Results[0].(*ast.Ident); ok && id.Name == "nil" {
			return true
		}
	}

	return false
}

// NakedReturnRule checks for naked returns in functions with named results.
//
// LEARN: Naked returns (return without arguments when results are named)
// can make code harder to read. This rule encourages explicit returns.
type NakedReturnRule struct {
	MaxLines int // Only flag naked returns in functions longer than this
}

func (r *NakedReturnRule) Name() string { return "naked-return" }

func (r *NakedReturnRule) Description() string {
	return "avoid naked returns in functions with named return values"
}

func (r *NakedReturnRule) Check(node ast.Node, ctx *LintContext) []Diagnostic {
	fn, ok := node.(*ast.FuncDecl)
	if !ok || fn.Body == nil {
		return nil
	}

	// Check if function has named return values
	if !hasNamedResults(fn.Type.Results) {
		return nil
	}

	// Default: flag naked returns in functions > 10 lines
	maxLines := r.MaxLines
	if maxLines == 0 {
		maxLines = 10
	}

	// Calculate function length
	start := ctx.Fset.Position(fn.Body.Lbrace)
	end := ctx.Fset.Position(fn.Body.Rbrace)
	funcLines := end.Line - start.Line

	if funcLines <= maxLines {
		return nil // Short functions are OK
	}

	// Find naked returns
	var diagnostics []Diagnostic
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}

		if len(ret.Results) == 0 {
			diagnostics = append(diagnostics, Diagnostic{
				Position: ctx.Fset.Position(ret.Pos()),
				Severity: SeverityHint,
				Rule:     r.Name(),
				Message:  fmt.Sprintf("naked return in function %s with named results (function has %d lines)", fn.Name.Name, funcLines),
			})
		}

		return true
	})

	return diagnostics
}

// hasNamedResults checks if a function has named return values
func hasNamedResults(results *ast.FieldList) bool {
	if results == nil || len(results.List) == 0 {
		return false
	}

	for _, field := range results.List {
		if len(field.Names) > 0 {
			return true
		}
	}

	return false
}
