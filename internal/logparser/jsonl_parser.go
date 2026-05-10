package logparser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"wanwanlog/internal/logrecord"
)

type JSONLinesParser struct{}

func NewJSONLinesParser() *JSONLinesParser {
	return &JSONLinesParser{}
}

func (p *JSONLinesParser) Parse(r io.Reader) ([]logrecord.LogRecord, error) {
	scanner := bufio.NewScanner(r)

	var records []logrecord.LogRecord
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Bytes()

		if len(line) == 0 {
			continue
		}

		var record logrecord.LogRecord
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, fmt.Errorf("failed to parse json line at line %d: %w", lineNumber, err)
		}

		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read json lines: %w", err)
	}

	return records, nil
}
