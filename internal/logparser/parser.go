package logparser

import (
	"io"
	"wanwanlog/internal/logrecord"
)

type Parser interface {
	Parse(r io.Reader) ([]logrecord.LogRecord, error)
}
