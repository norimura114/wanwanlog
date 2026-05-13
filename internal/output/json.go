package output

import (
	"context"
	"encoding/json"
	"io"
)

type JSONFormatter struct {
	Pretty bool
}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

func (f *JSONFormatter) Write(ctx context.Context, w io.Writer, out SearchOutput) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	encoder := json.NewEncoder(w)

	if f.Pretty {
		encoder.SetIndent("", "  ")
	}

	return encoder.Encode(out)
}
