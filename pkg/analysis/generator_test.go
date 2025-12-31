package analysis

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

// LEARN: Testing code generation involves checking that:
// 1. Generated code compiles (valid Go syntax)
// 2. Generated code has correct structure
// 3. Edge cases are handled

func TestGenerateMock_Simple(t *testing.T) {
	spec := MockSpec{
		Package: "test",
		Name:    "Reader",
		Methods: []MockMethod{
			{
				Name: "Read",
				Params: []MockParam{
					{Name: "p", Type: "[]byte"},
				},
				Results: []MockParam{
					{Type: "int"},
					{Type: "error"},
				},
			},
		},
	}

	output, err := GenerateMock(spec)
	if err != nil {
		t.Fatalf("GenerateMock error: %v", err)
	}

	code := string(output)

	// Check package declaration
	if !strings.Contains(code, "package test") {
		t.Error("missing package declaration")
	}

	// Check mock struct
	if !strings.Contains(code, "type ReaderMock struct") {
		t.Error("missing mock struct")
	}

	// Check function field
	if !strings.Contains(code, "ReadFunc func(p []byte) (int, error)") {
		t.Error("missing ReadFunc field")
	}

	// Check method implementation
	if !strings.Contains(code, "func (m *ReaderMock) Read(p []byte) (int, error)") {
		t.Error("missing Read method")
	}

	// Check delegation
	if !strings.Contains(code, "if m.ReadFunc != nil") {
		t.Error("missing delegation check")
	}

	// Check generated header
	if !strings.Contains(code, "Code generated") {
		t.Error("missing generated header")
	}

	t.Logf("Generated code:\n%s", code)
}

func TestGenerateMock_MultipleMethodsNoImports(t *testing.T) {
	spec := MockSpec{
		Package: "service",
		Name:    "Cache",
		Methods: []MockMethod{
			{
				Name: "Get",
				Params: []MockParam{
					{Name: "key", Type: "string"},
				},
				Results: []MockParam{
					{Type: "string"},
					{Type: "bool"},
				},
			},
			{
				Name: "Set",
				Params: []MockParam{
					{Name: "key", Type: "string"},
					{Name: "value", Type: "string"},
				},
			},
			{
				Name: "Delete",
				Params: []MockParam{
					{Name: "key", Type: "string"},
				},
				Results: []MockParam{
					{Type: "bool"},
				},
			},
		},
	}

	output, err := GenerateMock(spec)
	if err != nil {
		t.Fatalf("GenerateMock error: %v", err)
	}

	code := string(output)

	// Check all methods exist
	if !strings.Contains(code, "func (m *CacheMock) Get") {
		t.Error("missing Get method")
	}
	if !strings.Contains(code, "func (m *CacheMock) Set") {
		t.Error("missing Set method")
	}
	if !strings.Contains(code, "func (m *CacheMock) Delete") {
		t.Error("missing Delete method")
	}

	// Check function fields
	if !strings.Contains(code, "GetFunc") {
		t.Error("missing GetFunc field")
	}
	if !strings.Contains(code, "SetFunc") {
		t.Error("missing SetFunc field")
	}
	if !strings.Contains(code, "DeleteFunc") {
		t.Error("missing DeleteFunc field")
	}
}

func TestGenerateMock_WithImports(t *testing.T) {
	spec := MockSpec{
		Package: "handler",
		Name:    "Handler",
		Imports: []string{"context", "net/http"},
		Methods: []MockMethod{
			{
				Name: "ServeHTTP",
				Params: []MockParam{
					{Name: "w", Type: "http.ResponseWriter"},
					{Name: "r", Type: "*http.Request"},
				},
			},
			{
				Name: "Shutdown",
				Params: []MockParam{
					{Name: "ctx", Type: "context.Context"},
				},
				Results: []MockParam{
					{Type: "error"},
				},
			},
		},
	}

	output, err := GenerateMock(spec)
	if err != nil {
		t.Fatalf("GenerateMock error: %v", err)
	}

	code := string(output)

	// Check imports
	if !strings.Contains(code, `"context"`) {
		t.Error("missing context import")
	}
	if !strings.Contains(code, `"net/http"`) {
		t.Error("missing net/http import")
	}
}

