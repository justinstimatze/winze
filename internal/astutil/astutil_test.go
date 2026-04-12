package astutil

import (
	"go/ast"
	"go/token"
	"os"
	"strconv"
	"testing"
)

func stringLit(s string) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(s)}
}

func TestGoFileFilter(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"foo.go", true},
		{"schema.go", true},
		{"readme.txt", false},
		{"main", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fi := fakeFileInfo{name: tc.name}
			got := GoFileFilter(fi)
			if got != tc.want {
				t.Errorf("GoFileFilter(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

type fakeFileInfo struct {
	name string
	os.FileInfo
}

func (f fakeFileInfo) Name() string { return f.name }

func TestIsInfraFile(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"schema.go", true},
		{"roles.go", true},
		{"predicates.go", true},
		{"design_roles.go", true},
		{"corpus_test.go", true},
		{"tunguska.go", false},
		{"main.go", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsInfraFile(tc.name)
			if got != tc.want {
				t.Errorf("IsInfraFile(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestResolveStringExpr(t *testing.T) {
	cases := []struct {
		name string
		expr ast.Expr
		want string
	}{
		{"simple string", stringLit("hello"), "hello"},
		{"concatenation", &ast.BinaryExpr{
			Op: token.ADD,
			X:  stringLit("foo"),
			Y:  stringLit("bar"),
		}, "foobar"},
		{"nil expr", nil, ""},
		{"int literal", &ast.BasicLit{Kind: token.INT, Value: "42"}, ""},
		{"non-ADD binary", &ast.BinaryExpr{
			Op: token.SUB,
			X:  stringLit("a"),
			Y:  stringLit("b"),
		}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveStringExpr(tc.expr)
			if got != tc.want {
				t.Errorf("ResolveStringExpr = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveStringExpr_DepthLimit(t *testing.T) {
	// Build a 200-level nested BinaryExpr — should hit the depth limit
	var expr ast.Expr = stringLit("leaf")
	for i := 0; i < 200; i++ {
		expr = &ast.BinaryExpr{Op: token.ADD, X: stringLit(""), Y: expr}
	}
	got := ResolveStringExpr(expr)
	// Should return partial or empty result due to depth cutoff, not panic
	if len(got) > 200 {
		t.Errorf("ResolveStringExpr on depth-200 tree returned %d chars, expected truncation", len(got))
	}
}

func TestUnquote(t *testing.T) {
	got := Unquote(stringLit("test"))
	if got != "test" {
		t.Errorf("Unquote = %q, want %q", got, "test")
	}
}

func TestCompositeTypeName(t *testing.T) {
	cases := []struct {
		name string
		cl   *ast.CompositeLit
		want string
	}{
		{"simple ident", &ast.CompositeLit{
			Type: &ast.Ident{Name: "Foo"},
		}, "Foo"},
		{"generic index", &ast.CompositeLit{
			Type: &ast.IndexExpr{X: &ast.Ident{Name: "BinaryRelation"}},
		}, "BinaryRelation"},
		{"selector (unsupported)", &ast.CompositeLit{
			Type: &ast.SelectorExpr{X: &ast.Ident{Name: "pkg"}, Sel: &ast.Ident{Name: "Type"}},
		}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CompositeTypeName(tc.cl)
			if got != tc.want {
				t.Errorf("CompositeTypeName = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractStringField(t *testing.T) {
	cases := []struct {
		name  string
		cl    *ast.CompositeLit
		field string
		want  string
	}{
		{"direct field", &ast.CompositeLit{
			Elts: []ast.Expr{
				&ast.KeyValueExpr{
					Key:   &ast.Ident{Name: "Brief"},
					Value: stringLit("test brief"),
				},
			},
		}, "Brief", "test brief"},
		{"nested entity", &ast.CompositeLit{
			Elts: []ast.Expr{
				&ast.UnaryExpr{
					Op: token.AND,
					X: &ast.CompositeLit{
						Elts: []ast.Expr{
							&ast.KeyValueExpr{
								Key:   &ast.Ident{Name: "Name"},
								Value: stringLit("nested name"),
							},
						},
					},
				},
			},
		}, "Name", "nested name"},
		{"missing field", &ast.CompositeLit{
			Elts: []ast.Expr{
				&ast.KeyValueExpr{
					Key:   &ast.Ident{Name: "Other"},
					Value: stringLit("value"),
				},
			},
		}, "Brief", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractStringField(tc.cl, tc.field)
			if got != tc.want {
				t.Errorf("ExtractStringField(%q) = %q, want %q", tc.field, got, tc.want)
			}
		})
	}
}

func TestExtractEntityBrief(t *testing.T) {
	// Direct Brief field
	cl := &ast.CompositeLit{
		Elts: []ast.Expr{
			&ast.KeyValueExpr{
				Key:   &ast.Ident{Name: "Brief"},
				Value: stringLit("philosopher"),
			},
		},
	}
	got := ExtractEntityBrief(cl)
	if got != "philosopher" {
		t.Errorf("ExtractEntityBrief (direct) = %q, want %q", got, "philosopher")
	}

	// Nested: Person{&Entity{Brief: "..."}}
	nested := &ast.CompositeLit{
		Elts: []ast.Expr{
			&ast.UnaryExpr{
				Op: token.AND,
				X: &ast.CompositeLit{
					Elts: []ast.Expr{
						&ast.KeyValueExpr{
							Key:   &ast.Ident{Name: "Brief"},
							Value: stringLit("nested brief"),
						},
					},
				},
			},
		},
	}
	got = ExtractEntityBrief(nested)
	if got != "nested brief" {
		t.Errorf("ExtractEntityBrief (nested) = %q, want %q", got, "nested brief")
	}
}

func TestExprIdent(t *testing.T) {
	cases := []struct {
		name string
		expr ast.Expr
		want string
	}{
		{"ident", &ast.Ident{Name: "Foo"}, "Foo"},
		{"selector", &ast.SelectorExpr{
			X:   &ast.Ident{Name: "pkg"},
			Sel: &ast.Ident{Name: "Type"},
		}, "pkg.Type"},
		{"composite", &ast.CompositeLit{
			Type: &ast.Ident{Name: "Bar"},
		}, "Bar"},
		{"nil", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExprIdent(tc.expr)
			if got != tc.want {
				t.Errorf("ExprIdent = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractSubjectObject(t *testing.T) {
	cl := &ast.CompositeLit{
		Elts: []ast.Expr{
			&ast.KeyValueExpr{
				Key:   &ast.Ident{Name: "Subject"},
				Value: &ast.Ident{Name: "Alice"},
			},
			&ast.KeyValueExpr{
				Key:   &ast.Ident{Name: "Object"},
				Value: &ast.Ident{Name: "Bob"},
			},
		},
	}
	subj, obj := ExtractSubjectObject(cl)
	if subj != "Alice" {
		t.Errorf("Subject = %q, want %q", subj, "Alice")
	}
	if obj != "Bob" {
		t.Errorf("Object = %q, want %q", obj, "Bob")
	}

	// Missing Object (unary claim)
	unary := &ast.CompositeLit{
		Elts: []ast.Expr{
			&ast.KeyValueExpr{
				Key:   &ast.Ident{Name: "Subject"},
				Value: &ast.Ident{Name: "Concept"},
			},
		},
	}
	subj, obj = ExtractSubjectObject(unary)
	if subj != "Concept" {
		t.Errorf("Subject = %q, want %q", subj, "Concept")
	}
	if obj != "" {
		t.Errorf("Object = %q, want empty", obj)
	}
}
