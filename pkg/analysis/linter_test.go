package analysis

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

// LEARN: Testing linter rules requires carefully crafted test cases.
// Each test case should trigger (or not trigger) specific rules.

func TestContextFirstRule(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantDiag bool
	}{
		{
			name: "context first - ok",
			src: `package main

import "context"

func doWork(ctx context.Context, x int) error {
	return nil
}
`,
			wantDiag: false,
		},
		{
			name: "context not first - bad",
			src: `package main

import "context"

func doWork(x int, ctx context.Context) error {
	return nil
}
`,
			wantDiag: true,
		},
		{
			name: "no context - ok",
			src: `package main

func doWork(x int) error {
	return nil
}
`,
			wantDiag: false,
		},
		{
			name: "single context param - ok",
			src: `package main

import "context"

func doWork(ctx context.Context) error {
	return nil
}
`,
			wantDiag: false,
		},
		{
			name: "context third param - bad",
			src: `package main

import "context"

func doWork(a int, b string, ctx context.Context) error {
	return nil
}
`,
			wantDiag: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file, fset, err := ParseSource(tc.src)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			linter := NewLinter(fset)
			diagnostics := linter.RunRules(file, "context-first")

			if tc.wantDiag && len(diagnostics) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tc.wantDiag && len(diagnostics) > 0 {
				t.Errorf("expected no diagnostics, got: %v", diagnostics)
			}
		})
	}
}

func TestEmptyErrorHandlerRule(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantDiag bool
	}{
		{
			name: "proper error handling - ok",
			src: `package main

func work() error {
	err := doSomething()
	if err != nil {
		return err
	}
	return nil
}

func doSomething() error { return nil }
`,
			wantDiag: false,
		},
		{
			name: "error wrapped - ok",
			src: `package main

import "fmt"

func work() error {
	err := doSomething()
	if err != nil {
		return fmt.Errorf("work failed: %w", err)
	}
	return nil
}

func doSomething() error { return nil }
`,
			wantDiag: false,
		},
		{
			name: "error dropped - bad",
			src: `package main

func work() *Result {
	err := doSomething()
	if err != nil {
		return nil
	}
	return &Result{}
}

func doSomething() error { return nil }
type Result struct{}
`,
			wantDiag: true,
		},
		{
			name: "multiple return values - ok",
			src: `package main

func work() (int, error) {
	err := doSomething()
	if err != nil {
		return 0, err
	}
	return 42, nil
}

func doSomething() error { return nil }
`,
			wantDiag: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file, fset, err := ParseSource(tc.src)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			linter := NewLinter(fset)
			diagnostics := linter.RunRules(file, "empty-error-handler")

			if tc.wantDiag && len(diagnostics) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tc.wantDiag && len(diagnostics) > 0 {
				t.Errorf("expected no diagnostics, got: %v", diagnostics)
			}
		})
	}
}

func TestTODOWithoutIssueRule(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantDiag bool
	}{
		{
			name: "TODO with issue - ok",
			src: `package main

// TODO(#123): implement this feature
func notYet() {}
`,
			wantDiag: false,
		},
		{
			name: "TODO with JIRA - ok",
			src: `package main

// TODO(PROJ-456): fix this bug
func buggy() {}
`,
			wantDiag: false,
		},
		{
			name: "TODO without issue - bad",
			src: `package main

// TODO: implement later
func notYet() {}
`,
			wantDiag: true,
		},
		{
			name: "FIXME without issue - bad",
			src: `package main

// FIXME: this is broken
func broken() {}
`,
			wantDiag: true,
		},
		{
			name: "regular comment - ok",
			src: `package main

// This function does important work
func important() {}
`,
			wantDiag: false,
		},
		{
			name: "TODO with issue reference in text - ok",
			src: `package main

// TODO: see issue #789 for details
func willDo() {}
`,
			wantDiag: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file, fset, err := ParseSource(tc.src)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			linter := NewLinter(fset)
			diagnostics := linter.Run(file)

			// Filter to just TODO rule
			var todoDiags []Diagnostic
			for _, d := range diagnostics {
				if d.Rule == "todo-with-issue" {
					todoDiags = append(todoDiags, d)
				}
			}

			if tc.wantDiag && len(todoDiags) == 0 {
				t.Error("expected TODO diagnostic, got none")
			}
			if !tc.wantDiag && len(todoDiags) > 0 {
				t.Errorf("expected no TODO diagnostics, got: %v", todoDiags)
			}
		})
	}
}

