package filter

type Expr interface {
	exprNode()
}

type AndExpr struct {
	Exprs []Expr
}

func (e AndExpr) exprNode() {}

type NotExpr struct {
	Expr Expr
}

func (e NotExpr) exprNode() {}

type FieldExpr struct {
	Field string
	Value string
}

func (e FieldExpr) exprNode() {}

type AttributeExpr struct {
	Key   string
	Op    CompareOp
	Value Literal
}

func (e AttributeExpr) exprNode() {}

type KeywordExpr struct {
	Value string
}

func (e KeywordExpr) exprNode() {}

type PhraseExpr struct {
	Value string
}

func (e PhraseExpr) exprNode() {}

type CompareOp string

const (
	OpEq  CompareOp = "="
	OpNe  CompareOp = "!="
	OpGt  CompareOp = ">"
	OpGte CompareOp = ">="
	OpLt  CompareOp = "<"
	OpLte CompareOp = "<="
)

func (op CompareOp) IsRangeOp() bool {
	return op == OpGt || op == OpGte || op == OpLt || op == OpLte
}

type LiteralKind int

const (
	LiteralString LiteralKind = iota
	LiteralNumber
	LiteralBool
	LiteralNull
)

type Literal struct {
	Kind LiteralKind
	Text string
	Num  float64
	Bool bool
}
