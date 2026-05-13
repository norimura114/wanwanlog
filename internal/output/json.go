package output

import (
	"context"
	"encoding/json"
	"io"
)

type JSONFormatter struct {
	Pretty bool
}

func NewJSONFormatter(pretty bool) *JSONFormatter {
	return &JSONFormatter{Pretty: pretty}
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
