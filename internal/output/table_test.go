package output

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
	"wanwanlog/internal/logrecord"
)

func TestTableFormatterWritesSelectedAttributeColumns(t *testing.T) {
	formatter := NewTableFormatter("service", "@http.status", "message")
	record := logrecord.LogRecord{
		Timestamp: time.Date(2026, 7, 20, 12, 34, 56, 0, time.UTC),
		Service:   "api",
		Message:   "request failed",
		Attributes: map[string]any{
			"http": map[string]any{"status": float64(503)},
		},
	}

	var output bytes.Buffer
	err := formatter.Write(context.Background(), &output, SearchOutput{
		Total:   1,
		Results: []logrecord.LogRecord{record},
	})
	if err != nil {
		t.Fatal(err)
	}

	got := output.String()
	for _, want := range []string{
		"SERVICE",
		"@HTTP.STATUS",
		"MESSAGE",
		"api",
		"503",
		"request failed",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output does not contain %q:\n%s", want, got)
		}
	}
}

func TestTableFormatterLeavesMissingSelectedAttributeBlank(t *testing.T) {
	formatter := NewTableFormatter("@user.id", "message")

	var output bytes.Buffer
	err := formatter.Write(context.Background(), &output, SearchOutput{
		Total: 1,
		Results: []logrecord.LogRecord{{
			Message:    "hello",
			Attributes: map[string]any{},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output.String(), "@USER.ID  MESSAGE") {
		t.Fatalf("unexpected output:\n%s", output.String())
	}
}

func TestFormatTimeUsesDateAndTime(t *testing.T) {
	value := time.Date(2026, 7, 20, 12, 34, 56, 0, time.Local)
	if got, want := formatTime(value), "2026-07-20 12:34:56"; got != want {
		t.Fatalf("formatTime() = %q, want %q", got, want)
	}
}
