package parse

import (
	"context"
	"strings"
	"testing"
)

func TestJSONLinesParser_Parse(t *testing.T) {
	input := strings.NewReader(`{"id":"1","timestamp":"2026-05-10T12:00:00Z","message":"failed to connect database","service":"api","status":"error","user":{"id":"123"}}
{"id":"2","timestamp":"2026-05-10T12:01:00Z","message":"request completed","service":"web","status":"info"}`)

	parser := NewJSONLinesParser()

	logs, errs := parser.Parse(context.Background(), input)

	var records []ParsedLog
	for parsed := range logs {
		records = append(records, parsed)
	}

	for err := range errs {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	first := records[0].Record

	if first.ID != "1" {
		t.Errorf("expected id to be 1, got %s", first.ID)
	}

	if first.Message != "failed to connect database" {
		t.Errorf("unexpected message: %s", first.Message)
	}

	if first.Service != "api" {
		t.Errorf("expected service to be api, got %s", first.Service)
	}

	if first.Status != "error" {
		t.Errorf("expected status to be error, got %s", first.Status)
	}

	if first.Raw == "" {
		t.Error("expected raw json to be set")
	}

	if first.Attributes["user"] == nil {
		t.Error("expected user attribute to be preserved")
	}
}

func TestJSONLinesParser_Parse_InvalidJSON(t *testing.T) {
	input := strings.NewReader(`{"id":"1","message":"ok"}
invalid json
{"id":"2","message":"ok"}`)

	parser := NewJSONLinesParser()

	logs, errs := parser.Parse(context.Background(), input)

	var count int
	for range logs {
		count++
	}

	var errCount int
	for err := range errs {
		if err != nil {
			errCount++
		}
	}

	if count != 2 {
		t.Fatalf("expected 2 valid records, got %d", count)
	}

	if errCount != 1 {
		t.Fatalf("expected 1 parse error, got %d", errCount)
	}
}
