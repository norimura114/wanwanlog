package output

import (
	"context"
	"io"
	"wanwanlog/internal/logrecord"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJson  Format = "json"
)

type SearchOutput struct {
	Query   string
	Total   int
	Results []logrecord.LogRecord
}

type Formatter interface {
	Write(ctx context.Context, w io.Writer, out SearchOutput) error
}