func TestGenerateMock_CustomMockName(t *testing.T) {
	spec := MockSpec{
		Package:  "test",
		Name:     "Service",
		MockName: "FakeService",
		Methods: []MockMethod{
			{
				Name:    "Do",
				Results: []MockParam{{Type: "error"}},
			},
		},
	}

	output, err := GenerateMock(spec)
	if err != nil {
		t.Fatalf("GenerateMock error: %v", err)
	}

	code := string(output)

	if !strings.Contains(code, "type FakeService struct") {
		t.Error("custom mock name not used")
	}
	if !strings.Contains(code, "func (m *FakeService)") {
		t.Error("custom mock name not used in method")
	}
}

func TestMockMethod_ParamsString(t *testing.T) {
	tests := []struct {
		name   string
		params []MockParam
		want   string
	}{
		{
			name:   "empty",
			params: nil,
			want:   "",
		},
		{
			name: "single named",
			params: []MockParam{
				{Name: "x", Type: "int"},
			},
			want: "x int",
		},
		{
			name: "multiple",
			params: []MockParam{
				{Name: "ctx", Type: "context.Context"},
				{Name: "id", Type: "string"},
			},
			want: "ctx context.Context, id string",
		},
		{
			name: "unnamed",
			params: []MockParam{
				{Type: "int"},
				{Type: "string"},
			},
			want: "int, string",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := MockMethod{Params: tc.params}
			if got := m.ParamsString(); got != tc.want {
				t.Errorf("ParamsString() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMockMethod_ResultsString(t *testing.T) {
	tests := []struct {
		name    string
		results []MockParam
		want    string
	}{
		{
			name:    "empty",
			results: nil,
			want:    "",
		},
		{
			name: "single",
			results: []MockParam{
				{Type: "error"},
			},
			want: "error",
		},
		{
			name: "multiple",
			results: []MockParam{
				{Type: "int"},
				{Type: "error"},
			},
			want: "(int, error)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := MockMethod{Results: tc.results}
			if got := m.ResultsString(); got != tc.want {
				t.Errorf("ResultsString() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMockMethod_ZeroReturn(t *testing.T) {
	tests := []struct {
		name    string
		results []MockParam
		want    string
	}{
		{
			name:    "no results",
			results: nil,
			want:    "return",
		},
		{
			name:    "error",
			results: []MockParam{{Type: "error"}},
			want:    "return nil",
		},
		{
			name:    "string",
			results: []MockParam{{Type: "string"}},
			want:    `return ""`,
		},
		{
			name:    "int",
			results: []MockParam{{Type: "int"}},
			want:    "return 0",
		},
		{
			name:    "bool",
			results: []MockParam{{Type: "bool"}},
			want:    "return false",
		},
		{
			name:    "pointer",
			results: []MockParam{{Type: "*Foo"}},
			want:    "return nil",
		},
		{
			name:    "slice",
			results: []MockParam{{Type: "[]int"}},
			want:    "return nil",
		},
		{
			name: "multiple",
			results: []MockParam{
				{Type: "int"},
				{Type: "error"},
			},
			want: "return 0, nil",
		},
		{
			name:    "struct",
			results: []MockParam{{Type: "MyStruct"}},
			want:    "return MyStruct{}",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := MockMethod{Results: tc.results}
			if got := m.ZeroReturn(); got != tc.want {
				t.Errorf("ZeroReturn() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractInterfaceSpec(t *testing.T) {
	src := `package example

import "context"

// Doer does things.
type Doer interface {
	Do(ctx context.Context, action string) error
	Status() (bool, string)
}
`

	file, fset, err := ParseSource(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	spec, err := ExtractInterfaceSpec(file, fset, "Doer")
	if err != nil {
		t.Fatalf("ExtractInterfaceSpec error: %v", err)
	}

	if spec.Name != "Doer" {
		t.Errorf("Name = %q, want Doer", spec.Name)
	}

	if spec.Package != "example" {
		t.Errorf("Package = %q, want example", spec.Package)
	}

	if len(spec.Methods) != 2 {
		t.Fatalf("got %d methods, want 2", len(spec.Methods))
	}

	// Check Do method
	do := spec.Methods[0]
	if do.Name != "Do" {
		t.Errorf("first method = %q, want Do", do.Name)
	}
	if len(do.Params) != 2 {
		t.Errorf("Do has %d params, want 2", len(do.Params))
	}
	if len(do.Results) != 1 {
		t.Errorf("Do has %d results, want 1", len(do.Results))
	}

	// Check Status method
	status := spec.Methods[1]
	if status.Name != "Status" {
		t.Errorf("second method = %q, want Status", status.Name)
	}
	if len(status.Params) != 0 {
		t.Errorf("Status has %d params, want 0", len(status.Params))
	}
	if len(status.Results) != 2 {
		t.Errorf("Status has %d results, want 2", len(status.Results))
	}
}

func TestExtractInterfaceSpec_NotFound(t *testing.T) {
	src := `package example

type NotAnInterface struct{}
`

	file, fset, err := ParseSource(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	_, err = ExtractInterfaceSpec(file, fset, "Missing")
	if err == nil {
		t.Error("expected error for missing interface")
	}
}

func TestCreateStructAST(t *testing.T) {
	fields := []FieldDef{
		{Name: "ID", Type: "int"},
		{Name: "Name", Type: "string", Tag: "`json:\"name\"`"},
	}

	spec := CreateStructAST("Person", fields)

	if spec.Name.Name != "Person" {
		t.Errorf("struct name = %q, want Person", spec.Name.Name)
	}

	st, ok := spec.Type.(*ast.StructType)
	if !ok {
		t.Fatal("type is not *ast.StructType")
	}

	if len(st.Fields.List) != 2 {
		t.Errorf("got %d fields, want 2", len(st.Fields.List))
	}
}

func TestCreateFunctionAST(t *testing.T) {
	params := []ParamDef{
		{Name: "x", Type: "int"},
		{Name: "y", Type: "int"},
	}
	results := []ParamDef{
		{Type: "int"},
	}
	body := []ast.Stmt{
		CreateReturnStmt(
			&ast.BinaryExpr{
				X:  ast.NewIdent("x"),
				Op: token.ADD,
				Y:  ast.NewIdent("y"),
			},
		),
	}

	fn := CreateFunctionAST("Add", params, results, body)

	if fn.Name.Name != "Add" {
		t.Errorf("function name = %q, want Add", fn.Name.Name)
	}

	if len(fn.Type.Params.List) != 2 {
		t.Errorf("got %d params, want 2", len(fn.Type.Params.List))
	}

	if len(fn.Type.Results.List) != 1 {
		t.Errorf("got %d results, want 1", len(fn.Type.Results.List))
	}
}

func TestGenerateFromAST(t *testing.T) {
	// Create a simple file
	file := CreateFileAST("example", []string{"fmt"})

	// Add a function
	fn := CreateFunctionAST(
		"Hello",
		nil,
		nil,
		[]ast.Stmt{
			&ast.ExprStmt{
				X: CreateSelectorCall("fmt", "Println", CreateStringLit("Hello, World!")),
			},
		},
	)

	file.Decls = append(file.Decls, fn)

	output, err := GenerateFromAST(file)
	if err != nil {
		t.Fatalf("GenerateFromAST error: %v", err)
	}

	code := string(output)

	if !strings.Contains(code, "package example") {
		t.Error("missing package")
	}
	if !strings.Contains(code, `"fmt"`) {
		t.Error("missing import")
	}
	if !strings.Contains(code, "func Hello()") {
		t.Error("missing function")
	}
	if !strings.Contains(code, `fmt.Println("Hello, World!")`) {
		t.Error("missing print call")
	}

	t.Logf("Generated code:\n%s", code)
}

func TestCreateLiterals(t *testing.T) {
	// Test string literal
	strLit := CreateStringLit("hello")
	if strLit.Value != `"hello"` {
		t.Errorf("string literal = %q, want %q", strLit.Value, `"hello"`)
	}

	// Test int literal
	intLit := CreateIntLit(42)
	if intLit.Value != "42" {
		t.Errorf("int literal = %q, want 42", intLit.Value)
	}
}

func TestGenerateGetter(t *testing.T) {
	getter := GenerateGetter("Person", "Name", "string")

	if getter.Name.Name != "Name" {
		t.Errorf("getter name = %q, want Name", getter.Name.Name)
	}

	// Check it has a receiver
	if getter.Recv == nil || len(getter.Recv.List) != 1 {
		t.Error("getter should have receiver")
	}

	// Check return type
	if getter.Type.Results == nil || len(getter.Type.Results.List) != 1 {
		t.Error("getter should have return type")
	}
}

func TestGenerateSetter(t *testing.T) {
	setter := GenerateSetter("Person", "Name", "string")

	if setter.Name.Name != "SetName" {
		t.Errorf("setter name = %q, want SetName", setter.Name.Name)
	}

	// Check it has a receiver
	if setter.Recv == nil || len(setter.Recv.List) != 1 {
		t.Error("setter should have receiver")
	}

	// Check parameter
	if setter.Type.Params == nil || len(setter.Type.Params.List) != 1 {
		t.Error("setter should have parameter")
	}
}
