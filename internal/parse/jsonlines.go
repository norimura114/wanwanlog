package parse

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"wanwanlog/internal/logrecord"
)

type JSONLinesParser struct{}

func NewJSONLinesParser() *JSONLinesParser {
	return &JSONLinesParser{}
}
func (p *JSONLinesParser) Parse(
	ctx context.Context,
	r io.Reader,
) (<-chan ParsedLog, <-chan error) {
	out := make(chan ParsedLog)
	errs := make(chan error)

	go func() {
		defer close(out)
		defer close(errs)

		scanner := bufio.NewScanner(r)

		const maxLineSize = 10 * 1024 * 1024
		scanner.Buffer(make([]byte, 64*1024), maxLineSize)

		lineNo := 0

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			default:
			}

			lineNo++
			line := strings.TrimSpace(scanner.Text())

			if line == "" {
				continue
			}

			record, err := parseJSONLine(line)
			if err != nil {
				errs <- fmt.Errorf("line %d: %w", lineNo, err)
				continue
			}

			out <- ParsedLog{
				Record: record,
				Line:   lineNo,
			}
		}

		if err := scanner.Err(); err != nil {
			errs <- err
		}
	}()

	return out, errs
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
