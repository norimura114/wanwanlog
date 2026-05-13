package filter

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenString

	TokenAt
	TokenColon
	TokenEqual
	TokenBangEqual
	TokenGreater
	TokenGraterEqual
	TokenLess
	TokenLessEqual
	TokenMinus

	TokenLParen
	TokenRParen
	TokenOR
)

type Token struct {
	Type    TokenType
	Literal string
	Pos     int
	Quoted  bool
}
