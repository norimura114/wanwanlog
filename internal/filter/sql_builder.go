package filter

import (
	"fmt"
	"strings"
	"unicode"
)

type SQLQuery struct {
	Where string
	Args  []any
}

type SQLBuilder struct{}

func NewSQLBuilder() *SQLBuilder {
	return &SQLBuilder{}
}

func (b *SQLBuilder) Build(expr Expr) (SQLQuery, error) {
	where, args, err := b.buildExpr(expr)
	if err != nil {
		return SQLQuery{}, nil
	}

	return SQLQuery{
		Where: where,
		Args:  args,
	}, nil
}

func (b *SQLBuilder) buildExpr(expr Expr) (string, []any, error) {
	switch e := expr.(type) {
	case AndExpr:
		return b.buildAndExpr(e)

	case NotExpr:
		return b.buildNotExpr(e)

	case FieldExpr:
		return b.buildFieldExpr(e)

	case AttributeExpr:
		return b.buildAttributeExpr(e)

	case KeywordExpr:
		return b.buildKeywordExpr(e)

	case PhraseExpr:
		return b.buildPhraseExpr(e)

	default:
		return "", nil, fmt.Errorf("unsupported filter expression %T", expr)
	}
}

func (b *SQLBuilder) buildAndExpr(expr AndExpr) (string, []any, error) {
	if len(expr.Exprs) == 0 {
		return "1 = 1", nil, nil
	}

	var parts []string
	var args []any

	for _, child := range expr.Exprs {
		where, childArgs, err := b.buildExpr(child)
		if err != nil {
			return "", nil, err
		}

		parts = append(parts, "("+where+")")
		args = append(args, childArgs...)
	}

	return strings.Join(parts, " AND "), args, nil
}

func (b *SQLBuilder) buildNotExpr(expr NotExpr) (string, []any, error) {
	where, args, err := b.buildExpr(expr.Expr)
	if err != nil {
		return "", nil, err
	}

	return "NOT (" + where + ")", args, nil
}

func (b *SQLBuilder) buildFieldExpr(expr FieldExpr) (string, []any, error) {
	if err := validateFieldName(expr.Field); err != nil {
		return "", nil, err
	}

	return expr.Field + " = ?", []any{expr.Value}, nil
}

func (b *SQLBuilder) buildAttributeExpr(expr AttributeExpr) (string, []any, error) {
	if err := validateAttributeKey(expr.Key); err != nil {
		return "", nil, err
	}

	jsonPath := "$." + expr.Key

	return "json_extract(attributes, ?) = ?", []any{jsonPath, expr.Value}, nil
}

func (b *SQLBuilder) buildKeywordExpr(expr KeywordExpr) (string, []any, error) {
	return `message LIKE ? ESCAPE '\'`, []any{"%" + escapeLike(expr.Value) + "%"}, nil
}

func (b *SQLBuilder) buildPhraseExpr(expr PhraseExpr) (string, []any, error) {
	return `message LIKE ? ESCAPE '\'`, []any{"%" + escapeLike(expr.Value) + "%"}, nil
}

func validateFieldName(field string) error {
	if field == "" {
		return fmt.Errorf("field name must not be empty")
	}

	switch field {
	case "timestamp", "service", "status", "message":
		return nil
	default:
		return fmt.Errorf("unsupported field: %s", field)
	}
}

func validateAttributeKey(key string) error {
	if key == "" {
		return fmt.Errorf("attribute key must not be empty")
	}

	parts := strings.Split(key, ".")
	for _, part := range parts {
		if part == "" {
			return fmt.Errorf("invalid attribute key: %s", key)
		}

		for _, ch := range part {
			if !isSafeAttributeKeyChar(ch) {
				return fmt.Errorf("invalid attribute key: %s", key)
			}
		}
	}

	return nil
}

func isSafeAttributeKeyChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)

	return value
}
