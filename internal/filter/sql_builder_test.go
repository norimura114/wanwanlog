package filter

import (
	"reflect"
	"testing"
)

func TestBuildFieldExpr(t *testing.T) {
	builder := NewSQLBuilder()

	query, err := builder.Build(FieldExpr{
		Field: "service",
		Value: "api",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := SQLQuery{
		Where: "service = ?",
		Args:  []any{"api"},
	}

	if !reflect.DeepEqual(query, expected) {
		t.Fatalf("expected %#v, got %#v", expected, query)
	}
}

func TestBuildAttributeExpr(t *testing.T) {
	builder := NewSQLBuilder()

	query, err := builder.Build(AttributeExpr{
		Key:   "user.id",
		Value: "123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := SQLQuery{
		Where: "json_extract(attributes, ?) = ?",
		Args:  []any{"$.user.id", "123"},
	}

	if !reflect.DeepEqual(query, expected) {
		t.Fatalf("expected %#v, got %#v", expected, query)
	}
}

func TestBuildKeywordExpr(t *testing.T) {
	builder := NewSQLBuilder()

	query, err := builder.Build(KeywordExpr{
		Value: "timeout",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := SQLQuery{
		Where: `message LIKE ? ESCAPE '\'`,
		Args:  []any{"%timeout%"},
	}

	if !reflect.DeepEqual(query, expected) {
		t.Fatalf("expected %#v, got %#v", expected, query)
	}
}

func TestBuildPhraseExpr(t *testing.T) {
	builder := NewSQLBuilder()

	query, err := builder.Build(PhraseExpr{
		Value: "failed to connect",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := SQLQuery{
		Where: "message LIKE ?",
		Args:  []any{"%failed to connect%"},
	}

	if !reflect.DeepEqual(query, expected) {
		t.Fatalf("expected %#v, got %#v", expected, query)
	}
}

func TestBuildNotExpr(t *testing.T) {
	builder := NewSQLBuilder()

	query, err := builder.Build(NotExpr{
		Expr: FieldExpr{
			Field: "status",
			Value: "info",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := SQLQuery{
		Where: "NOT (status = ?)",
		Args:  []any{"info"},
	}

	if !reflect.DeepEqual(query, expected) {
		t.Fatalf("expected %#v, got %#v", expected, query)
	}
}

func TestBuildAndExpr(t *testing.T) {
	builder := NewSQLBuilder()

	query, err := builder.Build(AndExpr{
		Exprs: []Expr{
			FieldExpr{Field: "service", Value: "api"},
			FieldExpr{Field: "status", Value: "error"},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := SQLQuery{
		Where: "(service = ?) AND (status = ?)",
		Args:  []any{"api", "error"},
	}

	if !reflect.DeepEqual(query, expected) {
		t.Fatalf("expected %#v, got %#v", expected, query)
	}
}

func TestBuildEmptyAndExpr(t *testing.T) {
	builder := NewSQLBuilder()

	query, err := builder.Build(AndExpr{
		Exprs: []Expr{},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := SQLQuery{
		Where: "1 = 1",
		Args:  nil,
	}

	if !reflect.DeepEqual(query, expected) {
		t.Fatalf("expected %#v, got %#v", expected, query)
	}
}

func TestBuildUnknownField(t *testing.T) {
	builder := NewSQLBuilder()

	_, err := builder.Build(FieldExpr{
		Field: "unknown",
		Value: "value",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBuildInvalidAttributeKey(t *testing.T) {
	builder := NewSQLBuilder()

	_, err := builder.Build(AttributeExpr{
		Key:   "user.id;DROP",
		Value: "123",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBuildParsedFilter(t *testing.T) {
	expr, err := Parse(`service:api -status:info @user.id:123 "failed to connect" timeout`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	builder := NewSQLBuilder()

	query, err := builder.Build(expr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := SQLQuery{
		Where: "(service = ?) AND (NOT (status = ?)) AND (json_extract(attributes, ?) = ?) AND (message LIKE ?) AND (message LIKE ?)",
		Args: []any{
			"api",
			"info",
			"$.user.id",
			"123",
			"%failed to connect%",
			"%timeout%",
		},
	}

	if !reflect.DeepEqual(query, expected) {
		t.Fatalf("expected %#v, got %#v", expected, query)
	}
}
