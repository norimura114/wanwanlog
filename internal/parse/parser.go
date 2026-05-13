package parse

import (
	"context"
	"io"
	"wanwanlog/internal/logrecord"
)

type ParsedLog struct {
	Record logrecord.LogRecord
	Line   int
}

type Parser interface {
	Parse(ctx context.Context, r io.Reader) (<-chan ParsedLog, <-chan error)
}