func TestNakedReturnRule(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantDiag bool
	}{
		{
			name: "short function naked return - ok",
			src: `package main

func short() (result int) {
	result = 42
	return
}
`,
			wantDiag: false,
		},
		{
			name: "long function naked return - bad",
			src: `package main

func long() (result int) {
	// line 1
	// line 2
	// line 3
	// line 4
	// line 5
	// line 6
	// line 7
	// line 8
	// line 9
	// line 10
	// line 11
	result = 42
	return
}
`,
			wantDiag: true,
		},
		{
			name: "explicit return - ok",
			src: `package main

func explicit() (result int) {
	// line 1
	// line 2
	// line 3
	// line 4
	// line 5
	// line 6
	// line 7
	// line 8
	// line 9
	// line 10
	// line 11
	result = 42
	return result
}
`,
			wantDiag: false,
		},
		{
			name: "no named returns - ok",
			src: `package main

func noNamed() int {
	// line 1
	// line 2
	// line 3
	// line 4
	// line 5
	// line 6
	// line 7
	// line 8
	// line 9
	// line 10
	// line 11
	return 42
}
`,
			wantDiag: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file, fset, err := ParseSource(tc.src)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			linter := NewLinter(fset)
			diagnostics := linter.RunRules(file, "naked-return")

			if tc.wantDiag && len(diagnostics) == 0 {
				t.Error("expected naked-return diagnostic, got none")
			}
			if !tc.wantDiag && len(diagnostics) > 0 {
				t.Errorf("expected no naked-return diagnostics, got: %v", diagnostics)
			}
		})
	}
}

func TestLinter_AllRules(t *testing.T) {
	// Source with multiple issues
	src := `package main

import "context"

// TODO: fix this
func badFunc(x int, ctx context.Context) error {
	err := doSomething()
	if err != nil {
		return nil
	}
	return nil
}

func doSomething() error { return nil }
`

	file, fset, err := ParseSource(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	linter := NewLinter(fset)
	diagnostics := linter.Run(file)

	// Should find at least:
	// 1. context-first: context is second param
	// 2. todo-with-issue: TODO without issue ref
	// 3. empty-error-handler: error checked but returns nil

	rules := make(map[string]int)
	for _, d := range diagnostics {
		rules[d.Rule]++
		t.Logf("%s", d)
	}

	if rules["context-first"] == 0 {
		t.Error("expected context-first diagnostic")
	}
	if rules["todo-with-issue"] == 0 {
		t.Error("expected todo-with-issue diagnostic")
	}
	if rules["empty-error-handler"] == 0 {
		t.Error("expected empty-error-handler diagnostic")
	}
}

func TestDiagnostic_String(t *testing.T) {
	d := Diagnostic{
		Position: token.Position{
			Filename: "test.go",
			Line:     10,
			Column:   5,
		},
		Severity: SeverityWarning,
		Rule:     "test-rule",
		Message:  "test message",
	}

	s := d.String()

	if !strings.Contains(s, "test.go:10:5") {
		t.Errorf("missing position in: %s", s)
	}
	if !strings.Contains(s, "warning") {
		t.Errorf("missing severity in: %s", s)
	}
	if !strings.Contains(s, "test-rule") {
		t.Errorf("missing rule in: %s", s)
	}
	if !strings.Contains(s, "test message") {
		t.Errorf("missing message in: %s", s)
	}
}

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		sev  Severity
		want string
	}{
		{SeverityHint, "hint"},
		{SeverityWarning, "warning"},
		{SeverityError, "error"},
		{Severity(99), "unknown"},
	}

	for _, tc := range tests {
		if got := tc.sev.String(); got != tc.want {
			t.Errorf("Severity(%d).String() = %q, want %q", tc.sev, got, tc.want)
		}
	}
}

func TestLinter_AddRule(t *testing.T) {
	file, fset, err := ParseSource(`package main

func main() {}
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	linter := NewLinter(fset)
	initialRules := len(linter.rules)

	// Add custom rule
	linter.AddRule(&customTestRule{})

	if len(linter.rules) != initialRules+1 {
		t.Error("rule was not added")
	}

	diagnostics := linter.Run(file)

	// Custom rule should fire
	var found bool
	for _, d := range diagnostics {
		if d.Rule == "custom-test" {
			found = true
			break
		}
	}

	if !found {
		t.Error("custom rule did not produce diagnostic")
	}
}

// customTestRule is a simple rule for testing
type customTestRule struct{}

func (r *customTestRule) Name() string        { return "custom-test" }
func (r *customTestRule) Description() string { return "test rule" }

func (r *customTestRule) Check(node ast.Node, ctx *LintContext) []Diagnostic {
	// Fire on every function declaration
	if _, ok := node.(*ast.FuncDecl); ok {
		return []Diagnostic{{
			Severity: SeverityHint,
			Message:  "found a function",
		}}
	}
	return nil
}
