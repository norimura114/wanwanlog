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

	// case PhraseExpr:
	// 	return b.buildPhraseExpr(e)

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
	jsonPath, err := buildJSONPath(expr.Key)
	if err != nil {
		return "", nil, err
	}

	op := expr.Op

	switch expr.Value.Kind {
	case LiteralNumber:
		return buildNumberAttributeSQL(jsonPath, op, expr.Value.Num)

	case LiteralString:
		return buildStringAttributeSQL(jsonPath, op, expr.Value.Text)

	case LiteralBool:
		return buildBoolAttributeSQL(jsonPath, op, expr.Value.Bool)

	case LiteralNull:
		return buildNullAttributeSQL(jsonPath, op)

	default:
		return "", nil, fmt.Errorf("unknown literal kind")
	}
}

func buildNumberAttributeSQL(jsonPath string, op CompareOp, value float64) (string, []any, error) {
	if !isAllowedCompareOp(op) {
		return "", nil, fmt.Errorf("unsupported operator: %s", op)
	}

	sql := `
json_type(attributes, ?) IN ('integer', 'real')
AND json_extract(attributes, ?) ` + string(op) + ` ?
`
	return sql, []any{jsonPath, jsonPath, value}, nil
}

func buildStringAttributeSQL(jsonPath string, op CompareOp, value string) (string, []any, error) {
	if op.IsRangeOp() {
		return "", nil, fmt.Errorf("range comparison only supports number attribute")
	}

	if op != OpEq && op != OpNe {
		return "", nil, fmt.Errorf("unsupported string operator: %s", op)
	}

	sql := `
json_type(attributes, ?) = 'text'
AND json_extract(attributes, ?) ` + string(op) + ` ?
`
	return sql, []any{jsonPath, jsonPath, value}, nil
}

func buildBoolAttributeSQL(jsonPath string, op CompareOp, value bool) (string, []any, error) {
	if op.IsRangeOp() {
		return "", nil, fmt.Errorf("range comparison only supports number attribute")
	}

	if op != OpEq && op != OpNe {
		return "", nil, fmt.Errorf("unsupported string operator: %s", op)
	}

	sqlValue := 0
	if value {
		sqlValue = 1
	}

	sql := `
json_type(attributes, ?) = IN ('true', 'false')
AND json_extract(attributes, ?) ` + string(op) + ` ?
`
	return sql, []any{jsonPath, jsonPath, sqlValue}, nil
}

func buildNullAttributeSQL(jsonPath string, op CompareOp) (string, []any, error) {
	if op.IsRangeOp() {
		return "", nil, fmt.Errorf("range comparison only supports number attribute")
	}

	switch op {
	case OpEq:
		return `json_type(attributes, ?) = 'null'`, []any{jsonPath}, nil
	case OpNe:
		// path が存在し、かつ null ではないものだけを != null とする
		return `
json_type(attributes, ?) IS NOT NULL
AND json_type(attributes, ?) != 'null'
`, []any{jsonPath, jsonPath}, nil
	default:
		return "", nil, fmt.Errorf("unsupported null operator: %s", op)
	}
}

func isAllowedCompareOp(op CompareOp) bool {
	switch op {
	case OpEq, OpNe, OpGt, OpGte, OpLt, OpLte:
		return true
	default:
		return false
	}
}

func buildJSONPath(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("attribute key is empty")
	}

	parts := strings.Split(key, ".")
	for _, part := range parts {
		if part == "" {
			return "", fmt.Errorf("invalid attribute key: %s", key)
		}

		for _, r := range part {
			if !isSafeJSONPathChar(r) {
				return "", fmt.Errorf("invalid attribute key: %s", key)
			}
		}
	}

	return "$." + strings.Join(parts, "."), nil
}

func isSafeJSONPathChar(r rune) bool {
	return unicode.IsLetter(r) ||
		unicode.IsDigit(r) ||
		r == '_' ||
		r == '-'
}

// func (b *SQLBuilder) buildAttributeExpr(expr AttributeExpr) (string, []any, error) {
// 	if err := validateAttributeKey(expr.Key); err != nil {
// 		return "", nil, err
// 	}

// 	jsonPath := "$." + expr.Key

// 	return "json_extract(attributes, ?) = ?", []any{jsonPath, expr.Value}, nil
// }

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
