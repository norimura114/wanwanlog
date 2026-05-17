package filter

import (
	"reflect"
	"testing"
)

func TestParseFieldExpr(t *testing.T) {
	expr, err := Parse("service:api")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := FieldExpr{
		Field: "service",
		Value: "api",
	}

	if !reflect.DeepEqual(expr, expected) {
		t.Fatalf("expected %#v, got %#v", expected, expr)
	}
}

func TestParseAttributeExpr(t *testing.T) {
	expr, err := Parse("@user.id:123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := AttributeExpr{
		Key: "user.id",
		Op:  OpEq,
		Value: Literal{
			Kind: LiteralNumber,
			Text: "123",
			Num:  123,
		},
	}

	if !reflect.DeepEqual(expr, expected) {
		t.Fatalf("expected %#v, got %#v", expected, expr)
	}
}

func TestParseKeywordExpr(t *testing.T) {
	expr, err := Parse("timeout")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := KeywordExpr{
		Value: "timeout",
	}

	if !reflect.DeepEqual(expr, expected) {
		t.Fatalf("expected %#v, got %#v", expected, expr)
	}
}

func TestParsePhraseExpr(t *testing.T) {
	expr, err := Parse(`"failed to connect"`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := PhraseExpr{
		Value: "failed to connect",
	}

	if !reflect.DeepEqual(expr, expected) {
		t.Fatalf("expected %#v, got %#v", expected, expr)
	}
}

func TestParseAndExpr(t *testing.T) {
	expr, err := Parse("service:api status:error")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := AndExpr{
		Exprs: []Expr{
			FieldExpr{Field: "service", Value: "api"},
			FieldExpr{Field: "status", Value: "error"},
		},
	}

	if !reflect.DeepEqual(expr, expected) {
		t.Fatalf("expected %#v, got %#v", expected, expr)
	}
}

func TestParseNotFieldExpr(t *testing.T) {
	expr, err := Parse("-status:info")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := NotExpr{
		Expr: FieldExpr{
			Field: "status",
			Value: "info",
		},
	}

	if !reflect.DeepEqual(expr, expected) {
		t.Fatalf("expected %#v, got %#v", expected, expr)
	}
}

func TestParseMixedExpr(t *testing.T) {
	expr, err := Parse(`service:api -status:info @user.id:123 "failed to connect" timeout`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := AndExpr{
		Exprs: []Expr{
			FieldExpr{Field: "service", Value: "api"},
			NotExpr{
				Expr: FieldExpr{
					Field: "status",
					Value: "info",
				},
			},
			AttributeExpr{
				Key: "user.id",
				Op:  OpEq,
				Value: Literal{
					Kind: LiteralNumber,
					Text: "123",
					Num:  123,
				},
			},
			PhraseExpr{
				Value: "failed to connect",
			},
			KeywordExpr{
				Value: "timeout",
			},
		},
	}

	if !reflect.DeepEqual(expr, expected) {
		t.Fatalf("expected %#v, got %#v", expected, expr)
	}
}

func TestParseUnterminatedString(t *testing.T) {
	_, err := Parse(`"failed to connect`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseMinusWithoutExpr(t *testing.T) {
	_, err := Parse("-")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseMissingValueAfterColon(t *testing.T) {
	_, err := Parse("service:")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseEmptyInput(t *testing.T) {
	expr, err := Parse("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := AndExpr{Exprs: []Expr{}}

	if !reflect.DeepEqual(expr, expected) {
		t.Fatalf("expected %#v, got %#v", expected, expr)
	}
}
