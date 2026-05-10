package logparser

import (
	"strings"
	"testing"
)

func TestJSONLinesParser_Parse(t *testing.T) {
	input := strings.NewReader(`{"timestamp":"2026-05-10T10:00:00Z","level":"info","message":"started","service":"api","attrs":{"request_id":"req-1"}}
{"timestamp":"2026-05-10T10:01:00Z","level":"error","message":"failed","service":"worker","attrs":{"job_id":"job-1"}}
`)

	parser := NewJSONLinesParser()

	records, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	if records[0].Level != "info" {
		t.Errorf("expected first record level to be info, got %s", records[0].Level)
	}

	if records[0].Message != "started" {
		t.Errorf("expected first record message to be started, got %s", records[0].Message)
	}

	if records[0].Service != "api" {
		t.Errorf("expected first record service to be api, got %s", records[0].Service)
	}

	if records[1].Level != "error" {
		t.Errorf("expected second record level to be error, got %s", records[1].Level)
	}
}

func TestJSONLinesParser_Parse_InvalidJSON(t *testing.T) {
	input := strings.NewReader(`{"timestamp":"2026-05-10T10:00:00Z","level":"info"}
invalid-json
`)

	parser := NewJSONLinesParser()

	_, err := parser.Parse(input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("expected error to contain line number, got %v", err)
	}
}

func TestJSONLinesParser_Parse_SkipEmptyLines(t *testing.T) {
	input := strings.NewReader(`

{"timestamp":"2026-05-10T10:00:00Z","level":"info","message":"started","service":"api"}

`)

	parser := NewJSONLinesParser()

	records, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	if records[0].Message != "started" {
		t.Errorf("expected message to be started, got %s", records[0].Message)
	}
}
