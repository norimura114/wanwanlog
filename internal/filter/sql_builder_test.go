package filter

import (
	"testing"
)

func TestSQLBuilderBuildNumberAttributeGte(t *testing.T) {
	parser, err := NewParser(`@http.status >= 400`)
	if err != nil {
		t.Fatal(err)
	}

	expr, err := parser.Parse()
	if err != nil {
		t.Fatal(err)
	}

	builder := NewSQLBuilder()

	query, err := builder.Build(expr)
	if err != nil {
		t.Fatal(err)
	}

	if query.Where == "" {
		t.Fatal("expected where clause")
	}

	if len(query.Args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(query.Args))
	}

	if query.Args[0] != "$.http.status" {
		t.Fatalf("expected json path, got %v", query.Args[0])
	}

	if query.Args[1] != "$.http.status" {
		t.Fatalf("expected json path, got %v", query.Args[1])
	}

	if query.Args[2] != 400.0 {
		t.Fatalf("expected number arg 400.0, got %v", query.Args[2])
	}
}

func TestSQLBuilderBuildStringAttribute(t *testing.T) {
	parser, err := NewParser(`@trace.id:"400"`)
	if err != nil {
		t.Fatal(err)
	}

	expr, err := parser.Parse()
	if err != nil {
		t.Fatal(err)
	}

	builder := NewSQLBuilder()

	query, err := builder.Build(expr)
	if err != nil {
		t.Fatal(err)
	}

	if len(query.Args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(query.Args))
	}

	if query.Args[2] != "400" {
		t.Fatalf("expected string arg 400, got %v", query.Args[2])
	}
}

func TestParserRejectStringRangeComparison(t *testing.T) {
	parser, err := NewParser(`@env >= prod`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = parser.Parse()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParserParsesBoolAttribute(t *testing.T) {
	parser, err := NewParser(`@active:true`)
	if err != nil {
		t.Fatal(err)
	}

	expr, err := parser.Parse()
	if err != nil {
		t.Fatal(err)
	}

	attr, ok := expr.(AttributeExpr)
	if !ok {
		t.Fatalf("expected AttributeExpr, got %T", expr)
	}

	if attr.Value.Kind != LiteralBool {
		t.Fatalf("expected bool literal, got %v", attr.Value.Kind)
	}

	if !attr.Value.Bool {
		t.Fatal("expected true")
	}
}
