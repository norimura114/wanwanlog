package filter

import (
	"fmt"
)

type Parser struct {
	lexer *Lexer
	cur   Token
}

func NewParser(input string) (*Parser, error) {
	p := &Parser{
		lexer: NewLexer(input),
	}

	if err := p.next(); err != nil {
		return nil, err
	}

	return p, nil
}

func Parse(input string) (Expr, error) {
	p, err := NewParser(input)
	if err != nil {
		return nil, err
	}

	return p.Parse()
}

func (p *Parser) Parse() (Expr, error) {
	var exprs []Expr

	for p.cur.Type != TokenEOF {
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}

		exprs = append(exprs, expr)
	}

	if len(exprs) == 0 {
		return AndExpr{Exprs: []Expr{}}, nil
	}

	if len(exprs) == 1 {
		return exprs[0], nil
	}

	return AndExpr{Exprs: exprs}, nil
}

// func (p *Parser) parseExpr() (Expr, error) {
// 	if p.cur.Type == TokenMinus {
// 		return p.parseNot()
// 	}

// 	return p.parsePrimary()
// }

func (p *Parser) parseExpr() (Expr, error) {
	switch p.cur.Type {
	case TokenMinus:
		return p.parseNotExpr()
	case TokenAt:
		return p.parseAttributeExpr()
	case TokenIdent:
		return p.parseIdentLeadingExpr()
	case TokenString:
		value := p.cur.Literal
		if err := p.next(); err != nil {
			return nil, err
		}
		return KeywordExpr{Value: value}, nil
	default:
		return nil, fmt.Errorf("unexpected token: %s", p.cur.Literal)
	}
}

func (p *Parser) parseNotExpr() (Expr, error) {
	if err := p.next(); err != nil {
		return nil, err
	}

	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	return NotExpr{Expr: expr}, nil
}

func (p *Parser) parseAttributeExpr() (Expr, error) {
	if err := p.next(); err != nil {
		return nil, err
	}

	if p.cur.Type != TokenIdent {
		return nil, fmt.Errorf("expected attribute key after @")
	}

	key := p.cur.Literal

	if err := p.next(); err != nil {
		return nil, err
	}

	op, err := p.parseCompareOp()
	if err != nil {
		return nil, err
	}

	valueTok := p.cur
	if valueTok.Type != TokenIdent && valueTok.Type != TokenString {
		return nil, fmt.Errorf("expected attribute value")
	}

	lit, err := ParseLiteral(valueTok.Literal, valueTok.Quoted)
	if err != nil {
		return nil, err
	}

	if err := validateLiteralForOp(op, lit); err != nil {
		return nil, err
	}

	if err := p.next(); err != nil {
		return nil, err
	}

	return AttributeExpr{
		Key:   key,
		Op:    op,
		Value: lit,
	}, nil
}

func (p *Parser) parseIdentLeadingExpr() (Expr, error) {
	first := p.cur.Literal

	if err := p.next(); err != nil {
		return nil, err
	}

	// field:value
	if p.cur.Type == TokenColon {
		if err := p.next(); err != nil {
			return nil, err
		}

		if p.cur.Type != TokenIdent && p.cur.Type != TokenString {
			return nil, fmt.Errorf("expected field value")
		}

		value := p.cur.Literal

		if err := p.next(); err != nil {
			return nil, err
		}

		return FieldExpr{
			Field: first,
			Value: value,
		}, nil
	}

	// keyword
	return KeywordExpr{Value: first}, nil
}

func (p *Parser) parseCompareOp() (CompareOp, error) {
	switch p.cur.Type {
	case TokenColon:
		if err := p.next(); err != nil {
			return "", err
		}
		return OpEq, nil
	case TokenEqual:
		if err := p.next(); err != nil {
			return "", err
		}
		return OpEq, nil
	case TokenBangEqual:
		if err := p.next(); err != nil {
			return "", err
		}
		return OpNe, nil
	case TokenGreater:
		if err := p.next(); err != nil {
			return "", err
		}
		return OpGt, nil
	case TokenGreaterEqual:
		if err := p.next(); err != nil {
			return "", err
		}
		return OpGte, nil
	case TokenLess:
		if err := p.next(); err != nil {
			return "", err
		}
		return OpLt, nil
	case TokenLessEqual:
		if err := p.next(); err != nil {
			return "", err
		}
		return OpLte, nil
	default:
		return "", fmt.Errorf("expected comparison operator")
	}
}

func validateLiteralForOp(op CompareOp, lit Literal) error {
	if !op.IsRangeOp() {
		return nil
	}

	if lit.Kind != LiteralNumber {
		return fmt.Errorf("range comparison only supports number literal")
	}

	return nil
}

// func (p *Parser) parseNot() (Expr, error) {
// 	if err := p.expect(TokenMinus); err != nil {
// 		return nil, err
// 	}

// 	if p.cur.Type == TokenEOF {
// 		return nil, fmt.Errorf("expected expression after '-")
// 	}

// 	expr, err := p.parsePrimary()
// 	if err != nil {
// 		return nil, err
// 	}

// 	return NotExpr{Expr: expr}, nil
// }

// func (p *Parser) parsePrimary() (Expr, error) {
// 	switch p.cur.Type {
// 	case TokenString:
// 		value := p.cur.Literal

// 		if err := p.next(); err != nil {
// 			return nil, err
// 		}

// 		return PhraseExpr{Value: value}, nil

// 	case TokenIdent:
// 		return p.parseIdentBasedExpr()

// 	default:
// 		return nil, fmt.Errorf("unexpected token %q at position %d", p.cur.Literal, p.cur.Pos)
// 	}
// }

// func (p *Parser) parseIdentBasedExpr() (Expr, error) {
// 	left := p.cur

// 	if err := p.next(); err != nil {
// 		return nil, err
// 	}

// 	if p.cur.Type != TokenColon {
// 		return KeywordExpr{Value: left.Literal}, nil
// 	}

// 	if err := p.next(); err != nil {
// 		return nil, err
// 	}

// 	if p.cur.Type != TokenIdent && p.cur.Type != TokenString {
// 		return nil, fmt.Errorf("expected value after ':' at position %d", p.cur.Pos)
// 	}

// 	value := p.cur.Literal

// 	if err := p.next(); err != nil {
// 		return nil, err
// 	}

// 	if strings.HasPrefix(left.Literal, "@") {
// 		key := strings.TrimPrefix(left.Literal, "@")

// 		if key == "" {
// 			return nil, fmt.Errorf("attribute key must not be empty at position %d", p.cur.Pos)
// 		}

// 		return AttributeExpr{
// 			Key:   key,
// 			Value: value,
// 		}, nil
// 	}

// 	return FieldExpr{
// 		Field: left.Literal,
// 		Value: value,
// 	}, nil
// }

func (p *Parser) expect(tokenType TokenType) error {
	if p.cur.Type != tokenType {
		return fmt.Errorf("expected token %v but got %v at position %d", tokenType, p.cur.Type, p.cur.Pos)
	}

	return p.next()
}

func (p *Parser) next() error {
	tok, err := p.lexer.NextToken()
	if err != nil {
		return err
	}

	p.cur = tok
	return nil
}
