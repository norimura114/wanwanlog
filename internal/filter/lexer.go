package filter

import (
	"fmt"
	"strings"
	"unicode"
)

type Lexer struct {
	input []rune
	pos   int
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input: []rune(input),
		pos:   0,
	}
}

func (l *Lexer) NextToken() (Token, error) {
	l.skipWhitespace()

	if l.isEOF() {
		return Token{Type: TokenEOF, Pos: l.pos}, nil
	}

	ch := l.peek()
	start := l.pos

	switch ch {
	case '@':
		l.advance()
		return Token{Type: TokenAt, Literal: "@", Pos: start}, nil
	case ':':
		l.advance()
		return Token{Type: TokenColon, Literal: ":", Pos: start}, nil
	case '=':
		l.advance()
		return Token{Type: TokenEqual, Literal: "=", Pos: start}, nil
	case '!':
		if l.peekNext() == '=' {
			l.advance()
			l.advance()
			return Token{Type: TokenBangEqual, Literal: "!=", Pos: start}, nil
		}
		return Token{}, fmt.Errorf("unexpected character: !")
	case '>':
		if l.peekNext() == '=' {
			l.advance()
			l.advance()
			return Token{Type: TokenGreaterEqual, Literal: ">=", Pos: start}, nil
		}
		l.advance()
		return Token{Type: TokenGreater, Literal: ">", Pos: start}, nil
	case '<':
		if l.peekNext() == '=' {
			l.advance()
			l.advance()
			return Token{Type: TokenLessEqual, Literal: "<=", Pos: start}, nil
		}
		l.advance()
		return Token{Type: TokenLess, Literal: "<", Pos: start}, nil
	case '-':
		l.advance()
		return Token{Type: TokenMinus, Literal: "-", Pos: start}, nil
	case '(':
		l.advance()
		return Token{Type: TokenLParen, Literal: "("}, nil
	case ')':
		l.advance()
		return Token{Type: TokenRParen, Literal: ")"}, nil
	case '"':
		return l.readString()
	default:
		if isIdentChar(ch) {
			return l.readIdent(), nil
		}

		return Token{}, fmt.Errorf("unexpected character %q at position %d", ch, start)
	}
}

func (l *Lexer) skipWhitespace() {
	for !l.isEOF() && unicode.IsSpace(l.peek()) {
		l.advance()
	}
}

func (l *Lexer) readIdent() Token {
	start := l.pos

	for !l.isEOF() && isIdentChar(l.peek()) {
		l.advance()
	}

	lit := string(l.input[start:l.pos])

	if lit == "OR" {
		return Token{Type: TokenOR, Literal: lit}
	}

	return Token{
		Type:    TokenIdent,
		Literal: string(lit),
		Pos:     start,
	}
}

func (l *Lexer) readString() (Token, error) {
	start := l.pos

	l.advance()

	var b strings.Builder

	for !l.isEOF() {
		ch := l.peek()

		if ch == '"' {
			l.advance()
			return Token{
				Type:    TokenString,
				Literal: b.String(),
				Pos:     start,
				Quoted:  true,
			}, nil
		}

		if ch == '\\' {
			l.advance()

			if l.isEOF() {
				return Token{}, fmt.Errorf("unterminated escape sequence at position %d", l.pos)
			}

			escaped := l.peek()
			switch escaped {
			case '"', '\\':
				b.WriteRune(escaped)
			default:
				b.WriteRune(escaped)
			}

			l.advance()
			continue
		}

		b.WriteRune(ch)
		l.advance()
	}

	return Token{}, fmt.Errorf("unterminated string at position %d", start)
}

func (l *Lexer) peek() rune {
	return l.input[l.pos]
}

func (l *Lexer) peekNext() rune {
	if l.pos+1 >= len(l.input) {
		return 0
	}

	return l.input[l.pos+1]
}

func (l *Lexer) advance() {
	l.pos++
}

func (l *Lexer) isEOF() bool {
	return l.pos >= len(l.input)
}

func isIdentChar(ch rune) bool {
	return unicode.IsLetter(ch) ||
		unicode.IsDigit(ch) ||
		ch == '_' ||
		ch == '-' ||
		ch == '.' ||
		ch == '/' ||
		ch == '*'
}
