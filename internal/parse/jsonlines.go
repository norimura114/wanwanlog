package parse

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"wanwanlog/internal/logrecord"
)

type JSONLinesParser struct{}

const maxJSONLineSize = 10 * 1024 * 1024

func NewJSONLinesParser() *JSONLinesParser {
	return &JSONLinesParser{}
}

func (p *JSONLinesParser) Parse(r io.Reader) Iterator {
	scanner := bufio.NewScanner(r)

	scanner.Buffer(make([]byte, 64*1024), 64*1024*1024)

	return &JSONLinesIterator{
		scanner: scanner,
	}
}

type JSONLinesIterator struct {
	scanner *bufio.Scanner
	lineNo  int
	result  ParseResult
	err     error
}

func (it *JSONLinesIterator) Next() bool {
	for it.scanner.Scan() {
		it.lineNo++
		line := strings.TrimSpace(it.scanner.Text())

		if line == "" {
			continue
		}
		if len(line) > maxJSONLineSize {
			it.result = ParseResult{
				Line: it.lineNo,
				Err:  fmt.Errorf("line %d: line exceeds %d bytes", it.lineNo, maxJSONLineSize),
			}
			return true
		}

		record, err := parseJSONLine(line)
		if err != nil {
			it.result = ParseResult{
				Line: it.lineNo,
				Err:  fmt.Errorf("line %d: %w", it.lineNo, err),
			}
			return true
		}

		it.result = ParseResult{
			Log: ParsedLog{
				Record: record,
				Line:   it.lineNo,
			},
			Line: it.lineNo,
		}
		return true
	}

	if err := it.scanner.Err(); err != nil {
		it.err = err
	}

	return false
}

func (it *JSONLinesIterator) Result() ParseResult {
	return it.result
}

func (it *JSONLinesIterator) Err() error {
	return it.err
}

func parseJSONLine(line string) (logrecord.LogRecord, error) {
	var raw map[string]any

	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return logrecord.LogRecord{}, err
	}

	record := logrecord.LogRecord{
		ID:         parseId(raw),
		Timestamp:  parseTimestamp(raw),
		Message:    getString(raw, "message"),
		Service:    getString(raw, "service"),
		Status:     getString(raw, "status"),
		Attributes: raw,
		Raw:        line,
	}

	return record, nil
}

func getString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}

	switch value := v.(type) {
	case string:
		return value
	default:
		return fmt.Sprint(value)
	}
}

func parseId(m map[string]any) string {
	value := getString(m, "id")
	if value == "" {
		id, err := logrecord.NewID()
		if err != nil {
			return ""
		}

		return id
	}

	return value
}

func parseTimestamp(m map[string]any) time.Time {
	value := getString(m, "timestamp")
	if value == "" {
		return time.Time{}
	}

	t, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return t
	}

	t, err = time.Parse(time.RFC3339Nano, value)
	if err == nil {
		return t
	}

	return time.Time{}
}
