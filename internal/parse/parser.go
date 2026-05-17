package parse

import (
	"io"
	"wanwanlog/internal/logrecord"
)

type ParsedLog struct {
	Record logrecord.LogRecord
	Line   int
}

type ParseResult struct {
	Log  ParsedLog
	Line int
	Err  error
}

type Iterator interface {
	Next() bool
	Result() ParseResult
	Err() error
}

type Parser interface {
	Parse(r io.Reader) Iterator
}
