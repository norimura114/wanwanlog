package filter

import (
	"fmt"
	"strings"
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

func (p *Parser) parseExpr() (Expr, error) {
	if p.cur.Type == TokenMinus {
		return p.parseNot()
	}

	return p.parsePrimary()
}

func (p *Parser) parseNot() (Expr, error) {
	if err := p.expect(TokenMinus); err != nil {
		return nil, err
	}

	if p.cur.Type == TokenEOF {
		return nil, fmt.Errorf("expected expression after '-")
	}

	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	return NotExpr{Expr: expr}, nil
}

func (p *Parser) parsePrimary() (Expr, error) {
	switch p.cur.Type {
	case TokenString:
		value := p.cur.Literal

		if err := p.next(); err != nil {
			return nil, err
		}

		return PhraseExpr{Value: value}, nil

	case TokenIdent:
		return p.parseIdentBasedExpr()

	default:
		return nil, fmt.Errorf("unexpected token %q at position %d", p.cur.Literal, p.cur.Pos)
	}
}

func (p *Parser) parseIdentBasedExpr() (Expr, error) {
	left := p.cur

	if err := p.next(); err != nil {
		return nil, err
	}

	if p.cur.Type != TokenColon {
		return KeywordExpr{Value: left.Literal}, nil
	}

	if err := p.next(); err != nil {
		return nil, err
	}

	if p.cur.Type != TokenIdent && p.cur.Type != TokenString {
		return nil, fmt.Errorf("expected value after ':' at position %d", p.cur.Pos)
	}

	value := p.cur.Literal

	if err := p.next(); err != nil {
		return nil, err
	}

	if strings.HasPrefix(left.Literal, "@") {
		key := strings.TrimPrefix(left.Literal, "@")

		if key == "" {
			return nil, fmt.Errorf("attribute key must not be empty at position %d", p.cur.Pos)
		}

		return AttributeExpr{
			Key:   key,
			Value: value,
		}, nil
	}

	return FieldExpr{
		Field: left.Literal,
		Value: value,
	}, nil
}

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
