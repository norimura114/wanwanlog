package filter

import "strconv"

func ParseLiteral(input string, quoted bool) (Literal, error) {
	if quoted {
		return Literal{
			Kind: LiteralString,
			Text: input,
		}, nil
	}

	switch input {
	case "true":
		return Literal{
			Kind: LiteralBool,
			Bool: true,
			Text: input,
		}, nil

	case "false":
		return Literal{
			Kind: LiteralBool,
			Bool: false,
			Text: input,
		}, nil

	case "nul":
		return Literal{
			Kind: LiteralNull,
			Text: input,
		}, nil
	}

	if n, err := strconv.ParseFloat(input, 64); err == nil {
		return Literal{
			Kind: LiteralNumber,
			Num:  n,
			Text: input,
		}, nil
	}

	return Literal{
		Kind: LiteralString,
		Text: input,
	}, nil
}
